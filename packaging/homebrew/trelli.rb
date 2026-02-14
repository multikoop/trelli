class Trelli < Formula
  desc "Efficient Trello CLI for boards, lists, cards, comments, and checklists"
  homepage "https://github.com/multikoop/trelli"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/multikoop/trelli/releases/download/v#{version}/trelli_#{version}_darwin_arm64.tar.gz"
      sha256 "TODO_FILL_FROM_RELEASE_CHECKSUMS_DARWIN_ARM64"
    else
      url "https://github.com/multikoop/trelli/releases/download/v#{version}/trelli_#{version}_darwin_amd64.tar.gz"
      sha256 "TODO_FILL_FROM_RELEASE_CHECKSUMS_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/multikoop/trelli/releases/download/v#{version}/trelli_#{version}_linux_arm64.tar.gz"
      sha256 "TODO_FILL_FROM_RELEASE_CHECKSUMS_LINUX_ARM64"
    else
      url "https://github.com/multikoop/trelli/releases/download/v#{version}/trelli_#{version}_linux_amd64.tar.gz"
      sha256 "TODO_FILL_FROM_RELEASE_CHECKSUMS_LINUX_AMD64"
    end
  end

  def install
    bin.install "trelli"
  end

  test do
    assert_match "trelli - Efficient Trello CLI", shell_output("#{bin}/trelli --help")
  end
end
