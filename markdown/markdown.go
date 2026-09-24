// Package markdown converts common document formats to Markdown.
//
// PDF conversion is fully local: it never calls a remote service and never
// performs OCR. Pages that carry no text layer keep their embedded images, and
// pages that carry neither text nor images are rasterized as whole-page
// pictures; both are written through the configured ImageHandler, so
// WithAssetsDirectory turns them into files that Markdown links to.
//
// The format implementations live in subpackages under internal/. This package
// owns the public API: the engine, the option set and the built-in converter
// registration.
package markdown

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/scoming-dev/tools/markdown/internal/archive"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"github.com/scoming-dev/tools/markdown/internal/docx"
	"github.com/scoming-dev/tools/markdown/internal/email"
	"github.com/scoming-dev/tools/markdown/internal/imagefile"
	"github.com/scoming-dev/tools/markdown/internal/pdf"
	"github.com/scoming-dev/tools/markdown/internal/pptx"
	"github.com/scoming-dev/tools/markdown/internal/text"
	"github.com/scoming-dev/tools/markdown/internal/xlsx"
)

const (
	// defaultMaxInputSize is unlimited: the converters load the whole document
	// anyway, so a size cap only turns a large but valid file into an error.
	// Applications that accept untrusted uploads should set their own limit
	// with WithMaxInputSize.
	defaultMaxInputSize       = int64(0)
	defaultMaxArchiveFileSize = int64(32 << 20)
	defaultMaxArchiveFiles    = 512
	defaultMaxArchiveDepth    = 4
)

// Errors reported by the engine and its converters.
var (
	ErrUnsupportedFormat = core.ErrUnsupportedFormat
	ErrInputTooLarge     = core.ErrInputTooLarge
	ErrArchiveLimit      = core.ErrArchiveLimit
)

// Warnings reported through Result.Warnings.
const (
	PDFNoOCRWarning      = pdf.PDFNoOCRWarning
	PDFPageRenderWarning = pdf.PDFPageRenderWarning
)

// Public model aliases. They are the same types the internal converters use, so
// custom converters interoperate with the built-ins.
type (
	// StreamInfo describes an input stream.
	StreamInfo = core.StreamInfo
	// Result contains converted Markdown and source metadata.
	Result = core.Result
	// TableFormat controls table output.
	TableFormat = core.TableFormat
	// Image is an image extracted from a source document.
	Image = core.Image
	// ImageHandler stores or transforms an extracted image and returns the URL
	// to place in Markdown.
	ImageHandler = core.ImageHandler
	// Converter converts one or more formats.
	Converter = core.Converter
	// ExtensionProvider lets a converter advertise the extensions it handles.
	ExtensionProvider = core.ExtensionProvider
	// PDFOptions configures the built-in local PDF converter.
	PDFOptions = core.PDFOptions
	// PDFImageFormat selects the raster format for PDF pictures, including the
	// images embedded in a page. Defaults to JPEG.
	PDFImageFormat = core.PDFImageFormat
)

// Table rendering formats.
const (
	TableFormatMarkdown = core.TableFormatMarkdown
	TableFormatHTML     = core.TableFormatHTML
)

// PDF page raster formats.
const (
	PDFImageFormatPNG  = core.PDFImageFormatPNG
	PDFImageFormatJPEG = core.PDFImageFormatJPEG
)

// DataURIImageHandler embeds an image directly in Markdown.
var DataURIImageHandler = core.DataURIImageHandler

// Option configures an engine.
type Option func(*MarkItDown)

// MarkItDown converts documents to Markdown.
type MarkItDown struct {
	mu sync.RWMutex

	// Settings is the live configuration the built-in converters read.
	core.Settings

	converters       []Converter
	registeredGroups builtinGroup
	httpClient       *http.Client
	maxInputSize     int64
}

// New creates an engine with all built-in converters. This preserves the
// package's original behavior.
func New(options ...Option) *MarkItDown {
	return newMarkItDown(true, options...)
}

