package example

import (
	"cli-factory/internal/provider"
	"cli-factory/providers/example/echo"
)

type Provider struct{}

func New() Provider { return Provider{} }

func (Provider) ID() string               { return providerID }
func (Provider) Name() string             { return providerName }
func (Provider) ShortDescription() string { return providerShortDescription }
func (Provider) LongDescription() string  { return providerLongDescription }
func (Provider) Categories() []string {
	return append([]string(nil), providerCategories...)
}
func (Provider) Aliases() []string {
	return append([]string(nil), providerAliases...)
}
func (Provider) Parameters() []provider.Parameter {
	return append([]provider.Parameter(nil), providerParameters...)
}
func (Provider) Tools() []provider.Tool { return []provider.Tool{echo.Tool{}} }
