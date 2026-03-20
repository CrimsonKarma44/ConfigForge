package config

type Config struct {
	List       bool
	Stack      string
	OutputDir  string
	UseLLM     bool
	Model      string
	Provider   string
	PluginName string
	Watch      bool
}
