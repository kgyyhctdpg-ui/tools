package imagex

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigBase64AndDataURI(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.png")
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := Save(img, src, 0); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cfg, err := DecodeConfig(src)
	if err != nil {
		t.Fatalf("DecodeConfig failed: %v", err)
	}
	if cfg.Width != 20 || cfg.Height != 10 || cfg.Format != "png" {
		t.Fatalf("config = %#v, want 20x10 png", cfg)
	}

	uri, err := ToDataURI(src)
	if err != nil {
		t.Fatalf("ToDataURI failed: %v", err)
	}
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("uri = %q, want png data uri", uri[:22])
	}

	dst := filepath.Join(t.TempDir(), "dst.png")
	if err := FromBase64(uri, dst); err != nil {
		t.Fatalf("FromBase64 failed: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("written image missing: %v", err)
	}
}

func TestResizeFitAndCompressJPEG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	if err := Save(img, src, 0); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	resized := filepath.Join(dir, "small.png")
	if err := ResizeFit(src, resized, 20, 20, 0); err != nil {
		t.Fatalf("ResizeFit failed: %v", err)
	}
	cfg, err := DecodeConfig(resized)
	if err != nil {
		t.Fatalf("DecodeConfig failed: %v", err)
	}
	if cfg.Width != 20 || cfg.Height != 10 {
		t.Fatalf("resized = %dx%d, want 20x10", cfg.Width, cfg.Height)
	}

	jpg := filepath.Join(dir, "out.jpg")
	if err := CompressJPEG(src, jpg, 70); err != nil {
		t.Fatalf("CompressJPEG failed: %v", err)
	}
	mimeType, err := MIME(jpg)
	if err != nil {
		t.Fatalf("MIME failed: %v", err)
	}
	if mimeType != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", mimeType)
	}
}

func TestEncode(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	data, err := Encode(img, "jpeg", 80)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encoded data is empty")
	}
}
