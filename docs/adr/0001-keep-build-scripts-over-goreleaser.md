# Keep the build scripts instead of GoReleaser

`scripts/build.sh` and `scripts/formula.sh` produce the release archives, the checksum file and the Homebrew formula. GoReleaser does not.

There are two reasons:

- `tanem/release-action` creates the tag and the GitHub Release. The workflow only needs to upload what the scripts build to that release, so GoReleaser's own release step would go unused.
- The scripts add no third-party action to the release job. That job holds write permission on this repo and the deploy key for the tap.
