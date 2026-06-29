# security

## reporting

report security issues via [github security advisories](https://github.com/anna-stolbovskaja/CasperProver/security/advisories/new) or email anna.stolbovskaja@gmail.com.

do not open public issues for vulnerabilities.

## status

testnet prototype. internal audit completed (18 findings, risk 2/10). no external audit.

## scope

- contracts: proof-registry, verifier-gate, defi-mock
- engine: hasher, prover, verifier, submitter
- sdk: client.go, python_client.py, mcp_server.go

## keys

never commit deployer secrets. `.env.example` has placeholders only. testnet keys are disposable.

## limitations

see [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md).
