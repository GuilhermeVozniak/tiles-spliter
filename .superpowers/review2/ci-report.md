# CI / Release Automation — Implementation Report

## Deliverables

1. **`.github/workflows/ci.yml`** — two jobs:
   - `js` (ubuntu-latest): `oven-sh/setup-bun@v2`, `bun install --frozen-lockfile`,
     Playwright chromium install for the web e2e, then
     `bunx turbo run lint|test|build --filter=!@tiles-spliter/desktop`.
     `@tiles-spliter/desktop` is excluded because `apps/desktop`'s Go code
     uses Cocoa/Carbon APIs unconditionally (no build tags) and cannot
     compile on Linux — unlike the sibling repos, there's no separate
     `apps/cli` or cross-platform desktop target to worry about here.
   - `go` (macos-14): `actions/setup-go@v5` with
     `go-version-file: apps/desktop/go.mod` + `cache-dependency-path:
     apps/desktop/go.sum`, builds the desktop frontend first (satisfies the
     `//go:embed all:frontend/dist` in `main.go`), then
     `go build ./... && go vet ./... && go test ./... -race`, then a
     `gofmt -l` check.

2. **`.github/workflows/deploy-web.yml`** — verbatim house pattern (push to
   `main` on `apps/web/**`/`packages/shared/**` + `workflow_dispatch`,
   `concurrency: group: pages`, `upload-pages-artifact@v3` +
   `deploy-pages@v4`). Sets `NEXT_PUBLIC_BASE_PATH=/tiles-spliter` on the
   build step since this repo has no custom domain configured (unlike
   drag-zone) and is served under the project path.
   - `apps/web/next.config.ts` now reads `basePath` from
     `NEXT_PUBLIC_BASE_PATH` (default `""`), matching drag-zone's pattern.
   - `turbo.json`'s `build` task now declares
     `"env": ["NEXT_PUBLIC_BASE_PATH"]` so Turbo's cache key accounts for it.

3. **`.github/workflows/release.yml`** — on `v*` tags:
   - `test` job (macos-14, since `apps/desktop` only compiles on Darwin):
     `bunx turbo run test lint build --continue` (covers shared, desktop
     frontend, web, and `apps/desktop`'s own `go test`/`go vet` via its
     package.json scripts) plus a dedicated `go test ./... -race` step.
   - `build-release` job (needs: test, macos-14): derives `VERSION` from
     `GITHUB_REF_NAME`, stamps `apps/desktop/build/Info.plist`
     (`CFBundleShortVersionString` + `CFBundleVersion` via `PlistBuddy`),
     imports the signing cert into a temp keychain exactly like
     app-cleaner/drag-zone (base64 decode, 6h keychain timeout, curl the
     Developer ID G2 intermediate, `set-key-partition-list`), resolves
     `CODESIGN_IDENTITY` from the imported identity's full name string, then
     runs `./scripts/release.sh` with `VERSION`/`CODESIGN_IDENTITY`/
     `APPLE_ID`/`APPLE_TEAM_ID`/`APPLE_APP_PASSWORD` env, and uploads with
     `softprops/action-gh-release@v2` (`files: dist/tiles-spliter_*.dmg`,
     `generate_release_notes: true`).
   - Every signing/build/publish step is gated on
     `env.HAS_MACOS_SIGNING == 'true'` (secret presence surfaced once as a
     job-level env var, mirroring the sibling pattern). Unlike the siblings,
     `scripts/release.sh` is one atomic script (build → sign → notarize →
     package) with no unsigned-build fallback path, so when secrets are
     absent the whole pipeline is skipped rather than partially run.

