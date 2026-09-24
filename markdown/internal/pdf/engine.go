package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/gen2brain/go-fitz"
	"golang.org/x/image/draw"
	"golang.org/x/net/html"

	"github.com/scoming-dev/tools/markdown/internal/core"
)

// This file is the only place that talks to MuPDF, through go-fitz. It exposes
// an engine neutral view of a document: positioned text runs, placed image
// objects and page rasters. The layout reconstruction in pdf_layout.go only
// ever sees pdfTextLine values, so the PDF engine can be swapped without
// touching it.
//
// go-fitz does not publish MuPDF's structured text API, but it can print a page
// as HTML. That output is exactly what the layout stage needs: every visual run
// becomes a <p> carrying absolute top/left/line-height in points, inline runs
// carry font-size and bold/italic, and large horizontal gaps split one visual
// line into several <p> elements, which is how table cells survive. Images are
// inlined as base64 data URIs with the placement matrix.
//
// MuPDF prints HTML in a top-down coordinate system while the layout code
// expects native PDF space (origin bottom left, Y growing upwards). Every run
// is flipped with the page height right here, and buildPDFTextLines flips it
// back.
type pdfDocument struct {
	document *fitz.Document

	cachedPage int
	cached     *pdfPageHTML
	cachedOK   bool
}

// pdfPageHTML is the parsed view of one page: its box, its positioned text runs
// and its placed images.
type pdfPageHTML struct {
	width  float64
	height float64
	rects  []pdfTextRect
	images []pdfPlacedImage
}

// Font Descriptor flags defined by PDF 1.7 section 5.7.1, table 5.19
// (one-based bit numbers).
const (
	fontFlagItalic    = 1 << 6  // bit 7
	fontFlagForceBold = 1 << 17 // bit 18
)

// pdfCSSPointsPerPixel converts a CSS pixel to a PDF point (72 dpi against the
// 96 dpi MuPDF's HTML output assumes).
const pdfCSSPointsPerPixel = 72.0 / 96.0

// minImageRegionPoints is the smallest placement rectangle that is worth
// turning into a picture. Drawing sets carry many sub-point specks and masks
// next to the real drawings.
const minImageRegionPoints = 1.0

const (
	// maxPDFEmbeddedImageBytes bounds one embedded image payload. A scanned
	// page is a few megabytes; anything past this is a malformed data URI.
	maxPDFEmbeddedImageBytes = 64 << 20
	// maxPDFEmbeddedImagePixels bounds the decoded bitmap of one image.
	maxPDFEmbeddedImagePixels = 40 << 20
	// maxPDFEmbeddedImageBase64 bounds the base64 text decoded per image.
	maxPDFEmbeddedImageBase64 = maxPDFEmbeddedImageBytes/3*4 + 1024
)

// openPDFDocument loads a PDF from memory.
func openPDFDocument(data []byte) (*pdfDocument, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("markdown: open PDF: empty input")
	}
	document, err := fitz.NewFromMemory(data)
	if err != nil {
		// A partially initialised document cannot be closed safely: NewFromMemory
		// returns a non-nil value whose context may never have been created.
		return nil, fmt.Errorf("markdown: open PDF: %w", err)
	}
	return &pdfDocument{document: document}, nil
}

// Close releases the document.
func (document *pdfDocument) Close() {
	if document == nil || document.document == nil {
		return
	}
	_ = document.document.Close()
	document.document = nil
	document.cached = nil
	document.cachedOK = false
}

// PageCount returns the number of pages.
func (document *pdfDocument) PageCount() (int, error) {
	count := document.document.NumPage()
	if count <= 0 {
		return 0, fmt.Errorf("markdown: read PDF page count: document has no pages")
	}
	return count, nil
}

// Metadata returns the document information dictionary. MuPDF pads every value
// to a fixed width, so the padding is stripped here.
func (document *pdfDocument) Metadata() map[string]string {
	metadata := make(map[string]string)
	for key, value := range document.document.Metadata() {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
		if key != "" && value != "" {
			metadata[key] = value
		}
	}
	return metadata
}

