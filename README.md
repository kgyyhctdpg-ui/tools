# tools

`github.com/scoming-dev/tools` 是一组 Go 业务工具包，当前包含 DOCX 生成、文档转 Markdown、对象存储、验证码、OnlyOffice 配置、HTTP 请求、认证授权、OAuth、权限码、缓存抽象、文件工具、JSON、字符串、网络、业务校验、图片处理、加密签名、Excel、金额、短信、唯一 ID 和 Casbin RBAC。

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
| `markdown` | DOCX、XLSX、PPTX、PDF、EPUB、HTML、邮件、压缩包、图片和文本格式转 Markdown。 |
| `oss` | 统一对象存储兼容入口，保留 MinIO、阿里云 OSS、华为 OBS 旧调用方式。 |
| `oss/miniox` | 独立 MinIO 适配，不引入其他云厂商 SDK。 |
| `oss/aliyunx` | 独立阿里云 OSS 适配，不引入 MinIO 或华为 OBS SDK。 |
| `oss/huaweix` | 独立华为 OBS 适配，不引入 MinIO 或阿里云 OSS SDK。 |
| `captcha` | 点击、滑块、旋转验证码生成和缓存校验。 |
| `onlyoffice` | OnlyOffice 文档类型识别、JWT、文档配置构建。 |
| `authx` | 密码哈希、Bearer 解析、JWT、API Key、HTTP 鉴权中间件、Refresh Token 和基于缓存的 Session 管理。 |
| `oauthx` | OAuth2 授权 URL、PKCE、state 防重放、授权码/刷新 token、常见 provider 预设和用户信息获取。 |
| `cache` | 通用缓存接口和并发安全的内存实现，支持 TTL、SetNX、Remember。 |
| `filex` | 文件判断、大小格式化、MIME、hash、base64、HTTP 下载和本地路径准备。 |
| `jsonx` | JSON 编码/解码、压缩、美化、合法性判断、深拷贝、map 转换和点路径读写。 |
| `stringx` | 字符串截取、补齐、脱敏、命名风格转换、去重、分割清洗和安全随机字符串。 |
| `networkx` | 客户端 IP、IPv4/IPv6、公网/内网、CIDR、HostPort、本机 IP、可用端口和 TCP 探测。 |
| `validator` | 手机号、邮箱、URL、IP、身份证、统一社会信用代码、银行卡、金额、中文姓名、密码强度校验。 |
| `httpx` | HTTP 客户端，支持 JSON、query、默认 header、重试、状态错误、文件上传下载。 |
| `imagex` | 图片尺寸、MIME、base64/data URI、JPEG 压缩、等比缩放、PNG/JPEG/GIF 保存。 |
| `crypto` | MD5、SHA256、HMAC-SHA256、安全随机串、AES-GCM/CBC、RSA-SHA256 签名验签。 |
| `excel` | XLSX 快速导出和读取，支持 map/slice 导出、表头样式、冻结表头、筛选。 |
| `money` | 基于 decimal 的金额加减乘除、格式化、元分转换、人民币大写。 |
| `sms` | 聚合短信接口，内置阿里云、腾讯云、云片、Submail、聚合数据、螺丝帽、创蓝和通用 HTTP 适配。 |
| `uniqueid` | 业务单号和 Sonyflake ID 生成。 |
| `permissionx` | resource:action 权限码生成、解析、通配符匹配、批量判断和去重。 |
| `casbinx` | Casbin RBAC/租户域 RBAC 模型构建和常用权限操作封装，不默认绑定数据库。 |
| `casbinx/gormx` | 基于 GORM Adapter 的 Casbin 策略持久化初始化；数据库驱动由业务项目自行导入。 |

## Markdown 转换快速示例

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/scoming-dev/tools/markdown"
)

