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

PDF 全部在本地解析，不依赖任何远程服务，也不做 OCR。引擎是 [MuPDF](https://mupdf.com/)，通过 [go-fitz](https://github.com/gen2brain/go-fitz) 调用并静态链接其预编译库。go-fitz 输出每页带绝对坐标、字号与粗斜体样式的 HTML（同一视觉行中的大间距会被拆成多个元素，表格单元格因此得以保留），本包据此重建 Markdown：

- 按基线聚类把同一视觉行的文本矩形合并成一行，列间距较大时保留为独立单元格，供表格重建使用；
- 正文按行位置合并换行，恢复自然段；中英混排会自动决定是否补空格；
- 依据字号与正文字号的比值识别标题（`#` / `##` / `###`）；识别 `1.` `1、` `（1）` `•` 等列表标记并转成 Markdown 列表；
- 依据 left 位置聚类恢复表格网格（列对齐、单元格较短的连续行），输出 HTML 或 Markdown 表格；
- 页面内嵌图片按 `pdf-page-<页>-image-<序号>` 命名，通过 `ImageHandler` 保存到本地目录并在 Markdown 中引用；图片会先压缩再落盘（见下）；
- 没有文字层也没有图片的页面（例如纯矢量页）会整页渲染成图片，按 `pdf-page-<页>` 命名后引用，不会静默丢失；
- 扫描件只保留图片引用，不识别文字。

引擎只在 `markdown/internal/pdf/engine.go` 里出现，版面重建代码只面对引擎无关的文本行结构。

版面重建的产物是**语义 HTML**（`<h1>`–`<h6>`、`<p>`、`<ul>/<ol>`、`<strong>/<em>`），最终由 [lute](https://github.com/88250/lute) 的 `HTML2Markdown` 序列化为 Markdown。这一步不能省：go-fitz 的 HTML 是**定位数据**（每个视觉片段一个带 `top/left` 的 `<p>`，没有 `<table>`、没有 `<h1>`），而 lute 只认语义标签、完全不读 `style`/`top`/`left`，直接把引擎的原始 HTML 交给它会把每张表格拍成一堆单格段落。语义 HTML 由本包重建，Markdown 序列化交给 lute，同时获得了它的 Markdown 转义能力（正文里的 `*`、`_`、`$`、`` ` ``、`~` 会被转义，不再被误当成行内标记）。表格是唯一例外：它由本包直接渲染并保留 `--table-format` 的选择，因为 Markdown 表格无法表达合并单元格。

lute 会在**加粗文本以标点开头或结尾**时插入零宽空格（如 `**\u200b“赈”\u200b**`）。这是必要的：在中文语境里 `发挥**“赈”**的功能` 不符合 CommonMark 的左翼定界规则，会原样渲染出星号，加上零宽空格才会真正加粗。代价是这些位置含有不可见字符；若你的下游需要逐字精确匹配，可以去掉它们，但上述加粗会随之失效。

页面是**并行**转换的：页与页互不依赖，默认按 CPU 数开 worker（上限 8），`PageConcurrency`（CLI 为 `--pdf-workers`）可调，设为 1 即完全串行。这里真正的瓶颈是图片：实测 61 页中，关掉图片只要 2.2s CPU，开着要 14.4s CPU，也就是**解码/降采样/编码占了约 85%**，正是这部分能随 worker 扩展。

MuPDF 的 context 不是线程安全的，所以每个 worker 各自 `open` 一个 `fitz.Document`，共享同一份只读输入缓冲（`fz_open_memory` 不拷贝数据，因此不会把 PDF 复制 N 份）。图片落盘回调仍保证**同一时刻只有一次调用**（内置 handler 自身也带锁），页面结果按原页码回填，所以**并行输出与串行逐字节一致**，已由 `TestConvertPDFPageWorkersMatchSequential` 与真实 300 页文档的对比锁定。

两点实测说明：并发会带来约 1.5–1.8 倍的 CPU 开销（多出来的 MuPDF context、调度与内存带宽争用），所以它换来的是**墙钟时间**而不是省 CPU；另外在 4 性能核 + 4 能效核的机器上，worker 超过性能核数量后收益基本消失（8 与 4 相近，16 明显更差），这正是上限取 8 的原因。若 `MaxImages` 或 `MaxRenderedPages` 被设置，转换会自动退回串行——这两个预算是按页码顺序消耗的，并行会让结果不可复现。

图片默认按 **JPEG（质量 85）** 输出，并把有效分辨率压到 **200 DPI**：内嵌图片按它在页面上的实际尺寸（点）与原始像素数算出有效 DPI，超过上限的先降采样再编码；整页渲染图则由 `RenderDPI` 直接控制渲染分辨率。这样一份 300–600 DPI 的扫描页通常从几十 MB 降到 1–2 MB。`RenderDPI`、`RenderFormat`、`JPEGQuality` 三项对两类图片共用；把 `RenderFormat` 设为 `png` 可回到无损输出（体积会明显变大），调大 `RenderDPI` 可保留更多细节。JPEG 没有透明通道，透明图章/蒙版会先合成到白底，不会变成黑块。

> **构建与许可**：go-fitz 通过 CGO 静态链接 MuPDF，本机编译无需额外安装，但 `make build-markdown-cli-all` 这类跨平台构建需要对应平台的 C 工具链，不再像 PDFium 的 WebAssembly 后端那样用纯 Go 直接交叉编译。MuPDF 采用 **AGPL-3.0**（也可向 Artifex 购买商业授权），与 PDFium 的 BSD-3-Clause 不同：如果本工具会对外分发、或作为网络服务对外提供，请先确认许可合规。若这两点不可接受，可改回 `go-pdfium` 后端或换用许可更宽松的解析库。


```go
converter := markdown.New(
	markdown.WithAssetsDirectory("report_files", "report_files"),
	markdown.WithPDFOptions(markdown.PDFOptions{
		RenderDPI:        200, // 整页渲染分辨率，同时是内嵌图片的有效分辨率上限
		RenderFormat:     markdown.PDFImageFormatJPEG, // 默认值；设为 PNG 则无损
		JPEGQuality:      85,
		MaxImagesPerPage: 32, // 超过则整页渲染，避免图纸页炸出上千张图
		FirstPage:        0,  // 0 表示从第一页开始
		LastPage:         0,  // 0 表示到最后一页
	}),
)
```

`PDFOptions` 还提供 `MaxImages`、`MaxRenderedPages`，以及 `DisablePageRenderFallback`、`DisableTableReconstruction`、`DisableHeadingDetection`、`DisableParagraphReflow` 等开关，便于对复杂版面回退到保守输出。转换时如果出现没有文字层的页面，会返回一条提示性 warning。

命令行使用：

```bash
go run ./cmd/markdown -i report.docx
go run ./cmd/markdown -i report.docx -o report.md
go run ./cmd/markdown -i report.pdf -o report.md --assets-dir images
go run ./cmd/markdown -i report.pdf -o report.md --pdf-dpi 300 --pdf-image-format jpeg
make build-markdown-cli-all
```

PDF 相关参数：`--pdf-dpi`、`--pdf-image-format`、`--pdf-jpeg-quality`、`--pdf-first-page`、`--pdf-last-page`、`--pdf-max-images-per-page`、`--pdf-max-images`、`--pdf-max-rendered-pages`、`--pdf-workers`（并行页数，0 = 按 CPU 数、上限 8），以及关闭特定重建能力的 `--pdf-no-tables`、`--pdf-no-headings`、`--pdf-no-reflow`、`--pdf-no-render-fallback`。图片压缩默认 `--pdf-dpi 200 --pdf-image-format jpeg --pdf-jpeg-quality 85`；例如 `--pdf-dpi 300 --pdf-image-format png` 可换取更大、更清晰的图片。

输入大小**默认不设上限**：转换本来就要把整份文档读进内存，加上限只会把一份合法的大文件变成报错。需要防护不受信任的上传时，用 `markdown.WithMaxInputSize`（命令行对应 `--max-input-size`，接受 `512MB`、`1GB` 这类写法或纯字节数）显式设置；设了上限后，超出时的报错会写明实际大小、上限值和来源。注意归档类输入另有独立限制（单个成员 32MB、512 个成员、4 层嵌套），不随此设置变化。

附件目录默认使用输入文档文件名（basename，包含扩展名）的 MD5。也可以显式指定：`--assets-dir` 决定图片落地目录（相对路径在给出 `-o` 时相对于输出文件所在目录解析），`--assets-prefix` 决定写进 Markdown 的链接前缀，默认与 `--assets-dir` 相同。不传 `-o` 时目录创建在当前目录。Markdown 只保存相对路径链接。内容相同的图片即使来源名称不同也只保存一份并复用链接；同名且内容不同的图片会自动添加数字后缀，不会互相覆盖。

### markdown 代码结构

`markdown` 包本身只保留公开 API，各格式实现按格式拆到 `internal/` 子包：

| 位置 | 职责 |
| --- | --- |
| `markdown/` | 公开 API：`MarkItDown` 引擎、`Option` 选项集、内置转换器注册（`converter.go`）、公开类型别名与 `NewFileImageHandler`/`WithAssetsDirectory`（`images.go`）。 |
| `markdown/internal/core/` | 转换模型与公共层：`Result`/`StreamInfo`/`Image`/`ImageHandler`/`TableFormat`/`Converter`、`Settings`（引擎与转换器共享的实时配置）、表格与代码块渲染、图片落盘与命名、`PDFOptions`、XML/zip 读取工具。 |
| `markdown/internal/ooxml/` | DOCX 与 PPTX 共用的合并单元格表格网格渲染。 |
| `markdown/internal/docx/` | DOCX（含 OMML 公式）转换。 |
| `markdown/internal/pptx/` | PPTX 转换（幻灯片、合并单元格表格）。 |
| `markdown/internal/xlsx/` | XLSX 转换（工作表、合并单元格）。 |
| `markdown/internal/pdf/` | 本地 PDF 转换。`engine.go` 是唯一接触 MuPDF（go-fitz）的地方（打开文档、把页面 HTML 解析成定位文本与图片、渲染整页）；`pdf_layout.go` 是引擎无关的版面重建（行合并、段落重排、标题识别、表格网格）；`pdf.go` 编排逐页转换与图片外置。 |
| `markdown/internal/text/` | 纯文本、Markdown、HTML、CSV/TSV、JSON/XML/YAML。 |
| `markdown/internal/email/` | EML 邮件及 MIME 附件。 |
| `markdown/internal/archive/` | ZIP 与 EPUB。 |
| `markdown/internal/imagefile/` | 独立图片文件。 |

公开 API 通过类型别名（`type Result = core.Result` 等）保持完全兼容，外部代码和自定义 `Converter` 无需改动。需要新增格式时，在对应 `internal/` 子包实现 `NewConverter(settings *core.Settings) core.Converter`，再到 `converter.go` 的注册表里挂上即可。

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
