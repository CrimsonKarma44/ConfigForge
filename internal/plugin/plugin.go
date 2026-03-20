package plugin

import (
	"context"
	"fmt"
)

type Question struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // text, select, multiselect, confirm
	Options     []string `json:"options,omitempty"`
	Default     string   `json:"default,omitempty"`
	Required    bool     `json:"required"`
}

type ConfigPlugin interface {
	Name() string
	Description() string
	SupportedStacks() []string
	Questions() []*Question
	Generate(ctx context.Context, answers map[string]interface{}) (string, error)
	Validate(content string) error
	Files() []string
}

type Registry struct {
	plugins       map[string]ConfigPlugin
	externalPaths map[string]bool
}

func NewRegistry() *Registry {
	return &Registry{
		plugins:       make(map[string]ConfigPlugin),
		externalPaths: make(map[string]bool),
	}
}

func (r *Registry) Register(p ConfigPlugin) error {
	if p == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	r.plugins[name] = p
	return nil
}

func (r *Registry) RegisterExternal(p ConfigPlugin) error {
	if err := r.Register(p); err != nil {
		return err
	}
	r.externalPaths[p.Name()] = true
	return nil
}

func (r *Registry) ClearExternal() {
	for name := range r.plugins {
		if r.externalPaths[name] {
			delete(r.plugins, name)
			delete(r.externalPaths, name)
		}
	}
}

func (r *Registry) Get(name string) (ConfigPlugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) List() []ConfigPlugin {
	plugins := make([]ConfigPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (r *Registry) ListByStack(stack string) []ConfigPlugin {
	var result []ConfigPlugin
	for _, p := range r.plugins {
		for _, s := range p.SupportedStacks() {
			if s == stack {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
