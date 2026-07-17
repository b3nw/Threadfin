# Changelog

All notable changes to this repository are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning uses a `major.minor.patch` app version in `threadfin.go`.

This tree maintains open-source Threadfin for self-hosted / Docker use after the original author shifted development to a closed-source product. Pre-maintenance beta notes that lived in `changelog-beta.md` are not carried forward.

## [Unreleased]

### Changed

- Maintenance focus: Docker image + Plex DVR (HDHomeRun emulation). Non-Linux hosts are not a primary target.
- Web UI assets live under `src/html/` and are embedded with `//go:embed` (no generated base64 `webUI.go` / `html-build.go` step).
- New and changed web credentials use bcrypt (`golang.org/x/crypto/bcrypt`). Existing `authentication.json` entries that still use the legacy HMAC-SHA256 form continue to verify until the password is changed.
- Settings with buffer mode `threadfin` are normalized to `ffmpeg` on load (global and per-playlist).
- `storeBufferInRAM` again selects buffer storage: RAM (`memfs`) vs disk (`osfs`).
- Build toolchain is **Go 1.25** (`go.mod`, Docker builder images, GitHub Actions).
- README and project docs describe continued open-source maintenance after the original author shifted development to [Kernel Media](https://kernelmedia.tv/); remaining links and “upstream fork” framing were removed.

### Removed

- Binary self-update (`up2date`, GitHub release polling, `-branch` flag, scheduled binary update in maintenance).
- Leftover git-branch settings surface (`git.branch` / `System.Branch` / `THREADFIN_BRANCH` env) used only for old auto-update channels.
- Unused `golang.org/x/text` dependency (was only used to title-case branch names).
- Native Threadfin HLS buffer (`parseM3U8` / `switchBandwidth` and related paths). Use FFmpeg or VLC buffer, or direct (`-`) mode.
- Unused `/auto/` stub route and other dead handlers/commented experiments.
- Legacy unused pre-TypeScript web JS under the old `html/js/` tree.
- Multi-platform convenience paths that only served Windows/FreeBSD install quirks (e.g. Windows VFS volume hack, FreeBSD-specific `which` branches).
- Pre-maintenance beta changelog file (`changelog-beta.md`); this file is the source of truth going forward.
- Placeholder `README-DEV.md` and GitHub Actions workflow for the unused `beta` branch.
- Settings UI control for `ThreadfinAutoUpdate` (backend no longer updates the binary).

### Migration notes

When upgrading an existing Threadfin data directory:

1. **Config is mostly reusable.** Keep your `conf` volume (`settings.json`, playlists, XEPG mapping, etc.).
2. **Buffer mode.** If any playlist or global buffer was `threadfin`, it becomes `ffmpeg`. Ensure `ffmpeg` is available in the container/image (default Docker image includes it).
3. **No in-app binary updates.** Update by rebuilding/redeploying the container or binary. Ignore leftover `ThreadfinAutoUpdate` / `git.branch` keys in old `settings.json` (no longer read).
4. **Web authentication.** Existing users should continue to log in. Changing a password stores a bcrypt hash. If login fails after upgrade, remove or reset `authentication.json` and create the first user again in the web UI (`/web/`).
5. **Plex DVR.** After major network/URL changes, remove and re-add the Threadfin DVR in Plex if discovery or lineup looks wrong.
6. **Dev UI rebuild.** TypeScript sources are under `ts/`; compile with `tsc -p ./ts/tsconfig.json` (output: `src/html/js/`). Production builds serve embedded files; use `-dev` only for local asset reloads from `src/html/`.

### Security

- Credential hashing for new/changed passwords moved to bcrypt (see Changed).

---

## Earlier history

Release history before this maintenance cleanup is not restated here. This changelog starts at the Docker/Plex simplification and does not restate every pre-maintenance beta entry.
