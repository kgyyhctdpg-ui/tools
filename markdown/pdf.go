package markdown

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/88250/lute"
	fitz "github.com/gen2brain/go-fitz"
	nethtml "golang.org/x/net/html"
)

const PDFMinerUFallbackWarning = "MinerU is not configured; using local go-fitz HTML-to-Markdown PDF fallback. Tables, formulas, images, layout, and scanned-page OCR may be incomplete."

const pdfWholePageImageMinimumImages = 2

func newPDFConverter(engine *MarkItDown) Converter {
	return newExtensionConverter(
		[]string{".pdf"},
		[]string{"application/pdf"},
		func(ctx context.Context, data []byte, info StreamInfo) (*Result, error) {
			if engine.pdfHandler != nil {
				return engine.pdfHandler(ctx, data, info, engine.imageHandler)
			}
			return convertPDFFallback(ctx, data, engine.imageHandler)
		},
	)
}

func convertPDFFallback(ctx context.Context, data []byte, imageHandler ImageHandler) (*Result, error) {
	document, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("markdown: MinerU is not configured and local PDF fallback failed: %w", err)
	}
	defer document.Close()

	pageCount := document.NumPage()
	pages := make([]string, 0, pageCount)
	luteEngine := newPDFLuteEngine()
	textPageCount := 0
	for page := 0; page < pageCount; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pageText, textErr := document.Text(page)
		if textErr == nil && cleanPDFMarkdown(pageText) != "" {
			textPageCount++
		}
		pageHTML, err := document.HTML(page, false)
		if err != nil {
			return nil, fmt.Errorf("markdown: fallback extraction failed on PDF page %d: %w", page+1, err)
		}
		if shouldRenderPDFPageAsImage(pageText, pageHTML) {
			pageMarkdown, err := renderPDFPageImage(ctx, document, page, imageHandler)
			if err != nil {
				return nil, err
			}
			pages = append(pages, pageMarkdown)
			continue
		}
		pageHTML, err = externalizePDFHTMLImages(ctx, pageHTML, page+1, imageHandler)
		if err != nil {
			return nil, err
		}
		pageMarkdown, err := pdfHTMLToMarkdown(luteEngine, pageHTML)
		if err != nil || strings.TrimSpace(pageMarkdown) == "" {
			if textErr != nil {
				return nil, fmt.Errorf("markdown: fallback text extraction failed on PDF page %d: %w", page+1, textErr)
			}
			pageMarkdown = pageText
		}
		pageMarkdown = cleanPDFMarkdown(pageMarkdown)
		if pageMarkdown != "" {
			pages = append(pages, pageMarkdown)
		}
	}

	contentType := "text"
	if textPageCount == 0 {
		contentType = "image"
	}
	metadata := map[string]string{
		"pdf_content_type": contentType,
		"pdf_page_count":   strconv.Itoa(pageCount),
	}
	title := ""
	for key, value := range document.Metadata() {
		value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
		if value != "" && strings.EqualFold(key, "title") {
			title = value
		}
	}
	return &Result{
		Title:    title,
		Markdown: strings.Join(pages, "\n\n"),
		Metadata: metadata,
		Warnings: []string{PDFMinerUFallbackWarning},
	}, nil
}

func shouldRenderPDFPageAsImage(text, html string) bool {
	if cleanPDFMarkdown(text) != "" {
		return false
	}
	return countPDFDataImages(html) >= pdfWholePageImageMinimumImages
}

func countPDFDataImages(source string) int {
	return strings.Count(strings.ToLower(source), "data:image/")
}

