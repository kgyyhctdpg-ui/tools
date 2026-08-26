package markdown

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	defaultMinerUEndpoint        = "/tasks"
	defaultMinerUBackend         = "pipeline"
	defaultMinerUParseMethod     = "auto"
	defaultMinerULanguage        = "ch"
	defaultMinerUTimeout         = 20 * time.Minute
	defaultMinerUPollInterval    = 2 * time.Second
	defaultMinerUMaxResponseSize = int64(256 << 20)
)

const (
	minerUStatusCompleted = "completed"
	minerUStatusFailed    = "failed"
)

// MinerUConfig configures a self-hosted MinerU async task API.
type MinerUConfig struct {
	BaseURL         string
	Endpoint        string
	Token           string
	Backend         string
	ParseMethod     string
	Language        string
	DisableFormula  bool
	DisableTable    bool
	Timeout         time.Duration
	PollInterval    time.Duration
	MaxResponseSize int64
	HTTPClient      *http.Client
}

type minerUClient struct {
	endpoint     string
	token        string
	backend      string
	parseMethod  string
	language     string
	formula      bool
	table        bool
	timeout      time.Duration
	pollInterval time.Duration
	maxBodySize  int64
	http         *http.Client
}

type minerUTask struct {
	TaskID  string          `json:"task_id"`
	Status  string          `json:"status"`
	Error   string          `json:"error"`
	ErrMsg  string          `json:"err_msg"`
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail"`
}

type minerUResponse struct {
	Backend   string                    `json:"backend"`
	Version   string                    `json:"version"`
	Results   map[string]minerUDocument `json:"results"`
	MDContent string                    `json:"md_content"`
	Images    map[string]string         `json:"images"`
	Error     string                    `json:"error"`
	ErrMsg    string                    `json:"err_msg"`
	Message   string                    `json:"message"`
	Detail    json.RawMessage           `json:"detail"`
}

type minerUDocument struct {
	MDContent string            `json:"md_content"`
	Images    map[string]string `json:"images"`
	Error     string            `json:"error"`
	ErrMsg    string            `json:"err_msg"`
}

// NewMinerUPDFHandler creates a PDF handler for a self-hosted MinerU API.
func NewMinerUPDFHandler(config MinerUConfig) (PDFHandler, error) {
	client, err := newMinerUClient(config)
	if err != nil {
		return nil, err
	}
	return client.convert, nil
}

func newMinerUClient(config MinerUConfig) (*minerUClient, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, errors.New("markdown: MinerU base URL is empty")
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, fmt.Errorf("markdown: invalid MinerU base URL %q", baseURL)
	}
	if parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https" {
		return nil, fmt.Errorf("markdown: unsupported MinerU URL scheme %q", parsedBaseURL.Scheme)
	}

	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		endpoint = defaultMinerUEndpoint
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("markdown: invalid MinerU endpoint %q: %w", endpoint, err)
	}
	if !parsedEndpoint.IsAbs() {
		if !strings.HasPrefix(endpoint, "/") {
			endpoint = "/" + endpoint
			parsedEndpoint, _ = url.Parse(endpoint)
		}
		parsedEndpoint = parsedBaseURL.ResolveReference(parsedEndpoint)
	}

	backend := strings.TrimSpace(config.Backend)
	if backend == "" {
		backend = defaultMinerUBackend
	}
	parseMethod := strings.TrimSpace(config.ParseMethod)
	if parseMethod == "" {
		parseMethod = defaultMinerUParseMethod
	}
	language := strings.TrimSpace(config.Language)
	if language == "" {
		language = defaultMinerULanguage
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultMinerUTimeout
	}
	pollInterval := config.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultMinerUPollInterval
	}
	maxResponseSize := config.MaxResponseSize
	if maxResponseSize <= 0 {
		maxResponseSize = defaultMinerUMaxResponseSize
	}

	return &minerUClient{
		endpoint:     parsedEndpoint.String(),
		token:        strings.TrimSpace(config.Token),
		backend:      backend,
		parseMethod:  parseMethod,
		language:     language,
		formula:      !config.DisableFormula,
		table:        !config.DisableTable,
		timeout:      timeout,
		pollInterval: pollInterval,
		maxBodySize:  maxResponseSize,
		http:         httpClientOrDefault(config.HTTPClient),
	}, nil
}

