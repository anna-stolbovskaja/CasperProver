.PHONY: build test lint clean contracts contracts-test

build:
	cd engine && go build -o ../bin/casperprover ./cmd/casperprover

test:
	cd engine && go test -v -race ./...

lint:
	cd engine && golangci-lint run

contracts:
	cd contracts/proof-registry && cargo build --release --target wasm32-unknown-unknown
	cd contracts/verifier-gate && cargo build --release --target wasm32-unknown-unknown
	cd contracts/defi-mock && cargo build --release --target wasm32-unknown-unknown

contracts-test:
	cd contracts && cargo test

clean:
	rm -rf bin/ target/
