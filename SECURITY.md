# Security

## Reporting

Report security issues via [GitHub Security Advisories](https://github.com/anna-stolbovskaja/CasperProver/security/advisories/new).

Do not open public issues for vulnerabilities.

## Status

Testnet prototype. Internal audit completed. No external audit.

## Scope

- **Contracts:** proof-registry, verifier-gate, defi-mock, stake-slashing, proof-of-inference, model-registry, proof-aggregation
- **Engine:** hasher, prover, verifier, submitter, ZK (gnark), PQ crypto (ML-DSA-65, Lamport)
- **SDK:** client.go, python_client.py, mcp_server.go

## Keys

Never commit deployer secrets. `.env.example` has placeholders only. Testnet keys are disposable.

## Limitations

See [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md).
