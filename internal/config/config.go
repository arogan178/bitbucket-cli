// Package config manages `bt` on-disk configuration: contexts that bind a
// host, backend kind (cloud/dc), default workspace/project/repo, and the
// currently active context. Configuration lives at
// $XDG_CONFIG_HOME/bt/config.yaml (falling back to ~/.config/bt/config.yaml).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Kind identifies which Bitbucket backend a context targets.
type Kind string

const (
	KindCloud      Kind = "cloud"
	KindDataCenter Kind = "dc"
)

// Context is one named Bitbucket endpoint + defaults.
type Context struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Kind      Kind   `yaml:"kind"`
	Workspace string `yaml:"workspace,omitempty"`
	Project   string `yaml:"project,omitempty"`
	Repo      string `yaml:"repo,omitempty"`
	// Username is retained in plaintext to label credentials; the secret
	// lives in the OS keyring (see internal/auth).
	Username string `yaml:"username,omitempty"`
}

// Config is the top-level on-disk shape.
type Config struct {
	Active   string    `yaml:"active,omitempty"`
	Contexts []Context `yaml:"contexts,omitempty"`
}

// ErrNoActive means no context is selected.
var ErrNoActive = errors.New("no active context; run `bt auth login` or `bt context use <name>`")

// Path returns the resolved config file path. It honours BT_CONFIG_DIR,
// then XDG_CONFIG_HOME, then ~/.config.
func Path() (string, error) {
	if override := os.Getenv("BT_CONFIG_DIR"); override != "" {
		return filepath.Join(override, "config.yaml"), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bt", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "bt", "config.yaml"), nil
}

// Load reads config from disk. A missing file yields an empty Config.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// Save atomically writes the config to disk, creating parent dirs as needed.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Find returns a pointer to the named context, or nil if absent.
func (c *Config) Find(name string) *Context {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i]
		}
	}
	return nil
}

// Upsert inserts or replaces a context by name.
func (c *Config) Upsert(ctx Context) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == ctx.Name {
			c.Contexts[i] = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, ctx)
}

// Delete removes a context by name. Returns true if anything was removed.
func (c *Config) Delete(name string) bool {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			if c.Active == name {
				c.Active = ""
			}
			return true
		}
	}
	return false
}

// ActiveContext resolves the active context, honouring the BT_CONTEXT env
// override and the --context flag (caller passes empty string when no flag).
func (c *Config) ActiveContext(override string) (*Context, error) {
	name := override
	if name == "" {
		name = os.Getenv("BT_CONTEXT")
	}
	if name == "" {
		name = c.Active
	}
	if name == "" {
		// Single-context installs don't need a selection.
		if len(c.Contexts) == 1 {
			return &c.Contexts[0], nil
		}
		return nil, ErrNoActive
	}
	ctx := c.Find(name)
	if ctx == nil {
		return nil, fmt.Errorf("context %q not found", name)
	}
	return ctx, nil
}
