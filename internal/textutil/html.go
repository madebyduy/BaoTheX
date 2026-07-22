// Package textutil contains small text normalizers shared by ingestion and
// presentation boundaries.
package textutil

import "html"

// DecodeHTMLEntities decodes both ordinary and accidentally double-encoded
// HTML entities. A few RSS publishers return titles such as
// "c\u0026amp;eacute;p"; one pass only turns that into "c\u0026eacute;p", so a
// bounded loop is required before the title is safe to store or display.
func DecodeHTMLEntities(value string) string {
	decoded := value
	for range 3 {
		next := html.UnescapeString(decoded)
		if next == decoded {
			break
		}
		decoded = next
	}
	return decoded
}
