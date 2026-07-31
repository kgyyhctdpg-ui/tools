package markdown

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/scoming-dev/tools/httpx"
)

const (
	defaultMinerUEndpoint        = "/file_parse"
	defaultMinerUBackend         = "pipeline"
	defaultMinerUParseMethod     = "auto"
	defaultMinerULanguage        = "ch"
	defaultMinerUTimeout         = 20 * time.Minute
	defaultMinerUMaxResponseSize = int64(256 << 20)
)

// MinerUConfig configures a self-hosted MinerU /file_parse service.
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
	MaxResponseSize int64
	HTTPClient      *http.Client
}

type minerUClient struct {
	endpoint    string
	token       string
	backend     string
	parseMethod string
	language    string
	formula     bool
	table       bool
	timeout     time.Duration
	http        *httpx.Client
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
	maxResponseSize := config.MaxResponseSize
	if maxResponseSize <= 0 {
		maxResponseSize = defaultMinerUMaxResponseSize
	}

	return &minerUClient{
		endpoint:    parsedEndpoint.String(),
		token:       strings.TrimSpace(config.Token),
		backend:     backend,
		parseMethod: parseMethod,
		language:    language,
		formula:     !config.DisableFormula,
		table:       !config.DisableTable,
		timeout:     timeout,
		http: httpx.New(
			httpx.WithHTTPClient(config.HTTPClient),
			httpx.WithMaxBodySize(maxResponseSize),
		),
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
	}
	requestOptions := make([]httpx.RequestOption, 0, 1)
	if client.token != "" {
		requestOptions = append(requestOptions, httpx.WithRequestHeader("Authorization", "Bearer "+client.token))
	}
	response, err := client.http.UploadBytes(ctx, client.endpoint, "files", name, data, fields, requestOptions...)
	if err != nil {
		return nil, fmt.Errorf("markdown: MinerU request failed: %w", err)
	}
	var payload minerUResponse
	if err := httpx.DecodeJSON(response, &payload); err != nil {
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
	message := strings.TrimSpace(payload.Message)
	if len(payload.Detail) > 0 && string(payload.Detail) != "null" {
		var detail any
		if json.Unmarshal(payload.Detail, &detail) == nil {
			if encoded, err := json.Marshal(detail); err == nil {
				message = string(encoded)
			}
		}
	}
	if message == "" {
		return ""
	}
	return ": " + message
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
		if previous, ok := aliases[base]; !ok {
			aliases[base] = imageURL
		} else if previous != imageURL {
			aliases[base] = ""
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
