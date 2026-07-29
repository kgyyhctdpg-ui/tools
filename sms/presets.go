package sms

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// ProviderInfo describes a provider supported by this package.
type ProviderInfo struct {
	Provider Provider
	Name     string
	Mode     string
	Note     string
}

// Catalog lists common SMS providers. "native" means this package implements
// the provider's request shape directly; "generic" means use NewGeneric with
// the vendor's HTTP API details.
func Catalog() []ProviderInfo {
	return []ProviderInfo{
		{ProviderAliyun, "阿里云短信", "native", "Dysmsapi SendSms"},
		{ProviderTencentCloud, "腾讯云短信", "native", "SMS SendSms 2021-01-11"},
		{ProviderYunpian, "云片", "native", "single_send 表单接口"},
		{ProviderSubmail, "Submail", "native", "xsend JSON 接口"},
		{ProviderJuhe, "聚合数据", "native", "sms/send 表单接口"},
		{ProviderLuosimao, "螺丝帽", "native", "Basic Auth 表单接口"},
		{ProviderChuanglan, "创蓝253", "native", "send/json 接口"},
		{ProviderHuaweiCloud, "华为云短信", "generic", "可通过 NewGeneric 接入 AK/SK 签名或代理接口"},
		{ProviderVolcengine, "火山引擎短信", "generic", "可通过 NewGeneric 接入火山签名或代理接口"},
		{ProviderQiniu, "七牛云短信", "generic", "可通过 NewGeneric 接入"},
		{ProviderBaiduCloud, "百度智能云短信", "generic", "可通过 NewGeneric 接入"},
		{ProviderUCloud, "UCloud 短信", "generic", "可通过 NewGeneric 接入"},
		{ProviderCloopen, "容联云通讯", "generic", "可通过 NewGeneric 接入"},
		{ProviderMontnets, "梦网云通信", "generic", "可通过 NewGeneric 接入"},
		{ProviderEmay, "亿美软通", "generic", "可通过 NewGeneric 接入"},
	}
}

// YunpianConfig configures Yunpian SMS.
type YunpianConfig struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// NewYunpian creates a Yunpian single_send provider. Message.Content is used as
// text.
func NewYunpian(config YunpianConfig) (*GenericProvider, error) {
	if err := require(config.APIKey, "yunpian api key"); err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://sms.yunpian.com/v2/sms/single_send.json"
	}
	return NewGeneric(GenericConfig{
		Provider:   ProviderYunpian,
		Endpoint:   config.Endpoint,
		Encoding:   EncodingForm,
		HTTPClient: config.HTTPClient,
		BuildForm: func(message Message) (url.Values, error) {
			if err := require(message.Content, "content"); err != nil {
				return nil, err
			}
			return url.Values{
				"apikey": {config.APIKey},
				"mobile": {message.phones(",")},
				"text":   {message.Content},
			}, nil
		},
		Parse: parseCodeZeroResponse,
	})
}

// SubmailConfig configures Submail SMS.
type SubmailConfig struct {
	AppID      string
	AppKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// NewSubmail creates a Submail xsend provider.
func NewSubmail(config SubmailConfig) (*GenericProvider, error) {
	if err := require(config.AppID, "submail app id"); err != nil {
		return nil, err
	}
	if err := require(config.AppKey, "submail app key"); err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://api-v4.mysubmail.com/sms/xsend.json"
	}
	return NewGeneric(GenericConfig{
		Provider:   ProviderSubmail,
		Endpoint:   config.Endpoint,
		Encoding:   EncodingJSON,
		HTTPClient: config.HTTPClient,
		BuildJSON: func(message Message) (any, error) {
			if err := require(message.TemplateID, "template id"); err != nil {
				return nil, err
			}
			return map[string]any{
				"appid":     config.AppID,
				"to":        message.phones(","),
				"project":   message.TemplateID,
				"signature": config.AppKey,
				"vars":      mapStringAny(message.TemplateParams),
			}, nil
		},
		Parse: parseSubmailResponse,
	})
}

// JuheConfig configures Juhe SMS.
type JuheConfig struct {
	Key        string
	Endpoint   string
	HTTPClient *http.Client
}

