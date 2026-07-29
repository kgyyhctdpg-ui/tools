package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	"github.com/scoming-dev/tools/docx"
	"github.com/scoming-dev/tools/docx/common"
	"github.com/scoming-dev/tools/docx/plugins/cover"
	"github.com/scoming-dev/tools/docx/plugins/tableembed"
)

func main() {
	if err := runWithIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return runWithIO(args, os.Stdin, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var cfg cliConfig

	flags := flag.NewFlagSet("docx", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&cfg.input, "input", "", "Markdown input file path, or - for stdin")
	flags.StringVar(&cfg.inputShort, "i", "", "Markdown input file path, or - for stdin")
	flags.StringVar(&cfg.output, "output", "", "DOCX output file path")
	flags.StringVar(&cfg.outputShort, "o", "", "DOCX output file path")
	flags.StringVar(&cfg.headingStyle, "heading-style", "default", "heading style: default, center-page-break, left-page-break, 0, 1, 2")
	flags.StringVar(&cfg.cover, "cover", "none", "cover preset: none, report, special-debt, embodiment")
	flags.StringVar(&cfg.tableTags, "table-tags", "", "comma-separated s-tag types rendered as Excel tables")
	flags.BoolVar(&cfg.businessPresets, "business-presets", false, "enable legacy business table plugins")
	flags.BoolVar(&cfg.format, "format", true, "normalize Markdown before rendering")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: docx -input report.md -output report.docx [options]\n\n")
		flags.PrintDefaults()
	}

	if len(args) == 0 {
		flags.SetOutput(stdout)
		flags.Usage()
		return nil
	}

	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := cfg.normalize(); err != nil {
		return err
	}

	content, err := readInput(cfg.input, stdin)
	if err != nil {
		return err
	}
	if cfg.format {
		content = []byte(formatMarkdown(string(content)))
	}

	renderer, err := newRenderer(content, cfg)
	if err != nil {
		return err
	}
	renderer.Render()

	if err := os.MkdirAll(filepath.Dir(cfg.output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := renderer.Save(cfg.output); err != nil {
		return fmt.Errorf("save docx: %w", err)
	}

	fmt.Fprintf(stdout, "DOCX generated: %s\n", cfg.output)
	return nil
}

type cliConfig struct {
	input           string
	inputShort      string
	output          string
	outputShort     string
	headingStyle    string
	cover           string
	tableTags       string
	businessPresets bool
	format          bool
}

func (c *cliConfig) normalize() error {
	input, err := mergeAlias("input", c.input, "i", c.inputShort)
	if err != nil {
		return err
	}
	output, err := mergeAlias("output", c.output, "o", c.outputShort)
	if err != nil {
		return err
	}
	if input == "" {
		return errors.New("missing required -input value")
	}
	if output == "" {
		output = defaultOutputPath(input)
	}
	if output == "" {
		return errors.New("missing required -output value when reading from stdin")
	}

	c.input = input
	c.output = output
	c.cover = strings.TrimSpace(strings.ToLower(c.cover))
	c.headingStyle = strings.TrimSpace(strings.ToLower(c.headingStyle))
	return nil
}

func mergeAlias(longName, longValue, shortName, shortValue string) (string, error) {
	if longValue != "" && shortValue != "" && longValue != shortValue {
		return "", fmt.Errorf("-%s and -%s have different values", longName, shortName)
	}
	if longValue != "" {
		return longValue, nil
	}
	return shortValue, nil
}

func defaultOutputPath(input string) string {
	if input == "-" {
		return ""
	}
	ext := filepath.Ext(input)
	if ext == "" {
		return input + ".docx"
	}
	return strings.TrimSuffix(input, ext) + ".docx"
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		content, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return content, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file: %w", err)
	}
	return content, nil
}

func newRenderer(markdown []byte, cfg cliConfig) (*docx.DocxRenderer, error) {
	headingStyle, err := parseHeadingStyle(cfg.headingStyle)
	if err != nil {
		return nil, err
	}

	parseOptions := parse.NewOptions()
	parseOptions.HTMLTag2TextMark = true
	parseOptions.Spin = true
	tree := parse.Parse("", markdown, parseOptions)

	renderOptions := render.NewOptions()
	renderOptions.SoftBreak2HardBreak = false
	renderOptions.RenderListStyle = true

	rendererOptions, err := rendererOptionsFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return docx.NewDocxRenderer(tree, renderOptions, int(headingStyle), rendererOptions...), nil
}

func rendererOptionsFromConfig(cfg cliConfig) ([]docx.RendererOption, error) {
	options := make([]docx.RendererOption, 0)

	coverPlugin, err := coverFromName(cfg.cover)
	if err != nil {
		return nil, err
	}
	if coverPlugin != nil {
		options = append(options, docx.WithCover(coverPlugin))
	}

	plugins := make([]docx.HTMLBlockPlugin, 0)
	for _, tag := range splitCSV(cfg.tableTags) {
		plugins = append(plugins, tableembed.NewExcelTablePlugin(tag, nil))
	}
	if cfg.businessPresets {
		plugins = append(plugins, tableembed.BusinessPresetPlugins()...)
	}
	if len(plugins) > 0 {
		options = append(options, docx.WithHTMLBlockPlugins(plugins...))
	}

	return options, nil
}

func parseHeadingStyle(value string) (docx.HeadingStyle, error) {
	switch value {
	case "", "default", "0":
		return docx.HeadingStyleDefault, nil
	case "center", "center-page-break", "level-one-center-page-break", "1":
		return docx.HeadingStyleLevelOneCenterPageBreak, nil
	case "left", "left-page-break", "level-one-left-page-break", "2":
		return docx.HeadingStyleLevelOneLeftPageBreak, nil
	default:
		num, err := strconv.Atoi(value)
		if err == nil {
			switch num {
			case int(docx.HeadingStyleDefault):
				return docx.HeadingStyleDefault, nil
			case int(docx.HeadingStyleLevelOneCenterPageBreak):
				return docx.HeadingStyleLevelOneCenterPageBreak, nil
			case int(docx.HeadingStyleLevelOneLeftPageBreak):
				return docx.HeadingStyleLevelOneLeftPageBreak, nil
			}
		}
		return docx.HeadingStyleDefault, fmt.Errorf("unsupported heading style %q", value)
	}
}

func coverFromName(name string) (docx.CoverPlugin, error) {
	switch name {
	case "", "none":
		return nil, nil
	case "report":
		return cover.Report(), nil
	case "special-debt", "special_debt":
		return cover.SpecialDebt(), nil
	case "embodiment":
		return cover.Embodiment(), nil
	default:
		return nil, fmt.Errorf("unsupported cover preset %q", name)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func formatMarkdown(content string) string {
	content = common.FormatTag(content)
	content = common.FormatLatexInMathBlocks(content)
	content = common.RemoveSpacesBeforeMathBlockAndLineBreak(content)
	content = common.RemoveEmptyLines(content)
	content = common.RemoveFirstLineDash(content)
	return content
}
