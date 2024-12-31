package sg

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Generic renderer interface.
type Renderer interface {
	Render() error
}

// Renders a site in online mode. Watches the input directory for changes and
// renders them as files are changed. When a template changes, all pages are
// re-rendered currently. Re-rendering after a template change waits for a
// quiescence period to allow time for intital rendering of the site.
//
// TODO: only re-render when a page is affected by the template change.
type OnlineRenderer struct {
	Config Config

	fs     *fsutil
	site   *Site
	reader *fsReader

	gen *generation

	timer *timer

	// Internal queues.
	errs     chan error
	messages chan string
	rq       chan rqItem

	// Filesystem updates.
	fsupdates <-chan fsUpdate
	fserrs    <-chan error
}

// Render request item.
type rqItem struct {
	page *Page       // page to render.
	gen  *generation // should be nil on first render attempt.
}

// Tracks logical tiemstamp of elements being added to a site. Used to determine
// whether a page should be re-rendered.
type generation struct {
	sync.Mutex
	// Keep track of both slug time and template time; we can do finer-grained
	// retries based on why a render failed.
	slug     int
	template int
}

// Increment the slug generation.
func (g *generation) nextSlug() {
	g.Lock()
	g.slug++
	g.Unlock()
}

// Increment the template generation.
func (g *generation) nextTemplate() {
	g.Lock()
	g.template++
	g.Unlock()
}

// Return a copy of the current generation.
func (g *generation) current() *generation {
	g.Lock()
	defer g.Unlock()
	return &generation{slug: g.slug, template: g.template}
}

// Compare two generations.
func (g *generation) gt(other *generation) bool {
	g.Lock()
	defer g.Unlock()
	other.Lock()
	defer other.Unlock()
	return g.slug > other.slug || g.template > other.template
}

// Compare slug generation.
func (g *generation) gtSlug(other *generation) bool {
	g.Lock()
	defer g.Unlock()
	other.Lock()
	defer other.Unlock()
	return g.slug > other.slug
}

// Compare tempate generation.
func (g *generation) gtTemplate(other *generation) bool {
	g.Lock()
	defer g.Unlock()
	other.Lock()
	defer other.Unlock()
	return g.template > other.template
}

// Initialize the renderer.
func (r *OnlineRenderer) init() (err error) {
	r.gen = &generation{}
	r.errs = make(chan error)
	r.messages = make(chan string)
	r.rq = make(chan rqItem)

	r.timer = newTimer(r.Config.QuiescentSecs)

	r.fs = newFS(r.Config.InputDir, r.Config.OutputDir)
	r.reader = newFsReader(r.Config.InputDir)
	if r.site, err = openSite(r.fs, RenderContext{}.funcs()); err != nil {
		return err
	}
	return nil
}

// Start rendering. Watches the filesystem for any updates and renders updates
// as they arrive. Does not return unless rendering failed to start.
func (r *OnlineRenderer) Render() (err error) {
	logging()
	// Init internal structures.
	if err = r.init(); err != nil {
		return err
	}

	// Do a single, synchronous render.
	// TODO: This is a hack; make an initial inventory of the file system before
	// rendering online.
	tr := OfflineRenderer{Config: r.Config}
	if err = tr.Render(); err != nil {
		return err
	}

	// Start reading.
	if r.fsupdates, r.fserrs, err = r.reader.Indefinitely(); err != nil {
		return err
	}

	// Forward filesystem errors to internal error channel.
	go func() {
		for err := range r.fserrs {
			r.errs <- err
		}
	}()

	// Start listening for filesystem events.
	go r.fsUpdateThread()

	// Start rendering thread.
	go r.renderThread()

	// Output errors.
	for {
		select {
		case err, ok := <-r.errs:
			if !ok {
				return nil
			}
			fmt.Println("Error:", err)
		case msg, ok := <-r.messages:
			if !ok {
				return nil
			}
			fmt.Println("Message:", msg)
		}
	}
}

// Render a single page. If the page fails to render, the page is re-enqueued to
// be retried later.
//
// Only attempts to re-render pages after some change; otherwise a rendering
// error is assumed to be permanent (until another page or template has been
// added).
func (r *OnlineRenderer) render(item rqItem) {
	p := item.page
	// First, check whether the page is still valid.
	bp, _ := r.site.PageByPath(p.FilePath)
	bs, _ := r.site.PageBySlug(p.Slug)
	if bp != bs || bp != p {
		// In either case, we have a different page now; drop this update.
		go func() {
			r.errs <- fmt.Errorf("page %s (%s) is no longer valid", p.Slug, p.FilePath)
		}()
		return
	}

	requeue := func(item rqItem) {
		// Requeue the item.
		r.rq <- item
	}

	now := r.gen.current() // snapshot current time.
	if item.gen != nil {
		// Not the first render attempt.
		if !now.gt(item.gen) {
			// No time has advanced since the last render.
			go requeue(item)
			return
		}
	}
	// Either the first render, or a new render, but render time is as of now.
	// (a little racy, since actual render time may be later, but in the worst
	// case we retry twice per generation).
	item.gen = now

	ctx := RenderContext{
		cfg:  r.Config,
		Site: r.site,
		Page: p,
	}
	b, err := ctx.RenderPage(p)
	if err != nil {
		// Assume that all errors are retriable.
		// TODO: We can do finer-grained retries here based on the error. For
		// example, it doesn't make sense to retry a template error if a new
		// template has not been added. Similarly for missing slugs, etc.
		go requeue(item)
		err = fmt.Errorf("rendering page (will retry): %v", err)
		go func() {
			r.errs <- err
		}()
	}

	go func() { r.messages <- fmt.Sprintf("rendered page %s", p.Slug) }()

	if err := r.fs.WriteFile(p.UrlPath, b); err != nil {
		// Ok to block, last op.
		r.errs <- err
	}
}

