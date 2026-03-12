package rss

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func fullChannel() Channel {
	return Channel{
		Title:          "Horrible Birds Weekly",
		Link:           "https://horriblebirds.com",
		Description:    "Documenting birds that have wronged us.",
		Language:       "en-us",
		Copyright:      "Copyright 2026, Horrible Birds Ltd. The birds won't respect it either.",
		ManagingEditor: "cranky_goose@horriblebirds.com (Bill Gander)",
		WebMaster:      "furious_pelican@horriblebirds.com (Karen Gullet)",
		PubDate:        "Thu, 12 Mar 2026 12:00:00 GMT",
		LastBuildDate:  "Thu, 12 Mar 2026 13:00:00 GMT",
		Category: []Category{
			{Value: "Ornithology"},
			{Domain: "http://horriblebirds.com/tags", Value: "Complaints"},
		},
		Generator: "sg/1.0",
		Docs:      "https://cyber.harvard.edu/rss/rss.html",
		Cloud: &Cloud{
			Domain:            "rpc.horriblebirds.com",
			Port:              80,
			Path:              "/RPC2",
			RegisterProcedure: "squawk.notify",
			Protocol:          "xml-rpc",
		},
		TTL: 60,
		Image: &Image{
			URL:         "https://horriblebirds.com/angry-goose.png",
			Title:       "Horrible Birds Weekly",
			Link:        "https://horriblebirds.com",
			Width:       88,
			Height:      31,
			Description: "A goose. It knows what it did.",
		},
		Rating: "(PICS-1.1 \"http://www.classify.org/safesurf/\" 1 r (SS~~000 1))",
		TextInput: &TextInput{
			Title:       "Report a Bird",
			Description: "File a formal grievance against a bird",
			Name:        "q",
			Link:        "https://horriblebirds.com/report",
		},
		SkipHours: &SkipHours{Hours: []int{0, 1, 2}},
		SkipDays:  &SkipDays{Days: []string{"Saturday", "Sunday"}},
		Items: []Item{
			{
				Title:       "Area Goose Maintains Eye Contact for Entire Lunch",
				Link:        "https://horriblebirds.com/articles/goose-eye-contact",
				Description: "Witnesses report the goose did not blink. Not once. The sandwich was eventually surrendered.",
				Author:      "cranky_goose@horriblebirds.com (Bill Gander)",
				Category: []Category{
					{Value: "Geese"},
					{Domain: "http://horriblebirds.com/tags", Value: "Intimidation"},
				},
				Comments: "https://horriblebirds.com/articles/goose-eye-contact/comments",
				Enclosure: &Enclosure{
					URL:    "https://horriblebirds.com/audio/goose-honk-compilation.mp3",
					Length: 12345678,
					Type:   "audio/mpeg",
				},
				GUID: &GUID{
					IsPermaLink: "true",
					Value:       "https://horriblebirds.com/articles/goose-eye-contact",
				},
				PubDate: "Wed, 11 Mar 2026 10:00:00 GMT",
				Source: &Source{
					URL:   "https://waterfowlwatch.org/feed.xml",
					Value: "Waterfowl Watch",
				},
			},
			{
				Title:       "Magpie Reportedly Aware of What It Is Doing",
				Link:        "https://horriblebirds.com/articles/aware-magpie",
				Description: "Researchers have concluded, after a four-year study, that the magpie is fully aware of its behavior and has chosen to continue.",
				GUID: &GUID{
					IsPermaLink: "false",
					Value:       "aware-magpie-2026",
				},
				PubDate: "Thu, 12 Mar 2026 08:30:00 GMT",
			},
		},
	}
}

func marshalFeed(t *testing.T, ch Channel) string {
	t.Helper()
	data, err := ch.ToXML()
	if err != nil {
		t.Fatalf("ToXML: %v", err)
	}
	return string(data)
}

func parseFeed(t *testing.T, xmlStr string) *gofeed.Feed {
	t.Helper()
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(xmlStr)
	if err != nil {
		t.Fatalf("gofeed.ParseString: %v\nXML:\n%s", err, xmlStr)
	}
	return feed
}

