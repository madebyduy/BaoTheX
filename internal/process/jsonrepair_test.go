package process

import (
	"encoding/json"
	"testing"

	"repwire/internal/domain"
)

// mustParseClaims is what the caller actually does with the repaired text.
func mustParseClaims(t *testing.T, s string) domain.AnalysisClaims {
	t.Helper()
	var out domain.AnalysisClaims
	if err := json.Unmarshal([]byte(repairTruncatedJSON(s)), &out); err != nil {
		t.Fatalf("repaired text still will not parse: %v\n  input: %s\n  repaired: %s",
			err, s, repairTruncatedJSON(s))
	}
	return out
}

func TestRepairLeavesHealthyJSONAlone(t *testing.T) {
	good := `{"consensus":["Argentina thắng 2-1 [Nguồn: VnExpress]"],"conflicts":[],"unique_claims":[],"open_questions":[]}`
	if got := repairTruncatedJSON(good); got != good {
		t.Fatalf("repair modified valid JSON:\n  got:  %s\n  want: %s", got, good)
	}
}

func TestRepairRecoversTruncationMidString(t *testing.T) {
	// The real failure: the model hit the ceiling partway through a claim.
	// "unexpected end of JSON input" threw away the whole call; the first two
	// claims were already good.
	truncated := `{"consensus":["Argentina thắng 2-1 [Nguồn: VnExpress]","Messi kiến tạo 2 bàn [Nguồn: Dân trí]"],"conflicts":["Trọng tài có bỏ qua pha phạm l`
	claims := mustParseClaims(t, truncated)
	if len(claims.Consensus) != 2 {
		t.Fatalf("lost complete consensus entries: %v", claims.Consensus)
	}
	if len(claims.Conflicts) != 0 {
		t.Fatalf("kept a half-written conflict: %v", claims.Conflicts)
	}
}

func TestRepairRecoversTruncationAfterComma(t *testing.T) {
	truncated := `{"consensus":["a","b"],"conflicts":["c"],`
	claims := mustParseClaims(t, truncated)
	if len(claims.Consensus) != 2 || len(claims.Conflicts) != 1 {
		t.Fatalf("dropped complete entries: %+v", claims)
	}
}

func TestRepairRecoversTruncationInsideNestedArray(t *testing.T) {
	truncated := `{"consensus":["a"],"conflicts":["b","c"`
	claims := mustParseClaims(t, truncated)
	if len(claims.Conflicts) != 2 {
		t.Fatalf("conflicts = %v, want both complete entries", claims.Conflicts)
	}
}

func TestRepairHandlesEscapedQuotesInsideStrings(t *testing.T) {
	// A quote inside a claim must not be mistaken for the end of the string, or
	// the bracket tracking desynchronises and the repair corrupts good data.
	truncated := `{"consensus":["HLV nói \"chúng tôi xứng đáng\" [Nguồn: BBC]","Bàn thắng phút 90`
	claims := mustParseClaims(t, truncated)
	if len(claims.Consensus) != 1 {
		t.Fatalf("consensus = %v, want the one complete quoted claim", claims.Consensus)
	}
	if claims.Consensus[0] != `HLV nói "chúng tôi xứng đáng" [Nguồn: BBC]` {
		t.Fatalf("mangled the escaped quotes: %q", claims.Consensus[0])
	}
}

func TestRepairSkipsProsePrefix(t *testing.T) {
	// extractJSON returns the raw string when there is no closing brace, so the
	// repair has to find its own starting point.
	truncated := "Đây là kết quả phân tích:\n```json\n{\"consensus\":[\"a\"],\"conflicts\":[\"b"
	claims := mustParseClaims(t, truncated)
	if len(claims.Consensus) != 1 || claims.Consensus[0] != "a" {
		t.Fatalf("prose prefix confused the repair: %+v", claims)
	}
}

func TestRepairHandlesTruncationRightAfterOpeningBracket(t *testing.T) {
	claims := mustParseClaims(t, `{"consensus":["a"],"conflicts":[`)
	if len(claims.Consensus) != 1 || len(claims.Conflicts) != 0 {
		t.Fatalf("unexpected: %+v", claims)
	}
}

func TestRepairGivesUpGracefullyOnGarbage(t *testing.T) {
	// No brace at all: return the input rather than inventing structure. The
	// caller then reports the original parse error, which is the honest outcome.
	for _, s := range []string{"", "không phải json", "   "} {
		if got := repairTruncatedJSON(s); got != s {
			t.Fatalf("repair invented structure for %q: %q", s, got)
		}
	}
}

func TestRepairDoesNotBreakTranslationShape(t *testing.T) {
	// The same helper must be safe for any object, not just claims.
	truncated := `{"vietnamese_title":"Norris nhận án phạt","summary":"Tay đua bị lùi 10 bậc`
	var out ForeignDigest
	if err := json.Unmarshal([]byte(repairTruncatedJSON(truncated)), &out); err != nil {
		t.Fatalf("repair failed on a digest shape: %v", err)
	}
	if out.VietnameseTitle != "Norris nhận án phạt" {
		t.Fatalf("lost the complete title: %q", out.VietnameseTitle)
	}
}
