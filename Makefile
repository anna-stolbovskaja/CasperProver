.PHONY: build test lint clean contracts contracts-test sdk-test gate3-demo

build:
	cd engine && go build -o ../bin/casperprover ./cmd/casperprover

# One-command Gate 3 agentic vertical slice — approve + malicious + conflict
# + abstain paths, no network I/O. Prints a JSON receipt to stdout and a
# human summary to stderr. Exit code non-zero on any scenario failure.
gate3-demo: build
	./bin/casperprover gate3

test:
	cd engine && go test -v -race ./...

lint:
	cd engine && golangci-lint run

contracts:
	cd contracts/proof-registry && cargo build --release --target wasm32-unknown-unknown --no-default-features
	cd contracts/verifier-gate && cargo build --release --target wasm32-unknown-unknown --no-default-features
	cd contracts/defi-mock && cargo build --release --target wasm32-unknown-unknown --no-default-features
	cd contracts/stake-slashing && cargo build --release --target wasm32-unknown-unknown --no-default-features
	cd contracts/stake-slashing-session && cargo build --release --target wasm32-unknown-unknown --no-default-features

sdk-test:
	cd sdk && go build ./...
	cd sdk && go vet ./...
	cd sdk && go test -race -v ./...

contracts-test:
	cd contracts/tests && cargo test --release

clean:
	rm -rf bin/ target/
