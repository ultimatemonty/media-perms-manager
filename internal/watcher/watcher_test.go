package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ultimatemonty/media-perms-manager/internal/logger"
)

func TestHandleEventCreateDirectoryQueuesExistingChildren(t *testing.T) {
	tmp := t.TempDir()
	newDir := filepath.Join(tmp, "newdir")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	child := filepath.Join(newDir, "child.txt")
	if err := os.WriteFile(child, []byte("data"), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	defer func() { _ = fsw.Close() }()

	w := &Watch{
		logger:       logger.New("error"),
		debounce:     50 * time.Millisecond,
		followLinks:  false,
		fswatch:      fsw,
		pendingPaths: make(map[string]time.Time),
	}

	if err := w.addRecursive(tmp); err != nil {
		t.Fatalf("add recursive root: %v", err)
	}

	w.handleEvent(fsnotify.Event{Name: newDir, Op: fsnotify.Create})

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.pendingPaths[child]; !ok {
		t.Fatalf("expected child path to be queued for processing")
	}
}

func TestNewSkipsInvalidRootWhenOthersAreValid(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist")

	w, err := New(
		[]string{missing, tmp},
		nil,
		logger.New("error"),
		50*time.Millisecond,
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("expected watcher creation to succeed with at least one valid root, got: %v", err)
	}
	defer func() { _ = w.fswatch.Close() }()
}

func TestNewFailsWhenAllRootsInvalid(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist")

	w, err := New(
		[]string{missing},
		nil,
		logger.New("error"),
		50*time.Millisecond,
		nil,
		false,
	)
	if err == nil {
		if w != nil {
			_ = w.fswatch.Close()
		}
		t.Fatalf("expected watcher creation to fail when all roots are invalid")
	}
}