func TestMarshalAndParseFullFeed(t *testing.T) {
	ch := fullChannel()
	xmlStr := marshalFeed(t, ch)
	feed := parseFeed(t, xmlStr)

	// Channel-level required fields
	if feed.Title != ch.Title {
		t.Errorf("Title: got %q, want %q", feed.Title, ch.Title)
	}
	if feed.Link != ch.Link {
		t.Errorf("Link: got %q, want %q", feed.Link, ch.Link)
	}
	if feed.Description != ch.Description {
		t.Errorf("Description: got %q, want %q", feed.Description, ch.Description)
	}

	// Channel-level optional fields
	if feed.Language != ch.Language {
		t.Errorf("Language: got %q, want %q", feed.Language, ch.Language)
	}
	if feed.Copyright != ch.Copyright {
		t.Errorf("Copyright: got %q, want %q", feed.Copyright, ch.Copyright)
	}
	if feed.Generator != ch.Generator {
		t.Errorf("Generator: got %q, want %q", feed.Generator, ch.Generator)
	}

	if feed.Image == nil {
		t.Fatal("Image: got nil")
	}
	if feed.Image.URL != ch.Image.URL {
		t.Errorf("Image.URL: got %q, want %q", feed.Image.URL, ch.Image.URL)
	}
	if feed.Image.Title != ch.Image.Title {
		t.Errorf("Image.Title: got %q, want %q", feed.Image.Title, ch.Image.Title)
	}

	// Items
	if len(feed.Items) != len(ch.Items) {
		t.Fatalf("Items count: got %d, want %d", len(feed.Items), len(ch.Items))
	}

	item0 := feed.Items[0]
	want0 := ch.Items[0]
	if item0.Title != want0.Title {
		t.Errorf("Item[0].Title: got %q, want %q", item0.Title, want0.Title)
	}
	if item0.Link != want0.Link {
		t.Errorf("Item[0].Link: got %q, want %q", item0.Link, want0.Link)
	}
	if item0.Description != want0.Description {
		t.Errorf("Item[0].Description: got %q, want %q", item0.Description, want0.Description)
	}
	if item0.Author == nil || item0.Author.Email != "cranky_goose@horriblebirds.com" {
		t.Errorf("Item[0].Author: unexpected value %+v", item0.Author)
	}
	if item0.GUID != want0.GUID.Value {
		t.Errorf("Item[0].GUID: got %q, want %q", item0.GUID, want0.GUID.Value)
	}

	// Enclosure
	if len(item0.Enclosures) == 0 {
		t.Fatal("Item[0].Enclosures: got none")
	}
	if item0.Enclosures[0].URL != want0.Enclosure.URL {
		t.Errorf("Item[0].Enclosure.URL: got %q, want %q", item0.Enclosures[0].URL, want0.Enclosure.URL)
	}
	if item0.Enclosures[0].Type != want0.Enclosure.Type {
		t.Errorf("Item[0].Enclosure.Type: got %q, want %q", item0.Enclosures[0].Type, want0.Enclosure.Type)
	}

	// Categories on item
	if len(item0.Categories) < 2 {
		t.Errorf("Item[0].Categories: got %d, want >= 2", len(item0.Categories))
	}

	// Second item - minimal fields
	item1 := feed.Items[1]
	want1 := ch.Items[1]
	if item1.Title != want1.Title {
		t.Errorf("Item[1].Title: got %q, want %q", item1.Title, want1.Title)
	}
	if item1.GUID != want1.GUID.Value {
		t.Errorf("Item[1].GUID: got %q, want %q", item1.GUID, want1.GUID.Value)
	}
}

