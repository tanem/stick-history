# Releasing

The release workflow, `.github/workflows/release.yml`, runs every Monday and can be started by hand. No tag is pushed by hand.

It uses [tanem/release-action](https://github.com/tanem/release-action) to derive the next version from the labels on the pull requests merged since the last release. [AGENTS.md](../AGENTS.md) has the rules that map a label to a version bump.

When nothing was merged, it releases nothing. When there is something to release, the workflow:

1. Tags the commit and creates the GitHub Release, with notes GitHub generates from the pull requests.
2. Builds the archives with `scripts/build.sh` and attests them.
3. Attaches the archives and `SHA256SUMS` to the release.
4. Writes the formula with `scripts/formula.sh` and commits it to `tanem/homebrew-tap`.

[adr/0001-keep-build-scripts-over-goreleaser.md](adr/0001-keep-build-scripts-over-goreleaser.md) records why the build uses these scripts and not GoReleaser.

## The deploy key check

Before it tags anything, the workflow checks that the tap's deploy key can push. It clones `tanem/homebrew-tap` with the key and runs `git push --dry-run` against it. GitHub refuses a read-only deploy key at that point. A key that is missing, cannot reach the tap or cannot write to it therefore fails the run before a tag or a release exists.

## A dry run

A run started by hand with the dry-run option logs the version it would release and changes nothing. It skips the deploy key check and does not need the key.

## Finishing a failed release

A run that fails after the tag leaves a release without some of its archives, its attestations or its formula. Running the workflow again does not finish it. To finish it, start the workflow by hand with the `version` input set to that version, without a leading `v`, as in `0.1.0`.

That run creates no tag and no release. It fails before building if the tag `v<version>` or its GitHub Release does not exist. Otherwise it:

1. Checks out the tag.
2. Builds and attests the archives.
3. Uploads them in place of any files already on the release.
4. Commits the formula to the tap when it differs from the one there. When the tap already holds the formula of a later version, the formula is left as it is.

Two builds of a version are not byte-identical. Resuming a release that was already complete therefore replaces its archives and commits a formula with the new checksums. The attestations of the replaced archives stay in the attestation store.

`version` and the dry-run option cannot be set together. A run with both fails.
