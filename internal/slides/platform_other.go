//go:build !darwin && !windows

package slides

// platformConverters returns available converters for this platform.
// On Linux, only PDF conversion via pdftoppm is supported.
func platformConverters() []Converter {
	return []Converter{&PDFConverter{}}
}
