//go:build windows

package slides

func platformConverters() []Converter {
	return []Converter{&PowerPointConverter{}, &PDFConverter{}}
}
