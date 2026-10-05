package pdf

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/88250/lute"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

// PDFNoOCRWarning is reported when a PDF carried pages without a text layer.
// Those pages are converted to pictures and linked from the Markdown; this
// converter never performs OCR.
const PDFNoOCRWarning = "markdown: PDF pages without a text layer are stored as images and linked from the Markdown; this converter does not perform OCR."

// PDFPageRenderWarning is reported when pages had to be rasterized because they
// carried neither extractable text nor a content-sized embedded image — either no
// images at all, or only images too small to be the page's content (stamps, logos,
// header marks).
const PDFPageRenderWarning = "markdown: %d PDF page(s) carried no extractable text and no content-sized image and were rendered as pictures."

// NewConverter builds the local PDF converter.
func NewConverter(settings *core.Settings) core.Converter {
	return core.NewExtensionConverter(
		[]string{".pdf"},
		[]string{"application/pdf"},
		func(ctx context.Context, data []byte, info core.StreamInfo) (*core.Result, error) {
			return convertPDF(ctx, data, settings)
		},
	)
}

// pdfPageStats records what one page contributed.
type pdfPageStats struct {
	text       bool
	rendered   bool
	imageCount int
}

// pdfImageOnlyPageCoverageThreshold bounds how much of a page may be covered by
// placed images before those images are treated as the page's content.
//
// A page without a text layer whose images cover only a small part of it does not
// carry its content in those images: the content is vector artwork, which neither
// text extraction nor image extraction can capture. Keeping the pictures there
// silently drops the whole page.
//
// Calibrated against real documents:
//   - stamped drawing pages: 0.026 (590 of 592 pages of one drawing set)
//   - the largest picture on a drawing page: 0.549
//   - a full-page scan: 1.0
//
// 0.75 therefore separates the two cases with ample margin, and does not mistake a
// scan for a stamped page.
const pdfImageOnlyPageCoverageThreshold = 0.75

// shouldRasterizeInsteadOfImages reports whether a page that carries images but no
// text layer should be rasterized as a whole instead of keeping its pictures.
//
// Pages without a text layer whose only images are small (stamps, logos, header
// marks) are not carrying their content in those images: the content is vector
// artwork, which appears in neither the extracted text nor any embedded image.
// Keeping the pictures there loses the entire page, and reports no warning.
//
// The test compares the total placed-image area against the page area rather than
// counting images, so a scan assembled from several tiles still counts as content.
func shouldRasterizeInsteadOfImages(document *pdfDocument, page int, lines []pdfTextLine, images []pdfPlacedImage) bool {
	for _, line := range lines {
		if line.Text() != "" {
			return false // the page has a text layer, so this is not the case
		}
	}
	if len(images) == 0 {
		return false // no images at all: leave it to the existing empty-page fallback
	}
	width, height, err := document.PageSize(page)
	if err != nil {
		return false // without page dimensions, keep the previous behaviour
	}
	return imagesCoverLessThanShareOfPage(images, width, height, pdfImageOnlyPageCoverageThreshold)
}

// imagesCoverLessThanShareOfPage reports whether the placed images' areas sum to
// less than share of the page area. Pure arithmetic, so the threshold decision is
// testable without a live PDF engine. Non-positive page dimensions report false,
// which leaves the caller on its previous behaviour.
func imagesCoverLessThanShareOfPage(images []pdfPlacedImage, pageWidth, pageHeight, share float64) bool {
	if pageWidth <= 0 || pageHeight <= 0 {
		return false
	}
	var covered float64
	for _, image := range images {
		if w, h := image.Right-image.Left, image.Top-image.Bottom; w > 0 && h > 0 {
			covered += w * h
		}
	}
	return covered/(pageWidth*pageHeight) < share
}