// pageHTML returns the parsed HTML view of one page, parsing at most one page at
// a time. Text and images come from the same parse, so a page is only printed
// once.
func (document *pdfDocument) pageHTML(index int) (*pdfPageHTML, error) {
	pageCount := document.document.NumPage()
	if index < 0 || index >= pageCount {
		return nil, fmt.Errorf("markdown: PDF page %d is missing from a %d page document", index+1, pageCount)
	}
	if document.cachedOK && document.cachedPage == index {
		return document.cached, nil
	}
	source, err := document.document.HTML(index, false)
	if err != nil {
		return nil, fmt.Errorf("markdown: extract PDF page %d: %w", index+1, err)
	}
	page, err := parsePDFPageHTML(source)
	if err != nil {
		return nil, fmt.Errorf("markdown: extract PDF page %d: %w", index+1, err)
	}
	if page.height <= 0 {
		if bounds, err := document.document.Bound(index); err == nil {
			page.width, page.height = float64(bounds.Dx()), float64(bounds.Dy())
		}
	}
	if page.height <= 0 {
		return nil, fmt.Errorf("markdown: read PDF page %d size", index+1)
	}
	capPDFRunWidths(page.rects, page.height)
	document.cachedPage, document.cached, document.cachedOK = index, page, true
	return page, nil
}

// PageSize returns the page dimensions in points.
func (document *pdfDocument) PageSize(index int) (float64, float64, error) {
	page, err := document.pageHTML(index)
	if err != nil {
		return 0, 0, err
	}
	return page.width, page.height, nil
}

// pdfTextRect is one positioned run of text. Coordinates use native PDF space:
// the origin is the bottom left corner and Y grows upwards.
type pdfTextRect struct {
	Text     string
	Left     float64
	Bottom   float64
	Right    float64
	Top      float64
	FontSize float64
	Weight   int
	Flags    int
	FontName string
	// paragraph identifies the <p> this run came from, which is how the width
	// cap tells runs that belong to one visual line apart from runs that merely
	// share a baseline.
	paragraph int
}

func (rect pdfTextRect) bold() bool {
	if rect.Weight >= 600 || rect.Flags&fontFlagForceBold != 0 {
		return true
	}
	name := strings.ToLower(rect.FontName)
	return strings.Contains(name, "bold") || strings.Contains(name, "black") ||
		strings.Contains(name, "heavy") || strings.Contains(name, "semibold") ||
		strings.Contains(name, "demibold")
}

func (rect pdfTextRect) italic() bool {
	if rect.Flags&fontFlagItalic != 0 {
		return true
	}
	name := strings.ToLower(rect.FontName)
	return strings.Contains(name, "italic") || strings.Contains(name, "oblique")
}

// PageTextLines extracts the page text as positioned visual lines, already
// flipped into the top-down coordinate space the layout code uses.
func (document *pdfDocument) PageTextLines(index int) ([]pdfTextLine, error) {
	page, err := document.pageHTML(index)
	if err != nil {
		return nil, err
	}
	return buildPDFTextLines(page.rects, page.height), nil
}

// pdfPlacedImage is one image object drawn on a page.
type pdfPlacedImage struct {
	// Region is the placement rectangle in native PDF space.
	Left   float64
	Bottom float64
	Right  float64
	Top    float64
	// TopDown is how far the image starts from the top of the page, in the
	// coordinate space the layout code uses.
	TopDown float64
	// PixelWidth and PixelHeight are the image's native resolution.
	PixelWidth  uint
	PixelHeight uint

	// Data and MIMEType carry the picture MuPDF inlined into the page HTML.
	// Rendering the placement region again is unnecessary, and would lose soft
	// masks that MuPDF already flattened.
	Data     []byte
	MIMEType string
}

// PageImages lists the image objects drawn directly on a page.
func (document *pdfDocument) PageImages(index int) ([]pdfPlacedImage, error) {
	page, err := document.pageHTML(index)
	if err != nil {
		return nil, err
	}
	return page.images, nil
}

// RenderPageImage rasterises one page as a picture.
func (document *pdfDocument) RenderPageImage(index int, options core.PDFOptions) ([]byte, string, error) {
	dpi := options.RenderDPI
	if dpi <= 0 {
		dpi = 72
	}
	frame, err := document.document.ImageDPI(index, dpi)
	if err != nil {
		return nil, "", fmt.Errorf("markdown: render PDF page %d: %w", index+1, err)
	}
	return encodePDFImage(frame, options)
}

