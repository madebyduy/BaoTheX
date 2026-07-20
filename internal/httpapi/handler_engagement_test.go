package httpapi

import "testing"

func TestParseVNDExact(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int64
		ok    bool
	}{
		{"39000", 39000, true},
		{"39000.00", 39000, true},
		{" 39000 ", 39000, true},
		{"39000.01", 0, false},
		{"39000.99", 0, false},
		{"39000.", 0, false},
		{"3.9e4", 0, false},
		{"-39000", 0, false},
		{"0", 0, false},
		{"", 0, false},
	} {
		got, err := parseVND(tc.input)
		if tc.ok && (err != nil || got != tc.want) {
			t.Errorf("parseVND(%q) = %d, %v; want %d", tc.input, got, err, tc.want)
		}
		if !tc.ok && err == nil {
			t.Errorf("parseVND(%q) unexpectedly accepted as %d", tc.input, got)
		}
	}
}

func TestAnonymousClientIDValidation(t *testing.T) {
	for _, value := range []string{"550e8400-e29b-41d4-a716-446655440000", "device_abc123"} {
		if !validAnonymousClientID(value) {
			t.Errorf("valid client id %q rejected", value)
		}
	}
	for _, value := range []string{"", "short", "contains space", "../../admin", string(make([]byte, 65))} {
		if validAnonymousClientID(value) {
			t.Errorf("invalid client id %q accepted", value)
		}
	}
}