// convertPDF converts every page locally with MuPDF (through go-fitz). No OCR
// and no remote service are involved: text pages are rebuilt from the engine's
// positioned text runs, pages whose pictures are the content keep their pictures,
// and pages that carry neither text nor content-sized pictures are rasterized.
func convertPDF(ctx context.Context, data []byte, settings *core.Settings) (*core.Result, error) {
	document, err := openPDFDocument(data)
	if err != nil {
		return nil, err
	}
	defer document.Close()

	options := settings.PDF
	imageHandler := settings.ImageHandlerOrDefault()
	tableFormat := core.TableFormatFromContext(ctx)

	pageCount, err := document.PageCount()
	if err != nil {
		return nil, err
	}
	first, last, err := options.PageRange(pageCount)
	if err != nil {
		return nil, err
	}

	pages := make([]string, 0, last-first)
	var (
		textPageCount    int
		renderedPages    int
		embeddedImages   int
		convertedPageNum int
	)
	accumulate := func(pageMarkdown string, stats pdfPageStats) {
		convertedPageNum++
		embeddedImages += stats.imageCount
		if stats.rendered {
			renderedPages++
		}
		if stats.text {
			textPageCount++
		}
		if text := strings.TrimSpace(pageMarkdown); text != "" {
			pages = append(pages, text)
		}
	}

	// MuPDF contexts are not thread safe, so each worker gets its own document
	// over the same input buffer. That rules out the two document-wide image
	// budgets, which are spent in page order: honouring them sequentially keeps
	// the output reproducible.
	workers := options.PageWorkers()
	if workers > 1 && options.MaxImages == 0 && options.MaxRenderedPages == 0 {
		results, err := convertPDFPagesParallel(ctx, data, first, last, options, tableFormat, imageHandler, workers)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			accumulate(result.markdown, result.stats)
		}
	} else {
		for page := first; page < last; page++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			pageMarkdown, stats, err := convertPDFPage(ctx, document, page, options, tableFormat, imageHandler, embeddedImages, renderedPages)
			if err != nil {
				return nil, err
			}
			accumulate(pageMarkdown, stats)
		}
	}

	contentType := "text"
	if textPageCount == 0 {
		contentType = "image"
	}
	metadata := map[string]string{
		"pdf_content_type":    contentType,
		"pdf_page_count":      strconv.Itoa(pageCount),
		"pdf_embedded_images": strconv.Itoa(embeddedImages),
		"pdf_rendered_pages":  strconv.Itoa(renderedPages),
	}
	if len(pages) != 0 {
		metadata["pdf_converted_pages"] = strconv.Itoa(convertedPageNum)
	}

	warnings := make([]string, 0, 2)
	if renderedPages > 0 {
		warnings = append(warnings, fmt.Sprintf(PDFPageRenderWarning, renderedPages))
	}
	if textPageCount < convertedPageNum {
		warnings = append(warnings, PDFNoOCRWarning)
	}

	title := ""
	for key, value := range document.Metadata() {
		value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
		if value != "" && strings.EqualFold(key, "title") {
			title = value
		}
	}
	return &core.Result{
		Title:    title,
		Markdown: core.JoinBlocks(pages),
		Metadata: metadata,
		Warnings: warnings,
	}, nil
}

// convertPDFPageResult is one converted page, held until the pages can be
// reassembled in their original order.
type convertPDFPageResult struct {
	markdown string
	stats    pdfPageStats
}

// convertPDFPagesParallel converts the page range with one MuPDF context per
// worker: pages never depend on each other. The raster work (decoding, scaling
// and encoding every embedded image, plus whole-page rendering) dominates a
// conversion and parallelises cleanly.
//
// Each worker opens its own document because a fitz.Document is not safe for
// concurrent use; the input buffer is shared and never written.
func convertPDFPagesParallel(
	ctx context.Context,
	data []byte,
	first, last int,
	options core.PDFOptions,
	tableFormat core.TableFormat,
	imageHandler core.ImageHandler,
	workers int,
) ([]convertPDFPageResult, error) {
	pages := last - first
	if workers > pages {
		workers = pages
	}
	results := make([]convertPDFPageResult, pages)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The image handler is caller supplied and used to be called from a single
	// goroutine; keep that guarantee instead of requiring handlers to be thread
	// safe. Only the store call is serialised, never the encoding before it.
	var handlerMutex sync.Mutex
	guardedHandler := func(ctx context.Context, image core.Image) (string, error) {
		handlerMutex.Lock()
		defer handlerMutex.Unlock()
		return imageHandler(ctx, image)
	}

	var (
		nextPage  atomic.Int64
		waitGroup sync.WaitGroup
		failOnce  sync.Once
		failure   error
	)
	fail := func(err error) {
		failOnce.Do(func() {
			failure = err
			cancel()
		})
	}

	for worker := 0; worker < workers; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			document, err := openPDFDocument(data)
			if err != nil {
				fail(err)
				return
			}
			defer document.Close()
			for {
				index := int(nextPage.Add(1)) - 1
				if index >= pages || ctx.Err() != nil {
					return
				}
				markdown, stats, err := convertPDFPage(ctx, document, first+index, options, tableFormat, guardedHandler, 0, 0)
				if err != nil {
					fail(err)
					return
				}
				results[index] = convertPDFPageResult{markdown: markdown, stats: stats}
			}
		}()
	}
	waitGroup.Wait()
	if failure != nil {
		return nil, failure
	}
	return results, nil
}

