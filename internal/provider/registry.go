package provider

import (
	"fmt"
	"sort"
)

type Registry struct {
	providers map[string]Provider
	tools     map[string]RegisteredTool
}

type RegisteredTool struct {
	Provider Provider
	Tool     Tool
}

func NewRegistry(providers ...Provider) (*Registry, error) {
	r := &Registry{
		providers: map[string]Provider{},
		tools:     map[string]RegisteredTool{},
	}
	for _, p := range providers {
		if p.ID() == "" {
			return nil, fmt.Errorf("provider id is required")
		}
		if _, exists := r.providers[p.ID()]; exists {
			return nil, fmt.Errorf("duplicate provider %q", p.ID())
		}
		r.providers[p.ID()] = p
		for _, tool := range p.Tools() {
			id := ToolID(p.ID(), tool.ID())
			if _, exists := r.tools[id]; exists {
				return nil, fmt.Errorf("duplicate tool %q", id)
			}
			r.tools[id] = RegisteredTool{Provider: p, Tool: tool}
		}
	}
	return r, nil
}

func ToolID(providerID, toolID string) string {
	return providerID + "." + toolID
}

func (r *Registry) Providers() []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

func (r *Registry) Provider(id string) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

func (r *Registry) Tool(providerID, toolID string) (RegisteredTool, bool) {
	rt, ok := r.tools[ToolID(providerID, toolID)]
	return rt, ok
}

func (r *Registry) ToolByID(id string) (RegisteredTool, bool) {
	rt, ok := r.tools[id]
	return rt, ok
}

func (r *Registry) Tools() []RegisteredTool {
	out := make([]RegisteredTool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return ToolID(out[i].Provider.ID(), out[i].Tool.ID()) < ToolID(out[j].Provider.ID(), out[j].Tool.ID())
	})
	return out
}
