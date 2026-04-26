package fixer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ultimatemonty/media-perms-manager/internal/config"
	"github.com/ultimatemonty/media-perms-manager/internal/logger"
)

func TestRunHonorsCanceledContext(t *testing.T) {
	tmp := t.TempDir()
	for i := 0; i < 200; i++ {
		p := filepath.Join(tmp, fmt.Sprintf("file-%d.txt", i))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
	}

	cfg := config.Config{
		Roots:          []string{tmp},
		FileMode:       "0644",
		DirMode:        "0755",
		Owner:          fmt.Sprintf("%d", os.Getuid()),
		Group:          fmt.Sprintf("%d", os.Getgid()),
		Excludes:       []string{},
		FollowSymlinks: false,
		DryRun:         true,
		Concurrency:    1,
		LogLevel:       "error",
	}

	fx, err := New(cfg, logger.New("error"))
	if err != nil {
		t.Fatalf("new fixer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = fx.Run(ctx)
	if err == nil {
		t.Fatalf("expected canceled context error, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
