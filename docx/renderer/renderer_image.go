package renderer

import (
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/util"
	"github.com/scoming-dev/tools/docx/media"
)

func (r *DocxRenderer) renderImage(node *ast.Node, entering bool) ast.WalkStatus {
	if entering {
		if 0 == r.DisableTags {
			destTokens := node.ChildByType(ast.NodeLinkDest).Tokens
			src := util.BytesToStr(destTokens)
			src, ok, isTemp := r.DownloadImg(src)
			if ok {
				imgPath := src
				img, err := media.ImageFromFile(imgPath)
				if err != nil {
					log.Printf("Failed to load image [%s]: %s", imgPath, err)
					r.DisableTags++
					return ast.WalkContinue
				}

				imgRef, err := r.doc.AddImage(img)
				if err != nil {
					log.Printf("Failed to add image to document [%s]: %s", imgPath, err)
					r.DisableTags++
					return ast.WalkContinue
				}

				imgPara := r.doc.AddParagraph()
				imgParaProps := imgPara.Properties()
				imgParaProps.SetAlignment(r.config.Image.Alignment)

				spacing := imgParaProps.Spacing()
				spacing.SetBefore(r.config.Image.SpacingBefore)
				spacing.SetAfter(r.config.Image.SpacingAfter)

				imgRun := imgPara.AddRun()
				inline, err := imgRun.AddDrawingInline(imgRef)
				if err != nil {
					log.Printf("Failed to add drawing inline [%s]: %s", imgPath, err)
					r.DisableTags++
					return ast.WalkContinue
				}

				width, height := r.getImgSize(imgPath)
				inline.SetSize(float64(width), float64(height))

				if isTemp {
					r.files = append(r.files, imgPath)
				}
			}
		}
		r.DisableTags++
		return ast.WalkContinue
	}

	r.DisableTags--
	return ast.WalkContinue
}

// DownloadImg resolves an image source to a local file, downloading or decoding it when needed.
func (r *DocxRenderer) DownloadImg(src string) (localPath string, ok, isTemp bool) {
	if r.isBase64Image(src) {
		data, fileExt, err := r.parseBase64Image(src)
		if err != nil {
			log.Printf("Failed to parse base64 image: %s", err)
			return src, false, false
		}

		tempFile, err := os.CreateTemp("", "lute-docx-base64*"+fileExt)
		if err != nil {
			log.Printf("Failed to create temp file for base64 image: %s", err)
			return src, false, false
		}

		_, err = tempFile.Write(data)
		if err != nil {
			log.Printf("Failed to write base64 image data: %s", err)
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
			return src, false, false
		}

		_ = tempFile.Close()
		return tempFile.Name(), true, true
	}

	if strings.HasPrefix(src, "//") {
		src = "https:" + src
	}

	u, err := url.Parse(src)
	if nil != err {
		return src, true, false
	}

	if !strings.HasPrefix(u.Scheme, "http") {
		return src, true, false
	}

	u, _ = url.Parse(src)

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	req := &http.Request{
		URL: u,
	}
	resp, err := client.Do(req)
	if nil != err {
		log.Printf("download image [%s] failed: %s", src, err)
		return src, false, false
	}
	defer resp.Body.Close()
	if 200 != resp.StatusCode {
		log.Printf("download image [%s] failed, status code is [%d]", src, resp.StatusCode)
		return src, false, false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read image [%s] failed: %s", src, err)
		return src, false, false
	}
	file, err := os.CreateTemp("", "lute-docx.img.*")
	if nil != err {
		log.Printf("create temp image [%s] failed: %s", src, err)
		return src, false, false
	}
	_, err = file.Write(data)
	if nil != err {
		log.Printf("write temp image [%s] failed: %s", src, err)
		_ = file.Close()
		_ = os.Remove(file.Name())
		return src, false, false
	}
	_ = file.Close()
	return file.Name(), true, true
}

func (r *DocxRenderer) getDocumentContentWidth() float64 {
	pageWidthPx := 210.0 * 96 / 25.4

	marginPx := r.margin * 96 / 25.4
	if marginPx == 0 {
		marginPx = 25.0 * 96 / 25.4
	}

	contentWidth := pageWidthPx - (marginPx * 2)
	if contentWidth < 300 {
		contentWidth = 300
	}

	return contentWidth
}

func (r *DocxRenderer) constrainImageSize(width, height float64) (float64, float64) {
	maxWidth := r.getDocumentContentWidth()

	if width <= maxWidth {
		return width, height
	}

	scale := maxWidth / width
	newWidth := maxWidth
	newHeight := height * scale

	return newWidth, newHeight
}

