package example

import (
	"cli-factory/internal/provider"
	"cli-factory/providers/example/echo"
)

type Provider struct{}

func New() Provider { return Provider{} }

func (Provider) ID() string   { return "example" }
func (Provider) Name() string { return "Example" }
func (Provider) ShortDescription() string {
	return "Example provider for validating CLI Factory behavior."
}
func (Provider) LongDescription() string {
	return "The Example provider contains local non-network tools used to validate command routing, discovery, logging, and e2e tests."
}
func (Provider) Categories() []string { return []string{"example", "debug"} }
func (Provider) Aliases() []string    { return []string{"sample", "test"} }
func (Provider) Parameters() []provider.Parameter {
	return []provider.Parameter{
		{Name: "base_url", Description: "Optional base URL used by providers that proxy requests.", Required: false},
		{Name: "bearer_token", Description: "Optional bearer token for providers that need API auth.", Required: false, Secret: true},
	}
}
func (Provider) Tools() []provider.Tool { return []provider.Tool{echo.Tool{}} }
