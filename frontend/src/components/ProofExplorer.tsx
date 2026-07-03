import React, { useState, useEffect, useMemo, useCallback } from 'react';

// Define types for Proof and related data
export interface Proof {
  id: string;
  type: 'inference' | 'batch' | 'groth16';
  status: 'pending' | 'verified' | 'failed';
  timestamp: string;
  onChainLink?: string;
  merkleRoot?: string;
  dataHash: string;
  modelId: string;
  verifierAddress?: string;
  proofData?: string; // Simplified, could be a complex object
  verificationDetails?: string;
}

// Mock API utility (replace with actual API calls)
const mockApi = {
  fetchProofs: async (filters: { type?: string; search?: string }): Promise<Proof[]> => {
    await new Promise(resolve => setTimeout(resolve, 500)); // Simulate network delay
    const allProofs: Proof[] = [
      {
        id: 'proof_001', type: 'inference', status: 'verified', timestamp: '2023-10-26T10:00:00Z',
        onChainLink: 'https://casper.network/tx/0xabc123...', merkleRoot: '0x123abc...',
        dataHash: '0xdatahash1', modelId: 'model_A', verifierAddress: '0xverifier1',
        verificationDetails: 'Successfully verified on-chain.'
      },
      {
        id: 'proof_002', type: 'batch', status: 'pending', timestamp: '2023-10-26T10:15:00Z',
        onChainLink: '', merkleRoot: '0x456def...', dataHash: '0xdatahash2', modelId: 'model_B',
        verificationDetails: 'Awaiting finalization.'
      },
      {
        id: 'proof_003', type: 'groth16', status: 'verified', timestamp: '2023-10-26T10:30:00Z',
        onChainLink: 'https://casper.network/tx/0xdef456...', merkleRoot: '0x789ghi...',
        dataHash: '0xdatahash3', modelId: 'model_C', verifierAddress: '0xverifier2',
        verificationDetails: 'Groth16 proof verified off-chain and submitted.'
      },
      {
        id: 'proof_004', type: 'inference', status: 'failed', timestamp: '2023-10-26T10:45:00Z',
        onChainLink: '', merkleRoot: '0xabc789...', dataHash: '0xdatahash4', modelId: 'model_A',
        verificationDetails: 'Verification failed due to invalid witness.'
      },
      {
        id: 'proof_005', type: 'inference', status: 'verified', timestamp: '2023-10-26T11:00:00Z',
        onChainLink: 'https://casper.network/tx/0xghi012...', merkleRoot: '0xdef321...',
        dataHash: '0xdatahash5', modelId: 'model_D', verifierAddress: '0xverifier1',
        verificationDetails: 'Successfully verified on-chain.'
      },
      {
        id: 'proof_006', type: 'batch', status: 'verified', timestamp: '2023-10-26T11:30:00Z',
        onChainLink: 'https://casper.network/tx/0xjkl345...', merkleRoot: '0x012mno...',
        dataHash: '0xdatahash6', modelId: 'model_E', verifierAddress: '0xverifier3',
        verificationDetails: 'Batch proof verified, all sub-proofs valid.'
      },
    ];

    let filtered = allProofs;
    if (filters.type && filters.type !== 'all') {
      filtered = filtered.filter(p => p.type === filters.type);
    }
    if (filters.search) {
      const searchTerm = filters.search.toLowerCase();
      filtered = filtered.filter(p =>
        p.id.toLowerCase().includes(searchTerm) ||
        p.modelId.toLowerCase().includes(searchTerm) ||
        p.merkleRoot?.toLowerCase().includes(searchTerm) ||
        p.dataHash.toLowerCase().includes(searchTerm)
      );
    }
    return filtered;
  },
  fetchProofDetails: async (id: string): Promise<Proof | undefined> => {
    await new Promise(resolve => setTimeout(resolve, 300));
    const allProofs = await mockApi.fetchProofs({}); // Re-use mock data
    return allProofs.find(p => p.id === id);
  }
};

