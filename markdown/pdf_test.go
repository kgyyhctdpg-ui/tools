package markdown

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestPDFUsesGoFitzLuteFallbackWhenMinerUIsNotConfigured(t *testing.T) {
	result, err := New().ConvertReader(
		context.Background(),
		bytes.NewReader(minimalTextPDF("Fallback PDF body")),
		StreamInfo{Name: "report.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Markdown, "Fallback PDF body") {
		t.Fatalf("fallback Markdown did not include PDF text:\n%s", result.Markdown)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "MinerU is not configured") {
		t.Fatalf("fallback warning was not returned: %#v", result.Warnings)
	}
	if len(result.Metadata) != 2 || result.Metadata["pdf_content_type"] != "text" || result.Metadata["pdf_page_count"] != "1" {
		t.Fatalf("unexpected fallback metadata: %#v", result.Metadata)
	}
}

func TestPDFDetectsImagePDFWhenNoTextIsExtracted(t *testing.T) {
	result, err := New().ConvertReader(
		context.Background(),
		bytes.NewReader(minimalTextPDF("")),
		StreamInfo{Name: "scan.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Metadata) != 2 || result.Metadata["pdf_content_type"] != "image" || result.Metadata["pdf_page_count"] != "1" {
		t.Fatalf("PDF should be detected as image-only: %#v", result.Metadata)
	}
}

func TestPDFExternalizesDataURIImagesBeforeMarkdown(t *testing.T) {
	imageData := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	called := false
	html, err := externalizePDFHTMLImages(
		context.Background(),
		`<html><body><p>Chart</p><img alt="chart" src="data:image/png;base64,`+base64.StdEncoding.EncodeToString(imageData)+`"></body></html>`,
		2,
		func(_ context.Context, image Image) (string, error) {
			called = true
			if image.Name != "pdf-page-2-image-1.png" || image.MIMEType != "image/png" || image.AltText != "chart" {
				t.Fatalf("unexpected image metadata: %#v", image)
			}
			if !bytes.Equal(image.Data, imageData) {
				t.Fatalf("unexpected image data: %v", image.Data)
			}
			return "assets/pdf-page-2-image-1.png", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("image handler was not called")
	}
	if strings.Contains(html, "data:image/") || !strings.Contains(html, `src="assets/pdf-page-2-image-1.png"`) {
		t.Fatalf("data URI image was not externalized:\n%s", html)
	}
	markdown, err := pdfHTMLToMarkdown(newPDFLuteEngine(), html)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(markdown, "data:image/") || !strings.Contains(markdown, "assets/pdf-page-2-image-1.png") {
		t.Fatalf("Markdown should use external image link:\n%s", markdown)
	}
}

func TestPDFPageWithManyImagesAndNoTextRendersWholePage(t *testing.T) {
	html := `<html><body><img src="data:image/png;base64,a"><img src="data:image/png;base64,b"></body></html>`
	if !shouldRenderPDFPageAsImage("", html) {
		t.Fatal("expected image-only page with multiple image fragments to render as one page image")
	}
	if shouldRenderPDFPageAsImage("caption", html) {
		t.Fatal("page with text should keep HTML extraction path")
	}
	if shouldRenderPDFPageAsImage("", `<html><body><img src="data:image/png;base64,a"></body></html>`) {
		t.Fatal("single image page should not be forced through whole-page rendering")
	}
}

func TestPDFAlwaysUsesConfiguredHandler(t *testing.T) {
	called := false
	converter := New(WithPDFHandler(func(_ context.Context, data []byte, info StreamInfo, _ ImageHandler) (*Result, error) {
		called = bytes.Equal(data, []byte("pdf")) && info.Name == "report.pdf"
		return &Result{Markdown: "MinerU output"}, nil
	}))
	result, err := converter.ConvertReader(
		context.Background(),
		bytes.NewReader([]byte("pdf")),
		StreamInfo{Name: "report.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !called || result.Markdown != "MinerU output" {
		t.Fatalf("PDF handler was not used: called=%v result=%#v", called, result)
	}
}

func minimalTextPDF(text string) []byte {
	stream := "BT\n/F1 18 Tf\n72 720 Td\n(" + escapePDFString(text) + ") Tj\nET\n"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return out.Bytes()
}

func escapePDFString(value string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`(`, `\(`,
		`)`, `\)`,
		"\r", `\r`,
		"\n", `\n`,
	).Replace(value)
}
