#!/bin/sh
# Prints a Homebrew formula for a release, with the checksums from dist/SHA256SUMS.
# Usage: scripts/formula.sh <version> <dist dir>, where <version> has no leading v.
set -eu

version=$1
dist=$2
base="https://github.com/tanem/stick-history/releases/download/v$version"

sum() {
	awk -v f="stick-history_${version}_$1.tar.gz" '$2 == f { print $1 }' "$dist/SHA256SUMS"
}

cat <<FORMULA
class StickHistory < Formula
  desc "Print the tracklist of a set from the History a Pioneer DJ player wrote to a USB stick"
  homepage "https://github.com/tanem/stick-history"
  version "$version"
  license "MIT"

  on_macos do
    on_arm do
      url "$base/stick-history_${version}_darwin_arm64.tar.gz"
      sha256 "$(sum darwin_arm64)"
    end
    on_intel do
      url "$base/stick-history_${version}_darwin_amd64.tar.gz"
      sha256 "$(sum darwin_amd64)"
    end
  end

  on_linux do
    on_intel do
      url "$base/stick-history_${version}_linux_amd64.tar.gz"
      sha256 "$(sum linux_amd64)"
    end
  end

  def install
    bin.install "stick-history"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/stick-history --help 2>&1")
  end
end
FORMULA
