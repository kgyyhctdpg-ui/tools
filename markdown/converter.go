package markdown

import (
	"context"
	"strings"
)

type converterFunc func(context.Context, []byte, StreamInfo) (*Result, error)

type extensionConverter struct {
	extensions []string
	mimeTypes  []string
	convert    converterFunc
}

func newExtensionConverter(extensions, mimeTypes []string, convert converterFunc) *extensionConverter {
	for index := range extensions {
		extensions[index] = normalizeExtension(extensions[index])
	}
	return &extensionConverter{extensions: extensions, mimeTypes: mimeTypes, convert: convert}
}

func (converter *extensionConverter) Supports(info StreamInfo) bool {
	for _, extension := range converter.extensions {
		if extension == info.Extension {
			return true
		}
	}
	for _, mimeType := range converter.mimeTypes {
		if mimeType == info.MIMEType || (strings.HasSuffix(mimeType, "/") && strings.HasPrefix(info.MIMEType, mimeType)) {
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

func (engine *MarkItDown) registerBuiltins() {
	engine.converters = []Converter{
		newTextConverter(), newHTMLConverter(), newCSVConverter(), newStructuredTextConverter(),
		newDOCXConverter(engine), newXLSXConverter(), newPPTXConverter(), newEPUBConverter(),
		newEMLConverter(engine), newZIPConverter(engine), newImageConverter(engine), newPDFConverter(engine),
	}
}
