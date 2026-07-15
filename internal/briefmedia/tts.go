package briefmedia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const sampleRate = 24000

var ErrQuotaExceeded = fmt.Errorf("tts quota exceeded")

type TTS struct {
	apiKey string
	model  string
	voice  string
	client *http.Client
}

func NewTTS(apiKey, model, voice string) *TTS {
	return &TTS{apiKey: apiKey, model: model, voice: voice, client: &http.Client{Timeout: 90 * time.Second}}
}

func (t *TTS) Enabled() bool { return t.apiKey != "" && t.model != "" }

// Render generates several short PCM chunks for stable long-form narration,
// joins them and writes one browser-playable WAV file.
func (t *TTS) Render(ctx context.Context, transcript, outputPath string) (int, error) {
	if !t.Enabled() {
		return 0, fmt.Errorf("tts: API key not configured")
	}
	// Short chunks are more reliable with Gemini speech: long single responses
	// can finish naturally while silently omitting the tail of the transcript.
	chunks := splitTranscript(transcript, 950)
	var pcm []byte
	for _, chunk := range chunks {
		part, err := t.generatePCM(ctx, chunk)
		if err != nil {
			return 0, err
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
			"text": "Đọc bản tin thể thao tiếng Việt sau bằng giọng phát thanh viên tự nhiên, rõ ràng, khỏe khoắn; không đọc phần hướng dẫn. Nội dung:\n" + transcript,
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
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", t.apiKey)
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
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		if resp.StatusCode >= 400 {
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, fmt.Errorf("%w: %s", ErrQuotaExceeded, clip(string(data), 300))
			}
			return nil, fmt.Errorf("tts: Gemini http %d: %s", resp.StatusCode, clip(string(data), 300))
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
			return nil, err
		}
		for _, candidate := range out.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.InlineData.Data != "" {
					return base64.StdEncoding.DecodeString(part.InlineData.Data)
				}
			}
		}
		return nil, fmt.Errorf("tts: Gemini returned no audio")
	}
	return nil, lastErr
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
