package sendemail_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE2ESendEmailCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages/send" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["raw"] == "" {
			t.Fatal("missing raw message")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg-1","threadId":"thread-1","labelIds":["SENT"]}`))
	}))
	defer server.Close()

	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/factory", "--log-dir", t.TempDir(),
		"google-workspace", "send-email",
		"--bearer-token", "test-token",
		"--gmail-base-url", server.URL,
		"--to", "alice@example.com",
		"--subject", "Hello",
		"--body-text", "Test body",
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
