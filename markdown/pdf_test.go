package markdown

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// ---- PDF end-to-end conversion --------------------------------------------

func TestPDFConvertsTextLocally(t *testing.T) {
	result, err := New().ConvertReader(
		context.Background(),
		bytes.NewReader(minimalTextPDF("Local PDF body")),
		StreamInfo{Name: "report.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Markdown, "Local PDF body") {
		t.Fatalf("Markdown did not include PDF text:\n%s", result.Markdown)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("a text PDF should not warn: %#v", result.Warnings)
	}
	if result.Metadata["pdf_content_type"] != "text" || result.Metadata["pdf_page_count"] != "1" {
		t.Fatalf("unexpected metadata: %#v", result.Metadata)
	}
}

func TestPDFRendersPageWithoutTextOrImages(t *testing.T) {
	var rendered Image
	result, err := New(
		WithPDFOptions(PDFOptions{RenderDPI: 96}),
		WithImageHandler(func(_ context.Context, image Image) (string, error) {
			rendered = image
			return "assets/" + image.Name, nil
		}),
	).ConvertReader(
		context.Background(),
		bytes.NewReader(minimalTextPDF("")),
		StreamInfo{Name: "vector.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata["pdf_content_type"] != "image" {
		t.Fatalf("PDF should be detected as image-only: %#v", result.Metadata)
	}
	if rendered.Name != "pdf-page-1.jpg" || len(rendered.Data) == 0 {
		t.Fatalf("page was not rendered as a picture: %#v", rendered)
	}
	if !bytes.HasPrefix(rendered.Data, []byte{0xFF, 0xD8}) {
		t.Fatalf("a rendered page should default to JPEG: %x", rendered.Data[:2])
	}
	if !strings.Contains(result.Markdown, "assets/pdf-page-1.jpg") {
		t.Fatalf("rendered page was not linked: %q", result.Markdown)
	}
	if result.Metadata["pdf_rendered_pages"] != "1" {
		t.Fatalf("rendered page was not counted: %#v", result.Metadata)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("rendering a page should be reported as a warning")
	}
}

func TestPDFRendersPageAsJPEG(t *testing.T) {
	var rendered Image
	_, err := New(
		WithPDFOptions(PDFOptions{RenderFormat: PDFImageFormatJPEG, JPEGQuality: 70, RenderDPI: 72}),
		WithImageHandler(func(_ context.Context, image Image) (string, error) {
			rendered = image
			return image.Name, nil
		}),
	).ConvertReader(
		context.Background(),
		bytes.NewReader(minimalTextPDF("")),
		StreamInfo{Name: "vector.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if rendered.MIMEType != "image/jpeg" || rendered.Name != "pdf-page-1.jpg" {
		t.Fatalf("expected a JPEG page render: %#v", rendered)
	}
	if !bytes.HasPrefix(rendered.Data, []byte{0xFF, 0xD8}) {
		t.Fatalf("rendered data is not JPEG: %x", rendered.Data[:2])
	}
}

func TestPDFOptionDisablesPageRenderFallback(t *testing.T) {
	result, err := New(WithPDFOptions(PDFOptions{DisablePageRenderFallback: true})).ConvertReader(
		context.Background(),
		bytes.NewReader(minimalTextPDF("")),
		StreamInfo{Name: "vector.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.Markdown) != "" {
		t.Fatalf("expected an empty conversion, got %q", result.Markdown)
	}
	if result.Metadata["pdf_rendered_pages"] != "0" {
		t.Fatalf("no page should have been rendered: %#v", result.Metadata)
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