const ProofExplorer: React.FC = () => {
  const [proofs, setProofs] = useState<Proof[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedProofId, setSelectedProofId] = useState<string | null>(null);
  const [selectedProofDetails, setSelectedProofDetails] = useState<Proof | null>(null);

  const [filterType, setFilterType] = useState<'all' | Proof['type']>('all');
  const [searchTerm, setSearchTerm] = useState<string>('');

  const fetchProofs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await mockApi.fetchProofs({ type: filterType === 'all' ? undefined : filterType, search: searchTerm });
      setProofs(data);
    } catch (err) {
      setError('Failed to fetch proofs.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [filterType, searchTerm]);

  useEffect(() => {
    fetchProofs();
  }, [fetchProofs]);

  const fetchProofDetails = useCallback(async (id: string) => {
    setSelectedProofDetails(null); // Clear previous details
    try {
      const details = await mockApi.fetchProofDetails(id);
      setSelectedProofDetails(details || null);
    } catch (err) {
      setError('Failed to fetch proof details.');
      console.error(err);
    }
  }, []);

  useEffect(() => {
    if (selectedProofId) {
      fetchProofDetails(selectedProofId);
    }
  }, [selectedProofId, fetchProofDetails]);

  const handleProofClick = (proofId: string) => {
    setSelectedProofId(proofId);
  };

  const getStatusColor = (status: Proof['status']) => {
    switch (status) {
      case 'verified': return 'text-green-600 bg-green-100';
      case 'pending': return 'text-yellow-600 bg-yellow-100';
      case 'failed': return 'text-red-600 bg-red-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const renderMerkleTreeVisual = (merkleRoot?: string) => {
    if (!merkleRoot) return <p className="text-gray-500">No Merkle Root available.</p>;

    // Simplified Merkle tree visualization: just showing root and a few example hashes
    const exampleHashes = [
      merkleRoot.slice(0, 10) + '...',
      '0xleaf123...',
      '0xleaf456...',
      '0xnode789...',
    ];

    return (
      <div className="p-4 bg-gray-50 rounded-md border border-gray-200">
        <h4 className="font-semibold text-gray-700 mb-2">Merkle Tree (Simplified)</h4>
        <div className="flex flex-col items-center text-sm font-mono">
          <div className="bg-blue-100 text-blue-800 px-2 py-1 rounded-md mb-1">Root: {merkleRoot.slice(0, 10)}...</div>
          <div className="h-4 w-px bg-gray-400"></div>
          <div className="flex space-x-4">
            <div className="flex flex-col items-center">
              <div className="bg-gray-200 text-gray-700 px-2 py-1 rounded-md mb-1">Node: {exampleHashes[3]}</div>
              <div className="h-4 w-px bg-gray-400"></div>
              <div className="flex space-x-2">
                <div className="bg-gray-200 text-gray-700 px-2 py-1 rounded-md">Leaf: {exampleHashes[1]}</div>
                <div className="bg-gray-200 text-gray-700 px-2 py-1 rounded-md">Leaf: {exampleHashes[2]}</div>
              </div>
            </div>
          </div>
        </div>
        <p className="text-xs text-gray-500 mt-2">This is a conceptual visualization. Actual tree structure may vary.</p>
      </div>
    );
  };


  return (
    <div className="p-6 bg-gray-50 min-h-screen">
      <h2 className="text-3xl font-bold text-gray-800 mb-6">Proof Explorer</h2>

      <div className="flex flex-col md:flex-row gap-4 mb-6">
        <input
          type="text"
          placeholder="Search by ID, Model, Merkle Root..."
          className="flex-grow p-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
        <select
          className="p-2 border border-gray-300 rounded-md bg-white focus:ring-blue-500 focus:border-blue-500"
          value={filterType}
          onChange={(e) => setFilterType(e.target.value as 'all' | Proof['type'])}
        >
          <option value="all">All Types</option>
          <option value="inference">Inference</option>
          <option value="batch">Batch</option>
          <option value="groth16">Groth16</option>
        </select>
        <button
          onClick={fetchProofs}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
        >
          Apply Filters
        </button>
      </div>

      {error && <div className="p-3 mb-4 bg-red-100 text-red-700 rounded-md">{error}</div>}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Proof List */}
        <div className="md:col-span-1 bg-white shadow-md rounded-lg p-4 overflow-y-auto max-h-[70vh]">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">Proofs ({proofs.length})</h3>
          {loading ? (
            <p className="text-gray-600">Loading proofs...</p>
          ) : proofs.length === 0 ? (
            <p className="text-gray-600">No proofs found matching criteria.</p>
          ) : (
            <ul className="space-y-3">
              {proofs.map((proof) => (
                <li
                  key={proof.id}
                  className={`p-3 border rounded-md cursor-pointer hover:bg-gray-50 transition-colors ${
                    selectedProofId === proof.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200'
                  }`}
                  onClick={() => handleProofClick(proof.id)}
                >
                  <div className="flex justify-between items-center">
                    <span className="font-medium text-gray-800">{proof.id}</span>
                    <span className={`px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(proof.status)}`}>
                      {proof.status}
                    </span>
                  </div>
                  <p className="text-sm text-gray-600">Type: {proof.type}</p>
                  <p className="text-xs text-gray-500">Model: {proof.modelId}</p>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Proof Detail View */}
        <div className="md:col-span-2 bg-white shadow-md rounded-lg p-6">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">Proof Details</h3>
          {selectedProofId === null ? (
            <p className="text-gray-600">Select a proof from the list to view its details.</p>
          ) : !selectedProofDetails ? (
            <p className="text-gray-600">Loading proof details...</p>
          ) : (
            <div className="space-y-4">
              <div>
                <p className="text-sm text-gray-500">Proof ID</p>
                <p className="font-mono text-gray-800 break-all">{selectedProofDetails.id}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Type</p>
                <p className="font-medium text-gray-800">{selectedProofDetails.type}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Status</p>
                <span className={`px-3 py-1 text-sm font-semibold rounded-full ${getStatusColor(selectedProofDetails.status)}`}>
                  {selectedProofDetails.status}
                </span>
              </div>
              <div>
                <p className="text-sm text-gray-500">Verification Details</p>
                <p className="text-gray-700">{selectedProofDetails.verificationDetails || 'N/A'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Timestamp</p>
                <p className="text-gray-700">{new Date(selectedProofDetails.timestamp).toLocaleString()}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Model ID</p>
                <p className="font-mono text-gray-800">{selectedProofDetails.modelId}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Data Hash</p>
                <p className="font-mono text-gray-800 break-all">{selectedProofDetails.dataHash}</p>
              </div>
              {selectedProofDetails.merkleRoot && (
                <div>
                  <p className="text-sm text-gray-500">Merkle Root</p>
                  <p className="font-mono text-gray-800 break-all">{selectedProofDetails.merkleRoot}</p>
                </div>
              )}
              {selectedProofDetails.onChainLink && (
                <div>
                  <p className="text-sm text-gray-500">On-chain Link</p>
                  <a
                    href={selectedProofDetails.onChainLink}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:underline break-all"
                  >
                    {selectedProofDetails.onChainLink}
                  </a>
                </div>
              )}
              {selectedProofDetails.verifierAddress && (
                <div>
                  <p className="text-sm text-gray-500">Verifier Address</p>
                  <p className="font-mono text-gray-800 break-all">{selectedProofDetails.verifierAddress}</p>
                </div>
              )}
              {selectedProofDetails.merkleRoot && renderMerkleTreeVisual(selectedProofDetails.merkleRoot)}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ProofExplorer;
