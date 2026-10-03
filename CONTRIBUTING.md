# Contributing

- Open an issue before starting work, so the change can be agreed first.
- CI must pass: `gofmt -l .` prints nothing, `go vet ./...`, `golangci-lint run`, `go test ./...` on Linux, macOS and Windows and on Go 1.23, and the release build in `scripts/`.
- Write commit messages as plain imperative sentences. Conventional Commits are not used.
- The maintainer applies the release label to a pull request. The `release-label` check fails until then.
- Never add a real `export.pdb` as a fixture. It lists a whole collection. The tests build a synthetic one in `internal/pdbtest`.
