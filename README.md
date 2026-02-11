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

Config precedence: CLI flags > environment variables > YAML config > defaults.
Default YAML locations: `./config.yaml`, `$XDG_CONFIG_HOME/mpm/config.yaml`, `~/.config/mpm/config.yaml`.

Note: `chown` to arbitrary users usually requires root privileges. Permission errors are logged and the run continues.
