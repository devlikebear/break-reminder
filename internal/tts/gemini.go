package tts

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
	"strings"
	"time"

	"github.com/devlikebear/break-reminder/internal/config"
)

// ResolveAPIKey returns the Gemini API key, preferring the config value and
// falling back to the GEMINI_API_KEY environment variable.
func ResolveAPIKey(cfg config.Config) string {
	if key := strings.TrimSpace(cfg.TTSAPIKey); key != "" {
		return key
	}
	return strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
}

const (
	geminiSampleRate    = 24000
	geminiBitsPerSample = 16
	geminiChannels      = 1
	geminiHTTPTimeout   = 20 * time.Second
)

var geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

type geminiRequest struct {
	Contents         []geminiContent   `json:"contents"`
	GenerationConfig geminiGenerConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerConfig struct {
	ResponseModalities []string          `json:"responseModalities"`
	SpeechConfig       geminiSpeechConfg `json:"speechConfig"`
}

type geminiSpeechConfg struct {
	VoiceConfig geminiVoiceConfig `json:"voiceConfig"`
}

type geminiVoiceConfig struct {
	PrebuiltVoiceConfig geminiPrebuilt `json:"prebuiltVoiceConfig"`
}

type geminiPrebuilt struct {
	VoiceName string `json:"voiceName"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// synthesizeGemini calls the Gemini TTS REST endpoint and writes the returned
// PCM audio as a WAV file to a fresh temp path, returning that path. The
// caller owns cleanup of the returned file.
func synthesizeGemini(ctx context.Context, apiKey, model, voice, message string) (string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("gemini: api key is empty")
	}
	canonicalVoice := strings.TrimSpace(voice)
	if canonical, ok := canonicalGeminiVoice(voice); ok {
		canonicalVoice = canonical
	}
	if canonicalVoice == "" {
		return "", fmt.Errorf("gemini: voice is empty")
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: message}}}},
		GenerationConfig: geminiGenerConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: geminiSpeechConfg{
				VoiceConfig: geminiVoiceConfig{
					PrebuiltVoiceConfig: geminiPrebuilt{VoiceName: canonicalVoice},
				},
			},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf(geminiEndpoint, normalizeGeminiModel(model))
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	client := &http.Client{Timeout: geminiHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: http call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("gemini: api error %s: %s", parsed.Error.Status, parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty candidates")
	}
	inline := parsed.Candidates[0].Content.Parts[0].InlineData
	if inline == nil || strings.TrimSpace(inline.Data) == "" {
		return "", fmt.Errorf("gemini: missing inline audio data")
	}

	pcm, err := base64.StdEncoding.DecodeString(inline.Data)
	if err != nil {
		return "", fmt.Errorf("gemini: decode base64 audio: %w", err)
	}
	if len(pcm) == 0 {
		return "", fmt.Errorf("gemini: decoded audio is empty")
	}

	wav := wrapPCMAsWAV(pcm, geminiSampleRate, geminiChannels, geminiBitsPerSample)

	tmp, err := os.CreateTemp("", "gemini-tts-*.wav")
	if err != nil {
		return "", fmt.Errorf("gemini: create temp wav: %w", err)
	}
	if _, err := tmp.Write(wav); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("gemini: write wav: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("gemini: close wav: %w", err)
	}
	return tmp.Name(), nil
}

// wrapPCMAsWAV builds a 44-byte RIFF/WAVE header in front of PCM samples.
func wrapPCMAsWAV(pcm []byte, sampleRate, channels, bitsPerSample int) []byte {
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(pcm)
	chunkSize := 36 + dataSize

	buf := bytes.NewBuffer(make([]byte, 0, 44+dataSize))
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(chunkSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // PCM subchunk size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // PCM format
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(pcm)
	return buf.Bytes()
}
