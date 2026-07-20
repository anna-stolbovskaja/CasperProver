.PHONY: build test lint clean contracts contracts-test sdk-test sync-onchain

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

# Sync canonical on-chain manifest from deploy-out/onchain.json into consumer
# surfaces (frontend/public/onchain.json). Run after every redeploy.
sync-onchain:
	./scripts/sync-onchain-manifest.sh
