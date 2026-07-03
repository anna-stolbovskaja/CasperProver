import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { Play, Code, AlertTriangle, Loader2 } from 'lucide-react';
import * as api from '../../lib/api'; // Import all API functions and types
import { toast } from '../ui/toast';

// Define a structure for each endpoint in the playground
interface EndpointConfig {
  name: string;
  method: 'GET' | 'POST';
  path: string;
  description: string;
  exampleBody?: string;
  apiCall: (data?: any) => Promise<api.ApiResponse<any>>;
  params?: { name: string; type: 'string' | 'number'; optional?: boolean; default?: any }[];
}

const playgroundEndpoints: EndpointConfig[] = [
  {
    name: 'GET /health',
    method: 'GET',
    path: '/health',
    description: 'Check the health status of the API.',
    apiCall: api.getHealth,
  },
  {
    name: 'GET /proofs',
    method: 'GET',
    path: '/proofs',
    description: 'Retrieve a list of proofs with optional filtering and pagination.',
    apiCall: (params) => api.getProofs(params?.agent, params?.page, params?.limit),
    params: [
      { name: 'agent', type: 'string', optional: true, default: '' },
      { name: 'page', type: 'number', optional: true, default: 1 },
      { name: 'limit', type: 'number', optional: true, default: 10 },
    ],
  },
  {
    name: 'GET /proofs/{id}',
    method: 'GET',
    path: '/proofs/{id}',
    description: 'Retrieve details for a specific proof by ID.',
    apiCall: (params) => api.getProofById(params.id),
    params: [{ name: 'id', type: 'string', optional: false, default: 'example-proof-id' }],
  },
  {
    name: 'POST /proofs',
    method: 'POST',
    path: '/proofs',
    description: 'Create a new ZK proof.',
    exampleBody: JSON.stringify({
      agentId: 'agent-123',
      inputHash: '0xabc123def456',
      outputHash: '0x789ghi012jkl',
      proofData: JSON.stringify({
        a: ['0x1', '0x2'],
        b: [['0x3', '0x4'], ['0x5', '0x6']],
        c: ['0x7', '0x8'],
      }),
    }, null, 2),
    apiCall: api.createProof,
  },
  {
    name: 'POST /verify',
    method: 'POST',
    path: '/verify',
    description: 'Verify an existing ZK proof.',
    exampleBody: JSON.stringify({
      proofId: 'example-proof-id',
    }, null, 2),
    apiCall: api.verifyProof,
  },
  {
    name: 'POST /proofs/{id}/revoke',
    method: 'POST',
    path: '/proofs/{id}/revoke',
    description: 'Revoke a specific ZK proof.',
    apiCall: (params) => api.revokeProof(params.id),
    params: [{ name: 'id', type: 'string', optional: false, default: 'example-proof-id' }],
  },
  {
    name: 'GET /stats',
    method: 'GET',
    path: '/stats',
    description: 'Get overall statistics about the CasperProver system.',
    apiCall: api.getStats,
  },
  {
    name: 'POST /kyc/check',
    method: 'POST',
    path: '/kyc/check',
    description: 'Check KYC status for a user.',
    exampleBody: JSON.stringify({ userId: 'user-abc' }, null, 2),
    apiCall: api.checkKycStatus,
  },
  {
    name: 'POST /kyc/grant',
    method: 'POST',
    path: '/kyc/grant',
    description: 'Grant KYC access to a user.',
    exampleBody: JSON.stringify({ userId: 'user-xyz', reason: 'Manual verification' }, null, 2),
    apiCall: api.grantKycAccess,
  },
  {
    name: 'GET /kyc/whitelist/{user}',
    method: 'GET',
    path: '/kyc/whitelist/{user}',
    description: 'View KYC whitelist for a specific user (or all if "all").',
    apiCall: (params) => api.getKycWhitelist(params.user),
    params: [{ name: 'user', type: 'string', optional: false, default: 'user-abc' }],
  },
  {
    name: 'POST /inference/register-model',
    method: 'POST',
    path: '/inference/register-model',
    description: 'Register a new AI model.',
    exampleBody: JSON.stringify({
      modelName: 'FraudDetectionV1',
      modelHash: '0xmodelhash123',
      verifierContract: '0xverifiercontract456',
      description: 'AI model for detecting financial fraud.',
    }, null, 2),
    apiCall: api.registerModel,
  },
  {
    name: 'GET /inference/model/{id}',
    method: 'GET',
    path: '/inference/model/{id}',
    description: 'Get details of a registered AI model.',
    apiCall: (params) => api.getModelById(params.id),
    params: [{ name: 'id', type: 'string', optional: false, default: 'example-model-id' }],
  },
  {
    name: 'POST /inference/prove',
    method: 'POST',
    path: '/inference/prove',
    description: 'Run inference and generate a proof for an AI model.',
    exampleBody: JSON.stringify({
      modelId: 'example-model-id',
      inputData: '{"transactionAmount": 1000, "userHistory": "high-risk"}',
      agentId: 'agent-456',
    }, null, 2),
    apiCall: api.inferenceProve,
  },
  {
    name: 'POST /inference/verify',
    method: 'POST',
    path: '/inference/verify',
    description: 'Verify an AI inference proof.',
    exampleBody: JSON.stringify({
      modelId: 'example-model-id',
      proofId: 'example-proof-id',
      inputData: '{"transactionAmount": 1000, "userHistory": "high-risk"}',
      outputHash: '0xoutputhash789',
    }, null, 2),
    apiCall: api.inferenceVerify,
  },
  {
    name: 'POST /aggregation/create-batch',
    method: 'POST',
    path: '/aggregation/create-batch',
    description: 'Create a new batch for proof aggregation.',
    exampleBody: JSON.stringify({
      batchName: 'MonthlyFraudProofs',
      description: 'Aggregation of all fraud detection proofs for the month.',
    }, null, 2),
    apiCall: api.createAggregationBatch,
  },
  {
    name: 'POST /aggregation/add-proof',
    method: 'POST',
    path: '/aggregation/add-proof',
    description: 'Add a proof to an existing aggregation batch.',
    exampleBody: JSON.stringify({
      batchId: 'example-batch-id',
      proofId: 'example-proof-id',
    }, null, 2),
    apiCall: api.addProofToAggregationBatch,
  },
  {
    name: 'POST /aggregation/finalize',
    method: 'POST',
    path: '/aggregation/finalize',
    description: 'Finalize an aggregation batch, generating a Merkle root and final proof.',
    exampleBody: JSON.stringify({
      batchId: 'example-batch-id',
    }, null, 2),
    apiCall: api.finalizeAggregationBatch,
  },
  {
    name: 'GET /aggregation/batch/{id}',
    method: 'GET',
    path: '/aggregation/batch/{id}',
    description: 'Get details of an aggregation batch.',
    apiCall: (params) => api.getAggregationBatchById(params.id),
    params: [{ name: 'id', type: 'string', optional: false, default: 'example-batch-id' }],
  },
  {
    name: 'POST /zk/verify-groth16',
    method: 'POST',
    path: '/zk/verify-groth16',
    description: 'Verify a Groth16 ZK proof.',
    exampleBody: JSON.stringify({
      proof: '{"pi_a": ["0x...", "0x..."], "pi_b": [["0x...", "0x..."], ["0x...", "0x..."]], "pi_c": ["0x...", "0x..."]}',
      publicSignals: ['0xsignal1', '0xsignal2'],
    }, null, 2),
    apiCall: api.verifyGroth16,
  },
  {
    name: 'POST /zk/batch-verify',
    method: 'POST',
    path: '/zk/batch-verify',
    description: 'Batch verify multiple ZK proofs.',
    exampleBody: JSON.stringify({
      proofIds: ['proof-id-1', 'proof-id-2'],
    }, null, 2),
    apiCall: api.batchVerifyZK,
  },
  {
    name: 'POST /zk/challenge',
    method: 'POST',
    path: '/zk/challenge',
    description: 'Create a new ZK challenge.',
    exampleBody: JSON.stringify({
      challengerId: 'challenger-1',
      proofId: 'example-proof-id',
      challengeData: '{"challengeType": "recompute", "params": {"nonce": 123}}',
    }, null, 2),
    apiCall: api.challengeZK,
  },
  {
    name: 'GET /zk/challenge/{id}',
    method: 'GET',
    path: '/zk/challenge/{id}',
    description: 'Get details of a ZK challenge.',
    apiCall: (params) => api.getZKChallengeById(params.id),
    params: [{ name: 'id', type: 'string', optional: false, default: 'example-challenge-id' }],
  },
  {
    name: 'POST /pq/sign-sphincs',
    method: 'POST',
    path: '/pq/sign-sphincs',
    description: 'Sign a message using SPHINCS+ post-quantum cryptography.',
    exampleBody: JSON.stringify({
      message: 'This is a test message for SPHINCS+ signing.',
      // privateKey: '...' // In a real app, this would be handled securely
    }, null, 2),
    apiCall: api.signSphincs,
  },
  {
    name: 'POST /pq/verify-sphincs',
    method: 'POST',
    path: '/pq/verify-sphincs',
    description: 'Verify a SPHINCS+ signature.',
    exampleBody: JSON.stringify({
      message: 'This is a test message for SPHINCS+ signing.',
      signature: '0x...',
      publicKey: '0x...',
    }, null, 2),
    apiCall: api.verifySphincs,
  },
  {
    name: 'POST /pq/hybrid-sign',
    method: 'POST',
    path: '/pq/hybrid-sign',
    description: 'Sign a message using hybrid (classical + PQ) cryptography.',
    exampleBody: JSON.stringify({
      message: 'This is a test message for hybrid signing.',
      // classicalPrivateKey: '...',
      // pqPrivateKey: '...',
    }, null, 2),
    apiCall: api.hybridSign,
  },
  {
    name: 'POST /pq/hybrid-verify',
    method: 'POST',
    path: '/pq/hybrid-verify',
    description: 'Verify a hybrid signature.',
    exampleBody: JSON.stringify({
      message: 'This is a test message for hybrid signing.',
      classicalSignature: '0x...',
      pqSignature: '0x...',
      classicalPublicKey: '0x...',
      pqPublicKey: '0x...',
    }, null, 2),
    apiCall: api.hybridVerify,
  },
];

