import React, { useState, useEffect, useCallback } from 'react';

// Define types for Model and related data
export interface Model {
  id: string;
  name: string;
  description: string;
  hash: string; // Unique identifier for the model's code/weights
  version: string;
  proofCount: number; // Number of proofs generated using this model
  registeredAt: string;
  ownerAddress: string;
  inferenceStats?: {
    totalProofs: number;
    last24hProofs: number;
    avgVerificationTimeMs: number;
  };
}

// Mock API utility (replace with actual API calls)
const mockApi = {
  fetchModels: async (): Promise<Model[]> => {
    await new Promise(resolve => setTimeout(resolve, 600)); // Simulate network delay
    return [
      {
        id: 'model_A', name: 'ImageNet Classifier v1', description: 'ResNet-50 for image classification.',
        hash: '0xabc123def456...', version: '1.0.0', proofCount: 1250, registeredAt: '2023-01-15T09:00:00Z',
        ownerAddress: '0xownerA', inferenceStats: { totalProofs: 1250, last24hProofs: 50, avgVerificationTimeMs: 1200 }
      },
      {
        id: 'model_B', name: 'Fraud Detector v2', description: 'XGBoost model for transaction fraud detection.',
        hash: '0xdef456abc123...', version: '2.1.0', proofCount: 890, registeredAt: '2023-03-20T14:30:00Z',
        ownerAddress: '0xownerB', inferenceStats: { totalProofs: 890, last24hProofs: 30, avgVerificationTimeMs: 950 }
      },
      {
        id: 'model_C', name: 'Medical Diagnosis AI', description: 'CNN for early disease detection from scans.',
        hash: '0xghi789jkl012...', version: '1.1.2', proofCount: 340, registeredAt: '2023-06-01T11:00:00Z',
        ownerAddress: '0xownerC', inferenceStats: { totalProofs: 340, last24hProofs: 10, avgVerificationTimeMs: 2500 }
      },
      {
        id: 'model_D', name: 'Sentiment Analyzer', description: 'BERT-based model for text sentiment analysis.',
        hash: '0xmnb345qwe678...', version: '1.0.0', proofCount: 2100, registeredAt: '2023-02-10T16:00:00Z',
        ownerAddress: '0xownerA', inferenceStats: { totalProofs: 2100, last24hProofs: 80, avgVerificationTimeMs: 800 }
      },
    ];
  },
  registerModel: async (modelData: Omit<Model, 'id' | 'proofCount' | 'registeredAt' | 'inferenceStats'>): Promise<Model> => {
    await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate network delay
    const newModel: Model = {
      ...modelData,
      id: `model_${Math.random().toString(36).substr(2, 9)}`,
      proofCount: 0,
      registeredAt: new Date().toISOString(),
      inferenceStats: { totalProofs: 0, last24hProofs: 0, avgVerificationTimeMs: 0 }
    };
    console.log('Registered new model:', newModel);
    return newModel;
  }
};

