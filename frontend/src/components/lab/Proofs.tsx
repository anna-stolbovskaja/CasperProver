import React, { useEffect, useState, useCallback, useMemo } from 'react';
import {
  PlusCircle,
  CheckCircle,
  XCircle,
  FileText,
  Download,
  Eye,
  Loader2,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Search,
} from 'lucide-react';
import {
  getProofs,
  createProof,
  verifyProof,
  revokeProof,
  exportProof,
  Proof,
  CreateProofRequest,
  VerifyProofRequest,
} from '../../lib/api';
import { toast } from '../ui/toast'; // Assuming react-toastify for notifications

// Placeholder for a generic Modal component
const Modal: React.FC<{
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  className?: string;
}> = ({ isOpen, onClose, title, children, className }) => {
  if (!isOpen) return null;
  return (
    <div className="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50 p-4">
      <div className={`bg-[#13131d] border border-[#222235] rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto ${className}`}>
        <div className="flex justify-between items-center p-4 border-b border-[#222235]">
          <h2 className="text-xl font-semibold text-gray-100">{title}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-100">
            <XCircle size={24} />
          </button>
        </div>
        <div className="p-4 text-gray-200">
          {children}
        </div>
      </div>
    </div>
  );
};

const Proofs: React.FC = () => {
  const [proofs, setProofs] = useState<Proof[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [totalProofs, setTotalProofs] = useState(0);
  const [agentFilter, setAgentFilter] = useState('');

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newProofData, setNewProofData] = useState<CreateProofRequest>({
    agentId: '',
    inputHash: '',
    outputHash: '',
    proofData: '',
  });
  const [isCreating, setIsCreating] = useState(false);

  const [selectedProof, setSelectedProof] = useState<Proof | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [isVerifying, setIsVerifying] = useState<string | null>(null);
  const [isRevoking, setIsRevoking] = useState<string | null>(null);
  const [isExporting, setIsExporting] = useState<string | null>(null);

  const fetchProofs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getProofs(agentFilter || undefined, page, limit);
      if (res.success && res.data) {
        setProofs(res.data.proofs);
        setTotalProofs(res.data.total);
      } else {
        setError(res.error || 'Failed to fetch proofs');
        toast.error(res.error || 'Failed to fetch proofs');
      }
    } catch (err) {
      setError('An unexpected error occurred.');
      toast.error('An unexpected error occurred.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [page, limit, agentFilter]);

  useEffect(() => {
    fetchProofs();
  }, [fetchProofs]);

  const handleCreateProofChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setNewProofData((prev) => ({ ...prev, [name]: value }));
  };

  const handleCreateProofSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCreating(true);
    try {
      const res = await createProof(newProofData);
      if (res.success) {
        toast.success('Proof created successfully!');
        setIsCreateModalOpen(false);
        setNewProofData({ agentId: '', inputHash: '', outputHash: '', proofData: '' });
        fetchProofs();
      } else {
        toast.error(res.error || 'Failed to create proof');
      }
    } catch (err) {
      toast.error('An unexpected error occurred.');
      console.error(err);
    } finally {
      setIsCreating(false);
    }
  };

  const handleVerifyProof = async (proofId: string) => {
    setIsVerifying(proofId);
    try {
      const res = await verifyProof({ proofId });
      if (res.success) {
        toast.success(`Proof ${proofId} verification: ${res.data?.isValid ? 'Valid' : 'Invalid'}!`);
      } else {
        toast.error(res.error || 'Failed to verify proof');
      }
    } catch (err) {
      toast.error('An unexpected error occurred during verification.');
      console.error(err);
    } finally {
      setIsVerifying(null);
    }
  };

  const handleRevokeProof = async (proofId: string) => {
    if (!window.confirm(`Are you sure you want to revoke proof ${proofId}? This action cannot be undone.`)) {
      return;
    }
    setIsRevoking(proofId);
    try {
      const res = await revokeProof(proofId);
      if (res.success) {
        toast.success(`Proof ${proofId} revoked successfully!`);
        fetchProofs();
      } else {
        toast.error(res.error || 'Failed to revoke proof');
      }
    } catch (err) {
      toast.error('An unexpected error occurred during revocation.');
      console.error(err);
    } finally {
      setIsRevoking(null);
    }
  };

  const handleExportProof = async (proofId: string) => {
    setIsExporting(proofId);
    try {
      const res = await exportProof(proofId);
      if (res.success && res.data) {
        const blob = new Blob([res.data], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `proof-${proofId}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        toast.success(`Proof ${proofId} exported successfully!`);
      } else {
        toast.error(res.error || 'Failed to export proof');
      }
    } catch (err) {
      toast.error('An unexpected error occurred during export.');
      console.error(err);
    } finally {
      setIsExporting(null);
    }
  };

  const openDetailModal = (proof: Proof) => {
    setSelectedProof(proof);
    setIsDetailModalOpen(true);
  };

  const closeDetailModal = () => {
    setSelectedProof(null);
    setIsDetailModalOpen(false);
  };

  const totalPages = useMemo(() => Math.ceil(totalProofs / limit), [totalProofs, limit]);

  if (loading) {
    return (
      <div className="text-center p-8 text-gray-400">
        <Loader2 className="animate-spin mx-auto mb-4" size={32} />
        Loading proofs...
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center p-8 text-red-400">
        <AlertTriangle className="mx-auto mb-4" size={32} />
        Error: {error}
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-100">Proof Lab</h2>
        <div className="flex items-center space-x-4">
          <div className="relative">
            <input
              type="text"
              placeholder="Filter by Agent ID"
              value={agentFilter}
              onChange={(e) => setAgentFilter(e.target.value)}
              className="pl-10 pr-4 py-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 placeholder-gray-500 focus:ring-red-500 focus:border-red-500"
            />
            <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          </div>
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200"
          >
            <PlusCircle size={20} />
            Create New Proof
          </button>
        </div>
      </div>

      {proofs.length === 0 && !agentFilter ? (
        <div className="text-center p-8 text-gray-400 bg-[#1a1a2a] rounded-lg border border-[#222235]">
          <FileText className="mx-auto mb-4" size={48} />
          <p className="text-xl font-semibold">No proofs found.</p>
          <p className="mt-2">Start by creating a new proof.</p>
        </div>
      ) : proofs.length === 0 && agentFilter ? (
        <div className="text-center p-8 text-gray-400 bg-[#1a1a2a] rounded-lg border border-[#222235]">
          <FileText className="mx-auto mb-4" size={48} />
          <p className="text-xl font-semibold">No proofs found for agent "{agentFilter}".</p>
          <p className="mt-2">Try a different agent ID or clear the filter.</p>
        </div>
      ) : (
        <div className="overflow-x-auto bg-[#1a1a2a] rounded-lg border border-[#222235]">
          <table className="min-w-full divide-y divide-[#222235]">
            <thead className="bg-[#13131d]">
              <tr>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Proof ID
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Agent ID
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Input Hash
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Output Hash
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Status
                </th>
                <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#222235]">
              {proofs.map((proof) => (
                <tr key={proof.id} className="hover:bg-[#1a1a2a]/50 transition-colors duration-150">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                    <span className="cursor-pointer hover:text-red-400" onClick={() => openDetailModal(proof)}>
                      {proof.id.substring(0, 8)}...{proof.id.substring(proof.id.length - 8)}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                    {proof.agentId.substring(0, 8)}...
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                    {proof.inputHash.substring(0, 8)}...
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                    {proof.outputHash.substring(0, 8)}...
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        proof.status === 'valid'
                          ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
                          : proof.status === 'revoked'
                          ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300'
                          : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                      }`}
                    >
                      {proof.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex justify-end space-x-2">
                      <button
                        onClick={() => openDetailModal(proof)}
                        className="text-blue-400 hover:text-blue-300 p-1 rounded-full hover:bg-[#222235]"
                        title="View Details"
                      >
                        <Eye size={18} />
                      </button>
                      <button
                        onClick={() => handleVerifyProof(proof.id)}
                        disabled={isVerifying === proof.id}
                        className="text-green-400 hover:text-green-300 p-1 rounded-full hover:bg-[#222235] disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Verify Proof"
                      >
                        {isVerifying === proof.id ? <Loader2 size={18} className="animate-spin" /> : <CheckCircle size={18} />}
                      </button>
                      <button
                        onClick={() => handleRevokeProof(proof.id)}
                        disabled={proof.status === 'revoked' || isRevoking === proof.id}
                        className="text-yellow-400 hover:text-yellow-300 p-1 rounded-full hover:bg-[#222235] disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Revoke Proof"
                      >
                        {isRevoking === proof.id ? <Loader2 size={18} className="animate-spin" /> : <XCircle size={18} />}
                      </button>
                      <button
                        onClick={() => handleExportProof(proof.id)}
                        disabled={isExporting === proof.id}
                        className="text-purple-400 hover:text-purple-300 p-1 rounded-full hover:bg-[#222235] disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Export Proof"
                      >
                        {isExporting === proof.id ? <Loader2 size={18} className="animate-spin" /> : <Download size={18} />}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center items-center mt-6 space-x-4">
          <button
            onClick={() => setPage((prev) => Math.max(1, prev - 1))}
            disabled={page === 1}
            className="p-2 rounded-md bg-[#1a1a2a] border border-[#222235] text-gray-300 hover:bg-[#222235] disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft size={20} />
          </button>
          <span className="text-gray-300">
            Page {page} of {totalPages}
          </span>
          <button
            onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
            disabled={page === totalPages}
            className="p-2 rounded-md bg-[#1a1a2a] border border-[#222235] text-gray-300 hover:bg-[#222235] disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight size={20} />
          </button>
        </div>
      )}

      {/* Create Proof Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Proof"
      >
        <form onSubmit={handleCreateProofSubmit} className="space-y-4">
          <div>
            <label htmlFor="agentId" className="block text-sm font-medium text-gray-300 mb-1">
              Agent ID
            </label>
            <input
              type="text"
              id="agentId"
              name="agentId"
              value={newProofData.agentId}
              onChange={handleCreateProofChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
              required
            />
          </div>
          <div>
            <label htmlFor="inputHash" className="block text-sm font-medium text-gray-300 mb-1">
              Input Hash
            </label>
            <input
              type="text"
              id="inputHash"
              name="inputHash"
              value={newProofData.inputHash}
              onChange={handleCreateProofChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
              required
            />
          </div>
          <div>
            <label htmlFor="outputHash" className="block text-sm font-medium text-gray-300 mb-1">
              Output Hash
            </label>
            <input
              type="text"
              id="outputHash"
              name="outputHash"
              value={newProofData.outputHash}
              onChange={handleCreateProofChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
              required
            />
          </div>
          <div>
            <label htmlFor="proofData" className="block text-sm font-medium text-gray-300 mb-1">
              Proof Data (JSON/String)
            </label>
            <textarea
              id="proofData"
              name="proofData"
              rows={5}
              value={newProofData.proofData}
              onChange={handleCreateProofChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 font-mono focus:ring-red-500 focus:border-red-500"
              required
            ></textarea>
          </div>
          <button
            type="submit"
            disabled={isCreating}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isCreating ? <Loader2 size={20} className="animate-spin" /> : <PlusCircle size={20} />}
            {isCreating ? 'Creating...' : 'Create Proof'}
          </button>
        </form>
      </Modal>

      {/* Proof Detail Modal */}
      <Modal
        isOpen={isDetailModalOpen}
        onClose={closeDetailModal}
        title="Proof Details"
        className="max-w-3xl"
      >
        {selectedProof && (
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-medium text-gray-300">Proof ID:</h3>
              <p className="font-mono text-red-400 break-all">{selectedProof.id}</p>
            </div>
            <div>
              <h3 className="text-lg font-medium text-gray-300">Agent ID:</h3>
              <p className="font-mono break-all">{selectedProof.agentId}</p>
            </div>
            <div>
              <h3 className="text-lg font-medium text-gray-300">Input Hash:</h3>
              <p className="font-mono break-all">{selectedProof.inputHash}</p>
            </div>
            <div>
              <h3 className="text-lg font-medium text-gray-300">Output Hash:</h3>
              <p className="font-mono break-all">{selectedProof.outputHash}</p>
            </div>
            <div>
              <h3 className="text-lg font-medium text-gray-300">Status:</h3>
              <p className="font-mono break-all">{selectedProof.status}</p>
            </div>
            <div>
              <h3 className="text-lg font-medium text-gray-300">Created At:</h3>
              <p className="font-mono break-all">{new Date(selectedProof.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <h3 className="text-lg font-medium text-gray-300">Proof Data:</h3>
              <pre className="bg-[#0b0b10] p-3 rounded-md font-mono text-sm overflow-x-auto border border-[#222235]">
                {JSON.stringify(JSON.parse(selectedProof.proofData), null, 2)}
              </pre>
            </div>
            {selectedProof.merklePath && selectedProof.merklePath.length > 0 && (
              <div>
                <h3 className="text-lg font-medium text-gray-300">Merkle Path:</h3>
                <div className="bg-[#0b0b10] p-3 rounded-md font-mono text-sm overflow-x-auto border border-[#222235]">
                  <p className="text-gray-400 mb-2">Visualizing Merkle path (simplified):</p>
                  <div className="space-y-2">
                    {selectedProof.merklePath.map((node, index) => (
                      <div key={index} className="flex items-center">
                        <span className="text-red-500 mr-2">{index === 0 ? 'Leaf:' : `Node ${index}:`}</span>
                        <span className="break-all">{node}</span>
                      </div>
                    ))}
                    <div className="flex items-center">
                      <span className="text-red-500 mr-2">Root:</span>
                      <span className="break-all">... (derived from path)</span>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Proofs;
