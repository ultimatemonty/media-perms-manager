// Package config provides configuration loading and management for the media permissions manager.
// It supports loading from YAML files and overriding with environment variables.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration loaded from file or environment variables.
type Config struct {
	Roots            []string      `yaml:"roots"`
	FileMode         string        `yaml:"file_mode"`
	DirMode          string        `yaml:"dir_mode"`
	Owner            string        `yaml:"owner"`
	Group            string        `yaml:"group"`
	Excludes         []string      `yaml:"excludes"`
	FollowSymlinks   bool          `yaml:"follow_symlinks"`
	DryRun           bool          `yaml:"dry_run"`
	Concurrency      int           `yaml:"concurrency"`
	LogLevel         string        `yaml:"log_level"`
	Watch            bool          `yaml:"watch"`              // enable watch mode
	WatchDelay       time.Duration `yaml:"watch_delay"`        // Debounce delay for rapid events (e.g., 500ms)
	WatchInitialScan bool          `yaml:"watch_initial_scan"` // Run full scan before watching
}

// Load loads config from the given path. If path is empty it will try ./config.yaml
// and then $XDG_CONFIG_HOME/mpm/config.yaml. Missing file returns defaults.
func Load(path string) (Config, error) {
	cfg := defaultConfig()
	var candidates []string
	if path != "" {
		candidates = []string{path}
	} else {
		candidates = []string{"./config.yaml"}
		if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
			candidates = append(candidates, filepath.Join(x, "mpm", "config.yaml"))
		} else {
			// fallback to ~/.config
			if h := os.Getenv("HOME"); h != "" {
				candidates = append(candidates, filepath.Join(h, ".config", "mpm", "config.yaml"))
			}
		}
	}

	var data []byte
	for _, c := range candidates {
		b, err := os.ReadFile(c)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			// other read errors
			return cfg, err
		}
		data = b
		break
	}
	if data != nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	}

	// validate/normalize
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.FileMode == "" {
		cfg.FileMode = "0775"
	}
	if cfg.DirMode == "" {
		cfg.DirMode = "0775"
	}
	if len(cfg.Roots) == 0 {
		cfg.Roots = []string{"."}
	}
	if cfg.Owner == "" {
		cfg.Owner = "media"
	}
	if cfg.Group == "" {
		cfg.Group = "jellyfin"
	}
	if len(cfg.Excludes) == 0 {
		cfg.Excludes = []string{"*.trickplay"}
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.WatchDelay <= 0 {
		cfg.WatchDelay = 500 * time.Millisecond
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Roots:            []string{"."},
		FileMode:         "0775",
		DirMode:          "0775",
		Owner:            "media",
		Group:            "jellyfin",
		Excludes:         []string{"*.trickplay"},
		FollowSymlinks:   false,
		DryRun:           false,
		Concurrency:      4,
		LogLevel:         "info",
		Watch:            false,
		WatchDelay:       500 * time.Millisecond,
		WatchInitialScan: false,
	}
}

// OverrideFromEnv overlays environment variables if present.
func (c *Config) OverrideFromEnv() {
	if v := os.Getenv("MPM_ROOTS"); v != "" {
		c.Roots = splitCSV(v)
	}
	if v := os.Getenv("MPM_FILE_MODE"); v != "" {
		c.FileMode = v
	}
	if v := os.Getenv("MPM_DIR_MODE"); v != "" {
		c.DirMode = v
	}
	if v := os.Getenv("MPM_OWNER"); v != "" {
		c.Owner = v
	}
	if v := os.Getenv("MPM_GROUP"); v != "" {
		c.Group = v
	}
	if v := os.Getenv("MPM_EXCLUDES"); v != "" {
		c.Excludes = splitCSV(v)
	}
	if v := os.Getenv("MPM_FOLLOW_SYMLINKS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.FollowSymlinks = b
		}
	}
	if v := os.Getenv("MPM_DRY_RUN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.DryRun = b
		}
	}
	if v := os.Getenv("MPM_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.Concurrency = n
		}
	}
	if v := os.Getenv("MPM_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("MPM_WATCH"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Watch = b
		}
	}
	if v := os.Getenv("MPM_WATCH_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.WatchDelay = d
		}
	}
	if v := os.Getenv("MPM_WATCH_INITIAL_SCAN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.WatchInitialScan = b
		}
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseMode parses an octal mode string like "0755" into uint32
func ParseMode(s string) (uint32, error) {
	if s == "" {
		return 0, errors.New("empty mode")
	}
	// ensure it doesn't get parsed as decimal by strconv
	if strings.HasPrefix(s, "0") {
		v, err := strconv.ParseUint(s, 8, 32)
		return uint32(v), err
	}
	// try parse as octal anyway
	v, err := strconv.ParseUint(s, 8, 32)
	return uint32(v), err
}
