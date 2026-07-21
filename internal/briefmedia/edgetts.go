package briefmedia

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// EdgeTTS narrates through Microsoft Edge's read-aloud endpoint — the same free
// service the Edge browser uses. It needs no API key and has no per-day quota,
// which is the whole point: the Gemini free tier caps TTS at a few hundred calls
// a day and a single brief is several calls, so the morning edition kept dying
// on "quota exceeded". Edge has neither limit.
//
// The catch, stated plainly: this is an undocumented use of a Microsoft service.
// It is what the widely-used `edge-tts` tool speaks, it is stable in practice,
// but Microsoft owes it nothing. The Sec-MS-GEC handshake token below is the
// part most likely to change; if narration suddenly fails with a 403 on connect,
// that is where to look. The caller keeps Gemini as a fallback for exactly this.
type EdgeTTS struct {
	voice  string
	client *http.Client
}

// NewEdgeTTS constructs an Edge narrator for the given voice (e.g.
// "vi-VN-NamMinhNeural"). An empty voice disables it.
//
// proxyURL, if set, routes the connection through an HTTP(S) or SOCKS5 proxy —
// the escape hatch for the endpoint's IP block. Microsoft refuses the synthesis
// endpoint from datacenter ranges, so pointing it at a residential proxy (or
// running the whole worker behind a residential VPN, which needs no proxy here)
// is what gets the good neural voice from a server. Empty means a direct
// connection, and a system-wide VPN is picked up automatically without it.
func NewEdgeTTS(voice, proxyURL string) *EdgeTTS {
	e := &EdgeTTS{voice: strings.TrimSpace(voice)}
	if p := strings.TrimSpace(proxyURL); p != "" {
		if u, err := url.Parse(p); err == nil {
			e.client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(u)}}
		}
	}
	return e
}

// Enabled reports whether a voice is configured.
func (e *EdgeTTS) Enabled() bool { return e.voice != "" }

const (
	// trustedClientToken is the public constant every edge-tts client presents;
	// it is not a secret and is baked into the browser's read-aloud feature.
	edgeTrustedToken = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	edgeWSSBase      = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1"
	// A plausible recent Edge/Chromium build. The server checks the shape, not
	// the exact number.
	edgeChromiumVersion = "143.0.3650.75"
	edgeChromiumMajor   = "143"
	edgeGECVersion      = "1-" + edgeChromiumVersion
	// 24 kHz / 48 kbit mono MP3: small, browser- and Telegram-playable, and the
	// constant bitrate makes duration a clean function of byte length.
	edgeAudioFormat = "audio-24khz-48kbitrate-mono-mp3"
	edgeBitrate     = 48000 // bits per second, for duration estimation

	// edgeChunkRunes is deliberately smaller than the Gemini path's 1,800. Edge
	// streams the audio as it synthesises, so the wall-clock cost of a request
	// tracks the *spoken length* of the text, not its byte count: 1,800
	// characters is over two minutes of speech and the socket ran past the read
	// deadline mid-brief. Around 900 keeps each request to roughly a minute of
	// audio, which returns comfortably inside the timeout below.
	edgeChunkRunes = 900

	// edgeChunkTimeout bounds one request. It is generous relative to the chunk
	// size on purpose — a slow link should stretch, not fail, and a genuinely
	// wedged socket still gets cut off rather than hanging the whole edition.
	edgeChunkTimeout = 90 * time.Second
)

// Render narrates transcript to a single MP3 at outputPath and returns its
// duration in whole seconds.
//
// Long text is split the same way the Gemini path splits it — Edge, like Gemini,
// is more reliable on short requests — and every chunk's MP3 is concatenated.
// Frameless MP3 streams join by byte append and still play as one file, so no
// container muxing is needed.
func (e *EdgeTTS) Render(ctx context.Context, transcript, outputPath string) (int, error) {
	if !e.Enabled() {
		return 0, fmt.Errorf("edge tts: no voice configured")
	}
	text := normalizeSpeechText(transcript)
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("edge tts: empty transcript")
	}
	chunks := splitTranscript(text, edgeChunkRunes)

	var audio []byte
	for _, chunk := range chunks {
		part, err := e.synthesize(ctx, chunk)
		if err != nil {
			return 0, err
		}
		audio = append(audio, part...)
	}
	if len(audio) == 0 {
		return 0, fmt.Errorf("edge tts: no audio produced")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outputPath, audio, 0o644); err != nil {
		return 0, err
	}
	return len(audio) * 8 / edgeBitrate, nil
}