func (client *minerUClient) convert(ctx context.Context, data []byte, info StreamInfo, imageHandler ImageHandler) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	name := filepath.Base(strings.TrimSpace(info.Name))
	if name == "" || name == "." {
		name = "document.pdf"
	}
	if !strings.EqualFold(filepath.Ext(name), ".pdf") {
		name += ".pdf"
	}
	fields := map[string]string{
		"backend":             client.backend,
		"parse_method":        client.parseMethod,
		"lang_list":           client.language,
		"formula_enable":      strconv.FormatBool(client.formula),
		"table_enable":        strconv.FormatBool(client.table),
		"return_md":           "true",
		"return_middle_json":  "false",
		"return_model_output": "false",
		"return_content_list": "false",
		"return_images":       "true",
		"response_format_zip": "false",
	}
	task, err := client.submit(ctx, name, data, fields)
	if err != nil {
		return nil, fmt.Errorf("markdown: MinerU request failed: %w", err)
	}
	if err := client.waitForTask(ctx, task.TaskID); err != nil {
		return nil, err
	}
	response, err := client.fetchResult(ctx, task.TaskID)
	if err != nil {
		return nil, fmt.Errorf("markdown: MinerU task %s result failed: %w", task.TaskID, err)
	}
	var payload minerUResponse
	if err := json.Unmarshal(response, &payload); err != nil {
		return nil, fmt.Errorf("markdown: decode MinerU response: %w", err)
	}
	document, err := selectMinerUDocument(payload, name)
	if err != nil {
		return nil, err
	}
	markdown, err := prepareMinerUMarkdown(ctx, document.MDContent, document.Images, imageHandler)
	if err != nil {
		return nil, err
	}
	metadata := map[string]string{
		"pdf_content_type": "mineru",
		"pdf_renderer":     "mineru",
		"mineru_backend":   firstNonEmpty(payload.Backend, client.backend),
	}
	if payload.Version != "" {
		metadata["mineru_version"] = payload.Version
	}
	return &Result{
		Title:    strings.TrimSuffix(name, filepath.Ext(name)),
		Markdown: markdown,
		Metadata: metadata,
	}, nil
}

func (client *minerUClient) submit(ctx context.Context, name string, data []byte, fields map[string]string) (*minerUTask, error) {
	body, contentType, err := client.multipartBody(name, data, fields)
	if err != nil {
		return nil, err
	}
	response, err := client.do(ctx, http.MethodPost, client.endpoint, body, contentType)
	if err != nil {
		return nil, err
	}
	var task minerUTask
	if err := json.Unmarshal(response, &task); err != nil {
		return nil, fmt.Errorf("decode MinerU task response: %w", err)
	}
	if task.TaskID == "" {
		return nil, fmt.Errorf("MinerU task response has no task_id%s", minerUDetail(task.Message, task.Detail))
	}
	if task.Status == minerUStatusFailed {
		return nil, fmt.Errorf("MinerU task failed: %s", firstNonEmpty(task.Error, task.ErrMsg, task.Message))
	}
	return &task, nil
}

func (client *minerUClient) multipartBody(name string, data []byte, fields map[string]string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", err
		}
	}
	part, err := writer.CreateFormFile("files", name)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(data); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &buf, writer.FormDataContentType(), nil
}

func (client *minerUClient) do(ctx context.Context, method, target string, body io.Reader, contentType string) ([]byte, error) {
	response, err := client.request(ctx, method, target, body, contentType)
	if err != nil {
		return nil, err
	}
	if response.status < http.StatusOK || response.status >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("status %s: %s", response.statusText, strings.TrimSpace(string(response.body)))
	}
	return response.body, nil
}

type minerUHTTPResponse struct {
	status     int
	statusText string
	body       []byte
}

func (client *minerUClient) request(ctx context.Context, method, target string, body io.Reader, contentType string) (*minerUHTTPResponse, error) {
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Accept", "application/json")
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}

	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	reader := io.Reader(response.Body)
	if client.maxBodySize > 0 {
		reader = io.LimitReader(response.Body, client.maxBodySize+1)
	}
	payload, err := io.ReadAll(reader)
	closeErr := response.Body.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if client.maxBodySize > 0 && int64(len(payload)) > client.maxBodySize {
		return nil, fmt.Errorf("response body exceeds %d bytes", client.maxBodySize)
	}
	return &minerUHTTPResponse{
		status:     response.StatusCode,
		statusText: response.Status,
		body:       payload,
	}, nil
}

