package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-factory/internal/app"
)

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, nil
}

func TestExampleEchoDefaultOutputWritesLog(t *testing.T) {
	stdout, stderr, code, logDir := run(t, "example", "echo", "--message", "hello")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "SUCCESS") || !strings.Contains(stdout, "full logs at ") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("logs = %d, want 1", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"message": "hello"`) {
		t.Fatalf("log missing result: %s", string(data))
	}
}

func TestDebugPrintsFullOutput(t *testing.T) {
	stdout, stderr, code, _ := run(t, "--debug", "example", "echo", "--message", "hello")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"message":"hello"`) {
		t.Fatalf("debug stdout missing message: %s", stdout)
	}
	if strings.Contains(stdout, "full logs at") {
		t.Fatalf("debug stdout should not be terse: %s", stdout)
	}
}

func TestDiscoverShortTool(t *testing.T) {
	stdout, stderr, code, _ := run(t, "--debug", "discover", "short", "example", "echo")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{`"tool":"echo"`, `"input_params":"message"`, `"outputs":"message"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
	if strings.Contains(stdout, "input_schema") {
		t.Fatalf("short discovery included schema: %s", stdout)
	}
}

func TestDiscoverLongToolIncludesSchemas(t *testing.T) {
	stdout, stderr, code, _ := run(t, "--debug", "discover", "long", "example", "echo")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{"input_schema", "output_schema"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
}

func TestSearchIncludesDiscoverHints(t *testing.T) {
	stdout, stderr, code, _ := runWithEmbedder(t, fakeEmbedder{}, "--debug", "search", "echo")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{"factory discover short example", "factory discover long example echo"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
}

func TestMissingRequiredParamFails(t *testing.T) {
	stdout, stderr, code, _ := run(t, "example", "echo")
	if code == 0 {
		t.Fatalf("expected failure stdout=%s stderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "FAILURE") {
		t.Fatalf("stdout missing FAILURE: %s", stdout)
	}
}

func run(t *testing.T, args ...string) (string, string, int, string) {
	t.Helper()
	return runWithEmbedder(t, nil, args...)
}

func runWithEmbedder(t *testing.T, embedder any, args ...string) (string, string, int, string) {
	t.Helper()
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	logDir := t.TempDir()
	fullArgs := append([]string{"--log-dir", logDir}, args...)
	var stdout, stderr bytes.Buffer
	a := App{Registry: registry, Stdout: &stdout, Stderr: &stderr}
	if e, ok := embedder.(interface {
		Embed(context.Context, string) ([]float64, error)
	}); ok {
		a.Embedder = e
	}
	code := a.Run(context.Background(), fullArgs)
	return stdout.String(), stderr.String(), code, logDir
}
