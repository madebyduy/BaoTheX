package ingest

import (
	"net"
	"testing"
)

func TestValidatePublicHTTPURL(t *testing.T) {
	for _, raw := range []string{"https://example.com/feed.xml", "http://203.0.113.10/rss"} {
		if err := ValidatePublicHTTPURL(raw); err != nil {
			t.Errorf("public URL %q rejected: %v", raw, err)
		}
	}
	for _, raw := range []string{
		"file:///etc/passwd", "http://localhost/feed", "http://127.0.0.1/rss",
		"http://10.0.0.1/rss", "http://169.254.169.254/latest/meta-data", "https://user:pass@example.com/feed",
	} {
		if err := ValidatePublicHTTPURL(raw); err == nil {
			t.Errorf("unsafe URL %q accepted", raw)
		}
	}
}

func TestPublicIP(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "::1", "10.1.2.3", "172.16.1.2", "192.168.1.2", "169.254.1.1"} {
		if publicIP(net.ParseIP(raw)) {
			t.Errorf("private IP %s treated as public", raw)
		}
	}
}
