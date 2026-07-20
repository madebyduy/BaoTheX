package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsTrailingValue(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"ok":true} {"extra":true}`))
	w := httptest.NewRecorder()
	var body map[string]bool
	if decodeJSON(w, req, &body) {
		t.Fatal("decodeJSON accepted two JSON values")
	}
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestPaginationCapsExtremePage(t *testing.T) {
	req := httptest.NewRequest("GET", "/?page=9223372036854775807&per_page=100", nil)
	page, perPage, offset := pagination(req)
	if page != 1 || perPage != 100 || offset != 0 {
		t.Fatalf("pagination = %d,%d,%d; want 1,100,0", page, perPage, offset)
	}
}

func TestDecodeJSONAcceptsWhitespaceAfterValue(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("{\"ok\":true}\n\t"))
	w := httptest.NewRecorder()
	var body map[string]bool
	if !decodeJSON(w, req, &body) || !body["ok"] {
		t.Fatalf("valid JSON rejected: status=%d body=%v", w.Code, body)
	}
}
