package ingest

import "testing"

func TestNormalizeCanonicalizesWebURL(t *testing.T) {
	got, err := Normalize("http://www.Example.com/sport/?utm_source=test&b=2#top")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/sport?b=2" {
		t.Fatalf("unexpected canonical URL: %q", got)
	}
}

func TestNormalizeAcceptsSchemelessHost(t *testing.T) {
	got, err := Normalize("example.com/story")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/story" {
		t.Fatalf("unexpected canonical URL: %q", got)
	}
}

func TestNormalizeRejectsUnsafeOrIncompleteURL(t *testing.T) {
	for _, raw := range []string{"", "javascript://alert", "https:///missing-host", "https://user:pass@example.com/story"} {
		if got, err := Normalize(raw); err == nil {
			t.Fatalf("Normalize(%q) should fail, got %q", raw, got)
		}
	}
}
