package markdown

import (
	"context"
	"path/filepath"
	"strings"
)

func newImageConverter(engine *MarkItDown) Converter {
	return newExtensionConverter(
		[]string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".tif", ".tiff"},
		[]string{"image/"},
		func(ctx context.Context, data []byte, info StreamInfo) (*Result, error) {
			name := info.Name
			if name == "" {
				name = "image" + info.Extension
			}
			mimeType := info.MIMEType
			if mimeType == "" {
				mimeType = imageMIMEType(name, data)
			}
			handler := engine.imageHandler
			if handler == nil {
				handler = DataURIImageHandler
			}
			imageURL, err := handler(ctx, Image{Name: name, MIMEType: mimeType, AltText: name, Data: data})
			if err != nil {
				return nil, err
			}
			altText := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
			return &Result{Title: altText, Markdown: markdownImage(altText, imageURL)}, nil
		},
	)
}
