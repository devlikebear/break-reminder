package tts

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestResolveAPIKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")

	cfg := config.Default()
	cfg.TTSAPIKey = "config-key"
	if got := ResolveAPIKey(cfg); got != "config-key" {
		t.Fatalf("config-set -> %q, want config-key", got)
	}

	cfg.TTSAPIKey = "  "
	t.Setenv("GEMINI_API_KEY", "env-key")
	if got := ResolveAPIKey(cfg); got != "env-key" {
		t.Fatalf("env fallback -> %q, want env-key", got)
	}

	t.Setenv("GEMINI_API_KEY", "")
	cfg.TTSAPIKey = ""
	if got := ResolveAPIKey(cfg); got != "" {
		t.Fatalf("neither set -> %q, want empty", got)
	}

	cfg.TTSAPIKey = "  padded  "
	if got := ResolveAPIKey(cfg); got != "padded" {
		t.Fatalf("trim -> %q, want padded", got)
	}
}

func TestNormalizeGeminiModel(t *testing.T) {
	if got := normalizeGeminiModel(""); got != defaultGeminiModel {
		t.Fatalf("empty -> %q, want %q", got, defaultGeminiModel)
	}
	if got := normalizeGeminiModel("  custom-model  "); got != "custom-model" {
		t.Fatalf("trim -> %q, want custom-model", got)
	}
}

func TestGeminiVoiceAvailable(t *testing.T) {
	if !geminiVoiceAvailable("Zephyr") {
		t.Fatal("Zephyr should be supported by Gemini")
	}
	if !geminiVoiceAvailable("  kore  ") {
		t.Fatal("kore (case-insensitive + trim) should be supported")
	}
	if geminiVoiceAvailable("Yuna") {
		t.Fatal("Yuna should not be treated as a Gemini voice")
	}
}

func TestCanonicalGeminiVoice(t *testing.T) {
	got, ok := canonicalGeminiVoice("zephyr")
	if !ok || got != "Zephyr" {
		t.Fatalf("canonicalGeminiVoice(zephyr) = (%q, %v), want (Zephyr, true)", got, ok)
	}
	if _, ok := canonicalGeminiVoice("Yuna"); ok {
		t.Fatal("Yuna should not canonicalize")
	}
}

