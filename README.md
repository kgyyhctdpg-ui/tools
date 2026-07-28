# tools

`github.com/scoming-dev/tools` 是一组 Go 业务工具包，当前包含 DOCX 生成、对象存储、验证码、OnlyOffice 配置、HTTP 请求、唯一 ID、Casbin RBAC 和通用工具方法。

## 环境要求

- Go 1.26.1
- 使用 DOCX 渲染、OSS、验证码等包时，需要按各包配置对应的第三方服务或密钥。

## 安装

```bash
go get github.com/scoming-dev/tools
```

或在当前仓库内运行测试：

```bash
go test ./...
```

## 包说明

| 包 | 说明 |
| --- | --- |
| `docx` | Markdown 转 DOCX，支持标题、目录、列表、表格、图片、SVG、原生 Office/WPS 公式和可扩展插件。 |
| `oss` | 统一对象存储接口，支持 MinIO、阿里云 OSS、华为 OBS。 |
| `captcha` | 点击、滑块、旋转验证码生成和缓存校验。 |
| `onlyoffice` | OnlyOffice 文档类型识别、JWT、文档配置构建。 |
| `request` | HTTP GET/POST 请求工具，支持超时。 |
| `uniqueid` | 业务单号和 Sonyflake ID 生成。 |
| `casbinx` | 基于 GORM 的 Casbin RBAC 初始化工具。 |
| `utils` | 随机串、MD5、JSON 压缩、手机号/邮箱校验、文件和图片工具、精确小数运算等。 |

## DOCX 快速示例

```go
package main

import (
	"log"

	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	"github.com/scoming-dev/tools/docx"
)

func main() {
	markdown := []byte(`# 第一章

正文内容。

1. 一级列表
    * 二级列表会按报告序号渲染

行内公式 $a^2+b^2=c^2$

$$
E = mc^2
$$
`)

	parseOptions := parse.NewOptions()
	parseOptions.HTMLTag2TextMark = true
	tree := parse.Parse("", markdown, parseOptions)

	renderOptions := render.NewOptions()
	renderOptions.RenderListStyle = true

	renderer := docx.NewDocxRenderer(tree, renderOptions, docx.HeadingStyleDefault)
	renderer.Render()

	if err := renderer.Save("tmp/example.docx"); err != nil {
		log.Fatal(err)
	}
}
```

### DOCX 说明

- 列表使用 DOCX 原生 numbering，不把编号写成普通文本。
- 一般报告正文列表层级按 `（一）`、`1.`、`（1）`、`①`、`A.`、`a.`、`1）` 循环。
- 数学公式输出为 Office Math Markup Language，Office 和 WPS 可识别，不使用图片。
- SVG 图片按 SVG 媒体嵌入，不调用外部转换命令。
- 测试生成的 DOCX 文件会输出到 `docx/tmp/`，该目录应保持在 git 忽略列表中。

## OSS 快速示例

```go
package main

import (
	"context"
	"log"

	"github.com/scoming-dev/tools/oss"
)

func main() {
	client, err := oss.NewOSSClient(oss.OssConfig{
		Type: "minio",
		Minio: &oss.MinioConfig{
			Endpoint:        "localhost:9000",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			BucketName:      "example",
			UseSSL:          false,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	url, objectName, err := client.UploadFileWithPrefix(context.Background(), "tmp/example.docx", "doc")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("uploaded: %s %s", objectName, url)
}
```

更多 OSS 配置可参考 [oss/README.md](oss/README.md)。

## 验证码示例

验证码包不绑定具体 Redis 客户端，只需要传入实现 `captcha.Store` 的缓存对象。

```go
store := captcha.NewMemoryStore()
capt := captcha.NewCaptcha(ctx, store)
err, payload := capt.SlideCapt()
if err != nil {
	return err
}
_ = payload
```

生产环境可以用 Redis、Memcached 或项目已有缓存实现 `captcha.Store`，工具库本身不依赖具体框架；需要调整过期时间时可传入 `captcha.WithTTL(...)`。

## OnlyOffice 示例

```go
cfg, err := onlyoffice.BuildConfig(&onlyoffice.Config{
	DocumentId:   "doc-1",
	DocumentName: "report.docx",
	DocumentKey:  "report-doc-1",
	CallbackUrl:  "https://example.com/app/base/onlyoffice/callback",
	Secret:       "your-secret",
})
if err != nil {
	return err
}
_ = cfg
```

## 开发约定

- 代码格式化使用 `gofmt`。
- 修改后建议运行 `go test ./...`。
- 不提交 `tmp/`、`docx/tmp/`、测试输出文档、服务密钥和本地配置文件。
- 涉及第三方服务的测试应使用临时凭据或 mock，避免把真实密钥写入仓库。