func (r *OnlineRenderer) renderThread() {
	// Pull from render queue and render.
	for p := range r.rq {
		go r.render(p)
	}
}

// Handles a create operation from the filesystem. If the file is a page,
// renders the page. If a file is a template, adds the template and optionally
// re-renders all pages after a quiesence period.
func (r *OnlineRenderer) handleCreate(update fsUpdate) {
	lg.Debugf("handling create %+v", update)
	if update.IsDir {
		// Ignore, directories are created on file write.
		return
	}

	// For templates, or files that do not need to be treated as pages, we handle
	// them immediately.
	switch fileType(update.Path) {
	case pagesType:
		ext := strings.ToLower(filepath.Ext(update.Path))
		if ext == ".md" || ext == ".html" || ext == ".htm" {
			p, err := newPageFrom(r.fs, update.Path, r.Config)
			if err != nil {
				r.errs <- err
				return
			}
			r.site.AddPage(p)

			lg.Debugf("queueing page %s", p.Slug)
			r.gen.nextSlug()
			// The first time we attempt to render, we do it as of when it reaches
			// the head of the queue.
			r.rq <- rqItem{page: p, gen: nil}
			break
		}
		// All other files we just copy directly.
		lg.Debugf("copying file %s", update.Path)
		go func() { r.messages <- fmt.Sprintf("copying file %s", update.Path) }()
		if err := r.fs.CopyFile(update.Path, outputPath(update.Path)); err != nil {
			r.errs <- err
			return
		}
	case templatesType:
		byts, err := r.fs.ReadFile(update.Path)
		if err != nil {
			r.errs <- err
			return
		}
		if err := r.site.AddTemplate(objectKey(update.Path), byts); err != nil {
			r.errs <- err
			return
		}
		go func() { r.messages <- fmt.Sprintf("added template %s", update.Path) }()
		r.gen.nextTemplate()

		if r.timer.Done() {
			// We have been quiescent, but now a template has updated. Re-render all
			// pages.
			// TODO: track dependent pages.
			for _, p := range r.site.Pages() {
				r.rq <- rqItem{page: p, gen: nil}
			}
		}
	case sgConfig:
		// nothing to do.
		break
	default:
		if update.Path[0] == '.' {
			r.messages <- fmt.Sprintf("ignoring file %s", update.Path)
		} else {
			r.errs <- fmt.Errorf("unknown document type for file %s", update.Path)
		}
	}
}

// Handles a delete operation from the filesyste.
func (r *OnlineRenderer) handleDelete(update fsUpdate) {
	lg.Debugf("handling delete %+v", update)
	path := update.Path
	if !update.IsDir {
		// If this is a page, we remove it from the site.
		pg := r.site.RemovePageByPath(update.Path)
		if pg != nil {
			// The file we should remove is the UrlPath, not the filepath, which
			// includes the prefix of the pages directory.
			path = pg.UrlPath
		}
		// TODO: Re-render any pages with cross-references.
	}
	go func() { r.messages <- fmt.Sprintf("deleting file %s", path) }()
	if err := r.fs.Remove(path); err != nil {
		r.errs <- err
	}
}

// Watch for filesystem updates and forward operations to the appropriate
// handler.
func (r *OnlineRenderer) fsUpdateThread() {
	for update := range r.fsupdates {
		switch update.Op {
		case writeOp:
			go r.handleCreate(update)
		case deleteOp:
			go r.handleDelete(update)
		}
	}
}

// Render a site in one-shot mode.
type OfflineRenderer struct {
	Config Config
}

