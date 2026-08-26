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
	statusCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/tasks":
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
				"backend":             "pipeline",
				"parse_method":        "auto",
				"lang_list":           "ch",
				"formula_enable":      "true",
				"table_enable":        "true",
				"return_md":           "true",
				"return_images":       "true",
				"response_format_zip": "false",
			} {
				if actual := request.FormValue(field); actual != expected {
					t.Fatalf("unexpected %s: got %q want %q", field, actual, expected)
				}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"task_id": "task-123",
				"status":  "pending",
			})
		case request.Method == http.MethodGet && request.URL.Path == "/tasks/task-123":
			statusCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"task_id": "task-123",
				"status":  "completed",
			})
		case request.Method == http.MethodGet && request.URL.Path == "/tasks/task-123/result":
			encodedImage := base64.StdEncoding.EncodeToString(imageData)
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"backend": "pipeline",
				"version": "2.6.0",
				"results": map[string]any{
					"report": map[string]any{
						"md_content": minerUMarkdown,
						"images": map[string]string{
							"diagram.png":      encodedImage,
							"diagram-copy.png": encodedImage,
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()

	handler, err := NewMinerUPDFHandler(MinerUConfig{
		BaseURL:      server.URL,
		Token:        "secret",
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
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
	if statusCalls != 1 {
		t.Fatalf("task status should be polled once after completion, calls=%d", statusCalls)
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

func TestMinerUPDFHandlerPollsUntilCompleted(t *testing.T) {
	statusCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/tasks":
			_ = json.NewEncoder(writer).Encode(map[string]any{"task_id": "task-1", "status": "pending"})
		case request.Method == http.MethodGet && request.URL.Path == "/tasks/task-1":
			statusCalls++
			status := "processing"
			if statusCalls >= 3 {
				status = "completed"
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"task_id": "task-1", "status": status})
		case request.Method == http.MethodGet && request.URL.Path == "/tasks/task-1/result":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"results": map[string]any{
					"report": map[string]any{"md_content": "# done"},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()

	handler, err := NewMinerUPDFHandler(MinerUConfig{
		BaseURL:      server.URL,
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := New(WithPDFHandler(handler)).ConvertReader(
		context.Background(),
		strings.NewReader("pdf"),
		StreamInfo{Name: "report.pdf"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Markdown != "# done" {
		t.Fatalf("unexpected MinerU output: %q", result.Markdown)
	}
	if statusCalls != 3 {
		t.Fatalf("expected three status polls, got %d", statusCalls)
	}
}

func TestMinerUPDFHandlerReportsTaskFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/tasks":
			_ = json.NewEncoder(writer).Encode(map[string]any{"task_id": "task-1", "status": "pending"})
		case request.Method == http.MethodGet && request.URL.Path == "/tasks/task-1":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"task_id": "task-1",
				"status":  "failed",
				"error":   "model unavailable",
			})
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()

	handler, err := NewMinerUPDFHandler(MinerUConfig{
		BaseURL:      server.URL,
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(WithPDFHandler(handler)).ConvertReader(
		context.Background(),
		strings.NewReader("pdf"),
		StreamInfo{Name: "report.pdf"},
	)
	if err == nil || !strings.Contains(err.Error(), "failed") || !strings.Contains(err.Error(), "model unavailable") {
		t.Fatalf("unexpected MinerU task failure: %v", err)
	}
}
