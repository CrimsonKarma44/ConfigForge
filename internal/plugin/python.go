package plugin

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

type PythonPlugin struct{}

func (p *PythonPlugin) Name() string              { return "python" }
func (p *PythonPlugin) Description() string       { return "Python project configuration (pyproject.toml)" }
func (p *PythonPlugin) SupportedStacks() []string { return []string{"python"} }
func (p *PythonPlugin) Files() []string           { return []string{"pyproject.toml"} }

func (p *PythonPlugin) Questions() []*Question {
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
			Name:        "authorEmail",
			Description: "Author email",
			Type:        "text",
			Required:    false,
		},
		{
			Name:        "license",
			Description: "License",
			Type:        "select",
			Options:     []string{"MIT", "Apache-2.0", "GPL-3.0", "BSD-3-Clause", "UNLICENSE"},
			Default:     "MIT",
			Required:    false,
		},
		{
			Name:        "dependencies",
			Description: "Core dependencies (comma separated)",
			Type:        "text",
			Required:    false,
		},
		{
			Name:        "devDependencies",
			Description: "Dev dependencies (comma separated)",
			Type:        "text",
			Required:    false,
		},
	}
}

func (p *PythonPlugin) Generate(ctx context.Context, answers map[string]interface{}) (string, error) {
	projectName := getString(answers, "projectName", "my-project")
	version := getString(answers, "version", "0.1.0")
	description := getString(answers, "description", "")
	author := getString(answers, "author", "")
	authorEmail := getString(answers, "authorEmail", "")
	license := getString(answers, "license", "MIT")
	deps := getString(answers, "dependencies", "")
	devDeps := getString(answers, "devDependencies", "")

	tmpl := `[project]
name = "{{.ProjectName}}"
version = "{{.Version}}"
description = "{{.Description}}"
readme = "README.md"
requires-python = ">=3.9"
license = {text = "{{.License}}"}
authors = [
    {name = "{{.Author}}", email = "{{.AuthorEmail}}"}
]
dependencies = [
{{range $i, $dep := .Dependencies}}    "{{$dep}}"{{if notLast $i $.Dependencies}},{{end}}
{{end}}]

[project.optional-dependencies]
dev = [
{{range $i, $dep := .DevDependencies}}    "{{$dep}}"{{if notLast $i $.DevDependencies}},{{end}}
{{end}}]

[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = ["test_*.py"]
python_functions = ["test_*"]

[tool.black]
line-length = 88
target-version = ['py39']

[tool.isort]
profile = "black"
line_length = 88
`

	t, err := template.New("pyproject").
		Funcs(template.FuncMap{
			"notLast": notLast,
		}).
		Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]interface{}{
		"ProjectName":     projectName,
		"Version":         version,
		"Description":     description,
		"Author":          author,
		"AuthorEmail":     authorEmail,
		"License":         license,
		"Dependencies":    parseCommaList(deps),
		"DevDependencies": parseCommaList(devDeps),
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *PythonPlugin) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("empty pyproject content")
	}
	return nil
}