// NewCore creates an engine without built-in converters. Call one of the
// Register methods or pass converter group options to enable only the formats
// the application needs.
func NewCore(options ...Option) *MarkItDown {
	return newMarkItDown(false, options...)
}

// NewEmpty is an alias for NewCore.
func NewEmpty(options ...Option) *MarkItDown {
	return NewCore(options...)
}

// NewWithBuiltins is an explicit alias for New.
func NewWithBuiltins(options ...Option) *MarkItDown {
	return New(options...)
}

func newMarkItDown(registerBuiltins bool, options ...Option) *MarkItDown {
	engine := &MarkItDown{
		Settings: core.Settings{
			ImageHandler:       DataURIImageHandler,
			TableFormat:        TableFormatHTML,
			MaxArchiveFileSize: defaultMaxArchiveFileSize,
			MaxArchiveFiles:    defaultMaxArchiveFiles,
			MaxArchiveDepth:    defaultMaxArchiveDepth,
			PDF:                core.NormalizePDFOptions(PDFOptions{}),
		},
		httpClient:   http.DefaultClient,
		maxInputSize: defaultMaxInputSize,
	}
	// Converters recurse back through the engine for archive entries and email
	// attachments.
	engine.Settings.Convert = engine.convertData
	if registerBuiltins {
		engine.RegisterBuiltins()
	}
	for _, option := range options {
		if option != nil {
			option(engine)
		}
	}
	return engine
}

func WithHTTPClient(client *http.Client) Option {
	return func(engine *MarkItDown) {
		if client != nil {
			engine.httpClient = client
		}
	}
}

func WithImageHandler(handler ImageHandler) Option {
	return func(engine *MarkItDown) {
		if handler != nil {
			engine.Settings.ImageHandler = handler
		}
	}
}

// WithTableFormat selects the table rendering used by the built-in converters.
func WithTableFormat(format TableFormat) Option {
	return func(engine *MarkItDown) {
		if format == TableFormatHTML {
			engine.Settings.TableFormat = TableFormatHTML
		} else {
			engine.Settings.TableFormat = TableFormatMarkdown
		}
	}
}

// WithPDFOptions configures the built-in local PDF converter. The zero value
// keeps every default.
func WithPDFOptions(options PDFOptions) Option {
	return func(engine *MarkItDown) {
		engine.Settings.PDF = core.NormalizePDFOptions(options)
	}
}

func WithMaxInputSize(size int64) Option {
	return func(engine *MarkItDown) { engine.maxInputSize = size }
}

func WithArchiveLimits(files int, fileSize int64, depth int) Option {
	return func(engine *MarkItDown) {
		if files > 0 {
			engine.Settings.MaxArchiveFiles = files
		}
		if fileSize > 0 {
			engine.Settings.MaxArchiveFileSize = fileSize
		}
		if depth > 0 {
			engine.Settings.MaxArchiveDepth = depth
		}
	}
}

func WithConverter(converter Converter) Option {
	return func(engine *MarkItDown) { engine.Register(converter) }
}

