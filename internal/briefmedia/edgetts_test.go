package briefmedia

import (
	"strings"
	"testing"
)

func TestEdgeConnectHeadersMatchCurrentClient(t *testing.T) {
	const muid = "0123456789ABCDEF0123456789ABCDEF"
	headers := edgeConnectHeaders(muid)
	if got := headers.Get("Origin"); got != "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold" {
		t.Fatalf("Origin = %q", got)
	}
	if got := headers.Get("Cookie"); got != "muid="+muid+";" {
		t.Fatalf("Cookie = %q", got)
	}
	if got := headers.Get("User-Agent"); !strings.Contains(got, "Edg/"+edgeChromiumMajor+".0.0.0") {
		t.Fatalf("User-Agent = %q", got)
	}
}

func TestEdgeAudioPayload(t *testing.T) {
	header := []byte("Content-Type:audio/mpeg\r\nPath:audio\r\n")
	want := []byte{0xff, 0xfb, 0x90, 0x64}
	frame := append([]byte{byte(len(header) >> 8), byte(len(header))}, header...)
	frame = append(frame, want...)
	got, ok := edgeAudioPayload(frame)
	if !ok || string(got) != string(want) {
		t.Fatalf("edgeAudioPayload() = %x, %v", got, ok)
	}
}
