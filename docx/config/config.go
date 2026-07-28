package config

import "github.com/scoming-dev/tools/docx/schema/soo/wml"

// Config contains all renderer-level typography, spacing, page, and plugin preset settings.
type Config struct {
	Fonts        FontConfig
	Text         TextConfig
	Paragraph    ParagraphConfig
	Heading      HeadingConfig
	HeaderFooter HeaderFooterConfig
	TOC          TOCConfig
	Table        TableConfig
	Image        MediaConfig
	Math         MediaConfig
	Cover        CoverConfig
	Page         PageConfig
	List         ListConfig
}

// FontConfig defines the document font families used by generated runs.
type FontConfig struct {
	Content string
	Title   string
	Latin   string
}

// TextConfig defines point sizes for the main document text tiers.
type TextConfig struct {
	ContentSize  int
	SubTitleSize int
	TitleSize    int
}

// ParagraphConfig defines default paragraph layout for normal content.
type ParagraphConfig struct {
	LineHeightMultiplier float64
	FirstLineIndent      float64
	Alignment            wml.ST_Jc
}

// HeadingConfig defines heading font sizes and level-one alignment.
type HeadingConfig struct {
	Sizes             []int
	LevelOneAlignment wml.ST_Jc
}

// HeaderFooterConfig defines generated header and footer text styling.
type HeaderFooterConfig struct {
	FontSize     int
	Alignment    wml.ST_Jc
	FooterSuffix string
}

// TOCConfig defines table-of-contents typography and indentation.
type TOCConfig struct {
	TitleFontSize      int
	ItemFontSize       int
	TitleSpacingBefore float64
	TitleSpacingAfter  float64
	IndentPerDepth     float64
}

// TableConfig defines default table border, spacing, and text styling.
type TableConfig struct {
	HeaderFontSize       int
	CellFontSize         int
	CellCharacterSpacing float64
	BorderColor          string
	BorderWidth          float64
	AfterLineSpacing     float64
}

// MediaConfig defines paragraph layout for embedded media-like blocks.
type MediaConfig struct {
	SpacingBefore float64
	SpacingAfter  float64
	Alignment     wml.ST_Jc
}

// CoverConfig defines defaults used by cover page plugins.
type CoverConfig struct {
	Report           ThreePartCoverConfig
	SpecialDebt      ThreePartCoverConfig
	Embodiment       ThreePartCoverConfig
	DefaultFont      string
	DefaultAlignment wml.ST_Jc
}

// ThreePartCoverConfig defines title, subtitle, and detail styling for preset covers.
type ThreePartCoverConfig struct {
	TitleSize         int
	SubtitleSize      int
	DetailSize        int
	TitleSpacing      float64
	SubtitleSpacing   float64
	DetailAlignment   wml.ST_Jc
	DetailStartIndent float64
}

// PageConfig defines page-level defaults.
type PageConfig struct {
	MarginMM float64
}

// ListConfig defines numbering marker and nested-list typography.
type ListConfig struct {
	FirstLevelFontSize  int
	SecondLevelFontSize int
	ThirdLevelFontSize  int
	ThirdLevelFont      string
	FirstLineIndent     float64
	MarkerSuffix        string
}

