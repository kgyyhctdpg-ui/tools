package docx

import "github.com/scoming-dev/tools/docx/document"

type HTMLBlockPlugin interface {
	Type() string
	RenderHTML(renderer *DocxRenderer, content string) error
}

type CoverPlugin interface {
	RenderCover(renderer *DocxRenderer) error
}

type CoverFunc func(renderer *DocxRenderer) error

func (f CoverFunc) RenderCover(renderer *DocxRenderer) error {
	return f(renderer)
}

type RendererOption func(*DocxRenderer)

func WithConfig(config Config) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.config = config.withDefaults()
	}
}

func WithHTMLBlockPlugins(plugins ...HTMLBlockPlugin) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.RegisterHTMLBlockPlugins(plugins...)
	}
}

func WithCover(cover CoverPlugin) RendererOption {
	return func(renderer *DocxRenderer) {
		renderer.cover = cover
	}
}

func WithCoverFunc(render func(renderer *DocxRenderer) error) RendererOption {
	if render == nil {
		return func(renderer *DocxRenderer) {
			renderer.cover = nil
		}
	}
	return WithCover(CoverFunc(render))
}

func (r *DocxRenderer) RegisterHTMLBlockPlugins(plugins ...HTMLBlockPlugin) {
	for _, plugin := range plugins {
		r.RegisterHTMLBlockPlugin(plugin)
	}
}

func (r *DocxRenderer) RegisterHTMLBlockPlugin(plugin HTMLBlockPlugin) {
	if plugin == nil || plugin.Type() == "" {
		return
	}
	if r.htmlBlockPlugins == nil {
		r.htmlBlockPlugins = make(map[string]HTMLBlockPlugin)
	}
	r.htmlBlockPlugins[plugin.Type()] = plugin
}

func (r *DocxRenderer) Document() *document.Document {
	return r.doc
}

func (r *DocxRenderer) Config() Config {
	return r.config
}

func (r *DocxRenderer) BuildTocTree() *TocTree {
	return r.buildTocTree()
}

func (r *DocxRenderer) RenderTableOfContents(tree *TocTree) {
	r.renderTocInCover(tree)
}

func (r *DocxRenderer) AddTempFile(path string) {
	if path == "" {
		return
	}
	r.files = append(r.files, path)
}

func (r *DocxRenderer) SetParagraphSpacing(para *document.Paragraph) {
	r.setParagraphSpacing(para)
}

func (r *DocxRenderer) SetFontFamily(props *document.RunProperties, fontFamily string) {
	r.setFontFamily(props, fontFamily)
}

func (r *DocxRenderer) SetFontSize(props *document.RunProperties, fontSize int) {
	r.setFontSzie(props, fontSize)
}
