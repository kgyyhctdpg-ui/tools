package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AliyunConfig configures Aliyun Dysmsapi SendSms.
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	RegionID        string
	Endpoint        string
	HTTPClient      *http.Client
}

// AliyunProvider sends SMS through Aliyun Dysmsapi.
type AliyunProvider struct {
	config AliyunConfig
	now    func() time.Time
	nonce  func() string
}

// NewAliyun creates an Aliyun SMS provider.
func NewAliyun(config AliyunConfig) (*AliyunProvider, error) {
	if err := require(config.AccessKeyID, "aliyun access key id"); err != nil {
		return nil, err
	}
	if err := require(config.AccessKeySecret, "aliyun access key secret"); err != nil {
		return nil, err
	}
	if config.RegionID == "" {
		config.RegionID = "cn-hangzhou"
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://dysmsapi.aliyuncs.com"
	}
	return &AliyunProvider{
		config: config,
		now:    time.Now,
		nonce: func() string {
			return fmt.Sprintf("%d", time.Now().UnixNano())
		},
	}, nil
}

// Provider returns aliyun.
func (provider *AliyunProvider) Provider() Provider {
	return ProviderAliyun
}

// Send sends an Aliyun SendSms request.
func (provider *AliyunProvider) Send(ctx context.Context, message Message) (*Response, error) {
	if err := message.Validate(); err != nil {
		return nil, err
	}
	if err := require(message.SignName, "sign name"); err != nil {
		return nil, err
	}
	if err := require(message.TemplateID, "template id"); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	params, err := provider.params(message)
	if err != nil {
		return nil, err
	}
	params.Set("Signature", provider.signature(params))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.config.Endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	raw, _, err := do(req, provider.config.HTTPClient)
	if err != nil {
		return nil, err
	}

	var body struct {
		RequestID string `json:"RequestId"`
		BizID     string `json:"BizId"`
		Code      string `json:"Code"`
		Message   string `json:"Message"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return &Response{
		Provider:  provider.Provider(),
		RequestID: body.RequestID,
		BizID:     body.BizID,
		Code:      body.Code,
		Message:   body.Message,
		Success:   strings.EqualFold(body.Code, "OK"),
		Raw:       raw,
	}, nil
}

func (provider *AliyunProvider) params(message Message) (url.Values, error) {
	templateParams, err := message.paramsJSON()
	if err != nil {
		return nil, err
	}
	now := provider.now().UTC()
	params := url.Values{}
	params.Set("Action", "SendSms")
	params.Set("Version", "2017-05-25")
	params.Set("RegionId", provider.config.RegionID)
	params.Set("PhoneNumbers", message.phones(","))
	params.Set("SignName", message.SignName)
	params.Set("TemplateCode", message.TemplateID)
	params.Set("AccessKeyId", provider.config.AccessKeyID)
	params.Set("Format", "JSON")
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", provider.nonce())
	params.Set("Timestamp", now.Format("2006-01-02T15:04:05Z"))
	if templateParams != "" {
		params.Set("TemplateParam", templateParams)
	}
	if message.Extend != "" {
		params.Set("OutId", message.Extend)
	}
	return params, nil
}

func (provider *AliyunProvider) signature(params url.Values) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, aliyunEncode(key)+"="+aliyunEncode(params.Get(key)))
	}
	canonicalQuery := strings.Join(pairs, "&")
	stringToSign := "GET&%2F&" + aliyunEncode(canonicalQuery)
	mac := hmac.New(sha1.New, []byte(provider.config.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func aliyunEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
