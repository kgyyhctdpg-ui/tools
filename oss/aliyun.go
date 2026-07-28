package oss

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/scoming-dev/tools/uniqueid"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// AliyunOSSConfig 阿里云OSS配置
type AliyunOSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
}

// AliyunOSSClient 阿里云OSS客户端
type AliyunOSSClient struct {
	client          *oss.Client
	bucketName      string
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	region          string
	product         string
}

// NewAliyunOSSClient 创建阿里云OSS客户端
func NewAliyunOSSClient(config AliyunOSSConfig) (*AliyunOSSClient, error) {
	// 创建OSS配置
	credentialsProvider := credentials.NewStaticCredentialsProvider(
		config.AccessKeyID,
		config.SecretAccessKey,
	)
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentialsProvider).
		WithEndpoint(config.Endpoint).
		WithRegion(config.Region)

	// 创建OSS客户端
	client := oss.NewClient(cfg)

	return &AliyunOSSClient{
		client:          client,
		bucketName:      config.BucketName,
		endpoint:        config.Endpoint,
		accessKeyID:     config.AccessKeyID,
		secretAccessKey: config.SecretAccessKey,
		region:          config.Region,
		product:         "oss",
	}, nil
}

func (a *AliyunOSSClient) GetPreFileName(suffix string) string {
	filePrev := "file"
	if suffix == "" {
		filePrev = "file"
	} else if strings.Contains("mp4", suffix) {
		filePrev = "video"
	} else if strings.Contains("mp3", suffix) {
		filePrev = "audio"
	} else if strings.Contains("jpeg,jpg,png,gif,svg", suffix) {
		filePrev = "images"
	} else if strings.Contains("doc,xls,docx,xlsx,ppt,pptx,pdf", suffix) {
		filePrev = "doc"
	} else {
		filePrev = "file"
	}
	return filePrev
}

// UploadFile 上传文件到阿里云OSS
func (a *AliyunOSSClient) UploadFile(ctx context.Context, localFilePath, objectName string) (string, error) {
	// 打开本地文件
	file, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", localFilePath, err)
	}
	defer file.Close()

	// 获取文件类型
	contentType := mime.TypeByExtension(filepath.Ext(localFilePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 上传文件到OSS
	_, err = a.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:      &a.bucketName,
		Key:         &objectName,
		Body:        file,
		ContentType: &contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to aliyun oss: %w", err)
	}

	// 生成访问URL
	url := fmt.Sprintf("https://%s.%s/%s", a.bucketName, strings.TrimPrefix(a.endpoint, "https://"), objectName)
	return url, nil
}

// GeneratePostPolicy 生成POST策略
type PostPolicy struct {
	Expiration string          `json:"expiration"`
	Conditions [][]interface{} `json:"conditions"`
}

// PolicyToken POST策略令牌
type PolicyToken struct {
	AccessKeyId string `json:"ossAccessKeyId"`
	Signature   string `json:"signature"`
	Policy      string `json:"policy"`
	Directory   string `json:"dir"`
}

