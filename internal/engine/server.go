package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	// CodeRejected is returned when a valid request is refused by the timer
	// rules (for example snoozing outside a break).
	CodeRejected = -32001
)

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error lets a handler choose the JSON-RPC error code.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string { return e.Message }

// InvalidParams wraps a params decoding problem.
func InvalidParams(err error) error { return &Error{Code: CodeInvalidParams, Message: err.Error()} }

// Rejected wraps a refused but well-formed request.
func Rejected(err error) error { return &Error{Code: CodeRejected, Message: err.Error()} }

// Handler implements one method. The result is marshalled as JSON.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Server speaks newline-delimited JSON-RPC 2.0. Requests are handled in
// order; only JSON is ever written to out.
type Server struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	async    map[string]bool
	inflight sync.WaitGroup
	writeMu  sync.Mutex
	out      io.Writer
}

// NewServer returns a server writing responses and notifications to out.
func NewServer(out io.Writer) *Server {
	return &Server{handlers: map[string]Handler{}, async: map[string]bool{}, out: out}
}

// Handle registers a method handler.
func (s *Server) Handle(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// HandleAsync registers a handler that runs on its own goroutine, for slow
// methods (AI calls) that must not hold up the requests behind them.
func (s *Server) HandleAsync(method string, h Handler) {
	s.Handle(method, h)
	s.mu.Lock()
	s.async[method] = true
	s.mu.Unlock()
}

// Notify sends a JSON-RPC notification.
func (s *Server) Notify(method string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return s.write(message{Method: method, Params: raw})
}

func (s *Server) write(m message) error {
	m.JSONRPC = "2.0"
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err = s.out.Write(append(data, '\n'))
	return err
}

// Serve reads requests until in is exhausted or ctx is cancelled.
func (s *Server) Serve(ctx context.Context, in io.Reader) error {
	defer s.inflight.Wait()
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if s.isAsync(line) {
			copied := append([]byte(nil), line...)
			s.inflight.Add(1)
			go func() {
				defer s.inflight.Done()
				if resp := s.dispatch(ctx, copied); resp != nil {
					_ = s.write(*resp)
				}
			}()
			continue
		}
		if resp := s.dispatch(ctx, line); resp != nil {
			if err := s.write(*resp); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func (s *Server) isAsync(line []byte) bool {
	var probe struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(line, &probe) != nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.async[probe.Method]
}

func (s *Server) dispatch(ctx context.Context, line []byte) *message {
	var req message
	if err := json.Unmarshal(line, &req); err != nil {
		return &message{ID: json.RawMessage("null"), Error: &rpcError{Code: CodeParseError, Message: err.Error()}}
	}
	notification := len(req.ID) == 0
	if req.Method == "" {
		if notification {
			return nil
		}
		return &message{ID: req.ID, Error: &rpcError{Code: CodeInvalidRequest, Message: "missing method"}}
	}
	s.mu.RLock()
	h, ok := s.handlers[req.Method]
	s.mu.RUnlock()
	if !ok {
		if notification {
			return nil
		}
		return &message{ID: req.ID, Error: &rpcError{Code: CodeMethodNotFound, Message: "method not found: " + req.Method}}
	}
	params := req.Params
	if len(params) == 0 {
		params = json.RawMessage("{}")
	}
	result, err := h(ctx, params)
	if notification {
		return nil
	}
	if err != nil {
		var e *Error
		if errors.As(err, &e) {
			return &message{ID: req.ID, Error: &rpcError{Code: e.Code, Message: e.Message}}
		}
		return &message{ID: req.ID, Error: &rpcError{Code: CodeInternalError, Message: err.Error()}}
	}
	if result == nil {
		result = struct{}{}
	}
	return &message{ID: req.ID, Result: result}
}
