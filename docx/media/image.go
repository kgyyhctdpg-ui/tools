// Package media contains image loading helpers used by DOCX rendering.
package media

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// Image is a loaded image with its source path, bytes, and DOCX format.
type Image struct {
	Path   string
	Data   []byte
	Format domain.ImageFormat
}

// ImageFromFile loads a supported local image file.
func ImageFromFile(path string) (Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Image{}, err
	}

	format := detectImageFormat(path, data)
	if format == "" {
		return Image{}, fmt.Errorf("unsupported image format: %s", filepath.Ext(path))
	}

	return Image{Path: path, Data: data, Format: format}, nil
}

func detectImageFormat(path string, data []byte) domain.ImageFormat {
	format := normalizeImageFormat(domain.ImageFormat(filepath.Ext(path)))
	if format != "" {
		return format
	}

	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png":
		return domain.ImageFormatPNG
	case "image/jpeg":
		return domain.ImageFormatJPEG
	case "image/gif":
		return domain.ImageFormatGIF
	case "image/svg+xml":
		return domain.ImageFormatSVG
	}

	if looksLikeSVG(data) {
		return domain.ImageFormatSVG
	}
	return ""
}

func normalizeImageFormat(format domain.ImageFormat) domain.ImageFormat {
	if len(format) > 0 && format[0] == '.' {
		format = format[1:]
	}
	normalized := domain.ImageFormat(strings.ToLower(string(format)))
	switch normalized {
	case domain.ImageFormatJPG:
		return domain.ImageFormatJPEG
	case domain.ImageFormatPNG, domain.ImageFormatJPEG, domain.ImageFormatGIF, domain.ImageFormatSVG:
		return normalized
	default:
		return ""
	}
}

func looksLikeSVG(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return bytes.HasPrefix(trimmed, []byte("<svg")) ||
		bytes.HasPrefix(trimmed, []byte("<?xml")) && bytes.Contains(trimmed, []byte("<svg"))
}
