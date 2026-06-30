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
}

export interface HealthResponse {
  status: string
  version: string
}

export function getHealth(): Promise<HealthResponse> {
  return request('/health')
}

export function getProofs(): Promise<ProofRecord[]> {
  return request('/proofs')
}

export function getProof(id: string): Promise<ProofRecord> {
  return request(`/proofs/${id}`)
}

export function createProof(data: {
  agent: string
  input: string
  output: string
  model: string
}): Promise<ProofRecord> {
  return request('/proofs', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}
