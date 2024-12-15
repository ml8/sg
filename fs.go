package sg

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// File system operation.
type fsOp int

const (
	writeOp fsOp = iota
	deleteOp
)

// File system update operation.
type fsUpdate struct {
	Path  string
	IsDir bool
	Op    fsOp
}

// Reads the filesystem rooted at root. Can be used to read the filesystem once
// or indefinitely.
type fsReader struct {
	updates chan fsUpdate
	errs    chan error
	root    string
	dirs    *Set[string]

	watcher *fsnotify.Watcher // only valid for Indefinitely()
	wg      *sync.WaitGroup   // only valid for Once()
}

// Create a new FSReader rooted at the given directory.
func newFsReader(dir string) *fsReader {
	return &fsReader{
		root: dir,
	}
}

// Normalize a path to the root.
func (f *fsReader) normalize(path string) string {
	path = filepath.ToSlash(path)
	return strings.TrimPrefix(path, slashify(f.root))
}

func (f *fsReader) init() {
	f.updates = make(chan fsUpdate)
	f.errs = make(chan error)
	f.dirs = NewSet[string]()
}

// Start reading the filesystem once. Channels will be closed when the read is complete.
func (f *fsReader) Once() (<-chan fsUpdate, <-chan error, error) {
	f.init()
	f.wg = &sync.WaitGroup{}
	if err := f.walk(); err != nil {
		return nil, nil, err
	}

	// Start closing thread.
	go func() {
		f.wg.Wait()
		close(f.updates)
		close(f.errs)
	}()

	return f.updates, f.errs, nil
}

// Start reading the filesystem. Channels will remain open indefinitely unless
// the reader failed to start reading.
func (f *fsReader) Indefinitely() (<-chan fsUpdate, <-chan error, error) {
	f.init()
	var err error
	if f.watcher, err = fsnotify.NewWatcher(); err != nil {
		return nil, nil, err
	}

	if err = f.walk(); err != nil {
		return nil, nil, err
	}

	// Start forwarding thread.
	go func() {
		for {
			select {
			case e, ok := <-f.watcher.Events:
				if !ok {
					return
				}
				go func() {
					lg.Debugf("Event: %v", e)
					if e.Has(fsnotify.Chmod) &&
						!(e.Has(fsnotify.Write) || e.Has(fsnotify.Create) ||
							e.Has(fsnotify.Remove) || e.Has(fsnotify.Rename)) {
						lg.Debugf("Ignoring chmod for %s", e.Name)
						return
					}
					up := fsUpdate{
						Path: f.normalize(e.Name),
					}
					if e.Has(fsnotify.Write) || e.Has(fsnotify.Create) {
						stat, err := os.Stat(e.Name)
						if err != nil {
							f.errs <- err
							return
						}
						lg.Debugf("Write for %s", up.Path)
						up.Op = writeOp
						up.IsDir = stat.IsDir()
						if up.IsDir && e.Has(fsnotify.Create) {
							lg.Debugf("Watching directory %s", e.Name)
							f.dirs.Add(e.Name)
							f.watcher.Add(e.Name)
						}
					} else if e.Has(fsnotify.Remove) || e.Has(fsnotify.Rename) {
						lg.Debugf("Remove for %s", up.Path)
						up.Op = deleteOp
						up.IsDir = f.dirs.Has(e.Name)
						if up.IsDir && e.Has(fsnotify.Remove) {
							lg.Debugf("Unwatching directory %s", e.Name)
						}
						// ignore error; only given on unwatched path.
						f.watcher.Remove(e.Name)
					}
					f.updates <- up
				}()
			case err, ok := <-f.watcher.Errors:
				if !ok {
					return
				}
				go func() {
					f.errs <- err
				}()
			}
		}
	}()

	return f.updates, f.errs, nil
}

// Recursively walk the filesystem to get a snapshot of current state.
func (f *fsReader) walk() error {
	return filepath.WalkDir(f.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		isDir := d.IsDir()
		if isDir {
			f.dirs.Add(path)
		}

		if f.watcher != nil {
			if isDir {
				lg.Debugf("Watching directory %s", path)
				f.watcher.Add(path)
			}
		}

		if f.wg != nil {
			f.wg.Add(1)
		}

		// Normalize path for clients.
		path = f.normalize(path)
		lg.Debugf("Write for %s", path)
		go func() {
			f.updates <- fsUpdate{
				Path:  path,
				IsDir: isDir,
				Op:    writeOp,
			}
			if f.wg != nil {
				f.wg.Done()
			}
		}()
		return nil
	})
}

// Handles all operations on the filesystem.
type fsutil struct {
	infs   fs.FS
	indir  string
	outdir string
}

// Create a new FS with the given input/output directories.
func newFS(indir string, outdir string) *fsutil {
	return &fsutil{
		infs:   os.DirFS(indir),
		indir:  indir,
		outdir: outdir,
	}
}

// Recursively make directories under the given path.
func (f *fsutil) makeDirs(path string) error {
	path = filepath.Dir(path) // only make the enclosing dirs.
	return os.MkdirAll(path, 0755)
}

// Read a file in one shot.
func (f *fsutil) ReadFile(path string) ([]byte, error) {
	b, err := f.infs.(fs.ReadFileFS).ReadFile(path)
	return b, err
}

// Open a file for reading.
func (f *fsutil) Read(path string) (fs.File, error) {
	return f.infs.Open(path)
}

// Write a file to output directory in one shot.
func (f *fsutil) WriteFile(path string, data []byte) error {
	path = filepath.Join(f.outdir, path)
	if err := f.makeDirs(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Remove the given file form the output directory.
func (f *fsutil) Remove(path string) error {
	if path == "" {
		return nil
	}
	if err := os.RemoveAll(filepath.Join(f.outdir, path)); err != nil && err != os.ErrNotExist {
		return err
	}
	return nil
}

// Copy a file from input directory to output directory.
func (f *fsutil) CopyFile(src, dst string) error {
	lg.Debugf("Copying file %s to %s", src, dst)
	if stat, err := f.infs.(fs.StatFS).Stat(src); err != nil {
		return err
	} else if stat.IsDir() {
		// If a dir, ignore. Writing a file to it will create it.
		lg.Debugf("Ignoring directory %s", src)
		return nil
	}

	byts, err := f.ReadFile(src)
	if err != nil {
		return err
	}
	return f.WriteFile(dst, byts)
}

// Ensure a path is terminated with a /.
func slashify(s string) string {
	if s[len(s)-1] == '/' {
		return s
	}
	return s + "/"
}