// Renders the site and returns any error generated during rendering. Renders in
// one-shot and returns when rendering is complete.
func (r *OfflineRenderer) Render() error {
	// Grab all pages; render them.
	logging()

	fs := newFS(r.Config.InputDir, r.Config.OutputDir)

	var site *Site
	var err error
	if site, err = openSite(fs, RenderContext{}.funcs()); err != nil {
		return err
	}

	rd := newFsReader(r.Config.InputDir)
	files, errs, err := rd.Once()
	if err != nil {
		return err
	}

	// We buffer all renderable files in order to process them at once.
	var pages []string
	for file := range files {
		if file.IsDir {
			continue
		}

		path := file.Path

		switch fileType(path) {
		case pagesType:
			// We only care about files that we may need to render.
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".md" || ext == ".html" || ext == ".htm" {
				lg.Debugf("buffering renderable file %s", file.Path)
				pages = append(pages, path)
				break
			}
			// The rest of files we copy as assets.
			lg.Debugf("copying file %s", file.Path)
			if err := fs.CopyFile(path, outputPath(path)); err != nil {
				return err
			}
		case templatesType:
			byts, err := fs.ReadFile(path)
			if err != nil {
				return err
			}
			if err := site.AddTemplate(objectKey(path), byts); err != nil {
				return err
			}
		case sgConfig:
			// nothing to do.
			break
		default:
			if file.Path[0] == '.' {
				lg.Infof("ignoring file %s", file.Path)
			} else {
				return fmt.Errorf("unknown document type for file %s", file.Path)
			}
		}
	}

	var allErrs error
	for err := range errs {
		allErrs = errors.Join(allErrs, err)
	}
	if allErrs != nil {
		return allErrs
	}

	// Now we render all pages.
	for _, path := range pages {
		p, err := newPageFrom(fs, path, r.Config)
		if err != nil {
			return err
		}
		site.AddPage(p)
	}
	for _, p := range pages {
		page, err := site.PageByPath(p)
		if err != nil {
			return err
		}
		ctx := RenderContext{
			cfg:  r.Config,
			Site: site,
			Page: page,
		}
		b, err := ctx.RenderPage(page)
		if err != nil {
			return err
		}
		if err := fs.WriteFile(page.UrlPath, b); err != nil {
			return err
		}
	}

	return nil
}

func (ctx RenderContext) RenderPage(p *Page) ([]byte, error) {
	if p.Type == "raw" {
		lg.Debugf("rendering raw page %s", p.FilePath)
		// For raw pages, we allow them to use
		t, err := ctx.Site.Template("")
		cnt, err := p.RawContent()
		if err != nil {
			return nil, err
		}
		t, err = t.Funcs(ctx.funcs()).Parse(string(cnt))
		if err != nil {
			return nil, err
		}
		b, err := renderTemplate(t, ctx)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	lg.Debugf("rendering templated page %s (%s)", p.FilePath, p.Type)
	t, err := ctx.Site.Template(p.Type)
	if err != nil {
		return nil, err
	}
	t = t.Funcs(ctx.funcs())
	return renderTemplate(t, ctx)
}

// Context within which a page is rendered and defines top-level functions that
// can be called from the template.
type RenderContext struct {
	cfg  Config
	Site *Site
	Page *Page
}

// All pages from the site.
func (ctx RenderContext) Pages() []*Page {
	return ctx.Site.Pages()
}

// All tags from the site.
func (ctx RenderContext) Tags() []string {
	return ctx.Site.Tags()
}

func (ctx RenderContext) funcs() template.FuncMap {
	return template.FuncMap{
		"url":  ctx.SlugURL,
		"page": ctx.PageReference,
	}
}

// URL for a given slug.
func (ctx RenderContext) SlugURL(slug string) (string, error) {
	var u string
	if p, err := ctx.Site.PageBySlug(slug); err != nil {
		return "", err
	} else {
		u, _ = url.JoinPath(ctx.Site.RootUrl, p.UrlPath)
		if ctx.cfg.UseLocalRootUrl {
			u, _ = url.JoinPath(localRootUrl, p.UrlPath)
		}
		lg.Debugf("slug url %s: %s", slug, u)
		return u, nil
	}
}

// Get a page, given a slug.
func (ctx RenderContext) PageReference(slug string) (*Page, error) {
	return ctx.Site.PageBySlug(slug)
}

func renderTemplate(t *template.Template, ctx any) ([]byte, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, ctx); err != nil {
		lg.Errorf("error rendering template %s: %v", t.Name(), err)
		return nil, err
	}
	return b.Bytes(), nil
}

// basic timer to track quiescence by time since render start.
// timer resets with Reset() and starts at first call to Done().
type timer struct {
	sync.Mutex

	t time.Duration
	d bool
	l time.Time
}

func newTimer(timeoutSecs int) *timer {
	return &timer{t: time.Duration(timeoutSecs) * time.Second}
}

func (r *timer) Done() bool {
	r.Lock()
	defer r.Unlock()
	if r.l.IsZero() {
		r.l = time.Now()
	}

	if !r.d && time.Since(r.l) > r.t {
		r.d = true
	}
	r.l = time.Now()
	return r.d
}

func (r *timer) Reset() {
	r.Lock()
	defer r.Unlock()
	r.d = false
	r.l = time.Time{}
}
