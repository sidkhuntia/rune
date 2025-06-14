# Release Process for Rune

This document explains how to release new binaries and update the Homebrew formula for the Rune CLI tool.

## Prerequisites
- Ensure all code is merged to `main` and CI is passing.
- You have push access to the repository.
- GitHub secrets are set:
  - `GITHUB_TOKEN` (for release uploads)
  - `COMMITTER_TOKEN` (for Homebrew formula updates)

## 1. Tag and Push a Release

You can create a release either via the GitHub UI or CLI:

### Using Git CLI
```bash
git tag vX.Y.Z
# Example: git tag v1.2.0
git push origin vX.Y.Z
```

### Using GitHub CLI
```bash
gh release create vX.Y.Z --title "vX.Y.Z" --notes "Release notes here"
```

### Using GitHub UI
- Go to the Releases section.
- Click "Draft a new release".
- Set the tag (e.g., `v1.2.0`), title, and notes.
- Publish the release.

## 2. CI/CD Automation

Once a release is created:
- The GitHub Actions workflow will:
  1. Run tests and lint.
  2. Build binaries for all supported OS/architectures.
  3. Upload binaries as release assets.
  4. Update the Homebrew formula in your tap repository (if configured and secrets are set).

## 3. Homebrew Formula Update

- The workflow uses the [bump-homebrew-formula-action](https://github.com/mislav/bump-homebrew-formula-action) to update the formula in your Homebrew tap.
- Make sure the following are correct in `.github/workflows/ci.yml`:
  - `homebrew-tap: yourusername/homebrew-rune`
  - `formula-path: Formula/rune.rb`
  - `download-url` points to the correct release asset.

## 4. Manual Formula Update (First Release Only)
If this is your first release, you may need to manually update the formula:
```bash
cd homebrew-rune
# Update Formula/rune.rb with the new version, URL, and SHA256
# Commit and push
```

## 5. User Installation
Users can install the latest release via Homebrew:
```bash
brew tap yourusername/rune
brew install rune
```
Or directly:
```bash
brew install yourusername/rune/rune
```

## 6. Troubleshooting
- If the Homebrew formula does not update, check the GitHub Actions logs for errors.
- Ensure all required secrets are set in the repository settings.
- For more details, see `docs/homebrew-distribution.md`. 