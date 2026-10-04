# Installing

## Homebrew

On macOS or Linux:

```sh
brew install tanem/tap/stick-history
```

The formula is in [tanem/homebrew-tap](https://github.com/tanem/homebrew-tap). Each release updates it, so `brew upgrade` installs the latest release.

## Go

With Go 1.23 or later:

```sh
go install github.com/tanem/stick-history@latest
```

## Release archives

Each [GitHub Release](https://github.com/tanem/stick-history/releases) has archives for:

- macOS, Apple silicon and Intel
- Windows, x86-64
- Linux, x86-64

Their checksums are in `SHA256SUMS` on the same release.

Each archive has a build provenance attestation. With the [GitHub CLI](https://cli.github.com), this checks that an archive was built by this repo's release workflow:

```sh
gh attestation verify stick-history_0.1.0_darwin_arm64.tar.gz --repo tanem/stick-history
```

## macOS and Gatekeeper

The macOS binaries are not signed or notarised:

- Homebrew does not quarantine a formula's download, so the binary it installs runs without a Gatekeeper prompt.
- A binary from an archive downloaded with a browser is quarantined, and Gatekeeper can block it.

## What `--version` prints

`stick-history --version` prints:

- The version of the release for a release build, as in `stick-history 0.1.0`.
- The version of the module for a build installed with `go install github.com/tanem/stick-history@latest`.
- `stick-history (devel)` for any other build.
