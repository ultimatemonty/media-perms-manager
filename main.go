package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ultimatemonty/media-perms-manager/internal/config"
	"github.com/ultimatemonty/media-perms-manager/internal/fixer"
	"github.com/ultimatemonty/media-perms-manager/internal/logger"
)

func main() {
	var (
		cfgPath     string
		rootsFlag   string
		fileMode    string
		dirMode     string
		owner       string
		group       string
		excludes    string
		followLinks bool
		dryRun      bool
		concurrency int
		logLevel    string
	)

	flag.StringVar(&cfgPath, "config", "", "path to YAML config file (overridden by flags/env)")
	flag.StringVar(&rootsFlag, "roots", "", "comma-separated list of root directories to walk")
	flag.StringVar(&fileMode, "file-mode", "", "desired file mode (octal, e.g. 0775)")
	flag.StringVar(&dirMode, "dir-mode", "", "desired dir mode (octal, e.g. 0775)")
	flag.StringVar(&owner, "owner", "", "owner username or uid")
	flag.StringVar(&group, "group", "", "group name or gid")
	flag.StringVar(&excludes, "exclude", "", "comma-separated glob patterns to exclude (repeatable as comma list)")
	flag.BoolVar(&followLinks, "follow-symlinks", false, "follow symlinks")
	flag.BoolVar(&dryRun, "dry-run", false, "do not apply changes; just log what would change")
	flag.IntVar(&concurrency, "concurrency", 4, "number of worker goroutines")
	flag.StringVar(&logLevel, "log-level", "info", "log level: debug|info|warn|error")
	flag.Parse()

	// Load YAML config (with defaults)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(2)
	}

	// Overlay environment variables
	cfg.OverrideFromEnv()

	// Overlay flags (flags take precedence)
	if rootsFlag != "" {
		cfg.Roots = splitAndTrim(rootsFlag)
	}
	if fileMode != "" {
		cfg.FileMode = fileMode
	}
	if dirMode != "" {
		cfg.DirMode = dirMode
	}
	if owner != "" {
		cfg.Owner = owner
	}
	if group != "" {
		cfg.Group = group
	}
	if excludes != "" {
		cfg.Excludes = splitAndTrim(excludes)
	}
	cfg.FollowSymlinks = followLinks
	cfg.DryRun = dryRun
	if concurrency > 0 {
		cfg.Concurrency = concurrency
	}
	cfg.LogLevel = logLevel

	// Initialize logger
	logg := logger.New(cfg.LogLevel)

	logg.Info("starting permission-fixer")

	fx, err := fixer.New(cfg, logg)
	if err != nil {
		log.Fatalf("failed to initialize fixer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	if err := fx.Run(ctx); err != nil {
		log.Fatalf("run failed: %v", err)
	}

	logg.Info("completed")
}

func splitAndTrim(s string) []string {
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