// RenderImageRegion returns the picture of one placed image, compressed for
// embedding in Markdown.
//
// MuPDF inlines every image at its native resolution, so a scanned page arrives
// as a multi-megabyte PNG. The picture is therefore downscaled until its
// effective resolution no longer exceeds PDFOptions.RenderDPI, then re-encoded
// with PDFOptions.RenderFormat and PDFOptions.JPEGQuality: the same settings a
// rasterized page uses.
func (document *pdfDocument) RenderImageRegion(index int, placed pdfPlacedImage, options core.PDFOptions) ([]byte, string, error) {
	if len(placed.Data) == 0 {
		return nil, "", fmt.Errorf("markdown: PDF page %d image has no payload", index+1)
	}
	scale := pdfImageScale(placed, options.RenderDPI)
	if scale >= 1 && pdfImageFormatMatches(placed.MIMEType, options.RenderFormat) {
		// Already the requested format and resolution: keep the bytes as they are.
		return placed.Data, placed.MIMEType, nil
	}
	frame, _, err := image.Decode(bytes.NewReader(placed.Data))
	if err != nil {
		// The engine already handed over a usable picture; keep it rather than
		// drop the image because it could not be re-encoded.
		mimeType := placed.MIMEType
		if mimeType == "" {
			mimeType = "image/png"
		}
		return placed.Data, mimeType, nil
	}
	if scale < 1 {
		frame = downscalePDFImage(frame, scale)
	}
	return encodePDFImage(frame, options)
}

// pdfImageScale reports the factor a placed image has to be scaled by so that
// its effective resolution stays within maxDPI. One keeps the native
// resolution.
//
// The effective resolution comes from the picture's pixel size against the size
// the page gives it in points, which is what makes a small logo with a huge
// bitmap as compressible as a full page scan.
func pdfImageScale(placed pdfPlacedImage, maxDPI float64) float64 {
	if maxDPI <= 0 || placed.PixelWidth == 0 || placed.PixelHeight == 0 {
		return 1
	}
	widthPoints := placed.Right - placed.Left
	heightPoints := placed.Top - placed.Bottom
	if widthPoints <= 0 || heightPoints <= 0 {
		return 1
	}
	horizontal := float64(placed.PixelWidth) * 72 / widthPoints
	vertical := float64(placed.PixelHeight) * 72 / heightPoints
	effective := math.Max(horizontal, vertical)
	if effective <= maxDPI {
		return 1
	}
	return maxDPI / effective
}

// pdfImageFormatMatches reports whether a payload already is in the requested
// format, so the re-encode can be skipped.
func pdfImageFormatMatches(mimeType string, format core.PDFImageFormat) bool {
	mimeType = strings.ToLower(mimeType)
	if format == core.PDFImageFormatJPEG {
		return strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg")
	}
	return strings.Contains(mimeType, "png")
}

// downscalePDFImage resizes a decoded picture, keeping at least one pixel per
// side. Catmull-Rom is used because scanned pages are usually shrunk by a large
// factor, where a cheaper filter aliases badly.
func downscalePDFImage(frame image.Image, scale float64) image.Image {
	bounds := frame.Bounds()
	width := int(math.Round(float64(bounds.Dx()) * scale))
	height := int(math.Round(float64(bounds.Dy()) * scale))
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if width >= bounds.Dx() && height >= bounds.Dy() {
		return frame
	}
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(target, target.Bounds(), frame, bounds, draw.Src, nil)
	return target
}

func encodePDFImage(frame image.Image, options core.PDFOptions) ([]byte, string, error) {
	if frame == nil {
		return nil, "", fmt.Errorf("markdown: PDF engine returned no image")
	}
	var buffer bytes.Buffer
	if options.RenderFormat == core.PDFImageFormatJPEG {
		frame = flattenPDFImage(frame)
		if err := jpeg.Encode(&buffer, frame, &jpeg.Options{Quality: options.JPEGQuality}); err != nil {
			return nil, "", fmt.Errorf("markdown: encode PDF image as JPEG: %w", err)
		}
		return buffer.Bytes(), "image/jpeg", nil
	}
	if err := png.Encode(&buffer, frame); err != nil {
		return nil, "", fmt.Errorf("markdown: encode PDF image as PNG: %w", err)
	}
	return buffer.Bytes(), "image/png", nil
}

