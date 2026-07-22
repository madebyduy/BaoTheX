package textutil

import "testing"

func TestDecodeHTMLEntities(t *testing.T) {
	tests := map[string]string{
		"David Beckham cấp ph\u0026eacute;p cho Messi v\u0026agrave; De Paul": "David Beckham cấp phép cho Messi và De Paul",
		"Campuchia g\u0026#x1EED;i lời cảnh báo":                              "Campuchia gửi lời cảnh báo",
		"M\u0026amp;eacute;xico":                                              "México",
		"Bóng đá Việt Nam":                                                    "Bóng đá Việt Nam",
	}
	for input, want := range tests {
		if got := DecodeHTMLEntities(input); got != want {
			t.Fatalf("DecodeHTMLEntities(%q) = %q, want %q", input, got, want)
		}
	}
}
