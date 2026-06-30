const API_BASE = 'https://casperprover-api.onrender.com'

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const url = `${API_BASE}${path}`
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    ...opts,
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`API ${res.status}: ${body}`)
  }
  return res.json()
}

export interface ProofRecord {
  id: string
  agent: string
  proof_hash: string
  input_hash: string
  output_hash: string
  model_hash: string
  merkle_root: string
  merkle_path: string[]
  leaf_index: number
  timestamp: number
  valid: boolean
  revoked: boolean
  use_case: string
  public_key?: string
  deploy_hash?: string
  generation_ms: number
  mode?: string
}

export interface ProofListResponse {
  proofs: ProofRecord[]
  total: number
  page: number
  limit: number
}

export interface HealthResponse {
  status: string
  version: string
  uptime_s: number
  total_proofs: number
  chain: string
  contracts: {
    proof_registry: string
    verifier_gate: string
    defi_mock: string
  }
}

export interface StatsResponse {
  total_proofs: number
  valid_proofs: number
  revoked_proofs: number
  unique_agents: number
  avg_generation_ms: number
  max_merkle_depth: number
  use_cases: Record<string, number>
}

export interface VerifyResponse {
  proof_id: string
  valid: boolean
  revoked: boolean
  verified?: boolean
  error?: string
  checks?: {
    input_hash_match: boolean
    output_hash_match: boolean
    model_hash_match: boolean
    commit_valid: boolean
    merkle_valid: boolean
  }
}

export function getHealth(): Promise<HealthResponse> {
  return request('/health')
}

export function getProofs(params?: {
  agent?: string
  public_key?: string
  mode?: string
  page?: number
  limit?: number
}): Promise<ProofListResponse> {
  const q = new URLSearchParams()
  if (params?.agent) q.set('agent', params.agent)
  if (params?.public_key) q.set('public_key', params.public_key)
  if (params?.mode) q.set('mode', params.mode)
  if (params?.page) q.set('page', String(params.page))
  if (params?.limit) q.set('limit', String(params.limit))
  const qs = q.toString()
  return request(`/proofs${qs ? '?' + qs : ''}`)
}

export function getProof(id: string): Promise<ProofRecord> {
  return request(`/proofs/${id}`)
}

export function createProof(
  data: { agent: string; input: string; output: string; model: string; use_case?: string; mode?: string },
  publicKey?: string,
): Promise<ProofRecord> {
  const headers: Record<string, string> = {}
  if (publicKey) headers['X-Public-Key'] = publicKey
  return request('/proofs', {
    method: 'POST',
    body: JSON.stringify(data),
    headers,
  })
}

export function batchProofs(
  proofs: { agent: string; input: string; output: string; model: string; use_case?: string }[],
  mode?: string,
  publicKey?: string,
): Promise<{ proofs: ProofRecord[]; generated: number }> {
  const headers: Record<string, string> = {}
  if (publicKey) headers['X-Public-Key'] = publicKey
  return request('/proofs/batch', {
    method: 'POST',
    body: JSON.stringify({ proofs, mode: mode || 'local' }),
    headers,
  })
}

export function verifyProof(data: {
  proof_id: string
  input?: string
  output?: string
  model?: string
}): Promise<VerifyResponse> {
  return request('/verify', { method: 'POST', body: JSON.stringify(data) })
}

export function revokeProof(id: string, reason?: string): Promise<{ proof_id: string; revoked: boolean }> {
  return request(`/proofs/${id}/revoke`, {
    method: 'POST',
    body: JSON.stringify({ reason: reason || '' }),
  })
}

export function exportProof(id: string): Promise<Record<string, unknown>> {
  return request(`/proofs/${id}/export`)
}

export function getStats(): Promise<StatsResponse> {
  return request('/stats')
}

export function kycCheck(proofId: string): Promise<{ proof_id: string; verified: boolean; timestamp: number }> {
  return request('/kyc/check', { method: 'POST', body: JSON.stringify({ proof_id: proofId }) })
}

export function kycGrant(user: string, proofId: string): Promise<{ user: string; whitelisted: boolean; proof_id: string }> {
  return request('/kyc/grant', { method: 'POST', body: JSON.stringify({ user, proof_id: proofId }) })
}

export function kycWhitelist(user: string): Promise<{ user: string; whitelisted: boolean }> {
  return request(`/kyc/whitelist/${user}`)
}
