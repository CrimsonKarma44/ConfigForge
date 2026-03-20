package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/deus/configforge/internal/config"
	"github.com/deus/configforge/internal/llm"
	"github.com/deus/configforge/internal/plugin"
)

type CLI struct {
	registry    *plugin.Registry
	config      *config.Config
	llm         *llm.LLM
	hotReloader *plugin.HotReloader
}

func Run(args []string) error {
	c := &CLI{
		registry: plugin.NewRegistry(),
	}

	if err := c.parseFlags(args); err != nil {
		return err
	}

	if err := c.init(); err != nil {
		return err
	}

	if c.hotReloader != nil {
		defer c.hotReloader.Stop()
	}

	return c.run()
}

func (c *CLI) parseFlags(args []string) error {
	flags := flag.NewFlagSet("configforge", flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Println(`ConfigForge - Generate configuration files for your projects

Usage: configforge [options] [plugin]

Options:
  --list          List available plugins
  --stack <s>     Filter plugins by stack (go, python, node, rust)
  --output <d>    Output directory (default: current directory)
  --llm           Use LLM for enhanced generation (hybrid mode)
  --model <m>     LLM model to use
  --provider <p>  LLM provider: openai, anthropic, google, ollama
  --watch         Enable hot-reload for external plugins
  --help          Show this help message

Examples:
  configforge                    # Interactive mode
  configforge --list             # List all plugins
  configforge --stack go         # List Go-related plugins
  configforge docker             # Generate docker configs
  configforge --llm python       # Generate with LLM assistance
  configforge --llm --provider anthropic --model claude-3-opus python
  configforge --watch            # Run with hot-reload enabled`)
	}

	var list bool
	var stack string
	var output string
	var useLLM bool
	var model string
	var provider string
	var watch bool

	flags.BoolVar(&list, "list", false, "List plugins")
	flags.StringVar(&stack, "stack", "", "Filter by stack")
	flags.StringVar(&output, "output", "", "Output directory")
	flags.BoolVar(&useLLM, "llm", false, "Use LLM")
	flags.StringVar(&model, "model", "gpt-4o-mini", "LLM model")
	flags.StringVar(&provider, "provider", "", "LLM provider")
	flags.BoolVar(&watch, "watch", false, "Enable hot-reload")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	c.config = &config.Config{
		List:       list,
		Stack:      stack,
		OutputDir:  output,
		UseLLM:     useLLM,
		Model:      model,
		Provider:   provider,
		PluginName: flags.Arg(0),
		Watch:      watch,
	}

	return nil
}

func (c *CLI) init() error {
	dirs := plugin.GetDefaultPluginDirs()
	loader := plugin.NewLoader(dirs)
	if err := loader.LoadFromBuiltins(c.registry); err != nil {
		return fmt.Errorf("load builtins: %w", err)
	}

	if err := loader.LoadExternalPlugins(c.registry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load external plugins: %v\n", err)
	}

	if c.config.Watch {
		reloader := plugin.NewHotReloader(loader, c.registry, dirs, 2*time.Second)
		if err := reloader.Start(); err != nil {
			return fmt.Errorf("start hot reload: %w", err)
		}
		c.hotReloader = reloader
	}

	if c.config.UseLLM {
		model := c.config.Model
		if c.config.Provider != "" {
			model = c.config.Provider + "/" + model
		}
		l, err := llm.New(model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: LLM not available: %v\n", err)
		} else {
			c.llm = l
		}
	}

	return nil
}

func (c *CLI) run() error {
	if c.config.List {
		return c.listPlugins()
	}

	if c.config.PluginName == "" {
		return c.interactiveMode()
	}

	return c.generateForPlugin(c.config.PluginName)
}