// convertPDFPage converts a single page, falling back to a whole-page picture
// when the page contributes nothing to keep — either because it has neither text
// nor images, or because its only images are too small to be the page's content
// (see shouldRasterizeInsteadOfImages).
func convertPDFPage(
	ctx context.Context,
	document *pdfDocument,
	page int,
	options core.PDFOptions,
	tableFormat core.TableFormat,
	imageHandler core.ImageHandler,
	usedImages int,
	renderedSoFar int,
) (string, pdfPageStats, error) {
	var stats pdfPageStats

	lines, err := document.PageTextLines(page)
	if err != nil {
		return "", stats, err
	}
	images, err := document.PageImages(page)
	if err != nil {
		return "", stats, err
	}

	// 无文字层的页面上若只贴了一小块图（印章、页眉标识、logo），那多半不是页面
	// 正文；真正的正文是矢量内容（图纸线条、图表），它既不会被文字提取捕获，也不会
	// 出现在任何一张嵌入图里。此时「保留图片」等于把整页正文丢掉，因此改走整页光栅。
	// 判定依据与阈值标定见 shouldRasterizeInsteadOfImages。
	if shouldRasterizeInsteadOfImages(document, page, lines, images) &&
		!options.DisablePageRenderFallback &&
		!(options.MaxRenderedPages > 0 && renderedSoFar >= options.MaxRenderedPages) {
		rendered, err := renderPDFPageImage(ctx, document, page, imageHandler, options)
		if err != nil {
			return "", stats, err
		}
		stats.rendered = true
		return rendered, stats, nil
	}

	content, stats, err := renderPDFItems(ctx, document.RenderImageRegion, page, lines, images, options, tableFormat, imageHandler, usedImages)
	if err != nil {
		return "", stats, err
	}

	if strings.TrimSpace(content) != "" {
		return content, stats, nil
	}
	if options.DisablePageRenderFallback {
		return "", stats, nil
	}
	if options.MaxRenderedPages > 0 && renderedSoFar >= options.MaxRenderedPages {
		return "", stats, nil
	}
	rendered, err := renderPDFPageImage(ctx, document, page, imageHandler, options)
	if err != nil {
		return "", stats, err
	}
	stats.rendered = true
	return rendered, stats, nil
}

// pdfImageRenderer rasterises one placed image. It is injected so the layout
// tests can exercise image placement without a live PDF engine.
type pdfImageRenderer func(page int, image pdfPlacedImage, options core.PDFOptions) ([]byte, string, error)

// pdfImageSlot is one placed image together with the number of text lines that
// precede it on the page.
type pdfImageSlot struct {
	image   pdfPlacedImage
	ordinal int
}

// orderPDFImages places every image after the text lines that sit above it, so
// a picture stays next to the content it illustrates.
// orderPDFImages places every image after the text lines that end above it, so
// a picture stays next to the content it illustrates.
//
// The test is whether a line finishes before the image starts, not where its top
// sits. Comparing tops pushes an image that begins level with a row into the row
// above, and the two values are only ever float approximations of each other.
func orderPDFImages(lines []pdfTextLine, images []pdfPlacedImage) []pdfImageSlot {
	slots := make([]pdfImageSlot, 0, len(images))
	for _, image := range images {
		ordinal := 0
		for _, line := range lines {
			if line.Bottom <= image.TopDown {
				ordinal++
			}
		}
		slots = append(slots, pdfImageSlot{image: image, ordinal: ordinal})
	}
	sort.SliceStable(slots, func(left, right int) bool {
		if slots[left].ordinal != slots[right].ordinal {
			return slots[left].ordinal < slots[right].ordinal
		}
		return slots[left].image.TopDown < slots[right].image.TopDown
	})
	return slots
}

