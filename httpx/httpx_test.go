package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetAndPostJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/get" {
			if r.URL.Query().Get("name") != "gavin" || r.Header.Get("X-App") != "tools" {
				t.Fatalf("unexpected query/header: %s %s", r.URL.RawQuery, r.Header.Get("X-App"))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"name": body["name"]})
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL), WithHeader("X-App", "tools"))
	resp, err := client.Get(context.Background(), "/get", map[string]string{"name": "gavin"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	var getResp map[string]string
	if err := DecodeJSON(resp, &getResp); err != nil {
		t.Fatalf("DecodeJSON failed: %v", err)
	}
	if getResp["ok"] != "yes" {
		t.Fatalf("get response = %#v", getResp)
	}

	var postResp map[string]string
	if err := client.JSON(context.Background(), http.MethodPost, server.URL+"/post", &postResp, WithJSONBody(map[string]string{"name": "alice"})); err != nil {
		t.Fatalf("JSON failed: %v", err)
	}
	if postResp["name"] != "alice" {
		t.Fatalf("post response = %#v", postResp)
	}
}

func TestStatusErrorAndRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "try again", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := New(WithRetry(1, 0))
	resp, err := client.Get(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("Get failed after retry: %v", err)
	}
	if string(resp.Body) != "ok" || calls != 2 {
		t.Fatalf("body/calls = %q/%d, want ok/2", string(resp.Body), calls)
	}

	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer badServer.Close()

	_, err = New().Get(context.Background(), badServer.URL, nil)
	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.Response.StatusCode != http.StatusBadRequest {
		t.Fatalf("error = %v, want StatusError 400", err)
	}
}

func TestUploadAndDownload(t *testing.T) {
	uploaded := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upload" {
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile failed: %v", err)
			}
			defer file.Close()
			data, _ := io.ReadAll(file)
			uploaded = append(uploaded, string(data))
			_, _ = w.Write([]byte("uploaded"))
			return
		}
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()

	src := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	client := New()
	if _, err := client.UploadFile(context.Background(), server.URL+"/upload", "file", src, map[string]string{"kind": "text"}); err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if len(uploaded) != 1 || uploaded[0] != "hello" {
		t.Fatalf("uploaded = %q, want hello", uploaded)
	}
	if _, err := client.UploadBytes(context.Background(), server.URL+"/upload", "file", "memory.txt", []byte("memory"), map[string]string{"kind": "text"}); err != nil {
		t.Fatalf("UploadBytes failed: %v", err)
	}
	if len(uploaded) != 2 || uploaded[1] != "memory" {
		t.Fatalf("uploaded = %q, want hello and memory", uploaded)
	}

	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := client.DownloadFile(context.Background(), server.URL+"/download", dst); err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "downloaded" {
		t.Fatalf("downloaded = %q, want downloaded", string(data))
	}
}
