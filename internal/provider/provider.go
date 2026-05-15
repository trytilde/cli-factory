package provider

import (
	"context"

	"cli-factory/internal/schema"
)

type Parameter struct {
	Name        string
	Description string
	Required    bool
	Secret      bool
}

type Provider interface {
	ID() string
	Name() string
	ShortDescription() string
	LongDescription() string
	Categories() []string
	Aliases() []string
	Parameters() []Parameter
	Tools() []Tool
}

type Tool interface {
	ID() string
	Name() string
	ShortDescription() string
	LongDescription() string
	Categories() []string
	Aliases() []string
	InputSchema() schema.JSONSchema
	OutputSchema() schema.JSONSchema
	Invoke(ctx context.Context, req InvokeRequest, events EventSink) (InvokeResult, error)
}

type InvokeRequest struct {
	ProviderParams map[string]any `json:"provider_params"`
	Params         map[string]any `json:"params"`
}

type InvokeResult struct {
	Data map[string]any `json:"data"`
}

type Event struct {
	Type     string         `json:"type"`
	Time     string         `json:"time,omitempty"`
	Provider string         `json:"provider,omitempty"`
	Tool     string         `json:"tool,omitempty"`
	Message  string         `json:"message,omitempty"`
	Progress map[string]int `json:"progress,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
	Error    *Error         `json:"error,omitempty"`
}

type EventSink interface {
	Emit(Event)
}

type Error struct {
	Code           string         `json:"code"`
	Message        string         `json:"message"`
	Retryable      bool           `json:"retryable"`
	ProviderStatus string         `json:"provider_status,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
