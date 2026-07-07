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
  ExternalLink,
} from 'lucide-react';
import {
  getProofs,
  verifyProof,
  revokeProof,
  exportProof,
  Proof,
} from '../../lib/api';
import { toast } from '../ui/toast';

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

function proofStatus(p: Proof): string {
  if (p.revoked) return 'revoked';
  if (p.valid) return 'valid';
  return 'invalid';
}

function truncHash(h: string, len = 10): string {
  if (!h || h.length <= len * 2) return h || '—';
  return h.slice(0, len) + '...' + h.slice(-6);
}

const Proofs: React.FC = () => {
  const [proofs, setProofs] = useState<Proof[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [totalProofs, setTotalProofs] = useState(0);
  const [agentFilter, setAgentFilter] = useState('');

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
        setProofs(res.data.proofs || []);
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

  const handleVerifyProof = async (proofId: string) => {
    setIsVerifying(proofId);
    try {
      const res = await verifyProof({ proofId });
      if (res.success) {
        toast.success(`Proof ${proofId} verified.`);
      } else {
        toast.error(res.error || 'Verification failed');
      }
    } catch (err) {
      toast.error('Verification error.');
      console.error(err);
    } finally {
      setIsVerifying(null);
    }
  };

  const handleRevokeProof = async (proofId: string) => {
    if (!window.confirm(`Revoke proof ${proofId}? This cannot be undone.`)) return;
    setIsRevoking(proofId);
    try {
      const res = await revokeProof(proofId);
      if (res.success) {
        toast.success(`Proof ${proofId} revoked.`);
        fetchProofs();
      } else {
        toast.error(res.error || 'Revocation failed');
      }
    } catch (err) {
      toast.error('Revocation error.');
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
        const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `proof-${proofId}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        toast.success(`Proof ${proofId} exported.`);
      } else {
        toast.error(res.error || 'Export failed');
      }
    } catch (err) {
      toast.error('Export error.');
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
        <h2 className="text-2xl font-bold text-gray-100">Proof Registry</h2>
        <div className="flex items-center space-x-4">
          <div className="relative">
            <input
              type="text"
              placeholder="Filter by Agent"
              value={agentFilter}
              onChange={(e) => { setAgentFilter(e.target.value); setPage(1); }}
              className="pl-10 pr-4 py-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 placeholder-gray-500 focus:ring-red-500 focus:border-red-500"
            />
            <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          </div>
          <span className="text-sm text-gray-400">{totalProofs} proofs total</span>
        </div>
      </div>

      {proofs.length === 0 ? (
        <div className="text-center p-8 text-gray-400 bg-[#1a1a2a] rounded-lg border border-[#222235]">
          <FileText className="mx-auto mb-4" size={48} />
          <p className="text-xl font-semibold">No proofs found{agentFilter ? ` for "${agentFilter}"` : ''}.</p>
        </div>
      ) : (
        <div className="overflow-x-auto bg-[#1a1a2a] rounded-lg border border-[#222235]">
          <table className="min-w-full divide-y divide-[#222235]">
            <thead className="bg-[#13131d]">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">ID</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Agent</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Use Case</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Merkle Root</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Mode</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Status</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-gray-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#222235]">
              {proofs.map((proof) => {
                const status = proofStatus(proof);
                return (
                  <tr key={proof.id} className="hover:bg-[#222235]/50 transition-colors duration-150">
                    <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-red-400 cursor-pointer hover:text-red-300" onClick={() => openDetailModal(proof)}>
                      {proof.id}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-300">{proof.agent}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-400">{proof.use_case}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-gray-400" title={proof.merkle_root}>
                      {truncHash(proof.merkle_root)}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-400 capitalize">{proof.mode}</td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm">
                      <span className={`px-2 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        status === 'valid'
                          ? 'bg-green-900/60 text-green-300'
                          : status === 'revoked'
                          ? 'bg-yellow-900/60 text-yellow-300'
                          : 'bg-gray-700 text-gray-300'
                      }`}>
                        {status}
                      </span>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex justify-end space-x-1">
                        <button onClick={() => openDetailModal(proof)} className="text-blue-400 hover:text-blue-300 p-1 rounded hover:bg-[#222235]" title="Details">
                          <Eye size={16} />
                        </button>
                        <button onClick={() => handleVerifyProof(proof.id)} disabled={isVerifying === proof.id} className="text-green-400 hover:text-green-300 p-1 rounded hover:bg-[#222235] disabled:opacity-50" title="Verify">
                          {isVerifying === proof.id ? <Loader2 size={16} className="animate-spin" /> : <CheckCircle size={16} />}
                        </button>
                        <button onClick={() => handleRevokeProof(proof.id)} disabled={status === 'revoked' || isRevoking === proof.id} className="text-yellow-400 hover:text-yellow-300 p-1 rounded hover:bg-[#222235] disabled:opacity-50" title="Revoke">
                          {isRevoking === proof.id ? <Loader2 size={16} className="animate-spin" /> : <XCircle size={16} />}
                        </button>
                        <button onClick={() => handleExportProof(proof.id)} disabled={isExporting === proof.id} className="text-purple-400 hover:text-purple-300 p-1 rounded hover:bg-[#222235] disabled:opacity-50" title="Export">
                          {isExporting === proof.id ? <Loader2 size={16} className="animate-spin" /> : <Download size={16} />}
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex justify-center items-center mt-6 space-x-4">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className="p-2 rounded-md bg-[#1a1a2a] border border-[#222235] text-gray-300 hover:bg-[#222235] disabled:opacity-50">
            <ChevronLeft size={20} />
          </button>
          <span className="text-gray-300">Page {page} of {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="p-2 rounded-md bg-[#1a1a2a] border border-[#222235] text-gray-300 hover:bg-[#222235] disabled:opacity-50">
            <ChevronRight size={20} />
          </button>
        </div>
      )}

      {/* Proof Detail Modal */}
      <Modal isOpen={isDetailModalOpen} onClose={closeDetailModal} title="Proof Details" className="max-w-3xl">
        {selectedProof && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <h3 className="text-sm font-medium text-gray-400">Proof ID</h3>
                <p className="font-mono text-red-400">{selectedProof.id}</p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Agent</h3>
                <p className="font-mono">{selectedProof.agent}</p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Use Case</h3>
                <p>{selectedProof.use_case}</p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Mode</h3>
                <p className="capitalize">{selectedProof.mode}</p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Status</h3>
                <p className={selectedProof.revoked ? 'text-yellow-400' : selectedProof.valid ? 'text-green-400' : 'text-gray-400'}>
                  {proofStatus(selectedProof)}
                </p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Generated</h3>
                <p>{new Date(selectedProof.timestamp * 1000).toLocaleString()}</p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Generation Time</h3>
                <p>{selectedProof.generation_ms} ms</p>
              </div>
              <div>
                <h3 className="text-sm font-medium text-gray-400">Leaf Index</h3>
                <p>{selectedProof.leaf_index}</p>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-400 mb-1">Input Hash</h3>
              <p className="font-mono text-sm break-all bg-[#0b0b10] p-2 rounded border border-[#222235]">{selectedProof.input_hash}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-400 mb-1">Output Hash</h3>
              <p className="font-mono text-sm break-all bg-[#0b0b10] p-2 rounded border border-[#222235]">{selectedProof.output_hash}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-400 mb-1">Model Hash</h3>
              <p className="font-mono text-sm break-all bg-[#0b0b10] p-2 rounded border border-[#222235]">{selectedProof.model_hash}</p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-400 mb-1">Merkle Root</h3>
              <p className="font-mono text-sm break-all bg-[#0b0b10] p-2 rounded border border-[#222235]">{selectedProof.merkle_root}</p>
            </div>

            {selectedProof.deploy_hash && (
              <div>
                <h3 className="text-sm font-medium text-gray-400 mb-1">Deploy Hash (on-chain)</h3>
                <a
                  href={`https://testnet.cspr.live/deploy/${selectedProof.deploy_hash}`}
                  target="_blank" rel="noopener noreferrer"
                  className="font-mono text-sm text-red-400 hover:text-red-300 break-all flex items-center gap-1"
                >
                  {selectedProof.deploy_hash} <ExternalLink size={14} />
                </a>
              </div>
            )}

            {selectedProof.merkle_path && selectedProof.merkle_path.length > 0 && (
              <div>
                <h3 className="text-sm font-medium text-gray-400 mb-1">Merkle Path</h3>
                <div className="bg-[#0b0b10] p-3 rounded font-mono text-sm border border-[#222235] space-y-1">
                  {selectedProof.merkle_path.map((node, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <span className="text-red-500 w-16 text-right text-xs">Level {idx}:</span>
                      <span className="break-all text-gray-300">{node}</span>
                    </div>
                  ))}
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
