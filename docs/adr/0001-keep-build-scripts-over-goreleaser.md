# Keep the build scripts instead of GoReleaser

The release archives, the checksum file and the Homebrew formula are produced by `scripts/build.sh` and `scripts/formula.sh`, not by GoReleaser. `tanem/release-action` creates the tag and the GitHub Release, so the workflow only needs to upload what the scripts build to that release, and GoReleaser's own release step would go unused. The scripts also add no third-party action to the release job, which holds write permission on this repo and the deploy key for the tap.