func (client *minerUClient) waitForTask(ctx context.Context, taskID string) error {
	for {
		task, err := client.taskStatus(ctx, taskID)
		if err != nil {
			return fmt.Errorf("markdown: query MinerU task %s: %w", taskID, err)
		}
		switch task.Status {
		case minerUStatusCompleted:
			return nil
		case minerUStatusFailed:
			return fmt.Errorf("markdown: MinerU task %s failed: %s", taskID, firstNonEmpty(task.Error, task.ErrMsg, task.Message))
		}
		if err := waitMinerUInterval(ctx, client.pollInterval); err != nil {
			return fmt.Errorf("markdown: waiting for MinerU task %s: %w", taskID, err)
		}
	}
}

func (client *minerUClient) taskStatus(ctx context.Context, taskID string) (*minerUTask, error) {
	target, err := client.taskURL(taskID, "")
	if err != nil {
		return nil, err
	}
	body, err := client.do(ctx, http.MethodGet, target, nil, "")
	if err != nil {
		return nil, err
	}
	var task minerUTask
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("decode MinerU task status: %w", err)
	}
	if task.TaskID != "" && task.TaskID != taskID {
		return nil, fmt.Errorf("unexpected MinerU task id %q for %q", task.TaskID, taskID)
	}
	return &task, nil
}

func (client *minerUClient) fetchResult(ctx context.Context, taskID string) ([]byte, error) {
	target, err := client.taskURL(taskID, "result")
	if err != nil {
		return nil, err
	}
	for {
		response, err := client.request(ctx, http.MethodGet, target, nil, "")
		if err != nil {
			return nil, err
		}
		if response.status >= http.StatusOK && response.status < http.StatusMultipleChoices {
			return response.body, nil
		}
		if response.status == http.StatusAccepted || response.status == http.StatusConflict {
			var task minerUTask
			if json.Unmarshal(response.body, &task) == nil {
				if task.Status == minerUStatusFailed || response.status == http.StatusConflict {
					return nil, fmt.Errorf("MinerU task %s failed: %s", taskID, firstNonEmpty(task.Error, task.ErrMsg, task.Message))
				}
			}
			if err := waitMinerUInterval(ctx, client.pollInterval); err != nil {
				return nil, fmt.Errorf("waiting for MinerU task %s result: %w", taskID, err)
			}
			continue
		}
		return nil, fmt.Errorf("status %s: %s", response.statusText, strings.TrimSpace(string(response.body)))
	}
}

