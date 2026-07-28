package onlyoffice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// DocumentType 文档类型
type DocumentType string

const (
	DocumentTypeText         DocumentType = "text"         // Word文档
	DocumentTypeSpreadsheet  DocumentType = "spreadsheet"  // Excel文档
	DocumentTypePresentation DocumentType = "presentation" // PowerPoint文档
)

// GetDocumentType 根据文件扩展名获取文档类型
func GetDocumentType(fileName string) DocumentType {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".doc", ".docx", ".odt", ".rtf", ".txt", ".html", ".htm", ".mht", ".pdf", ".djvu", ".fb2", ".epub", ".xps":
		return DocumentTypeText
	case ".xls", ".xlsx", ".ods", ".csv":
		return DocumentTypeSpreadsheet
	case ".ppt", ".pptx", ".odp":
		return DocumentTypePresentation
	default:
		return DocumentTypeText
	}
}

// GetFileType 获取文件类型（用于OnlyOffice配置）
func GetFileType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	return strings.TrimPrefix(ext, ".")
}

// GenerateToken 生成OnlyOffice JWT token
func GenerateToken(payload map[string]interface{}, secret string) (string, error) {
	claims := jwt.MapClaims{}
	for k, v := range payload {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyToken 验证OnlyOffice JWT token
func VerifyToken(tokenString, secret string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		result := make(map[string]interface{})
		for k, v := range claims {
			result[k] = v
		}
		return result, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateDocumentAccessToken 生成文档访问token（只包含文档ID，不包含真实URL）
// 真实文件URL存储在Redis中，避免在token中暴露
// 返回的token只包含文档ID，即使token被解密也无法获取真实文件URL
func GenerateDocumentAccessToken(documentId string, secret string, expiresIn time.Duration) (string, error) {
	payload := jwt.MapClaims{
		"documentId": documentId, // 只存储文档ID，不存储真实URL
		"exp":        time.Now().Add(expiresIn).Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString([]byte(secret))
}

// VerifyDocumentAccessToken 验证并解析文档访问token，只返回文档ID
// 真实文件URL需要从Redis中获取
func VerifyDocumentAccessToken(tokenString, secret string) (documentId string, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	documentId, _ = claims["documentId"].(string)

	if documentId == "" {
		return "", fmt.Errorf("missing documentId in token claims")
	}

	return documentId, nil
}

// GetDocumentUrlKey 获取Redis中存储文档URL的key
func GetDocumentUrlKey(documentId string) string {
	return fmt.Sprintf("onlyoffice:document:url:%s", documentId)
}

// BuildConfig 构建OnlyOffice配置
// 注意：此函数不会在配置中暴露真实文件URL，而是使用加密的token
func BuildConfig(config *Config) (*OnlyOfficeConfig, error) {
	docType := GetDocumentType(config.DocumentName)
	fileType := GetFileType(config.DocumentName)

	// 生成文档密钥（用于标识文档）
	key := config.DocumentKey
	if key == "" {
		key = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// 生成文档访问token（只包含文档ID，不包含真实URL）
	// 真实文件URL将存储在Redis中，避免在token中暴露
	// 前端只能看到这个token，无法看到真实URL
	var docAccessToken string
	if config.Secret != "" {
		token, err := GenerateDocumentAccessToken(
			config.DocumentId,
			config.Secret,
			2*time.Hour, // token有效期2小时
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate document access token: %w", err)
		}
		docAccessToken = token

		// 注意：真实文件URL需要调用方存储到Redis中
		// 使用 GetDocumentUrlKey(config.DocumentId) 作为Redis key
		// 设置过期时间为2小时（与token过期时间一致）
	}

	// 获取基础URL（从配置或请求中获取）
	baseUrl := strings.TrimSuffix(config.CallbackUrl, "/app/base/onlyoffice/callback")

	// 构建文档URL（使用后端代理地址，传递加密的token而不是真实URL）
	docUrl := config.DocumentProxyUrl
	if docUrl == "" {
		path := ""
		if docAccessToken != "" {
			// 使用加密的token作为文档标识符
			path = fmt.Sprintf("/app/base/onlyoffice/document/%s", docAccessToken)
		} else {
			// 如果没有配置密钥，使用文档ID（不推荐，但为了兼容性保留）
			path = fmt.Sprintf("/app/base/onlyoffice/document/%s", config.DocumentId)
		}

		// 如果启用了签名验证，生成签名参数
		if config.EnableSignature && config.Secret != "" {
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			nonce := fmt.Sprintf("%d", time.Now().UnixNano())
			signature := GenerateRequestSignature(path, timestamp, nonce, config.Secret)

			// 将签名参数添加到URL
			docUrl = fmt.Sprintf(
				"%s%s?signature=%s&timestamp=%s&nonce=%s",
				baseUrl, path, signature, timestamp, nonce,
			)
		} else {
			docUrl = fmt.Sprintf("%s%s", baseUrl, path)
		}
	}

	// 构建配置对象
	// 注意：配置中的Url字段是后端代理地址，不包含真实文件URL
	onlyOfficeConfig := &OnlyOfficeConfig{
		Document: DocumentConfig{
			FileType: fileType,
			Key:      key,
			Title:    config.DocumentName,
			Url:      docUrl, // 这是后端代理地址，前端看不到真实文件URL
			Permissions: OnlyOfficePermissions{
				Chat:     false,
				Comment:  false,
				Copy:     false,
				Download: false,
				Print:    false,
				Edit:     true,
				Review:   false,
				CommentGroups: OnlyOfficeCommentGroups{
					Edit:   []string{},
					Remove: []string{},
					View:   "",
				},
				ReviewGroups:   []string{},
				UserInfoGroups: []string{},
			},
		},
		DocumentType: string(docType),
		EditorConfig: EditorConfig{
			Lang:        "zh-CN",
			Mode:        config.Mode,
			CallbackUrl: config.CallbackUrl,
			User: UserConfig{
				Id:   config.UserId,
				Name: config.UserName,
			},
			Customization: CustomizationConfig{
				CompactHeader:       true,
				CompactToolbar:      true,
				ToolbarHideFileName: true,
				Logo: LogoConfig{
					Visible: false,
					Url:     config.BaseUrl,
				},
				Plugins:   false,
				ForceSave: true, // 启用保存按钮
			},
		},
	}

	// 如果配置了JWT密钥，生成配置token（用于验证整个配置）
	if config.Secret != "" {
		// 先将完整的配置（不含token）转换为map，以便生成token
		var payload map[string]interface{}
		configBytes, err := json.Marshal(onlyOfficeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config to JSON for token generation: %w", err)
		}
		if err := json.Unmarshal(configBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config to map for token generation: %w", err)
		}
		delete(payload, "token")

		token, err := GenerateToken(payload, config.Secret)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}
		onlyOfficeConfig.Token = token
	}

	return onlyOfficeConfig, nil
}

// VerifyCallbackSignature 验证OnlyOffice回调请求的签名
func VerifyCallbackSignature(body []byte, signature string, secret string) bool {
	if secret == "" {
		return true // 如果没有配置密钥，跳过验证
	}

	// OnlyOffice使用HMAC-SHA256签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// GenerateRequestSignature 生成文档代理请求的签名
// 用于防止机器人伪装，只有知道密钥的服务器才能生成有效签名
// 签名包含：请求路径 + 时间戳 + nonce
func GenerateRequestSignature(path, timestamp, nonce, secret string) string {
	// 构建签名字符串：path + timestamp + nonce
	signString := fmt.Sprintf("%s|%s|%s", path, timestamp, nonce)

	// 使用HMAC-SHA256生成签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signString))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// VerifyRequestSignature 验证文档代理请求的签名
func VerifyRequestSignature(path, timestamp, nonce, signature, secret string, maxAge time.Duration) error {
	if secret == "" {
		return fmt.Errorf("密钥未配置")
	}

	// 验证时间戳（防止重放攻击）
	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的时间戳: %w", err)
	}

	requestTime := time.Unix(timestampInt, 0)
	now := time.Now()

	// 检查时间戳是否在有效范围内（防止过期请求和未来请求）
	if requestTime.Before(now.Add(-maxAge)) {
		return fmt.Errorf("请求已过期: %v", requestTime)
	}
	if requestTime.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("请求时间戳无效（未来时间）: %v", requestTime)
	}

	// 验证签名
	expectedSignature := GenerateRequestSignature(path, timestamp, nonce, secret)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("签名验证失败")
	}

	return nil
}

// Config OnlyOffice配置构建参数
type Config struct {
	DocumentId       string        // 文档ID
	DocumentName     string        // 文档名称
	DocumentKey      string        // 文档密钥（可选，不提供则自动生成）
	DocumentProxyUrl string        // 文档代理URL（可选，不提供则自动生成）
	BaseUrl          string        // 基础URL（用于生成代理URL和回调URL）
	CallbackUrl      string        // 回调URL（可选，不提供则自动生成）
	Mode             string        // 编辑模式: edit/view
	UserId           string        // 用户ID
	UserName         string        // 用户名称
	Secret           string        // JWT密钥（必需，用于加密文档访问token）
	EnableSignature  bool          // 是否启用请求签名验证
	SignatureMaxAge  time.Duration // 签名有效期
}

// OnlyOfficeConfig OnlyOffice配置结构
type OnlyOfficeConfig struct {
	Document     DocumentConfig `json:"document"`
	DocumentType string         `json:"documentType"`
	EditorConfig EditorConfig   `json:"editorConfig"`
	Token        string         `json:"token,omitempty"`
}

type OnlyOfficeCommentGroups struct {
	Edit   []string `json:"edit"`
	Remove []string `json:"remove"`
	View   string   `json:"view"`
}

type OnlyOfficePermissions struct {
	Chat                    bool                    `json:"chat"`
	Comment                 bool                    `json:"comment"`
	Copy                    bool                    `json:"copy"`
	CommentGroups           OnlyOfficeCommentGroups `json:"commentGroups"`
	DeleteCommentAuthorOnly bool                    `json:"deleteCommentAuthorOnly"`
	Download                bool                    `json:"download"`
	Edit                    bool                    `json:"edit"`
	EditCommentAuthorOnly   bool                    `json:"editCommentAuthorOnly"`
	FillForms               bool                    `json:"fillForms"`
	ModifyContentControl    bool                    `json:"modifyContentControl"`
	ModifyFilter            bool                    `json:"modifyFilter"`
	Print                   bool                    `json:"print"`
	Protect                 bool                    `json:"protect"`
	Review                  bool                    `json:"review"`
	ReviewGroups            []string                `json:"reviewGroups"`
	UserInfoGroups          []string                `json:"userInfoGroups"`
}

// DocumentConfig 文档配置
type DocumentConfig struct {
	FileType    string                `json:"fileType"`
	Key         string                `json:"key"`
	Title       string                `json:"title"`
	Url         string                `json:"url"`
	Permissions OnlyOfficePermissions `json:"permissions"`
}

// EditorConfig 编辑器配置
type EditorConfig struct {
	Lang          string              `json:"lang"`
	Mode          string              `json:"mode"`
	CallbackUrl   string              `json:"callbackUrl"`
	User          UserConfig          `json:"user"`
	Customization CustomizationConfig `json:"customization"`
}

// UserConfig 用户配置
type UserConfig struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// CustomizationConfig 自定义配置
type CustomizationConfig struct {
	Logo                LogoConfig `json:"logo"`
	Plugins             bool       `json:"plugins"`
	CompactHeader       bool       `json:"compactHeader"`
	CompactToolbar      bool       `json:"compactToolbar"`
	ToolbarHideFileName bool       `json:"toolbarHideFileName"`
	ForceSave           bool       `json:"forcesave"` // 启用强制保存按钮
}

// LogoConfig 自定义logo配置
type LogoConfig struct {
	Visible bool   `json:"visible"`
	Url     string `json:"url"`
}

// CallbackRequest OnlyOffice回调请求
type CallbackRequest struct {
	Actions       []CallbackAction `json:"actions"`
	Key           string           `json:"key"`
	Status        int              `json:"status"`
	Url           string           `json:"url,omitempty"`
	ChangesUrl    string           `json:"changesurl,omitempty"`
	History       *CallbackHistory `json:"history,omitempty"`
	Forcesavetype int              `json:"forcesavetype,omitempty"`
}

// CallbackAction 回调操作
type CallbackAction struct {
	Type   int    `json:"type"`
	Userid string `json:"userid"`
	Key    string `json:"key"`
}

// CallbackHistory 回调历史
type CallbackHistory struct {
	ServerVersion string           `json:"serverVersion"`
	Changes       []CallbackChange `json:"changes"`
}

// CallbackChange 回调变更
type CallbackChange struct {
	Created string `json:"created"`
	User    struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
}

// CacheKey 缓存键
type CacheValue struct {
	ThrId   string `json:"thr_id"`
	ThrType string `json:"thr_type"`
	File    string `json:"file"`
}

// ParseCallbackRequest 解析回调请求
func ParseCallbackRequest(body []byte) (*CallbackRequest, error) {
	var req CallbackRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse callback request: %w", err)
	}
	return &req, nil
}

// GetDocumentServerUrl 获取OnlyOffice Document Server的API地址
func GetDocumentServerUrl(baseUrl string) string {
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	return fmt.Sprintf("%s/web-apps/apps/api/documents/api.js", baseUrl)
}

// BuildDocumentServerScriptUrl 构建OnlyOffice Document Server脚本URL
func BuildDocumentServerScriptUrl(documentServerUrl string) string {
	u, err := url.Parse(documentServerUrl)
	if err != nil {
		return ""
	}
	u.Path = "/web-apps/apps/api/documents/api.js"
	return u.String()
}
