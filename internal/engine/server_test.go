package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type lines struct{ bytes.Buffer }

func (l *lines) messages(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimRight(l.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("non-JSON line on stdout: %q", line)
		}
		out = append(out, m)
	}
	return out
}

func TestServeRoundTripAndErrors(t *testing.T) {
	var out lines
	s := NewServer(&out)
	s.Handle("echo", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct{ V int }
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, InvalidParams(err)
		}
		return map[string]int{"v": p.V}, nil
	})
	s.Handle("boom", func(context.Context, json.RawMessage) (any, error) { return nil, errors.New("kaboom") })
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"echo","params":{"V":7}}`,
		`{"jsonrpc":"2.0","id":"b","method":"missing"}`,
		`not json`,
		`{"jsonrpc":"2.0","method":"echo","params":{"V":1}}`,
		`{"jsonrpc":"2.0","id":3,"method":"boom"}`,
		`{"jsonrpc":"2.0","id":4,"method":"echo","params":"bad"}`,
	}, "\n") + "\n"
	if err := s.Serve(context.Background(), strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
	msgs := out.messages(t)
	if len(msgs) != 5 {
		t.Fatalf("got %d responses, want 5 (notification gets none): %v", len(msgs), msgs)
	}
	if msgs[0]["id"].(float64) != 1 || msgs[0]["result"].(map[string]any)["v"].(float64) != 7 {
		t.Errorf("echo = %v", msgs[0])
	}
	code := func(m map[string]any) float64 { return m["error"].(map[string]any)["code"].(float64) }
	if msgs[1]["id"] != "b" || code(msgs[1]) != CodeMethodNotFound {
		t.Errorf("missing = %v", msgs[1])
	}
	if msgs[2]["id"] != nil || code(msgs[2]) != CodeParseError {
		t.Errorf("parse error = %v", msgs[2])
	}
	if code(msgs[3]) != CodeInternalError {
		t.Errorf("boom = %v", msgs[3])
	}
	if code(msgs[4]) != CodeInvalidParams {
		t.Errorf("invalid params = %v", msgs[4])
	}
	for _, m := range msgs {
		if m["jsonrpc"] != "2.0" {
			t.Errorf("missing jsonrpc version: %v", m)
		}
	}
}

func TestNotifyWritesJSONRPCNotification(t *testing.T) {
	var out lines
	s := NewServer(&out)
	if err := s.Notify("event", map[string]string{"type": "break_start"}); err != nil {
		t.Fatal(err)
	}
	msgs := out.messages(t)
	if len(msgs) != 1 || msgs[0]["method"] != "event" || msgs[0]["id"] != nil {
		t.Fatalf("notification = %v", msgs)
	}
}