const Playground: React.FC = () => {
  const [selectedEndpoint, setSelectedEndpoint] = useState<EndpointConfig | null>(null);
  const [requestBody, setRequestBody] = useState<string>('');
  const [pathParams, setPathParams] = useState<Record<string, string | number>>({});
  const [queryParams, setQueryParams] = useState<Record<string, string | number>>({});
  const [response, setResponse] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Update request body and params when endpoint changes
  useEffect(() => {
    if (selectedEndpoint) {
      setRequestBody(selectedEndpoint.exampleBody || '');
      const initialPathParams: Record<string, string | number> = {};
      const initialQueryParams: Record<string, string | number> = {};
      selectedEndpoint.params?.forEach(p => {
        if (selectedEndpoint.path.includes(`{${p.name}}`)) {
          initialPathParams[p.name] = p.default !== undefined ? p.default : (p.type === 'number' ? 0 : '');
        } else {
          initialQueryParams[p.name] = p.default !== undefined ? p.default : (p.type === 'number' ? 0 : '');
        }
      });
      setPathParams(initialPathParams);
      setQueryParams(initialQueryParams);
      setResponse(null);
      setError(null);
    }
  }, [selectedEndpoint]);

  const handleEndpointChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const endpointName = e.target.value;
    const endpoint = playgroundEndpoints.find((ep) => ep.name === endpointName);
    setSelectedEndpoint(endpoint || null);
  };

  const handleParamChange = (paramName: string, value: string, type: 'path' | 'query') => {
    if (type === 'path') {
      setPathParams((prev) => ({ ...prev, [paramName]: value }));
    } else {
      setQueryParams((prev) => ({ ...prev, [paramName]: value }));
    }
  };

  const handleRunRequest = useCallback(async () => {
    if (!selectedEndpoint) {
      setError('Please select an endpoint.');
      return;
    }

    setLoading(true);
    setResponse(null);
    setError(null);

    try {
      let finalPath = selectedEndpoint.path;
      const callParams: Record<string, any> = {};

      // Process path parameters
      for (const paramName in pathParams) {
        if (finalPath.includes(`{${paramName}}`)) {
          const value = pathParams[paramName];
          if (!value) {
            throw new Error(`Path parameter '${paramName}' is required.`);
          }
          finalPath = finalPath.replace(`{${paramName}}`, String(value));
          callParams[paramName] = value; // Add to callParams for API function if needed
        }
      }

      // Process query parameters
      const queryString = new URLSearchParams();
      for (const paramName in queryParams) {
        const value = queryParams[paramName];
        if (value !== '' && value !== null && value !== undefined) {
          queryString.append(paramName, String(value));
          callParams[paramName] = value; // Add to callParams for API function if needed
        }
      }
      if (queryString.toString()) {
        finalPath = `${finalPath}?${queryString.toString()}`;
      }

      let bodyData: any = undefined;
      if (selectedEndpoint.method === 'POST' && requestBody) {
        try {
          bodyData = JSON.parse(requestBody);
        } catch (e) {
          throw new Error('Invalid JSON in request body.');
        }
      }

      // Determine arguments for the API call function
      let apiArgs: any[] = [];
      if (selectedEndpoint.method === 'GET' && selectedEndpoint.params) {
        // For GETs with params, pass them directly
        apiArgs = [callParams];
      } else if (selectedEndpoint.method === 'GET' && selectedEndpoint.path.includes('{id}')) {
        // Special case for GET /{id}
        apiArgs = [{ id: pathParams.id }];
      } else if (selectedEndpoint.method === 'POST') {
        // For POSTs, pass the body data
        apiArgs = [bodyData];
        // If POST also has path params (like revokeProof), combine them
        if (Object.keys(pathParams).length > 0) {
          apiArgs = [{ ...bodyData, ...pathParams }];
        }
      }

      // Handle specific API function signatures
      let apiCallPromise: Promise<api.ApiResponse<any>>;
      if (selectedEndpoint.name === 'GET /proofs') {
        apiCallPromise = api.getProofs(
          callParams.agent as string | undefined,
          callParams.page as number | undefined,
          callParams.limit as number | undefined
        );
      } else if (selectedEndpoint.name === 'GET /proofs/{id}') {
        apiCallPromise = api.getProofById(callParams.id as string);
      } else if (selectedEndpoint.name === 'POST /proofs/{id}/revoke') {
        apiCallPromise = api.revokeProof(callParams.id as string);
      } else if (selectedEndpoint.name === 'GET /kyc/whitelist/{user}') {
        apiCallPromise = api.getKycWhitelist(callParams.user as string);
      } else if (selectedEndpoint.name === 'GET /inference/model/{id}') {
        apiCallPromise = api.getModelById(callParams.id as string);
      } else if (selectedEndpoint.name === 'GET /aggregation/batch/{id}') {
        apiCallPromise = api.getAggregationBatchById(callParams.id as string);
      } else if (selectedEndpoint.name === 'GET /zk/challenge/{id}') {
        apiCallPromise = api.getZKChallengeById(callParams.id as string);
      } else {
        // Default for other API calls (mostly POST with body, or GET without params)
        apiCallPromise = selectedEndpoint.apiCall(...apiArgs);
      }

      const res = await apiCallPromise;

      if (res.success) {
        setResponse(res.data);
        toast.success('API call successful!');
      } else {
        setError(res.error || 'API call failed with unknown error.');
        toast.error(res.error || 'API call failed.');
      }
    } catch (err: any) {
      setError(err.message || 'An unexpected error occurred.');
      toast.error(err.message || 'An unexpected error occurred.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [selectedEndpoint, requestBody, pathParams, queryParams]);

  const formattedResponse = useMemo(() => {
    if (response === null) return '';
    try {
      return JSON.stringify(response, null, 2);
    } catch {
      return String(response); // Fallback for non-JSON responses
    }
  }, [response]);

  return (
    <div className="p-4">
      <h2 className="text-2xl font-bold text-gray-100 mb-6">API Playground</h2>
      <p className="text-gray-400 mb-6">
        Experiment with CasperProver API endpoints. Select an endpoint, fill in parameters/body, and see the live response.
      </p>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Request Panel */}
        <div className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
          <h3 className="text-xl font-semibold text-gray-100 mb-4 flex items-center gap-2">
            <Code size={24} className="text-red-500" />
            Request
          </h3>

          <div className="mb-4">
            <label htmlFor="endpoint-select" className="block text-sm font-medium text-gray-300 mb-1">
              Select Endpoint
            </label>
            <select
              id="endpoint-select"
              onChange={handleEndpointChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
              value={selectedEndpoint?.name || ''}
            >
              <option value="">-- Select an API Endpoint --</option>
              {playgroundEndpoints.map((ep) => (
                <option key={ep.name} value={ep.name}>
                  {ep.name}
                </option>
              ))}
            </select>
          </div>

          {selectedEndpoint && (
            <>
              <p className="text-gray-500 text-sm mb-4">{selectedEndpoint.description}</p>

              {/* Path Parameters */}
              {Object.keys(pathParams).length > 0 && (
                <div className="mb-4">
                  <h4 className="text-md font-medium text-gray-300 mb-2">Path Parameters</h4>
                  {selectedEndpoint.params?.filter(p => selectedEndpoint.path.includes(`{${p.name}}`)).map(param => (
                    <div key={param.name} className="mb-2">
                      <label htmlFor={`path-param-${param.name}`} className="block text-xs font-medium text-gray-400 mb-1">
                        {param.name} {param.optional ? '(Optional)' : '(Required)'}
                      </label>
                      <input
                        type={param.type === 'number' ? 'number' : 'text'}
                        id={`path-param-${param.name}`}
                        value={pathParams[param.name]}
                        onChange={(e) => handleParamChange(param.name, e.target.value, 'path')}
                        className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 font-mono text-sm focus:ring-red-500 focus:border-red-500"
                        required={!param.optional}
                      />
                    </div>
                  ))}
                </div>
              )}

              {/* Query Parameters */}
              {(selectedEndpoint?.params?.filter(p => !selectedEndpoint.path.includes(`{${p.name}}`))?.length ?? 0) > 0 && (
                <div className="mb-4">
                  <h4 className="text-md font-medium text-gray-300 mb-2">Query Parameters</h4>
                  {selectedEndpoint.params?.filter(p => !selectedEndpoint.path.includes(`{${p.name}}`)).map(param => (
                    <div key={param.name} className="mb-2">
                      <label htmlFor={`query-param-${param.name}`} className="block text-xs font-medium text-gray-400 mb-1">
                        {param.name} {param.optional ? '(Optional)' : '(Required)'}
                      </label>
                      <input
                        type={param.type === 'number' ? 'number' : 'text'}
                        id={`query-param-${param.name}`}
                        value={queryParams[param.name]}
                        onChange={(e) => handleParamChange(param.name, e.target.value, 'query')}
                        className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 font-mono text-sm focus:ring-red-500 focus:border-red-500"
                        required={!param.optional}
                      />
                    </div>
                  ))}
                </div>
              )}

              {/* Request Body */}
              {selectedEndpoint.method === 'POST' && (
                <div className="mb-4">
                  <label htmlFor="request-body" className="block text-sm font-medium text-gray-300 mb-1">
                    Request Body (JSON)
                  </label>
                  <textarea
                    id="request-body"
                    rows={10}
                    value={requestBody}
                    onChange={(e) => setRequestBody(e.target.value)}
                    className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 font-mono text-sm focus:ring-red-500 focus:border-red-500"
                  ></textarea>
                </div>
              )}

              <button
                onClick={handleRunRequest}
                disabled={loading || !selectedEndpoint}
                className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? <Loader2 size={20} className="animate-spin" /> : <Play size={20} />}
                {loading ? 'Running...' : 'Run Request'}
              </button>
            </>
          )}
        </div>

        {/* Response Panel */}
        <div className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
          <h3 className="text-xl font-semibold text-gray-100 mb-4 flex items-center gap-2">
            <Code size={24} className="text-red-500" />
            Response
          </h3>

          {loading && (
            <div className="text-center p-8 text-blue-400">
              <Loader2 className="animate-spin mx-auto mb-4" size={32} />
              Fetching response...
            </div>
          )}

          {error && (
            <div className="p-4 bg-red-900/30 text-red-300 border border-red-700 rounded-md flex items-center gap-2">
              <AlertTriangle size={20} />
              Error: {error}
            </div>
          )}

          {!loading && !error && (
            <pre className="bg-[#0b0b10] p-3 rounded-md font-mono text-sm overflow-x-auto border border-[#222235] min-h-[300px]">
              {formattedResponse || 'No response yet. Run a request to see results.'}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
};

export default Playground;
