package app

import (
	"cli-factory/internal/provider"
	"cli-factory/providers/example"
)

func Registry() (*provider.Registry, error) {
	return provider.NewRegistry(example.New())
}
