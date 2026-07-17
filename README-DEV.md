# Developer notes

## Web interfaces

Threadfin ships two web interfaces:

- **`/web/`** — the legacy interface. TypeScript sources in `ts/` are compiled
  with `ts/compileJS.sh` into `src/html/js/` and served through Go's template
  engine (localization placeholders are substituted per request).
- **`/ui/`** — the new single-page interface (Vite + Svelte). Sources live in
  `web/`; the production bundle is committed to `webui/dist/` and embedded
  into the binary via `go:embed` (`webui/embed.go`). Assets are static — no
  Go templating.

Both talk to the same WebSocket API at `/data/` (see `WS()` in
`src/webserver.go` and the payload types in `src/struct-webserver.go`).

### Working on the new UI

```bash
cd web
npm install
npm run dev     # dev server on :5173, proxies /data/, /web/, … to :34400
```

Run a local Threadfin (`go run threadfin.go`) alongside the dev server so the
proxy has a backend.

Before committing UI changes, rebuild the embedded bundle:

```bash
cd web
npm run check   # svelte-check / type check
npm run build   # writes webui/dist/ (commit this)
```

The Docker builds (`Dockerfile`, `Dockerfile.arm`) also rebuild the bundle in
a Node stage, so images never depend on the committed `webui/dist/`. A plain
`go build` does — keep the committed bundle in sync with `web/src`.

### Building the binary

```bash
go build -mod=mod -o threadfin threadfin.go
```

(`-mod=mod` matches the Dockerfiles; the checked-in `vendor/` tree is stale.)
