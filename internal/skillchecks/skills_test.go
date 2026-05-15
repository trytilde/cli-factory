package skillchecks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderSkillsContainSafetyAndE2ERequirements(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"skills/add-provider/SKILL.md",
		"skills/update-provider/SKILL.md",
	} {
		body := readFile(t, filepath.Join(root, rel))
		for _, required := range []string{
			"ask one high-impact question at a time",
			"destructive",
			"override_test_secrets.yaml",
			"test_secrets.enc.yaml",
			"Read the invocation log path",
			"e2e",
			"SOPS",
			"make generate-docs",
			"projects/cli-factory/providers/<provider>",
		} {
			if !strings.Contains(body, required) {
				t.Fatalf("%s missing required phrase %q", rel, required)
			}
		}
	}
}

func TestUseFactoryCLISkillDocumentsProductionAgentFlow(t *testing.T) {
	root := repoRoot(t)
	body := readFile(t, filepath.Join(root, "skills/use-factory-cli/SKILL.md"))
	for _, required := range []string{
		"factory search",
		"factory discover short",
		"factory discover long",
		"SUCCESS",
		"FAILURE",
		"full logs at",
		"--debug",
		"Preserve context",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("use-factory-cli skill missing %q", required)
		}
	}
}

func TestUpdateProviderSkillPreservesMetadataByDefault(t *testing.T) {
	root := repoRoot(t)
	body := readFile(t, filepath.Join(root, "skills/update-provider/SKILL.md"))
	for _, required := range []string{
		"preserve existing metadata",
		"backwards compatibility",
		"Do not rename",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("update-provider skill missing %q", required)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
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
