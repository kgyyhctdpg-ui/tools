package markdown

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"path/filepath"
	"strings"
)

type emlConverter struct {
	engine *MarkItDown
}

func newEMLConverter(engine *MarkItDown) Converter {
	converter := &emlConverter{engine: engine}
	return newExtensionConverter(
		[]string{".eml"},
		[]string{"message/rfc822"},
		converter.convert,
	)
}

func (converter *emlConverter) convert(ctx context.Context, data []byte, _ StreamInfo) (*Result, error) {
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
	return &Result{Title: subject, Markdown: joinMarkdownBlocks(body), Metadata: metadata}, nil
}

func (converter *emlConverter) convertMIMEPart(ctx context.Context, header mail.Header, reader io.Reader, depth int) ([]string, error) {
	if depth > 16 {
		return nil, fmt.Errorf("%w: email MIME nesting is too deep", ErrArchiveLimit)
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
	data, err := readLimited(decoded, converter.engine.maxArchiveFileSize)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case "text/plain":
		return []string{strings.TrimSpace(string(data))}, nil
	case "text/html":
		result, err := newHTMLConverter().Convert(ctx, data, StreamInfo{Extension: ".html", MIMEType: mediaType})
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
		handler := converter.engine.imageHandler
		if handler == nil {
			handler = DataURIImageHandler
		}
		imageURL, err := handler(ctx, Image{Name: name, MIMEType: mediaType, AltText: name, Data: data})
		if err != nil {
			return nil, err
		}
		return []string{markdownImage(name, imageURL)}, nil
	}
	if name == "" {
		return nil, nil
	}
	result, err := converter.engine.convertData(ctx, data, StreamInfo{Name: name, Extension: filepath.Ext(name), MIMEType: mediaType})
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
