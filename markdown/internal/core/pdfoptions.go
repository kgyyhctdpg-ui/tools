package core

import (
	"fmt"
	"runtime"
	"strings"
)

// PDFImageFormat selects the raster format used for pictures produced by the
// PDF converter: both rasterized pages and the images embedded in a page.
type PDFImageFormat string

const (
	PDFImageFormatPNG  PDFImageFormat = "png"
	PDFImageFormatJPEG PDFImageFormat = "jpeg"
)

const (
	defaultPDFRenderDPI         = 200.0
	defaultPDFRenderFormat      = PDFImageFormatJPEG
	defaultPDFJPEGQuality       = 85
	defaultPDFMaxImagesPerPage  = 32
	defaultPDFMinTableRows      = 2
	defaultPDFMinTableColumns   = 2
	defaultPDFMaxTableCellRunes = 60
	defaultPDFMaxTableMedian    = 24
)

// maxPDFPageWorkers caps the automatic page concurrency. Each worker opens its
// own MuPDF context, which reserves up to 256 MB of store and needs room for one
// decoded page, so the worker count is kept well below what a very wide machine
// could otherwise start.
const maxPDFPageWorkers = 8

// PDFOptions configures the built-in local PDF converter. The zero value keeps
// every default, so PDFOptions{} converts exactly like the defaults.
//
// The local converter never performs OCR: a page without a text layer is
// either kept as its embedded images or rasterized as a whole-page picture and
// referenced from the Markdown.
type PDFOptions struct {
	// RenderDPI is the resolution used when a page is rasterized, and the
	// ceiling for the effective resolution of an embedded image. Defaults to
	// 200. An embedded image that was placed above this resolution is
	// downscaled before it is encoded, which is what keeps a scanned page from
	// yielding a multi-megabyte picture.
	RenderDPI float64
	// RenderFormat is "jpeg" (default) or "png" and applies to embedded images
	// as well as to rasterized pages.
	RenderFormat PDFImageFormat
	// JPEGQuality applies when RenderFormat is JPEG. Defaults to 85.
	JPEGQuality int

	// FirstPage and LastPage bound the converted page range, both 1-based and
	// inclusive. Zero means the first respectively the last page.
	FirstPage int
	LastPage  int

	// MaxImagesPerPage bounds embedded-image extraction for a single page. A
	// page that carries more image fragments than this is rasterized as one
	// picture instead, which keeps drawings and chart-heavy pages from
	// exploding into hundreds of files. Defaults to 32.
	MaxImagesPerPage int
	// MaxImages bounds the embedded images extracted across the whole
	// document. Zero means no limit. Pages past the limit are rasterized
	// instead.
	MaxImages int
	// MaxRenderedPages bounds how many pages are rasterized per document. Zero
	// means no limit.
	MaxRenderedPages int

	// DisablePageRenderFallback keeps pages that yield neither text nor
	// embedded images empty instead of rasterizing them.
	DisablePageRenderFallback bool
	// DisableTableReconstruction turns off the table grid detection.
	DisableTableReconstruction bool
	// DisableHeadingDetection turns off font-size based heading detection.
	DisableHeadingDetection bool
	// DisableParagraphReflow turns off joining wrapped visual lines into one
	// paragraph.
	DisableParagraphReflow bool

	// PageConcurrency is the number of pages converted at once. Zero picks one
	// worker per available CPU, capped at 8; 1 converts sequentially. MuPDF
	// contexts are not thread safe, so each worker opens its own document over
	// the same input buffer instead of sharing one.
	//
	// Pages are independent, and the raster work dominates a PDF conversion, so
	// this is close to a linear speedup. It is ignored when MaxImages or
	// MaxRenderedPages is set, because those budgets are spent in page order and
	// the result would stop being reproducible.
	PageConcurrency int
}

// NormalizePDFOptions fills in the defaults for every zero field.
func NormalizePDFOptions(options PDFOptions) PDFOptions {
	if options.RenderDPI <= 0 {
		options.RenderDPI = defaultPDFRenderDPI
	}
	switch PDFImageFormat(strings.ToLower(strings.TrimSpace(string(options.RenderFormat)))) {
	case PDFImageFormatPNG:
		options.RenderFormat = PDFImageFormatPNG
	case PDFImageFormatJPEG:
		options.RenderFormat = PDFImageFormatJPEG
	default:
		options.RenderFormat = defaultPDFRenderFormat
	}
	if options.JPEGQuality <= 0 || options.JPEGQuality > 100 {
		options.JPEGQuality = defaultPDFJPEGQuality
	}
	if options.MaxImagesPerPage <= 0 {
		options.MaxImagesPerPage = defaultPDFMaxImagesPerPage
	}
	if options.FirstPage < 0 {
		options.FirstPage = 0
	}
	if options.LastPage < 0 {
		options.LastPage = 0
	}
	if options.PageConcurrency < 0 {
		options.PageConcurrency = 0
	}
	return options
}

// PageWorkers resolves PageConcurrency into an actual worker count.
func (options PDFOptions) PageWorkers() int {
	if options.PageConcurrency > 0 {
		return options.PageConcurrency
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > maxPDFPageWorkers {
		workers = maxPDFPageWorkers
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}

// PageRange resolves the configured range against the document page count into
// a zero-based half-open interval.
func (options PDFOptions) PageRange(pageCount int) (int, int, error) {
	if pageCount <= 0 {
		return 0, 0, nil
	}
	first := 1
	if options.FirstPage > 0 {
		first = options.FirstPage
	}
	last := pageCount
	if options.LastPage > 0 && options.LastPage < last {
		last = options.LastPage
	}
	if first > pageCount {
		return 0, 0, fmt.Errorf("markdown: PDF first page %d is past the last page %d", first, pageCount)
	}
	if last < first {
		return 0, 0, fmt.Errorf("markdown: PDF page range %d-%d is empty", first, last)
	}
	return first - 1, last, nil
}

// PDF table reconstruction limits, exposed so the PDF package can share them.
const (
	PDFMinTableRows      = defaultPDFMinTableRows
	PDFMinTableColumns   = defaultPDFMinTableColumns
	PDFMaxTableCellRunes = defaultPDFMaxTableCellRunes
	PDFMaxTableMedian    = defaultPDFMaxTableMedian
)
