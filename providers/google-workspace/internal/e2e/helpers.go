package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cli-factory/internal/invocationlog"
	"cli-factory/internal/testharness"
	"cli-factory/providers/google-workspace/internal/googleapi"
	"gopkg.in/yaml.v3"
)

const tokenURL = "https://oauth2.googleapis.com/token"

type Secrets struct {
	ClientID      string `yaml:"client_id"`
	ClientSecret  string `yaml:"client_secret"`
	RefreshToken  string `yaml:"refresh_token"`
	TestUserEmail string `yaml:"test_user_email"`
	CalendarID    string `yaml:"calendar_id"`
}

func LoadSecrets(t *testing.T) Secrets {
	t.Helper()
	root := RepoRoot(t)
	loaded, err := testharness.LoadProviderTestSecrets(context.Background(), testharness.SecretOptionsFromEnv(filepath.Join(root, "providers", "google-workspace")))
	if err != nil {
		if errors.Is(err, testharness.ErrNoSecrets) {
			t.Skip("Google Workspace e2e secrets not configured")
		}
		t.Fatalf("load Google Workspace test secrets: %v", err)
	}
	var secrets Secrets
	if err := yaml.Unmarshal(loaded.Data, &secrets); err != nil {
		t.Fatalf("parse %s: %v", loaded.Path, err)
	}
	if secrets.ClientID == "" || secrets.ClientSecret == "" || secrets.RefreshToken == "" {
		t.Fatalf("%s must define client_id, client_secret, and refresh_token", loaded.Path)
	}
	if secrets.TestUserEmail == "" {
		secrets.TestUserEmail = "bot@trytilde.ai"
	}
	if secrets.CalendarID == "" {
		secrets.CalendarID = "primary"
	}
	return secrets
}

func (s Secrets) ProviderParamsJSON(t *testing.T) string {
	t.Helper()
	data, err := json.Marshal(s.ProviderParams(t))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (s Secrets) ProviderParams(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{
		"access_token": s.AccessToken(t, context.Background()),
		"user_id":      "me",
		"calendar_id":  s.CalendarID,
	}
}

func (s Secrets) Client(t *testing.T) *googleapi.Client {
	t.Helper()
	client, err := googleapi.New(s.ProviderParams(t))
	if err != nil {
		t.Fatalf("create Google API client: %v", err)
	}
	return client
}

func (s Secrets) AccessToken(t *testing.T, ctx context.Context) string {
	t.Helper()
	form := url.Values{}
	form.Set("client_id", s.ClientID)
	form.Set("client_secret", s.ClientSecret)
	form.Set("refresh_token", s.RefreshToken)
	form.Set("grant_type", "refresh_token")
	form.Set("scope", "https://www.googleapis.com/auth/gmail.modify https://www.googleapis.com/auth/gmail.send https://www.googleapis.com/auth/calendar.events")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("Google token refresh failed: %s", resp.Status)
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.AccessToken == "" {
		t.Fatal("Google token refresh response did not include access_token")
	}
	return parsed.AccessToken
}

func RunFactory(t *testing.T, args ...string) invocationlog.Log {
	t.Helper()
	logDir, err := os.MkdirTemp("", "cli-factory-google-workspace-e2e-*")
	if err != nil {
		t.Fatalf("create e2e log dir: %v", err)
	}
	t.Logf("factory invocation logs: %s", logDir)
	fullArgs := append([]string{"run", "./cmd/factory", "--log-dir", logDir}, args...)
	cmd := exec.Command("go", fullArgs...)
	cmd.Dir = RepoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("factory command failed: %v\n%s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "SUCCESS") {
		t.Fatalf("missing SUCCESS output:\n%s", text)
	}
	logPath := logPathFromOutput(t, text)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read invocation log %s: %v", logPath, err)
	}
	var log invocationlog.Log
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("parse invocation log %s: %v", logPath, err)
	}
	if log.Status != "SUCCESS" {
		t.Fatalf("log status = %s, want SUCCESS: %+v", log.Status, log.Error)
	}
	return log
}

func Data(t *testing.T, log invocationlog.Log) map[string]any {
	t.Helper()
	raw, err := json.Marshal(log.Result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse result data: %v", err)
	}
	return parsed.Data
}

func Unique(prefix string) string {
	return prefix + "-" + time.Now().UTC().Format("20060102T150405.000000000Z")
}

func RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func logPathFromOutput(t *testing.T, output string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if path, ok := strings.CutPrefix(line, "full logs at "); ok {
			return strings.TrimSpace(path)
		}
	}
	t.Fatalf("missing log path in output:\n%s", output)
	return ""
}
