// Package rss provides types for representing RSS 2.0 feeds.
package rss

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
)

// rssFeed wraps a Channel in an <rss version="2.0"> element.
type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	AtomNS  string   `xml:"xmlns:atom,attr,omitempty"`
	Channel Channel  `xml:"channel"`
}

var (
	// Channel validation errors.
	ErrMissingChannelTitle       = errors.New("channel: missing title")
	ErrMissingChannelLink        = errors.New("channel: missing link")
	ErrMissingChannelDescription = errors.New("channel: missing description")

	// Item validation errors.
	ErrMissingItemTitleOrDescription = errors.New("item: missing title or description")
)

// NewChannel creates a Channel with the required fields populated.
func NewChannel(title, link, description string) Channel {
	return Channel{
		Title:       title,
		Link:        link,
		Description: description,
	}
}

// NewItem creates an Item with a title and description. Per the RSS 2.0 spec,
// at least one of title or description must be non-empty.
func NewItem(title, description string) Item {
	return Item{
		Title:       title,
		Description: description,
	}
}

// NewGUID creates a GUID element. If isPermaLink is true, the value is assumed
// to be a URL that can be used as a permanent link to the item.
func NewGUID(value string, isPermaLink bool) *GUID {
	g := &GUID{Value: value}
	if !isPermaLink {
		g.IsPermaLink = "false"
	}
	return g
}

// Validate checks that all required channel fields are present and that all
// contained items are valid.
func (c Channel) Validate() error {
	if c.Title == "" {
		return ErrMissingChannelTitle
	}
	if c.Link == "" {
		return ErrMissingChannelLink
	}
	if c.Description == "" {
		return ErrMissingChannelDescription
	}
	for i, item := range c.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item[%d]: %w", i, err)
		}
	}
	return nil
}

// Validate checks that the item has at least one of Title or Description set,
// as required by the RSS 2.0 spec.
func (it Item) Validate() error {
	if it.Title == "" && it.Description == "" {
		return ErrMissingItemTitleOrDescription
	}
	return nil
}

// AddItem appends an item to the channel.
func (c *Channel) AddItem(item Item) {
	c.Items = append(c.Items, item)
}

// ToXML marshals the channel as a complete RSS 2.0 XML document.
func (c Channel) ToXML() ([]byte, error) {
	feed := rssFeed{Version: "2.0", Channel: c}
	if c.AtomLink != nil {
		feed.AtomNS = "http://www.w3.org/2005/Atom"
	}
	data, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), nil
}

// ToJSON marshals the channel as JSON.
func (c Channel) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// FromXML parses a complete RSS 2.0 XML document and returns the channel.
func FromXML(data []byte) (Channel, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return Channel{}, err
	}
	return feed.Channel, nil
}

// FromJSON parses JSON data into a Channel.
func FromJSON(data []byte) (Channel, error) {
	var ch Channel
	if err := json.Unmarshal(data, &ch); err != nil {
		return Channel{}, err
	}
	return ch, nil
}

