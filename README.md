# Tiles Spliter

A macOS window-tiling utility. This monorepo contains the desktop application
(Go + Wails v3), the marketing site (Next.js), and shared TypeScript
libraries that power the Tiles Spliter ecosystem.

## Quick Start

```bash
bun install
bun run build
bun run test
```

`bun run build` builds every workspace via Turbo — including compiling the
desktop frontend and the Go binary (`apps/desktop/bin/tilespliter`). `bun run
test` runs Go tests (engine/app/platform), Vitest (shared, desktop frontend),
and Playwright (web) across all workspaces.

To iterate on the desktop app with hot reload, install the [Wails v3
CLI](https://v3.wails.io/) and run `wails3 dev` from `apps/desktop` (not
wired into `bun run dev` — the CLI is not a project dependency).

### Wails bindings

The frontend calls Go through generated bindings in
`apps/desktop/frontend/bindings/`, which are **committed** so fresh clones
build without the Wails CLI. After changing any Go method bound to the
frontend (`SettingsService`), regenerate them:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
cd apps/desktop && wails3 generate bindings
```

The desktop build script runs `wails3 generate bindings` automatically when
the CLI is installed and falls back to the committed bindings otherwise.

## Repo layout

```
apps/
  desktop/          Go + Wails v3 macOS app (engine, platform, app packages)
    frontend/        React preferences UI (Vite, Vitest)
  web/               Next.js marketing site (Playwright)
packages/
  shared/            Shared TypeScript utilities (Vitest)
docs/
  SMOKE.md           Manual parity checklist run before every release
scripts/
  release.sh         Builds, signs, notarizes, and packages the release DMG
```

## Accessibility permission

The desktop app moves and resizes other applications' windows via the macOS
Accessibility API. On first run (or after certain rebuilds), macOS requires
the user to grant Accessibility permission in System Settings → Privacy &
Security → Accessibility before window actions will work.

## Release

Releases are built and packaged with `scripts/release.sh` (universal binary,
codesign, notarization, and DMG creation) and published automatically by
`.github/workflows/release.yml` on every `v*` tag push. See
[docs/RELEASE.md](docs/RELEASE.md) for the full release process, required
repository secrets, and the local fallback. Before cutting a release, run
through the manual parity checklist in [docs/SMOKE.md](docs/SMOKE.md).