// DefaultConfig returns the renderer defaults used when options leave fields unset.
func DefaultConfig() Config {
	return Config{
		Fonts: FontConfig{
			Content: "仿宋",
			Title:   "黑体",
			Latin:   "Times New Roman",
		},
		Text: TextConfig{
			ContentSize:  14,
			SubTitleSize: 16,
			TitleSize:    22,
		},
		Paragraph: ParagraphConfig{
			LineHeightMultiplier: 1.3,
			FirstLineIndent:      float64(28),
			Alignment:            wml.ST_JcBoth,
		},
		Heading: HeadingConfig{
			Sizes:             []int{24, 22, 20, 18, 16, 14},
			LevelOneAlignment: wml.ST_JcCenter,
		},
		HeaderFooter: HeaderFooterConfig{
			FontSize:     10,
			Alignment:    wml.ST_JcCenter,
			FooterSuffix: "",
		},
		TOC: TOCConfig{
			TitleFontSize:      22,
			ItemFontSize:       14,
			TitleSpacingBefore: float64(12),
			TitleSpacingAfter:  float64(18),
			IndentPerDepth:     float64(24),
		},
		Table: TableConfig{
			HeaderFontSize:       10,
			CellFontSize:         8,
			CellCharacterSpacing: float64(0),
			BorderColor:          "#000000",
			BorderWidth:          0.5,
			AfterLineSpacing:     float64(14),
		},
		Image: MediaConfig{
			SpacingBefore: float64(6),
			SpacingAfter:  float64(6),
			Alignment:     wml.ST_JcCenter,
		},
		Math: MediaConfig{
			SpacingBefore: float64(6),
			SpacingAfter:  float64(6),
			Alignment:     wml.ST_JcCenter,
		},
		Cover: CoverConfig{
			Report: ThreePartCoverConfig{
				TitleSize:       36,
				SubtitleSize:    24,
				DetailSize:      16,
				TitleSpacing:    float64(240),
				SubtitleSpacing: float64(240),
				DetailAlignment: wml.ST_JcCenter,
			},
			SpecialDebt: ThreePartCoverConfig{
				TitleSize:       36,
				SubtitleSize:    24,
				DetailSize:      16,
				TitleSpacing:    float64(240),
				SubtitleSpacing: float64(240),
				DetailAlignment: wml.ST_JcCenter,
			},
			Embodiment: ThreePartCoverConfig{
				TitleSize:         20,
				SubtitleSize:      26,
				DetailSize:        16,
				TitleSpacing:      float64(150),
				SubtitleSpacing:   float64(150),
				DetailAlignment:   wml.ST_JcLeft,
				DetailStartIndent: float64(24) * 5,
			},
			DefaultFont:      "黑体",
			DefaultAlignment: wml.ST_JcCenter,
		},
		Page: PageConfig{
			MarginMM: 60,
		},
		List: ListConfig{
			FirstLevelFontSize:  10,
			SecondLevelFontSize: 10,
			ThirdLevelFontSize:  14,
			ThirdLevelFont:      "Times New Roman",
			FirstLineIndent:     float64(28),
			MarkerSuffix:        "space",
		},
	}
}