// Channel represents an RSS 2.0 channel element containing metadata about the
// feed and its contents.
type Channel struct {
	XMLName xml.Name `xml:"channel" json:"-"`
	// Title is the name of the channel.
	Title string `xml:"title" json:"title"`
	// AtomLink is a self-referencing link using the Atom namespace, recommended
	// for interoperability with feed readers. Must precede Link in the struct
	// so Go's encoding/xml can distinguish namespaced and plain <link> elements
	// during unmarshaling.
	AtomLink *AtomLink `xml:"http://www.w3.org/2005/Atom link,omitempty" json:"atomLink,omitempty"`
	// Link is the URL to the HTML website corresponding to the channel.
	Link string `xml:"link" json:"link"`
	// Description is a phrase or sentence describing the channel.
	Description string `xml:"description" json:"description"`
	// Language is the language the channel is written in (e.g. "en-us").
	Language string `xml:"language,omitempty" json:"language,omitempty"`
	// Copyright is the copyright notice for content in the channel.
	Copyright string `xml:"copyright,omitempty" json:"copyright,omitempty"`
	// ManagingEditor is the email address for the person responsible for editorial content.
	ManagingEditor string `xml:"managingEditor,omitempty" json:"managingEditor,omitempty"`
	// WebMaster is the email address for the person responsible for technical issues.
	WebMaster string `xml:"webMaster,omitempty" json:"webMaster,omitempty"`
	// PubDate is the publication date for the content in the channel (RFC 822).
	PubDate string `xml:"pubDate,omitempty" json:"pubDate,omitempty"`
	// LastBuildDate is the last time the content of the channel changed (RFC 822).
	LastBuildDate string `xml:"lastBuildDate,omitempty" json:"lastBuildDate,omitempty"`
	// Category specifies one or more categories that the channel belongs to.
	Category []Category `xml:"category,omitempty" json:"category,omitempty"`
	// Generator is a string indicating the program used to generate the channel.
	Generator string `xml:"generator,omitempty" json:"generator,omitempty"`
	// Docs is a URL that points to the documentation for the RSS format used.
	Docs string `xml:"docs,omitempty" json:"docs,omitempty"`
	// Cloud allows processes to register with a cloud to be notified of channel updates.
	Cloud *Cloud `xml:"cloud,omitempty" json:"cloud,omitempty"`
	// TTL is the number of minutes the channel can be cached before refreshing.
	TTL int `xml:"ttl,omitempty" json:"ttl,omitempty"`
	// Image specifies a GIF, JPEG, or PNG image that can be displayed with the channel.
	Image *Image `xml:"image,omitempty" json:"image,omitempty"`
	// Rating is the PICS rating for the channel.
	Rating string `xml:"rating,omitempty" json:"rating,omitempty"`
	// TextInput specifies a text input box that can be displayed with the channel.
	TextInput *TextInput `xml:"textInput,omitempty" json:"textInput,omitempty"`
	// SkipHours is a hint for aggregators telling them which hours they can skip.
	SkipHours *SkipHours `xml:"skipHours,omitempty" json:"skipHours,omitempty"`
	// SkipDays is a hint for aggregators telling them which days they can skip.
	SkipDays *SkipDays `xml:"skipDays,omitempty" json:"skipDays,omitempty"`
	// Items is the list of items contained in the channel.
	Items []Item `xml:"item,omitempty" json:"items,omitempty"`
}

// Item represents an RSS 2.0 item element. An item may represent a story,
// blog post, or other content. At least one of Title or Description must
// be present.
type Item struct {
	XMLName xml.Name `xml:"item" json:"-"`
	// Title is the title of the item.
	Title string `xml:"title,omitempty" json:"title,omitempty"`
	// Link is the URL of the item.
	Link string `xml:"link,omitempty" json:"link,omitempty"`
	// Description is the item synopsis.
	Description string `xml:"description,omitempty" json:"description,omitempty"`
	// Author is the email address of the author of the item.
	Author string `xml:"author,omitempty" json:"author,omitempty"`
	// Category includes the item in one or more categories.
	Category []Category `xml:"category,omitempty" json:"category,omitempty"`
	// Comments is the URL of a page for comments relating to the item.
	Comments string `xml:"comments,omitempty" json:"comments,omitempty"`
	// Enclosure describes a media object that is attached to the item.
	Enclosure *Enclosure `xml:"enclosure,omitempty" json:"enclosure,omitempty"`
	// GUID is a string that uniquely identifies the item.
	GUID *GUID `xml:"guid,omitempty" json:"guid,omitempty"`
	// PubDate indicates when the item was published (RFC 822).
	PubDate string `xml:"pubDate,omitempty" json:"pubDate,omitempty"`
	// Source is the RSS channel that the item came from.
	Source *Source `xml:"source,omitempty" json:"source,omitempty"`
}

// Category represents a category element with an optional domain attribute.
type Category struct {
	// Domain is a string that identifies a categorization taxonomy.
	Domain string `xml:"domain,attr,omitempty" json:"domain,omitempty"`
	// Value is the category name, a forward-slash-separated hierarchic location.
	Value string `xml:",chardata" json:"value"`
}

