// Package imagex provides lightweight image helpers.
package imagex

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Config contains image dimensions and format.
type Config struct {
	Width  int
	Height int
	Format string
	MIME   string
}

// DecodeConfig reads image dimensions and format without decoding all pixels.
func DecodeConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Width:  cfg.Width,
		Height: cfg.Height,
		Format: format,
		MIME:   MIMEByFormat(format),
	}, nil
}

// MIME detects an image MIME type by reading its leading bytes.
func MIME(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

// MIMEByFormat returns a MIME type for a Go image format name or extension.
func MIMEByFormat(format string) string {
	switch strings.ToLower(strings.TrimPrefix(format, ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// ToBase64 encodes an image file using standard base64.
func ToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ToDataURI encodes an image file as a data URI.
func ToDataURI(path string) (string, error) {
	encoded, err := ToBase64(path)
	if err != nil {
		return "", err
	}
	mimeType, err := MIME(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// FromBase64 writes base64 image data into path. Data URI prefixes are accepted.
func FromBase64(data, path string) error {
	decoded, err := DecodeBase64(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, decoded, 0o600)
}

// DecodeBase64 decodes raw base64 or a base64 data URI.
func DecodeBase64(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if idx := strings.Index(data, ","); idx >= 0 && strings.HasPrefix(strings.ToLower(data[:idx]), "data:") {
		data = data[idx+1:]
	}
	return base64.StdEncoding.DecodeString(data)
}

// CompressJPEG decodes srcPath and writes it as JPEG to dstPath.
func CompressJPEG(srcPath, dstPath string, quality int) error {
	img, err := Open(srcPath)
	if err != nil {
		return err
	}
	return SaveJPEG(img, dstPath, quality)
}

// ResizeFit resizes srcPath proportionally to fit within maxWidth/maxHeight and
// saves it to dstPath. If one max dimension is <= 0, it is derived from the
// other dimension.
func ResizeFit(srcPath, dstPath string, maxWidth, maxHeight, quality int) error {
	img, err := Open(srcPath)
	if err != nil {
		return err
	}
	resized, err := ResizeImageFit(img, maxWidth, maxHeight)
	if err != nil {
		return err
	}
	return Save(resized, dstPath, quality)
}

// Open decodes an image file.
func Open(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}

// Save writes img based on dstPath extension. JPEG quality defaults to 85 when
// quality is outside 1..100.
func Save(img image.Image, dstPath string, quality int) error {
	ext := strings.ToLower(filepath.Ext(dstPath))
	switch ext {
	case ".jpg", ".jpeg":
		return SaveJPEG(img, dstPath, quality)
	case ".gif":
		return saveWithEncoder(dstPath, func(w io.Writer) error {
			return gif.Encode(w, img, nil)
		})
	default:
		return saveWithEncoder(dstPath, func(w io.Writer) error {
			return png.Encode(w, img)
		})
	}
}

// SaveJPEG writes img as JPEG.
func SaveJPEG(img image.Image, dstPath string, quality int) error {
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	return saveWithEncoder(dstPath, func(w io.Writer) error {
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	})
}

// ResizeImageFit resizes img proportionally to fit within maxWidth/maxHeight.
func ResizeImageFit(img image.Image, maxWidth, maxHeight int) (image.Image, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("imagex: image has invalid bounds")
	}
	if maxWidth <= 0 && maxHeight <= 0 {
		return nil, fmt.Errorf("imagex: max width or max height is required")
	}
	if maxWidth <= 0 {
		maxWidth = width * maxHeight / height
	}
	if maxHeight <= 0 {
		maxHeight = height * maxWidth / width
	}

	scaleW := float64(maxWidth) / float64(width)
	scaleH := float64(maxHeight) / float64(height)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	if scale > 1 {
		scale = 1
	}

	newWidth := int(float64(width)*scale + 0.5)
	newHeight := int(float64(height)*scale + 0.5)
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}
	return ResizeNearest(img, newWidth, newHeight), nil
}

// ResizeNearest resizes img using nearest-neighbor sampling.
func ResizeNearest(src image.Image, width, height int) image.Image {
	srcBounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y*srcBounds.Dy()/height
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x*srcBounds.Dx()/width
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

// Encode encodes img to memory based on format.
func Encode(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		if quality <= 0 || quality > 100 {
			quality = 85
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	case "gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, err
		}
	default:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func saveWithEncoder(dstPath string, encode func(io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := encode(file); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
