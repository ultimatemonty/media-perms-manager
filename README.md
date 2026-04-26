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

Build with explicit version metadata:
```bash
go build -ldflags "-X main.version=1.2.3" -o bin/mpm ./...
```

Release packages:
```bash
make release RELEASE_NUMBER=1.2.3
```

Or run the stages separately:
```bash
make build RELEASE_NUMBER=1.2.3
make package RELEASE_NUMBER=1.2.3
```

This produces `.tar.xz` archives in `dist/` for `darwin/arm64`, `linux/amd64`, and `linux/arm64`. Each archive contains the `media-perms-manager` executable with execute permissions preserved and a matching `media-perms-manager.sha256` file.

GitHub release automation:
```bash
git tag v1.2.3
git push origin v1.2.3
```

Or use Makefile helpers to automate tag creation and push:
```bash
make tag-and-push VERSION=1.2.3
```

Optional controls:
- `REMOTE` (default: `origin`)
- `TAG_PREFIX` (default: `v`)

Examples:
```bash
make tag VERSION=1.2.3
make push-tag VERSION=1.2.3
make tag-and-push VERSION=1.2.3 REMOTE=origin TAG_PREFIX=v
```

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which runs `make release RELEASE_NUMBER=<tag without v>`, uploads artifacts, and publishes a GitHub Release with the generated `.tar.xz` files plus an archive checksum manifest.

Run (dry-run):
```bash
./media-perms-manager --dry-run
```

Show version:
```bash
./media-perms-manager --version
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
