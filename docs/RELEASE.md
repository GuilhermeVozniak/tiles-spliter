# Releasing Tiles Spliter

Tiles Spliter ships as a **signed and notarized** universal macOS `.app`
inside a DMG, published to GitHub Releases. Two GitHub Actions workflows
drive this:

- **`.github/workflows/ci.yml`** — runs on every push to `main` and every
  PR: the `js` job lints/tests/builds every JS workspace except the desktop
  Go app (shared, desktop frontend, web — including the web's Playwright
  e2e suite); the `go` job builds the desktop frontend (for the `//go:embed`
  in `main.go`), then runs `go build`, `go vet`, `go test -race`, and a
  `gofmt` check for `apps/desktop`.
- **`.github/workflows/release.yml`** — runs when a `v*` tag is pushed:
  gates on the full test suite (`bunx turbo run test lint build --continue`
  plus a dedicated `go test -race`), then builds a universal binary, signs
  it with the Developer ID + hardened runtime, packages a DMG, notarizes and
  staples it, and attaches it to a GitHub Release. A final
  `bump-homebrew-cask` job then points the Homebrew tap at the new DMG (see
  [Homebrew tap](#homebrew-tap)).

## One-time setup

You need an **Apple Developer account** ($99/yr). Gather two things.

### 1. Developer ID Application certificate

1. In Xcode (Settings › Accounts › Manage Certificates) or the Apple
   Developer portal, create a **Developer ID Application** certificate.
2. Export it (with its private key) as a `.p12`, either from Keychain
   Access (select the certificate → right-click → Export) or from the
   command line:
   ```sh
   security export -k login.keychain-db -t identities \
     -f pkcs12 -P "<export-password>" -o DeveloperID.p12 \
     -c "Developer ID Application: Guilherme Vozniak (AB12CD34EF)"
   ```
3. Base64-encode it for the secret:
   ```sh
   base64 -i DeveloperID.p12 | pbcopy
   ```

### 2. App-specific password (for notarization)

1. Sign in at <https://appleid.apple.com> › Sign-In and Security ›
   App-Specific Passwords → generate one (e.g. named "notarytool"). It looks
   like `abcd-efgh-ijkl-mnop`.
2. Note your **Apple ID email** and your 10-character **Team ID** (the code
   in parentheses in the signing identity from step 1, or
   developer.apple.com › Membership).

> Alternative: instead of shipping the Apple ID email + app-specific
> password to CI, you can store credentials once locally with
> `xcrun notarytool store-credentials` and reuse the resulting keychain
> profile for **local** releases via `NOTARY_KEYCHAIN_PROFILE` (see below).
> CI always uses the explicit `APPLE_ID`/`APPLE_TEAM_ID`/`APPLE_APP_PASSWORD`
> secrets since a keychain profile can't be shared across runners.

### 3. Add the repository secrets

Repo › Settings › Secrets and variables › Actions → add:

| Secret | Value |
| --- | --- |
| `MACOS_CERT_P12` | base64 of the Developer ID Application `.p12` |
| `MACOS_CERT_PASSWORD` | the `.p12` export password |
| `APPLE_ID` | your Apple ID email |
| `APPLE_TEAM_ID` | your 10-character Team ID |
| `APPLE_APP_PASSWORD` | the app-specific password (`xxxx-xxxx-xxxx-xxxx`) |
| `HOMEBREW_TAP_TOKEN` | fine-grained PAT, **Contents: read/write** on `homebrew-tap` only (see [Homebrew tap](#homebrew-tap)) |

The CI keychain password is generated per-run; no keychain secret is
needed. The codesign identity is resolved from the imported certificate at
run time (via `security find-identity`), so no identity-name secret is
needed either.

If any of `MACOS_CERT_P12` etc. are missing, `release.yml`'s `build-release`
job still runs but every signing/build step is skipped — the tag simply
does not produce a release artifact until secrets are configured.

## Cutting a release

```sh
git tag v0.1.0
git push origin v0.1.0
```

Any tag matching `v*` triggers `release.yml`. The tag version (without the
leading `v`) is stamped into `apps/desktop/build/Info.plist`
(`CFBundleShortVersionString` / `CFBundleVersion`) and used to name the
asset: `tiles-spliter_<version>_darwin_universal.dmg` — the same contract
`@tiles-spliter/shared`'s `releaseAssetName()`/`downloadUrl()` use to build
the landing page's download links. Watch the run under the repo's Actions
tab; on success a Release with the notarized DMG appears, followed by a
commit on the Homebrew tap so `brew upgrade` picks the version up.

## Homebrew tap

Users install with:

```sh
brew install --cask GuilhermeVozniak/tap/tiles-spliter
```

The cask lives in <https://github.com/GuilhermeVozniak/homebrew-tap>
(`Casks/tiles-spliter.rb`) and pins one `version` plus the DMG's `sha256`.
`release.yml`'s `bump-homebrew-cask` job rewrites both after every published
release, runs `brew audit` and `brew fetch` on the result (so the checksum is
verified against the real asset), and pushes to the tap's `main`. Pushing to
another repository is beyond the default `GITHUB_TOKEN`, hence the extra
secret:

1. GitHub › Settings › Developer settings › Personal access tokens ›
   Fine-grained tokens → Generate new token.
2. Repository access: **Only select repositories** → `homebrew-tap`.
   Permissions: **Contents → Read and write**. Nothing else.
3. Save it as the `HOMEBREW_TAP_TOKEN` secret on *this* repository.

Without the secret the job still audits the cask but skips the push and
emits a warning on the run. Bump by hand in that case:

```sh
brew tap GuilhermeVozniak/tap
cd "$(brew --repository guilhermevozniak/tap)"
# edit Casks/tiles-spliter.rb: version, and sha256 from `shasum -a 256 <dmg>`
brew audit --cask guilhermevozniak/tap/tiles-spliter
brew fetch --cask guilhermevozniak/tap/tiles-spliter
git commit -am "tiles-spliter <version>" && git push
```

## Local release fallback

`scripts/release.sh` runs the same build/sign/notarize/package pipeline
locally:

```sh
export CODESIGN_IDENTITY="Developer ID Application: Guilherme Vozniak (AB12CD34EF)"
# Either a stored notarytool profile...
export NOTARY_KEYCHAIN_PROFILE="tiles-spliter-notary"
# ...or explicit Apple ID credentials:
export APPLE_ID="you@example.com"
export APPLE_TEAM_ID="AB12CD34EF"
export APPLE_APP_PASSWORD="abcd-efgh-ijkl-mnop"

./scripts/release.sh
```

`VERSION` defaults to whatever is stamped in
`apps/desktop/build/Info.plist`; set the `VERSION` env var to override it
(this is how CI passes the tag version through without touching the
checked-in `Info.plist` outside of CI).

Use `./scripts/release.sh --dry-run` to sanity-check the frontend + Go
build and app bundle assembly without codesigning, notarizing, or building
a DMG (arm64-only, fast, no Apple credentials required).

## Notes & troubleshooting

- **Universal cross-compile:** the app builds `darwin/arm64` and
  `darwin/amd64` separately (with `CGO_ENABLED=1` forced, since cgo
  defaults off when `GOARCH` differs from the host) and joins them with
  `lipo`. If cross-compiling `amd64` from an Apple Silicon runner ever
  breaks, fall back to arm64-only and drop `universal` from the asset name.
- **Entitlements:** hardened-runtime entitlements live in
  `apps/desktop/build/entitlements.plist`.
- **Notarization failures:** if `notarytool submit --wait` reports
  `Invalid`, run `xcrun notarytool log <submission-id> --key ...` (or with
  `--apple-id`/`--team-id`/`--password`) to see which nested binary was
  unsigned or lacked the hardened runtime.
- Run through the manual parity checklist in [SMOKE.md](SMOKE.md) before
  cutting a release.
