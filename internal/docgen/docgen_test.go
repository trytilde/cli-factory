package docgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProviderAndToolDocs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "providers", "google", "cli-metadata.yaml"), `
id: google
name: Google
short_description: Google Workspace provider.
long_description: Long Google provider docs.
categories: [email, calendar]
aliases: [gmail]
provider_parameters:
  - name: bearer_token
    description: Optional OAuth bearer token.
`)
	writeFile(t, filepath.Join(root, "providers", "google", "send-email", "cli-metadata.yaml"), `
id: send-email
name: Send Email
short_description: Send an email.
long_description: Sends a Gmail message.
categories: [email]
command_path: [google, send-email]
`)
	writeFile(t, filepath.Join(root, "providers", "google", "send-email", "input-schema.yaml"), `
type: object
properties:
  to:
    type: string
  subject:
    type: string
`)
	writeFile(t, filepath.Join(root, "providers", "google", "send-email", "output-schema.yaml"), `
type: object
properties:
  message_id:
    type: string
`)

	if err := Generate(root); err != nil {
		t.Fatal(err)
	}

	providerPage := readFile(t, filepath.Join(root, "projects", "cli-factory", "providers", "google", "index.mdx"))
	for _, want := range []string{"# Google", "Long Google provider docs.", "`bearer_token`", "[Send Email](./tools/send-email)"} {
		if !strings.Contains(providerPage, want) {
			t.Fatalf("provider page missing %q\n%s", want, providerPage)
		}
	}

	toolPage := readFile(t, filepath.Join(root, "projects", "cli-factory", "providers", "google", "tools", "send-email.mdx"))
	for _, want := range []string{"# Send Email", "Provider Parameters", "bearer_token", "Input Parameters", "subject, to", "Output Fields", "message_id", "Input Schema", "Output Schema"} {
		if !strings.Contains(toolPage, want) {
			t.Fatalf("tool page missing %q\n%s", want, toolPage)
		}
	}

	config := readFile(t, filepath.Join(root, "docs.json"))
	for _, want := range []string{"https://mintlify.com/docs.json", "Projects", "CLI Factory", "projects/cli-factory/providers/google/index", "projects/cli-factory/providers/google/tools/send-email"} {
		if !strings.Contains(config, want) {
			t.Fatalf("docs.json missing %q\n%s", want, config)
		}
	}
}

func TestGenerateWithoutProvidersKeepsProvidersOverview(t *testing.T) {
	root := t.TempDir()
	if err := Generate(root); err != nil {
		t.Fatal(err)
	}
	config := readFile(t, filepath.Join(root, "docs.json"))
	if !strings.Contains(config, "projects/cli-factory/providers/overview") {
		t.Fatalf("docs.json missing providers overview\n%s", config)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(data)+"\n"), 0o644); err != nil {
		t.Fatal(err)
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