// WithDefaults fills unset fields from DefaultConfig.
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.Fonts.Content == "" {
		c.Fonts.Content = defaults.Fonts.Content
	}
	if c.Fonts.Title == "" {
		c.Fonts.Title = defaults.Fonts.Title
	}
	if c.Fonts.Latin == "" {
		c.Fonts.Latin = defaults.Fonts.Latin
	}
	if c.Text.ContentSize == 0 {
		c.Text.ContentSize = defaults.Text.ContentSize
	}
	if c.Text.SubTitleSize == 0 {
		c.Text.SubTitleSize = defaults.Text.SubTitleSize
	}
	if c.Text.TitleSize == 0 {
		c.Text.TitleSize = defaults.Text.TitleSize
	}
	if c.Paragraph.LineHeightMultiplier == 0 {
		c.Paragraph.LineHeightMultiplier = defaults.Paragraph.LineHeightMultiplier
	}
	if c.Paragraph.FirstLineIndent == 0 {
		c.Paragraph.FirstLineIndent = defaults.Paragraph.FirstLineIndent
	}
	if c.Paragraph.Alignment == 0 {
		c.Paragraph.Alignment = defaults.Paragraph.Alignment
	}
	if len(c.Heading.Sizes) == 0 {
		c.Heading.Sizes = append([]int(nil), defaults.Heading.Sizes...)
	}
	if c.Heading.LevelOneAlignment == 0 {
		c.Heading.LevelOneAlignment = defaults.Heading.LevelOneAlignment
	}
	if c.HeaderFooter.FontSize == 0 {
		c.HeaderFooter.FontSize = defaults.HeaderFooter.FontSize
	}
	if c.HeaderFooter.Alignment == 0 {
		c.HeaderFooter.Alignment = defaults.HeaderFooter.Alignment
	}
	if c.HeaderFooter.FooterSuffix == "" {
		c.HeaderFooter.FooterSuffix = defaults.HeaderFooter.FooterSuffix
	}
	if c.TOC.TitleFontSize == 0 {
		c.TOC.TitleFontSize = defaults.TOC.TitleFontSize
	}
	if c.TOC.ItemFontSize == 0 {
		c.TOC.ItemFontSize = defaults.TOC.ItemFontSize
	}
	if c.TOC.TitleSpacingBefore == 0 {
		c.TOC.TitleSpacingBefore = defaults.TOC.TitleSpacingBefore
	}
	if c.TOC.TitleSpacingAfter == 0 {
		c.TOC.TitleSpacingAfter = defaults.TOC.TitleSpacingAfter
	}
	if c.TOC.IndentPerDepth == 0 {
		c.TOC.IndentPerDepth = defaults.TOC.IndentPerDepth
	}
	if c.Table.HeaderFontSize == 0 {
		c.Table.HeaderFontSize = defaults.Table.HeaderFontSize
	}
	if c.Table.CellFontSize == 0 {
		c.Table.CellFontSize = defaults.Table.CellFontSize
	}
	if c.Table.BorderColor == "" {
		c.Table.BorderColor = defaults.Table.BorderColor
	}
	if c.Table.BorderWidth == 0 {
		c.Table.BorderWidth = defaults.Table.BorderWidth
	}
	if c.Table.AfterLineSpacing == 0 {
		c.Table.AfterLineSpacing = defaults.Table.AfterLineSpacing
	}
	if c.Image.SpacingBefore == 0 {
		c.Image.SpacingBefore = defaults.Image.SpacingBefore
	}
	if c.Image.SpacingAfter == 0 {
		c.Image.SpacingAfter = defaults.Image.SpacingAfter
	}
	if c.Image.Alignment == 0 {
		c.Image.Alignment = defaults.Image.Alignment
	}
	if c.Math.SpacingBefore == 0 {
		c.Math.SpacingBefore = defaults.Math.SpacingBefore
	}
	if c.Math.SpacingAfter == 0 {
		c.Math.SpacingAfter = defaults.Math.SpacingAfter
	}
	if c.Math.Alignment == 0 {
		c.Math.Alignment = defaults.Math.Alignment
	}
	if c.Cover.DefaultFont == "" {
		c.Cover.DefaultFont = defaults.Cover.DefaultFont
	}
	if c.Cover.DefaultAlignment == 0 {
		c.Cover.DefaultAlignment = defaults.Cover.DefaultAlignment
	}
	c.Cover.Report = c.Cover.Report.withDefaults(defaults.Cover.Report)
	c.Cover.SpecialDebt = c.Cover.SpecialDebt.withDefaults(defaults.Cover.SpecialDebt)
	c.Cover.Embodiment = c.Cover.Embodiment.withDefaults(defaults.Cover.Embodiment)
	if c.Page.MarginMM == 0 {
		c.Page.MarginMM = defaults.Page.MarginMM
	}
	if c.List.FirstLevelFontSize == 0 {
		c.List.FirstLevelFontSize = defaults.List.FirstLevelFontSize
	}
	if c.List.SecondLevelFontSize == 0 {
		c.List.SecondLevelFontSize = defaults.List.SecondLevelFontSize
	}
	if c.List.ThirdLevelFontSize == 0 {
		c.List.ThirdLevelFontSize = defaults.List.ThirdLevelFontSize
	}
	if c.List.ThirdLevelFont == "" {
		c.List.ThirdLevelFont = defaults.List.ThirdLevelFont
	}
	if c.List.FirstLineIndent == 0 {
		c.List.FirstLineIndent = defaults.List.FirstLineIndent
	}
	if c.List.MarkerSuffix == "" {
		c.List.MarkerSuffix = defaults.List.MarkerSuffix
	}
	return c
}

func (c ThreePartCoverConfig) withDefaults(defaults ThreePartCoverConfig) ThreePartCoverConfig {
	if c.TitleSize == 0 {
		c.TitleSize = defaults.TitleSize
	}
	if c.SubtitleSize == 0 {
		c.SubtitleSize = defaults.SubtitleSize
	}
	if c.DetailSize == 0 {
		c.DetailSize = defaults.DetailSize
	}
	if c.TitleSpacing == 0 {
		c.TitleSpacing = defaults.TitleSpacing
	}
	if c.SubtitleSpacing == 0 {
		c.SubtitleSpacing = defaults.SubtitleSpacing
	}
	if c.DetailStartIndent == 0 {
		c.DetailStartIndent = defaults.DetailStartIndent
	}
	if c.DetailAlignment == 0 {
		c.DetailAlignment = defaults.DetailAlignment
	}
	return c
}
