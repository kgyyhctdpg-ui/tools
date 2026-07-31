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
	"strings"
	"time"

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
	minerUURL := flags.String("mineru-url", "", "self-hosted MinerU server URL")
	minerUEndpoint := flags.String("mineru-endpoint", "/file_parse", "MinerU file parsing endpoint")
	minerUToken := flags.String("mineru-token", "", "optional MinerU bearer token")
	minerUBackend := flags.String("mineru-backend", "pipeline", "MinerU parsing backend")
	minerUParseMethod := flags.String("mineru-parse-method", "auto", "MinerU parse method")
	minerULanguage := flags.String("mineru-language", "ch", "MinerU document language")
	minerUTimeout := flags.Duration("mineru-timeout", 20*time.Minute, "MinerU request timeout")
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
	assetsDirectory, assetsPrefix := markdownAssetsLocation(source, output, *name)
	options := []markdown.Option{
		markdown.WithTableFormat(format),
		markdown.WithAssetsDirectory(assetsDirectory, assetsPrefix),
	}
	if strings.TrimSpace(*minerUURL) != "" {
		handler, err := markdown.NewMinerUPDFHandler(markdown.MinerUConfig{
			BaseURL:     *minerUURL,
			Endpoint:    *minerUEndpoint,
			Token:       *minerUToken,
			Backend:     *minerUBackend,
			ParseMethod: *minerUParseMethod,
			Language:    *minerULanguage,
			Timeout:     *minerUTimeout,
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		options = append(options, markdown.WithPDFHandler(handler))
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
