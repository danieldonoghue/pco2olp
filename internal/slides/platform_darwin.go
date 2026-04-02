//go:build darwin

package slides

func platformConverters() []Converter {
	return []Converter{&KeynoteConverter{}, &PDFConverter{}}
}
