# contributing

## requirements

- go 1.24+
- rust nightly (for edition 2024 contracts)
- `wasm32-unknown-unknown` target

## build

```bash
cd engine && go build ./cmd/casperprover
cd contracts/proof-registry && cargo +nightly build --release --target wasm32-unknown-unknown --no-default-features
```

## test

```bash
cd engine && go test -race ./...
cd engine && go vet ./...
cd contracts/tests && cargo test --release
```

## style

- go: `gofmt` + `golangci-lint run`
- rust: `cargo fmt` + `cargo clippy`

## commits

conventional commits: `feat:`, `fix:`, `test:`, `docs:`, `refactor:`

## pull requests

fork → branch → test → pr against `main`

keep prs focused. one concern per pr.
