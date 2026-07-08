.PHONY: build test lint clean contracts contracts-test sdk-test

build:
	cd engine && go build -o ../bin/casperprover ./cmd/casperprover

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
