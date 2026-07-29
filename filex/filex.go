// Package filex provides small file helpers shared by business tools.
package filex

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// IsFile reports whether path exists and is a regular file.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// IsDir reports whether path exists and is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FormatSize formats a byte count using binary units.
func FormatSize(size int64) string {
	sign := ""
	if size < 0 {
		sign = "-"
		size = -size
	}
	if size < 1024 {
		return fmt.Sprintf("%s%d B", sign, size)
	}

	value := float64(size)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return fmt.Sprintf("%s%.1f %s", sign, value, unit)
		}
	}
	return fmt.Sprintf("%s%.1f EB", sign, value/1024)
}

// Ext returns the lower-case extension without a leading dot. It also handles
// HTTP(S) URLs with query strings.
func Ext(name string) string {
	path := name
	if parsed, err := neturl.Parse(name); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

// MIMEByExt returns a MIME type from a file name or URL extension.
func MIMEByExt(name string) string {
	ext := Ext(name)
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension("." + ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// DetectMIME reads the first bytes of a file and detects its content type.
func DetectMIME(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

// FileMD5 returns the hex-encoded MD5 digest of a file.
func FileMD5(path string) (string, error) {
	return hashFile(path, md5.New())
}

// FileSHA256 returns the hex-encoded SHA-256 digest of a file.
func FileSHA256(path string) (string, error) {
	return hashFile(path, sha256.New())
}

// Base64EncodeFile returns the standard base64 encoding of a file.
func Base64EncodeFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// IsImage reports whether a name has a common image extension.
func IsImage(name string) bool {
	switch Ext(name) {
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "tif", "tiff", "svg", "ico":
		return true
	default:
		return false
	}
}

// IsPDF reports whether a name has a PDF extension.
func IsPDF(name string) bool {
	return Ext(name) == "pdf"
}

// IsOffice reports whether a name has a common Office document extension.
func IsOffice(name string) bool {
	switch Ext(name) {
	case "doc", "docx", "xls", "xlsx", "ppt", "pptx", "wps", "et", "dps":
		return true
	default:
		return false
	}
}

// IsRemoteURL reports whether rawURL is an HTTP or HTTPS URL.
func IsRemoteURL(rawURL string) bool {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// PreparePath returns a usable local file path. Local files are returned as-is;
// HTTP(S) URLs are downloaded into a temporary file.
func PreparePath(ctx context.Context, pathOrURL string, opts ...DownloadOption) (path string, temporary bool, err error) {
	if IsRemoteURL(pathOrURL) {
		path, err := Download(ctx, pathOrURL, opts...)
		return path, true, err
	}
	if !IsFile(pathOrURL) {
		return "", false, fmt.Errorf("filex: file does not exist: %s", pathOrURL)
	}
	return pathOrURL, false, nil
}

func hashFile(path string, h hash.Hash) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" || name == "." {
		return "download"
	}
	return name
}

type downloadConfig struct {
	client   *http.Client
	timeout  time.Duration
	maxBytes int64
	dir      string
	filename string
}

// DownloadOption configures Download.
type DownloadOption func(*downloadConfig)

// WithHTTPClient sets the HTTP client used by Download.
func WithHTTPClient(client *http.Client) DownloadOption {
	return func(cfg *downloadConfig) {
		if client != nil {
			cfg.client = client
		}
	}
}

// WithTimeout sets a request timeout. The default is 30 seconds.
func WithTimeout(timeout time.Duration) DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.timeout = timeout
	}
}

// WithMaxBytes rejects downloads larger than maxBytes. maxBytes <= 0 disables
// the limit.
func WithMaxBytes(maxBytes int64) DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.maxBytes = maxBytes
	}
}

// WithDownloadDir stores downloaded files in dir.
func WithDownloadDir(dir string) DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.dir = dir
	}
}

// WithFilename stores the download using filename. Path components are removed.
func WithFilename(filename string) DownloadOption {
	return func(cfg *downloadConfig) {
		cfg.filename = filename
	}
}

// Download downloads an HTTP(S) URL into a local file and returns its path.
func Download(ctx context.Context, rawURL string, opts ...DownloadOption) (string, error) {
	if !IsRemoteURL(rawURL) {
		return "", fmt.Errorf("filex: unsupported download url: %s", rawURL)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cfg := downloadConfig{
		client:  http.DefaultClient,
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := cfg.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("filex: download failed with status %s", resp.Status)
	}

	if cfg.dir != "" {
		if err := os.MkdirAll(cfg.dir, 0o755); err != nil {
			return "", err
		}
	}

	filename := cfg.filename
	if filename == "" {
		filename = responseFilename(resp)
	}
	if filename == "" {
		parsed, _ := neturl.Parse(rawURL)
		filename = filepath.Base(parsed.Path)
	}
	filename = safeFilename(filename)

	file, err := createDownloadFile(cfg.dir, filename, cfg.filename != "")
	if err != nil {
		return "", err
	}
	path := file.Name()

	reader := resp.Body
	if cfg.maxBytes > 0 {
		reader = io.NopCloser(io.LimitReader(resp.Body, cfg.maxBytes+1))
	}

	n, err := io.Copy(file, reader)
	if err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if cfg.maxBytes > 0 && n > cfg.maxBytes {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("filex: download exceeds %d bytes", cfg.maxBytes)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func responseFilename(resp *http.Response) string {
	disposition := resp.Header.Get("Content-Disposition")
	if disposition == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func createDownloadFile(dir, filename string, fixedName bool) (*os.File, error) {
	if fixedName {
		if dir == "" {
			dir = os.TempDir()
		}
		return os.OpenFile(filepath.Join(dir, safeFilename(filename)), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if base == "" {
		base = "download"
	}
	return os.CreateTemp(dir, base+"-*"+ext)
}
