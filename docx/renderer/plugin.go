package renderer

import "github.com/scoming-dev/tools/docx/document"

// HTMLBlockPlugin renders custom HTML block tags into the active DOCX document.
type HTMLBlockPlugin interface {
	Type() string
	RenderHTML(renderer *DocxRenderer, content string) error
}

// CoverPlugin renders optional front matter before the Markdown body.
type CoverPlugin interface {
	RenderCover(renderer *DocxRenderer) error
}

// CoverFunc adapts a function into a CoverPlugin.
type CoverFunc func(renderer *DocxRenderer) error

// RenderCover calls the wrapped cover function.
func (f CoverFunc) RenderCover(renderer *DocxRenderer) error {
	return f(renderer)
}

// RendererOption customizes a DocxRenderer at construction time.
type RendererOption func(*DocxRenderer)

// WithConfig applies renderer configuration, filling unset fields from defaults.
func WithConfig(config Config) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.config = config.WithDefaults()
	}
}

// WithHeadingStyle applies one of the supported heading rendering styles.
func WithHeadingStyle(style HeadingStyle) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.headingStyle = normalizeHeadingStyle(int(style))
	}
}

// WithHTMLBlockPlugins registers custom HTML block renderers.
func WithHTMLBlockPlugins(plugins ...HTMLBlockPlugin) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.RegisterHTMLBlockPlugins(plugins...)
	}
}

// WithCover configures a cover renderer.
func WithCover(cover CoverPlugin) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.cover = cover
	}
}

// WithCoverFunc configures a function-backed cover renderer.
func WithCoverFunc(render func(renderer *DocxRenderer) error) RendererOption {
	if render == nil {
		return func(renderer *DocxRenderer) {
			renderer.cover = nil
		}
	}
	return WithCover(CoverFunc(render))
}

// RegisterHTMLBlockPlugins registers multiple HTML block plugins.
func (r *DocxRenderer) RegisterHTMLBlockPlugins(plugins ...HTMLBlockPlugin) {
	for _, plugin := range plugins {
		r.RegisterHTMLBlockPlugin(plugin)
	}
}

// RegisterHTMLBlockPlugin registers one HTML block plugin by its Type value.
func (r *DocxRenderer) RegisterHTMLBlockPlugin(plugin HTMLBlockPlugin) {
	if plugin == nil || plugin.Type() == "" {
		return
	}
	if r.htmlBlockPlugins == nil {
		r.htmlBlockPlugins = make(map[string]HTMLBlockPlugin)
	}
	r.htmlBlockPlugins[plugin.Type()] = plugin
}

// Document exposes the in-progress document for plugins.
func (r *DocxRenderer) Document() *document.Document {
	return r.doc
}

// Config returns the renderer's effective configuration.
func (r *DocxRenderer) Config() Config {
	return r.config
}

// BuildTocTree builds a table-of-contents tree from the Markdown headings.
func (r *DocxRenderer) BuildTocTree() *TocTree {
	return r.buildTocTree()
}

// RenderTableOfContents renders a previously built table-of-contents tree.
func (r *DocxRenderer) RenderTableOfContents(tree *TocTree) {
	r.renderTocInCover(tree)
}

// AddTempFile marks a generated file for cleanup after Save.
func (r *DocxRenderer) AddTempFile(path string) {
	if path == "" {
		return
	}
	r.files = append(r.files, path)
}

// SetParagraphSpacing applies the renderer's default paragraph spacing.
func (r *DocxRenderer) SetParagraphSpacing(para *document.Paragraph) {
	r.setParagraphSpacing(para)
}

// SetFontFamily applies a font family to run properties.
func (r *DocxRenderer) SetFontFamily(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
}

// SetFontSize applies a point size to run properties.
func (r *DocxRenderer) SetFontSize(props *document.RunProperties, fontSize int) {
	r.setFontSize(props, fontSize)
}