func (r *DocxRenderer) getImgSize(imgPath string) (width, height float64) {
	if r.IsSVG(imgPath) {
		width, height = r.getSVGSize(imgPath)
		return r.constrainImageSize(width, height)
	}

	file, err := os.Open(imgPath)
	if nil != err {
		log.Printf("failed to open image file [%s]: %s", imgPath, err)
		return 400, 300
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if nil != err {
		log.Printf("failed to decode image file [%s]: %s", imgPath, err)
		return 400, 300
	}

	imageRect := img.Bounds()
	k := 1
	w := -128
	h := -128
	if w < 0 {
		w = -imageRect.Dx() * 72 / w / k
	}
	if h < 0 {
		h = -imageRect.Dy() * 72 / h / k
	}
	if w == 0 {
		w = h * imageRect.Dx() / imageRect.Dy()
	}
	if h == 0 {
		h = w * imageRect.Dy() / imageRect.Dx()
	}

	width = float64(w)
	height = float64(h)

	return r.constrainImageSize(width, height)
}

// IsSVG reports whether a local file appears to be SVG.
func (r *DocxRenderer) IsSVG(imgPath string) bool {
	ext := strings.ToLower(filepath.Ext(imgPath))
	if ext == ".svg" {
		return true
	}

	data, err := os.ReadFile(imgPath)
	if err != nil {
		return false
	}

	content := string(data)
	return strings.Contains(content, "<svg")
}

// getSVGSize extracts SVG dimensions from width and height attributes or viewBox.
func (r *DocxRenderer) getSVGSize(imgPath string) (width, height float64) {
	data, err := os.ReadFile(imgPath)
	if err != nil {
		log.Printf("failed to read local SVG file [%s]: %s", imgPath, err)
		return 400, 300
	}
	content := string(data)

	width = svgLengthAttribute(content, "width")
	height = svgLengthAttribute(content, "height")

	if width == 0 || height == 0 {
		viewBoxWidth, viewBoxHeight := svgViewBoxSize(content)
		if width == 0 {
			width = viewBoxWidth
		}
		if height == 0 {
			height = viewBoxHeight
		}
	}

	if width == 0 {
		width = 400
	}
	if height == 0 {
		height = 300
	}

	return width, height
}

func svgLengthAttribute(content, attr string) float64 {
	pattern := `(?i)\b` + regexp.QuoteMeta(attr) + `\s*=\s*["']?([^"'\s>]+)["']?`
	matches := regexp.MustCompile(pattern).FindStringSubmatch(content)
	if len(matches) < 2 {
		return 0
	}
	return parseSVGLength(matches[1])
}

func parseSVGLength(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasSuffix(value, "%") {
		return 0
	}

	unit := ""
	for _, suffix := range []string{"px", "pt", "in", "mm", "cm"} {
		if strings.HasSuffix(strings.ToLower(value), suffix) {
			unit = suffix
			value = strings.TrimSpace(value[:len(value)-len(suffix)])
			break
		}
	}

	length, err := strconv.ParseFloat(value, 64)
	if err != nil || length <= 0 {
		return 0
	}

	switch unit {
	case "in":
		return length * 72
	case "mm":
		return length * 72 / 25.4
	case "cm":
		return length * 72 / 2.54
	default:
		return length
	}
}

func svgViewBoxSize(content string) (width, height float64) {
	matches := regexp.MustCompile(`(?i)viewBox\s*=\s*["']?([^"']+)["']?`).FindStringSubmatch(content)
	if len(matches) < 2 {
		return 0, 0
	}

	parts := strings.FieldsFunc(matches[1], func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(parts) < 4 {
		return 0, 0
	}

	width, _ = strconv.ParseFloat(parts[2], 64)
	height, _ = strconv.ParseFloat(parts[3], 64)
	return width, height
}

func (r *DocxRenderer) isBase64Image(src string) bool {
	return strings.HasPrefix(src, "data:image/")
}

func (r *DocxRenderer) parseBase64Image(src string) (data []byte, fileExt string, err error) {
	if !strings.HasPrefix(src, "data:image/") {
		return nil, "", fmt.Errorf("not a valid base64 image format")
	}

	base64Index := strings.Index(src, "base64,")
	if base64Index == -1 {
		return nil, "", fmt.Errorf("no base64 data found")
	}

	mimeType := src[5 : base64Index-1]

	switch {
	case strings.Contains(mimeType, "png"):
		fileExt = ".png"
	case strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg"):
		fileExt = ".jpg"
	case strings.Contains(mimeType, "gif"):
		fileExt = ".gif"
	case strings.Contains(mimeType, "svg+xml"):
		fileExt = ".svg"
	case strings.Contains(mimeType, "webp"):
		fileExt = ".webp"
	case strings.Contains(mimeType, "bmp"):
		fileExt = ".bmp"
	default:
		fileExt = ".png"
		log.Printf("Unknown image MIME type [%s], defaulting to PNG", mimeType)
	}

	base64Data := src[base64Index+7:]
	data, err = base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 data: %s", err)
	}

	return data, fileExt, nil
}
