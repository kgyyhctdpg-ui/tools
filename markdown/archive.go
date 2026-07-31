package markdown

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
)

type archiveDepthKey struct{}

func newZIPConverter(engine *MarkItDown) Converter {
	return newExtensionConverter(
		[]string{".zip"},
		[]string{"application/zip"},
		func(ctx context.Context, data []byte, _ StreamInfo) (*Result, error) {
			depth, _ := ctx.Value(archiveDepthKey{}).(int)
			if depth >= engine.maxArchiveDepth {
				return nil, fmt.Errorf("%w: maximum nesting depth is %d", ErrArchiveLimit, engine.maxArchiveDepth)
			}
			reader, err := openZip(data)
			if err != nil {
				return nil, err
			}
			if len(reader.File) > engine.maxArchiveFiles {
				return nil, fmt.Errorf("%w: archive contains %d files", ErrArchiveLimit, len(reader.File))
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
			childContext := context.WithValue(ctx, archiveDepthKey{}, depth+1)
			for _, name := range names {
				content, err := readZipEntry(reader.File[entries[name]], engine.maxArchiveFileSize)
				if err != nil {
					return nil, err
				}
				result, err := engine.convertData(childContext, content, StreamInfo{Name: name})
				if errors.Is(err, ErrUnsupportedFormat) {
					continue
				}
				if err != nil {
					return nil, fmt.Errorf("markdown: convert archive entry %q: %w", name, err)
				}
				if result.String() != "" {
					blocks = append(blocks, "## "+name+"\n\n"+result.String())
				}
			}
			return &Result{Markdown: joinMarkdownBlocks(blocks)}, nil
		},
	)
}

func newEPUBConverter() Converter {
	return newExtensionConverter(
		[]string{".epub"},
		[]string{"application/epub+zip"},
		func(ctx context.Context, data []byte, _ StreamInfo) (*Result, error) {
			reader, err := openZip(data)
			if err != nil {
				return nil, err
			}
			parts := make(map[string][]byte)
			for _, file := range reader.File {
				if file.FileInfo().IsDir() {
					continue
				}
				content, err := readZipEntry(file, 64<<20)
				if err != nil {
					return nil, err
				}
				parts[path.Clean(strings.TrimPrefix(file.Name, "/"))] = content
			}

			packagePath, err := epubPackagePath(parts["META-INF/container.xml"])
			if err != nil {
				return nil, err
			}
			packageRoot, err := parseXML(parts[packagePath])
			if err != nil {
				return nil, fmt.Errorf("markdown: parse epub package: %w", err)
			}
			manifest := make(map[string]string)
			for _, item := range packageRoot.descendants("item") {
				href, _ := url.PathUnescape(item.attr("href"))
				manifest[item.attr("id")] = path.Clean(path.Join(path.Dir(packagePath), href))
			}
			spine := make([]string, 0)
			for _, item := range packageRoot.descendants("itemref") {
				if name := manifest[item.attr("idref")]; name != "" {
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
			htmlConverter := newHTMLConverter()
			blocks := make([]string, 0, len(spine))
			for _, name := range spine {
				content, ok := parts[name]
				if !ok {
					continue
				}
				result, err := htmlConverter.Convert(ctx, content, StreamInfo{Name: name, Extension: path.Ext(name), MIMEType: "application/xhtml+xml"})
				if err != nil {
					return nil, fmt.Errorf("markdown: convert epub chapter %q: %w", name, err)
				}
				blocks = append(blocks, result.String())
			}
			title := ""
			if node := packageRoot.first("title"); node != nil {
				title = strings.TrimSpace(node.textContent())
			}
			return &Result{Title: title, Markdown: joinMarkdownBlocks(blocks)}, nil
		},
	)
}

func epubPackagePath(containerXML []byte) (string, error) {
	if len(containerXML) == 0 {
		return "", fmt.Errorf("markdown: epub has no META-INF/container.xml")
	}
	root, err := parseXML(containerXML)
	if err != nil {
		return "", fmt.Errorf("markdown: parse epub container: %w", err)
	}
	rootFile := root.first("rootfile")
	if rootFile == nil || rootFile.attr("full-path") == "" {
		return "", fmt.Errorf("markdown: epub package path is missing")
	}
	return path.Clean(strings.TrimPrefix(rootFile.attr("full-path"), "/")), nil
}
