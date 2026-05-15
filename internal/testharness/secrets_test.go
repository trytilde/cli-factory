package testharness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeDecryptor struct {
	data []byte
	err  error
	path string
}

func (f *fakeDecryptor) Decrypt(_ context.Context, path string) ([]byte, error) {
	f.path = path
	return f.data, f.err
}

func TestLoadProviderTestSecretsPrefersOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "override_test_secrets.yaml"), "token: override\n")
	writeFile(t, filepath.Join(dir, "test_secrets.enc.yaml"), "encrypted\n")

	got, err := LoadProviderTestSecrets(context.Background(), SecretOptions{
		ProviderDir: dir,
		Decryptor:   &fakeDecryptor{data: []byte("token: decrypted\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SecretsSourceOverride {
		t.Fatalf("source = %q, want %q", got.Source, SecretsSourceOverride)
	}
	if string(got.Data) != "token: override\n" {
		t.Fatalf("data = %q", string(got.Data))
	}
}

func TestLoadProviderTestSecretsDecryptsEncrypted(t *testing.T) {
	dir := t.TempDir()
	encPath := filepath.Join(dir, "test_secrets.enc.yaml")
	writeFile(t, encPath, "encrypted\n")
	decryptor := &fakeDecryptor{data: []byte("token: decrypted\n")}

	got, err := LoadProviderTestSecrets(context.Background(), SecretOptions{
		ProviderDir: dir,
		Decryptor:   decryptor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SecretsSourceEncrypted {
		t.Fatalf("source = %q, want %q", got.Source, SecretsSourceEncrypted)
	}
	if decryptor.path != encPath {
		t.Fatalf("decrypt path = %q, want %q", decryptor.path, encPath)
	}
}

func TestLoadProviderTestSecretsUsesOverrideWithoutSOPS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "override_test_secrets.yaml"), "token: override\n")

	got, err := LoadProviderTestSecrets(context.Background(), SecretOptions{
		ProviderDir:            dir,
		UseOverrideTestSecrets: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != SecretsSourceOverride {
		t.Fatalf("source = %q, want %q", got.Source, SecretsSourceOverride)
	}
}

func TestGitignoreIgnoresPlaintextProviderSecrets(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, pattern := range []string{
		"providers/**/test_secrets.yaml",
		"providers/**/override_test_secrets.yaml",
		".tmp/",
	} {
		if !strings.Contains(text, pattern) {
			t.Fatalf(".gitignore missing %q", pattern)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "test_secrets.enc.yaml") {
			t.Fatal(".gitignore must not ignore committed test_secrets.enc.yaml files")
		}
	}
}

func TestSecretOptionsFromEnv(t *testing.T) {
	t.Setenv("TEST_SECRETS_FILE", "/tmp/secrets.yaml")
	t.Setenv("USE_OVERRIDE_TEST_SECRETS", "true")
	got := SecretOptionsFromEnv("/provider")
	if got.ProviderDir != "/provider" {
		t.Fatalf("provider dir = %q", got.ProviderDir)
	}
	if got.TestSecretsFile != "/tmp/secrets.yaml" {
		t.Fatalf("test secrets file = %q", got.TestSecretsFile)
	}
	if !got.UseOverrideTestSecrets {
		t.Fatal("expected override test secrets")
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
