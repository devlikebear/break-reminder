package mcphost

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/devlikebear/break-reminder/internal/i18n"
)

// Discovery tells the stdio bridge (and anything else local) where the
// engine's MCP endpoint is. The file is 0600: the token is the only thing
// keeping other local users out.
type Discovery struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	PID       int    `json:"pid"`
	StartedAt int64  `json:"started_at"`
}

// HTTPServer is a loopback MCP endpoint guarded by a bearer token.
type HTTPServer struct {
	URL   string
	Token string

	srv           *http.Server
	discoveryPath string
}

// StartHTTP serves h's tools on 127.0.0.1 at a random port and, when
// discoveryPath is set, publishes the endpoint there.
func StartHTTP(h *Host, opts ServerOptions, discoveryPath string) (*HTTPServer, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return h.Server(opts) }, nil)
	s := &HTTPServer{
		URL:           "http://" + ln.Addr().String() + "/mcp",
		Token:         token,
		srv:           &http.Server{Handler: BearerAuth(token, handler), ReadHeaderTimeout: 10 * time.Second},
		discoveryPath: discoveryPath,
	}
	go func() { _ = s.srv.Serve(ln) }()
	if discoveryPath != "" {
		if err := writeDiscovery(discoveryPath, Discovery{URL: s.URL, Token: token, PID: os.Getpid(), StartedAt: time.Now().Unix()}); err != nil {
			s.Close()
			return nil, err
		}
	}
	return s, nil
}

// Close stops serving and removes the discovery file if it is still ours.
func (s *HTTPServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
	if s.discoveryPath == "" {
		return
	}
	if d, err := ReadDiscovery(s.discoveryPath); err == nil && d.Token == s.Token {
		_ = os.Remove(s.discoveryPath)
	}
}

// BearerAuth admits requests carrying the token. A request with an Origin
// header came from a browser page (MCP clients send none) and is refused
// before the token is compared.
func BearerAuth(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ReadDiscovery reads the endpoint file written by StartHTTP.
func ReadDiscovery(path string) (Discovery, error) {
	var d Discovery
	data, err := os.ReadFile(path)
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return d, err
	}
	if d.URL == "" || d.Token == "" {
		return d, errors.New(i18n.Text("incomplete MCP discovery file"))
	}
	return d, nil
}

func writeDiscovery(path string, d Discovery) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// randomToken returns 32 random bytes as hex.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
