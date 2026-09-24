package markdown

// builtinGroup selects which families of built-in converters an engine holds.
type builtinGroup uint8

const (
	builtinText builtinGroup = 1 << iota
	builtinOffice
	builtinArchive
	builtinImage
	builtinPDF
	builtinAll = builtinText | builtinOffice | builtinArchive | builtinImage | builtinPDF
)

// WithBuiltins enables every built-in converter on an engine created with
// NewCore. Calling it for New is harmless.
func WithBuiltins() Option {
	return func(engine *MarkItDown) { engine.RegisterBuiltins() }
}

// WithTextConverters enables text, Markdown, HTML, CSV, JSON, YAML and XML.
func WithTextConverters() Option {
	return func(engine *MarkItDown) { engine.RegisterTextConverters() }
}

// WithOfficeConverters enables DOCX, XLSX and PPTX.
func WithOfficeConverters() Option {
	return func(engine *MarkItDown) { engine.RegisterOfficeConverters() }
}

// WithArchiveConverters enables EPUB, EML and ZIP.
func WithArchiveConverters() Option {
	return func(engine *MarkItDown) { engine.RegisterArchiveConverters() }
}

// WithImageConverter enables standalone image conversion.
func WithImageConverter() Option {
	return func(engine *MarkItDown) { engine.RegisterImageConverter() }
}

// WithPDFConverter enables PDF conversion.
func WithPDFConverter() Option {
	return func(engine *MarkItDown) { engine.RegisterPDFConverter() }
}

// RegisterBuiltins registers all built-in converters in their original order.
func (engine *MarkItDown) RegisterBuiltins() {
	engine.registerBuiltinGroups(builtinAll)
}

// RegisterTextConverters registers text and structured-text converters.
func (engine *MarkItDown) RegisterTextConverters() {
	engine.registerBuiltinGroups(builtinText)
}

// RegisterOfficeConverters registers DOCX, XLSX and PPTX converters.
func (engine *MarkItDown) RegisterOfficeConverters() {
	engine.registerBuiltinGroups(builtinOffice)
}

// RegisterArchiveConverters registers EPUB, EML and ZIP converters.
func (engine *MarkItDown) RegisterArchiveConverters() {
	engine.registerBuiltinGroups(builtinArchive)
}

// RegisterImageConverter registers the standalone image converter.
func (engine *MarkItDown) RegisterImageConverter() {
	engine.registerBuiltinGroups(builtinImage)
}

// RegisterPDFConverter registers the PDF converter.
func (engine *MarkItDown) RegisterPDFConverter() {
	engine.registerBuiltinGroups(builtinPDF)
}

func (engine *MarkItDown) registerBuiltinGroups(groups builtinGroup) {
	if engine == nil {
		return
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()

	if groups&builtinText != 0 && engine.registeredGroups&builtinText == 0 {
		engine.converters = append(engine.converters,
			newTextConverter(), newHTMLConverter(), newCSVConverter(), newStructuredConverter(),
		)
		engine.registeredGroups |= builtinText
	}
	if groups&builtinOffice != 0 && engine.registeredGroups&builtinOffice == 0 {
		engine.converters = append(engine.converters,
			newDOCXConverter(&engine.Settings), newXLSXConverter(), newPPTXConverter(),
		)
		engine.registeredGroups |= builtinOffice
	}
	if groups&builtinArchive != 0 && engine.registeredGroups&builtinArchive == 0 {
		engine.converters = append(engine.converters,
			newEPUBConverter(), newEMLConverter(&engine.Settings), newZIPConverter(&engine.Settings),
		)
		engine.registeredGroups |= builtinArchive
	}
	if groups&builtinImage != 0 && engine.registeredGroups&builtinImage == 0 {
		engine.converters = append(engine.converters, newImageConverter(&engine.Settings))
		engine.registeredGroups |= builtinImage
	}
	if groups&builtinPDF != 0 && engine.registeredGroups&builtinPDF == 0 {
		engine.converters = append(engine.converters, newPDFConverter(&engine.Settings))
		engine.registeredGroups |= builtinPDF
	}
}
