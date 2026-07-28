# OSS 统一接口包

这个包提供了统一的OSS（对象存储服务）接口，支持MinIO和阿里云OSS，确保API的一致性。

## 功能特性

- 统一的OSS客户端接口
- 支持MinIO和阿里云OSS
- 文件上传、预签名URL生成
- 自动文件类型分类
- 统一的错误处理

## 支持的OSS类型

- **MinIO**: 开源对象存储服务
- **阿里云OSS**: 阿里云对象存储服务

## 使用方法

### 1. 创建MinIO客户端

```go
import "github.com/scoming-dev/tools/oss"

// 配置OSS
ossConfig := oss.OssConfig{
    Type: "minio",
    Minio: oss.MinioConfig{
        Endpoint:        "localhost:9000",
        AccessKeyID:     "minioadmin",
        SecretAccessKey: "minioadmin",
        BucketName:      "my-bucket",
        UseSSL:          false,
    },
}

// 创建客户端
client, err := oss.NewOSSClient(ossConfig)
if err != nil {
    log.Fatal(err)
}
```

### 2. 创建阿里云OSS客户端

```go
// 配置OSS
ossConfig := oss.OssConfig{
    Type: "aliyun",
    Aliyun: oss.AliyunOSSConfig{
        Endpoint:        "https://oss-cn-hangzhou.aliyuncs.com",
        AccessKeyID:     "your-access-key-id",
        SecretAccessKey: "your-secret-access-key",
        BucketName:      "my-bucket",
        Region:          "cn-hangzhou",
    },
}

// 创建客户端
client, err := oss.NewOSSClient(ossConfig)
if err != nil {
    log.Fatal(err)
}
```

### 3. 使用统一的接口

```go
ctx := context.Background()

// 上传文件
url, objectName, err := client.UploadFileWithPrefix(ctx, "/path/to/file.jpg", "images")
if err != nil {
    log.Printf("Upload failed: %v", err)
} else {
    fmt.Printf("Uploaded to: %s\n", url)
}

// 获取预签名URL
presignedURL, formData, err := client.GetUploadParams("jpg", "put")
if err != nil {
    log.Printf("Failed to get presigned URL: %v", err)
} else {
    fmt.Printf("Presigned URL: %s\n", presignedURL)
}

// 删除本地文件
err = client.DeleteLocalFile("/path/to/file.jpg")
if err != nil {
    log.Printf("Failed to delete local file: %v", err)
}
```

## 接口方法

### OSSClient 接口

```go
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
```

## 文件类型分类

系统会根据文件后缀自动分类：

- `mp4`: video/
- `mp3`: audio/
- `jpeg,jpg,png,gif`: images/
- `doc,xls,docx,xlsx,ppt,pptx,pdf`: doc/
- 其他: file/

## 配置说明

### OSS配置结构

```go
type OssConfig struct {
    Type   string           // OSS类型: "minio" 或 "aliyun"
    Minio  MinioConfig      // MinIO配置
    Aliyun AliyunOSSConfig  // 阿里云OSS配置
}
```

### MinIO配置

```go
type MinioConfig struct {
    Endpoint        string // MinIO服务地址
    AccessKeyID     string // 访问密钥ID
    SecretAccessKey string // 访问密钥
    BucketName      string // 存储桶名称
    UseSSL          bool   // 是否使用SSL
}
```

### 阿里云OSS配置

```go
type AliyunOSSConfig struct {
    Endpoint        string // OSS端点
    AccessKeyID     string // 访问密钥ID
    SecretAccessKey string // 访问密钥
    BucketName      string // 存储桶名称
    Region          string // 区域
}
```

## 配置解析支持

`NewOSSClient` 函数支持多种配置格式：

1. **OssConfig结构体**：直接传入OssConfig类型
2. **map[string]interface{}**：从配置文件解析的map类型
3. **结构体指针**：指向OssConfig的指针
4. **反射解析**：通过反射解析任意结构体

这使得该包可以轻松集成到各种配置系统中。

## 注意事项

1. 确保已安装相应的依赖包
2. 阿里云OSS需要有效的AccessKey和SecretKey
3. MinIO需要正确配置的存储桶
4. 预签名URL的POST方法在阿里云OSS中暂未完全实现
5. 配置中的Type字段必须指定为"minio"或"aliyun"

## 依赖

- `github.com/minio/minio-go/v7` - MinIO客户端
- `github.com/aliyun/alibabacloud-oss-go-sdk-v2` - 阿里云OSS客户端
- `github.com/scoming-dev/tools/uniqueid` - 唯一ID生成
