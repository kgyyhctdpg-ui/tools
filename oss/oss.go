package oss

import (
	"context"
	"fmt"
)

type OssConfig struct {
	Type   string
	Minio  *MinioConfig
	Aliyun *AliyunOSSConfig
	Huawei *HuaweiOBSConfig
}

// OSSClient 统一的OSS客户端接口
type OSSClient interface {
	// GetPreFileName 根据文件后缀获取文件前缀
	GetPreFileName(suffix string) string

	// UploadFile 上传文件到OSS
	UploadFile(ctx context.Context, localFilePath, objectName string) (string, error)

	// GetUploadParams 获取上传参数（用于预签名URL）
	GetUploadParams(suffix string, method string) (uploadUrl string, formData map[string]string, err error)

	// UploadFileWithPrefix 上传文件到OSS，自动生成对象名称
	UploadFileWithPrefix(ctx context.Context, localFilePath string, prefix string) (string, string, error)

	// DeleteLocalFile 删除本地文件
	DeleteLocalFile(filePath string) error
}

// NewOSSClient 创建OSS客户端
func NewOSSClient(config OssConfig) (OSSClient, error) {
	switch config.Type {
	case "minio":
		return NewMinioClient(*config.Minio)
	case "aliyun":
		return NewAliyunOSSClient(*config.Aliyun)
	case "huawei":
		return NewHuaweiOBSClient(*config.Huawei)
	default:
		return nil, fmt.Errorf("unsupported OSS type: %s", config.Type)
	}
}
