
const BASE_URL = 'https://casperprover-api.onrender.com';

// --- Utility Types ---
type ApiResponse<T> = {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
};

async function fetcher<T>(
  endpoint: string,
  options?: RequestInit
): Promise<ApiResponse<T>> {
  try {
    const response = await fetch(`${BASE_URL}${endpoint}`, {
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    });

    const data = await response.json();

    if (!response.ok) {
      return { success: false, error: data.message || 'An unknown error occurred', message: data.message };
    }

    return { success: true, data };
  } catch (error) {
    console.error(`API call to ${endpoint} failed:`, error);
    return { success: false, error: (error as Error).message || 'Network error' };
  }
}

// --- Main API Schemas & Types ---

// Health
export interface HealthResponse {
  status: string
  version: string
  contracts: Record<string, number>
}


// Proofs
export interface Proof {
  id: string
  agentId: string
  inputHash: string
  outputHash: string
  proofData: string
  createdAt: string
  status: any
  merklePath?: string[]
}

export interface ProofsListResponse {
  proofs: any[]
  total: number
  page: number
  limit: number
}


export interface CreateProofRequest {
  agentId: string
  inputHash: string
  outputHash: string
  proofData: string
}


export type BatchProofRequest = CreateProofRequest[];

export interface VerifyProofRequest {
  proofId: string
}

export interface VerifyProofResponse {
  isValid: boolean
  message: string
}

// Stats
export interface StatsResponse {
  totalProofs: number
  validProofs: number
  revokedProofs: number
  uniqueAgents: number
  averageGenerationTimeMs: number
}


// KYC
export interface KYCStatusRequest {
  userId: string
}

export interface KYCStatusResponse {
  userId: string
  isWhitelisted: boolean
  status: string
}

export interface KYCGrantRequest {
  userId: string
  reason?: string
}

export interface KYCGrantResponse {
  userId: string
  granted: boolean
  message: string
}

export interface KYCWhitelistResponse {
  users: string[]
}

// Inference
export interface RegisterModelRequest {
  modelName: string
  modelHash: string
  verifierContract: string
  description?: string
}

export interface RegisterModelResponse {
  modelId: string
  message: string
}

export interface ModelDetails {
  modelId: string
  modelName: string
  modelHash: string
  verifierContract: string
  description?: string
  registeredAt: string
}

export interface InferenceProveRequest {
  modelId: string
  inputData: string
  agentId: string
}

export interface InferenceProveResponse {
  proofId: string
  proofData: string
  outputHash: string
  message: string
}

export interface InferenceVerifyRequest {
  modelId: string
  proofId: string
  inputData: string
  outputHash: string
}

export interface InferenceVerifyResponse {
  isValid: boolean
  message: string
}

// Aggregation
export interface CreateBatchRequest {
  batchName: string
  description?: string
}

export interface CreateBatchResponse {
  batchId: string
  message: string
}

export interface AddProofToBatchRequest {
  batchId: string
  proofId: string
}

export interface AddProofToBatchResponse {
  batchId: string
  proofId: string
  message: string
}

export interface FinalizeBatchRequest {
  batchId: string
}

export interface FinalizeBatchResponse {
  batchId: string
  merkleRoot: string
  finalProof: string
  message: string
}

export interface BatchDetails {
  batchId: string
  batchName: string
  description?: string
  proofIds: string[]
  status: any
  merkleRoot?: string
  finalProof?: string
  createdAt: string
  finalizedAt?: string
}

// ZK
export interface ZKVerifyGroth16Request {
  proof: string
  publicSignals: string[]
}

export interface ZKVerifyGroth16Response {
  isValid: boolean
  message: string
}

export interface ZKBatchVerifyRequest {
  proofIds: string[]
}

export interface ZKBatchVerifyResponse {
  results: Record<string, number>
  message: string
}

export interface ZKChallengeRequest {
  challengerId: string
  proofId: string
  challengeData: string
}

export interface ZKChallengeResponse {
  challengeId: string
  message: string
}

export interface ZKChallengeDetails {
  challengeId: string
  challengerId: string
  proofId: string
  challengeData: string
  status: any
  resolvedAt?: string
  resolution?: string
}

// PQ
export interface PQSignSphincsRequest {
  message: string
  privateKey?: string
}

export interface PQSignSphincsResponse {
  signature: string
  publicKey: string
}

export interface PQVerifySphincsRequest {
  message: string
  signature: string
  publicKey: string
}

export interface PQVerifySphincsResponse {
  isValid: boolean
}

export interface PQHybridSignRequest {
  message: string
  classicalPrivateKey?: string
  pqPrivateKey?: string
}

export interface PQHybridSignResponse {
  classicalSignature: string
  pqSignature: string
  classicalPublicKey: string
  pqPublicKey: string
}

export interface PQHybridVerifyRequest {
  message: string
  classicalSignature: string
  pqSignature: string
  classicalPublicKey: string
  pqPublicKey: string
}

