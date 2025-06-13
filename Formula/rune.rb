class Rune < Formula
  desc "AI-powered Git commit message generator"
  homepage "https://github.com/sidkhuntia/homebrew-rune"
  version "1.0.2"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/sidkhuntia/homebrew-rune/releases/download/v1.0.2/rune-darwin-arm64.tar.gz"
      sha256 "sha256:79b87727dc06efedc00a62c9bbf941fd950778b0441c87ad650661849ee44495"
    else
      url "https://github.com/sidkhuntia/homebrew-rune/releases/download/v1.0.2/rune-darwin-amd64.tar.gz"
      sha256 "sha256:2cc1f35d1c65242e0096c79fec944c79c1927de29726fa3de76517cae9bfa568"
    end
  end

  def install
    bin.install "rune"
  end

  test do
    assert_match "Generate AI-powered Git commit messages", shell_output("#{bin}/rune --help")
  end
end