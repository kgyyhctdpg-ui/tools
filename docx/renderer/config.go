package renderer

import docxconfig "github.com/scoming-dev/tools/docx/config"

type Config = docxconfig.Config
type FontConfig = docxconfig.FontConfig
type TextConfig = docxconfig.TextConfig
type ParagraphConfig = docxconfig.ParagraphConfig
type HeadingConfig = docxconfig.HeadingConfig
type HeaderFooterConfig = docxconfig.HeaderFooterConfig
type TOCConfig = docxconfig.TOCConfig
type TableConfig = docxconfig.TableConfig
type MediaConfig = docxconfig.MediaConfig
type CoverConfig = docxconfig.CoverConfig
type ThreePartCoverConfig = docxconfig.ThreePartCoverConfig
type PageConfig = docxconfig.PageConfig
type ListConfig = docxconfig.ListConfig

func DefaultConfig() Config {
	return docxconfig.DefaultConfig()
}
