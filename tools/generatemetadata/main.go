package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type providerMetadata struct {
	ID               string      `yaml:"id"`
	Name             string      `yaml:"name"`
	ShortDescription string      `yaml:"short_description"`
	LongDescription  string      `yaml:"long_description"`
	Categories       []string    `yaml:"categories"`
	Aliases          []string    `yaml:"aliases"`
	Parameters       []parameter `yaml:"provider_parameters"`
}

type toolMetadata struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	ShortDescription string   `yaml:"short_description"`
	LongDescription  string   `yaml:"long_description"`
	Categories       []string `yaml:"categories"`
	Aliases          []string `yaml:"aliases"`
	CommandPath      []string `yaml:"command_path"`
}

type parameter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Secret      bool   `yaml:"secret"`
}

func main() {
	if err := run("."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	providersDir := filepath.Join(root, "providers")
	entries, err := os.ReadDir(providersDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		providerDir := filepath.Join(providersDir, entry.Name())
		if !fileExists(filepath.Join(providerDir, "cli-metadata.yaml")) {
			continue
		}
		if err := generateProvider(providerDir); err != nil {
			return err
		}
		if err := generateTools(providerDir); err != nil {
			return err
		}
	}
	return nil
}

func generateProvider(dir string) error {
	var meta providerMetadata
	if err := readYAML(filepath.Join(dir, "cli-metadata.yaml"), &meta); err != nil {
		return fmt.Errorf("read provider metadata %s: %w", dir, err)
	}
	pkg, err := packageName(dir)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	writeGeneratedHeader(&b)
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString("import \"cli-factory/internal/provider\"\n\n")
	writeStringConst(&b, "providerID", meta.ID)
	writeStringConst(&b, "providerName", meta.Name)
	writeStringConst(&b, "providerShortDescription", meta.ShortDescription)
	writeStringConst(&b, "providerLongDescription", meta.LongDescription)
	writeStringSlice(&b, "providerCategories", meta.Categories)
	writeStringSlice(&b, "providerAliases", meta.Aliases)
	writeParameters(&b, "providerParameters", meta.Parameters)
	return writeFormatted(filepath.Join(dir, "metadata_gen.go"), b.Bytes())
}

func generateTools(providerDir string) error {
	entries, err := os.ReadDir(providerDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		toolDir := filepath.Join(providerDir, entry.Name())
		if !fileExists(filepath.Join(toolDir, "cli-metadata.yaml")) {
			continue
		}
		if err := generateTool(toolDir); err != nil {
			return err
		}
	}
	return nil
}

func generateTool(dir string) error {
	var meta toolMetadata
	if err := readYAML(filepath.Join(dir, "cli-metadata.yaml"), &meta); err != nil {
		return fmt.Errorf("read tool metadata %s: %w", dir, err)
	}
	pkg, err := packageName(dir)
	if err != nil {
		return err
	}
	inputSchema, err := readSchema(filepath.Join(dir, "input-schema.yaml"))
	if err != nil {
		return err
	}
	outputSchema, err := readSchema(filepath.Join(dir, "output-schema.yaml"))
	if err != nil {
		return err
	}

	var b bytes.Buffer
	writeGeneratedHeader(&b)
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString("import \"cli-factory/internal/schema\"\n\n")
	writeStringConst(&b, "toolID", meta.ID)
	writeStringConst(&b, "toolName", meta.Name)
	writeStringConst(&b, "toolShortDescription", meta.ShortDescription)
	writeStringConst(&b, "toolLongDescription", meta.LongDescription)
	writeStringSlice(&b, "toolCategories", meta.Categories)
	writeStringSlice(&b, "toolAliases", meta.Aliases)
	writeStringSlice(&b, "toolCommandPath", meta.CommandPath)
	fmt.Fprintf(&b, "var toolInputSchema = schema.JSONSchema(%s)\n\n", goLiteral(inputSchema))
	fmt.Fprintf(&b, "var toolOutputSchema = schema.JSONSchema(%s)\n", goLiteral(outputSchema))
	return writeFormatted(filepath.Join(dir, "metadata_gen.go"), b.Bytes())
}

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func readSchema(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("read schema %s: %w", path, err)
	}
	schema, ok := normalize(raw).(map[string]any)
	if !ok || schema == nil {
		return map[string]any{}, nil
	}
	return schema, nil
}

func packageName(dir string) (string, error) {
	path := filepath.Join(dir, "mod.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("read package name %s: %w", path, err)
	}
	return file.Name.Name, nil
}

func writeFormatted(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("format %s: %w\n%s", path, err, src)
	}
	return os.WriteFile(path, formatted, 0o644)
}

func writeStringConst(b *bytes.Buffer, name, value string) {
	fmt.Fprintf(b, "const %s = %s\n\n", name, strconv.Quote(normalizeText(value)))
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func writeGeneratedHeader(b *bytes.Buffer) {
	b.WriteString("// Code generated by make generate-metadata; DO NOT EDIT.\n")
	b.WriteString("// Source of truth: cli-metadata.yaml plus input-schema.yaml/output-schema.yaml.\n")
	b.WriteString("// The generator compiles provider/tool metadata and JSON schemas into static Go values for the factory binary.\n\n")
}

func writeStringSlice(b *bytes.Buffer, name string, values []string) {
	fmt.Fprintf(b, "var %s = []string{%s}\n\n", name, quoteStrings(values))
}

func writeParameters(b *bytes.Buffer, name string, values []parameter) {
	fmt.Fprintf(b, "var %s = []provider.Parameter{\n", name)
	for _, p := range values {
		fmt.Fprintf(
			b,
			"{Name: %s, Description: %s, Required: %t, Secret: %t},\n",
			strconv.Quote(p.Name),
			strconv.Quote(p.Description),
			p.Required,
			p.Secret,
		)
	}
	b.WriteString("}\n")
}

func quoteStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return strings.Join(quoted, ", ")
}

func goLiteral(value any) string {
	switch v := value.(type) {
	case nil:
		return "nil"
	case map[string]any:
		if len(v) == 0 {
			return "map[string]any{}"
		}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var parts []string
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", strconv.Quote(key), goLiteral(v[key])))
		}
		return "map[string]any{" + strings.Join(parts, ", ") + "}"
	case []any:
		if len(v) == 0 {
			return "[]any{}"
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, goLiteral(item))
		}
		return "[]any{" + strings.Join(parts, ", ") + "}"
	case []string:
		if len(v) == 0 {
			return "[]any{}"
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, strconv.Quote(item))
		}
		return "[]any{" + strings.Join(parts, ", ") + "}"
	case string:
		return strconv.Quote(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return strconv.Quote(fmt.Sprint(v))
	}
}

func normalize(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = normalize(item)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[fmt.Sprint(key)] = normalize(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, normalize(item))
		}
		return out
	default:
		return v
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
