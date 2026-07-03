import React, { useState, useEffect, useCallback } from 'react';
import { PlusCircle, Search, FileText, Loader2, AlertTriangle, Eye } from 'lucide-react';
import {
  registerModel,
  getModelById,
  RegisterModelRequest,
  RegisterModelResponse,
  ModelDetails,
} from '../../lib/api';
import { toast } from 'react-toastify';
import Modal from '../ui/Modal'; // Assuming a generic Modal component

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

const Models: React.FC = () => {
  const [isRegisterModalOpen, setIsRegisterModalOpen] = useState(false);
  const [newModelData, setNewModelData] = useState<RegisterModelRequest>({
    modelName: '',
    modelHash: '',
    verifierContract: '',
    description: '',
  });
  const [isRegistering, setIsRegistering] = useState(false);

  const [searchModelId, setSearchModelId] = useState('');
  const [foundModel, setFoundModel] = useState<ModelDetails | null>(null);
  const [isSearchingModel, setIsSearchingModel] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);

  // For "List registered models" - since API doesn't have a direct list, we'll simulate or show registered models after creation
  const [registeredModels, setRegisteredModels] = useState<ModelDetails[]>([]);

  const handleRegisterModelChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setNewModelData((prev) => ({ ...prev, [name]: value }));
  };

  const handleRegisterModelSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsRegistering(true);
    setSearchError(null);
    try {
      const res = await registerModel(newModelData);
      if (res.success && res.data) {
        toast.success(`Model "${newModelData.modelName}" registered successfully! ID: ${res.data.modelId}`);
        // Add to local list for display (assuming we can fetch it back)
        const fetchedModelRes = await getModelById(res.data.modelId);
        if (fetchedModelRes.success && fetchedModelRes.data) {
          setRegisteredModels((prev) => [...prev, fetchedModelRes.data]);
        }
        setIsRegisterModalOpen(false);
        setNewModelData({ modelName: '', modelHash: '', verifierContract: '', description: '' });
      } else {
        toast.error(res.error || 'Failed to register model');
      }
    } catch (err) {
      toast.error('An unexpected error occurred during model registration.');
      console.error(err);
    } finally {
      setIsRegistering(false);
    }
  };

  const handleSearchModel = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchModelId) {
      setSearchError('Please enter a Model ID to search.');
      setFoundModel(null);
      return;
    }
    setIsSearchingModel(true);
    setSearchError(null);
    setFoundModel(null);
    try {
      const res = await getModelById(searchModelId);
      if (res.success && res.data) {
        setFoundModel(res.data);
        toast.success(`Model ${searchModelId} found.`);
      } else {
        setSearchError(res.error || `Model with ID ${searchModelId} not found.`);
        toast.error(res.error || `Model with ID ${searchModelId} not found.`);
      }
    } catch (err) {
      setSearchError('An unexpected error occurred during model search.');
      toast.error('An unexpected error occurred during model search.');
      console.error(err);
    } finally {
      setIsSearchingModel(false);
    }
  };

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-100">AI Model Registry</h2>
        <button
          onClick={() => setIsRegisterModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200"
        >
          <PlusCircle size={20} />
          Register New Model
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Register Model Form (or button to open modal) */}
        <div className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
          <h3 className="text-xl font-semibold text-gray-100 mb-4">Register AI Model</h3>
          <p className="text-gray-400 mb-4">
            Register a new AI model with its unique hash and the associated verifier contract on Casper.
          </p>
          <button
            onClick={() => setIsRegisterModalOpen(true)}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200"
          >
            <PlusCircle size={20} />
            Open Registration Form
          </button>
        </div>

        {/* View Model Details */}
        <div className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
          <h3 className="text-xl font-semibold text-gray-100 mb-4">View Model Details</h3>
          <form onSubmit={handleSearchModel} className="space-y-4">
            <div>
              <label htmlFor="searchModelId" className="block text-sm font-medium text-gray-300 mb-1">
                Model ID
              </label>
              <div className="relative">
                <input
                  type="text"
                  id="searchModelId"
                  value={searchModelId}
                  onChange={(e) => setSearchModelId(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 placeholder-gray-500 focus:ring-red-500 focus:border-red-500"
                  placeholder="Enter Model ID"
                  required
                />
                <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              </div>
            </div>
            <button
              type="submit"
              disabled={isSearchingModel}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSearchingModel ? <Loader2 size={20} className="animate-spin" /> : <Eye size={20} />}
              {isSearchingModel ? 'Searching...' : 'View Model'}
            </button>
          </form>

          {searchError && (
            <div className="mt-4 p-3 bg-red-900/30 text-red-300 border border-red-700 rounded-md flex items-center gap-2">
              <AlertTriangle size={20} /> {searchError}
            </div>
          )}

          {foundModel && (
            <div className="mt-6 p-4 bg-[#0b0b10] border border-[#222235] rounded-md space-y-2 text-sm">
              <h4 className="text-lg font-semibold text-red-400">Model Found:</h4>
              <p><span className="font-medium text-gray-300">ID:</span> <span className="font-mono break-all">{foundModel.modelId}</span></p>
              <p><span className="font-medium text-gray-300">Name:</span> {foundModel.modelName}</p>
              <p><span className="font-medium text-gray-300">Hash:</span> <span className="font-mono break-all">{foundModel.modelHash}</span></p>
              <p><span className="font-medium text-gray-300">Verifier Contract:</span> <span className="font-mono break-all">{foundModel.verifierContract}</span></p>
              {foundModel.description && <p><span className="font-medium text-gray-300">Description:</span> {foundModel.description}</p>}
              <p><span className="font-medium text-gray-300">Registered At:</span> {new Date(foundModel.registeredAt).toLocaleString()}</p>
            </div>
          )}
        </div>
      </div>

      {/* List Registered Models (placeholder) */}
      <div className="mt-8 bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md">
        <h3 className="text-xl font-semibold text-gray-100 mb-4">Recently Registered Models</h3>
        {registeredModels.length === 0 ? (
          <div className="text-center p-4 text-gray-400">
            <FileText className="mx-auto mb-2" size={32} />
            <p>No models registered yet in this session.</p>
            <p className="text-sm mt-1">Registered models would appear here if a "list all models" API endpoint existed.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-[#222235]">
              <thead className="bg-[#13131d]">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                    Model ID
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                    Name
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                    Hash
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                    Verifier Contract
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#222235]">
                {registeredModels.map((model) => (
                  <tr key={model.modelId} className="hover:bg-[#1a1a2a]/50 transition-colors duration-150">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                      {model.modelId.substring(0, 8)}...{model.modelId.substring(model.modelId.length - 8)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{model.modelName}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                      {model.modelHash.substring(0, 8)}...
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-300">
                      {model.verifierContract.substring(0, 8)}...
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>


      {/* Register Model Modal */}
      <Modal
        isOpen={isRegisterModalOpen}
        onClose={() => setIsRegisterModalOpen(false)}
        title="Register New AI Model"
      >
        <form onSubmit={handleRegisterModelSubmit} className="space-y-4">
          <div>
            <label htmlFor="modelName" className="block text-sm font-medium text-gray-300 mb-1">
              Model Name
            </label>
            <input
              type="text"
              id="modelName"
              name="modelName"
              value={newModelData.modelName}
              onChange={handleRegisterModelChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
              required
            />
          </div>
          <div>
            <label htmlFor="modelHash" className="block text-sm font-medium text-gray-300 mb-1">
              Model Hash (e.g., IPFS CID, cryptographic hash)
            </label>
            <input
              type="text"
              id="modelHash"
              name="modelHash"
              value={newModelData.modelHash}
              onChange={handleRegisterModelChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 font-mono focus:ring-red-500 focus:border-red-500"
              required
            />
          </div>
          <div>
            <label htmlFor="verifierContract" className="block text-sm font-medium text-gray-300 mb-1">
              Verifier Contract Address (on Casper)
            </label>
            <input
              type="text"
              id="verifierContract"
              name="verifierContract"
              value={newModelData.verifierContract}
              onChange={handleRegisterModelChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 font-mono focus:ring-red-500 focus:border-red-500"
              required
            />
          </div>
          <div>
            <label htmlFor="description" className="block text-sm font-medium text-gray-300 mb-1">
              Description (Optional)
            </label>
            <textarea
              id="description"
              name="description"
              rows={3}
              value={newModelData.description}
              onChange={handleRegisterModelChange}
              className="w-full p-2 bg-[#0b0b10] border border-[#222235] rounded-md text-gray-100 focus:ring-red-500 focus:border-red-500"
            ></textarea>
          </div>
          <button
            type="submit"
            disabled={isRegistering}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isRegistering ? <Loader2 size={20} className="animate-spin" /> : <PlusCircle size={20} />}
            {isRegistering ? 'Registering...' : 'Register Model'}
          </button>
        </form>
      </Modal>
    </div>
  );
};

export default Models;
