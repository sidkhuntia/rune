# Releasing Rune

Releases are fully automated. Merge to `main` and the next version is tagged, built, published to GitHub Releases, and pushed to the Homebrew tap ([sidkhuntia/homebrew-tap](https://github.com/sidkhuntia/homebrew-tap)).

## How the version is chosen

`.github/workflows/release.yml` reads the commit subjects since the last `v*` tag:

| Commits since last tag | Bump |
| --- | --- |
| `type!:` subject or a `BREAKING CHANGE` body | major |
| any `feat:` / `feat(scope):` | minor |
| anything else user-facing (`fix:`, `refactor:`, `perf:`, unprefixed) | patch |
| only `docs`, `ci`, `chore`, `test`, `style` | no release |

Escape hatches:
- Add `[skip release]` to the head commit message to skip a release.
- Changes limited to `*.md`, `docs/` or `.github/` never trigger a release.
- Pushing a `vX.Y.Z` tag by hand also releases that tag.

## What a release does

1. Runs `go test -race ./...`.
2. Creates and pushes the tag.
3. GoReleaser builds darwin/linux × amd64/arm64, stamps `main.version`, publishes the archives and `checksums.txt`, and commits `rune.rb` to the tap.

## One-time setup

The `COMMITTER_TOKEN` repository secret must be a personal access token with **Contents: read & write** on `sidkhuntia/homebrew-tap` (`GITHUB_TOKEN` cannot push to another repository).

## Installing and updating

```bash
brew install sidkhuntia/tap/rune     # install
brew upgrade rune                    # update via Homebrew
rune --update                        # self-update (delegates to brew for Homebrew installs)
rune --version
```

`rune --update` downloads the release archive for the current platform, verifies its SHA-256 against `checksums.txt`, and atomically replaces the binary. If the binary lives in a Homebrew Cellar it runs `brew upgrade sidkhuntia/tap/rune` instead, so Homebrew's records stay correct.

## Testing a release locally

```bash
goreleaser release --snapshot --clean --skip=publish
```
