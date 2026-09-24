// Package core holds the conversion model shared by the markdown package and
// its internal format implementations: the result types, the live settings the
// engine hands to every converter, and the small formatting, asset and XML
// helpers they all build on.
package core

import (
	"context"
	"encoding/base64"
	"errors"
)

var (
	ErrUnsupportedFormat = errors.New("markdown: unsupported format")
	ErrInputTooLarge     = errors.New("markdown: input is too large")
	ErrArchiveLimit      = errors.New("markdown: archive limit exceeded")
)

// StreamInfo describes an input stream.
type StreamInfo struct {
	Name      string
	Extension string
	MIMEType  string
	URL       string
}

// Result contains converted Markdown and source metadata.
type Result struct {
	Title       string
	Markdown    string
	TextContent string
	Metadata    map[string]string
	Warnings    []string
}

func (result *Result) String() string {
	if result == nil {
		return ""
	}
	if result.Markdown != "" {
		return result.Markdown
	}
	return result.TextContent
}

// TableFormat controls table output.
type TableFormat string

const (
	TableFormatMarkdown TableFormat = "markdown"
	TableFormatHTML     TableFormat = "html"
)

// Image is an image extracted from a source document.
type Image struct {
	Name     string
	MIMEType string
	AltText  string
	Data     []byte
}

// ImageHandler stores or transforms an extracted image and returns the URL to
// place in Markdown. The default handler returns a self-contained data URI.
type ImageHandler func(ctx context.Context, image Image) (string, error)

// Converter converts one or more formats.
type Converter interface {
	Supports(info StreamInfo) bool
	Convert(ctx context.Context, data []byte, info StreamInfo) (*Result, error)
}

// ExtensionProvider lets a converter advertise the extensions it handles.
type ExtensionProvider interface {
	Extensions() []string
}

// DataURIImageHandler embeds an image directly in Markdown.
func DataURIImageHandler(_ context.Context, image Image) (string, error) {
	mimeType := image.MIMEType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(image.Data), nil
}

// Settings is the live configuration shared by the engine and every built-in
// converter. Converters keep the pointer, so later option changes are visible
// to them.
type Settings struct {
	ImageHandler ImageHandler
	TableFormat  TableFormat

	MaxArchiveFileSize int64
	MaxArchiveFiles    int
	MaxArchiveDepth    int

	PDF PDFOptions

	// Convert converts nested content (archive entries, email attachments)
	// through the engine that owns this settings value.
	Convert func(ctx context.Context, data []byte, info StreamInfo) (*Result, error)
}

// ImageHandlerOrDefault returns the configured handler, or the data URI handler.
func (settings *Settings) ImageHandlerOrDefault() ImageHandler {
	if settings == nil || settings.ImageHandler == nil {
		return DataURIImageHandler
	}
	return settings.ImageHandler
}

// TableFormatValue returns the configured table format.
func (settings *Settings) TableFormatValue() TableFormat {
	if settings == nil {
		return TableFormatHTML
	}
	return settings.TableFormat
}