func TestMarshalMinimalFeed(t *testing.T) {
	ch := Channel{
		Title:       "Minimal Feed",
		Link:        "https://minimal.example.com",
		Description: "A minimal RSS feed.",
		Items: []Item{
			{Title: "Only Title"},
			{Description: "Only description."},
		},
	}
	xmlStr := marshalFeed(t, ch)
	feed := parseFeed(t, xmlStr)

	if feed.Title != ch.Title {
		t.Errorf("Title: got %q, want %q", feed.Title, ch.Title)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("Items count: got %d, want 2", len(feed.Items))
	}
	if feed.Items[0].Title != "Only Title" {
		t.Errorf("Item[0].Title: got %q, want %q", feed.Items[0].Title, "Only Title")
	}
	if feed.Items[1].Description != "Only description." {
		t.Errorf("Item[1].Description: got %q, want %q", feed.Items[1].Description, "Only description.")
	}
}

func TestMarshalProducesValidXML(t *testing.T) {
	ch := fullChannel()
	xmlStr := marshalFeed(t, ch)

	if !strings.Contains(xmlStr, `<rss version="2.0">`) {
		t.Error("missing <rss version=\"2.0\"> element")
	}
	if !strings.Contains(xmlStr, "<channel>") {
		t.Error("missing <channel> element")
	}
	if !strings.Contains(xmlStr, "<item>") {
		t.Error("missing <item> element")
	}
	if !strings.Contains(xmlStr, `<enclosure url="https://horriblebirds.com/audio/goose-honk-compilation.mp3"`) {
		t.Error("missing enclosure with url attribute")
	}
	if !strings.Contains(xmlStr, `<guid isPermaLink="true">`) {
		t.Error("missing guid with isPermaLink attribute")
	}
	if !strings.Contains(xmlStr, `<cloud domain="rpc.horriblebirds.com"`) {
		t.Error("missing cloud element with attributes")
	}
	if !strings.Contains(xmlStr, `<source url="https://waterfowlwatch.org/feed.xml">Waterfowl Watch</source>`) {
		t.Error("missing source element")
	}
}

func TestMarshalOmitsEmptySkipElements(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "A description.")
	ch.AddItem(NewItem("Item", "Description"))
	xmlStr := marshalFeed(t, ch)

	if strings.Contains(xmlStr, "<skipHours") {
		t.Error("empty skipHours should not appear in XML")
	}
	if strings.Contains(xmlStr, "<skipDays") {
		t.Error("empty skipDays should not appear in XML")
	}
}

func TestMarshalIncludesSkipElementsWhenPopulated(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "A description.")
	ch.SkipHours = &SkipHours{Hours: []int{0, 1, 2}}
	ch.SkipDays = &SkipDays{Days: []string{"Saturday", "Sunday"}}
	xmlStr := marshalFeed(t, ch)

	if !strings.Contains(xmlStr, "<skipHours>") {
		t.Error("populated skipHours should appear in XML")
	}
	if !strings.Contains(xmlStr, "<hour>0</hour>") {
		t.Error("skipHours should contain hour elements")
	}
	if !strings.Contains(xmlStr, "<skipDays>") {
		t.Error("populated skipDays should appear in XML")
	}
	if !strings.Contains(xmlStr, "<day>Saturday</day>") {
		t.Error("skipDays should contain day elements")
	}
}

func TestMarshalAtomLink(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "A description.")
	ch.AtomLink = &AtomLink{
		Href: "https://example.com/feed.xml",
		Rel:  "self",
		Type: "application/rss+xml",
	}
	xmlStr := marshalFeed(t, ch)

	if !strings.Contains(xmlStr, `xmlns:atom="http://www.w3.org/2005/Atom"`) {
		t.Error("missing atom namespace declaration")
	}
	if !strings.Contains(xmlStr, `<atom:link href="https://example.com/feed.xml"`) {
		t.Error("missing atom:link element")
	}
	if !strings.Contains(xmlStr, `rel="self"`) {
		t.Error("missing rel=self attribute on atom:link")
	}
	if !strings.Contains(xmlStr, `type="application/rss+xml"`) {
		t.Error("missing type attribute on atom:link")
	}
}

func TestMarshalOmitsAtomNamespaceWhenNoAtomLink(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "A description.")
	xmlStr := marshalFeed(t, ch)

	if strings.Contains(xmlStr, "xmlns:atom") {
		t.Error("atom namespace should not appear when AtomLink is nil")
	}
}

