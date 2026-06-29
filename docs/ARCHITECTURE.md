# Architecture

## System Overview

```mermaid
graph TB
    subgraph Engine
        CLI[CLI Entry] --> PR[Prover]
        CLI --> VR[Verifier]
        CLI --> KYC[KYC Demo]
        CLI --> API[API Server]
        PR --> H[Hasher SHA-256]
        PR --> MT[Merkle Tree]
        VR --> MT
    end

    subgraph Contracts
        REG[proof-registry]
        VG[verifier-gate]
        DM[defi-mock]
        VG --> REG
        DM --> VG
    end

    subgraph Casper
        RPC[Casper RPC]
    end

    API --> SUB[Submitter]
    SUB --> RPC
    RPC --> REG
```

## Proof Generation

```mermaid
sequenceDiagram
    participant A as Agent
    participant E as Engine
    participant H as Hasher
    participant M as Merkle

    A->>E: (input, output, model)
    E->>H: SHA-256(input)
    E->>H: SHA-256(output)
    E->>H: SHA-256(model)
    H-->>M: leaf_0, leaf_1, leaf_2

    M->>M: Build tree bottom-up
    Note over M: root = H(H(leaf_0||leaf_1) || leaf_2)
    M-->>E: (root, path, leaves)
    E-->>A: Proof{id, root, path, verified}
```

## KYC Whitelisting Flow

```mermaid
sequenceDiagram
    participant U as User Agent
    participant E as Engine
    participant PR as proof-registry
    participant VG as verifier-gate
    participant DM as defi-mock

    U->>E: prove(kyc_input, kyc_result, model)
    E->>PR: submit_proof(root, agent, metadata)
    PR-->>E: proof_id

    U->>DM: check_kyc(user, proof_id)
    DM->>VG: verify_proof(proof_id)
    VG->>PR: get_proof(proof_id)
    PR-->>VG: proof data
    VG-->>DM: is_valid = true

    DM->>DM: grant_access(user)
    Note over DM: User whitelisted for DeFi
```

## Verification Logic

```
Given: leaf L, path P = [p_0, p_1, ...], root R

verify(L, P, R):
    h = H(L)
    for p_i in P:
        h = H(h || p_i)   if index is even
        h = H(p_i || h)   if index is odd
    return h == R
```
