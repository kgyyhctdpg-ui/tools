package markdown

import "github.com/scoming-dev/tools/markdown/internal/core"

// NewFileImageHandler returns an ImageHandler that writes every extracted image
// into directory and returns a Markdown link built from linkPrefix. Images with
// identical content are written once and reused.
func NewFileImageHandler(directory, linkPrefix string) ImageHandler {
	return core.NewFileImageHandler(directory, linkPrefix)
}

// WithAssetsDirectory stores extracted images as files instead of data URIs.
// linkPrefix is written into Markdown and is usually the directory name
// relative to the Markdown output file.
func WithAssetsDirectory(directory, linkPrefix string) Option {
	return WithImageHandler(NewFileImageHandler(directory, linkPrefix))
}
