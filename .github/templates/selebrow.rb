class Selebrow < Formula
  desc "Run WebDriver/Playwright UI tests in Docker or Kubernetes environments"
  homepage "https://selebrow.dev/"
  version "$SELEBROW_VERSION"
  license "Apache-2.0"

  if OS.mac? && Hardware::CPU.intel?
    url "https://github.com/selebrow/selebrow/releases/download/v#{version}/selebrow-darwin-amd64"
    sha256 "$SELEBROW_DARWIN_AMD64"

    def install
      bin.install "selebrow-darwin-amd64" => "selebrow"
    end
  end

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/selebrow/selebrow/releases/download/v#{version}/selebrow-darwin-arm64"
    sha256 "$SELEBROW_DARWIN_ARM64"

    def install
      bin.install "selebrow-darwin-arm64" => "selebrow"
    end
  end

  if OS.linux? && Hardware::CPU.intel?
    url "https://github.com/selebrow/selebrow/releases/download/v#{version}/selebrow-linux-amd64"
    sha256 "$SELEBROW_LINUX_AMD64"

    def install
      bin.install "selebrow-linux-amd64" => "selebrow"
    end
  end

  if OS.linux? && Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
    url "https://github.com/selebrow/selebrow/releases/download/v#{version}selebrow-linux-arm64"
    sha256 "$SELEBROW_LINUX_ARM64"

    def install
      bin.install "selebrow-linux-arm64" => "selebrow"
    end
  end

  test do
    output = shell_output("#{bin}/selebrow -h 2>&1", 1)
    assert_match "selebrow build v#{version}", output
  end
end