// synthesize opens one connection, sends the config and one SSML request, and
// returns the concatenated MP3 audio for that chunk.
//
// A connection per chunk is deliberate: it keeps each request independent, so a
// mid-brief failure is retryable in isolation and a stuck socket cannot wedge
// the whole edition. The chunks are short and there are only a handful.
func (e *EdgeTTS) synthesize(ctx context.Context, text string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, edgeChunkTimeout)
	defer cancel()

	// ConnectionId is required on the socket URL even though the voices REST
	// endpoint, which shares the same token, does not ask for it. Omitting it is
	// a 403 on the upgrade — the token is accepted but the handshake is not.
	var conn *websocket.Conn
	var err error
	var skew time.Duration
	for attempt := 0; attempt < 2; attempt++ {
		now := time.Now().Add(skew)
		url := fmt.Sprintf("%s?TrustedClientToken=%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=%s&ConnectionId=%s",
			edgeWSSBase, edgeTrustedToken, secMSGEC(now), edgeGECVersion, newRequestID())
		muid := strings.ToUpper(newRequestID())
		var resp *http.Response
		conn, resp, err = websocket.Dial(ctx, url, &websocket.DialOptions{
			HTTPHeader:      edgeConnectHeaders(muid),
			HTTPClient:      e.client, // nil = direct connection (or a system-wide VPN)
			CompressionMode: websocket.CompressionContextTakeover,
		})
		if err == nil {
			break
		}
		if resp == nil {
			break
		}
		statusCode := resp.StatusCode
		serverDate := resp.Header.Get("Date")
		_ = resp.Body.Close()
		if statusCode != http.StatusForbidden || attempt > 0 {
			break
		}
		serverTime, parseErr := http.ParseTime(serverDate)
		if parseErr != nil {
			break
		}
		skew = time.Until(serverTime)
	}
	if err != nil {
		return nil, fmt.Errorf("edge tts: dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	// Audio arrives as binary frames that can exceed the 32 KiB default.
	conn.SetReadLimit(16 << 20)

	if err := conn.Write(ctx, websocket.MessageText, []byte(edgeConfigMessage())); err != nil {
		return nil, fmt.Errorf("edge tts: send config: %w", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, []byte(edgeSSMLMessage(e.voice, text))); err != nil {
		return nil, fmt.Errorf("edge tts: send ssml: %w", err)
	}

	var audio []byte
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return nil, fmt.Errorf("edge tts: read: %w", err)
		}
		switch typ {
		case websocket.MessageText:
			if strings.Contains(string(data), "Path:turn.end") {
				return audio, nil
			}
		case websocket.MessageBinary:
			part, ok := edgeAudioPayload(data)
			if ok {
				audio = append(audio, part...)
			}
		}
	}
}

// secMSGEC computes the anti-abuse token the endpoint requires on connect.
//
// It is SHA-256 of a Windows FILETIME tick count (100-nanosecond units since
// 1601), rounded down to the last five minutes, concatenated with the public
// trusted-client token, uppercased. Five-minute rounding is why one token
// serves a short burst of requests and why a minute or two of clock skew is
// harmless.
//
// The float64 arithmetic is deliberate and must not be "corrected" to integer
// math. The reference edge-tts client computes this in floating point, and the
// tick count is around 1.3e17 — past 2^53, where float64 can no longer hold
// every integer exactly. So the reference value is a specific rounded-off
// number, and the server validates against that exact value. Computing it
// precisely with int64 yields the mathematically-correct-but-different number,
// a different hash, and a 403 on connect. This reproduces the reference's
// rounding on purpose.
func secMSGEC(now time.Time) string {
	ticks := float64(now.UTC().Unix()) + 11644473600.0
	ticks -= math.Mod(ticks, 300.0) // round down to 5 minutes (seconds)
	ticks *= 1e7                    // seconds → 100-nanosecond units
	sum := sha256.Sum256([]byte(fmt.Sprintf("%.0f%s", ticks, edgeTrustedToken)))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func edgeConnectHeaders(muid string) http.Header {
	return http.Header{
		"Pragma":          {"no-cache"},
		"Cache-Control":   {"no-cache"},
		"Origin":          {"chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"},
		"Cookie":          {"muid=" + muid + ";"},
		"Accept-Encoding": {"gzip, deflate, br, zstd"},
		"Accept-Language": {"en-US,en;q=0.9"},
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/" + edgeChromiumMajor + ".0.0.0 Safari/537.36 " +
			"Edg/" + edgeChromiumMajor + ".0.0.0"},
	}
}

func edgeConfigMessage() string {
	return "X-Timestamp:" + edgeTimestamp() + "\r\n" +
		"Content-Type:application/json; charset=utf-8\r\n" +
		"Path:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{` +
		`"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"false"},` +
		`"outputFormat":"` + edgeAudioFormat + `"}}}}`
}

func edgeSSMLMessage(voice, text string) string {
	ssml := "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='vi-VN'>" +
		"<voice name='" + voice + "'>" +
		"<prosody pitch='+0Hz' rate='+0%' volume='+0%'>" + escapeSSML(text) + "</prosody>" +
		"</voice></speak>"
	return "X-RequestId:" + newRequestID() + "\r\n" +
		"Content-Type:application/ssml+xml\r\n" +
		"X-Timestamp:" + edgeTimestamp() + "\r\n" +
		"Path:ssml\r\n\r\n" + ssml
}

// edgeAudioPayload extracts the MP3 bytes from one binary frame. Each frame is
// [uint16 big-endian header length][header text][audio], and only frames whose
// header says Path:audio carry sound.
func edgeAudioPayload(frame []byte) ([]byte, bool) {
	if len(frame) < 2 {
		return nil, false
	}
	headerLen := int(frame[0])<<8 | int(frame[1])
	if 2+headerLen > len(frame) {
		return nil, false
	}
	header := string(frame[2 : 2+headerLen])
	if !strings.Contains(header, "Path:audio") {
		return nil, false
	}
	return frame[2+headerLen:], true
}

func edgeTimestamp() string {
	return time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func escapeSSML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "'", "&apos;", `"`, "&quot;")
	return r.Replace(s)
}