func renderPDFPageImage(ctx context.Context, document *fitz.Document, page int, imageHandler ImageHandler) (string, error) {
	if imageHandler == nil {
		imageHandler = DataURIImageHandler
	}
	data, err := document.ImagePNG(page, 144)
	if err != nil {
		return "", fmt.Errorf("markdown: render PDF page %d image: %w", page+1, err)
	}
	altText := fmt.Sprintf("page %d", page+1)
	imageURL, err := imageHandler(ctx, Image{
		Name:     fmt.Sprintf("pdf-page-%d.png", page+1),
		MIMEType: "image/png",
		AltText:  altText,
		Data:     data,
	})
	if err != nil {
		return "", fmt.Errorf("markdown: store PDF page %d image: %w", page+1, err)
	}
	return markdownImage(altText, imageURL), nil
}

func newPDFLuteEngine() *lute.Lute {
	engine := lute.New()
	engine.SetAutoSpace(true)
	return engine
}

func externalizePDFHTMLImages(ctx context.Context, source string, page int, imageHandler ImageHandler) (string, error) {
	source = strings.TrimSpace(strings.ToValidUTF8(source, ""))
	if source == "" {
		return "", nil
	}
	if !strings.Contains(source, "data:image/") {
		return source, nil
	}
	if imageHandler == nil {
		imageHandler = DataURIImageHandler
	}
	root, err := nethtml.Parse(strings.NewReader(source))
	if err != nil {
		return "", fmt.Errorf("markdown: parse PDF page %d HTML: %w", page, err)
	}
	imageIndex := 0
	var walk func(*nethtml.Node) error
	walk = func(node *nethtml.Node) error {
		if node == nil {
			return nil
		}
		if node.Type == nethtml.ElementNode && strings.EqualFold(node.Data, "img") {
			for index := range node.Attr {
				if !strings.EqualFold(node.Attr[index].Key, "src") || !strings.HasPrefix(node.Attr[index].Val, "data:image/") {
					continue
				}
				mimeType, data, err := decodePDFDataImage(node.Attr[index].Val)
				if err != nil {
					return fmt.Errorf("markdown: decode PDF page %d image: %w", page, err)
				}
				imageIndex++
				imageURL, err := imageHandler(ctx, Image{
					Name:     fmt.Sprintf("pdf-page-%d-image-%d%s", page, imageIndex, imageExtension(mimeType)),
					MIMEType: mimeType,
					AltText:  htmlNodeAttribute(node, "alt"),
					Data:     data,
				})
				if err != nil {
					return fmt.Errorf("markdown: store PDF page %d image: %w", page, err)
				}
				node.Attr[index].Val = imageURL
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := nethtml.Render(&out, root); err != nil {
		return "", fmt.Errorf("markdown: render PDF page %d HTML: %w", page, err)
	}
	return out.String(), nil
}

func decodePDFDataImage(source string) (string, []byte, error) {
	if !strings.HasPrefix(source, "data:") {
		return "", nil, fmt.Errorf("invalid data URI")
	}
	header, payload, ok := strings.Cut(strings.TrimPrefix(source, "data:"), ",")
	if !ok {
		return "", nil, fmt.Errorf("invalid data URI")
	}
	parts := strings.Split(header, ";")
	mimeType := strings.TrimSpace(parts[0])
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return "", nil, fmt.Errorf("unsupported data URI MIME type %q", mimeType)
	}
	base64Encoded := false
	for _, part := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(part), "base64") {
			base64Encoded = true
			break
		}
	}
	if !base64Encoded {
		data, err := url.PathUnescape(payload)
		if err != nil {
			return "", nil, err
		}
		return mimeType, []byte(data), nil
	}
	payload = strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return -1
		}
		return character
	}, payload)
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, err
	}
	return mimeType, data, nil
}

func pdfHTMLToMarkdown(engine *lute.Lute, source string) (markdown string, err error) {
	source = strings.TrimSpace(strings.ToValidUTF8(source, ""))
	if source == "" {
		return "", nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("lute HTML2Md panic: %v", recovered)
		}
	}()
	return engine.HTML2Md(source), nil
}

func htmlNodeAttribute(node *nethtml.Node, name string) string {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val
		}
	}
	return ""
}

func cleanPDFMarkdown(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ToValidUTF8(value, ""), "\x00", ""))
}
