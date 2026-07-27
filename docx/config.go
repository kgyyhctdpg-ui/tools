package docx

import (
	"github.com/scoming-dev/tools/docx/measurement"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

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

type FontConfig struct {
	Content string
	Title   string
	Latin   string
}

type TextConfig struct {
	ContentSize  int
	SubTitleSize int
	TitleSize    int
}

type ParagraphConfig struct {
	LineHeightMultiplier float64
	FirstLineIndent      measurement.Distance
	Alignment            wml.ST_Jc
}

type HeadingConfig struct {
	Sizes             []int
	LevelOneAlignment wml.ST_Jc
}

type HeaderFooterConfig struct {
	FontSize     int
	Alignment    wml.ST_Jc
	FooterSuffix string
}

type TOCConfig struct {
	TitleFontSize      int
	ItemFontSize       int
	TitleSpacingBefore measurement.Distance
	TitleSpacingAfter  measurement.Distance
	IndentPerDepth     measurement.Distance
}

type TableConfig struct {
	HeaderFontSize       int
	CellFontSize         int
	CellCharacterSpacing measurement.Distance
	BorderColor          string
	BorderWidth          measurement.Distance
	AfterLineSpacing     measurement.Distance
}

type MediaConfig struct {
	SpacingBefore measurement.Distance
	SpacingAfter  measurement.Distance
	Alignment     wml.ST_Jc
}

type CoverConfig struct {
	Report           ThreePartCoverConfig
	SpecialDebt      ThreePartCoverConfig
	Embodiment       ThreePartCoverConfig
	DefaultFont      string
	DefaultAlignment wml.ST_Jc
}

type ThreePartCoverConfig struct {
	TitleSize         int
	SubtitleSize      int
	DetailSize        int
	TitleSpacing      measurement.Distance
	SubtitleSpacing   measurement.Distance
	DetailAlignment   wml.ST_Jc
	DetailStartIndent measurement.Distance
}

type PageConfig struct {
	MarginMM float64
}

type ListConfig struct {
	FirstLevelFontSize  int
	SecondLevelFontSize int
	ThirdLevelFontSize  int
	ThirdLevelFont      string
	FirstLineIndent     measurement.Distance
	MarkerSuffix        string
}

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
			FirstLineIndent:      measurement.Distance(28),
			Alignment:            wml.ST_JcBoth,
		},
		Heading: HeadingConfig{
			Sizes:             []int{24, 22, 20, 18, 16, 14},
			LevelOneAlignment: wml.ST_JcCenter,
		},
		HeaderFooter: HeaderFooterConfig{
			FontSize:     10,
			Alignment:    wml.ST_JcCenter,
			FooterSuffix: "                                        此内容由(DP-Ai)生成",
		},
		TOC: TOCConfig{
			TitleFontSize:      22,
			ItemFontSize:       14,
			TitleSpacingBefore: measurement.Distance(12),
			TitleSpacingAfter:  measurement.Distance(18),
			IndentPerDepth:     measurement.Distance(24),
		},
		Table: TableConfig{
			HeaderFontSize:       10,
			CellFontSize:         8,
			CellCharacterSpacing: measurement.Distance(0),
			BorderColor:          "#000000",
			BorderWidth:          measurement.Point * 0.5,
			AfterLineSpacing:     measurement.Distance(14),
		},
		Image: MediaConfig{
			SpacingBefore: measurement.Distance(6),
			SpacingAfter:  measurement.Distance(6),
			Alignment:     wml.ST_JcCenter,
		},
		Math: MediaConfig{
			SpacingBefore: measurement.Distance(6),
			SpacingAfter:  measurement.Distance(6),
			Alignment:     wml.ST_JcCenter,
		},
		Cover: CoverConfig{
			Report: ThreePartCoverConfig{
				TitleSize:       36,
				SubtitleSize:    24,
				DetailSize:      16,
				TitleSpacing:    measurement.Distance(240),
				SubtitleSpacing: measurement.Distance(240),
				DetailAlignment: wml.ST_JcCenter,
			},
			SpecialDebt: ThreePartCoverConfig{
				TitleSize:       36,
				SubtitleSize:    24,
				DetailSize:      16,
				TitleSpacing:    measurement.Distance(240),
				SubtitleSpacing: measurement.Distance(240),
				DetailAlignment: wml.ST_JcCenter,
			},
			Embodiment: ThreePartCoverConfig{
				TitleSize:         20,
				SubtitleSize:      26,
				DetailSize:        16,
				TitleSpacing:      measurement.Distance(150),
				SubtitleSpacing:   measurement.Distance(150),
				DetailAlignment:   wml.ST_JcLeft,
				DetailStartIndent: measurement.Distance(24) * 5,
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
			FirstLineIndent:     measurement.Distance(28),
			MarkerSuffix:        "space",
		},
	}
}

func (c Config) withDefaults() Config {
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
