package markdown

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMinerUPDFHandlerConvertsEveryPDF(t *testing.T) {
	imageData := []byte("mineru image")
	minerUMarkdown := strings.Join([]string{
		"# Parsed",
		"![diagram](images/diagram.png)",
		`<img src="images/diagram-copy.png" alt="copy">`,
		"| Name | Value |\n| --- | --- |\n| **Total** | $x^2$ |",
		"<table><tr><td><b>Raw</b></td></tr></table>",
		"$$\ny=x+1\n$$",
	}, "\n\n")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/file_parse" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		file, header, err := request.FormFile("files")
		if err != nil {
			t.Fatalf("read uploaded PDF: %v", err)
		}
		defer file.Close()
		uploaded, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read uploaded data: %v", err)
		}
		if header.Filename != "report.pdf" || !bytes.Equal(uploaded, []byte("not a valid local PDF")) {
			t.Fatalf("unexpected upload: name=%q data=%q", header.Filename, uploaded)
		}
		for field, expected := range map[string]string{
			"backend":        "pipeline",
			"parse_method":   "auto",
			"lang_list":      "ch",
			"formula_enable": "true",
			"table_enable":   "true",
			"return_md":      "true",
			"return_images":  "true",
		} {
			if actual := request.FormValue(field); actual != expected {
				t.Fatalf("unexpected %s: got %q want %q", field, actual, expected)
			}
		}

		encodedImage := base64.StdEncoding.EncodeToString(imageData)
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"backend": "pipeline",
			"version": "2.6.0",
			"results": map[string]any{
				"report": map[string]any{
					"md_content": minerUMarkdown,
					"images": map[string]string{
						"images/diagram.png":      encodedImage,
						"images/diagram-copy.png": encodedImage,
					},
				},
			},
		})
	}))
	defer server.Close()

	handler, err := NewMinerUPDFHandler(MinerUConfig{
		BaseURL: server.URL,
		Token:   "secret",
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	imageCalls := 0
	converter := New(
		WithPDFHandler(handler),
		WithImageHandler(func(_ context.Context, image Image) (string, error) {
			imageCalls++
			if !bytes.Equal(image.Data, imageData) {
				t.Fatalf("unexpected image data: %q", image.Data)
			}
			return "assets/shared.png", nil
		}),
	)
	result, err := converter.ConvertReader(
		context.Background(),
		strings.NewReader("not a valid local PDF"),
		StreamInfo{Name: "report.pdf", MIMEType: "application/pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if imageCalls != 1 {
		t.Fatalf("identical MinerU images should be externalized once, calls=%d", imageCalls)
	}
	expected := strings.ReplaceAll(minerUMarkdown, "images/diagram.png", "assets/shared.png")
	expected = strings.ReplaceAll(expected, "images/diagram-copy.png", "assets/shared.png")
	if result.Markdown != expected {
		t.Fatalf("MinerU Markdown should remain unchanged except image links:\nwant:\n%s\n\ngot:\n%s", expected, result.Markdown)
	}
	if result.Metadata["pdf_renderer"] != "mineru" || result.Metadata["mineru_version"] != "2.6.0" {
		t.Fatalf("unexpected MinerU metadata: %#v", result.Metadata)
	}
}

func TestNewMinerUPDFHandlerValidatesURL(t *testing.T) {
	for _, baseURL := range []string{"", "localhost:8000", "ftp://localhost/mineru"} {
		if _, err := NewMinerUPDFHandler(MinerUConfig{BaseURL: baseURL}); err == nil {
			t.Fatalf("expected invalid MinerU URL error for %q", baseURL)
		}
	}
}

func TestMinerUPDFHandlerReportsServiceErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "model unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	handler, err := NewMinerUPDFHandler(MinerUConfig{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(WithPDFHandler(handler)).ConvertReader(
		context.Background(),
		strings.NewReader("pdf"),
		StreamInfo{Name: "report.pdf"},
	)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("unexpected MinerU service error: %v", err)
	}
}
