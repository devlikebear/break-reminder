package mcphost

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(c)
}

func TestHTTPServerRoundTripAndDiscovery(t *testing.T) {
	disc := filepath.Join(t.TempDir(), "mcp.json")
	srv, err := StartHTTP(&Host{Backend: &fakeBackend{}}, ServerOptions{Actions: true}, disc)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ReadDiscovery(disc)
	if err != nil || d.URL != srv.URL || d.Token == "" || d.PID != os.Getpid() {
		t.Fatalf("discovery = %+v, %v", d, err)
	}
	if info, _ := os.Stat(disc); info.Mode().Perm() != 0o600 {
		t.Fatalf("discovery perm = %o", info.Mode().Perm())
	}

	ctx := context.Background()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx,
		&mcp.StreamableClientTransport{Endpoint: d.URL, HTTPClient: &http.Client{Transport: bearer{d.Token}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_status"})
	if err != nil || res.IsError {
		t.Fatalf("call = %+v, %v", res, err)
	}
	cs.Close()

	// No token, wrong token, or a browser Origin: refused.
	for _, tc := range []struct {
		auth, origin string
		want         int
	}{{"", "", 401}, {"Bearer nope", "", 401}, {"Bearer " + d.Token, "https://evil.example", 403}} {
		req, _ := http.NewRequest("POST", d.URL, nil)
		if tc.auth != "" {
			req.Header.Set("Authorization", tc.auth)
		}
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("auth=%q origin=%q -> %d, want %d", tc.auth, tc.origin, resp.StatusCode, tc.want)
		}
	}

	srv.Close()
	if _, err := os.Stat(disc); !os.IsNotExist(err) {
		t.Fatal("discovery file left behind")
	}
}
