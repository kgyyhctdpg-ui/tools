package filex

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatSizeExtAndKinds(t *testing.T) {
	if got := FormatSize(1536); got != "1.5 KB" {
		t.Fatalf("FormatSize = %q, want %q", got, "1.5 KB")
	}
	if got := Ext("https://example.com/a/Report.PDF?token=1"); got != "pdf" {
		t.Fatalf("Ext = %q, want pdf", got)
	}
	if !IsPDF("report.pdf") || !IsOffice("table.xlsx") || !IsImage("photo.webp") {
		t.Fatal("file kind helpers returned false for known extensions")
	}
	if MIMEByExt("photo.jpg") == "application/octet-stream" {
		t.Fatal("MIMEByExt should detect jpg")
	}
}

func TestPreparePathLocal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	got, temporary, err := PreparePath(context.Background(), path)
	if err != nil {
		t.Fatalf("PreparePath failed: %v", err)
	}
	if got != path || temporary {
		t.Fatalf("PreparePath = %q/%v, want %q/false", got, temporary, path)
	}
}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="report.txt"`)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()

	path, err := Download(context.Background(), server.URL+"/file", WithDownloadDir(t.TempDir()))
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "downloaded" {
		t.Fatalf("downloaded data = %q, want downloaded", string(data))
	}
	if !strings.HasSuffix(path, ".txt") {
		t.Fatalf("download path = %q, want .txt suffix", path)
	}
}

func TestDownloadRejectsStatusAndMaxBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Error(w, "bad", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("too large"))
	}))
	defer server.Close()

	if _, err := Download(context.Background(), server.URL+"/bad"); err == nil {
		t.Fatal("Download should reject non-2xx status")
	}
	if _, err := Download(context.Background(), server.URL+"/large", WithMaxBytes(3)); err == nil {
		t.Fatal("Download should reject oversized body")
	}
}

func TestFileHashMIMEAndBase64(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.txt")
	content := []byte("hello")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	sum := md5.Sum(content)
	wantMD5 := hex.EncodeToString(sum[:])
	gotMD5, err := FileMD5(path)
	if err != nil {
		t.Fatalf("FileMD5 failed: %v", err)
	}
	if gotMD5 != wantMD5 {
		t.Fatalf("FileMD5 = %q, want %q", gotMD5, wantMD5)
	}

	mimeType, err := DetectMIME(path)
	if err != nil {
		t.Fatalf("DetectMIME failed: %v", err)
	}
	if mimeType != "text/plain; charset=utf-8" {
		t.Fatalf("DetectMIME = %q, want text/plain", mimeType)
	}

	encoded, err := Base64EncodeFile(path)
	if err != nil {
		t.Fatalf("Base64EncodeFile failed: %v", err)
	}
	if encoded != "aGVsbG8=" {
		t.Fatalf("Base64EncodeFile = %q, want aGVsbG8=", encoded)
	}
}
