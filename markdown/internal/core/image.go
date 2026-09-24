package core

import (
	"net/http"
	"path"
	"strings"
)

// ImageMIMEType resolves the MIME type of an embedded image from its file name,
// falling back to content sniffing.
func ImageMIMEType(name string, data []byte) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".bmp", ".dib":
		return "image/bmp"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".emf":
		return "image/x-emf"
	case ".wmf":
		return "image/wmf"
	default:
		return http.DetectContentType(data)
	}
}
