package plugin

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

type NodePlugin struct{}

func (p *NodePlugin) Name() string              { return "node" }
func (p *NodePlugin) Description() string       { return "Node.js package.json configuration" }
func (p *NodePlugin) SupportedStacks() []string { return []string{"node"} }
func (p *NodePlugin) Files() []string           { return []string{"package.json"} }

func (p *NodePlugin) Questions() []*Question {
	return []*Question{
		{
			Name:        "projectName",
			Description: "Project name",
			Type:        "text",
			Required:    true,
		},
		{
			Name:        "version",
			Description: "Initial version",
			Type:        "text",
			Default:     "0.1.0",
			Required:    false,
		},
		{
			Name:        "description",
			Description: "Project description",
			Type:        "text",
			Required:    false,
		},
		{
			Name:        "author",
			Description: "Author name",
			Type:        "text",
			Required:    false,
		},
		{
			Name:        "license",
			Description: "License",
			Type:        "select",
			Options:     []string{"MIT", "Apache-2.0", "GPL-3.0", "ISC", "UNLICENSE"},
			Default:     "MIT",
			Required:    false,
		},
		{
			Name:        "main",
			Description: "Main entry point",
			Type:        "text",
			Default:     "index.js",
			Required:    false,
		},
		{
			Name:        "scripts",
			Description: "npm scripts (e.g., start:node index.js, test:jest)",
			Type:        "text",
			Required:    false,
		},
		{
			Name:        "dependencies",
			Description: "Dependencies (name@version, comma separated)",
			Type:        "text",
			Required:    false,
		},
		{
			Name:        "devDependencies",
			Description: "Dev dependencies (name@version, comma separated)",
			Type:        "text",
			Required:    false,
		},
	}
}

func (p *NodePlugin) Generate(ctx context.Context, answers map[string]interface{}) (string, error) {
	projectName := getString(answers, "projectName", "my-project")
	version := getString(answers, "version", "0.1.0")
	description := getString(answers, "description", "")
	author := getString(answers, "author", "")
	license := getString(answers, "license", "MIT")
	mainEntry := getString(answers, "main", "index.js")
	scriptsStr := getString(answers, "scripts", "")
	deps := getString(answers, "dependencies", "")
	devDeps := getString(answers, "devDependencies", "")

	tmpl := `{
  "name": "{{.ProjectName}}",
  "version": "{{.Version}}",
  "description": "{{.Description}}",
  "main": "{{.Main}}",
  "scripts": {
    {{.Scripts}}
  },
  "keywords": [],
  "author": "{{.Author}}",
  "license": "{{.License}}",
  "dependencies": {
    {{.Dependencies}}
  },
  "devDependencies": {
    {{.DevDependencies}}
  }
}
`

	scripts := parseNpmScripts(scriptsStr)
	depsMap := parseNpmDeps(deps)
	devDepsMap := parseNpmDeps(devDeps)

	t, err := template.New("package").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]interface{}{
		"ProjectName":     projectName,
		"Version":         version,
		"Description":     description,
		"Main":            mainEntry,
		"Scripts":         scripts,
		"Author":          author,
		"License":         license,
		"Dependencies":    depsMap,
		"DevDependencies": devDepsMap,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *NodePlugin) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("empty package.json content")
	}
	return nil
}

func parseNpmScripts(s string) []string {
	if s == "" {
		return []string{`"start": "node index.js"`}
	}
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return []string{`"start": "node index.js"`}
	}
	return result
}

func parseNpmDeps(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, `"`+item+`"`)
		}
	}
	return result
}