// Cloud specifies a web service that supports the rssCloud interface for
// lightweight publish-subscribe notifications.
type Cloud struct {
	// Domain is the hostname of the cloud service.
	Domain string `xml:"domain,attr" json:"domain"`
	// Port is the port number of the cloud service.
	Port int `xml:"port,attr" json:"port"`
	// Path is the path to the cloud service endpoint.
	Path string `xml:"path,attr" json:"path"`
	// RegisterProcedure is the name of the procedure to call for notification registration.
	RegisterProcedure string `xml:"registerProcedure,attr" json:"registerProcedure"`
	// Protocol is the protocol used to communicate with the cloud service (e.g. "xml-rpc", "soap", "http-post").
	Protocol string `xml:"protocol,attr" json:"protocol"`
}

// Image specifies a GIF, JPEG, or PNG image that can be displayed with a channel.
type Image struct {
	// URL is the URL of the image.
	URL string `xml:"url" json:"url"`
	// Title describes the image, used in the ALT attribute of the HTML img tag.
	Title string `xml:"title" json:"title"`
	// Link is the URL of the site; the image acts as a link to this URL.
	Link string `xml:"link" json:"link"`
	// Width is the width of the image in pixels (max 144, default 88).
	Width int `xml:"width,omitempty" json:"width,omitempty"`
	// Height is the height of the image in pixels (max 400, default 31).
	Height int `xml:"height,omitempty" json:"height,omitempty"`
	// Description is text included in the TITLE attribute of the link around the image.
	Description string `xml:"description,omitempty" json:"description,omitempty"`
}

// TextInput specifies a text input box that can be displayed with the channel.
type TextInput struct {
	// Title is the label of the Submit button in the text input area.
	Title string `xml:"title" json:"title"`
	// Description explains the text input area.
	Description string `xml:"description" json:"description"`
	// Name is the name of the text object in the text input area.
	Name string `xml:"name" json:"name"`
	// Link is the URL of the CGI script that processes text input requests.
	Link string `xml:"link" json:"link"`
}

// Enclosure describes a media object that is attached to an item.
type Enclosure struct {
	// URL is the location of the enclosure.
	URL string `xml:"url,attr" json:"url"`
	// Length is the size of the enclosure in bytes.
	Length int64 `xml:"length,attr" json:"length"`
	// Type is the MIME type of the enclosure.
	Type string `xml:"type,attr" json:"type"`
}

// GUID is a globally unique identifier for an item.
type GUID struct {
	// IsPermaLink indicates whether the GUID is a permalink (defaults to "true" if omitted).
	IsPermaLink string `xml:"isPermaLink,attr,omitempty" json:"isPermaLink,omitempty"`
	// Value is the unique identifier string.
	Value string `xml:",chardata" json:"value"`
}

// Source identifies the RSS channel that an item came from.
type Source struct {
	// URL is the URL of the source RSS feed.
	URL string `xml:"url,attr" json:"url"`
	// Value is the name of the source channel.
	Value string `xml:",chardata" json:"value"`
}

// AtomLink represents an Atom link element used for self-referencing feeds.
type AtomLink struct {
	Href string `xml:"href,attr" json:"href"`
	Rel  string `xml:"rel,attr" json:"rel"`
	Type string `xml:"type,attr" json:"type"`
}

// MarshalXML writes an <atom:link> element using the conventional namespace
// prefix rather than Go's default inline xmlns declaration.
func (a AtomLink) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "atom:link"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "href"}, Value: a.Href},
		{Name: xml.Name{Local: "rel"}, Value: a.Rel},
		{Name: xml.Name{Local: "type"}, Value: a.Type},
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// SkipHours wraps a list of hours for proper XML marshaling. Using a struct
// avoids Go's encoding/xml emitting empty <skipHours> wrapper elements.
type SkipHours struct {
	Hours []int `xml:"hour" json:"hours"`
}

// SkipDays wraps a list of days for proper XML marshaling. Using a struct
// avoids Go's encoding/xml emitting empty <skipDays> wrapper elements.
type SkipDays struct {
	Days []string `xml:"day" json:"days"`
}
