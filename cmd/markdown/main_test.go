package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConvertsStandardInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"--name", "input.md", "-"}, strings.NewReader("# Hello"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "# Hello" {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunRequiresSource(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), nil, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage error, code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunListsFormats(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"--formats"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), ".docx") || !strings.Contains(stdout.String(), ".xlsx") {
		t.Fatalf("formats were not listed, code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunSupportsInputAndOutputFlags(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.md")
	output := filepath.Join(directory, "output.md")
	if err := os.WriteFile(input, []byte("# Flag input"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", input, "-o", output}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "# Flag input" {
		t.Fatalf("unexpected output file: %q", data)
	}
}

func TestRunRejectsDuplicateInputSources(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", "input.md", "other.md"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "both -i and a positional argument") {
		t.Fatalf("expected duplicate input error, code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunStoresImagesBesideMarkdownOutput(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "photo.png")
	output := filepath.Join(directory, "report.md")
	imageData := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 1, 2, 3}
	if err := os.WriteFile(input, imageData, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", input, "-o", output}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	markdownData, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	assetsFolder := fmt.Sprintf("%x", md5.Sum([]byte("photo.png")))
	if strings.Contains(string(markdownData), "data:image/") || !strings.Contains(string(markdownData), assetsFolder+"/photo.png") {
		t.Fatalf("image should use a file link, got: %s", markdownData)
	}
	storedImage, err := os.ReadFile(filepath.Join(directory, assetsFolder, "photo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(storedImage, imageData) {
		t.Fatalf("stored image differs from source: %v", storedImage)
	}
}

func TestAssetsFolderUsesInputFileNameMD5(t *testing.T) {
	output := filepath.Join("docs", "different-name.md")
	directory, prefix := markdownAssetsLocation(filepath.Join("source", "report.docx"), output, "stdin.txt")
	want := fmt.Sprintf("%x", md5.Sum([]byte("report.docx")))
	if prefix != want || directory != filepath.Join("docs", want) {
		t.Fatalf("unexpected assets location: directory=%q prefix=%q want=%q", directory, prefix, want)
	}
}

func TestRunConvertsPDFLocally(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.pdf")
	if err := os.WriteFile(input, minimalTextPDF("CLI local PDF body"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", input}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CLI local PDF body") {
		t.Fatalf("local output did not include PDF text: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "warning:") {
		t.Fatalf("a text PDF should not warn, stderr=%q", stderr.String())
	}
}

func TestRunHonoursPDFPageRange(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.pdf")
	if err := os.WriteFile(input, multiPageTextPDF("first page body", "second page body"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", input, "--pdf-first-page", "2"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "first page body") || !strings.Contains(stdout.String(), "second page body") {
		t.Fatalf("page range was not honoured: %q", stdout.String())
	}
}

func TestRunRejectsPDFPageRangePastTheEnd(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.pdf")
	if err := os.WriteFile(input, minimalTextPDF("only page"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", input, "--pdf-first-page", "5"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "past the last page") {
		t.Fatalf("expected page range error, code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunUsesExplicitAssetsDirectory(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "photo.png")
	output := filepath.Join(directory, "report.md")
	imageData := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 4, 5, 6}
	if err := os.WriteFile(input, imageData, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{
		"-i", input, "-o", output,
		"--assets-dir", "images",
		"--assets-prefix", "assets",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	markdownData, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdownData), "assets/photo.png") {
		t.Fatalf("expected the custom link prefix, got: %s", markdownData)
	}
	if _, err := os.Stat(filepath.Join(directory, "images", "photo.png")); err != nil {
		t.Fatalf("image was not stored in the custom directory: %v", err)
	}
}

// minimalTextPDF builds a one-page PDF whose single text run is text.
func minimalTextPDF(text string) []byte {
	return multiPageTextPDF(text)
}

// multiPageTextPDF builds a PDF with one text page per entry, which keeps the
// page-range handling testable without shipping binary fixtures.
func multiPageTextPDF(pages ...string) []byte {
	if len(pages) == 0 {
		pages = []string{""}
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	kids := make([]string, 0, len(pages))
	for _, text := range pages {
		pageObject := len(objects) + 1
		contentObject := pageObject + 1
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObject))
		objects = append(objects,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentObject),
			contentStream(text),
		)
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages))

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

func contentStream(text string) string {
	stream := "BT\n/F1 18 Tf\n72 720 Td\n(" + escapePDFString(text) + ") Tj\nET\n"
	return fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)
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

func TestParseMarkdownSize(t *testing.T) {
	for _, testCase := range []struct {
		value string
		want  int64
	}{
		{"", 0},
		{"1024", 1024},
		{"512KB", 512 << 10},
		{"512kb", 512 << 10},
		{"128MB", 128 << 20},
		{"1GB", 1 << 30},
		{"1gb", 1 << 30},
		{"2TB", 2 << 40},
		{"2048B", 2048},
		{"0", 0},
	} {
		got, err := parseMarkdownSize(testCase.value)
		if err != nil || got != testCase.want {
			t.Fatalf("parseMarkdownSize(%q) = %d, %v; want %d", testCase.value, got, err, testCase.want)
		}
	}
	for _, bad := range []string{"abc", "-1", "10XB"} {
		if _, err := parseMarkdownSize(bad); err == nil {
			t.Fatalf("parseMarkdownSize(%q) should fail", bad)
		}
	}
}

func TestRunRejectsInvalidMaxInputSize(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"-i", "report.pdf", "-max-input-size", "huge"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "-max-input-size") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunConvertsWithoutAnInputLimit(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "vector.pdf")
	if err := os.WriteFile(source, minimalTextPDF("Large enough input"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "out.md")
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"-i", source, "-o", output}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Large enough input") {
		t.Fatalf("conversion lost the text: %s", data)
	}
}