func main() {
	converter := markdown.New(
		markdown.WithAssetsDirectory("report_files", "report_files"),
	)
	result, err := converter.Convert(context.Background(), "report.docx")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Markdown)
}
```

`markdown.New()` 会继续启用全部内置格式，兼容原有行为。只需要部分格式时，可以使用轻量构造器按组启用转换器：

```go
converter := markdown.NewCore(
	markdown.WithTextConverters(),
	markdown.WithOfficeConverters(),
)
```

可选分组包括 `WithTextConverters`、`WithOfficeConverters`、`WithArchiveConverters`、`WithImageConverter` 和 `WithPDFConverter`。也可以在创建后调用对应的 `Register...` 方法；重复注册不会产生重复转换器。`NewEmpty` 是 `NewCore` 的别名，`NewWithBuiltins` 是 `New` 的显式别名。

DOCX 转换会将原生 OMML 公式输出为行内 `$...$` 或块级 `$$...$$` LaTeX。`WithAssetsDirectory` 会把 DrawingML、VML、SVG 和 MathType/OLE 预览图保存到独立目录，Markdown 中只保留相对文件链接；需要上传对象存储时，也可以通过 `WithImageHandler` 返回最终 URL：

```go
converter := markdown.New(markdown.WithImageHandler(
	func(ctx context.Context, image markdown.Image) (string, error) {
		return upload(ctx, image.Name, image.MIMEType, image.Data)
	},
))
```

表格默认输出 HTML。合并列使用 `colspan`，合并行使用 `rowspan`，两种合并同时存在时会分别保留。需要输出简单 Markdown 表格时，可以使用 `markdown.WithTableFormat(markdown.TableFormatMarkdown)`；含合并单元格的表格仍会使用 HTML，避免结构丢失。

PDF 优先使用自部署 MinerU。服务端应启动 MinerU HTTP API 并提供同步 `POST /file_parse` 接口。客户端会上传 `files` multipart 文件，启用公式和表格识别，并请求返回 Markdown 与 Base64 图片。MinerU 的 `md_content` 会原样返回，不会在 Go 客户端重新排版或转换表格；服务端返回的图片仍通过 `ImageHandler` 外置、按内容去重并替换附件链接。

```go
minerUHandler, err := markdown.NewMinerUPDFHandler(markdown.MinerUConfig{
	BaseURL: "http://127.0.0.1:8000",
	Backend: "pipeline",
	Language: "ch",
})
if err != nil {
	log.Fatal(err)
}
converter := markdown.New(
	markdown.WithPDFHandler(minerUHandler),
	markdown.WithAssetsDirectory("report_files", "report_files"),
)
```

`WithPDFHandler` 用于配置所有 PDF 的 MinerU 转换。未配置处理器时，PDF 会先用 go-fitz 生成 HTML，再交给 Lute 转成 Markdown，并返回 warning；这个降级路径只保留正文文本，表格、公式、图片、版面和扫描件 OCR 仍可能缺失。MinerU 请求失败时直接返回错误，没有再额外走本地降级。

命令行使用：

```bash
go run ./cmd/markdown -i report.docx
go run ./cmd/markdown -i report.docx -o report.md
go run ./cmd/markdown -i report.pdf -o report.md --mineru-url http://127.0.0.1:8000
make build-markdown-cli-all
```

MinerU CLI 还支持 `--mineru-endpoint`、`--mineru-token`、`--mineru-backend`、`--mineru-parse-method`、`--mineru-language` 和 `--mineru-timeout`。默认端点为 `/file_parse`，后端为 `pipeline`，解析方法为 `auto`，语言为 `ch`，超时为 20 分钟。

CLI 使用输入文档文件名（basename，包含扩展名）的 MD5 作为附件目录名。使用 `-o result.md` 时，目录创建在输出文件同级；不传 `-o` 时创建在当前目录。Markdown 只保存 `<文件名MD5>/image.png` 形式的相对路径。内容相同的图片即使来源名称不同也只保存一份并复用链接；同名且内容不同的图片会自动添加数字后缀，不会互相覆盖。

## DOCX 快速示例

### DOCX CLI

仓库提供了一个 Markdown 转 DOCX 的命令行工具，可以通过 Makefile 构建多平台版本。

构建多平台版本：

```bash
make build-docx-cli-all
```

CLI 默认使用 `-trimpath -ldflags="-s -w"` 构建，去掉本地路径、符号表和调试信息以减小二进制体积；表格插件仍然保留。

默认会生成：

```text
bin/docx-darwin-amd64
bin/docx-darwin-arm64
bin/docx-linux-amd64
bin/docx-linux-arm64
bin/docx-windows-amd64.exe
bin/docx-windows-arm64.exe
```

选择当前平台的二进制直接执行：

```bash
bin/docx-darwin-arm64 -input report.md -output tmp/report.docx -cover none -heading-style default
```

不传任何参数时，CLI 会直接输出帮助信息。

如果 Markdown 中使用通用表格标签，例如 `<s-tag type="excel" url="data.xlsx" name="Sheet1"></s-tag>`，可以启用对应标签插件：

```bash
bin/docx-darwin-arm64 -input report.md -output tmp/report.docx -table-tags excel
```

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
