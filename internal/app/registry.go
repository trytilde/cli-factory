package app

import (
	"cli-factory/internal/provider"
	"cli-factory/providers/example"
	googleworkspace "cli-factory/providers/google-workspace"
)

func Registry() (*provider.Registry, error) {
	return provider.NewRegistry(example.New(), googleworkspace.New())
}
