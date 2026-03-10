package sg

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// MsgKind describes the category of a watch-mode message.
type MsgKind int

const (
	MsgRendered      MsgKind = iota // page was rendered
	MsgCopied                       // file was copied as-is
	MsgTemplateAdded                // template was added/updated
	MsgIgnored                      // file was ignored
	MsgDeleted                      // file was deleted
)

// Message is a typed watch-mode message.
type Message struct {
	Kind MsgKind
	Text string
}

// OutputFormatter formats watch-mode messages and errors for display.
type OutputFormatter interface {
	FormatMessage(Message)
	FormatError(error)
}

type colorEntry struct {
	symbol string
	color  *color.Color
}

var colorSymbols = map[MsgKind]colorEntry{
	MsgRendered:      {" ✓ ", color.New(color.FgGreen)},
	MsgCopied:        {" → ", color.New(color.FgCyan)},
	MsgTemplateAdded: {" + ", color.New(color.FgMagenta)},
	MsgIgnored:       {" ⊘ ", color.New(color.FgWhite, color.Faint)},
	MsgDeleted:       {" − ", color.New(color.FgYellow)},
}

var plainLabels = map[MsgKind]string{
	MsgRendered:      "rendered",
	MsgCopied:        "copied",
	MsgTemplateAdded: "template added",
	MsgIgnored:       "ignored",
	MsgDeleted:       "deleted",
}

// ColorFormatter prints colored, symbol-prefixed output.
type ColorFormatter struct {
	w io.Writer
}

func NewColorFormatter() *ColorFormatter {
	return &ColorFormatter{w: os.Stdout}
}

func (f *ColorFormatter) FormatMessage(m Message) {
	if e, ok := colorSymbols[m.Kind]; ok {
		e.color.Fprintf(f.w, "%s", e.symbol)
	}
	fmt.Fprintln(f.w, m.Text)
}

func (f *ColorFormatter) FormatError(err error) {
	color.New(color.FgRed).Fprintf(f.w, " ✗ ")
	fmt.Fprintln(f.w, err)
}

// PlainFormatter prints plain-text output (no color, no symbols).
type PlainFormatter struct {
	w io.Writer
}

func NewPlainFormatter() *PlainFormatter {
	return &PlainFormatter{w: os.Stdout}
}

func (f *PlainFormatter) FormatMessage(m Message) {
	if label, ok := plainLabels[m.Kind]; ok {
		fmt.Fprintln(f.w, label, m.Text)
	} else {
		fmt.Fprintln(f.w, m.Text)
	}
}

func (f *PlainFormatter) FormatError(err error) {
	fmt.Fprintln(f.w, "error:", err)
}