func (engine *MarkItDown) Register(converter Converter) {
	if engine == nil || converter == nil {
		return
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.converters = append([]Converter{converter}, engine.converters...)
}

func (engine *MarkItDown) SupportedExtensions() []string {
	if engine == nil {
		return nil
	}
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, converter := range engine.converters {
		provider, ok := converter.(ExtensionProvider)
		if !ok {
			continue
		}
		for _, extension := range provider.Extensions() {
			seen[normalizeExtension(extension)] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for extension := range seen {
		if extension != "" {
			result = append(result, extension)
		}
	}
	sort.Strings(result)
	return result
}

func (engine *MarkItDown) Convert(ctx context.Context, source string) (*Result, error) {
	if engine == nil {
		engine = New()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := url.Parse(source)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return engine.convertURL(ctx, source, parsed)
	}
	return engine.convertFile(ctx, source)
}

func (engine *MarkItDown) ConvertReader(ctx context.Context, reader io.Reader, info StreamInfo) (*Result, error) {
	if engine == nil {
		engine = New()
	}
	if reader == nil {
		return nil, errors.New("markdown: reader is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	data, err := core.ReadLimited(reader, engine.maxInputSize)
	if err != nil {
		return nil, err
	}
	return engine.convertData(ctx, data, normalizeInfo(info))
}

func Convert(source string) (*Result, error) {
	return New().Convert(context.Background(), source)
}

func ConvertReader(reader io.Reader, info StreamInfo) (*Result, error) {
	return New().ConvertReader(context.Background(), reader, info)
}

func (engine *MarkItDown) convertFile(ctx context.Context, path string) (*Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("markdown: open %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("markdown: %q is a directory", path)
	}
	if engine.maxInputSize > 0 && info.Size() > engine.maxInputSize {
		return nil, fmt.Errorf("%w: %d bytes exceeds the %d byte limit set by WithMaxInputSize", ErrInputTooLarge, info.Size(), engine.maxInputSize)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return engine.ConvertReader(ctx, file, StreamInfo{Name: filepath.Base(path)})
}

func (engine *MarkItDown) convertURL(ctx context.Context, source string, parsed *url.URL) (*Result, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}
	response, err := engine.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("markdown: download %q: %w", source, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("markdown: download %q: http status %s", source, response.Status)
	}
	name := filepath.Base(parsed.Path)
	if disposition := response.Header.Get("Content-Disposition"); disposition != "" {
		if _, params, parseErr := mime.ParseMediaType(disposition); parseErr == nil && params["filename"] != "" {
			name = filepath.Base(params["filename"])
		}
	}
	return engine.ConvertReader(ctx, response.Body, StreamInfo{Name: name, MIMEType: response.Header.Get("Content-Type"), URL: source})
}

func (engine *MarkItDown) convertData(ctx context.Context, data []byte, info StreamInfo) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx = core.WithTableFormat(ctx, engine.Settings.TableFormatValue())
	engine.mu.RLock()
	converters := append([]Converter(nil), engine.converters...)
	engine.mu.RUnlock()
	for _, converter := range converters {
		if converter.Supports(info) {
			result, err := converter.Convert(ctx, data, info)
			if err != nil {
				return nil, err
			}
			return normalizeResult(result), nil
		}
	}
	if utf8.Valid(data) && !bytes.ContainsRune(data, '\x00') {
		return normalizeResult(&Result{Markdown: string(data)}), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, info.Extension)
}

func normalizeResult(result *Result) *Result {
	if result == nil {
		result = &Result{}
	}
	if result.Markdown == "" {
		result.Markdown = result.TextContent
	}
	result.Markdown = strings.TrimSpace(strings.ReplaceAll(result.Markdown, "\r\n", "\n"))
	result.TextContent = result.Markdown
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}
	return result
}

func normalizeInfo(info StreamInfo) StreamInfo {
	info.Extension = normalizeExtension(info.Extension)
	if info.Extension == "" {
		info.Extension = normalizeExtension(filepath.Ext(info.Name))
	}
	if mediaType, _, err := mime.ParseMediaType(info.MIMEType); err == nil {
		info.MIMEType = strings.ToLower(mediaType)
	}
	return info
}

func normalizeExtension(extension string) string {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	return extension
}

// The built-in converter constructors, kept package-local so the registration
// table below reads as the converter catalog.
var (
	newTextConverter       = text.NewTextConverter
	newHTMLConverter       = text.NewHTMLConverter
	newCSVConverter        = text.NewCSVConverter
	newStructuredConverter = text.NewStructuredTextConverter
	newDOCXConverter       = docx.NewConverter
	newXLSXConverter       = xlsx.NewConverter
	newPPTXConverter       = pptx.NewConverter
	newEPUBConverter       = archive.NewEPUBConverter
	newEMLConverter        = email.NewConverter
	newZIPConverter        = archive.NewZIPConverter
	newImageConverter      = imagefile.NewConverter
	newPDFConverter        = pdf.NewConverter
)