// flattenPDFImage composites a picture onto white. JPEG has no alpha channel,
// so a transparent stamp or mask would otherwise come out black.
func flattenPDFImage(frame image.Image) image.Image {
	if opaque, ok := frame.(interface{ Opaque() bool }); ok && opaque.Opaque() {
		return frame
	}
	bounds := frame.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, bounds, frame, bounds.Min, draw.Over)
	return flat
}

// ---------------------------------------------------------------------------
// MuPDF HTML parsing
// ---------------------------------------------------------------------------

// parsePDFPageHTML turns one page of MuPDF's HTML into positioned text runs and
// placed images.
func parsePDFPageHTML(source string) (*pdfPageHTML, error) {
	root, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	page := &pdfPageHTML{}
	if !findPDFPageBox(root, page) || page.height <= 0 {
		return nil, fmt.Errorf("MuPDF output carries no page box")
	}

	paragraph := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "p":
				rects := parsePDFParagraph(node, page.height, paragraph)
				if len(rects) > 0 {
					paragraph++
					page.rects = append(page.rects, rects...)
				}
				return
			case "img":
				if placed, ok := parsePDFImage(node, page.height); ok {
					page.images = append(page.images, placed)
				}
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return page, nil
}

// findPDFPageBox reads the width and height of the page container div.
func findPDFPageBox(node *html.Node, page *pdfPageHTML) bool {
	if node.Type == html.ElementNode && node.Data == "div" {
		style := parsePDFStyle(elementStyle(node))
		width, widthOK := parsePDFPoints(style["width"])
		height, heightOK := parsePDFPoints(style["height"])
		if widthOK && heightOK && width > 0 && height > 0 {
			page.width, page.height = width, height
			return true
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if findPDFPageBox(child, page) {
			return true
		}
	}
	return false
}

// pdfTextSpan is one inline run of a paragraph before it is given a position.
type pdfTextSpan struct {
	text   string
	bold   bool
	italic bool
	size   float64
	family string
}

// parsePDFParagraph converts one <p> of MuPDF's HTML into the positioned runs
// of that visual line. A <p> may carry several inline runs, so emphasis and
// mixed font sizes survive.
func parsePDFParagraph(node *html.Node, pageHeight float64, paragraph int) []pdfTextRect {
	style := parsePDFStyle(elementStyle(node))
	top, ok := parsePDFPoints(style["top"])
	if !ok {
		return nil
	}
	left, _ := parsePDFPoints(style["left"])
	lineHeight, _ := parsePDFPoints(style["line-height"])
	if lineHeight < 0 {
		lineHeight = 0
	}

	spans := make([]pdfTextSpan, 0, 4)
	pending := ""
	var collect func(*html.Node, bool, bool, float64, string)
	collect = func(current *html.Node, bold, italic bool, size float64, family string) {
		if current.Type == html.ElementNode {
			switch current.Data {
			case "b", "strong":
				bold = true
			case "i", "em":
				italic = true
			case "span":
				spanStyle := parsePDFStyle(elementStyle(current))
				if value, ok := parsePDFPoints(spanStyle["font-size"]); ok && value > 0 {
					size = value
				}
				if value := spanStyle["font-family"]; value != "" {
					family = value
				}
			}
		}
		if current.Type == html.TextNode {
			data := current.Data
			if data == "" {
				return
			}
			if strings.TrimSpace(data) == "" {
				// Whitespace between two runs has to stay attached to a run:
				// the layout stage drops runs that hold nothing but spaces.
				if len(spans) > 0 {
					spans[len(spans)-1].text += data
				} else {
					pending += data
				}
				return
			}
			data = pending + data
			pending = ""
			if len(spans) > 0 {
				last := &spans[len(spans)-1]
				if last.bold == bold && last.italic == italic && last.size == size && last.family == family {
					last.text += data
					return
				}
			}
			spans = append(spans, pdfTextSpan{text: data, bold: bold, italic: italic, size: size, family: family})
			return
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			collect(child, bold, italic, size, family)
		}
	}
	collect(node, false, false, 0, "")

	hasText := false
	for _, span := range spans {
		if strings.TrimSpace(span.text) != "" {
			hasText = true
			break
		}
	}
	if !hasText {
		return nil
	}

	baseSize := 0.0
	for _, span := range spans {
		if span.size > 0 {
			baseSize = span.size
			break
		}
	}
	if baseSize <= 0 {
		baseSize = lineHeight
	}
	if lineHeight <= 0 {
		lineHeight = baseSize
	}

	rects := make([]pdfTextRect, 0, len(spans))
	cursor := left
	for _, span := range spans {
		if strings.TrimSpace(span.text) == "" {
			continue
		}
		size := span.size
		if size <= 0 {
			size = baseSize
		}
		width := pdfTextWidth(span.text, size)
		weight := 0
		if span.bold {
			weight = 700
		}
		flags := 0
		if span.italic {
			flags = fontFlagItalic
		}
		rects = append(rects, pdfTextRect{
			Text:      span.text,
			Left:      cursor,
			Right:     cursor + width,
			Top:       pageHeight - top,
			Bottom:    pageHeight - (top + lineHeight),
			FontSize:  size,
			Weight:    weight,
			Flags:     flags,
			FontName:  span.family,
			paragraph: paragraph,
		})
		cursor += width
	}
	return rects
}

// capPDFRunWidths keeps the estimated width of a run from swallowing the next
// run when both sit on one visual line but come from different <p> elements.
//
// MuPDF already decided where one cell ends and the next begins, so the estimate
// only has to preserve the gap it reported. Without this a long CJK cell
// followed by a narrow one would be measured as overlapping and the two would
// merge back into a single cell.
func capPDFRunWidths(rects []pdfTextRect, pageHeight float64) {
	groups := make(map[int64][]int, len(rects))
	for index := range rects {
		topDown := pageHeight - rects[index].Top
		key := int64(math.Round(topDown * 100))
		groups[key] = append(groups[key], index)
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(left, right int) bool {
			return rects[group[left]].Left < rects[group[right]].Left
		})
		for position := 0; position+1 < len(group); position++ {
			current := &rects[group[position]]
			next := rects[group[position+1]]
			if current.paragraph == next.paragraph {
				// Runs of one paragraph are adjacent by construction.
				continue
			}
			// The gap has to stay wider than what either the line merger or the
			// cell splitter treats as word spacing at this font size.
			limit := next.Left - pdfGapLimit(math.Max(current.FontSize, next.FontSize)) - 1
			if limit < current.Left {
				limit = current.Left
			}
			if current.Right > limit {
				current.Right = limit
			}
		}
	}
}

