package createcalendarevent_test

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

func TestE2ECreateCalendarEventCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendar/v3/calendars/primary/events" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["summary"] != "Planning" {
			t.Fatalf("summary = %v", body["summary"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"event-1","htmlLink":"https://calendar.google.com/event","status":"confirmed"}`))
	}))
	defer server.Close()

	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/factory", "--log-dir", t.TempDir(),
		"google-workspace", "create-calendar-event",
		"--bearer-token", "test-token",
		"--calendar-base-url", server.URL,
		"--summary", "Planning",
		"--start", "2026-05-15T10:00:00Z",
		"--end", "2026-05-15T10:30:00Z",
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
