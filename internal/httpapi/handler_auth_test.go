package httpapi

import "testing"

func TestValidEmail(t *testing.T) {
	for _, value := range []string{"reader@example.com", "name+sport@sub.example.vn"} {
		if !validEmail(value) {
			t.Errorf("validEmail(%q) = false", value)
		}
	}
	for _, value := range []string{"", "not-an-email", "a@", "Name <reader@example.com>", "a\r\nb@example.com"} {
		if validEmail(value) {
			t.Errorf("validEmail(%q) = true", value)
		}
	}
}
