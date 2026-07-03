import React, { useState, useEffect, useCallback } from 'react';

// Define types for PQ Crypto Status
export interface PQCryptoStatus {
  isPqEnabled: boolean;
  availableAlgorithms: {
    sphincsPlus: {
      isEnabled: boolean;
      securityLevels: number[]; // NIST security levels (1-5)
      details: string;
    };
    mlDsa: {
      isEnabled: boolean;
      securityLevels: number[]; // NIST security levels (1-5)
      details: string;
    };
  };
  hybridSignatureStatus: {
    isEnabled: boolean;
    currentScheme: string; // e.g., "ED25519 + SPHINCS+"
    fallbackScheme: string; // e.g., "ED25519"
    details: string;
  };
  overallSecurityLevel: number; // Aggregate NIST security level
  lastUpdated: string;
}

// Mock API utility (replace with actual API calls)
const mockApi = {
  fetchPQCryptoStatus: async (): Promise<PQCryptoStatus> => {
    await new Promise(resolve => setTimeout(resolve, 700)); // Simulate network delay
    return {
      isPqEnabled: true,
      availableAlgorithms: {
        sphincsPlus: {
          isEnabled: true,
          securityLevels: [3, 5],
          details: 'SPHINCS+ (SHA2-256, 128-bit security, 256-bit security) available for stateless signatures.'
        },
        mlDsa: {
          isEnabled: true,
          securityLevels: [2, 3, 5],
          details: 'ML-DSA (Dilithium) available for digital signatures, offering various security levels.'
        }
      },
      hybridSignatureStatus: {
        isEnabled: true,
        currentScheme: 'ED25519 + SPHINCS+-SHA2-128s',
        fallbackScheme: 'ED25519',
        details: 'Hybrid signatures are active, combining classical ED25519 with SPHINCS+ for forward secrecy and post-quantum resistance.'
      },
      overallSecurityLevel: 3, // Assuming the lowest common denominator or configured level
      lastUpdated: new Date().toISOString(),
    };
  }
};

const PQCryptoStatus: React.FC = () => {
  const [status, setStatus] = useState<PQCryptoStatus | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await mockApi.fetchPQCryptoStatus();
      setStatus(data);
    } catch (err) {
      setError('Failed to fetch PQ Crypto status.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  const getSecurityLevelBadge = (level: number) => {
    let colorClass = '';
    switch (level) {
      case 1: colorClass = 'bg-blue-100 text-blue-800'; break;
      case 2: colorClass = 'bg-green-100 text-green-800'; break;
      case 3: colorClass = 'bg-yellow-100 text-yellow-800'; break;
      case 4: colorClass = 'bg-orange-100 text-orange-800'; break;
      case 5: colorClass = 'bg-red-100 text-red-800'; break;
      default: colorClass = 'bg-gray-100 text-gray-800'; break;
    }
    return (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colorClass}`}>
        NIST Level {level}
      </span>
    );
  };

  const getEnabledStatusBadge = (isEnabled: boolean) => {
    return (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${isEnabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
        {isEnabled ? 'Enabled' : 'Disabled'}
      </span>
    );
  };

  if (loading) {
    return (
      <div className="p-6 bg-gray-50 min-h-screen flex items-center justify-center">
        <p className="text-lg text-gray-600">Loading Post-Quantum Crypto Status...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6 bg-gray-50 min-h-screen flex items-center justify-center">
        <div className="p-4 bg-red-100 text-red-700 rounded-md">Error: {error}</div>
      </div>
    );
  }

  if (!status) {
    return (
      <div className="p-6 bg-gray-50 min-h-screen flex items-center justify-center">
        <p className="text-lg text-gray-600">No status data available.</p>
      </div>
    );
  }

  return (
    <div className="p-6 bg-gray-50 min-h-screen">
      <h2 className="text-3xl font-bold text-gray-800 mb-6">Post-Quantum Readiness Lab</h2>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Overall Status */}
        <div className="bg-white shadow-md rounded-lg p-6 border border-gray-200">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">Overall PQ Readiness</h3>
          <div className="space-y-3">
            <p className="text-gray-700 text-lg">
              Post-Quantum Cryptography: {getEnabledStatusBadge(status.isPqEnabled)}
            </p>
            <p className="text-gray-700 text-lg">
              Overall Security Level: {getSecurityLevelBadge(status.overallSecurityLevel)}
            </p>
            <p className="text-sm text-gray-500">Last Updated: {new Date(status.lastUpdated).toLocaleString()}</p>
          </div>
        </div>

        {/* Available Algorithms */}
        <div className="bg-white shadow-md rounded-lg p-6 border border-gray-200">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">Available PQ Algorithms</h3>
          <div className="space-y-4">
            {/* SPHINCS+ */}
            <div>
              <h4 className="font-bold text-lg text-gray-800 flex items-center gap-2">
                SPHINCS+ {getEnabledStatusBadge(status.availableAlgorithms.sphincsPlus.isEnabled)}
              </h4>
              <p className="text-gray-600 text-sm mt-1">{status.availableAlgorithms.sphincsPlus.details}</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {status.availableAlgorithms.sphincsPlus.securityLevels.map(level => (
                  <React.Fragment key={`sphincs-${level}`}>
                    {getSecurityLevelBadge(level)}
                  </React.Fragment>
                ))}
              </div>
            </div>
            {/* ML-DSA (Dilithium) */}
            <div>
              <h4 className="font-bold text-lg text-gray-800 flex items-center gap-2">
                ML-DSA (Dilithium) {getEnabledStatusBadge(status.availableAlgorithms.mlDsa.isEnabled)}
              </h4>
              <p className="text-gray-600 text-sm mt-1">{status.availableAlgorithms.mlDsa.details}</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {status.availableAlgorithms.mlDsa.securityLevels.map(level => (
                  <React.Fragment key={`mldsa-${level}`}>
                    {getSecurityLevelBadge(level)}
                  </React.Fragment>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Hybrid Signature Status */}
        <div className="lg:col-span-2 bg-white shadow-md rounded-lg p-6 border border-gray-200">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">Hybrid Signature Status</h3>
          <div className="space-y-3">
            <p className="text-gray-700 text-lg">
              Hybrid Signatures: {getEnabledStatusBadge(status.hybridSignatureStatus.isEnabled)}
            </p>
            <p className="text-gray-700">
              <strong>Current Scheme:</strong> <span className="font-mono bg-gray-100 px-2 py-1 rounded text-sm">{status.hybridSignatureStatus.currentScheme}</span>
            </p>
            <p className="text-gray-700">
              <strong>Fallback Scheme:</strong> <span className="font-mono bg-gray-100 px-2 py-1 rounded text-sm">{status.hybridSignatureStatus.fallbackScheme}</span>
            </p>
            <p className="text-gray-600 text-sm">{status.hybridSignatureStatus.details}</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PQCryptoStatus;
