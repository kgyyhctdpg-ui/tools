package core

import (
	"context"
	"strings"
)

// ConverterFunc is the conversion function an extension converter wraps.
type ConverterFunc func(context.Context, []byte, StreamInfo) (*Result, error)

type extensionConverter struct {
	extensions []string
	mimeTypes  []string
	convert    ConverterFunc
}

// NewExtensionConverter builds a Converter that claims the given extensions and
// MIME types.
func NewExtensionConverter(extensions, mimeTypes []string, convert ConverterFunc) Converter {
	normalized := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		normalized = append(normalized, normalizeExtension(extension))
	}
	mimes := make([]string, 0, len(mimeTypes))
	for _, mimeType := range mimeTypes {
		mimes = append(mimes, strings.ToLower(strings.TrimSpace(mimeType)))
	}
	return &extensionConverter{extensions: normalized, mimeTypes: mimes, convert: convert}
}

func (converter *extensionConverter) Supports(info StreamInfo) bool {
	extension := normalizeExtension(info.Extension)
	if extension == "" {
		extension = normalizeExtension(extensionOf(info.Name))
	}
	for _, candidate := range converter.extensions {
		if candidate != "" && candidate == extension {
			return true
		}
	}
	mediaType := strings.ToLower(strings.TrimSpace(info.MIMEType))
	if index := strings.IndexByte(mediaType, ';'); index >= 0 {
		mediaType = strings.TrimSpace(mediaType[:index])
	}
	for _, candidate := range converter.mimeTypes {
		if candidate == "" {
			continue
		}
		// A trailing slash registers a whole MIME family, e.g. "image/".
		if candidate == mediaType || (strings.HasSuffix(candidate, "/") && strings.HasPrefix(mediaType, candidate)) {
			return true
		}
	}
	return false
}

func (converter *extensionConverter) Convert(ctx context.Context, data []byte, info StreamInfo) (*Result, error) {
	return converter.convert(ctx, data, info)
}

func (converter *extensionConverter) Extensions() []string {
	return append([]string(nil), converter.extensions...)
}

func normalizeExtension(extension string) string {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	return extension
}

func extensionOf(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return ""
	}
	return name[index:]
}
