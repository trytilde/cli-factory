package main

import (
	"testing"

	"cli-factory/internal/app"
)

func TestSelectedToolsAll(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	tools, err := selectedTools(registry, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) == 0 {
		t.Fatal("expected tools")
	}
}

func TestSelectedToolsProvider(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	tools, err := selectedTools(registry, "example", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].ID != "example.echo" {
		t.Fatalf("tools = %#v, want example.echo", tools)
	}
}

func TestSelectedToolsTool(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	tools, err := selectedTools(registry, "example", "echo")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].ID != "example.echo" {
		t.Fatalf("tools = %#v, want example.echo", tools)
	}
}

func TestSelectedToolsRequiresProviderForTool(t *testing.T) {
	registry, err := app.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selectedTools(registry, "", "echo"); err == nil {
		t.Fatal("expected error")
	}
}
