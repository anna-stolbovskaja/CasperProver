/**
 * Client-side transaction building for wallet-signed on-chain operations.
 *
 * When a Casper wallet is connected, write operations (submit_proof,
 * register_agent) can be signed and submitted directly by the user's wallet
 * instead of the backend's server key. Standard CSPR.click flow:
 * build a ContractCallBuilder tx → clickRef.send() → wallet popup
 * → user signs → tx submitted to Casper testnet.
 *
 * Contract: proof_registry (deployed on casper-test)
 * Entry points: submit_proof, register_agent, revoke_proof
 */
import {
  Args,
  CLValue,
  ContractCallBuilder,
  PublicKey,
} from 'casper-js-sdk'
import type { ICSPRClickSDK } from '@make-software/csprclick-core-types'
import { getContractHash, getOnchainSync } from './onchain'

export const CASPER_CHAIN_NAME = 'casper-test'
// PROOF_REGISTRY_HASH is now a getter so a redeploy that lands in
// /onchain.json is picked up without a rebuild. Kept as an exported
// constant expression for callers that captured it once at import
// time; those callers now resolve through the on-chain manifest.
export const PROOF_REGISTRY_HASH: string =
  getContractHash('proof_registry') ??
  // Fallback to the compile-time snapshot's proof_registry entry — the
  // one shipped in SNAPSHOT inside onchain.ts. This branch is unreachable
  // in practice (SNAPSHOT always has the four deployed hashes) but keeps
  // the return type as string, not string | null, at the call sites.
  getOnchainSync().contracts.proof_registry?.contract_hash ??
  ''
export const PAYMENT_MOTES = 3_000_000_000 // 3 CSPR — sufficient for dictionary writes

export type LiveTxResult =
  | { ok: true; transactionHash: string }
  | { ok: false; cancelled: true }
  | { ok: false; cancelled: false; error: string }

/**
 * Submit a proof on-chain via the connected wallet.
 * Calls the proof_registry contract's `submit_proof` entry point.
 */
export async function submitProofOnChain(
  clickRef: ICSPRClickSDK,
  opts: {
    proofHash: string
    inputHash: string
    outputHash: string
    modelHash: string
    senderPublicKeyHex: string
  },
): Promise<LiveTxResult> {
  try {
    const tx = new ContractCallBuilder()
      .byHash(PROOF_REGISTRY_HASH)
      .entryPoint('submit_proof')
      .runtimeArgs(
        Args.fromMap({
          proof_hash: CLValue.newCLString(opts.proofHash),
          input_hash: CLValue.newCLString(opts.inputHash),
          output_hash: CLValue.newCLString(opts.outputHash),
          model_hash: CLValue.newCLString(opts.modelHash),
        }),
      )
      .from(PublicKey.fromHex(opts.senderPublicKeyHex))
      .chainName(CASPER_CHAIN_NAME)
      .payment(PAYMENT_MOTES)
      .build()

    const res = await clickRef.send(tx.toJSON() as object, opts.senderPublicKeyHex)

    if (res?.transactionHash) {
      return { ok: true, transactionHash: res.transactionHash }
    }
    if (res?.cancelled) {
      return { ok: false, cancelled: true }
    }
    const rawError = (res as any)?.error ?? (res as any)?.errorData ?? 'Unknown error from wallet SDK'
    return { ok: false, cancelled: false, error: typeof rawError === 'string' ? rawError : JSON.stringify(rawError) }
  } catch (err: any) {
    return { ok: false, cancelled: false, error: err?.message || String(err) }
  }
}

/**
 * Register an agent on-chain via the connected wallet.
 * Calls the proof_registry contract's `register_agent` entry point.
 */
export async function registerAgentOnChain(
  clickRef: ICSPRClickSDK,
  opts: {
    agentId: string
    modelHash: string
    senderPublicKeyHex: string
  },
): Promise<LiveTxResult> {
  try {
    const tx = new ContractCallBuilder()
      .byHash(PROOF_REGISTRY_HASH)
      .entryPoint('register_agent')
      .runtimeArgs(
        Args.fromMap({
          agent_id: CLValue.newCLString(opts.agentId),
          model_hash: CLValue.newCLString(opts.modelHash),
        }),
      )
      .from(PublicKey.fromHex(opts.senderPublicKeyHex))
      .chainName(CASPER_CHAIN_NAME)
      .payment(PAYMENT_MOTES)
      .build()

    const res = await clickRef.send(tx.toJSON() as object, opts.senderPublicKeyHex)

    if (res?.transactionHash) {
      return { ok: true, transactionHash: res.transactionHash }
    }
    if (res?.cancelled) {
      return { ok: false, cancelled: true }
    }
    const rawError = (res as any)?.error ?? (res as any)?.errorData ?? 'Unknown error'
    return { ok: false, cancelled: false, error: typeof rawError === 'string' ? rawError : JSON.stringify(rawError) }
  } catch (err: any) {
    return { ok: false, cancelled: false, error: err?.message || String(err) }
  }
}

/**
 * Revoke a proof on-chain via the connected wallet.
 * Calls the proof_registry contract's `revoke_proof` entry point.
 */
export async function revokeProofOnChain(
  clickRef: ICSPRClickSDK,
  opts: {
    proofId: string
    senderPublicKeyHex: string
  },
): Promise<LiveTxResult> {
  try {
    const tx = new ContractCallBuilder()
      .byHash(PROOF_REGISTRY_HASH)
      .entryPoint('revoke_proof')
      .runtimeArgs(
        Args.fromMap({
          proof_id: CLValue.newCLString(opts.proofId),
        }),
      )
      .from(PublicKey.fromHex(opts.senderPublicKeyHex))
      .chainName(CASPER_CHAIN_NAME)
      .payment(PAYMENT_MOTES)
      .build()

    const res = await clickRef.send(tx.toJSON() as object, opts.senderPublicKeyHex)

    if (res?.transactionHash) {
      return { ok: true, transactionHash: res.transactionHash }
    }
    if (res?.cancelled) {
      return { ok: false, cancelled: true }
    }
    const rawError = (res as any)?.error ?? (res as any)?.errorData ?? 'Unknown error'
    return { ok: false, cancelled: false, error: typeof rawError === 'string' ? rawError : JSON.stringify(rawError) }
  } catch (err: any) {
    return { ok: false, cancelled: false, error: err?.message || String(err) }
  }
}
