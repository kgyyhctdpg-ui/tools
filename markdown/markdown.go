// Package markdown converts common document formats to Markdown.
package markdown

import (
	"bytes"
	"context"
	"encoding/base64"
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
)

const (
	defaultMaxInputSize       = int64(128 << 20)
	defaultMaxArchiveFileSize = int64(32 << 20)
	defaultMaxArchiveFiles    = 512
	defaultMaxArchiveDepth    = 4
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

// PDFHandler converts every PDF through MinerU. imageHandler must be used for
// images returned by MinerU so WithAssetsDirectory keeps working.
type PDFHandler func(ctx context.Context, data []byte, info StreamInfo, imageHandler ImageHandler) (*Result, error)

// Converter converts one or more formats.
type Converter interface {
	Supports(info StreamInfo) bool
	Convert(ctx context.Context, data []byte, info StreamInfo) (*Result, error)
}

type ExtensionProvider interface {
	Extensions() []string
}

type Option func(*MarkItDown)

type MarkItDown struct {
	mu sync.RWMutex

	converters   []Converter
	httpClient   *http.Client
	imageHandler ImageHandler
	pdfHandler   PDFHandler
	tableFormat  TableFormat

	maxInputSize       int64
	maxArchiveFileSize int64
	maxArchiveFiles    int
	maxArchiveDepth    int
}

func New(options ...Option) *MarkItDown {
	engine := &MarkItDown{
		httpClient:         http.DefaultClient,
		imageHandler:       DataURIImageHandler,
		tableFormat:        TableFormatHTML,
		maxInputSize:       defaultMaxInputSize,
		maxArchiveFileSize: defaultMaxArchiveFileSize,
		maxArchiveFiles:    defaultMaxArchiveFiles,
		maxArchiveDepth:    defaultMaxArchiveDepth,
	}
	engine.registerBuiltins()
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
			engine.imageHandler = handler
		}
	}
}

// WithPDFHandler configures the MinerU conversion used for every PDF.
func WithPDFHandler(handler PDFHandler) Option {
	return func(engine *MarkItDown) {
		engine.pdfHandler = handler
	}
}

func WithTableFormat(format TableFormat) Option {
	return func(engine *MarkItDown) {
		if format == TableFormatHTML {
			engine.tableFormat = TableFormatHTML
		} else {
			engine.tableFormat = TableFormatMarkdown
		}
	}
}

func WithMaxInputSize(size int64) Option {
	return func(engine *MarkItDown) { engine.maxInputSize = size }
}

func WithArchiveLimits(files int, fileSize int64, depth int) Option {
	return func(engine *MarkItDown) {
		if files > 0 {
			engine.maxArchiveFiles = files
		}
		if fileSize > 0 {
			engine.maxArchiveFileSize = fileSize
		}
		if depth > 0 {
			engine.maxArchiveDepth = depth
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
	data, err := readLimited(reader, engine.maxInputSize)
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

// DataURIImageHandler embeds an image directly in Markdown.
func DataURIImageHandler(_ context.Context, image Image) (string, error) {
	mimeType := image.MIMEType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(image.Data), nil
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
		return nil, fmt.Errorf("%w: %d bytes", ErrInputTooLarge, info.Size())
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

type conversionConfigKey struct{}
type conversionConfig struct{ tableFormat TableFormat }

func (engine *MarkItDown) convertData(ctx context.Context, data []byte, info StreamInfo) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, conversionConfigKey{}, conversionConfig{tableFormat: engine.tableFormat})
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

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(reader)
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: limit is %d bytes", ErrInputTooLarge, limit)
	}
	return data, nil
}
