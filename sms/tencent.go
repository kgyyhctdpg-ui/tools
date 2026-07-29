package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TencentCloudConfig configures Tencent Cloud SMS SendSms.
type TencentCloudConfig struct {
	SecretID    string
	SecretKey   string
	SmsSdkAppID string
	Region      string
	Endpoint    string
	ParamKeys   []string
	HTTPClient  *http.Client
}

// TencentCloudProvider sends SMS through Tencent Cloud SMS.
type TencentCloudProvider struct {
	config TencentCloudConfig
	now    func() time.Time
}

// NewTencentCloud creates a Tencent Cloud SMS provider.
func NewTencentCloud(config TencentCloudConfig) (*TencentCloudProvider, error) {
	if err := require(config.SecretID, "tencent secret id"); err != nil {
		return nil, err
	}
	if err := require(config.SecretKey, "tencent secret key"); err != nil {
		return nil, err
	}
	if err := require(config.SmsSdkAppID, "tencent sms sdk app id"); err != nil {
		return nil, err
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://sms.tencentcloudapi.com"
	}
	return &TencentCloudProvider{
		config: config,
		now:    time.Now,
	}, nil
}

// Provider returns tencentcloud.
func (provider *TencentCloudProvider) Provider() Provider {
	return ProviderTencentCloud
}

// Send sends a Tencent Cloud SendSms request.
func (provider *TencentCloudProvider) Send(ctx context.Context, message Message) (*Response, error) {
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

	payload := map[string]any{
		"PhoneNumberSet":   message.To,
		"SmsSdkAppId":      provider.config.SmsSdkAppID,
		"SignName":         message.SignName,
		"TemplateId":       message.TemplateID,
		"TemplateParamSet": message.paramSet(provider.config.ParamKeys),
	}
	if message.Extend != "" {
		payload["SessionContext"] = message.Extend
	}
	if message.SenderID != "" {
		payload["SenderId"] = message.SenderID
	}

	req, body, err := newJSONRequest(ctx, provider.config.Endpoint, payload)
	if err != nil {
		return nil, err
	}
	provider.sign(req, body)

	raw, _, err := do(req, provider.config.HTTPClient)
	if err != nil {
		return nil, err
	}

	var result struct {
		Response struct {
			RequestID     string `json:"RequestId"`
			SendStatusSet []struct {
				Code        string `json:"Code"`
				Message     string `json:"Message"`
				SerialNo    string `json:"SerialNo"`
				PhoneNumber string `json:"PhoneNumber"`
			} `json:"SendStatusSet"`
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	resp := &Response{
		Provider:  provider.Provider(),
		RequestID: result.Response.RequestID,
		Success:   true,
		Raw:       raw,
	}
	if result.Response.Error != nil {
		resp.Code = result.Response.Error.Code
		resp.Message = result.Response.Error.Message
		resp.Success = false
		return resp, nil
	}
	for _, item := range result.Response.SendStatusSet {
		if !strings.EqualFold(item.Code, "Ok") {
			resp.Success = false
		}
		if resp.Code == "" {
			resp.Code = item.Code
			resp.Message = item.Message
			resp.BizID = item.SerialNo
		}
	}
	return resp, nil
}

func (provider *TencentCloudProvider) sign(req *http.Request, payload []byte) {
	const service = "sms"
	const action = "SendSms"
	const version = "2021-01-11"
	const algorithm = "TC3-HMAC-SHA256"

	now := provider.now().UTC()
	timestamp := now.Unix()
	date := now.Format("2006-01-02")
	host := req.URL.Host

	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Version", version)
	if provider.config.Region != "" {
		req.Header.Set("X-TC-Region", provider.config.Region)
	}

	hashedPayload := sha256Hex(payload)
	canonicalHeaders := "content-type:application/json; charset=utf-8\nhost:" + host + "\nx-tc-action:" + strings.ToLower(action) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := "POST\n/\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + hashedPayload

	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := algorithm + "\n" + fmt.Sprintf("%d", timestamp) + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalRequest))
	secretDate := hmacSHA256([]byte("TC3"+provider.config.SecretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	authorization := algorithm + " Credential=" + provider.config.SecretID + "/" + credentialScope + ", SignedHeaders=" + signedHeaders + ", Signature=" + signature
	req.Header.Set("Authorization", authorization)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}
