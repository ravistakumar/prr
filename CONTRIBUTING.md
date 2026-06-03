# Contributing to prr

Thanks for your interest! prr is written in Go and uses standard tooling.

## Development

```bash
make build   # build the binary
make test    # run all tests
make lint    # golangci-lint
make fmt     # gofmt
```

## Conventions

- Functional core, imperative shell: keep logic in `config`/`signals`/`optimize`
  (pure, unit-tested) and IO in `agent`/`interview`/`handoff`.
- New agent CLIs: add a constructor in `internal/agent` — one file, behind the
  `Agent` interface.
- TDD: write the failing test first. Every PR runs `go test ./...` and lint.
