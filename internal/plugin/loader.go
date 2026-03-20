package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"
)

type Loader struct {
	pluginDirs []string
}

type HotReloader struct {
	loader         *Loader
	registry       *Registry
	watchDirs      []string
	interval       time.Duration
	stopCh         chan struct{}
	mu             sync.RWMutex
	pluginModTimes map[string]int64
}

func NewLoader(dirs []string) *Loader {
	return &Loader{
		pluginDirs: dirs,
	}
}

func (l *Loader) LoadFromBuiltins(registry *Registry) error {
	registry.Register(&DockerPlugin{})
	registry.Register(&GitHubActionsPlugin{})
	registry.Register(&PythonPlugin{})
	registry.Register(&GoPlugin{})
	registry.Register(&NodePlugin{})
	return nil
}

func (l *Loader) LoadExternalPlugins(registry *Registry) error {
	for _, dir := range l.pluginDirs {
		if err := l.loadPluginsFromDir(dir, registry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load plugins from %s: %v\n", dir, err)
		}
	}
	return nil
}

func (l *Loader) loadPluginsFromDir(dir string, registry *Registry) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			path := filepath.Join(dir, entry.Name())
			if err := l.loadPlugin(path, registry); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", entry.Name(), err)
			}
		}
	}
	return nil
}

func (l *Loader) loadPlugin(path string, registry *Registry) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("open plugin: %w", err)
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("lookup Plugin symbol: %w", err)
	}

	plug, ok := sym.(ConfigPlugin)
	if !ok {
		return fmt.Errorf("Plugin does not implement ConfigPlugin interface")
	}

	return registry.RegisterExternal(plug)
}

func GetDefaultPluginDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"./plugins",
		filepath.Join(home, ".config", "configforge", "plugins"),
	}
}

func NewHotReloader(loader *Loader, registry *Registry, dirs []string, interval time.Duration) *HotReloader {
	return &HotReloader{
		loader:         loader,
		registry:       registry,
		watchDirs:      dirs,
		interval:       interval,
		stopCh:         make(chan struct{}),
		pluginModTimes: make(map[string]int64),
	}
}

func (hr *HotReloader) Start() error {
	hr.reloadPlugins()

	go hr.watchLoop()
	fmt.Println("Hot reload enabled: watching for plugin changes...")
	return nil
}

func (hr *HotReloader) Stop() {
	close(hr.stopCh)
}

func (hr *HotReloader) watchLoop() {
	ticker := time.NewTicker(hr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hr.stopCh:
			return
		case <-ticker.C:
			hr.checkForChanges()
		}
	}
}

func (hr *HotReloader) checkForChanges() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	needsReload := false

	for _, dir := range hr.watchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}

		currentFiles := make(map[string]int64)

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
				path := filepath.Join(dir, entry.Name())
				info, err := os.Stat(path)
				if err != nil {
					continue
				}

				currentFiles[path] = info.ModTime().UnixNano()

				prevTime, exists := hr.pluginModTimes[path]
				if !exists || info.ModTime().UnixNano() > prevTime {
					needsReload = true
				}
			}
		}

		for path := range hr.pluginModTimes {
			if _, exists := currentFiles[path]; !exists {
				needsReload = true
			}
		}

		hr.pluginModTimes = currentFiles
	}

	if needsReload {
		hr.reloadPlugins()
	}
}

func (hr *HotReloader) reloadPlugins() {
	hr.registry.ClearExternal()

	for _, dir := range hr.watchDirs {
		if err := hr.loader.loadPluginsFromDir(dir, hr.registry); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load plugins from %s: %v\n", dir, err)
		}
	}

	fmt.Println("Plugins reloaded")
}