4. **`scripts/release.sh`**:
   - `VERSION="${VERSION:-$(PlistBuddy ...)}"` — CI (or a manual override)
     can inject the tag version directly; local runs still default to the
     value stamped in `Info.plist`.
   - DMG renamed to the house convention:
     `tiles-spliter_${VERSION}_darwin_universal.dmg` (was
     `Tiles Spliter-$VERSION.dmg`).
   - `CODESIGN_IDENTITY` usage unchanged (`codesign --sign "$CODESIGN_IDENTITY"`)
     — already works whether it's a local Keychain-trusted name or the
     full identity string resolved in CI.
   - `--dry-run` behavior untouched (still arm64-only, skips
     codesign/notarize/DMG).

5. **`packages/shared`** download contract (`src/index.ts`):
   - `GITHUB_REPO = "GuilhermeVozniak/tiles-spliter"`
   - `releaseAssetName(version)` → `tiles-spliter_<version>_darwin_universal.dmg`
   - `downloadUrl(version?)` → `releases/latest` with no arg, or the direct
     tagged asset URL otherwise.
   - Added a `describe` block in `src/index.test.ts` (4 new assertions,
     13 total tests, all passing).
   - `apps/web/app/layout.tsx`, `apps/web/app/page.tsx`,
     `apps/web/components/Hero.tsx` now import `downloadUrl` (and
     `GITHUB_REPO` for the layout's repo link) instead of hardcoding the
     releases URL three times.

6. **`docs/RELEASE.md`** — house-style doc (overview, one-time Apple
   Developer setup incl. `security export` and app-specific password,
   secrets table, tag-cutting flow, local `scripts/release.sh` fallback
   with `VERSION`/`NOTARY_KEYCHAIN_PROFILE` env vars, troubleshooting).
   Linked from README's Release section.

## Verification

- `bash -n scripts/release.sh` — OK.
- `./scripts/release.sh --dry-run` — builds frontend + arm64 binary,
  assembles `dist/Tiles Spliter.app`, exits 0.
- `actionlint -color .github/workflows/*.yml` — clean, exit 0 (actionlint
  1.7.12 was available via `brew`).
- `cd packages/shared && bunx vitest run` — 13/13 tests pass.
- `cd apps/web && bun run build` — succeeds, `out/index.html` references
  unprefixed `/_next/...` assets.
- `cd apps/web && NEXT_PUBLIC_BASE_PATH=/tiles-spliter bun run build` —
  succeeds, `out/index.html` references `/tiles-spliter/_next/...` assets
  (basePath prefix confirmed present).
- Root `bunx turbo run test lint build --continue` — 11/11 tasks green
  (one biome formatting nit in the new shared test was caught and fixed
  with `bunx biome check --write src`).
- `apps/desktop/frontend/dist/.gitkeep` got clobbered by the local Vite
  build during verification; restored with `git checkout --
  apps/desktop/frontend/dist/.gitkeep` before committing.

## Deviations from the sibling pattern (and why)

- **No `golangci-lint` step in `ci.yml`**: the task spec only asked for
  `go build && go vet && go test -race` + `gofmt`, and there's no
  `.golangci.yml` in this repo yet. Siblings run golangci-lint; adding it
  here would require introducing a lint config with no spec to base it on.
- **No inline `actionlint` step embedded in `ci.yml`** (app-cleaner does
  this): the task's verify step asked to run actionlint locally, not to
  wire it into CI; I ran it locally instead and it's clean.
- **`test` job gate runs entirely on macos-14** (not split ubuntu/macos
  like `ci.yml`): `apps/desktop` cannot build on Linux, and the release
  gate needs the full workspace including the Go app, so one macOS runner
  covers `bunx turbo run test lint build --continue` plus the dedicated
  `-race` step, closer to app-cleaner's single-runner release gate.
- **`build-release`'s conditional structure is coarser than the
  siblings'**: app-cleaner/option-tab/drag-zone can produce an *unsigned*
  build even without secrets (each signing/notarize step is independently
  gated). `scripts/release.sh` here is one atomic pipeline with signing and
  notarizing baked into the middle of it (no unsigned fallback exists), so
  the whole build/publish sequence is gated on `HAS_MACOS_SIGNING` instead.
