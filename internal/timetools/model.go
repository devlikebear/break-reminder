// Package timetools owns time utilities independently of the work/break state.
package timetools

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"strings"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion = 1
const MaxDurationMS int64 = 24 * 60 * 60 * 1000

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string        { return e.Message }
func fail(code, message string) error { return &Error{code, message} }
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "io_error"
}

type Countdown struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	DurationMS  int64  `json:"duration_ms"`
	Phase       string `json:"phase"`
	Deadline    *int64 `json:"deadline_unix_ms"`
	RemainingMS *int64 `json:"remaining_ms"`
	CreatedAt   int64  `json:"created_at"`
	CompletedAt *int64 `json:"completed_at"`
}
type Recent struct {
	Label      string `json:"label"`
	DurationMS int64  `json:"duration_ms"`
}
type Event struct {
	ID             string `json:"id"`
	SourceID       string `json:"source_id"`
	Kind           string `json:"kind"`
	Label          string `json:"label"`
	DueAt          int64  `json:"due_at"`
	DeliveryState  string `json:"delivery_state"`
	AttemptedAt    *int64 `json:"attempted_at"`
	AcknowledgedAt *int64 `json:"acknowledged_at"`
	ErrorCode      string `json:"error_code,omitempty"`
}
type Snapshot struct {
	SchemaVersion int                        `json:"schema_version"`
	Revision      uint64                     `json:"revision"`
	Countdown     *Countdown                 `json:"countdown"`
	Recent        []Recent                   `json:"recent_countdowns"`
	Events        []Event                    `json:"events"`
	Extra         map[string]json.RawMessage `json:"-"`
}

// Preserve later tool fields when an earlier binary updates the countdown.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	type plain Snapshot
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*s = Snapshot(p)
	if err := json.Unmarshal(data, &s.Extra); err != nil {
		return err
	}
	for _, k := range []string{"schema_version", "revision", "countdown", "recent_countdowns", "events"} {
		delete(s.Extra, k)
	}
	if s.Extra == nil {
		s.Extra = map[string]json.RawMessage{}
	}
	if s.Events == nil {
		s.Events = []Event{}
	}
	if s.Recent == nil {
		s.Recent = []Recent{}
	}
	return nil
}
func (s Snapshot) MarshalJSON() ([]byte, error) {
	type plain Snapshot
	data, err := json.Marshal(plain(s))
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	for k, v := range s.Extra {
		if _, exists := raw[k]; !exists {
			raw[k] = v
		}
	}
	return json.Marshal(raw)
}
func NewSnapshot() Snapshot {
	return Snapshot{SchemaVersion: 1, Events: []Event{}, Recent: []Recent{}, Extra: map[string]json.RawMessage{"stopwatch": json.RawMessage(`null`), "stopwatch_history": json.RawMessage(`[]`), "alarms": json.RawMessage(`[]`)}}
}
func (s Snapshot) clone() Snapshot {
	data, _ := json.Marshal(s)
	var copy Snapshot
	_ = json.Unmarshal(data, &copy)
	return copy
}
func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func ptr(v int64) *int64 { return &v }
func labelValue(label string) (string, error) {
	// Reject embedded newlines before trimming so whitespace cannot hide them.
	for _, r := range label {
		if unicode.IsControl(r) {
			return "", fail("invalid_input", i18n.Text("Names cannot contain line breaks or control characters"))
		}
	}
	label = strings.TrimSpace(label)
	if !utf8.ValidString(label) || utf8.RuneCountInString(label) > 80 {
		return "", fail("invalid_input", i18n.Text("Names must be 80 characters or fewer"))
	}
	return label, nil
}
func (s Snapshot) Validate() error {
	if s.SchemaVersion != SchemaVersion {
		return fail("unsupported_schema", i18n.Text("Unsupported time-tools version: {0}", s.SchemaVersion))
	}
	if c := s.Countdown; c != nil {
		if c.ID == "" || c.DurationMS < 1000 || c.DurationMS > MaxDurationMS {
			return fail("store_corrupt", i18n.Text("Invalid timer data"))
		}
		switch c.Phase {
		case "running":
			if c.Deadline == nil {
				return fail("store_corrupt", i18n.Text("Timer deadline is missing"))
			}
		case "paused":
			if c.RemainingMS == nil || *c.RemainingMS <= 0 || *c.RemainingMS > MaxDurationMS {
				return fail("store_corrupt", i18n.Text("Invalid remaining time"))
			}
		case "completed", "canceled":
		default:
			return fail("store_corrupt", i18n.Text("Unknown timer state"))
		}
	}
	seen := map[string]bool{}
	for _, e := range s.Events {
		if e.ID == "" || seen[e.ID] {
			return fail("store_corrupt", i18n.Text("Invalid completion event ID"))
		}
		seen[e.ID] = true
		switch e.DeliveryState {
		case "pending", "claimed", "sent", "failed", "unknown", "silent":
		default:
			return fail("store_corrupt", i18n.Text("Unknown notification state"))
		}
	}
	return nil
}