// NewJuhe creates a Juhe SMS provider.
func NewJuhe(config JuheConfig) (*GenericProvider, error) {
	if err := require(config.Key, "juhe key"); err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://v.juhe.cn/sms/send"
	}
	return NewGeneric(GenericConfig{
		Provider:   ProviderJuhe,
		Endpoint:   config.Endpoint,
		Encoding:   EncodingForm,
		HTTPClient: config.HTTPClient,
		BuildForm: func(message Message) (url.Values, error) {
			if err := require(message.TemplateID, "template id"); err != nil {
				return nil, err
			}
			return url.Values{
				"mobile":    {message.firstPhone()},
				"tpl_id":    {message.TemplateID},
				"tpl_value": {juheTplValue(message.TemplateParams)},
				"key":       {config.Key},
			}, nil
		},
		Parse: parseErrorCodeZeroResponse,
	})
}

// LuosimaoConfig configures Luosimao SMS.
type LuosimaoConfig struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// NewLuosimao creates a Luosimao SMS provider. Message.Content is used as text.
func NewLuosimao(config LuosimaoConfig) (*GenericProvider, error) {
	if err := require(config.APIKey, "luosimao api key"); err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://sms-api.luosimao.com/v1/send.json"
	}
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("api:"+config.APIKey)))
	return NewGeneric(GenericConfig{
		Provider:   ProviderLuosimao,
		Endpoint:   config.Endpoint,
		Encoding:   EncodingForm,
		Headers:    headers,
		HTTPClient: config.HTTPClient,
		BuildForm: func(message Message) (url.Values, error) {
			if err := require(message.Content, "content"); err != nil {
				return nil, err
			}
			return url.Values{
				"mobile":  {message.firstPhone()},
				"message": {message.Content},
			}, nil
		},
		Parse: parseErrorCodeZeroResponse,
	})
}

// ChuanglanConfig configures Chuanglan 253 SMS.
type ChuanglanConfig struct {
	Account    string
	Password   string
	Endpoint   string
	HTTPClient *http.Client
}

// NewChuanglan creates a Chuanglan send/json provider. Message.Content is used
// as msg.
func NewChuanglan(config ChuanglanConfig) (*GenericProvider, error) {
	if err := require(config.Account, "chuanglan account"); err != nil {
		return nil, err
	}
	if err := require(config.Password, "chuanglan password"); err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://smssh1.253.com/msg/send/json"
	}
	return NewGeneric(GenericConfig{
		Provider:   ProviderChuanglan,
		Endpoint:   config.Endpoint,
		Encoding:   EncodingJSON,
		HTTPClient: config.HTTPClient,
		BuildJSON: func(message Message) (any, error) {
			if err := require(message.Content, "content"); err != nil {
				return nil, err
			}
			return map[string]any{
				"account":  config.Account,
				"password": config.Password,
				"phone":    message.phones(","),
				"msg":      message.Content,
				"report":   "true",
			}, nil
		},
		Parse: parseCodeZeroResponse,
	})
}

func juheTplValue(params map[string]string) string {
	values := make(url.Values, len(params))
	for key, value := range params {
		if !strings.HasPrefix(key, "#") {
			key = "#" + key + "#"
		}
		values.Set(key, value)
	}
	return values.Encode()
}

func parseCodeZeroResponse(provider Provider, _ int, _ http.Header, raw []byte) (*Response, error) {
	body := map[string]any{}
	_ = json.Unmarshal(raw, &body)
	code := firstString(body, "code", "Code")
	return &Response{
		Provider:  provider,
		RequestID: firstString(body, "sid", "request_id", "RequestId"),
		Code:      code,
		Message:   firstString(body, "msg", "message", "Message", "detail"),
		Success:   code == "0",
		Raw:       raw,
	}, nil
}

func parseErrorCodeZeroResponse(provider Provider, _ int, _ http.Header, raw []byte) (*Response, error) {
	body := map[string]any{}
	_ = json.Unmarshal(raw, &body)
	code := firstString(body, "error_code", "code", "Code")
	return &Response{
		Provider:  provider,
		RequestID: firstString(body, "sid", "request_id", "RequestId"),
		Code:      code,
		Message:   firstString(body, "reason", "msg", "message", "Message"),
		Success:   code == "0",
		Raw:       raw,
	}, nil
}

func parseSubmailResponse(provider Provider, _ int, _ http.Header, raw []byte) (*Response, error) {
	body := map[string]any{}
	_ = json.Unmarshal(raw, &body)
	status := strings.ToLower(firstString(body, "status"))
	return &Response{
		Provider:  provider,
		RequestID: firstString(body, "send_id", "request_id"),
		Code:      firstString(body, "status", "code"),
		Message:   firstString(body, "msg", "message"),
		Success:   status == "success",
		Raw:       raw,
	}, nil
}
