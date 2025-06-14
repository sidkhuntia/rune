class Rune < Formula
  desc "AI-powered Git commit message generator"
  homepage "https://github.com/sidkhuntia/rune"
  version "1.0.8"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/sidkhuntia/rune/releases/download/v1.0.8/rune-darwin-arm64.tar.gz"
      sha256 "2a5bb9919b6868e41ed131fd0a865f7a755533603ed1eb8494d3777ceb58da37"
    else
      url "https://github.com/sidkhuntia/rune/releases/download/v1.0.8/rune-darwin-amd64.tar.gz"
      sha256 "c39f454bd55469f94d332a095d52a37e4fcc595b59bbea8edff147b4e8047462"
    end
  end

  def install
    bin.install "rune"
  end

  test do
    assert_match "Generate AI-powered Git commit messages", shell_output("#{bin}/rune --help")
  end
end
