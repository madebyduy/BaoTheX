package briefmedia

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GoogleTTS narrates through Google Translate's "listen" endpoint — the speaker
// button on translate.google.com. Like Edge it needs no key and has no daily
// budget, and unlike Edge its synthesis endpoint is not IP-blocked from servers,
// which is why it sits in the fallback chain between the two: Edge sounds better
// but 403s from datacenter IPs, Google always answers.
//
// It is the plainest voice of the three — flat and a little robotic, the
// Translate reader rather than a news anchor — and it is rate-limited: the
// endpoint takes ~200 characters per request and starts returning 429 if hit too
// fast, so a nine-minute brief is dozens of paced requests. That is the price of
// a voice that works from anywhere for free.
type GoogleTTS struct {
	lang   string
	client *http.Client
	// pause between requests keeps the burst under Google's rate limit.
	pause time.Duration
}

// NewGoogleTTS constructs a narrator for the given language code (e.g. "vi").
// An empty language disables it.
func NewGoogleTTS(lang string) *GoogleTTS {
	return &GoogleTTS{
		lang:   strings.TrimSpace(lang),
		client: &http.Client{Timeout: 30 * time.Second},
		pause:  350 * time.Millisecond,
	}
}

// Enabled reports whether a language is configured.
func (g *GoogleTTS) Enabled() bool { return g.lang != "" }

// googleTTSChunkRunes is well under the endpoint's ~200-character ceiling; past
// it Google silently truncates the request, dropping the tail of the sentence.
const googleTTSChunkRunes = 180

// googleTTSBitrate is the endpoint's approximate MP3 bitrate, used only to
// estimate the finished brief's duration. The audio itself carries its own
// timing; this figure just answers "roughly how many seconds" for the premium
// length check.
const googleTTSBitrate = 32000

// Render narrates transcript to a single MP3 at outputPath and returns its
// approximate duration in seconds.
func (g *GoogleTTS) Render(ctx context.Context, transcript, outputPath string) (int, error) {
	if !g.Enabled() {
		return 0, fmt.Errorf("google tts: no language configured")
	}
	text := normalizeSpeechText(transcript)
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("google tts: empty transcript")
	}
	chunks := splitTranscript(text, googleTTSChunkRunes)

	var audio []byte
	for i, chunk := range chunks {
		if i > 0 {
			select {
			case <-time.After(g.pause):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		part, err := g.fetchChunk(ctx, chunk, i, len(chunks))
		if err != nil {
			return 0, err
		}
		seconds := float64(len(part)*8) / googleTTSBitrate
		if err := validateNarrationChunk(chunk, seconds); err != nil {
			return 0, fmt.Errorf("google tts: chunk %d: %w", i+1, err)
		}
		audio = append(audio, part...)
	}
	if len(audio) == 0 {
		return 0, fmt.Errorf("google tts: no audio produced")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outputPath, audio, 0o644); err != nil {
		return 0, err
	}
	return len(audio) * 8 / googleTTSBitrate, nil
}

// fetchChunk retrieves the MP3 for one chunk, retrying a 429 a few times with a
// growing pause. A rate-limit rejection is expected under load and is not a
// reason to abandon the whole brief.
func (g *GoogleTTS) fetchChunk(ctx context.Context, text string, idx, total int) ([]byte, error) {
	q := url.Values{
		"ie":      {"UTF-8"},
		"client":  {"tw-ob"},
		"tl":      {g.lang},
		"q":       {text},
		"total":   {fmt.Sprintf("%d", total)},
		"idx":     {fmt.Sprintf("%d", idx)},
		"textlen": {fmt.Sprintf("%d", len([]rune(text)))},
	}
	endpoint := "https://translate.google.com/translate_tts?" + q.Encode()

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		// The endpoint serves audio only to what looks like the Translate page.
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Referer", "https://translate.google.com/")

		resp, err := g.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("google tts: rate limited (429)")
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("google tts: http %d", resp.StatusCode)
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if len(body) == 0 {
			lastErr = fmt.Errorf("google tts: empty audio for chunk %d", idx)
			continue
		}
		return body, nil
	}
	return nil, lastErr
}