export interface PQHybridVerifyResponse {
  isValid: boolean
}


// --- API Functions ---

// Main
export const getHealth = () => fetcher<HealthResponse>('/health');
export const getProofs = (agent?: string, page: number = 1, limit: number = 10) =>
  fetcher<ProofsListResponse>(`/proofs?page=${page}&limit=${limit}${agent ? `&agent=${agent}` : ''}`);
export const getProofById = (id: string) => fetcher<Proof>(`/proofs/${id}`);
export const createProof = (data: CreateProofRequest) =>
  fetcher<Proof>('/proofs', { method: 'POST', body: JSON.stringify(data) });
export const createBatchProofs = (data: BatchProofRequest) =>
  fetcher<ProofsListResponse>('/proofs/batch', { method: 'POST', body: JSON.stringify(data) });
export const verifyProof = (data: VerifyProofRequest) =>
  fetcher<VerifyProofResponse>('/verify', { method: 'POST', body: JSON.stringify(data) });
export const revokeProof = (id: string) =>
  fetcher<{ message: string }>(`/proofs/${id}/revoke`, { method: 'POST' });
export const exportProof = (id: string) =>
  fetcher<string>(`/proofs/${id}/export`, { method: 'GET' }); // Returns raw proof data

export const getStats = () => fetcher<StatsResponse>('/stats');

// KYC
export const checkKycStatus = (data: KYCStatusRequest) =>
  fetcher<KYCStatusResponse>('/kyc/check', { method: 'POST', body: JSON.stringify(data) });
export const grantKycAccess = (data: KYCGrantRequest) =>
  fetcher<KYCGrantResponse>('/kyc/grant', { method: 'POST', body: JSON.stringify(data) });
export const getKycWhitelist = (user: string) =>
  fetcher<KYCWhitelistResponse>(`/kyc/whitelist/${user}`);

// Inference
export const registerModel = (data: RegisterModelRequest) =>
  fetcher<RegisterModelResponse>('/inference/register-model', { method: 'POST', body: JSON.stringify(data) });
export const getModelById = (id: string) =>
  fetcher<ModelDetails>(`/inference/model/${id}`);
export const inferenceProve = (data: InferenceProveRequest) =>
  fetcher<InferenceProveResponse>('/inference/prove', { method: 'POST', body: JSON.stringify(data) });
export const inferenceVerify = (data: InferenceVerifyRequest) =>
  fetcher<InferenceVerifyResponse>('/inference/verify', { method: 'POST', body: JSON.stringify(data) });

// Aggregation
export const createAggregationBatch = (data: CreateBatchRequest) =>
  fetcher<CreateBatchResponse>('/aggregation/create-batch', { method: 'POST', body: JSON.stringify(data) });
export const addProofToAggregationBatch = (data: AddProofToBatchRequest) =>
  fetcher<AddProofToBatchResponse>('/aggregation/add-proof', { method: 'POST', body: JSON.stringify(data) });
export const finalizeAggregationBatch = (data: FinalizeBatchRequest) =>
  fetcher<FinalizeBatchResponse>('/aggregation/finalize', { method: 'POST', body: JSON.stringify(data) });
export const getAggregationBatchById = (id: string) =>
  fetcher<BatchDetails>(`/aggregation/batch/${id}`);

// ZK
export const verifyGroth16 = (data: ZKVerifyGroth16Request) =>
  fetcher<ZKVerifyGroth16Response>('/zk/verify-groth16', { method: 'POST', body: JSON.stringify(data) });
export const batchVerifyZK = (data: ZKBatchVerifyRequest) =>
  fetcher<ZKBatchVerifyResponse>('/zk/batch-verify', { method: 'POST', body: JSON.stringify(data) });
export const challengeZK = (data: ZKChallengeRequest) =>
  fetcher<ZKChallengeResponse>('/zk/challenge', { method: 'POST', body: JSON.stringify(data) });
export const getZKChallengeById = (id: string) =>
  fetcher<ZKChallengeDetails>(`/zk/challenge/${id}`);

// PQ
export const signSphincs = (data: PQSignSphincsRequest) =>
  fetcher<PQSignSphincsResponse>('/pq/sign-sphincs', { method: 'POST', body: JSON.stringify(data) });
export const verifySphincs = (data: PQVerifySphincsRequest) =>
  fetcher<PQVerifySphincsResponse>('/pq/verify-sphincs', { method: 'POST', body: JSON.stringify(data) });
export const hybridSign = (data: PQHybridSignRequest) =>
  fetcher<PQHybridSignResponse>('/pq/hybrid-sign', { method: 'POST', body: JSON.stringify(data) });
export const hybridVerify = (data: PQHybridVerifyRequest) =>
  fetcher<PQHybridVerifyResponse>('/pq/hybrid-verify', { method: 'POST', body: JSON.stringify(data) });