func TestNewGUID(t *testing.T) {
	g := NewGUID("https://example.com/post/1", true)
	if g.Value != "https://example.com/post/1" {
		t.Errorf("Value: got %q", g.Value)
	}
	if g.IsPermaLink != "" {
		t.Errorf("IsPermaLink should be empty for permalink GUIDs (defaults to true), got %q", g.IsPermaLink)
	}

	g2 := NewGUID("unique-id-123", false)
	if g2.Value != "unique-id-123" {
		t.Errorf("Value: got %q", g2.Value)
	}
	if g2.IsPermaLink != "false" {
		t.Errorf("IsPermaLink: got %q, want \"false\"", g2.IsPermaLink)
	}
}

func TestUnmarshalFeed(t *testing.T) {
	rawXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://test.example.com</link>
    <description>A test feed.</description>
    <language>en-us</language>
    <ttl>30</ttl>
    <category domain="Syndic8">1765</category>
    <skipHours>
      <hour>0</hour>
      <hour>6</hour>
    </skipHours>
    <skipDays>
      <day>Sunday</day>
    </skipDays>
    <image>
      <url>https://test.example.com/img.png</url>
      <title>Test</title>
      <link>https://test.example.com</link>
    </image>
    <item>
      <title>Test Item</title>
      <link>https://test.example.com/item/1</link>
      <description>A test item.</description>
      <guid isPermaLink="false">test-item-1</guid>
      <enclosure url="https://test.example.com/file.mp3" length="9876543" type="audio/mpeg"/>
      <category>Tech</category>
      <category domain="http://example.com/ns">Go</category>
    </item>
  </channel>
</rss>`

	ch, err := FromXML([]byte(rawXML))
	if err != nil {
		t.Fatalf("FromXML: %v", err)
	}

	if ch.Title != "Test Feed" {
		t.Errorf("Title: got %q, want %q", ch.Title, "Test Feed")
	}
	if ch.Link != "https://test.example.com" {
		t.Errorf("Link: got %q", ch.Link)
	}
	if ch.Language != "en-us" {
		t.Errorf("Language: got %q", ch.Language)
	}
	if ch.TTL != 30 {
		t.Errorf("TTL: got %d, want 30", ch.TTL)
	}
	if len(ch.Category) != 1 || ch.Category[0].Domain != "Syndic8" || ch.Category[0].Value != "1765" {
		t.Errorf("Category: got %+v", ch.Category)
	}
	if ch.SkipHours == nil || len(ch.SkipHours.Hours) != 2 || ch.SkipHours.Hours[0] != 0 || ch.SkipHours.Hours[1] != 6 {
		t.Errorf("SkipHours: got %v, want [0 6]", ch.SkipHours)
	}
	if ch.SkipDays == nil || len(ch.SkipDays.Days) != 1 || ch.SkipDays.Days[0] != "Sunday" {
		t.Errorf("SkipDays: got %v, want [Sunday]", ch.SkipDays)
	}
	if ch.Image == nil || ch.Image.URL != "https://test.example.com/img.png" {
		t.Errorf("Image: got %+v", ch.Image)
	}

	if len(ch.Items) != 1 {
		t.Fatalf("Items count: got %d, want 1", len(ch.Items))
	}
	item := ch.Items[0]
	if item.Title != "Test Item" {
		t.Errorf("Item.Title: got %q", item.Title)
	}
	if item.GUID == nil || item.GUID.Value != "test-item-1" || item.GUID.IsPermaLink != "false" {
		t.Errorf("Item.GUID: got %+v", item.GUID)
	}
	if item.Enclosure == nil || item.Enclosure.URL != "https://test.example.com/file.mp3" || item.Enclosure.Length != 9876543 {
		t.Errorf("Item.Enclosure: got %+v", item.Enclosure)
	}
	if len(item.Category) != 2 {
		t.Errorf("Item.Category count: got %d, want 2", len(item.Category))
	}
	if item.Category[0].Value != "Tech" || item.Category[0].Domain != "" {
		t.Errorf("Item.Category[0]: got %+v", item.Category[0])
	}
	if item.Category[1].Value != "Go" || item.Category[1].Domain != "http://example.com/ns" {
		t.Errorf("Item.Category[1]: got %+v", item.Category[1])
	}
}

func TestRoundTrip(t *testing.T) {
	ch := fullChannel()
	data, err := ch.ToXML()
	if err != nil {
		t.Fatalf("ToXML: %v", err)
	}

	rt, err := FromXML(data)
	if err != nil {
		t.Fatalf("FromXML: %v", err)
	}

	if rt.Title != ch.Title {
		t.Errorf("Title: got %q, want %q", rt.Title, ch.Title)
	}
	if rt.Link != ch.Link {
		t.Errorf("Link: got %q, want %q", rt.Link, ch.Link)
	}
	if rt.Description != ch.Description {
		t.Errorf("Description: got %q, want %q", rt.Description, ch.Description)
	}
	if rt.TTL != ch.TTL {
		t.Errorf("TTL: got %d, want %d", rt.TTL, ch.TTL)
	}
	if rt.Cloud == nil || rt.Cloud.Domain != ch.Cloud.Domain {
		t.Errorf("Cloud.Domain: got %+v", rt.Cloud)
	}
	if rt.Cloud.Port != ch.Cloud.Port {
		t.Errorf("Cloud.Port: got %d, want %d", rt.Cloud.Port, ch.Cloud.Port)
	}
	if len(rt.Items) != len(ch.Items) {
		t.Fatalf("Items count: got %d, want %d", len(rt.Items), len(ch.Items))
	}
	if rt.Items[0].Enclosure == nil || rt.Items[0].Enclosure.Length != ch.Items[0].Enclosure.Length {
		t.Errorf("Item[0].Enclosure.Length: got %+v", rt.Items[0].Enclosure)
	}
	if rt.Items[0].Source == nil || rt.Items[0].Source.URL != ch.Items[0].Source.URL {
		t.Errorf("Item[0].Source.URL: got %+v", rt.Items[0].Source)
	}
}

func TestJSONMarshal(t *testing.T) {
	ch := fullChannel()
	data, err := ch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if result["title"] != ch.Title {
		t.Errorf("JSON title: got %v, want %q", result["title"], ch.Title)
	}
	if result["link"] != ch.Link {
		t.Errorf("JSON link: got %v, want %q", result["link"], ch.Link)
	}
	items, ok := result["items"].([]any)
	if !ok || len(items) != len(ch.Items) {
		t.Fatalf("JSON items: got %v", result["items"])
	}
	item0, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("JSON items[0]: not an object")
	}
	if item0["title"] != ch.Items[0].Title {
		t.Errorf("JSON items[0].title: got %v, want %q", item0["title"], ch.Items[0].Title)
	}
}

func TestNewChannel(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "A description.")
	if ch.Title != "Title" {
		t.Errorf("Title: got %q", ch.Title)
	}
	if ch.Link != "https://example.com" {
		t.Errorf("Link: got %q", ch.Link)
	}
	if ch.Description != "A description." {
		t.Errorf("Description: got %q", ch.Description)
	}
	if err := ch.Validate(); err != nil {
		t.Errorf("valid channel should pass Validate: %v", err)
	}
}

func TestNewItem(t *testing.T) {
	item := NewItem("A Title", "A synopsis.")
	if item.Title != "A Title" {
		t.Errorf("Title: got %q", item.Title)
	}
	if item.Description != "A synopsis." {
		t.Errorf("Description: got %q", item.Description)
	}
	if err := item.Validate(); err != nil {
		t.Errorf("valid item should pass Validate: %v", err)
	}
}

func TestNewItemTitleOnly(t *testing.T) {
	item := NewItem("Just a Title", "")
	if err := item.Validate(); err != nil {
		t.Errorf("item with only title should pass Validate: %v", err)
	}
}

func TestNewItemDescriptionOnly(t *testing.T) {
	item := NewItem("", "Just a description.")
	if err := item.Validate(); err != nil {
		t.Errorf("item with only description should pass Validate: %v", err)
	}
}

func TestChannelValidateMissingTitle(t *testing.T) {
	ch := NewChannel("", "https://example.com", "desc")
	if err := ch.Validate(); !errors.Is(err, ErrMissingChannelTitle) {
		t.Errorf("got %v, want %v", err, ErrMissingChannelTitle)
	}
}

func TestChannelValidateMissingLink(t *testing.T) {
	ch := NewChannel("Title", "", "desc")
	if err := ch.Validate(); !errors.Is(err, ErrMissingChannelLink) {
		t.Errorf("got %v, want %v", err, ErrMissingChannelLink)
	}
}

func TestChannelValidateMissingDescription(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "")
	if err := ch.Validate(); !errors.Is(err, ErrMissingChannelDescription) {
		t.Errorf("got %v, want %v", err, ErrMissingChannelDescription)
	}
}

func TestItemValidateEmpty(t *testing.T) {
	item := Item{}
	if err := item.Validate(); !errors.Is(err, ErrMissingItemTitleOrDescription) {
		t.Errorf("got %v, want %v", err, ErrMissingItemTitleOrDescription)
	}
}

func TestChannelValidatePropagatesItemError(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "desc")
	ch.AddItem(NewItem("Good Item", ""))
	ch.AddItem(Item{})
	err := ch.Validate()
	if err == nil {
		t.Fatal("expected error for invalid item")
	}
	if !errors.Is(err, ErrMissingItemTitleOrDescription) {
		t.Errorf("got %v, want wrapped %v", err, ErrMissingItemTitleOrDescription)
	}
	if !strings.Contains(err.Error(), "item[1]") {
		t.Errorf("error should identify item index: %v", err)
	}
}

func TestAddItem(t *testing.T) {
	ch := NewChannel("Title", "https://example.com", "desc")
	if len(ch.Items) != 0 {
		t.Fatalf("new channel should have 0 items, got %d", len(ch.Items))
	}
	ch.AddItem(NewItem("First", ""))
	ch.AddItem(NewItem("Second", ""))
	ch.AddItem(NewItem("", "Third description"))
	if len(ch.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(ch.Items))
	}
	if ch.Items[0].Title != "First" {
		t.Errorf("Items[0].Title: got %q", ch.Items[0].Title)
	}
	if ch.Items[2].Description != "Third description" {
		t.Errorf("Items[2].Description: got %q", ch.Items[2].Description)
	}
}

func TestFullChannelValidates(t *testing.T) {
	ch := fullChannel()
	if err := ch.Validate(); err != nil {
		t.Errorf("fullChannel should validate: %v", err)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	ch := fullChannel()
	data, err := ch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	rt, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if rt.Title != ch.Title {
		t.Errorf("Title: got %q, want %q", rt.Title, ch.Title)
	}
	if rt.Link != ch.Link {
		t.Errorf("Link: got %q, want %q", rt.Link, ch.Link)
	}
	if rt.Description != ch.Description {
		t.Errorf("Description: got %q, want %q", rt.Description, ch.Description)
	}
	if rt.TTL != ch.TTL {
		t.Errorf("TTL: got %d, want %d", rt.TTL, ch.TTL)
	}
	if len(rt.Items) != len(ch.Items) {
		t.Fatalf("Items count: got %d, want %d", len(rt.Items), len(ch.Items))
	}
	if rt.Items[0].Title != ch.Items[0].Title {
		t.Errorf("Items[0].Title: got %q, want %q", rt.Items[0].Title, ch.Items[0].Title)
	}
	if rt.Items[0].Enclosure == nil || rt.Items[0].Enclosure.URL != ch.Items[0].Enclosure.URL {
		t.Errorf("Items[0].Enclosure.URL: got %+v", rt.Items[0].Enclosure)
	}
}

func TestFromXMLInvalid(t *testing.T) {
	_, err := FromXML([]byte("not xml at all"))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestFromJSONInvalid(t *testing.T) {
	_, err := FromJSON([]byte("{invalid"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
