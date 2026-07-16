package briefmedia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"repwire/internal/ratelimit"
)

const sampleRate = 24000
const speechChunkRunes = 1800

var ErrQuotaExceeded = fmt.Errorf("tts quota exceeded")

// maxChunkAttempts is how many times one key is tried for a single narration
// chunk before we accept that it really is out of quota and rotate. It must be
// at least 2, or a rate-limit wait can never actually be followed by a retry.
const maxChunkAttempts = 3

// TTS narrates the audio brief. It holds a dedicated pool of API keys (kept
// separate from the summarizer's keys so summarization can't drain the audio
// quota) and rotates to the next key when the active one is quota-exhausted.
type TTS struct {
	model  string
	voice  string
	client *http.Client

	// pacer is shared with the summarizer: both talk to the same Gemini project
	// and draw on the same per-minute allowance.
	pacer *ratelimit.Pacer

	mu       sync.Mutex
	apiKeys  []string
	keyIndex int
}

// NewTTS constructs a TTS narrator. apiKeys is the rotation pool; empty and
// blank entries are dropped.
// pacer may be nil to disable rate pacing. Pass the same Pacer given to the
// summarizer so the two share one per-minute budget.
func NewTTS(apiKeys []string, model, voice string, pacer *ratelimit.Pacer) *TTS {
	keys := make([]string, 0, len(apiKeys))
	for _, k := range apiKeys {
		if s := strings.TrimSpace(k); s != "" {
			keys = append(keys, s)
		}
	}
	return &TTS{
		apiKeys: keys,
		model:   model,
		voice:   voice,
		pacer:   pacer,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (t *TTS) Enabled() bool { return len(t.apiKeys) > 0 && t.model != "" }

// currentKey returns the active key and its index.
func (t *TTS) currentKey() (int, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.keyIndex, t.apiKeys[t.keyIndex]
}

// rotateFrom advances to the next key, but only if the active key is still the
// one that failed, so concurrent callers don't get bumped backwards.
func (t *TTS) rotateFrom(failed int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.keyIndex == failed {
		t.keyIndex = (t.keyIndex + 1) % len(t.apiKeys)
	}
}

// Render generates several short PCM chunks for stable long-form narration,
// joins them and writes one browser-playable WAV file.
func (t *TTS) Render(ctx context.Context, transcript, outputPath string) (int, error) {
	if !t.Enabled() {
		return 0, fmt.Errorf("tts: API key not configured")
	}
	// Short chunks are more reliable with Gemini speech: long single responses
	// can finish naturally while silently omitting the tail of the transcript.
	transcript = normalizeSpeechText(transcript)
	// Keep a long brief reliable without spending the request quota on many
	// tiny fragments. Around 1,800 Vietnamese characters is short enough for
	// stable speech output while allowing both the 6h and 20h editions to fit
	// within a modest daily request allowance.
	chunks := splitTranscript(transcript, speechChunkRunes)
	var pcm []byte
	for index, chunk := range chunks {
		part, err := t.generatePCM(ctx, chunk)
		if err != nil {
			return 0, err
		}
		if index > 0 {
			pcm = append(pcm, make([]byte, sampleRate*2*90/1000)...)
		}
		pcm = append(pcm, part...)
	}
	if len(pcm) == 0 {
		return 0, fmt.Errorf("tts: empty audio")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return 0, err
	}
	if err := writeWAV(outputPath, pcm); err != nil {
		return 0, err
	}
	return len(pcm) / (sampleRate * 2), nil
}

func (t *TTS) generatePCM(ctx context.Context, transcript string) ([]byte, error) {
	payload := map[string]any{
		"contents": []any{map[string]any{"parts": []any{map[string]string{
			"text": `Hãy tổng hợp giọng nói cho một bản tin thể thao tiếng Việt.
Giọng phát thanh viên trong trẻo, ấm và dễ nghe; tự nhiên như đang trò chuyện với thính giả, không lên gân và không mang âm sắc quảng cáo.
Đọc với tốc độ vừa phải, nhấn nhẹ tên người, đội bóng và tỷ số. Ngắt hơi rõ giữa các câu; giữ âm lượng, cao độ và chất giọng ổn định xuyên suốt.
Tuyệt đối không đọc phần hướng dẫn này. Chỉ đọc nguyên văn nội dung nằm giữa hai thẻ BẢN_TIN.

<BẢN_TIN>
` + transcript + `
</BẢN_TIN>`,
		}}}},
		"generationConfig": map[string]any{
			"responseModalities": []string{"AUDIO"},
			"speechConfig": map[string]any{"voiceConfig": map[string]any{
				"prebuiltVoiceConfig": map[string]string{"voiceName": t.voice},
			}},
		},
	}
	body, _ := json.Marshal(payload)
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + t.model + ":generateContent"

	var lastErr error
	// Walk the key pool: each key gets its own transient-retry budget inside
	// requestChunk; only a real quota exhaustion rotates us to the next key.
	for k := 0; k < len(t.apiKeys); k++ {
		idx, key := t.currentKey()
		audio, quota, err := t.requestChunk(ctx, endpoint, body, key)
		if err == nil {
			return audio, nil
		}
		lastErr = err
		if !quota {
			return nil, err // non-quota failure (network/bad response) — another key won't help
		}
		slog.Warn("tts key quota exhausted, rotating to next key",
			"key_index", idx, "keys", len(t.apiKeys))
		t.rotateFrom(idx)
	}
	return nil, lastErr
}

// requestChunk sends one narration request with the given key, retrying a few
// times on transient (network / 5xx) failures. It returns the decoded audio,
// whether the failure was a quota exhaustion (so the caller should rotate keys),
// and the error.
func (t *TTS) requestChunk(ctx context.Context, endpoint string, body []byte, key string) ([]byte, bool, error) {
	var lastErr error
	for attempt := 0; attempt < maxChunkAttempts; attempt++ {
		// Share the summarizer's pacer: one Gemini project, one per-minute
		// allowance. A brief is several chunks, so without this the audio job
		// alone can exhaust the minute and take translation down with it.
		if err := t.pacer.Wait(ctx); err != nil {
			return nil, false, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, false, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", key)
		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 24<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("tts: Gemini temporary error %d", resp.StatusCode)
			select {
			case <-ctx.Done():
				return nil, false, ctx.Err()
			case <-time.After(ratelimit.Backoff(attempt + 1)):
			}
			continue
		}
		if resp.StatusCode >= 400 {
			if resp.StatusCode == http.StatusTooManyRequests {
				// A 429 is "you are going too fast", not "this key is spent".
				// Free-tier Gemini allows only a few requests per minute and a
				// brief is rendered in several chunks, so a long edition trips
				// this every time. Returning immediately meant we rotated to the
				// second key, tripped the same limit within milliseconds, and
				// declared both keys exhausted — which is why the audio brief has
				// produced exactly one file. Wait out the window the way the
				// provider asked, and only give up on the key if it keeps saying
				// no after that.
				lastErr = fmt.Errorf("%w: %s", ErrQuotaExceeded, clip(string(data), 300))
				if attempt == maxChunkAttempts-1 {
					return nil, true, lastErr
				}
				wait := ratelimit.Wait(attempt+1, string(data))
				slog.Warn("tts rate limited, waiting before retry",
					"attempt", attempt+1, "wait", wait)
				select {
				case <-ctx.Done():
					return nil, false, ctx.Err()
				case <-time.After(wait):
				}
				continue
			}
			return nil, false, fmt.Errorf("tts: Gemini http %d: %s", resp.StatusCode, clip(string(data), 300))
		}
		var out struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						InlineData struct {
							Data string `json:"data"`
						} `json:"inlineData"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, false, err
		}
		for _, candidate := range out.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.InlineData.Data != "" {
					decoded, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
					return decoded, false, err
				}
			}
		}
		return nil, false, fmt.Errorf("tts: Gemini returned no audio")
	}
	return nil, false, lastErr
}

var speechTags = regexp.MustCompile(`<[^>]+>`)
var speechSentence = regexp.MustCompile(`[^.!?]+(?:[.!?]+|$)`)

func normalizeSpeechText(text string) string {
	text = html.UnescapeString(text)
	text = speechTags.ReplaceAllString(text, " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	replacer := strings.NewReplacer(
		"BaoTheX", "Báo Thể Ích",
		"HLV", "huấn luyện viên",
		"ĐT Việt Nam", "đội tuyển Việt Nam",
		"FIFA", "Phi-pha",
		"UEFA", "U-ê-pha",
		"NBA", "en-bi-ây",
		"AFF Cup", "A ép ép Cúp",
		"World Cup", "Uôn Cúp",
	)
	text = replacer.Replace(text)
	text = strings.ReplaceAll(text, "…", ".")
	blocks := strings.Split(text, "\n\n")
	clean := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block = strings.Join(strings.Fields(block), " "); block != "" {
			clean = append(clean, block)
		}
	}
	return strings.Join(clean, "\n\n")
}

func splitTranscript(text string, max int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len([]rune(p)) > max {
			if current != "" {
				chunks = append(chunks, current)
				current = ""
			}
			chunks = append(chunks, splitLongSpeech(p, max)...)
			continue
		}
		if current != "" && len([]rune(current))+len([]rune(p))+2 > max {
			chunks = append(chunks, current)
			current = p
		} else if current == "" {
			current = p
		} else {
			current += "\n\n" + p
		}
	}
	if current != "" {
		chunks = append(chunks, current)
	}
	return chunks
}

func splitLongSpeech(text string, max int) []string {
	sentences := speechSentence.FindAllString(text, -1)
	if len(sentences) > 1 {
		var chunks []string
		var current string
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			if len([]rune(sentence)) > max {
				if current != "" {
					chunks = append(chunks, current)
					current = ""
				}
				chunks = append(chunks, splitSpeechWords(sentence, max)...)
				continue
			}
			if current != "" && len([]rune(current))+len([]rune(sentence))+1 > max {
				chunks = append(chunks, current)
				current = sentence
			} else if current == "" {
				current = sentence
			} else {
				current += " " + sentence
			}
		}
		if current != "" {
			chunks = append(chunks, current)
		}
		return chunks
	}
	return splitSpeechWords(text, max)
}

func splitSpeechWords(text string, max int) []string {
	words := strings.Fields(text)
	var chunks []string
	var current []string
	for _, word := range words {
		candidate := strings.Join(append(current, word), " ")
		if len([]rune(candidate)) > max && len(current) > 0 {
			chunks = append(chunks, strings.Join(current, " "))
			current = []string{word}
			continue
		}
		current = append(current, word)
	}
	if len(current) > 0 {
		chunks = append(chunks, strings.Join(current, " "))
	}
	return chunks
}

func writeWAV(path string, pcm []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	byteRate := uint32(sampleRate * 2)
	dataSize := uint32(len(pcm))
	_, _ = f.WriteString("RIFF")
	_ = binary.Write(f, binary.LittleEndian, uint32(36)+dataSize)
	_, _ = f.WriteString("WAVEfmt ")
	_ = binary.Write(f, binary.LittleEndian, uint32(16))
	_ = binary.Write(f, binary.LittleEndian, uint16(1))
	_ = binary.Write(f, binary.LittleEndian, uint16(1))
	_ = binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(f, binary.LittleEndian, byteRate)
	_ = binary.Write(f, binary.LittleEndian, uint16(2))
	_ = binary.Write(f, binary.LittleEndian, uint16(16))
	_, _ = f.WriteString("data")
	_ = binary.Write(f, binary.LittleEndian, dataSize)
	_, err = f.Write(pcm)
	return err
}

func clip(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
