package checkcalendarevents_test

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE2ECheckCalendarEventsCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendar/v3/calendars/primary/events" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if r.URL.Query().Get("singleEvents") != "true" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"timeZone":"UTC","items":[{"id":"event-1","summary":"Planning","status":"confirmed","htmlLink":"https://calendar.google.com/event","start":{"dateTime":"2026-05-15T10:00:00Z"},"end":{"dateTime":"2026-05-15T10:30:00Z"}}]}`))
	}))
	defer server.Close()

	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/factory", "--log-dir", t.TempDir(),
		"google-workspace", "check-calendar-events",
		"--bearer-token", "test-token",
		"--calendar-base-url", server.URL,
		"--time-min", "2026-05-15T00:00:00Z",
		"--time-max", "2026-05-16T00:00:00Z",
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
