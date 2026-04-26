package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ultimatemonty/media-perms-manager/internal/config"
	"github.com/ultimatemonty/media-perms-manager/internal/fixer"
	"github.com/ultimatemonty/media-perms-manager/internal/logger"
	"github.com/ultimatemonty/media-perms-manager/internal/watcher"
)

func main() {
	var (
		cfgPath          string
		rootsFlag        string
		fileMode         string
		dirMode          string
		owner            string
		group            string
		excludes         string
		followLinks      bool
		dryRun           bool
		concurrency      int
		logLevel         string
		watch            bool
		watchDelay       time.Duration
		watchInitialScan bool
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
	flag.BoolVar(&watch, "watch", false, "enable watch mode (continuously watch for changes)")
	flag.DurationVar(&watchDelay, "watch-delay", 500*time.Millisecond, "debounce delay for rapid fs events")
	flag.BoolVar(&watchInitialScan, "watch-initial-scan", false, "run full scan before entering watch mode")
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
	if flagPassed("follow-symlinks") {
		cfg.FollowSymlinks = followLinks
	}
	if flagPassed("dry-run") {
		cfg.DryRun = dryRun
	}
	if flagPassed("concurrency") && concurrency > 0 {
		cfg.Concurrency = concurrency
	}
	if flagPassed("log-level") {
		cfg.LogLevel = logLevel
	}
	if flagPassed("watch") {
		cfg.Watch = watch
	}
	if flagPassed("watch-delay") && watchDelay > 0 {
		cfg.WatchDelay = watchDelay
	}
	if flagPassed("watch-initial-scan") {
		cfg.WatchInitialScan = watchInitialScan
	}

	// Initialize logger
	logg := logger.New(cfg.LogLevel)

	logg.Info("starting permission-fixer")

	fx, err := fixer.New(cfg, logg)
	if err != nil {
		log.Fatalf("failed to initialize fixer: %v", err)
	}

	if cfg.Watch {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if cfg.WatchInitialScan {
			logg.Info("running initial scan before watch mode")
			if err := fx.Run(ctx); err != nil {
				log.Fatalf("initial scan failed: %v", err)
			}
		}

		wt, err := watcher.New(cfg.Roots, fx, logg, cfg.WatchDelay, cfg.Excludes, cfg.FollowSymlinks)
		if err != nil {
			log.Fatalf("failed to initialize watcher: %v", err)
		}

		logg.Info("watch mode enabled")
		if err := wt.Watch(ctx); err != nil {
			log.Fatalf("watch failed: %v", err)
		}
		logg.Info("watch mode stopped")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	if err := fx.Run(ctx); err != nil {
		log.Fatalf("scan run failed: %v", err)
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

func flagPassed(name string) bool {
	found := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
