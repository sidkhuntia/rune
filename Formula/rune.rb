class Rune < Formula
  desc "AI-powered Git commit message generator"
  homepage "https://github.com/sidkhuntia/rune"
  version "1.0.7"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/sidkhuntia/rune/releases/download/v1.0.7/rune-darwin-arm64.tar.gz"
      sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
    else
      url "https://github.com/sidkhuntia/rune/releases/download/v1.0.7/rune-darwin-amd64.tar.gz"
      sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
    end
  end

  def install
    bin.install "rune"
  end

  test do
    assert_match "Generate AI-powered Git commit messages", shell_output("#{bin}/rune --help")
  end
end
