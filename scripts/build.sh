#!/bin/sh
# Builds a static binary per platform into dist/ and writes SHA256SUMS.
# Usage: scripts/build.sh <version>, where <version> has no leading v.
set -eu

version=$1
rm -rf dist
mkdir -p dist

for target in darwin/arm64 darwin/amd64 windows/amd64 linux/amd64; do
	os=${target%/*}
	arch=${target#*/}
	name="stick-history_${version}_${os}_${arch}"
	dir="dist/$name"
	mkdir -p "$dir"
	bin=stick-history
	if [ "$os" = windows ]; then
		bin=stick-history.exe
	fi
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$dir/$bin" .
	cp LICENSE README.md "$dir/"
	if [ "$os" = windows ]; then
		(cd dist && zip -q -r "$name.zip" "$name")
	else
		tar -C dist -czf "dist/$name.tar.gz" "$name"
	fi
	rm -r "$dir"
done

if command -v sha256sum >/dev/null; then
	(cd dist && sha256sum ./* | sed 's|\./||' > SHA256SUMS)
else
	(cd dist && shasum -a 256 ./* | sed 's|\./||' > SHA256SUMS)
fi
