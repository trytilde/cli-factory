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

func TestInvokeAcceptsCommandStyleFlags(t *testing.T) {
	stdout, stderr, code, _ := run(t, "--debug", "invoke", "example.echo", "--message", "hello")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"message":"hello"`) {
		t.Fatalf("stdout missing message: %s", stdout)
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

func TestHelpIncludesGeneratedProviderCommands(t *testing.T) {
	stdout, stderr, code, _ := run(t, "help")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{
		"CLI Factory discovers and invokes curated SaaS/tool provider commands.",
		"factory search <query>",
		"google-workspace - Use Gmail and Google Calendar from a Google Workspace account.",
		"gmail-send - Send a Gmail message from a Google Workspace account.",
		"example - Example provider for validating CLI Factory behavior.",
		"echo - Echo a message for CLI and e2e validation.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q: %s", want, stdout)
		}
	}
	if strings.Contains(stdout, "full logs at") {
		t.Fatalf("help should not write invocation log output: %s", stdout)
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

func TestMissingRequiredParamsReportsAllMissingFields(t *testing.T) {
	stdout, stderr, code, _ := run(t, "--debug", "google-workspace", "gmail-send")
	if code == 0 {
		t.Fatalf("expected failure stdout=%s stderr=%s", stdout, stderr)
	}
	want := "missing required parameters: access_token, body_text, subject, to"
	if !strings.Contains(stderr, want) {
		t.Fatalf("stderr missing %q: %s", want, stderr)
	}
}

func TestRunLoadsEnvFilesFromCurrentWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	restoreEnv(t, "CLI_FACTORY_ENV_TEST")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("CLI_FACTORY_ENV_TEST=from-env\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.secrets"), []byte("CLI_FACTORY_ENV_TEST=from-secrets\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	_, stderr, code, _ := run(t, "--debug", "discover", "short", "example")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr)
	}
	if got := os.Getenv("CLI_FACTORY_ENV_TEST"); got != "from-secrets" {
		t.Fatalf("env = %q, want from-secrets", got)
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

func restoreEnv(t *testing.T, key string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
}
