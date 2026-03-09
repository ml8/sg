package sg

import (
	"testing"
	"time"
)

func TestSlashify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"foo", "foo/"},
		{"foo/", "foo/"},
		{"", ""},
	}
	for _, tt := range tests {
		got := slashify(tt.in)
		if got != tt.want {
			t.Errorf("slashify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestObjectKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"templates/base.html", "base"},
		{"pages/about.md", "about"},
		{"pages/style.css", "style"},
	}
	for _, tt := range tests {
		got := objectKey(tt.in)
		if got != tt.want {
			t.Errorf("objectKey(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFileType(t *testing.T) {
	tests := []struct {
		in   string
		want documentType
	}{
		{"pages/foo.md", pagesType},
		{"templates/bar.html", templatesType},
		{"sg.yaml", sgConfig},
		{"unknown.txt", ""},
	}
	for _, tt := range tests {
		got := fileType(tt.in)
		if got != tt.want {
			t.Errorf("fileType(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPageValidation(t *testing.T) {
	valid := &Page{Slug: "test", Type: "post", FilePath: "pages/test.md", Title: "Test"}
	if err := valid.validate(); err != nil {
		t.Errorf("expected valid page, got error: %v", err)
	}

	missingSlug := &Page{Type: "post", FilePath: "pages/test.md", Title: "Test"}
	if err := missingSlug.validate(); err != ErrMissingSlug {
		t.Errorf("expected ErrMissingSlug, got: %v", err)
	}

	missingType := &Page{Slug: "test", FilePath: "pages/test.md", Title: "Test"}
	if err := missingType.validate(); err != ErrMissingType {
		t.Errorf("expected ErrMissingType, got: %v", err)
	}

	missingPath := &Page{Slug: "test", Type: "post", Title: "Test"}
	if err := missingPath.validate(); err != ErrMissingPath {
		t.Errorf("expected ErrMissingPath, got: %v", err)
	}

	missingTitle := &Page{Slug: "test", Type: "post", FilePath: "pages/test.md"}
	if err := missingTitle.validate(); err != ErrMissingTitle {
		t.Errorf("expected ErrMissingTitle, got: %v", err)
	}
}

func TestPageValidationOrdering(t *testing.T) {
	p := &Page{Slug: "test", Type: "post", FilePath: "p.md", Title: "T", Ordering: "5"}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.OrderHint != 5 {
		t.Errorf("expected OrderHint 5, got %d", p.OrderHint)
	}

	bad := &Page{Slug: "test", Type: "post", FilePath: "p.md", Title: "T", Ordering: "abc"}
	if err := bad.validate(); err == nil {
		t.Error("expected error for non-numeric ordering")
	}
}

func TestComparePagesOrdering(t *testing.T) {
	a := &Page{Slug: "a", Ordering: "1", OrderHint: 1}
	b := &Page{Slug: "b", Ordering: "3", OrderHint: 3}
	if comparePages(a, b) >= 0 {
		t.Error("expected a < b by ordering")
	}
}

func TestComparePagesDate(t *testing.T) {
	now := time.Now()
	a := &Page{Slug: "a", Date: now}
	b := &Page{Slug: "b", Date: now.Add(-24 * time.Hour)}
	// More recent first, so a should come before b.
	if comparePages(a, b) >= 0 {
		t.Error("expected more recent page to sort first")
	}
}

func TestComparePagesSlug(t *testing.T) {
	a := &Page{Slug: "alpha"}
	b := &Page{Slug: "beta"}
	if comparePages(a, b) >= 0 {
		t.Error("expected alpha < beta by slug")
	}
}
