package process

import (
	"encoding/json"
	"testing"
)

// The live failure this guards: translation jobs died five attempts deep with
// "parse foreign digest: invalid character '}' after top-level value", because
// extractJSON sliced to the LAST '}' and swept trailing junk into the payload.
func TestExtractJSONStopsAtFirstBalancedObject(t *testing.T) {
	cases := map[string]string{
		"trailing stray brace": `{"vietnamese_title":"Tin","summary":"Tóm tắt"}` + "\n}",
		"trailing prose":       `{"vietnamese_title":"Tin","summary":"Tóm tắt"} Hy vọng giúp ích!`,
		"markdown fence":       "```json\n" + `{"vietnamese_title":"Tin","summary":"Tóm tắt"}` + "\n```",
		"second object":        `{"vietnamese_title":"Tin","summary":"Tóm tắt"} {"khac":1}`,
		"leading prose":        `Đây là JSON: {"vietnamese_title":"Tin","summary":"Tóm tắt"}`,
	}
	for name, raw := range cases {
		var out struct {
			VietnameseTitle string `json:"vietnamese_title"`
			Summary         string `json:"summary"`
		}
		if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
			t.Fatalf("%s: still unparseable: %v", name, err)
		}
		if out.VietnameseTitle != "Tin" || out.Summary != "Tóm tắt" {
			t.Fatalf("%s: wrong payload: %+v", name, out)
		}
	}
}

// A '}' inside a string value must not be mistaken for the end of the object.
func TestExtractJSONIgnoresBracesInsideStrings(t *testing.T) {
	raw := `{"summary":"Tỷ số {2-1} gây tranh cãi","vietnamese_title":"Tin"}` + "\ntrailing"
	var out struct {
		Summary         string `json:"summary"`
		VietnameseTitle string `json:"vietnamese_title"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		t.Fatalf("brace inside string broke extraction: %v", err)
	}
	if out.Summary != "Tỷ số {2-1} gây tranh cãi" || out.VietnameseTitle != "Tin" {
		t.Fatalf("wrong payload: %+v", out)
	}
}

// An escaped quote must not flip the in-string state and end the object early.
func TestExtractJSONHandlesEscapedQuotes(t *testing.T) {
	raw := `{"summary":"HLV nói \"chúng tôi sẽ thắng\" hôm qua","vietnamese_title":"Tin"}}`
	var out struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		t.Fatalf("escaped quotes broke extraction: %v", err)
	}
	if out.Summary != `HLV nói "chúng tôi sẽ thắng" hôm qua` {
		t.Fatalf("wrong payload: %q", out.Summary)
	}
}

// Truncated output keeps flowing to the repair path rather than being mangled.
func TestExtractJSONKeepsTruncatedBodyForRepair(t *testing.T) {
	raw := `Đây là JSON: {"vietnamese_body":"nội dung bị cắt giữa chừng`
	got := extractJSON(raw)
	if got[0] != '{' {
		t.Fatalf("expected extraction to start at the opening brace, got %q", got)
	}
}
