package echo_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE2EEchoCommand(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/factory", "--log-dir", t.TempDir(), "example", "echo", "--message", "hello")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "SUCCESS") {
		t.Fatalf("missing SUCCESS output:\n%s", text)
	}
	if !strings.Contains(text, "full logs at ") {
		t.Fatalf("missing log path:\n%s", text)
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