// GeneratePostSignature 生成POST签名
func (a *AliyunOSSClient) GeneratePostSignature(policy string) (string, error) {
	// 使用HMAC-SHA1签名
	h := hmac.New(sha1.New, []byte(a.secretAccessKey))
	h.Write([]byte(policy))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

// GetUploadParams 获取上传参数（用于预签名URL）
func (a *AliyunOSSClient) GetUploadParams(suffix string, method string) (uploadUrl string, formData map[string]string, err error) {
	filePrev := a.GetPreFileName(suffix)
	key := uniqueid.GenSn("")
	datepath := filePrev
	filename := key + "." + suffix
	if suffix == "" {
		filename = key
	}
	filepath := path.Join(datepath, filename)

	if strings.ToLower(method) == "" || strings.ToLower(method) == "put" {
		// 生成PUT预签名URL
		result, err := a.client.Presign(context.Background(), &oss.PutObjectRequest{
			Bucket: &a.bucketName,
			Key:    &filepath,
		}, oss.PresignExpires(2*time.Hour))
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate presigned put url: %w", err)
		}
		return result.URL, nil, nil
	} else {
		// 生成POST策略
		utcTime := time.Now().UTC()
		date := utcTime.Format("20060102")
		expiration := utcTime.Add(1 * time.Hour)
		credential := fmt.Sprintf("%v/%v/%v/%v/aliyun_v4_request", a.accessKeyID, date, a.region, a.product)
		ossdata := utcTime.Format("20060102T150405Z")
		policyMap := map[string]any{
			"expiration": expiration.Format("2006-01-02T15:04:05.000Z"),
			"conditions": []any{
				map[string]string{"bucket": a.bucketName},
				map[string]string{"x-oss-signature-version": "OSS4-HMAC-SHA256"},
				map[string]string{"x-oss-credential": credential},
				map[string]string{"x-oss-date": ossdata},
			},
		}

		// 序列化策略
		policyJSON, err := json.Marshal(policyMap)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal policy: %w", err)
		}

		// Base64编码策略
		stringToSign := base64.StdEncoding.EncodeToString([]byte(policyJSON))

		// signing key
		hmacHash := func() hash.Hash { return sha256.New() }
		signingKey := "aliyun_v4" + a.secretAccessKey
		h1 := hmac.New(hmacHash, []byte(signingKey))
		io.WriteString(h1, date)
		h1Key := h1.Sum(nil)

		h2 := hmac.New(hmacHash, h1Key)
		io.WriteString(h2, a.region)
		h2Key := h2.Sum(nil)

		h3 := hmac.New(hmacHash, h2Key)
		io.WriteString(h3, a.product)
		h3Key := h3.Sum(nil)

		h4 := hmac.New(hmacHash, h3Key)
		io.WriteString(h4, "aliyun_v4_request")
		h4Key := h4.Sum(nil)

		// Signature
		h := hmac.New(hmacHash, h4Key)
		io.WriteString(h, stringToSign)
		signature := hex.EncodeToString(h.Sum(nil))

		formData = map[string]string{
			"x-oss-signature-version": "OSS4-HMAC-SHA256",
			"x-oss-credential":        credential,
			"x-oss-signature":         signature,
			"policy":                  stringToSign,
			"key":                     filepath,
			"x-oss-date":              ossdata,
		}

		// 构建POST上传URL
		uploadUrl := fmt.Sprintf("https://%s.%s", a.bucketName, strings.TrimPrefix(a.endpoint, "https://"))

		return uploadUrl, formData, nil
	}
}

// UploadFileWithPrefix 上传文件到阿里云OSS，自动生成对象名称
func (a *AliyunOSSClient) UploadFileWithPrefix(ctx context.Context, localFilePath string, prefix string) (string, string, error) {
	// 生成唯一的文件名
	fileName := filepath.Base(localFilePath)
	ext := filepath.Ext(fileName)
	fileType := strings.TrimPrefix(ext, ".")
	filePrev := a.GetPreFileName(fileType)

	nameWithoutExt := strings.TrimSuffix(fileName, ext)
	uniqueFileName := fmt.Sprintf("%s%s", nameWithoutExt, ext)

	// 构建对象名称
	objectName := ""
	if prefix != "" {
		objectName = path.Join(prefix, uniqueFileName)
	} else {
		objectName = path.Join(filePrev, uniqueFileName)
	}

	// 上传文件
	url, err := a.UploadFile(ctx, localFilePath, objectName)
	if err != nil {
		return "", "", err
	}

	return url, objectName, nil
}

// DeleteLocalFile 删除本地文件
func (a *AliyunOSSClient) DeleteLocalFile(filePath string) error {
	return deleteLocalFile(filePath)
}

// GeneratePostPolicy 生成POST上传策略
func (a *AliyunOSSClient) GeneratePostPolicy(suffix string, filepath string, expireTime int64) (map[string]any, error) {
	utcTime := time.Now().UTC()
	date := utcTime.Format("20060102")
	expiration := utcTime.Add(1 * time.Hour)
	policyMap := map[string]any{
		"expiration": expiration.Format("2006-01-02T15:04:05.000Z"),
		"conditions": []any{
			map[string]string{"bucket": a.bucketName},
			map[string]string{"x-oss-signature-version": "OSS4-HMAC-SHA256"},
			map[string]string{"x-oss-credential": fmt.Sprintf("%v/%v/%v/%v/aliyun_v4_request", a.accessKeyID, date, a.region, a.product)},
			map[string]string{"x-oss-date": utcTime.Format("20060102T150405Z")},
		},
	}

	return policyMap, nil
}
