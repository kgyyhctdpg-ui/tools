package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientRoutesProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
	}))
	defer server.Close()

	sender, err := NewYunpian(YunpianConfig{APIKey: "key", Endpoint: server.URL})
	if err != nil {
		t.Fatalf("NewYunpian failed: %v", err)
	}
	client := NewClient(sender)
	resp, err := client.Send(context.Background(), Message{
		To:      []string{"13800138000"},
		Content: "【签名】验证码 1234",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !resp.Success || resp.Provider != ProviderYunpian {
		t.Fatalf("response = %#v", resp)
	}
}

func TestYunpianFormProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm failed: %v", err)
		}
		if r.Form.Get("apikey") != "key" || r.Form.Get("mobile") != "13800138000,13900139000" {
			t.Fatalf("unexpected form: %s", r.Form.Encode())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
	}))
	defer server.Close()

	sender, err := NewYunpian(YunpianConfig{APIKey: "key", Endpoint: server.URL})
	if err != nil {
		t.Fatalf("NewYunpian failed: %v", err)
	}
	resp, err := sender.Send(context.Background(), Message{
		To:      []string{"13800138000", "13900139000"},
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !resp.Success {
		t.Fatalf("response = %#v", resp)
	}
}

func TestAliyunProviderBuildsSignedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("Action") != "SendSms" || query.Get("PhoneNumbers") != "13800138000" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		if query.Get("Signature") == "" || query.Get("TemplateParam") == "" {
			t.Fatalf("missing signature or template params: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"Code":      "OK",
			"Message":   "OK",
			"RequestId": "req-1",
			"BizId":     "biz-1",
		})
	}))
	defer server.Close()

	sender, err := NewAliyun(AliyunConfig{
		AccessKeyID:     "id",
		AccessKeySecret: "secret",
		Endpoint:        server.URL,
	})
	if err != nil {
		t.Fatalf("NewAliyun failed: %v", err)
	}
	resp, err := sender.Send(context.Background(), Message{
		To:             []string{"13800138000"},
		SignName:       "签名",
		TemplateID:     "SMS_123",
		TemplateParams: map[string]string{"code": "1234"},
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !resp.Success || resp.RequestID != "req-1" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestTencentCloudProviderBuildsSignedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "TC3-HMAC-SHA256") {
			t.Fatalf("missing authorization: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-TC-Action") != "SendSms" {
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["SmsSdkAppId"] != "app" || payload["TemplateId"] != "123" {
			t.Fatalf("payload = %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Response": map[string]any{
				"RequestId": "req-1",
				"SendStatusSet": []map[string]string{
					{"Code": "Ok", "Message": "send success", "SerialNo": "s-1"},
				},
			},
		})
	}))
	defer server.Close()

	sender, err := NewTencentCloud(TencentCloudConfig{
		SecretID:    "id",
		SecretKey:   "secret",
		SmsSdkAppID: "app",
		Endpoint:    server.URL,
		ParamKeys:   []string{"code"},
	})
	if err != nil {
		t.Fatalf("NewTencentCloud failed: %v", err)
	}
	resp, err := sender.Send(context.Background(), Message{
		To:             []string{"+8613800138000"},
		SignName:       "签名",
		TemplateID:     "123",
		TemplateParams: map[string]string{"code": "1234"},
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !resp.Success || resp.BizID != "s-1" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestCatalogContainsCommonProviders(t *testing.T) {
	items := Catalog()
	seen := map[Provider]bool{}
	for _, item := range items {
		seen[item.Provider] = true
	}
	for _, provider := range []Provider{ProviderAliyun, ProviderTencentCloud, ProviderHuaweiCloud, ProviderVolcengine, ProviderYunpian, ProviderSubmail, ProviderJuhe, ProviderChuanglan} {
		if !seen[provider] {
			t.Fatalf("catalog missing %s", provider)
		}
	}
}
