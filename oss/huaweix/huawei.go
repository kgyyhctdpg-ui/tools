package huaweix

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/scoming-dev/tools/oss/internal/common"
)

// Config 华为云OBS配置
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
}

// Client 华为云OBS客户端
type Client struct {
	client          *obs.ObsClient
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	bucketName      string
}

// NewClient 创建华为云OBS客户端
func NewClient(config Config) (*Client, error) {
	if config.Endpoint == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" || config.BucketName == "" {
		return nil, fmt.Errorf("invalid huawei obs config")
	}

	client, err := obs.New(config.AccessKeyID, config.SecretAccessKey, fmt.Sprintf("https://%s", config.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create huawei obs client: %w", err)
	}

	return &Client{
		client:          client,
		endpoint:        config.Endpoint,
		accessKeyID:     config.AccessKeyID,
		secretAccessKey: config.SecretAccessKey,
		bucketName:      config.BucketName,
	}, nil
}

// GetPreFileName 根据文件后缀获取文件前缀
func (h *Client) GetPreFileName(suffix string) string {
	filePrev := "file"
	if suffix == "" {
		filePrev = "file"
	} else if strings.Contains("mp4", suffix) {
		filePrev = "video"
	} else if strings.Contains("mp3", suffix) {
		filePrev = "audio"
	} else if strings.Contains("jpeg,jpg,png,gif", suffix) {
		filePrev = "images"
	} else if strings.Contains("doc,xls,docx,xlsx,ppt,pptx,pdf", suffix) {
		filePrev = "doc"
	} else {
		filePrev = "file"
	}
	return filePrev
}

// UploadFile 上传文件到华为云OBS
func (h *Client) UploadFile(ctx context.Context, localFilePath, objectName string) (string, error) {
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

	// 生成上传URL (使用预签名URL)
	putObjectOutput, err := h.client.CreateSignedUrl(&obs.CreateSignedUrlInput{
		Method:  obs.HttpMethodPut,
		Bucket:  h.bucketName,
		Key:     objectName,
		Expires: 3600,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}
	fmt.Println(objectName)
	fmt.Printf("SignedUrl:%s\n", putObjectOutput.SignedUrl)
	fmt.Printf("ActualSignedRequestHeaders:%v\n", putObjectOutput.ActualSignedRequestHeaders)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putObjectOutput.SignedUrl, file)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置Content-Type
	req.Header = putObjectOutput.ActualSignedRequestHeaders
	req.Header.Set("Content-Type", contentType)
	// req.ContentLength = fileInfo.Size()

	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to huawei obs: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload file to huawei obs, status: %d, body: %s", resp.StatusCode, string(body))
	}

	// 生成访问URL (不带签名的公共URL)
	accessUrl := fmt.Sprintf("https://%s.%s/%s", h.bucketName, h.endpoint, objectName)
	return accessUrl, nil
}

// GetUploadParams 获取上传参数（用于预签名URL或表单上传）
func (h *Client) GetUploadParams(suffix string, method string) (uploadUrl string, formData map[string]string, err error) {
	// 生成文件名
	filePrev := h.GetPreFileName(suffix)
	uuid := common.GenNumericID()

	objectName := fmt.Sprintf("%s/%s.%s", filePrev, uuid, suffix)
	if suffix == "" {
		objectName = fmt.Sprintf("%s/%s", filePrev, uuid)
	}

	if method == "post" {

		output, _ := h.client.CreateBrowserBasedSignature(&obs.CreateBrowserBasedSignatureInput{
			Bucket:  h.bucketName,
			Key:     objectName,
			Expires: 3600,
		})

		// 获取文件类型
		contentType := "application/octet-stream"
		if suffix != "" {
			if ct := mime.TypeByExtension("." + suffix); ct != "" {
				contentType = ct
			}
		}

		formData = map[string]string{
			"AccessKeyId":  h.accessKeyID,
			"signature":    output.Signature,
			"policy":       output.Policy,
			"key":          objectName,
			"Content-Type": contentType,
		}
		uploadUrl := fmt.Sprintf("https://%s.%s/", h.bucketName, strings.TrimPrefix(h.endpoint, "https://"))
		return uploadUrl, formData, nil
	} else {
		// 生成预签名URL
		output, err := h.client.CreateSignedUrl(&obs.CreateSignedUrlInput{
			Method:  obs.HttpMethodPut,
			Bucket:  h.bucketName,
			Key:     objectName,
			Expires: 3600,
		})
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate presigned url: %w", err)
		}

		return output.SignedUrl, nil, nil
	}
}

// UploadFileWithPrefix 上传文件到OSS，自动生成对象名称
func (h *Client) UploadFileWithPrefix(ctx context.Context, localFilePath string, prefix string) (string, string, error) {
	// 生成唯一的文件名
	fileName := filepath.Base(localFilePath)
	ext := filepath.Ext(fileName)
	fileType := strings.TrimPrefix(ext, ".")
	filePrev := h.GetPreFileName(fileType)

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
	accessUrl, err := h.UploadFile(ctx, localFilePath, objectName)
	if err != nil {
		return "", "", err
	}

	return accessUrl, objectName, nil
}

// DeleteLocalFile 删除本地文件
func (h *Client) DeleteLocalFile(filePath string) error {
	return common.DeleteLocalFile(filePath)
}
