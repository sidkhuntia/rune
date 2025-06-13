# Homebrew Distribution Guide for Rune

This guide explains how to distribute Rune via Homebrew, making it easy for users to install and update your CLI tool.

## Overview

Homebrew distribution involves:
1. Creating a Homebrew tap (custom repository)
2. Creating a formula file that defines how to install Rune
3. Automating updates through GitHub Actions
4. Publishing releases

## Step 1: Create a Homebrew Tap

### 1.1 Create the Tap Repository

Create a new GitHub repository named `homebrew-rune` (the `homebrew-` prefix is required):

```bash
# Create new repository on GitHub
gh repo create yourusername/homebrew-rune --public

# Clone it locally
git clone https://github.com/yourusername/homebrew-rune.git
cd homebrew-rune
```

### 1.2 Set Up Repository Structure

```bash
mkdir -p Formula
touch Formula/rune.rb
touch README.md
```

## Step 2: Create the Formula

### 2.1 Initial Formula File

Create `Formula/rune.rb`:

```ruby
class Rune < Formula
  desc "AI-powered Git commit message generator"
  homepage "https://github.com/yourusername/rune"
  url "https://github.com/yourusername/rune/archive/v1.0.0.tar.gz"
  sha256 "YOUR_SHA256_HASH_HERE"
  license "MIT"
  version "1.0.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/rune"
    bin.install "rune"
  end

  test do
    assert_match "Generate AI-powered Git commit messages", shell_output("#{bin}/rune --help")
  end
end
```

### 2.2 Generate SHA256 Hash

```bash
# Download the release archive
curl -L -o rune-v1.0.0.tar.gz https://github.com/yourusername/rune/archive/v1.0.0.tar.gz

# Generate SHA256
shasum -a 256 rune-v1.0.0.tar.gz
```

Update the `sha256` field in the formula with the generated hash.

## Step 3: Test the Formula Locally

### 3.1 Install from Local Formula

```bash
# From the homebrew-rune directory
brew install --build-from-source ./Formula/rune.rb

# Test the installation
rune --help

# Uninstall for cleanup
brew uninstall rune
```

### 3.2 Audit the Formula

```bash
brew audit --strict Formula/rune.rb
brew style Formula/rune.rb
```

## Step 4: Automation Setup

### 4.1 GitHub Secrets

Add these secrets to your main Rune repository:

1. `COMMITTER_TOKEN`: Personal Access Token with repo permissions
   - Go to GitHub Settings → Developer settings → Personal access tokens
   - Generate new token with `repo` scope
   - Add as secret in your main repository

### 4.2 Update GitHub Actions

The CI workflow in `.github/workflows/ci.yml` already includes Homebrew automation. Update the `yourusername` placeholders:

```yaml
homebrew-tap: yourusername/homebrew-rune
```

## Step 5: Create Initial Release

### 5.1 Tag and Release

```bash
# In your main rune repository
git tag v1.0.0
git push origin v1.0.0

# Or create release via GitHub UI
gh release create v1.0.0 --title "v1.0.0" --notes "Initial release"
```

### 5.2 Manual Formula Update (First Time)

Since automation requires an existing formula, update manually for the first release:

```bash
cd homebrew-rune

# Update Formula/rune.rb with correct URL and SHA256
# Commit and push
git add Formula/rune.rb
git commit -m "Add rune formula v1.0.0"
git push origin main
```

## Step 6: User Installation

### 6.1 Add Tap

Users can now install Rune via Homebrew:

```bash
# Add your tap
brew tap yourusername/rune

# Install rune
brew install rune
```

### 6.2 Alternative One-Line Install

```bash
# Install directly without adding tap
brew install yourusername/rune/rune
```

## Step 7: Advanced Configuration

### 7.1 Multiple Architectures

For universal binaries or multiple architectures:

```ruby
class Rune < Formula
  desc "AI-powered Git commit message generator"
  homepage "https://github.com/yourusername/rune"
  license "MIT"
  version "1.0.0"

  if OS.mac?
    if Hardware::CPU.intel?
      url "https://github.com/yourusername/rune/releases/download/v1.0.0/rune-darwin-amd64.tar.gz"
      sha256 "INTEL_MAC_SHA256"
    elsif Hardware::CPU.arm?
      url "https://github.com/yourusername/rune/releases/download/v1.0.0/rune-darwin-arm64.tar.gz"
      sha256 "ARM_MAC_SHA256"
    end
  elsif OS.linux?
    url "https://github.com/yourusername/rune/releases/download/v1.0.0/rune-linux-amd64.tar.gz"
    sha256 "LINUX_SHA256"
  end

  def install
    bin.install "rune"
  end

  test do
    assert_match "Generate AI-powered Git commit messages", shell_output("#{bin}/rune --help")
  end
end
```

### 7.2 Development Version

For HEAD installation:

```ruby
class Rune < Formula
  # ... existing configuration ...

  head do
    url "https://github.com/yourusername/rune.git", branch: "main"
    depends_on "go" => :build
  end

  def install
    if build.head?
      system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/commitmsg"
      bin.install "commitmsg" => "rune"
    else
      bin.install "rune"
    end
  end
end
```

## Step 8: Maintenance

### 8.1 Automated Updates

After initial setup, the GitHub Actions workflow will automatically:
1. Build releases for multiple platforms
2. Update the Homebrew formula
3. Test the formula

### 8.2 Manual Updates

If needed, update the formula manually:

```bash
cd homebrew-rune

# Edit Formula/rune.rb
# Update version, URL, and SHA256

# Test locally
brew install --build-from-source ./Formula/rune.rb
brew test rune
brew audit --strict Formula/rune.rb

# Commit and push
git add Formula/rune.rb
git commit -m "Update rune to v1.1.0"
git push origin main
```

## Step 9: Troubleshooting

### Common Issues

**Formula audit failures:**
```bash
brew audit --strict --online Formula/rune.rb
```

**Build failures:**
```bash
brew install --build-from-source --verbose Formula/rune.rb
```

**Testing failures:**
```bash
brew test --verbose rune
```

### Best Practices

1. **Always test locally** before pushing formula updates
2. **Use semantic versioning** for releases
3. **Keep SHA256 hashes updated** for security
4. **Monitor GitHub Actions** for automated updates
5. **Respond to user issues** promptly

## Step 10: Publishing to Main Homebrew

To get Rune into the main Homebrew repository (optional):

1. **Meet requirements**: Popular, stable, notable project
2. **Submit PR**: To [Homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core)
3. **Follow guidelines**: [Homebrew's contribution guide](https://docs.brew.sh/How-To-Open-a-Homebrew-Pull-Request)

## Resources

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Python Guide](https://docs.brew.sh/Python-for-Formula-Authors)
- [Homebrew Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
- [GitHub Actions for Homebrew](https://github.com/mislav/bump-homebrew-formula-action)

## Example Commands Summary

```bash
# Create tap
gh repo create yourusername/homebrew-rune --public

# Install from tap
brew tap yourusername/rune
brew install rune

# Update formula (automated via GitHub Actions)
git tag v1.1.0
git push origin v1.1.0

# Test locally
brew install --build-from-source ./Formula/rune.rb
brew test rune
brew audit --strict Formula/rune.rb
``` 