package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOptionalLoadsEnvAndSecretsFromDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), `
PLAIN=from-env
SHARED=from-env
INLINE=value # comment
export EXPORTED=yes
`)
	writeFile(t, filepath.Join(dir, ".env.secrets"), `
SECRET="from secrets"
SHARED=from-secrets
`)

	restoreEnv(t, "PLAIN", "SHARED", "INLINE", "EXPORTED", "SECRET")

	if err := LoadOptional(dir, ".env", ".env.secrets"); err != nil {
		t.Fatal(err)
	}
	assertEnv(t, "PLAIN", "from-env")
	assertEnv(t, "SHARED", "from-secrets")
	assertEnv(t, "INLINE", "value")
	assertEnv(t, "EXPORTED", "yes")
	assertEnv(t, "SECRET", "from secrets")
}

func TestLoadOptionalPreservesProcessEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "EXISTING=from-file\n")
	t.Setenv("EXISTING", "from-process")

	if err := LoadOptional(dir, ".env"); err != nil {
		t.Fatal(err)
	}
	assertEnv(t, "EXISTING", "from-process")
}

func TestLoadOptionalIgnoresMissingFiles(t *testing.T) {
	if err := LoadOptional(t.TempDir(), ".env", ".env.secrets"); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertEnv(t *testing.T, key, want string) {
	t.Helper()
	if got := os.Getenv(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func restoreEnv(t *testing.T, keys ...string) {
	t.Helper()
	previous := map[string]string{}
	present := map[string]bool{}
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			previous[key] = value
			present[key] = true
		}
		_ = os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				_ = os.Setenv(key, previous[key])
				continue
			}
			_ = os.Unsetenv(key)
		}
	})
}
