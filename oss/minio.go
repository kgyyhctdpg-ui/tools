package oss

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/scoming-dev/tools/uniqueid"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioConfig MinIO配置
type MinioConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

// MinioClient MinIO客户端
type MinioClient struct {
	client     *minio.Client
	bucketName string
}

// NewMinioClient 创建MinIO客户端
func NewMinioClient(config MinioConfig) (*MinioClient, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioClient{
		client:     client,
		bucketName: config.BucketName,
	}, nil
}

func (m *MinioClient) GetPreFileName(suffix string) string {
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

// UploadFile 上传文件到MinIO
func (m *MinioClient) UploadFile(ctx context.Context, localFilePath, objectName string) (string, error) {
	// 打开本地文件
	file, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", localFilePath, err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// 获取文件类型
	contentType := mime.TypeByExtension(filepath.Ext(localFilePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 上传文件到MinIO
	_, err = m.client.PutObject(ctx, m.bucketName, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to minio: %w", err)
	}

	// 生成访问URL
	url := fmt.Sprintf("%s://%s/%s/%s", m.client.EndpointURL().Scheme, m.client.EndpointURL().Host, m.bucketName, objectName)
	return url, nil
}

func (m *MinioClient) GetUploadParams(suffix string, method string) (uploadUrl string, formData map[string]string, err error) {

	filePrev := m.GetPreFileName(suffix)
	key := uniqueid.GenSn("")
	datepath := filePrev
	filename := key + "." + suffix
	if suffix == "" {
		filename = key
	}
	filepath := path.Join(datepath, filename)

	var presignedURL *url.URL
	if strings.ToLower(method) == "" || strings.ToLower(method) == "put" {
		presignedURL, err = m.client.PresignedPutObject(context.Background(), m.bucketName, filepath, time.Second*2*60*60)
		if err != nil {
			return "", nil, err
		}
	} else {
		contextType := mime.TypeByExtension(suffix)
		policy := minio.NewPostPolicy()
		policy.SetBucket(m.bucketName)
		policy.SetKey(filepath)
		policy.SetExpires(time.Now().UTC().AddDate(0, 2, 0))
		policy.SetContentType(contextType)

		presignedURL, formData, err = m.client.PresignedPostPolicy(context.Background(), policy)
		if err != nil {
			return "", nil, err
		}
	}
	return presignedURL.String(), formData, nil
}

// UploadFileWithPrefix 上传文件到MinIO，自动生成对象名称
func (m *MinioClient) UploadFileWithPrefix(ctx context.Context, localFilePath string, prefix string) (string, string, error) {
	// 生成唯一的文件名
	fileName := filepath.Base(localFilePath)
	ext := filepath.Ext(fileName)
	fileType := strings.TrimPrefix(ext, ".")
	filePrev := m.GetPreFileName(fileType)

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
	url, err := m.UploadFile(ctx, localFilePath, objectName)
	if err != nil {
		return "", "", err
	}

	return url, objectName, nil
}

// DeleteLocalFile 删除本地文件
func (m *MinioClient) DeleteLocalFile(filePath string) error {
	return deleteLocalFile(filePath)
}
