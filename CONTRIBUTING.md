# Contributing

- Open an issue before starting work, so the change can be agreed first.
- CI must pass. It checks that:
  - `gofmt -l .` prints nothing.
  - `go vet ./...` passes.
  - `golangci-lint run` passes.
  - `go test ./...` passes on Linux, macOS and Windows.
  - The same tests pass on Go 1.23 on Linux.
  - The release build in `scripts/` succeeds.
- Write commit messages as plain imperative sentences. Conventional Commits are not used.
- The maintainer applies the release label to a pull request. The `release-label` check fails until then.
- Never add a real `export.pdb` as a fixture. It lists a whole collection. The tests build a synthetic one in `internal/pdbtest`.

## Tests

Run the tests:

```sh
go test ./...
```

The tests build a synthetic `export.pdb` in memory, in `internal/pdbtest`. It holds no track, artist or History from a real collection. `*.pdb` is ignored by git, so a real `export.pdb` is not committed by accident.

The parser has a fuzz target. `go test` runs it on its seed, and this fuzzes it:

```sh
go test -fuzz=FuzzParse ./pdb
```

The test in `internal/doclinks` checks every Markdown file in the repo. It fails on a relative link whose target file or directory does not exist. It does not check a `#fragment` or an external URL.
