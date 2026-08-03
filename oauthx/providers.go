package oauthx

import (
	"errors"
	"net/http"
	"strings"
)

const (
	ProviderGitHub         = "github"
	ProviderGitLab         = "gitlab"
	ProviderGoogle         = "google"
	ProviderMicrosoft      = "microsoft"
	ProviderWechatOpen     = "wechat_open"
	ProviderWechatOfficial = "wechat_official"
	ProviderDingTalk       = "dingtalk"
	ProviderFeishu         = "feishu"
)

// ProviderConfig returns a preset Config for common OAuth providers.
func ProviderConfig(provider, clientID, clientSecret, redirectURL string, scopes ...string) (Config, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderGitHub:
		return GitHub(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderGitLab:
		return GitLab(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderGoogle:
		return Google(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderMicrosoft:
		return Microsoft(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderWechatOpen:
		return WechatOpen(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderWechatOfficial:
		return WechatOfficial(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderDingTalk:
		return DingTalk(clientID, clientSecret, redirectURL, scopes...), nil
	case ProviderFeishu:
		return Feishu(clientID, clientSecret, redirectURL, scopes...), nil
	default:
		return Config{}, errors.New("oauthx: unknown provider")
	}
}

func GitHub(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:     ProviderGitHub,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       defaultScopes(scopes, "read:user", "user:email"),
	}
}

func GitLab(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:     ProviderGitLab,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://gitlab.com/oauth/authorize",
		TokenURL:     "https://gitlab.com/oauth/token",
		UserInfoURL:  "https://gitlab.com/oauth/userinfo",
		Scopes:       defaultScopes(scopes, "openid", "profile", "email"),
	}
}

func Google(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:     ProviderGoogle,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		RevokeURL:    "https://oauth2.googleapis.com/revoke",
		UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:       defaultScopes(scopes, "openid", "profile", "email"),
	}
}

func Microsoft(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return MicrosoftTenant("common", clientID, clientSecret, redirectURL, scopes...)
}

func MicrosoftTenant(tenant, clientID, clientSecret, redirectURL string, scopes ...string) Config {
	tenant = strings.Trim(strings.TrimSpace(tenant), "/")
	if tenant == "" {
		tenant = "common"
	}
	baseURL := "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0"
	return Config{
		Provider:     ProviderMicrosoft,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      baseURL + "/authorize",
		TokenURL:     baseURL + "/token",
		UserInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
		Scopes:       defaultScopes(scopes, "openid", "profile", "email", "offline_access"),
	}
}

func WechatOpen(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:          ProviderWechatOpen,
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		RedirectURL:       redirectURL,
		AuthURL:           "https://open.weixin.qq.com/connect/qrconnect",
		TokenURL:          "https://api.weixin.qq.com/sns/oauth2/access_token",
		UserInfoURL:       "https://api.weixin.qq.com/sns/userinfo",
		Scopes:            defaultScopes(scopes, "snsapi_login"),
		AuthClientIDParam: "appid",
		TokenMethod:       http.MethodGet,
		TokenStyle:        TokenRequestQuery,
		TokenFieldMap:     wechatTokenFieldMap(),
		UserInfoAuthStyle: UserInfoAuthQuery,
	}
}

func WechatOfficial(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:          ProviderWechatOfficial,
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		RedirectURL:       redirectURL,
		AuthURL:           "https://open.weixin.qq.com/connect/oauth2/authorize#wechat_redirect",
		TokenURL:          "https://api.weixin.qq.com/sns/oauth2/access_token",
		UserInfoURL:       "https://api.weixin.qq.com/sns/userinfo",
		Scopes:            defaultScopes(scopes, "snsapi_userinfo"),
		AuthClientIDParam: "appid",
		TokenMethod:       http.MethodGet,
		TokenStyle:        TokenRequestQuery,
		TokenFieldMap:     wechatTokenFieldMap(),
		UserInfoAuthStyle: UserInfoAuthQuery,
	}
}

func DingTalk(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:     ProviderDingTalk,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://login.dingtalk.com/oauth2/auth",
		TokenURL:     "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
		UserInfoURL:  "https://api.dingtalk.com/v1.0/contact/users/me",
		Scopes:       defaultScopes(scopes, "openid"),
		TokenStyle:   TokenRequestJSON,
		TokenFieldMap: map[string]string{
			"grant_type":    "grantType",
			"client_id":     "clientId",
			"client_secret": "clientSecret",
			"refresh_token": "refreshToken",
		},
	}
}

func Feishu(clientID, clientSecret, redirectURL string, scopes ...string) Config {
	return Config{
		Provider:     ProviderFeishu,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      "https://open.feishu.cn/open-apis/authen/v1/authorize",
		TokenURL:     "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
		UserInfoURL:  "https://open.feishu.cn/open-apis/authen/v1/user_info",
		Scopes:       append([]string(nil), scopes...),
		TokenStyle:   TokenRequestJSON,
	}
}

func defaultScopes(scopes []string, defaults ...string) []string {
	if len(scopes) > 0 {
		return append([]string(nil), scopes...)
	}
	return append([]string(nil), defaults...)
}

func wechatTokenFieldMap() map[string]string {
	return map[string]string{
		"client_id":     "appid",
		"client_secret": "secret",
	}
}