// parsePDFImage reads one inlined image and its placement matrix.
func parsePDFImage(node *html.Node, pageHeight float64) (pdfPlacedImage, bool) {
	style := parsePDFStyle(elementStyle(node))
	matrix, ok := parsePDFMatrix(style["transform"])
	if !ok {
		return pdfPlacedImage{}, false
	}
	mimeType, data, ok := parsePDFDataURI(attributeValue(node, "src"))
	if !ok {
		return pdfPlacedImage{}, false
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return pdfPlacedImage{}, false
	}
	if int64(config.Width)*int64(config.Height) > maxPDFEmbeddedImagePixels {
		return pdfPlacedImage{}, false
	}

	// The matrix maps the element's pixel box into CSS pixels; the element is
	// positioned by its top left corner plus half of its own pixel size, and CSS
	// pixels are converted to points at 72/96.
	pixelWidth := float64(config.Width)
	pixelHeight := float64(config.Height)
	width := (math.Abs(matrix.a)*pixelWidth + math.Abs(matrix.c)*pixelHeight) * pdfCSSPointsPerPixel
	height := (math.Abs(matrix.b)*pixelWidth + math.Abs(matrix.d)*pixelHeight) * pdfCSSPointsPerPixel
	if width < minImageRegionPoints || height < minImageRegionPoints {
		return pdfPlacedImage{}, false
	}
	centerX := (matrix.e + pixelWidth/2) * pdfCSSPointsPerPixel
	centerY := (matrix.f + pixelHeight/2) * pdfCSSPointsPerPixel
	left := centerX - width/2
	topDown := centerY - height/2
	return pdfPlacedImage{
		Left:        left,
		Right:       left + width,
		Top:         pageHeight - topDown,
		Bottom:      pageHeight - (topDown + height),
		TopDown:     topDown,
		PixelWidth:  uint(config.Width),
		PixelHeight: uint(config.Height),
		Data:        data,
		MIMEType:    mimeType,
	}, true
}

