package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestRunUsesSelfHostedMinerU(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/file_parse" || request.FormValue("return_md") != "true" {
			t.Fatalf("unexpected MinerU request: path=%q return_md=%q", request.URL.Path, request.FormValue("return_md"))
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"results": map[string]any{
				"input": map[string]any{"md_content": "# Parsed by MinerU"},
			},
		})
	}))
	defer server.Close()

	directory := t.TempDir()
	input := filepath.Join(directory, "input.pdf")
	output := filepath.Join(directory, "output.md")
	if err := os.WriteFile(input, []byte("remote parser input"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{
		"-i", input,
		"-o", output,
		"--mineru-url", server.URL,
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "# Parsed by MinerU" {
		t.Fatalf("unexpected MinerU output: %q", data)
	}
}

func TestRunFallsBackWithoutMinerUURLForPDF(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.pdf")
	if err := os.WriteFile(input, minimalTextPDF("CLI fallback PDF body"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"-i", input}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), "MinerU is not configured") {
		t.Fatalf("expected fallback warning, stderr=%q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "CLI fallback PDF body") {
		t.Fatalf("fallback output did not include PDF text: %q", stdout.String())
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
