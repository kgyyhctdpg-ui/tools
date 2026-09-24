package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scoming-dev/tools/markdown"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("markdown", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var input string
	var output string
	flags.StringVar(&input, "i", "", "read from this file, URL, or - for stdin")
	flags.StringVar(&input, "input", "", "read from this file, URL, or - for stdin")
	flags.StringVar(&output, "o", "", "write Markdown to this file")
	flags.StringVar(&output, "output", "", "write Markdown to this file")
	tableFormat := flags.String("table-format", "html", "table format: html or markdown")
	name := flags.String("name", "stdin.txt", "source name when reading stdin")
	mimeType := flags.String("mime-type", "", "source MIME type when reading stdin")
	listFormats := flags.Bool("formats", false, "list supported file extensions")
	assetsDirectory := flags.String("assets-dir", "", "directory for extracted images (default: a folder named after the source)")
	assetsPrefix := flags.String("assets-prefix", "", "link prefix written into Markdown (defaults to -assets-dir)")
	maxInputSize := flags.String("max-input-size", "", "optional cap on the input size, e.g. 512MB or 1GB (default: unlimited)")

	pdfDPI := flags.Float64("pdf-dpi", 200, "resolution for rendered PDF pages and the resolution ceiling for embedded images")
	pdfImageFormat := flags.String("pdf-image-format", "jpeg", "format for PDF pictures: jpeg or png (applies to embedded images too)")
	pdfJPEGQuality := flags.Int("pdf-jpeg-quality", 85, "JPEG quality for PDF pictures (1-100)")
	pdfFirstPage := flags.Int("pdf-first-page", 0, "first PDF page to convert (1-based, 0 = first page)")
	pdfLastPage := flags.Int("pdf-last-page", 0, "last PDF page to convert (1-based, 0 = last page)")
	pdfMaxImagesPerPage := flags.Int("pdf-max-images-per-page", 32, "rasterize a PDF page that carries more embedded images than this")
	pdfMaxImages := flags.Int("pdf-max-images", 0, "maximum embedded images extracted per PDF (0 = unlimited)")
	pdfMaxRenderedPages := flags.Int("pdf-max-rendered-pages", 0, "maximum PDF pages rasterized per document (0 = unlimited)")
	pdfNoRenderFallback := flags.Bool("pdf-no-render-fallback", false, "leave pages without text or images empty instead of rendering them")
	pdfNoTables := flags.Bool("pdf-no-tables", false, "disable PDF table reconstruction")
	pdfNoHeadings := flags.Bool("pdf-no-headings", false, "disable PDF heading detection")
	pdfNoReflow := flags.Bool("pdf-no-reflow", false, "disable joining wrapped PDF lines into paragraphs")
	pdfWorkers := flags.Int("pdf-workers", 0, "PDF pages converted at once (0 = one per CPU, capped at 8; 1 = sequential)")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	format := markdown.TableFormatMarkdown
	if strings.EqualFold(*tableFormat, string(markdown.TableFormatHTML)) {
		format = markdown.TableFormatHTML
	} else if !strings.EqualFold(*tableFormat, string(markdown.TableFormatMarkdown)) {
		fmt.Fprintf(stderr, "markdown: invalid table format %q\n", *tableFormat)
		return 2
	}
	if *listFormats {
		engine := markdown.New(markdown.WithTableFormat(format))
		fmt.Fprintln(stdout, strings.Join(engine.SupportedExtensions(), "\n"))
		return 0
	}
	source := input
	if source != "" {
		if flags.NArg() != 0 {
			fmt.Fprintln(stderr, "markdown: input cannot be provided by both -i and a positional argument")
			return 2
		}
	} else {
		if flags.NArg() != 1 {
			fmt.Fprintln(stderr, "usage: markdown -i <file-or-url|-> [-o output.md]")
			return 2
		}
		source = flags.Arg(0)
	}
	defaultDirectory, defaultPrefix := markdownAssetsLocation(source, output, *name)
	resolvedDirectory := strings.TrimSpace(*assetsDirectory)
	switch {
	case resolvedDirectory == "":
		resolvedDirectory = defaultDirectory
	case !filepath.IsAbs(resolvedDirectory) && output != "":
		// Keep the Markdown link valid: an explicit relative directory is
		// resolved next to the Markdown file being written.
		resolvedDirectory = filepath.Join(filepath.Dir(output), resolvedDirectory)
	}
	resolvedPrefix := strings.TrimSpace(*assetsPrefix)
	if resolvedPrefix == "" {
		if strings.TrimSpace(*assetsDirectory) != "" {
			resolvedPrefix = filepath.ToSlash(strings.TrimSpace(*assetsDirectory))
		} else {
			resolvedPrefix = defaultPrefix
		}
	}
	inputLimit, limitErr := parseMarkdownSize(*maxInputSize)
	if limitErr != nil {
		fmt.Fprintf(stderr, "markdown: invalid -max-input-size %q: %v\n", *maxInputSize, limitErr)
		return 2
	}
	options := []markdown.Option{
		markdown.WithTableFormat(format),
		markdown.WithAssetsDirectory(resolvedDirectory, resolvedPrefix),
		markdown.WithPDFOptions(markdown.PDFOptions{
			RenderDPI:                  *pdfDPI,
			RenderFormat:               markdown.PDFImageFormat(strings.ToLower(strings.TrimSpace(*pdfImageFormat))),
			JPEGQuality:                *pdfJPEGQuality,
			FirstPage:                  *pdfFirstPage,
			LastPage:                   *pdfLastPage,
			MaxImagesPerPage:           *pdfMaxImagesPerPage,
			MaxImages:                  *pdfMaxImages,
			MaxRenderedPages:           *pdfMaxRenderedPages,
			DisablePageRenderFallback:  *pdfNoRenderFallback,
			DisableTableReconstruction: *pdfNoTables,
			DisableHeadingDetection:    *pdfNoHeadings,
			DisableParagraphReflow:     *pdfNoReflow,
			PageConcurrency:            *pdfWorkers,
		}),
	}
	if inputLimit > 0 {
		options = append(options, markdown.WithMaxInputSize(inputLimit))
	}
	engine := markdown.New(options...)

	var (
		result *markdown.Result
		err    error
	)
	if source == "-" {
		result, err = engine.ConvertReader(ctx, stdin, markdown.StreamInfo{Name: *name, MIMEType: *mimeType})
	} else {
		result, err = engine.Convert(ctx, source)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, warning := range result.Warnings {
		if warning = strings.TrimSpace(warning); warning != "" {
			fmt.Fprintln(stderr, "warning:", warning)
		}
	}
	markdown := result.String()
	if output != "" {
		if err := os.WriteFile(output, []byte(markdown+"\n"), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if _, err := fmt.Fprintln(stdout, markdown); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// parseMarkdownSize reads a byte count, optionally with a KB/MB/GB/TB suffix.
// An empty value means "no limit".
func parseMarkdownSize(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	upper := strings.ToUpper(trimmed)
	multiplier := float64(1)
	for _, unit := range []struct {
		suffix string
		factor float64
	}{
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	} {
		if strings.HasSuffix(upper, unit.suffix) {
			multiplier = unit.factor
			upper = strings.TrimSpace(strings.TrimSuffix(upper, unit.suffix))
			break
		}
	}
	number, err := strconv.ParseFloat(upper, 64)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("expected a size such as 512MB or 1GB")
	}
	return int64(number * multiplier), nil
}

func markdownAssetsLocation(source, output, stdinName string) (string, string) {
	name := source
	if source == "-" {
		name = stdinName
	} else if parsed, err := url.Parse(source); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		name = parsed.Path
	}
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "document"
	}
	digest := md5.Sum([]byte(name))
	folder := fmt.Sprintf("%x", digest)
	if output == "" {
		return folder, folder
	}
	return filepath.Join(filepath.Dir(output), folder), folder
}
