package sg

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
	"go.abhg.dev/goldmark/mermaid"
)

var (
	// Rendering errors
	ErrTemplateNotFound = errors.New("template not found")
	ErrSlugNotFound     = errors.New("slug not found")
	ErrPathNotFound     = errors.New("path not found")
	ErrTagNotFound      = errors.New("tag not found")
	ErrTypeNotFound     = errors.New("type not found")

	// Page validation/parsing errors.
	ErrMissingSlug        = errors.New("invalid slug")
	ErrMissingType        = errors.New("invalid type")
	ErrMissingPath        = errors.New("missing path")
	ErrMissingTitle       = errors.New("missing title")
	ErrMissingFrontmatter = errors.New("missing frontmatter")

	// Frontmatter delimiter
	FrontmatterDelimiter = []byte("---")

	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Typographer,
			extension.TaskList,
			extension.Linkify,
			&anchor.Extender{},
			&mermaid.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
	)
)

// Config for sites.
type Config struct {
	InputDir        string
	OutputDir       string
	UseLocalRootUrl bool // if true, use http://localhost:8080 as the root url
	QuiescentSecs   int  // period to wait before re-rendering pages on a template change
}

type documentType string

const (
	pagesType     = documentType("pages")
	templatesType = documentType("templates")
	sgConfig      = documentType("sg.yaml")
	localRootUrl  = "http://localhost:8080"
)

// Given a path, return the type of document.
func fileType(path string) documentType {
	if strings.HasPrefix(path, string(pagesType)) {
		return pagesType
	}
	if strings.HasPrefix(path, string(templatesType)) {
		return templatesType
	}
	if strings.HasPrefix(path, string(sgConfig)) {
		return sgConfig
	}
	return ""
}

// For a given path, return a key that can be used to look up the object in the
// site.
func objectKey(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// For a given path, return the output path to use.
func outputPath(path string) string {
	return strings.TrimPrefix(path, slashify(string(fileType(path))))
}

// A Page.
type Page struct {
	// Required fields.
	FilePath  string // filename relative to the input directory
	UrlPath   string // url relative to the site root
	Slug      string `yaml:"slug"`
	Title     string `yaml:"title"`
	Type      string `yaml:"type"`
	IsRaw     bool
	OrderHint int

	// Fields for markdown pages.
	Draft    bool      `yaml:"draft"`
	Date     time.Time `yaml:"date"`
	Tags     []string  `yaml:"tags"`
	Ordering string    `yaml:"ordering"`

	contentIndex int     // index of content
	fs           *fsutil // fs to read page from
}

// A Site.
type Site struct {
	sync.Mutex

	Title   string `yaml:"title"`
	RootUrl string `yaml:"root_url"`

	templates *template.Template
	bySlug    map[string]*Page
	byPath    map[string]*Page
	byTag     map[string]*Set[*Page]
	byType    map[string]*Set[*Page]

	tags []string
}

// Create a new site, using the given funcmap for rendering templates.
func newSite(funcs template.FuncMap) *Site {
	s := &Site{}
	s.templates = template.New("").Funcs(funcs)
	s.bySlug = make(map[string]*Page)
	s.byPath = make(map[string]*Page)
	s.byTag = make(map[string]*Set[*Page])
	s.byType = make(map[string]*Set[*Page])
	return s
}

// Open a site rooted at fs, using the given funcmap for rendering templates.
func openSite(fs *fsutil, funcs template.FuncMap) (*Site, error) {
	s := newSite(funcs)
	if err := s.readConfig(fs); err != nil {
		return nil, err
	}
	return s, nil
}

// Read site config and initialize the site.
func (s *Site) readConfig(fs *fsutil) error {
	byts, err := fs.ReadFile(string(sgConfig))
	lg.Debugf("Read %s", string(byts))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(byts, s); err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}
	lg.Debugf("title: %s; rootUrl: %s", s.Title, s.RootUrl)
	return nil
}

// Look up the tempate associated with the given key.
func (s *Site) Template(key string) (*template.Template, error) {
	s.Lock()
	defer s.Unlock()
	if key == "" {
		lg.Debugf("New template")
		if t, err := s.templates.Clone(); err != nil {
			return nil, err
		} else {
			return t.New(key), nil
		}
	}

	lg.Debugf("Looking up template %s", key)
	if t := s.templates.Lookup(key); t != nil {
		return t.Clone()
	}
	return nil, ErrTemplateNotFound
}

