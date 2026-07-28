// Package docx exposes the public Markdown-to-DOCX renderer API.
package docx

import (
	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	docxrenderer "github.com/scoming-dev/tools/docx/renderer"
)

// DocxRenderer converts a Lute Markdown tree into a DOCX document model.
type DocxRenderer = docxrenderer.DocxRenderer

// Config contains all renderer-level typography, spacing, page, and plugin preset settings.
type Config = docxrenderer.Config

// FontConfig defines the document font families used by generated runs.
type FontConfig = docxrenderer.FontConfig

// TextConfig defines point sizes for the main document text tiers.
type TextConfig = docxrenderer.TextConfig

// ParagraphConfig defines default paragraph layout for normal content.
type ParagraphConfig = docxrenderer.ParagraphConfig

// HeadingConfig defines heading font sizes and level-one alignment.
type HeadingConfig = docxrenderer.HeadingConfig

// HeadingStyle limits the supported visual modes for Markdown heading rendering.
type HeadingStyle = docxrenderer.HeadingStyle

const (
	// HeadingStyleDefault renders headings in place with configured heading sizes.
	HeadingStyleDefault = docxrenderer.HeadingStyleDefault
	// HeadingStyleLevelOneCenterPageBreak page-breaks before level-one headings and centers them.
	HeadingStyleLevelOneCenterPageBreak = docxrenderer.HeadingStyleLevelOneCenterPageBreak
	// HeadingStyleLevelOneLeftPageBreak page-breaks before level-one headings and left-aligns them.
	HeadingStyleLevelOneLeftPageBreak = docxrenderer.HeadingStyleLevelOneLeftPageBreak
)

// HeaderFooterConfig defines generated header and footer text styling.
type HeaderFooterConfig = docxrenderer.HeaderFooterConfig

// TOCConfig defines table-of-contents typography and indentation.
type TOCConfig = docxrenderer.TOCConfig

// TableConfig defines default table border, spacing, and text styling.
type TableConfig = docxrenderer.TableConfig

// MediaConfig defines paragraph layout for embedded media-like blocks.
type MediaConfig = docxrenderer.MediaConfig

// CoverConfig defines defaults used by cover page plugins.
type CoverConfig = docxrenderer.CoverConfig

// ThreePartCoverConfig defines title, subtitle, and detail styling for preset covers.
type ThreePartCoverConfig = docxrenderer.ThreePartCoverConfig

// PageConfig defines page-level defaults.
type PageConfig = docxrenderer.PageConfig

// ListConfig defines numbering marker and nested-list typography.
type ListConfig = docxrenderer.ListConfig

// HTMLBlockPlugin renders custom HTML block tags into the active DOCX document.
type HTMLBlockPlugin = docxrenderer.HTMLBlockPlugin

// CoverPlugin renders optional front matter before the Markdown body.
type CoverPlugin = docxrenderer.CoverPlugin

// CoverFunc adapts a function into a CoverPlugin.
type CoverFunc = docxrenderer.CoverFunc

// RendererOption customizes a DocxRenderer at construction time.
type RendererOption = docxrenderer.RendererOption

// TocNode represents a table-of-contents entry.
type TocNode = docxrenderer.TocNode

// TocTree contains the root table-of-contents entries.
type TocTree = docxrenderer.TocTree

// NewDocxRenderer creates a DOCX renderer.
func NewDocxRenderer(tree *parse.Tree, options *render.Options, headingStyle int, rendererOptions ...RendererOption) *DocxRenderer {
	return docxrenderer.NewDocxRenderer(tree, options, headingStyle, rendererOptions...)
}

// DefaultConfig returns renderer defaults used when options leave fields unset.
func DefaultConfig() Config {
	return docxrenderer.DefaultConfig()
}

// WithConfig applies renderer configuration, filling unset fields from defaults.
func WithConfig(config Config) RendererOption {
	return docxrenderer.WithConfig(config)
}

// WithHeadingStyle applies one of the supported heading rendering styles.
func WithHeadingStyle(style HeadingStyle) RendererOption {
	return docxrenderer.WithHeadingStyle(style)
}

// WithHTMLBlockPlugins registers custom HTML block renderers.
func WithHTMLBlockPlugins(plugins ...HTMLBlockPlugin) RendererOption {
	return docxrenderer.WithHTMLBlockPlugins(plugins...)
}

// WithCover configures a cover renderer.
func WithCover(cover CoverPlugin) RendererOption {
	return docxrenderer.WithCover(cover)
}

// WithCoverFunc configures a function-backed cover renderer.
func WithCoverFunc(render func(renderer *DocxRenderer) error) RendererOption {
	return docxrenderer.WithCoverFunc(render)
}
