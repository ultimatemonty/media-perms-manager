package fixer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	cfgpkg "github.com/ultimatemonty/media-perms-manager/internal/config"
	"github.com/ultimatemonty/media-perms-manager/internal/fsops"
	"github.com/ultimatemonty/media-perms-manager/internal/logger"
)

type Fixer struct {
	cfg      cfgpkg.Config
	logger   *logger.Logger
	uid      int
	gid      int
	fileMode os.FileMode
	dirMode  os.FileMode
}

// ProcessPath applies ownership and mode fixes to a single path.
// It is exported so the watcher can reuse the same fixing logic as the scan mode.
func (f *Fixer) ProcessPath(path string) error {
	return f.processPath(path)
}

func New(cfg cfgpkg.Config, logg *logger.Logger) (*Fixer, error) {
	uid, gid, err := fsops.ResolveUserGroup(cfg.Owner, cfg.Group)
	if err != nil {
		return nil, err
	}
	fm, err := cfgpkg.ParseMode(cfg.FileMode)
	if err != nil {
		return nil, err
	}
	dm, err := cfgpkg.ParseMode(cfg.DirMode)
	if err != nil {
		return nil, err
	}
	return &Fixer{
		cfg:      cfg,
		logger:   logg,
		uid:      uid,
		gid:      gid,
		fileMode: os.FileMode(fm),
		dirMode:  os.FileMode(dm),
	}, nil
}

func (f *Fixer) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// channel of paths to process
	paths := make(chan string, 1024)
	var wg sync.WaitGroup
	// start workers
	for i := 0; i < f.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case p, ok := <-paths:
					if !ok {
						return
					}
					if err := f.processPath(p); err != nil {
						f.logger.Warnf("path=%s: %v", p, err)
					}
				}
			}
		}()
	}

	// walk roots and send paths
	for _, root := range f.cfg.Roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				f.logger.Warnf("walk error %s: %v", path, err)
				return nil
			}
			// skip symlinks unless configured
			if d.Type()&os.ModeSymlink != 0 && !f.cfg.FollowSymlinks {
				return nil
			}
			// skip excluded patterns; if it's a directory, skip descending into it
			base := filepath.Base(path)
			for _, pat := range f.cfg.Excludes {
				ok, _ := filepath.Match(pat, base)
				if ok {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case paths <- path:
			}
			return nil
		})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				close(paths)
				wg.Wait()
				return ctxErr
			}
			f.logger.Warnf("walk root %s failed: %v", root, err)
		}
	}

	close(paths)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (f *Fixer) processPath(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	// skip symlink
	if st.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	isDir := st.IsDir()

	desired := f.fileMode
	if isDir {
		desired = f.dirMode
	}

	// check mode
	curPerm := st.Mode().Perm()
	if curPerm != desired.Perm() {
		if err := fsops.Chmod(path, desired, f.cfg.DryRun); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
		f.logger.Infof("chmod %s -> %o", path, desired)
	}

	// check owner
	var stat *syscall.Stat_t
	if s, ok := st.Sys().(*syscall.Stat_t); ok {
		stat = s
	}
	if stat == nil {
		return errors.New("unsupported platform: can't get stat_t")
	}
	curUid := int(stat.Uid)
	curGid := int(stat.Gid)
	if (f.uid != -1 && curUid != f.uid) || (f.gid != -1 && curGid != f.gid) {
		if err := fsops.Chown(path, f.uid, f.gid, f.cfg.DryRun); err != nil {
			return fmt.Errorf("chown: %w", err)
		}
		f.logger.Infof("chown %s -> %d:%d", path, f.uid, f.gid)
	}

	return nil
}