// Lookup page by slug.
func (s *Site) PageBySlug(slug string) (*Page, error) {
	s.Lock()
	defer s.Unlock()
	if p, ok := s.bySlug[slug]; ok {
		return p, nil
	}
	return nil, ErrSlugNotFound
}

// Lookup page by input path.
func (s *Site) PageByPath(path string) (*Page, error) {
	s.Lock()
	defer s.Unlock()
	if p, ok := s.byPath[path]; ok {
		return p, nil
	}
	return nil, ErrPathNotFound
}

// Get all pages with the given tag.
func (s *Site) Tag(tag string) ([]*Page, error) {
	s.Lock()
	defer s.Unlock()
	if p, ok := s.byTag[tag]; ok {
		t := p.Slice()
		slices.SortFunc(t, comparePages)
		return t, nil
	}
	return []*Page{}, ErrTagNotFound
}

// Get all pages with the given type.
func (s *Site) Type(typ string) ([]*Page, error) {
	s.Lock()
	defer s.Unlock()
	if p, ok := s.byType[typ]; ok {
		t := p.Slice()
		slices.SortFunc(t, comparePages)
		return t, nil
	}
	return []*Page{}, ErrTypeNotFound
}

// Remove all cross-references to the given page.
func (s *Site) cleanPageRefs(p *Page) {
	tags := NewSet[string]()
	typs := NewSet[string]()
	bs, ok := s.bySlug[p.Slug]
	if ok {
		delete(s.bySlug, p.Slug)
		tags.AddAll(bs.Tags...)
		typs.Add(bs.Type)
	}
	bp, ok := s.byPath[p.FilePath]
	if ok {
		delete(s.byPath, p.FilePath)
		tags.AddAll(bp.Tags...)
		typs.Add(bs.Type)
	}
	for tag := range tags.Elements() {
		s.byTag[tag].Remove(p)
		if s.byTag[tag].Size() == 0 {
			delete(s.byTag, tag)
		}
	}

	if typs.Size() > 1 {
		lg.Warnf("Removing page %s from multiple types %v", p.Slug, typs)
	}
	for typ := range typs.Elements() {
		s.byType[typ].Remove(p)
		if s.byType[typ].Size() == 0 {
			delete(s.byType, typ)
		}
	}
}

// Add a page to the site.
func (s *Site) AddPage(p *Page) {
	s.Lock()
	defer s.Unlock()
	lg.Debugf("Adding page %s (%s)", p.Slug, p.FilePath)
	s.cleanPageRefs(p)
	s.bySlug[p.Slug] = p
	s.byPath[p.FilePath] = p
	if s.byType[p.Type] == nil {
		s.byType[p.Type] = NewSet[*Page]()
	}
	s.byType[p.Type].Add(p)
	for _, tag := range p.Tags {
		if s.byTag[tag] == nil {
			s.byTag[tag] = NewSet[*Page]()
			s.tags = append(s.tags, tag)
		}
		s.byTag[tag].Add(p)
	}
}

// Remove the page with the given path and return the removed page. If page does
// not exist, returns nil.
func (s *Site) RemovePageByPath(path string) *Page {
	s.Lock()
	defer s.Unlock()
	if p, ok := s.byPath[path]; ok {
		s.cleanPageRefs(p)
		return p
	}
	return nil
}

// Retrieve all pages.
func (s *Site) Pages() []*Page {
	s.Lock()
	defer s.Unlock()
	ret := make([]*Page, 0, len(s.bySlug))
	for _, p := range s.bySlug {
		ret = append(ret, p)
	}
	lg.Debugf("Getting %n pages", len(ret))
	return ret
}

// Retrieve all tags.
func (s *Site) Tags() []string {
	s.Lock()
	defer s.Unlock()
	return s.tags
}

// Add a template keyed by the given string.
func (s *Site) AddTemplate(key string, byts []byte) error {
	s.Lock()
	defer s.Unlock()
	lg.Debugf("Adding template %s", key)
	_, err := s.templates.New(key).Parse(string(byts))
	return err
}

