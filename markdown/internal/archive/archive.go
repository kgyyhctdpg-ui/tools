package archive

import (
	"context"
	"errors"
	"fmt"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"github.com/scoming-dev/tools/markdown/internal/text"
	"net/url"
	"path"
	"sort"
	"strings"
)

// NewZIPConverter builds the ZIP archive converter.
func NewZIPConverter(settings *core.Settings) core.Converter {
	return core.NewExtensionConverter(
		[]string{".zip"},
		[]string{"application/zip"},
		func(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
			depth := core.ArchiveDepthFromContext(ctx)
			if depth >= settings.MaxArchiveDepth {
				return nil, fmt.Errorf("%w: maximum nesting depth is %d", core.ErrArchiveLimit, settings.MaxArchiveDepth)
			}
			reader, err := core.OpenZip(data)
			if err != nil {
				return nil, err
			}
			if len(reader.File) > settings.MaxArchiveFiles {
				return nil, fmt.Errorf("%w: archive contains %d files", core.ErrArchiveLimit, len(reader.File))
			}
			names := make([]string, 0, len(reader.File))
			entries := make(map[string]int)
			for index, file := range reader.File {
				name := path.Clean(strings.TrimPrefix(file.Name, "/"))
				if file.FileInfo().IsDir() || strings.HasPrefix(name, "__MACOSX/") || path.Base(name) == ".DS_Store" {
					continue
				}
				names = append(names, name)
				entries[name] = index
			}
			sort.Strings(names)
			blocks := make([]string, 0, len(names))
			childContext := core.WithArchiveDepth(ctx, depth+1)
			for _, name := range names {
				content, err := core.ReadZipEntry(reader.File[entries[name]], settings.MaxArchiveFileSize)
				if err != nil {
					return nil, err
				}
				result, err := settings.Convert(childContext, content, core.StreamInfo{Name: name})
				if errors.Is(err, core.ErrUnsupportedFormat) {
					continue
				}
				if err != nil {
					return nil, fmt.Errorf("markdown: convert archive entry %q: %w", name, err)
				}
				if result.String() != "" {
					blocks = append(blocks, "## "+name+"\n\n"+result.String())
				}
			}
			return &core.Result{Markdown: core.JoinBlocks(blocks)}, nil
		},
	)
}

// NewEPUBConverter builds the EPUB converter.
func NewEPUBConverter() core.Converter {
	return core.NewExtensionConverter(
		[]string{".epub"},
		[]string{"application/epub+zip"},
		func(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
			reader, err := core.OpenZip(data)
			if err != nil {
				return nil, err
			}
			parts := make(map[string][]byte)
			for _, file := range reader.File {
				if file.FileInfo().IsDir() {
					continue
				}
				content, err := core.ReadZipEntry(file, 64<<20)
				if err != nil {
					return nil, err
				}
				parts[path.Clean(strings.TrimPrefix(file.Name, "/"))] = content
			}

			packagePath, err := epubPackagePath(parts["META-INF/container.xml"])
			if err != nil {
				return nil, err
			}
			packageRoot, err := core.ParseXML(parts[packagePath])
			if err != nil {
				return nil, fmt.Errorf("markdown: parse epub package: %w", err)
			}
			manifest := make(map[string]string)
			for _, item := range packageRoot.Descendants("item") {
				href, _ := url.PathUnescape(item.Attr("href"))
				manifest[item.Attr("id")] = path.Clean(path.Join(path.Dir(packagePath), href))
			}
			spine := make([]string, 0)
			for _, item := range packageRoot.Descendants("itemref") {
				if name := manifest[item.Attr("idref")]; name != "" {
					spine = append(spine, name)
				}
			}
			if len(spine) == 0 {
				for _, name := range manifest {
					if strings.HasSuffix(name, ".xhtml") || strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".htm") {
						spine = append(spine, name)
					}
				}
				sort.Strings(spine)
			}
			htmlConverter := text.NewHTMLConverter()
			blocks := make([]string, 0, len(spine))
			for _, name := range spine {
				content, ok := parts[name]
				if !ok {
					continue
				}
				result, err := htmlConverter.Convert(ctx, content, core.StreamInfo{Name: name, Extension: path.Ext(name), MIMEType: "application/xhtml+xml"})
				if err != nil {
					return nil, fmt.Errorf("markdown: convert epub chapter %q: %w", name, err)
				}
				blocks = append(blocks, result.String())
			}
			title := ""
			if node := packageRoot.First("title"); node != nil {
				title = strings.TrimSpace(node.TextContent())
			}
			return &core.Result{Title: title, Markdown: core.JoinBlocks(blocks)}, nil
		},
	)
}

func epubPackagePath(containerXML []byte) (string, error) {
	if len(containerXML) == 0 {
		return "", fmt.Errorf("markdown: epub has no META-INF/container.xml")
	}
	root, err := core.ParseXML(containerXML)
	if err != nil {
		return "", fmt.Errorf("markdown: parse epub container: %w", err)
	}
	rootFile := root.First("rootfile")
	if rootFile == nil || rootFile.Attr("full-path") == "" {
		return "", fmt.Errorf("markdown: epub package path is missing")
	}
	return path.Clean(strings.TrimPrefix(rootFile.Attr("full-path"), "/")), nil
}
