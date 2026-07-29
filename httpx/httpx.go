// Package httpx provides a small HTTP client for JSON APIs and file transfer.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

const defaultContentType = "application/json; charset=utf-8"

// Client wraps http.Client with common request helpers.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	headers     http.Header
	retry       int
	retryWait   time.Duration
	maxBodySize int64
}

// Option configures a Client.
type Option func(*Client)

// New creates a Client.
func New(opts ...Option) *Client {
	client := &Client{
		httpClient: http.DefaultClient,
		headers:    http.Header{},
		retryWait:  100 * time.Millisecond,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

// WithHTTPClient sets the underlying http.Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// WithTimeout sets the timeout on the underlying client.
func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		base := client.httpClient
		if base == nil {
			base = http.DefaultClient
		}
		copyClient := *base
		copyClient.Timeout = timeout
		client.httpClient = &copyClient
	}
}

// WithBaseURL sets a base URL for relative request paths.
func WithBaseURL(baseURL string) Option {
	return func(client *Client) {
		client.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHeader adds a default header.
func WithHeader(key, value string) Option {
	return func(client *Client) {
		client.headers.Add(key, value)
	}
}

// WithHeaders adds default headers.
func WithHeaders(headers http.Header) Option {
	return func(client *Client) {
		for key, values := range headers {
			for _, value := range values {
				client.headers.Add(key, value)
			}
		}
	}
}

// WithRetry configures retries for network errors and 5xx responses.
func WithRetry(times int, wait time.Duration) Option {
	return func(client *Client) {
		if times > 0 {
			client.retry = times
		}
		if wait > 0 {
			client.retryWait = wait
		}
	}
}

// WithMaxBodySize limits response bodies read by Do. maxBodySize <= 0 disables
// the limit.
func WithMaxBodySize(maxBodySize int64) Option {
	return func(client *Client) {
		client.maxBodySize = maxBodySize
	}
}

// Response contains an HTTP response body and metadata.
type Response struct {
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
}

// OK reports whether the response status code is 2xx.
func (resp *Response) OK() bool {
	return resp != nil && resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
}

// StatusError reports a non-2xx response.
type StatusError struct {
	Response *Response
}

func (err *StatusError) Error() string {
	if err == nil || err.Response == nil {
		return "httpx: status error"
	}
	body := strings.TrimSpace(string(err.Response.Body))
	if body == "" {
		return fmt.Sprintf("httpx: status %s", err.Response.Status)
	}
	return fmt.Sprintf("httpx: status %s: %s", err.Response.Status, body)
}

type requestConfig struct {
	query       any
	headers     http.Header
	body        []byte
	contentType string
}

// RequestOption configures a single request.
type RequestOption func(*requestConfig) error

// WithQuery adds query parameters. It accepts url.Values, map[string]string,
// map[string]any, slices, arrays, and JSON-marshalable structs.
func WithQuery(params any) RequestOption {
	return func(cfg *requestConfig) error {
		cfg.query = params
		return nil
	}
}

// WithRequestHeader adds a header to a single request.
func WithRequestHeader(key, value string) RequestOption {
	return func(cfg *requestConfig) error {
		cfg.headers.Add(key, value)
		return nil
	}
}

// WithRequestHeaders adds headers to a single request.
func WithRequestHeaders(headers http.Header) RequestOption {
	return func(cfg *requestConfig) error {
		for key, values := range headers {
			for _, value := range values {
				cfg.headers.Add(key, value)
			}
		}
		return nil
	}
}

// WithJSONBody marshals body as JSON.
func WithJSONBody(body any) RequestOption {
	return func(cfg *requestConfig) error {
		if body == nil {
			cfg.body = nil
			return nil
		}
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		cfg.body = data
		cfg.contentType = defaultContentType
		return nil
	}
}

// WithRawBody sets a raw request body and content type.
func WithRawBody(body []byte, contentType string) RequestOption {
	return func(cfg *requestConfig) error {
		cfg.body = append([]byte(nil), body...)
		cfg.contentType = contentType
		return nil
	}
}

// Do sends a request and returns its response. Non-2xx responses return
// *StatusError with the response attached.
func (client *Client) Do(ctx context.Context, method, rawURL string, opts ...RequestOption) (*Response, error) {
	cfg := requestConfig{headers: http.Header{}}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	reqURL, err := client.requestURL(rawURL, cfg.query)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var lastErr error
	for attempt := 0; attempt <= client.retry; attempt++ {
		resp, err := client.doOnce(ctx, method, reqURL, cfg)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !shouldRetry(err) || attempt == client.retry {
			return nil, err
		}
		if err := wait(ctx, client.retryWait); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

// Get sends a GET request.
func (client *Client) Get(ctx context.Context, rawURL string, query any, opts ...RequestOption) (*Response, error) {
	return client.Do(ctx, http.MethodGet, rawURL, append([]RequestOption{WithQuery(query)}, opts...)...)
}

// PostJSON sends a JSON POST request.
func (client *Client) PostJSON(ctx context.Context, rawURL string, body any, opts ...RequestOption) (*Response, error) {
	return client.Do(ctx, http.MethodPost, rawURL, append([]RequestOption{WithJSONBody(body)}, opts...)...)
}

// DecodeJSON decodes a response body into out.
func DecodeJSON(resp *Response, out any) error {
	if resp == nil {
		return fmt.Errorf("httpx: response is nil")
	}
	return json.Unmarshal(resp.Body, out)
}

// JSON sends a request and decodes the response body into out.
func (client *Client) JSON(ctx context.Context, method, rawURL string, out any, opts ...RequestOption) error {
	resp, err := client.Do(ctx, method, rawURL, opts...)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return DecodeJSON(resp, out)
}

// UploadFile sends a multipart/form-data POST request.
func (client *Client) UploadFile(ctx context.Context, rawURL, fieldName, filePath string, fields map[string]string, opts ...RequestOption) (*Response, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	opts = append([]RequestOption{WithRawBody(buf.Bytes(), writer.FormDataContentType())}, opts...)
	return client.Do(ctx, http.MethodPost, rawURL, opts...)
}

// DownloadFile downloads rawURL into dstPath.
func (client *Client) DownloadFile(ctx context.Context, rawURL, dstPath string, opts ...RequestOption) error {
	resp, err := client.Do(ctx, http.MethodGet, rawURL, opts...)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dstPath, resp.Body, 0o600)
}

func (client *Client) doOnce(ctx context.Context, method, reqURL string, cfg requestConfig) (*Response, error) {
	var body io.Reader
	if cfg.body != nil {
		body = bytes.NewReader(cfg.body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	for key, values := range client.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	for key, values := range cfg.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if cfg.contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", cfg.contentType)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	reader := resp.Body
	if client.maxBodySize > 0 {
		reader = io.NopCloser(io.LimitReader(resp.Body, client.maxBodySize+1))
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if client.maxBodySize > 0 && int64(len(data)) > client.maxBodySize {
		return nil, fmt.Errorf("httpx: response body exceeds %d bytes", client.maxBodySize)
	}

	result := &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Header:     resp.Header.Clone(),
		Body:       data,
	}
	if !result.OK() {
		return nil, &StatusError{Response: result}
	}
	return result, nil
}

func (client *Client) requestURL(rawURL string, params any) (string, error) {
	resolved := rawURL
	if client.baseURL != "" && strings.HasPrefix(rawURL, "/") {
		resolved = client.baseURL + rawURL
	}

	parsed, err := url.Parse(resolved)
	if err != nil {
		return "", err
	}
	values, err := queryValues(params)
	if err != nil {
		return "", err
	}
	if len(values) > 0 {
		query := parsed.Query()
		for key, items := range values {
			for _, item := range items {
				query.Add(key, item)
			}
		}
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

func shouldRetry(err error) bool {
	if statusErr, ok := err.(*StatusError); ok {
		return statusErr.Response != nil && statusErr.Response.StatusCode >= http.StatusInternalServerError
	}
	return true
}

func wait(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func queryValues(params any) (url.Values, error) {
	values := url.Values{}
	if params == nil {
		return values, nil
	}

	switch v := params.(type) {
	case url.Values:
		return cloneValues(v), nil
	case map[string][]string:
		return cloneValues(url.Values(v)), nil
	case map[string]string:
		for key, value := range v {
			values.Add(key, value)
		}
		return values, nil
	case map[string]any:
		addMapValues(values, v)
		return values, nil
	default:
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, err
		}
		addMapValues(values, decoded)
		return values, nil
	}
}

func cloneValues(source url.Values) url.Values {
	values := url.Values{}
	for key, items := range source {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return values
}

func addMapValues(values url.Values, params map[string]any) {
	for key, value := range params {
		addValue(values, key, value)
	}
}

func addValue(values url.Values, key string, value any) {
	if key == "" || value == nil {
		return
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			addValue(values, key, rv.Index(i).Interface())
		}
		return
	}
	values.Add(key, fmt.Sprint(value))
}