// Create a page from the given path and rooted filesystem. File should exist at
// call time, as page metadata is parsed from the file.
func newPageFrom(fs *fsutil, path string, _ Config) (*Page, error) {
	var pg *Page
	var err error
	if filepath.Ext(path) == ".md" {
		pg, err = newMarkdownPage(fs, path)
	} else {
		pg, err = newRawPage(fs, path)
	}
	if pg != nil {
		pg.UrlPath = outputPath(pg.UrlPath)
		pg.fs = fs
	}
	return pg, err
}

// Create a raw page from the given path.
func newRawPage(_ *fsutil, path string) (*Page, error) {
	page := &Page{
		FilePath: path,
		UrlPath:  path,
		Slug:     path,
		Title:    path,
		Type:     "raw",
	}
	return page, page.validate()
}

// Create a markdown page from the given path. Parses the markdown frontmatter.
func newMarkdownPage(fs *fsutil, path string) (*Page, error) {
	fmlen := len(FrontmatterDelimiter)
	page := &Page{
		FilePath: path,
		UrlPath:  strings.TrimSuffix(path, filepath.Ext(path)) + ".html",
	}
	f, err := fs.Read(path)
	if err != nil {
		return nil, err
	}
	byts := make([]byte, 1024)
	lg.Debugf("byts %v", len(byts))
	if _, err := f.Read(byts); err != nil {
		if !bytes.HasPrefix(byts, FrontmatterDelimiter) {
			return nil, ErrMissingFrontmatter
		}
	}
	idx := bytes.Index(byts[fmlen:], FrontmatterDelimiter)
	offset := 0
	for idx < 0 {
		byts = append(byts, make([]byte, 1024)...)
		offset += 1024
		if _, err := f.Read(byts[offset:]); err != nil {
			return nil, ErrMissingFrontmatter
		}
		idx = bytes.Index(byts[offset:], FrontmatterDelimiter)
	}

	page.contentIndex = idx + 2*fmlen
	front := byts[fmlen : fmlen+idx]
	if err := yaml.Unmarshal(front, page); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}
	return page, page.validate()
}

// Get the raw content from the file. Reads the file each call.
func (p *Page) RawContent() ([]byte, error) {
	return p.fs.ReadFile(p.FilePath)
}

// Return the content of the page, converted to HTML. Reads page from
// filesystem.
func (p *Page) Content() (template.HTML, error) {
	var b []byte
	var err error
	if b, err = p.RawContent(); err != nil {
		return "", err
	}
	if p.IsRaw {
		return template.HTML(string(b)), nil
	}
	b = b[p.contentIndex:]
	var buf bytes.Buffer
	if err := md.Convert(b, &buf); err != nil {
		return "", fmt.Errorf("error converting markdown for %s: %w", p.FilePath, err)
	}
	return template.HTML(buf.String()), nil
}

// Validate page metadata.
func (p *Page) validate() error {
	if p.Slug == "" {
		return ErrMissingSlug
	}
	if p.Type == "" {
		return ErrMissingType
	}
	if p.FilePath == "" {
		return ErrMissingPath
	}
	if p.Title == "" {
		return ErrMissingTitle
	}
	if p.Ordering != "" {
		var err error
		if p.OrderHint, err = strconv.Atoi(p.Ordering); err != nil {
			return err
		}
	}
	return nil
}

// Ordering for pages.
// Pages are ordered according to the following rules:
// - if orderint is used, the page is ordered by the orderhint.
// - if orderhint is not used, the page is ordered by the date.
// - otherwise, the page is ordered by the slug.
func comparePages(a, b *Page) int {
	if a.Ordering != "" || b.Ordering != "" {
		return compareOrdering(a, b)
	} else if !a.Date.IsZero() || !b.Date.IsZero() {
		return compareDate(a, b)
	} else {
		return compareSlug(a, b)
	}
}

func compareOrdering(a, b *Page) int {
	if a.Ordering == "" {
		// a has no ordering, so it is always greater.
		return 1
	}
	if b.Ordering == "" {
		// b has no ordering, so it is always greater.
		return -1
	}
	return a.OrderHint - b.OrderHint
}

func compareDate(a, b *Page) int {
	if a.Date.IsZero() {
		// a has no date, so it is always greater.
		return 1
	}
	if b.Date.IsZero() {
		// b has no date, so it is always greater.
		return -1
	}
	// Invert ordering -- most recent first.
	return int(b.Date.Sub(a.Date).Seconds())
}

func compareSlug(a, b *Page) int {
	return strings.Compare(a.Slug, b.Slug)
}
