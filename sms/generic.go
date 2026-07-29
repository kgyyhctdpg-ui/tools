package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Encoding controls how GenericProvider encodes the request body.
type Encoding string

const (
	EncodingJSON Encoding = "json"
	EncodingForm Encoding = "form"
)

// GenericConfig configures a generic HTTP provider adapter.
type GenericConfig struct {
	Provider Provider
	Endpoint string
	Method   string
	Encoding Encoding
	Headers  http.Header

	BuildJSON func(Message) (any, error)
	BuildForm func(Message) (url.Values, error)
	Parse     func(provider Provider, status int, headers http.Header, raw []byte) (*Response, error)

	HTTPClient *http.Client
}

// GenericProvider adapts simple HTTP JSON or form SMS APIs.
type GenericProvider struct {
	config GenericConfig
}

// NewGeneric creates a generic HTTP provider.
func NewGeneric(config GenericConfig) (*GenericProvider, error) {
	if config.Provider == "" {
		config.Provider = ProviderGeneric
	}
	if config.Method == "" {
		config.Method = http.MethodPost
	}
	if config.Encoding == "" {
		config.Encoding = EncodingJSON
	}
	if config.Endpoint == "" {
		return nil, fmt.Errorf("sms: endpoint is required")
	}
	if config.Headers == nil {
		config.Headers = http.Header{}
	}
	return &GenericProvider{config: config}, nil
}

// Provider returns the provider id.
func (provider *GenericProvider) Provider() Provider {
	return provider.config.Provider
}

// Send sends a message through the configured HTTP adapter.
func (provider *GenericProvider) Send(ctx context.Context, message Message) (*Response, error) {
	if err := message.Validate(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := provider.request(ctx, message)
	if err != nil {
		return nil, err
	}
	raw, headers, err := do(req, provider.config.HTTPClient)
	if err != nil {
		return nil, err
	}
	if provider.config.Parse != nil {
		return provider.config.Parse(provider.Provider(), http.StatusOK, headers, raw)
	}
	return parseLooseJSON(provider.Provider(), raw), nil
}

func (provider *GenericProvider) request(ctx context.Context, message Message) (*http.Request, error) {
	var body []byte
	contentType := ""
	switch provider.config.Encoding {
	case EncodingForm:
		values := url.Values{}
		if provider.config.BuildForm != nil {
			var err error
			values, err = provider.config.BuildForm(message)
			if err != nil {
				return nil, err
			}
		}
		body = []byte(values.Encode())
		contentType = "application/x-www-form-urlencoded"
	default:
		payload := any(message)
		if provider.config.BuildJSON != nil {
			var err error
			payload, err = provider.config.BuildJSON(message)
			if err != nil {
				return nil, err
			}
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = data
		contentType = "application/json; charset=utf-8"
	}

	req, err := http.NewRequestWithContext(ctx, provider.config.Method, provider.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	for key, values := range provider.config.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return req, nil
}

func parseLooseJSON(provider Provider, raw []byte) *Response {
	resp := &Response{
		Provider: provider,
		Raw:      raw,
		Success:  true,
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return resp
	}
	resp.Code = firstString(body, "code", "Code", "status", "Status", "error_code")
	resp.Message = firstString(body, "message", "Message", "msg", "Msg", "reason", "detail")
	resp.RequestID = firstString(body, "request_id", "RequestId", "requestId")
	if resp.Code != "" {
		normalized := strings.ToLower(resp.Code)
		resp.Success = normalized == "0" || normalized == "ok" || normalized == "success"
	}
	return resp
}

func firstString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := body[key]; ok && value != nil {
			return fmt.Sprint(value)
		}
	}
	return ""
}