// renderPDFItems rebuilds the Markdown of one page from its positioned lines
// and placed images.
//
// Lines are laid out first and images are then re-inserted at the block
// boundary that matches their position on the page. That keeps a table
// containing pictures (signature blocks, part tables) intact instead of being
// torn apart at every image.
func renderPDFItems(
	ctx context.Context,
	renderImages pdfImageRenderer,
	page int,
	lines []pdfTextLine,
	placed []pdfPlacedImage,
	options core.PDFOptions,
	tableFormat core.TableFormat,
	imageHandler core.ImageHandler,
	usedImages int,
) (string, pdfPageStats, error) {
	var stats pdfPageStats
	for _, line := range lines {
		if line.Text() != "" {
			stats.text = true
		}
	}

	slots := orderPDFImages(lines, placed)
	imageOrdinal := make([]int, 0, len(slots))
	for _, slot := range slots {
		imageOrdinal = append(imageOrdinal, slot.ordinal)
	}

	blocks := buildPDFLineBlocks(lines, imageOrdinal, options, tableFormat)
	ordered := make([]string, 0, len(blocks)+len(slots))
	// One engine per page: lute is not documented as safe for concurrent use,
	// and the converter may run on several documents at once.
	engine := lute.New()
	imageIndex := 0
	emittedImages := 0
	emitImages := func(limit int) error {
		for imageIndex < len(slots) && slots[imageIndex].ordinal <= limit {
			slot := slots[imageIndex]
			imageIndex++
			if emittedImages >= options.MaxImagesPerPage {
				continue
			}
			if options.MaxImages > 0 && usedImages+emittedImages >= options.MaxImages {
				continue
			}
			data, mimeType, err := renderImages(page, slot.image, options)
			if err != nil {
				// One picture that cannot be rasterized must not fail the whole
				// document: drawing sets routinely carry specks and masks that
				// have no usable raster size.
				continue
			}
			emittedImages++
			markdown, err := storePDFEmbeddedImage(ctx, imageHandler, &pdfEmbeddedImage{
				MIMEType: mimeType,
				Data:     data,
			}, page, emittedImages)
			if err != nil {
				return err
			}
			ordered = append(ordered, markdown)
		}
		return nil
	}

	consumed := 0
	for _, block := range blocks {
		if err := emitImages(consumed); err != nil {
			return "", stats, err
		}
		markdown, err := renderPDFBlock(engine, block)
		if err != nil {
			return "", stats, err
		}
		if markdown = strings.TrimSpace(markdown); markdown != "" {
			ordered = append(ordered, markdown)
		}
		consumed += block.lines
	}
	if err := emitImages(len(lines)); err != nil {
		return "", stats, err
	}
	stats.imageCount = emittedImages
	return core.JoinBlocks(ordered), stats, nil
}

func storePDFEmbeddedImage(ctx context.Context, imageHandler core.ImageHandler, image *pdfEmbeddedImage, page, index int) (string, error) {
	name := fmt.Sprintf("pdf-page-%d-image-%d%s", page+1, index, core.ImageExtension(image.MIMEType))
	imageURL, err := imageHandler(ctx, core.Image{
		Name:     name,
		MIMEType: image.MIMEType,
		AltText:  image.AltText,
		Data:     image.Data,
	})
	if err != nil {
		return "", fmt.Errorf("markdown: store PDF page %d image %d: %w", page+1, index, err)
	}
	return core.MarkdownImage(image.AltText, imageURL), nil
}

// renderPDFPageImage rasterizes one whole page, used when the page carries no
// extractable text and no image to keep.
func renderPDFPageImage(ctx context.Context, document *pdfDocument, page int, imageHandler core.ImageHandler, options core.PDFOptions) (string, error) {
	data, mimeType, err := document.RenderPageImage(page, options)
	if err != nil {
		return "", err
	}
	altText := fmt.Sprintf("第 %d 页", page+1)
	imageURL, err := imageHandler(ctx, core.Image{
		Name:     fmt.Sprintf("pdf-page-%d%s", page+1, core.ImageExtension(mimeType)),
		MIMEType: mimeType,
		AltText:  altText,
		Data:     data,
	})
	if err != nil {
		return "", fmt.Errorf("markdown: store PDF page %d image: %w", page+1, err)
	}
	return core.MarkdownImage(altText, imageURL), nil
}
