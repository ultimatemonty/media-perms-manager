# media-perms-manager

Small CLI to enforce file/directory ownership and permissions.

Defaults enforced:
- owner: media
- group: jellyfin
- file and dir mode: 0775
- excludes: *.trickplay

Usage:

Build (output to `bin/`):
```bash
go build -o bin/mpm ./...
```

Run (dry-run):
```bash
./media-perms-manager --dry-run
```

Run in watch mode (continuously monitor roots for new/changed files and directories):
```bash
./media-perms-manager --watch --roots /media,/downloads
```

Run watch mode with an initial full scan:
```bash
./media-perms-manager --watch --watch-initial-scan --watch-delay 750ms
```

Config precedence: CLI flags > environment variables > YAML config > defaults.
Default YAML locations: `./config.yaml`, `$XDG_CONFIG_HOME/mpm/config.yaml`, `~/.config/mpm/config.yaml`.

Watch mode environment variables:
- `MPM_WATCH` (true|false)
- `MPM_WATCH_DELAY` (duration, e.g. `500ms`)
- `MPM_WATCH_INITIAL_SCAN` (true|false)

Watch mode root handling:
- If one or more configured roots are invalid/unavailable, they are skipped and watching continues for valid roots.
- If all configured roots are invalid/unavailable, startup fails.

Note: `chown` to arbitrary users usually requires root privileges. Permission errors are logged and the run continues.