// pdfMatrix is the linear part and translation of a CSS matrix() transform.
type pdfMatrix struct {
	a, b, c, d, e, f float64
}

// parsePDFMatrix reads a CSS `matrix(a,b,c,d,e,f)` transform.
func parsePDFMatrix(value string) (pdfMatrix, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "matrix(") || !strings.HasSuffix(value, ")") {
		return pdfMatrix{}, false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(value, "matrix("), ")")
	parts := strings.Split(inner, ",")
	if len(parts) != 6 {
		return pdfMatrix{}, false
	}
	numbers := make([]float64, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return pdfMatrix{}, false
		}
		numbers = append(numbers, number)
	}
	return pdfMatrix{a: numbers[0], b: numbers[1], c: numbers[2], d: numbers[3], e: numbers[4], f: numbers[5]}, true
}

// parsePDFDataURI decodes one `data:` URI into its MIME type and bytes.
func parsePDFDataURI(value string) (string, []byte, bool) {
	const prefix = "data:"
	if !strings.HasPrefix(value, prefix) {
		return "", nil, false
	}
	comma := strings.IndexByte(value, ',')
	if comma < 0 {
		return "", nil, false
	}
	header := value[len(prefix):comma]
	payload := value[comma+1:]
	mimeType, encoding, _ := strings.Cut(header, ";")
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if mimeType == "" {
		mimeType = "image/png"
	}
	if !strings.EqualFold(strings.TrimSpace(encoding), "base64") {
		return "", nil, false
	}
	if len(payload) > maxPDFEmbeddedImageBase64 {
		return "", nil, false
	}
	cleaned := strings.Map(func(character rune) rune {
		if character == '\n' || character == '\r' || character == ' ' || character == '\t' {
			return -1
		}
		return character
	}, payload)
	data, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil || len(data) == 0 || len(data) > maxPDFEmbeddedImageBytes {
		return "", nil, false
	}
	return mimeType, data, true
}

// parsePDFStyle splits an inline style attribute into its declarations.
func parsePDFStyle(value string) map[string]string {
	style := make(map[string]string)
	for _, declaration := range strings.Split(value, ";") {
		name, setting, found := strings.Cut(declaration, ":")
		if !found {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		style[name] = strings.TrimSpace(setting)
	}
	return style
}

// parsePDFPoints reads a CSS length given in points.
func parsePDFPoints(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	value = strings.TrimSuffix(strings.ToLower(value), "pt")
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, false
	}
	return number, true
}

// elementStyle returns the style attribute of an element.
func elementStyle(node *html.Node) string {
	return attributeValue(node, "style")
}

// attributeValue returns one attribute of an element.
func attributeValue(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

// pdfTextWidth estimates the advance width of text at the given size. It only
// has to be good enough to keep the gaps MuPDF reported, so it favours the
// script's usual proportions rather than exact metrics.
func pdfTextWidth(text string, size float64) float64 {
	if size <= 0 {
		size = 10
	}
	width := 0.0
	for _, character := range text {
		switch {
		case character == ' ' || character == '\t' || character == '\n':
			width += size * 0.28
		case isCJKRune(character):
			width += size
		case character >= '0' && character <= '9':
			width += size * 0.55
		case character >= 'A' && character <= 'Z':
			width += size * 0.67
		case character >= 'a' && character <= 'z':
			width += size * 0.5
		default:
			width += size * 0.5
		}
	}
	return width
}
