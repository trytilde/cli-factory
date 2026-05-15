package invocationlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupRemovesOldLogs(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.json")
	newer := filepath.Join(dir, "new.json")
	if err := os.WriteFile(old, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(old, now.Add(-25*time.Hour), now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := Cleanup(dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old log still exists: %v", err)
	}
	if _, err := os.Stat(newer); err != nil {
		t.Fatalf("new log missing: %v", err)
	}
}

func TestRecorderWritesLog(t *testing.T) {
	rec, err := New(t.TempDir(), []string{"factory", "example", "echo"})
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Finish("SUCCESS", 0, map[string]any{"ok": true}, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rec.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty log")
	}
}
