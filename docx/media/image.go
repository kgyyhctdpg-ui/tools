package media

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mmonterroca/docxgo/v2/domain"
)

type Image struct {
	Path   string
	Data   []byte
	Format domain.ImageFormat
}

func ImageFromFile(path string) (Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Image{}, err
	}

	format := domain.ImageFormat(filepath.Ext(path))
	if len(format) > 0 && format[0] == '.' {
		format = format[1:]
	}
	switch format {
	case domain.ImageFormatJPG:
		format = domain.ImageFormatJPEG
	case domain.ImageFormatPNG, domain.ImageFormatJPEG, domain.ImageFormatGIF:
	default:
		return Image{}, fmt.Errorf("unsupported image format: %s", format)
	}

	return Image{Path: path, Data: data, Format: format}, nil
}
