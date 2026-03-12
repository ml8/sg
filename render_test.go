package sg

import (
	"testing"
	"time"

	"github.com/ml8/sg/rss"
	"github.com/mmcdole/gofeed"
)

func testSite(feedUrl, feedTag string) *Site {
	s := newSite(nil)
	s.Title = "Horrible Birds Weekly"
	s.RootUrl = "https://horriblebirds.com"
	s.Description = "Documenting birds that have wronged us."
	s.FeedUrl = feedUrl
	s.FeedTag = feedTag
	return s
}

func testPage(slug, title, desc string, date time.Time, tags []string) *Page {
	return &Page{
		Slug:        slug,
		Title:       title,
		Description: desc,
		Date:        date,
		Type:        "post",
		FilePath:    "pages/" + slug + ".md",
		UrlPath:     slug + ".html",
		Tags:        tags,
	}
}

func TestToChannel(t *testing.T) {
	site := testSite("/feed.xml", "news")
	ctx := RenderContext{Site: site}
	ch := ToChannel(ctx, site)

	if ch.Title != site.Title {
		t.Errorf("Title: got %q, want %q", ch.Title, site.Title)
	}
	if ch.Link != site.RootUrl {
		t.Errorf("Link: got %q, want %q", ch.Link, site.RootUrl)
	}
	if ch.Description != site.Description {
		t.Errorf("Description: got %q, want %q", ch.Description, site.Description)
	}
}

func TestToItem(t *testing.T) {
	site := testSite("/feed.xml", "news")
	now := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	p := testPage("goose-incident", "Goose Incident", "A goose looked at someone.", now, []string{"news"})
	site.AddPage(p)

	ctx := RenderContext{Site: site}
	item, err := ToItem(ctx, p)
	if err != nil {
		t.Fatalf("ToItem: %v", err)
	}

	if item.Title != p.Title {
		t.Errorf("Title: got %q, want %q", item.Title, p.Title)
	}
	if item.Description != p.Description {
		t.Errorf("Description: got %q, want %q", item.Description, p.Description)
	}
	wantDate := now.Format(time.RFC1123Z)
	if item.PubDate != wantDate {
		t.Errorf("PubDate: got %q, want %q", item.PubDate, wantDate)
	}
	wantLink := "https://horriblebirds.com/goose-incident.html"
	if item.Link != wantLink {
		t.Errorf("Link: got %q, want %q", item.Link, wantLink)
	}
}

func TestToItemSlugNotFound(t *testing.T) {
	site := testSite("/feed.xml", "news")
	p := testPage("nonexistent", "Ghost Bird", "", time.Now(), nil)
	// Don't add page to site — slug lookup should fail.
	ctx := RenderContext{Site: site}
	_, err := ToItem(ctx, p)
	if err == nil {
		t.Error("expected error for page not in site")
	}
}

func TestRenderFeedUnconfigured(t *testing.T) {
	tests := []struct {
		name    string
		feedUrl string
		feedTag string
	}{
		{"both empty", "", ""},
		{"url only", "/feed.xml", ""},
		{"tag only", "", "news"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := testSite(tt.feedUrl, tt.feedTag)
			ctx := RenderContext{Site: site}
			b, err := ctx.RenderFeed()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if b != nil {
				t.Error("expected nil bytes when feed not fully configured")
			}
		})
	}
}

func TestRenderFeedNoMatchingPages(t *testing.T) {
	site := testSite("/feed.xml", "news")
	// No pages with "news" tag exist.
	ctx := RenderContext{Site: site}
	b, err := ctx.RenderFeed()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if b != nil {
		t.Error("expected nil when no pages match feed tag")
	}
}

func TestRenderFeedRoundTrip(t *testing.T) {
	site := testSite("/feed.xml", "news")
	now := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	pages := []*Page{
		testPage("goose-stare", "Goose Maintains Eye Contact Through Window",
			"It has been four hours.", now, []string{"news", "geese"}),
		testPage("magpie-aware", "Magpie Reportedly Aware of What It Is Doing",
			"Researchers confirm the bird is acting with intent.",
			now.Add(-24*time.Hour), []string{"news"}),
	}
	for _, p := range pages {
		site.AddPage(p)
	}

	ctx := RenderContext{Site: site}
	b, err := ctx.RenderFeed()
	if err != nil {
		t.Fatalf("RenderFeed: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil XML")
	}

	// Parse with our own types.
	ch, err := rss.FromXML(b)
	if err != nil {
		t.Fatalf("FromXML: %v", err)
	}
	if ch.Title != site.Title {
		t.Errorf("Channel.Title: got %q, want %q", ch.Title, site.Title)
	}
	if ch.Link != site.RootUrl {
		t.Errorf("Channel.Link: got %q, want %q", ch.Link, site.RootUrl)
	}
	if ch.Description != site.Description {
		t.Errorf("Channel.Description: got %q, want %q", ch.Description, site.Description)
	}
	if len(ch.Items) != len(pages) {
		t.Fatalf("Items count: got %d, want %d", len(ch.Items), len(pages))
	}

	// Parse with third-party library.
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(b))
	if err != nil {
		t.Fatalf("gofeed parse: %v", err)
	}
	if feed.Title != site.Title {
		t.Errorf("gofeed Title: got %q, want %q", feed.Title, site.Title)
	}
	if len(feed.Items) != len(pages) {
		t.Fatalf("gofeed Items count: got %d, want %d", len(feed.Items), len(pages))
	}
	// Pages are sorted by comparePages (most recent first), so first item
	// should be "goose-stare" (newer date).
	if feed.Items[0].Title != "Goose Maintains Eye Contact Through Window" {
		t.Errorf("gofeed Items[0].Title: got %q", feed.Items[0].Title)
	}
	if feed.Items[0].Link != "https://horriblebirds.com/goose-stare.html" {
		t.Errorf("gofeed Items[0].Link: got %q", feed.Items[0].Link)
	}
	if feed.Items[1].Title != "Magpie Reportedly Aware of What It Is Doing" {
		t.Errorf("gofeed Items[1].Title: got %q", feed.Items[1].Title)
	}
}

func TestSiteValidateFeedUrlWithoutTag(t *testing.T) {
	s := &Site{Title: "T", RootUrl: "https://x.com", Description: "d", FeedUrl: "/feed.xml"}
	if err := s.validate(); err == nil {
		t.Error("expected error for feed_url without feed_tag")
	}
}

func TestSiteValidateFeedTagWithoutUrl(t *testing.T) {
	s := &Site{Title: "T", RootUrl: "https://x.com", Description: "d", FeedTag: "news"}
	if err := s.validate(); err == nil {
		t.Error("expected error for feed_tag without feed_url")
	}
}

func TestSiteValidateFeedWithoutDescription(t *testing.T) {
	s := &Site{Title: "T", RootUrl: "https://x.com", FeedUrl: "/feed.xml", FeedTag: "news"}
	if err := s.validate(); err == nil {
		t.Error("expected error for feed_url without description")
	}
}

func TestSiteValidateFeedComplete(t *testing.T) {
	s := &Site{Title: "T", RootUrl: "https://x.com", Description: "d", FeedUrl: "/feed.xml", FeedTag: "news"}
	if err := s.validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestSiteValidateNoFeed(t *testing.T) {
	s := &Site{Title: "T", RootUrl: "https://x.com"}
	if err := s.validate(); err != nil {
		t.Errorf("expected valid without feed config, got: %v", err)
	}
}
