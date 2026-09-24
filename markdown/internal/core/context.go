package core

import (
	"context"
	"fmt"
	"io"
)

type conversionConfigKey struct{}

// WithTableFormat records the table format for one conversion so converters can
// read it without holding the engine.
func WithTableFormat(ctx context.Context, format TableFormat) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, conversionConfigKey{}, format)
}

// TableFormatFromContext returns the table format recorded for this conversion,
// falling back to the HTML default.
func TableFormatFromContext(ctx context.Context) TableFormat {
	if ctx != nil {
		if format, ok := ctx.Value(conversionConfigKey{}).(TableFormat); ok && format != "" {
			return format
		}
	}
	return TableFormatHTML
}

type archiveDepthKey struct{}

// WithArchiveDepth records how deeply nested the current archive conversion is.
func WithArchiveDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, archiveDepthKey{}, depth)
}

// ArchiveDepthFromContext returns the current archive nesting depth.
func ArchiveDepthFromContext(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	depth, _ := ctx.Value(archiveDepthKey{}).(int)
	return depth
}

// ReadLimited reads at most limit bytes; limit <= 0 means no limit.
func ReadLimited(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(reader)
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: limit is %d bytes", ErrInputTooLarge, limit)
	}
	return data, nil
}