func TestWrapPCMAsWAV(t *testing.T) {
	pcm := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	wav := wrapPCMAsWAV(pcm, 24000, 1, 16)

	if len(wav) != 44+len(pcm) {
		t.Fatalf("wav length = %d, want %d", len(wav), 44+len(pcm))
	}
	if string(wav[0:4]) != "RIFF" {
		t.Fatalf("magic = %q, want RIFF", wav[0:4])
	}
	if string(wav[8:12]) != "WAVE" {
		t.Fatalf("format = %q, want WAVE", wav[8:12])
	}
	if string(wav[12:16]) != "fmt " {
		t.Fatalf("fmt subchunk id = %q, want 'fmt '", wav[12:16])
	}
	if string(wav[36:40]) != "data" {
		t.Fatalf("data subchunk id = %q, want data", wav[36:40])
	}

	if got := binary.LittleEndian.Uint32(wav[4:8]); got != uint32(36+len(pcm)) {
		t.Fatalf("chunk size = %d, want %d", got, 36+len(pcm))
	}
	if got := binary.LittleEndian.Uint16(wav[20:22]); got != 1 {
		t.Fatalf("audio format = %d, want 1 (PCM)", got)
	}
	if got := binary.LittleEndian.Uint16(wav[22:24]); got != 1 {
		t.Fatalf("channels = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(wav[24:28]); got != 24000 {
		t.Fatalf("sample rate = %d, want 24000", got)
	}
	if got := binary.LittleEndian.Uint32(wav[28:32]); got != 24000*2 {
		t.Fatalf("byte rate = %d, want %d", got, 24000*2)
	}
	if got := binary.LittleEndian.Uint16(wav[34:36]); got != 16 {
		t.Fatalf("bits per sample = %d, want 16", got)
	}
	if got := binary.LittleEndian.Uint32(wav[40:44]); got != uint32(len(pcm)) {
		t.Fatalf("data size = %d, want %d", got, len(pcm))
	}
	if !bytesEqual(wav[44:], pcm) {
		t.Fatal("pcm payload mismatch")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSynthesizeGeminiSuccess(t *testing.T) {
	samplePCM := []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60}
	encoded := base64.StdEncoding.EncodeToString(samplePCM)

	var capturedAuth, capturedPath string
	var capturedBody geminiRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("x-goog-api-key")
		capturedPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &capturedBody)

		resp := geminiResponse{}
		resp.Candidates = append(resp.Candidates, struct {
			Content struct {
				Parts []struct {
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		}{})
		resp.Candidates[0].Content.Parts = append(resp.Candidates[0].Content.Parts, struct {
			InlineData *struct {
				MimeType string `json:"mimeType"`
				Data     string `json:"data"`
			} `json:"inlineData"`
		}{
			InlineData: &struct {
				MimeType string `json:"mimeType"`
				Data     string `json:"data"`
			}{MimeType: "audio/L16;rate=24000", Data: encoded},
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origEndpoint := geminiEndpoint
	geminiEndpoint = server.URL + "/v1beta/models/%s:generateContent"
	defer func() { geminiEndpoint = origEndpoint }()

	path, err := synthesizeGemini(context.Background(), "fake-key", "", "Zephyr", "hello world")
	if err != nil {
		t.Fatalf("synthesizeGemini err = %v", err)
	}
	defer os.Remove(path)

	if capturedAuth != "fake-key" {
		t.Fatalf("x-goog-api-key = %q, want fake-key", capturedAuth)
	}
	if !strings.Contains(capturedPath, defaultGeminiModel) {
		t.Fatalf("path = %q, want contain %s", capturedPath, defaultGeminiModel)
	}
	if capturedBody.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName != "Zephyr" {
		t.Fatalf("voice = %q, want Zephyr", capturedBody.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
	}
	if len(capturedBody.GenerationConfig.ResponseModalities) == 0 || capturedBody.GenerationConfig.ResponseModalities[0] != "AUDIO" {
		t.Fatalf("responseModalities = %v, want [AUDIO]", capturedBody.GenerationConfig.ResponseModalities)
	}
	if len(capturedBody.Contents) == 0 || len(capturedBody.Contents[0].Parts) == 0 || capturedBody.Contents[0].Parts[0].Text != "hello world" {
		t.Fatalf("contents text mismatch: %+v", capturedBody.Contents)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wav: %v", err)
	}
	if len(data) != 44+len(samplePCM) {
		t.Fatalf("wav length = %d, want %d", len(data), 44+len(samplePCM))
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatalf("wav header invalid: % x", data[0:12])
	}
	if !bytesEqual(data[44:], samplePCM) {
		t.Fatal("pcm payload not preserved in wav")
	}
}

func TestSynthesizeGeminiVoiceCaseInsensitive(t *testing.T) {
	var capturedVoice string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body geminiRequest
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		capturedVoice = body.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName

		resp := geminiResponse{}
		resp.Candidates = append(resp.Candidates, struct {
			Content struct {
				Parts []struct {
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		}{})
		resp.Candidates[0].Content.Parts = append(resp.Candidates[0].Content.Parts, struct {
			InlineData *struct {
				MimeType string `json:"mimeType"`
				Data     string `json:"data"`
			} `json:"inlineData"`
		}{
			InlineData: &struct {
				MimeType string `json:"mimeType"`
				Data     string `json:"data"`
			}{Data: base64.StdEncoding.EncodeToString([]byte{0x00, 0x01})},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origEndpoint := geminiEndpoint
	geminiEndpoint = server.URL + "/v1beta/models/%s:generateContent"
	defer func() { geminiEndpoint = origEndpoint }()

	path, err := synthesizeGemini(context.Background(), "k", "", "  puck  ", "hi")
	if err != nil {
		t.Fatalf("synthesizeGemini err = %v", err)
	}
	defer os.Remove(path)

	if capturedVoice != "Puck" {
		t.Fatalf("voice sent = %q, want Puck (canonicalized)", capturedVoice)
	}
}

func TestSynthesizeGeminiEmptyAPIKey(t *testing.T) {
	_, err := synthesizeGemini(context.Background(), "  ", "", "Zephyr", "x")
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("err = %v, want api key error", err)
	}
}

func TestSynthesizeGeminiHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":401,"message":"API key not valid","status":"UNAUTHENTICATED"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	origEndpoint := geminiEndpoint
	geminiEndpoint = server.URL + "/v1beta/models/%s:generateContent"
	defer func() { geminiEndpoint = origEndpoint }()

	_, err := synthesizeGemini(context.Background(), "bad", "", "Zephyr", "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want 401 context", err)
	}
}

func TestSynthesizeGeminiEmptyCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}))
	defer server.Close()

	origEndpoint := geminiEndpoint
	geminiEndpoint = server.URL + "/v1beta/models/%s:generateContent"
	defer func() { geminiEndpoint = origEndpoint }()

	_, err := synthesizeGemini(context.Background(), "k", "", "Zephyr", "x")
	if err == nil || !strings.Contains(err.Error(), "empty candidates") {
		t.Fatalf("err = %v, want empty candidates", err)
	}
}

func TestSynthesizeGeminiMissingInline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{}]}}]}`))
	}))
	defer server.Close()

	origEndpoint := geminiEndpoint
	geminiEndpoint = server.URL + "/v1beta/models/%s:generateContent"
	defer func() { geminiEndpoint = origEndpoint }()

	_, err := synthesizeGemini(context.Background(), "k", "", "Zephyr", "x")
	if err == nil || !strings.Contains(err.Error(), "inline") {
		t.Fatalf("err = %v, want missing inline audio", err)
	}
}
