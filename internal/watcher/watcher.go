// Package watcher provides filesystem watch mode for continuously applying
// ownership and permissions to changed paths under configured roots.
package watcher

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ultimatemonty/media-perms-manager/internal/fixer"
	"github.com/ultimatemonty/media-perms-manager/internal/logger"
)

// Watch encapsulates fsnotify-based monitoring state and debounce buffers.
type Watch struct {
	roots        []string
	fixer        *fixer.Fixer // reuse existing fixer logic for applying fixes on events
	logger       *logger.Logger
	debounce     time.Duration
	excludes     []string
	followLinks  bool
	fswatch      *fsnotify.Watcher
	processQueue chan string
	workers      int
	pendingPaths map[string]time.Time // for debounce tracking
	mu           sync.Mutex           // protects pendingPaths
}

// New initializes a debounced filesystem watcher across all provided roots.
func New(roots []string, f *fixer.Fixer, l *logger.Logger, debounce time.Duration, excludes []string, followLinks bool) (*Watch, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watch{
		roots:        roots,
		fixer:        f,
		logger:       l,
		debounce:     debounce,
		excludes:     excludes,
		followLinks:  followLinks,
		fswatch:      fsw,
		processQueue: make(chan string, 1024),
		workers:      max(1, runtime.NumCPU()),
		pendingPaths: make(map[string]time.Time),
	}

	watchableRoots := 0
	for _, root := range roots {
		if err := w.addRecursive(root); err != nil {
			w.logger.Warnf("watch root %s skipped: %v", root, err)
			continue
		}
		watchableRoots++
		w.logger.Infof("watching root %s", root)
	}

	if watchableRoots == 0 {
		_ = fsw.Close()
		return nil, fmt.Errorf("no watchable roots configured")
	}

	return w, nil
}

// Watch runs the event loop until the provided context is canceled.
func (w *Watch) Watch(ctx context.Context) error {
	var wg sync.WaitGroup
	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case p, ok := <-w.processQueue:
					if !ok {
						return
					}
					if err := w.processPath(p); err != nil {
						w.logger.Warnf("process path %s failed: %v", p, err)
					}
				}
			}
		}()
	}

	defer func() {
		close(w.processQueue)
		wg.Wait()
		if err := w.fswatch.Close(); err != nil {
			w.logger.Warnf("failed to close watcher: %v", err)
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.fswatch.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Chmod|fsnotify.Rename) != 0 {
				w.handleEvent(event)
			}
		case err, ok := <-w.fswatch.Errors:
			if !ok {
				return nil
			}
			w.logger.Warnf("watcher error: %v", err)
		case <-ticker.C:
			w.flushPending()
		}
	}
}

func (w *Watch) handleEvent(event fsnotify.Event) {
	w.logger.Debugf("watch event op=%s path=%s", event.Op.String(), event.Name)

	if event.Op&fsnotify.Create != 0 {
		if st, err := os.Lstat(event.Name); err == nil && st.IsDir() {
			if err := w.addRecursive(event.Name); err != nil {
				w.logger.Warnf("failed to add recursive watch for %s: %v", event.Name, err)
			}
			w.enqueueExistingPaths(event.Name)
		}
	}

	w.enqueuePending(event.Name, time.Now().Add(w.debounce))
}

func (w *Watch) processPath(path string) error {
	if !w.shouldProcessPath(path) {
		return nil
	}
	if err := w.fixer.ProcessPath(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func (w *Watch) shouldProcessPath(path string) bool {
	base := filepath.Base(path)
	for _, pat := range w.excludes {
		ok, _ := filepath.Match(pat, base)
		if ok {
			return false
		}
	}

	st, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if st.Mode()&os.ModeSymlink != 0 && !w.followLinks {
		return false
	}

	return true
}

func (w *Watch) addRecursive(root string) error {
	st, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("watch root is not a directory: %s", root)
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			w.logger.Warnf("watch walk error %s: %v", path, err)
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if !w.shouldProcessPath(path) {
			return filepath.SkipDir
		}

		if err := w.fswatch.Add(path); err != nil {
			w.logger.Warnf("failed to watch %s: %v", path, err)
		}
		return nil
	})
}

func (w *Watch) flushPending() {
	now := time.Now()
	due := make([]string, 0)

	w.mu.Lock()
	for p, when := range w.pendingPaths {
		if !now.Before(when) {
			due = append(due, p)
			delete(w.pendingPaths, p)
		}
	}
	w.mu.Unlock()

	for _, p := range due {
		select {
		case w.processQueue <- p:
		default:
			// Keep intake loop responsive: defer this path to next flush if workers are saturated.
			w.enqueuePending(p, now.Add(100*time.Millisecond))
		}
	}
}

func (w *Watch) enqueuePending(path string, when time.Time) {
	w.mu.Lock()
	w.pendingPaths[path] = when
	w.mu.Unlock()
}

func (w *Watch) enqueueExistingPaths(root string) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		w.enqueuePending(path, time.Now().Add(w.debounce))
		return nil
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
