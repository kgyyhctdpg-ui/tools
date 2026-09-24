package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/scoming-dev/tools/markdown/internal/core"
	"github.com/scoming-dev/tools/markdown/internal/text"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"path/filepath"
	"strings"
)

type emlConverter struct {
	settings *core.Settings
}

// NewConverter builds the RFC 822 email converter.
func NewConverter(settings *core.Settings) core.Converter {
	converter := &emlConverter{settings: settings}
	return core.NewExtensionConverter(
		[]string{".eml"},
		[]string{"message/rfc822"},
		converter.convert,
	)
}

func (converter *emlConverter) convert(ctx context.Context, data []byte, _ core.StreamInfo) (*core.Result, error) {
	message, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("markdown: parse email: %w", err)
	}
	subject := decodeMIMEHeader(message.Header.Get("Subject"))
	metadata := map[string]string{
		"subject": subject,
		"from":    decodeMIMEHeader(message.Header.Get("From")),
		"to":      decodeMIMEHeader(message.Header.Get("To")),
		"date":    message.Header.Get("Date"),
	}
	body, err := converter.convertMIMEPart(ctx, message.Header, message.Body, 0)
	if err != nil {
		return nil, err
	}
	return &core.Result{Title: subject, Markdown: core.JoinBlocks(body), Metadata: metadata}, nil
}

func (converter *emlConverter) convertMIMEPart(ctx context.Context, header mail.Header, reader io.Reader, depth int) ([]string, error) {
	if depth > 16 {
		return nil, fmt.Errorf("%w: email MIME nesting is too deep", core.ErrArchiveLimit)
	}
	mediaType, params, _ := mime.ParseMediaType(header.Get("Content-Type"))
	if mediaType == "" {
		mediaType = "text/plain"
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		multipartReader := multipart.NewReader(reader, params["boundary"])
		blocks := make([]string, 0)
		for {
			part, err := multipartReader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("markdown: read email MIME part: %w", err)
			}
			partHeader := mail.Header(part.Header)
			converted, err := converter.convertMIMEPart(ctx, partHeader, part, depth+1)
			part.Close()
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, converted...)
		}
		return blocks, nil
	}

	decoded := decodeMIMEBody(reader, header.Get("Content-Transfer-Encoding"))
	data, err := core.ReadLimited(decoded, converter.settings.MaxArchiveFileSize)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case "text/plain":
		return []string{strings.TrimSpace(string(data))}, nil
	case "text/html":
		result, err := text.NewHTMLConverter().Convert(ctx, data, core.StreamInfo{Extension: ".html", MIMEType: mediaType})
		if err != nil {
			return nil, err
		}
		return []string{result.String()}, nil
	}

	_, disposition, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
	name := decodeMIMEHeader(disposition["filename"])
	if name == "" {
		name = decodeMIMEHeader(params["name"])
	}
	if strings.HasPrefix(mediaType, "image/") {
		if name == "" {
			name = "email-image"
		}
		handler := converter.settings.ImageHandlerOrDefault()
		imageURL, err := handler(ctx, core.Image{Name: name, MIMEType: mediaType, AltText: name, Data: data})
		if err != nil {
			return nil, err
		}
		return []string{core.MarkdownImage(name, imageURL)}, nil
	}
	if name == "" {
		return nil, nil
	}
	result, err := converter.settings.Convert(ctx, data, core.StreamInfo{Name: name, Extension: filepath.Ext(name), MIMEType: mediaType})
	if err != nil {
		return nil, nil
	}
	return []string{"### " + name + "\n\n" + result.String()}, nil
}

func decodeMIMEBody(reader io.Reader, transferEncoding string) io.Reader {
	switch strings.ToLower(strings.TrimSpace(transferEncoding)) {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, reader)
	case "quoted-printable":
		return quotedprintable.NewReader(reader)
	default:
		return reader
	}
}

func decodeMIMEHeader(value string) string {
	decoded, err := (&mime.WordDecoder{}).DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}