const ModelMarketplace: React.FC = () => {
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isRegistering, setIsRegistering] = useState<boolean>(false);
  const [registrationError, setRegistrationError] = useState<string | null>(null);
  const [registrationSuccess, setRegistrationSuccess] = useState<boolean>(false);

  const [newModel, setNewModel] = useState<Omit<Model, 'id' | 'proofCount' | 'registeredAt' | 'inferenceStats'>>({
    name: '',
    description: '',
    hash: '',
    version: '',
    ownerAddress: '',
  });

  const fetchModels = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await mockApi.fetchModels();
      setModels(data);
    } catch (err) {
      setError('Failed to fetch models.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setNewModel(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmitRegisterModel = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsRegistering(true);
    setRegistrationError(null);
    setRegistrationSuccess(false);

    // Basic validation
    if (!newModel.name || !newModel.description || !newModel.hash || !newModel.version || !newModel.ownerAddress) {
      setRegistrationError('All fields are required.');
      setIsRegistering(false);
      return;
    }

    try {
      const registeredModel = await mockApi.registerModel(newModel);
      setModels(prev => [...prev, registeredModel]);
      setRegistrationSuccess(true);
      setNewModel({ name: '', description: '', hash: '', version: '', ownerAddress: '' }); // Clear form
    } catch (err) {
      setRegistrationError('Failed to register model. Please try again.');
      console.error(err);
    } finally {
      setIsRegistering(false);
    }
  };

  return (
    <div className="p-6 bg-gray-50 min-h-screen">
      <h2 className="text-3xl font-bold text-gray-800 mb-6">AI Model Marketplace</h2>

      {/* Model Registry Browser */}
      <div className="mb-8">
        <h3 className="text-2xl font-semibold text-gray-700 mb-4">Registered Models</h3>
        {loading ? (
          <p className="text-gray-600">Loading models...</p>
        ) : error ? (
          <div className="p-3 mb-4 bg-red-100 text-red-700 rounded-md">{error}</div>
        ) : models.length === 0 ? (
          <p className="text-gray-600">No models registered yet.</p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {models.map((model) => (
              <div key={model.id} className="bg-white shadow-md rounded-lg p-5 border border-gray-200 hover:shadow-lg transition-shadow">
                <h4 className="text-xl font-bold text-blue-700 mb-2">{model.name} <span className="text-sm font-normal text-gray-500">v{model.version}</span></h4>
                <p className="text-gray-600 mb-3 text-sm">{model.description}</p>
                <div className="space-y-1 text-sm">
                  <p className="text-gray-700"><strong>ID:</strong> <span className="font-mono text-xs bg-gray-100 px-1 py-0.5 rounded">{model.id}</span></p>
                  <p className="text-gray-700"><strong>Hash:</strong> <span className="font-mono text-xs bg-gray-100 px-1 py-0.5 rounded break-all">{model.hash.slice(0, 10)}...{model.hash.slice(-8)}</span></p>
                  <p className="text-gray-700"><strong>Owner:</strong> <span className="font-mono text-xs bg-gray-100 px-1 py-0.5 rounded break-all">{model.ownerAddress.slice(0, 8)}...{model.ownerAddress.slice(-6)}</span></p>
                  <p className="text-gray-700"><strong>Proofs Generated:</strong> <span className="font-semibold text-blue-600">{model.proofCount.toLocaleString()}</span></p>
                  <p className="text-gray-700"><strong>Registered:</strong> {new Date(model.registeredAt).toLocaleDateString()}</p>
                </div>
                {model.inferenceStats && (
                  <div className="mt-4 pt-4 border-t border-gray-100">
                    <h5 className="font-semibold text-gray-700 mb-2">Proof-of-Inference Stats:</h5>
                    <ul className="text-sm text-gray-600 space-y-1">
                      <li>Total Proofs: <span className="font-medium">{model.inferenceStats.totalProofs.toLocaleString()}</span></li>
                      <li>Proofs (last 24h): <span className="font-medium">{model.inferenceStats.last24hProofs.toLocaleString()}</span></li>
                      <li>Avg. Verification Time: <span className="font-medium">{model.inferenceStats.avgVerificationTimeMs} ms</span></li>
                    </ul>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Register Model Form */}
      <div className="bg-white shadow-md rounded-lg p-6 mt-8">
        <h3 className="text-2xl font-semibold text-gray-700 mb-4">Register New AI Model</h3>
        <form onSubmit={handleSubmitRegisterModel} className="space-y-4">
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700">Model Name</label>
            <input
              type="text"
              id="name"
              name="name"
              value={newModel.name}
              onChange={handleInputChange}
              className="mt-1 block w-full p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
              required
            />
          </div>
          <div>
            <label htmlFor="description" className="block text-sm font-medium text-gray-700">Description</label>
            <textarea
              id="description"
              name="description"
              value={newModel.description}
              onChange={handleInputChange}
              rows={3}
              className="mt-1 block w-full p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
              required
            ></textarea>
          </div>
          <div>
            <label htmlFor="hash" className="block text-sm font-medium text-gray-700">Model Hash (e.g., IPFS CID or content hash)</label>
            <input
              type="text"
              id="hash"
              name="hash"
              value={newModel.hash}
              onChange={handleInputChange}
              className="mt-1 block w-full p-2 border border-gray-300 rounded-md shadow-sm font-mono text-sm focus:ring-blue-500 focus:border-blue-500"
              placeholder="e.g., Qm... or 0x..."
              required
            />
          </div>
          <div>
            <label htmlFor="version" className="block text-sm font-medium text-gray-700">Version</label>
            <input
              type="text"
              id="version"
              name="version"
              value={newModel.version}
              onChange={handleInputChange}
              className="mt-1 block w-full p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
              placeholder="e.g., 1.0.0"
              required
            />
          </div>
          <div>
            <label htmlFor="ownerAddress" className="block text-sm font-medium text-gray-700">Owner Casper Address</label>
            <input
              type="text"
              id="ownerAddress"
              name="ownerAddress"
              value={newModel.ownerAddress}
              onChange={handleInputChange}
              className="mt-1 block w-full p-2 border border-gray-300 rounded-md shadow-sm font-mono text-sm focus:ring-blue-500 focus:border-blue-500"
              placeholder="e.g., 0x..."
              required
            />
          </div>
          {registrationError && (
            <div className="p-3 bg-red-100 text-red-700 rounded-md">{registrationError}</div>
          )}
          {registrationSuccess && (
            <div className="p-3 bg-green-100 text-green-700 rounded-md">Model registered successfully!</div>
          )}
          <button
            type="submit"
            className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50"
            disabled={isRegistering}
          >
            {isRegistering ? 'Registering...' : 'Register Model'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default ModelMarketplace;