func (c *CLI) listPlugins() error {
	var plugins []plugin.ConfigPlugin

	if c.config.Stack != "" {
		plugins = c.registry.ListByStack(c.config.Stack)
	} else {
		plugins = c.registry.List()
	}

	if len(plugins) == 0 {
		fmt.Println("No plugins found")
		return nil
	}

	fmt.Println("Available plugins:")
	for _, p := range plugins {
		fmt.Printf("  %-20s %s\n", p.Name()+":", p.Description())
		if c.config.Stack == "" {
			fmt.Printf("                       Stacks: %v\n", p.SupportedStacks())
		}
	}
	return nil
}

func (c *CLI) interactiveMode() error {
	fmt.Println("ConfigForge - Interactive Config Generator")
	fmt.Println("============================================")

	var plugins []plugin.ConfigPlugin
	if c.config.Stack != "" {
		plugins = c.registry.ListByStack(c.config.Stack)
	} else {
		plugins = c.registry.List()
	}

	if len(plugins) == 0 {
		return fmt.Errorf("no plugins available")
	}

	fmt.Println("Available plugins:")
	for i, p := range plugins {
		fmt.Printf("  %d. %s - %s\n", i+1, p.Name(), p.Description())
	}

	fmt.Print("\nSelect a plugin (number): ")
	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return err
	}

	var selection int
	if _, err := fmt.Sscanf(string(line), "%d", &selection); err != nil || selection < 1 || selection > len(plugins) {
		return fmt.Errorf("invalid selection")
	}

	selected := plugins[selection-1]
	return c.generateForPlugin(selected.Name())
}

func (c *CLI) generateForPlugin(name string) error {
	p, ok := c.registry.Get(name)
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	answers, err := c.promptQuestions(p)
	if err != nil {
		return err
	}

	ctx := context.Background()
	content, err := p.Generate(ctx, answers)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if c.llm != nil && c.config.UseLLM {
		fmt.Printf("\n🤖 Applying LLM enhancement (%s)...\n", c.llm.Name())
		enhanced, err := c.llm.Enhance(ctx, name, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: LLM enhancement failed: %v\n", err)
		} else {
			content = enhanced
			fmt.Println("✅ LLM enhancement complete")
		}
	}

	return c.writeOutput(p, content)
}

func (c *CLI) promptQuestions(p plugin.ConfigPlugin) (map[string]interface{}, error) {
	questions := p.Questions()
	if len(questions) == 0 {
		return make(map[string]interface{}), nil
	}

	answers := make(map[string]interface{})
	reader := bufio.NewReader(os.Stdin)

	for _, q := range questions {
		fmt.Printf("%s", q.Description)
		if q.Required {
			fmt.Printf(" *")
		}
		if len(q.Options) > 0 {
			fmt.Printf(" (%s)", strings.Join(q.Options, ", "))
		}
		if q.Default != "" {
			fmt.Printf(" [default: %s]", q.Default)
		}
		fmt.Printf(": ")

		line, _, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}

		answer := strings.TrimSpace(string(line))
		if answer == "" {
			answer = q.Default
		}

		if q.Required && answer == "" {
			return nil, fmt.Errorf("required: %s", q.Name)
		}

		answers[q.Name] = answer
	}

	return answers, nil
}

func (c *CLI) writeOutput(p plugin.ConfigPlugin, content string) error {
	dir := c.config.OutputDir
	if dir == "" {
		dir = "."
	}

	files := p.Files()
	if len(files) == 0 {
		files = []string{p.Name() + ".config"}
	}

	outputs := strings.Split(content, "\n---\n")
	for i, filename := range files {
		path := dir + "/" + filename

		var contentToWrite string
		if i < len(outputs) {
			contentToWrite = strings.TrimSpace(outputs[i])
		} else if len(outputs) == 1 {
			contentToWrite = strings.TrimSpace(outputs[0])
		} else {
			contentToWrite = ""
		}

		if contentToWrite == "" {
			continue
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}

		if err := os.WriteFile(path, []byte(contentToWrite), 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}

		fmt.Printf("Created: %s\n", path)
	}

	return nil
}

func init() {
	sort.Strings(nil)
}
