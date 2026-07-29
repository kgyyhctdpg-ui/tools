// Package sms provides a unified SMS sending interface with common provider
// adapters.
package sms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Provider identifies an SMS provider.
type Provider string

const (
	ProviderAliyun       Provider = "aliyun"
	ProviderTencentCloud Provider = "tencentcloud"
	ProviderHuaweiCloud  Provider = "huaweicloud"
	ProviderVolcengine   Provider = "volcengine"
	ProviderYunpian      Provider = "yunpian"
	ProviderSubmail      Provider = "submail"
	ProviderJuhe         Provider = "juhe"
	ProviderLuosimao     Provider = "luosimao"
	ProviderChuanglan    Provider = "chuanglan"
	ProviderQiniu        Provider = "qiniu"
	ProviderBaiduCloud   Provider = "baiducloud"
	ProviderUCloud       Provider = "ucloud"
	ProviderCloopen      Provider = "cloopen"
	ProviderMontnets     Provider = "montnets"
	ProviderEmay         Provider = "emay"
	ProviderGeneric      Provider = "generic"
)

var (
	ErrInvalidMessage      = errors.New("sms: invalid message")
	ErrProviderNotFound    = errors.New("sms: provider not found")
	ErrUnsupportedProvider = errors.New("sms: unsupported provider")
)

// Message is the provider-neutral SMS request.
type Message struct {
	To []string

	SignName         string
	TemplateID       string
	TemplateParams   map[string]string
	TemplateParamSet []string

	Content  string
	Extend   string
	SenderID string
}

// Validate checks the message has enough information to send.
func (m Message) Validate() error {
	if len(m.To) == 0 {
		return fmt.Errorf("%w: recipient is required", ErrInvalidMessage)
	}
	for _, phone := range m.To {
		if strings.TrimSpace(phone) == "" {
			return fmt.Errorf("%w: recipient is empty", ErrInvalidMessage)
		}
	}
	if strings.TrimSpace(m.TemplateID) == "" && strings.TrimSpace(m.Content) == "" {
		return fmt.Errorf("%w: template id or content is required", ErrInvalidMessage)
	}
	return nil
}

func (m Message) phones(sep string) string {
	return strings.Join(m.To, sep)
}

func (m Message) firstPhone() string {
	if len(m.To) == 0 {
		return ""
	}
	return m.To[0]
}

func (m Message) paramSet(keys []string) []string {
	if len(m.TemplateParamSet) > 0 {
		return append([]string(nil), m.TemplateParamSet...)
	}
	if len(m.TemplateParams) == 0 {
		return nil
	}
	if len(keys) == 0 {
		keys = make([]string, 0, len(m.TemplateParams))
		for key := range m.TemplateParams {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, m.TemplateParams[key])
	}
	return result
}

func (m Message) paramsJSON() (string, error) {
	if len(m.TemplateParams) == 0 {
		return "", nil
	}
	data, err := json.Marshal(m.TemplateParams)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Response is a normalized provider response.
type Response struct {
	Provider  Provider
	RequestID string
	BizID     string
	Code      string
	Message   string
	Success   bool
	Raw       []byte
}

// Sender sends SMS messages through one provider.
type Sender interface {
	Provider() Provider
	Send(ctx context.Context, message Message) (*Response, error)
}

// StatusError reports a non-2xx provider HTTP response.
type StatusError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (err *StatusError) Error() string {
	body := strings.TrimSpace(string(err.Body))
	if body == "" {
		return fmt.Sprintf("sms: provider http status %s", err.Status)
	}
	return fmt.Sprintf("sms: provider http status %s: %s", err.Status, body)
}
