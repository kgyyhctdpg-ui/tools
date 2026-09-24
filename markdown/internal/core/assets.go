package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// NewFileImageHandler stores extracted images in directory and returns links
// relative to the generated Markdown file. Images with identical content
// share one file even when their source names differ; conflicting names with
// different content receive a numeric suffix.
func NewFileImageHandler(directory, linkPrefix string) ImageHandler {
	var (
		mu             sync.Mutex
		contentFiles   map[[sha256.Size]byte]string
		contentIndexed bool
	)
	return func(ctx context.Context, image Image) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if strings.TrimSpace(directory) == "" {
			return "", errors.New("markdown: image directory is empty")
		}
		name := SafeImageFileName(image.Name, image.MIMEType)

		mu.Lock()
		defer mu.Unlock()
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return "", fmt.Errorf("markdown: create image directory %q: %w", directory, err)
		}
		if !contentIndexed {
			var err error
			contentFiles, err = IndexImageFilesByContent(directory)
			if err != nil {
				return "", err
			}
			contentIndexed = true
		}
		digest := sha256.Sum256(image.Data)
		if existingName, ok := contentFiles[digest]; ok {
			existing, err := os.ReadFile(filepath.Join(directory, existingName))
			if err == nil && bytes.Equal(existing, image.Data) {
				return ImageFileLink(linkPrefix, existingName), nil
			}
			delete(contentFiles, digest)
		}
		name, err := AvailableImageFileName(directory, name, image.Data)
		if err != nil {
			return "", err
		}
		target := filepath.Join(directory, name)
		if existing, readErr := os.ReadFile(target); readErr == nil && bytes.Equal(existing, image.Data) {
			contentFiles[digest] = name
			return ImageFileLink(linkPrefix, name), nil
		}
		if err := WriteImageFile(target, image.Data); err != nil {
			return "", err
		}
		contentFiles[digest] = name
		return ImageFileLink(linkPrefix, name), nil
	}
}

func IndexImageFilesByContent(directory string) (map[[sha256.Size]byte]string, error) {
	files := make(map[[sha256.Size]byte]string)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("markdown: inspect image directory %q: %w", directory, err)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || strings.HasPrefix(entry.Name(), ".markdown-image-") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("markdown: inspect image file %q: %w", entry.Name(), err)
		}
		digest := sha256.Sum256(data)
		if _, exists := files[digest]; !exists {
			files[digest] = entry.Name()
		}
	}
	return files, nil
}

func SafeImageFileName(name, mimeType string) string {
	name = filepath.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	name = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || strings.ContainsRune(`/\\:*?"<>|`, character) {
			return '_'
		}
		return character
	}, name)
	if name == "" || name == "." {
		name = "image"
	}
	if filepath.Ext(name) == "" {
		name += ImageExtension(mimeType)
	}
	return name
}

func ImageExtension(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	}
	if extensions, err := mime.ExtensionsByType(mimeType); err == nil && len(extensions) > 0 {
		return extensions[0]
	}
	return ".bin"
}

func AvailableImageFileName(directory, name string, data []byte) (string, error) {
	extension := filepath.Ext(name)
	base := strings.TrimSuffix(name, extension)
	for index := 1; ; index++ {
		candidate := name
		if index > 1 {
			candidate = fmt.Sprintf("%s-%d%s", base, index, extension)
		}
		existing, err := os.ReadFile(filepath.Join(directory, candidate))
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("markdown: inspect image file %q: %w", candidate, err)
		}
		if bytes.Equal(existing, data) {
			return candidate, nil
		}
	}
}

func WriteImageFile(target string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".markdown-image-*")
	if err != nil {
		return fmt.Errorf("markdown: create image file %q: %w", target, err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("markdown: prepare image file %q: %w", target, err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("markdown: write image file %q: %w", target, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("markdown: close image file %q: %w", target, err)
	}
	if err := os.Rename(temporaryName, target); err != nil {
		return fmt.Errorf("markdown: store image file %q: %w", target, err)
	}
	return nil
}

func ImageFileLink(prefix, name string) string {
	name = url.PathEscape(name)
	prefix = strings.TrimRight(filepath.ToSlash(strings.TrimSpace(prefix)), "/")
	if prefix == "" || prefix == "." {
		return name
	}
	return prefix + "/" + name
}
