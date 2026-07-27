package cover

import (
	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/measurement"
	"github.com/scoming-dev/tools/docx/schema/soo/wml"
)

type ThreePartCover struct {
	Preset            Preset
	Title             string
	Subtitle          string
	Details           []string
	TitleSize         int
	SubtitleSize      int
	DetailSize        int
	TitleSpacing      measurement.Distance
	SubtitleSpacing   measurement.Distance
	DetailAlignment   *wml.ST_Jc
	DetailStartIndent measurement.Distance
	VerticalSubtitle  bool
	IncludeTOC        bool
}

type Preset int

const (
	PresetCustom Preset = iota
	PresetReport
	PresetSpecialDebt
	PresetEmbodiment
)

func Report() docx.CoverPlugin {
	return ThreePartCover{
		Preset:     PresetReport,
		Title:      "报告",
		Subtitle:   "（**版）",
		Details:    []string{"编制单位：XXXX有限公司", "编制时间：XXXX年XX月"},
		IncludeTOC: true,
	}
}

func SpecialDebt() docx.CoverPlugin {
	return ThreePartCover{
		Preset:     PresetSpecialDebt,
		Title:      "可行性研究报告",
		Subtitle:   "（**版）",
		Details:    []string{"编制单位：XXXX有限公司", "编制时间：XXXX年XX月"},
		IncludeTOC: true,
	}
}

func Embodiment() docx.CoverPlugin {
	return ThreePartCover{
		Preset:           PresetEmbodiment,
		Title:            "XX项目",
		Subtitle:         "实施方案",
		Details:          []string{"申报单位：XXXX有限公司", "主管部门：XXXX委员会", "项目业主：XXXX有限公司", "编制时间：XXXX年XX月"},
		VerticalSubtitle: true,
		IncludeTOC:       true,
	}
}

func (c ThreePartCover) RenderCover(renderer *docx.DocxRenderer) error {
	config := renderer.Config()
	c = c.withDefaults(config)
	doc := renderer.Document()

	titlePara := doc.AddParagraph()
	titleParaProps := titlePara.Properties()
	titleParaProps.SetAlignment(wml.ST_JcCenter)
	titleParaProps.Spacing().SetAfter(c.TitleSpacing)

	titleRun := titlePara.AddRun()
	titleProps := titleRun.Properties()
	renderer.SetFontFamily(&titleProps, config.Cover.DefaultFont)
	renderer.SetFontSize(&titleProps, c.TitleSize)
	titleRun.AddBreak()
	titleRun.AddText(c.Title)

	subtitlePara := doc.AddParagraph()
	subtitleParaProps := subtitlePara.Properties()
	subtitleParaProps.SetAlignment(wml.ST_JcCenter)
	subtitleParaProps.Spacing().SetAfter(c.SubtitleSpacing)

	subtitleRun := subtitlePara.AddRun()
	subtitleProps := subtitleRun.Properties()
	renderer.SetFontFamily(&subtitleProps, config.Cover.DefaultFont)
	renderer.SetFontSize(&subtitleProps, c.SubtitleSize)
	addCoverText(subtitleRun, c.Subtitle, c.VerticalSubtitle)

	detailPara := doc.AddParagraph()
	detailParaProps := detailPara.Properties()
	detailParaProps.SetAlignment(c.detailAlignment())
	if c.DetailStartIndent > 0 {
		detailParaProps.SetStartIndent(c.DetailStartIndent)
	}

	detailRun := detailPara.AddRun()
	detailProps := detailRun.Properties()
	renderer.SetFontFamily(&detailProps, config.Cover.DefaultFont)
	renderer.SetFontSize(&detailProps, c.DetailSize)
	for i, detail := range c.Details {
		if i > 0 {
			detailRun.AddBreak()
		}
		detailRun.AddText(detail)
	}
	detailRun.AddPageBreak()

	if c.IncludeTOC {
		renderer.RenderTableOfContents(renderer.BuildTocTree())
	}
	return nil
}

func (c ThreePartCover) withDefaults(config docx.Config) ThreePartCover {
	defaults := coverPresetDefaults(config, c.Preset)
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
	if c.DetailAlignment == nil {
		c.DetailAlignment = Alignment(defaults.DetailAlignment)
	}
	return c
}

func coverPresetDefaults(config docx.Config, preset Preset) docx.ThreePartCoverConfig {
	switch preset {
	case PresetReport:
		return config.Cover.Report
	case PresetSpecialDebt:
		return config.Cover.SpecialDebt
	case PresetEmbodiment:
		return config.Cover.Embodiment
	default:
		return config.Cover.Report
	}
}

func (c ThreePartCover) detailAlignment() wml.ST_Jc {
	if c.DetailAlignment == nil {
		return wml.ST_JcCenter
	}
	return *c.DetailAlignment
}

func Alignment(alignment wml.ST_Jc) *wml.ST_Jc {
	return &alignment
}

func addCoverText(run interface {
	AddBreak()
	AddText(string)
}, text string, vertical bool) {
	if !vertical {
		run.AddText(text)
		return
	}
	for i, r := range []rune(text) {
		if i > 0 {
			run.AddBreak()
		}
		run.AddText(string(r))
	}
}
