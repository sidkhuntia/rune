class Rune < Formula
  desc "AI-powered Git commit message generator"
  homepage "https://github.com/sidkhuntia/homebrew-rune"
  version "1.0.4"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/sidkhuntia/homebrew-rune/archive/v1.0.4.tar.gz"
      sha256 "0c758cb4ac4f376ded6ee9dcd6406a931ad423b6df4dd3595a5f132f7545bce3"
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