<div align="center">
  <img width="285" height="80" src="src/html/img/threadfin.png" alt="Threadfin" />
</div>

# Threadfin (maintenance fork)

M3U proxy for **Plex DVR** and Emby/Jellyfin Live TV. Originally based on [xTeVe](https://github.com/xteve-project/xTeVe); this tree is a fork of [Threadfin/Threadfin](https://github.com/Threadfin/Threadfin).

**Upstream is frozen** (maintainer stepped away). This repository is maintained for self-hosted / Docker use. Do not expect binary auto-updates from GitHub.

| Doc | Purpose |
|-----|---------|
| [CHANGELOG.md](CHANGELOG.md) | Fork changes, breaking behavior, migration notes |
| [xTeVe configuration guide](https://github.com/xteve-project/xTeVe-Documentation/blob/master/en/configuration.md) | Setup concepts still largely apply |

## Features

- Merge external M3U / HDHomeRun / XMLTV sources
- Channel filter, mapping, logos, categories, backup streams
- Scheduled playlist / EPG refresh and XEPG generation
- HDHomeRun-compatible discovery for Plex DVR
- Stream buffer via **FFmpeg** or **VLC**, or direct pass-through (`-`)
- Optional RAM or disk buffer (`storeBufferInRAM`)
- Web UI (embedded static assets under `src/html/`)

## Requirements

### Plex

- Plex Media Server (1.11.1.4730 or newer)
- Plex client with DVR support
- Plex Pass

### Emby

- Emby Server (3.5.3.0 or newer)
- Emby client with Live TV
- Emby Premiere

### Jellyfin

- Jellyfin Server (10.7.1 or newer)
- Jellyfin client with Live TV

### Runtime

- **Linux** is the primary target (especially Docker).
- **FFmpeg** (and optionally **VLC/cvlc**) when using a buffered playlist mode.
- Go **1.23+** to build from source (see `go.mod` for the module’s declared version).

## Docker

Build from this repository:

```bash
docker build -t threadfin:local .
```

Compose example:

```yaml
services:
  threadfin:
    image: threadfin:local
    container_name: threadfin
    ports:
      - "34400:34400"
    environment:
      - TZ=America/Los_Angeles
      - THREADFIN_DEBUG=0
    volumes:
      - ./data/conf:/home/threadfin/conf
      - ./data/temp:/tmp/threadfin:rw
    restart: unless-stopped
```

Web UI: `http://<host>:34400/web/`

Default container entrypoint binds `0.0.0.0` and uses `/home/threadfin/conf` for config.

> Historical Hub image `fyb3roptik/threadfin` and TrueCharts charts track **upstream**, not this fork. Prefer building from this repo.

## CLI

| Flag | Description |
|------|-------------|
| `-config <path>` | Config directory (default: `~/.threadfin/`) |
| `-port <port>` | HTTP port (default: `34400`) |
| `-bind <ip>` | Bind address |
| `-debug <0-3>` | Debug level |
| `-info` | Print system info and exit |
| `-restore <zip>` | Restore from backup zip |
| `-dev` | Serve UI files from `src/html/` (development) |
| `-h` | Help |

Binary self-update and `-branch` are **not** available in this fork. Deploy a new image or binary to upgrade.

## Build from source

```bash
go mod tidy
go build -o threadfin threadfin.go
```

### Web UI (TypeScript)

If you change files under `ts/`:

```bash
tsc -p ./ts/tsconfig.json
```

Output goes to `src/html/js/`. Production builds embed `src/html/` via `//go:embed`. Use `-dev` while iterating on static assets without re-embedding.

### Dependencies

- [gorilla/websocket](https://github.com/gorilla/websocket)
- [koron/go-ssdp](https://github.com/koron/go-ssdp)
- [avfs](https://github.com/avfs/avfs)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) (bcrypt)
- [golang.org/x/text](https://pkg.go.dev/golang.org/x/text)

## Upgrading / migration

See **[CHANGELOG.md](CHANGELOG.md)** (Migration notes). Short version:

1. Keep your config volume.
2. `threadfin` buffer → `ffmpeg` (automatic on load).
3. No in-app binary updates — redeploy the container/binary.
4. Web users: legacy password hashes still verify; change password to rehash with bcrypt. If locked out, reset `authentication.json` and create the first user again.
5. Re-add Plex DVR if discovery or lineup breaks after network changes.

## License

MIT — see [LICENSE](LICENSE). Copyright retained from upstream Threadfin / xTeVe lineage.
