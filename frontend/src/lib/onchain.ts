// Single source of truth for on-chain contract addresses.
//
// The canonical manifest lives at /deploy-out/onchain.json in the repo
// root; it is copied into frontend/public/onchain.json at build time by
// scripts/sync-onchain.sh (invoked from the frontend build) so the
// static site can fetch it at runtime.
//
// Historically the four testnet contract hashes were duplicated inline
// across Contracts.tsx / Models.tsx / AgentDemo.tsx / Playground.tsx /
// liveTx.ts / CtaFooter.tsx. Every redeploy meant hunting them down.
// Now those components import from here, and the values come from the
// runtime fetch — with a compile-time snapshot as fallback so a first
// paint never blocks on the network.
//
// If you ever need to swap a deployed contract, you only edit two
// places: `deploy-out/onchain.json` (root of repo, canonical) and the
// SNAPSHOT constant below (compile-time fallback). Every other
// reference resolves from `getContractHashes()` at runtime.

export type ContractName =
  | 'proof_registry'
  | 'verifier_gate'
  | 'defi_mock'
  | 'stake_slashing'
  | 'proof_of_inference'
  | 'model_registry'
  | 'proof_aggregation'

export interface ContractRecord {
  contract_hash: string
  contract_package_hash?: string
  deploy_hash?: string
  deployer?: string
  version?: number
  deployed_at?: string
  source?: string
  entry_points?: string[]
  notes?: string
}

export interface OnchainManifest {
  network: string
  chain_name: string
  project: string
  explorer: string
  cspr_cloud: string
  contracts: Partial<Record<ContractName, ContractRecord>>
}

// SNAPSHOT is the last-known-good manifest baked into the build. It is
// used both as an initial synchronous value (so React components can
// render immediately) and as a fallback if the runtime fetch of
// /onchain.json fails.
//
// Keep this in sync with deploy-out/onchain.json. A CI check will
// enforce it once the shared script is added.
const SNAPSHOT: OnchainManifest = {
  network: 'casper-test',
  chain_name: 'casper-test',
  project: 'CasperProver',
  explorer: 'https://testnet.cspr.live',
  cspr_cloud: 'https://testnet.cspr.cloud',
  contracts: {
    proof_registry: {
      contract_hash: '96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708',
      contract_package_hash: '894d167e22c70462af9b265c23fb14126af30b91644ca696f4a6a2f3311e5309',
      deploy_hash: 'd64299b651750b6996595d81b812038750c353f5220b5e61cd6c129e90a07d56',
      deployed_at: '2026-06-29T09:33:43Z',
    },
    verifier_gate: {
      contract_hash: 'a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3',
      contract_package_hash: 'bd6b2ff66e2416c6b532932d328402d3ed648dde0c6ff9d556f2b2635fbe6115',
      deploy_hash: 'c1320d182c0323e671183cb7aef603f1bb19b86f97637e3a386ae14dd28422ff',
      deployed_at: '2026-06-29T09:39:01Z',
    },
    defi_mock: {
      contract_hash: 'fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef',
      contract_package_hash: '54757fa72e6ca1898f3a8bc6e5af1d643b120a8c8605e0b1581fdcc3b76f9a04',
      deploy_hash: '7e590fb94fb0c3e41cd01e44e14157c3e537f4766a546d1dedbe5b137210625e',
      deployed_at: '2026-07-07T13:19:48Z',
    },
    stake_slashing: {
      contract_hash: '1ad1b3d94be631532d6daf3a195fafc9dfe8a16504e87d87784d51089b983d52',
      contract_package_hash: 'e33812f9c9c88e0d3202fdd7a7718cce686cbd21bc673c216b6dbf23c26e2947',
      deploy_hash: 'ac4712a3ecc29c058330df88781d488f61c3993b7ee2720c2024fc2a231d2532',
      deployed_at: '2026-07-19T07:33:00Z',
    },
  },
}

let cache: OnchainManifest = SNAPSHOT
let inflight: Promise<OnchainManifest> | null = null

/**
 * Load the canonical on-chain manifest at runtime. Cheap after the
 * first call — the fetch is memoized until the tab reloads. Components
 * that must react to a redeploy without a full reload can call this in
 * a useEffect and update local state.
 *
 * If the network fetch fails (offline, 404, malformed JSON) we keep
 * serving the compile-time snapshot. The intent is to degrade
 * gracefully, not to hard-fail rendering.
 */
export async function loadOnchainManifest(): Promise<OnchainManifest> {
  if (inflight) return inflight
  inflight = (async () => {
    try {
      const resp = await fetch('/onchain.json', { cache: 'no-store' })
      if (!resp.ok) return SNAPSHOT
      const parsed = (await resp.json()) as OnchainManifest
      if (!parsed || !parsed.contracts) return SNAPSHOT
      cache = parsed
      return parsed
    } catch {
      return SNAPSHOT
    }
  })()
  return inflight
}

/**
 * Synchronous accessor for consumers that must render before the fetch
 * resolves. Returns the compile-time snapshot on first call and the
 * runtime-fetched manifest after loadOnchainManifest() has resolved.
 */
export function getOnchainSync(): OnchainManifest {
  return cache
}

/**
 * Convenience: 32-byte hex contract_hash for a given contract, or `null`
 * for known-written-but-not-deployed contracts. Callers used to inline
 * these — see git blame for the migration.
 */
export function getContractHash(name: ContractName): string | null {
  const rec = cache.contracts[name]
  return rec?.contract_hash ?? null
}

/**
 * Kick off the runtime fetch once at module load so the first render
 * after mount already has the fresh manifest.
 */
if (typeof window !== 'undefined') {
  void loadOnchainManifest()
}
