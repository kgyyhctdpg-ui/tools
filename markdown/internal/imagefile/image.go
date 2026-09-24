package imagefile

import (
	"context"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"path/filepath"
	"strings"
)

// NewConverter builds the standalone-image converter.
func NewConverter(settings *core.Settings) core.Converter {
	return core.NewExtensionConverter(
		[]string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".tif", ".tiff"},
		[]string{"image/"},
		func(ctx context.Context, data []byte, info core.StreamInfo) (*core.Result, error) {
			name := info.Name
			if name == "" {
				name = "image" + info.Extension
			}
			mimeType := info.MIMEType
			if mimeType == "" {
				mimeType = core.ImageMIMEType(name, data)
			}
			handler := settings.ImageHandlerOrDefault()
			imageURL, err := handler(ctx, core.Image{Name: name, MIMEType: mimeType, AltText: name, Data: data})
			if err != nil {
				return nil, err
			}
			altText := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			return &core.Result{Title: altText, Markdown: core.MarkdownImage(altText, imageURL)}, nil
		},
	)
}
