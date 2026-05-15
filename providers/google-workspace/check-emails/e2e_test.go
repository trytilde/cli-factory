package checkemails_test

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE2ECheckEmailsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/gmail/v1/users/me/messages":
			if r.URL.Query().Get("q") != "is:unread" {
				t.Fatalf("query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"messages":[{"id":"msg-1","threadId":"thread-1"}],"resultSizeEstimate":1}`))
		case "/gmail/v1/users/me/messages/msg-1":
			_, _ = w.Write([]byte(`{"id":"msg-1","threadId":"thread-1","snippet":"Hello","payload":{"headers":[{"name":"From","value":"alice@example.com"},{"name":"Subject","value":"Hi"},{"name":"Date","value":"Fri, 15 May 2026 10:00:00 +0000"}]}}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/factory", "--log-dir", t.TempDir(),
		"google-workspace", "check-emails",
		"--bearer-token", "test-token",
		"--gmail-base-url", server.URL,
		"--query", "is:unread",
		"--max-results", "1",
	)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "SUCCESS") {
		t.Fatalf("missing SUCCESS output:\n%s", string(out))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if filepath.Base(dir) == "cli-factory" {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