func (client *minerUClient) taskURL(taskID, suffix string) (string, error) {
	parsed, err := url.Parse(client.endpoint)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/" + url.PathEscape(taskID)
	if suffix != "" {
		parsed.Path += "/" + suffix
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func waitMinerUInterval(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func httpClientOrDefault(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return http.DefaultClient
}

func selectMinerUDocument(payload minerUResponse, name string) (minerUDocument, error) {
	if payload.Error != "" || payload.ErrMsg != "" {
		return minerUDocument{}, fmt.Errorf("markdown: MinerU failed: %s", firstNonEmpty(payload.Error, payload.ErrMsg))
	}
	if len(payload.Results) == 0 {
		if payload.MDContent != "" || payload.Images != nil {
			return minerUDocument{MDContent: payload.MDContent, Images: payload.Images}, nil
		}
		return minerUDocument{}, fmt.Errorf("markdown: MinerU response has no result%s", minerUResponseDetail(payload))
	}

	baseName := filepath.Base(name)
	candidates := []string{baseName, strings.TrimSuffix(baseName, filepath.Ext(baseName))}
	for _, candidate := range candidates {
		if document, ok := payload.Results[candidate]; ok {
			return validateMinerUDocument(document)
		}
	}
	if len(payload.Results) == 1 {
		for _, document := range payload.Results {
			return validateMinerUDocument(document)
		}
	}
	return minerUDocument{}, fmt.Errorf("markdown: MinerU returned %d results for one PDF", len(payload.Results))
}

func validateMinerUDocument(document minerUDocument) (minerUDocument, error) {
	if document.Error != "" || document.ErrMsg != "" {
		return minerUDocument{}, fmt.Errorf("markdown: MinerU failed: %s", firstNonEmpty(document.Error, document.ErrMsg))
	}
	return document, nil
}

func minerUResponseDetail(payload minerUResponse) string {
	return minerUDetail(payload.Message, payload.Detail)
}

func minerUDetail(message string, detail json.RawMessage) string {
	text := strings.TrimSpace(message)
	if len(detail) > 0 && string(detail) != "null" {
		var value any
		if json.Unmarshal(detail, &value) == nil {
			if encoded, err := json.Marshal(value); err == nil {
				text = string(encoded)
			}
		}
	}
	if text == "" {
		return ""
	}
	return ": " + text
}

func prepareMinerUMarkdown(ctx context.Context, source string, images map[string]string, imageHandler ImageHandler) (string, error) {
	links, err := externalizeMinerUImages(ctx, images, imageHandler)
	if err != nil {
		return "", err
	}
	return rewriteMinerUImageLinks(source, links), nil
}

func externalizeMinerUImages(ctx context.Context, images map[string]string, imageHandler ImageHandler) (map[string]string, error) {
	if imageHandler == nil {
		imageHandler = DataURIImageHandler
	}
	names := make([]string, 0, len(images))
	for name := range images {
		names = append(names, name)
	}
	sort.Strings(names)
	links := make(map[string]string, len(images)*2)
	aliases := make(map[string]string)
	contentLinks := make(map[[sha256.Size]byte]string)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		mimeType, data, err := decodeMinerUImage(name, images[name])
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(data)
		imageURL, exists := contentLinks[digest]
		if !exists {
			fileName := path.Base(normalizeMinerUImagePath(name))
			if fileName == "." || fileName == "" {
				fileName = "image" + imageExtension(mimeType)
			}
			imageURL, err = imageHandler(ctx, Image{
				Name:     fileName,
				MIMEType: mimeType,
				AltText:  strings.TrimSuffix(fileName, path.Ext(fileName)),
				Data:     data,
			})
			if err != nil {
				return nil, fmt.Errorf("markdown: store MinerU image %q: %w", name, err)
			}
			contentLinks[digest] = imageURL
		}
		normalized := normalizeMinerUImagePath(name)
		links[normalized] = imageURL
		base := path.Base(normalized)
		for _, candidate := range []string{base, path.Join("images", base)} {
			if previous, ok := aliases[candidate]; !ok {
				aliases[candidate] = imageURL
			} else if previous != imageURL {
				aliases[candidate] = ""
			}
		}
	}
	for name, imageURL := range aliases {
		if imageURL != "" {
			links[name] = imageURL
		}
	}
	return links, nil
}

func decodeMinerUImage(name, encoded string) (string, []byte, error) {
	mimeType := ""
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(strings.ToLower(encoded), "data:") {
		parts := strings.SplitN(encoded, ",", 2)
		if len(parts) != 2 || !strings.HasSuffix(strings.ToLower(parts[0]), ";base64") {
			return "", nil, fmt.Errorf("markdown: MinerU image %q has an invalid data URI", name)
		}
		mimeType = strings.TrimSuffix(strings.TrimPrefix(parts[0], "data:"), ";base64")
		encoded = parts[1]
	}
	encoded = strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return -1
		}
		return character
	}, encoded)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return "", nil, fmt.Errorf("markdown: decode MinerU image %q: %w", name, err)
	}
	if mimeType == "" {
		mimeType = imageMIMEType(name, data)
	}
	return mimeType, data, nil
}

func normalizeMinerUImagePath(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "<>")
	if parsed, err := url.Parse(value); err == nil {
		value = parsed.Path
	}
	if unescaped, err := url.PathUnescape(value); err == nil {
		value = unescaped
	}
	value = strings.ReplaceAll(value, "\\", "/")
	value = path.Clean(value)
	value = strings.TrimPrefix(value, "/")
	return strings.TrimPrefix(value, "./")
}

func rewriteMinerUImageLinks(source string, links map[string]string) string {
	keys := make([]string, 0, len(links))
	for key := range links {
		if key != "" && key != "." {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(first, second int) bool {
		return len(keys[first]) > len(keys[second])
	})

	replacements := make([]string, 0, len(keys)*16)
	seen := make(map[string]bool)
	for _, key := range keys {
		imageURL := links[key]
		variants := []string{key, "./" + key, (&url.URL{Path: key}).EscapedPath()}
		for _, variant := range variants {
			if variant == "" || seen[variant] {
				continue
			}
			seen[variant] = true
			replacements = append(replacements,
				"]("+variant+")", "]("+imageURL+")",
				"](<"+variant+">)", "](<"+imageURL+">)",
				"]("+variant+" ", "]("+imageURL+" ",
				`src="`+variant+`"`, `src="`+imageURL+`"`,
				"src='"+variant+"'", "src='"+imageURL+"'",
			)
		}
	}
	if len(replacements) == 0 {
		return source
	}
	return strings.NewReplacer(replacements...).Replace(source)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
