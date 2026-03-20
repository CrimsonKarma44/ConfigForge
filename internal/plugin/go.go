package plugin

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

type GoPlugin struct{}

func (p *GoPlugin) Name() string              { return "go" }
func (p *GoPlugin) Description() string       { return "Go module configuration" }
func (p *GoPlugin) SupportedStacks() []string { return []string{"go"} }
func (p *GoPlugin) Files() []string           { return []string{"go.mod"} }

func (p *GoPlugin) Questions() []*Question {
	return []*Question{
		{
			Name:        "moduleName",
			Description: "Module name (e.g., github.com/user/project)",
			Type:        "text",
			Required:    true,
		},
		{
			Name:        "goVersion",
			Description: "Go version",
			Type:        "text",
			Default:     "1.21",
			Required:    false,
		},
		{
			Name:        "dependencies",
			Description: "Dependencies (module@version, comma separated)",
			Type:        "text",
			Required:    false,
		},
	}
}

func (p *GoPlugin) Generate(ctx context.Context, answers map[string]interface{}) (string, error) {
	moduleName := getString(answers, "moduleName", "github.com/user/project")
	goVersion := getString(answers, "goVersion", "1.21")
	deps := getString(answers, "dependencies", "")

	tmpl := `module {{.ModuleName}}

go {{.GoVersion}}

require (
{{range $i, $dep := .Dependencies}}	{{$dep}}
{{end}})
`

	t, err := template.New("gomod").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]interface{}{
		"ModuleName":   moduleName,
		"GoVersion":    goVersion,
		"Dependencies": parseCommaList(deps),
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *GoPlugin) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("empty go.mod content")
	}
	return nil
}
