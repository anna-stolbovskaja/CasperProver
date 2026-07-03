import React, { useState, useCallback } from 'react';
import {
  Play,
  CheckCircle,
  XCircle,
  Loader2,
  Box,
  FlaskConical,
  ShieldCheck,
  GitMerge,
  ArrowRight,
  PlusCircle,
  AlertTriangle,
} from 'lucide-react';
import {
  registerModel,
  inferenceProve,
  verifyProof,
  createAggregationBatch,
  addProofToAggregationBatch,
  finalizeAggregationBatch,
  RegisterModelRequest,
  InferenceProveRequest,
  VerifyProofRequest,
  CreateBatchRequest,
  AddProofToBatchRequest,
  FinalizeBatchRequest,
} from '../../lib/api';
import { toast } from '../ui/toast';

// Helper to generate a simple UUID-like string
const generateId = (prefix: string) => `${prefix}-${Math.random().toString(36).substring(2, 10)}`;

interface Step {
  id: number;
  name: string;
  icon: React.ElementType;
  description: string;
  action: () => Promise<any>;
  status: 'idle' | 'loading' | 'success' | 'error';
  response: any;
  error: string | null;
}

const AgentDemo: React.FC = () => {
  const [currentStepIndex, setCurrentStepIndex] = useState(-1);
  const [isDemoRunning, setIsDemoRunning] = useState(false);
  const [globalError, setGlobalError] = useState<string | null>(null);

  // Data generated during the demo
  const [modelId, setModelId] = useState('');
  const [proofId, setProofId] = useState('');
  const [outputHash, setOutputHash] = useState('');
  const [batchId, setBatchId] = useState('');
  const [agentId] = useState(generateId('agent')); // Static agent ID for the demo
  const [inputData] = useState(JSON.stringify({ temperature: 0.7, prompt: 'Generate a secure proof for AI decision.' }));
  const [modelHash] = useState(generateId('modelhash'));
  const [verifierContract] = useState('09c1f7f4ff8b6b8e2fb16c2b52b38a0d3d1e3c4f5a6b7c8d9e0f1a2b3c4d5e6f'); // Verifier Gate contract

  const initialSteps: Step[] = [
    {
      id: 1,
      name: 'Register AI Box',
      icon: Box,
      description: 'Register a new AI model on the CasperProver registry.',
      action: async () => {
        const request: RegisterModelRequest = {
          modelName: `AI-Decision-Box-${generateId('')}`,
          modelHash: modelHash,
          verifierContract: verifierContract,
          description: 'Box for secure AI decision-making.',
        };
        const res = await registerModel(request);
        if (res.success && res.data) {
          setModelId(res.data.modelId);
          return res.data;
        }
        throw new Error(res.error || 'Failed to register model');
      },
      status: 'idle',
      response: null,
      error: null,
    },
    {
      id: 2,
      name: 'Run Inference & Generate Proof',
      icon: FlaskConical,
      description: 'Run an AI inference and generate a ZK proof for its decision.',
      action: async () => {
        if (!modelId) throw new Error('Model ID not available. Complete Step 1 first.');
        const request: InferenceProveRequest = {
          modelId: modelId,
          inputData: inputData,
          agentId: agentId,
        };
        const res = await inferenceProve(request);
        if (res.success && res.data) {
          setProofId(res.data.proofId);
          setOutputHash(res.data.outputHash);
          return res.data;
        }
        throw new Error(res.error || 'Failed to run inference and generate proof');
      },
      status: 'idle',
      response: null,
      error: null,
    },
    {
      id: 3,
      name: 'Verify Proof',
      icon: ShieldCheck,
      description: 'Verify the generated ZK proof on the CasperProver engine.',
      action: async () => {
        if (!proofId) throw new Error('Proof ID not available. Complete Step 2 first.');
        const request: VerifyProofRequest = {
          proofId: proofId,
        };
        const res = await verifyProof(request);
        if (res.success && res.data) {
          if (!res.data.isValid) throw new Error('Proof verification failed.');
          return res.data;
        }
        throw new Error(res.error || 'Failed to verify proof');
      },
      status: 'idle',
      response: null,
      error: null,
    },
    {
      id: 4,
      name: 'Create Proof Batch',
      icon: GitMerge,
      description: 'Create a new batch for aggregating multiple proofs.',
      action: async () => {
        const request: CreateBatchRequest = {
          batchName: `AI-Decision-Batch-${generateId('')}`,
          description: 'Batch for aggregating AI decision proofs.',
        };
        const res = await createAggregationBatch(request);
        if (res.success && res.data) {
          setBatchId(res.data.batchId);
          return res.data;
        }
        throw new Error(res.error || 'Failed to create batch');
      },
      status: 'idle',
      response: null,
      error: null,
    },
    {
      id: 5,
      name: 'Add Proof to Batch',
      icon: PlusCircle,
      description: 'Add the generated proof to the newly created batch.',
      action: async () => {
        if (!batchId) throw new Error('Batch ID not available. Complete Step 4 first.');
        if (!proofId) throw new Error('Proof ID not available. Complete Step 2 first.');
        const request: AddProofToBatchRequest = {
          batchId: batchId,
          proofId: proofId,
        };
        const res = await addProofToAggregationBatch(request);
        if (res.success) {
          return res.data;
        }
        throw new Error(res.error || 'Failed to add proof to batch');
      },
      status: 'idle',
      response: null,
      error: null,
    },
    {
      id: 6,
      name: 'Finalize Batch & Aggregate',
      icon: CheckCircle,
      description: 'Finalize the batch, generating an aggregated proof and Merkle root.',
      action: async () => {
        if (!batchId) throw new Error('Batch ID not available. Complete Step 4 first.');
        const request: FinalizeBatchRequest = {
          batchId: batchId,
        };
        const res = await finalizeAggregationBatch(request);
        if (res.success) {
          return res.data;
        }
        throw new Error(res.error || 'Failed to finalize batch');
      },
      status: 'idle',
      response: null,
      error: null,
    },
  ];

  const [steps, setSteps] = useState<Step[]>(initialSteps);

  const resetDemo = useCallback(() => {
    setSteps(initialSteps);
    setCurrentStepIndex(-1);
    setIsDemoRunning(false);
    setGlobalError(null);
    setModelId('');
    setProofId('');
    setOutputHash('');
    setBatchId('');
  }, []);

  const runStep = useCallback(async (index: number) => {
    setSteps((prev) => prev.map((s, i) => (i === index ? { ...s, status: 'loading', error: null, response: null } : s)));
    setGlobalError(null);

    try {
      const response = await steps[index].action();
      setSteps((prev) => prev.map((s, i) => (i === index ? { ...s, status: 'success', response: response } : s)));
      toast.success(`Step ${index + 1}: ${steps[index].name} completed!`);
      return true;
    } catch (err: any) {
      const errorMessage = err.message || 'An unknown error occurred.';
      setSteps((prev) => prev.map((s, i) => (i === index ? { ...s, status: 'error', error: errorMessage } : s)));
      setGlobalError(`Step ${index + 1} failed: ${errorMessage}`);
      toast.error(`Step ${index + 1}: ${steps[index].name} failed!`);
      return false;
    }
  }, [steps, modelId, proofId, batchId, agentId, inputData, modelHash, verifierContract]); // Include all dependencies

  const runFullDemo = useCallback(async () => {
    resetDemo();
    setIsDemoRunning(true);
    for (let i = 0; i < steps.length; i++) {
      setCurrentStepIndex(i);
      const success = await runStep(i);
      if (!success) {
        setIsDemoRunning(false);
        return;
      }
      // Small delay for visual effect
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    setIsDemoRunning(false);
    setCurrentStepIndex(steps.length); // Indicate completion
    toast.success('Agent Demo completed successfully!');
  }, [steps, runStep, resetDemo]);

  return (
    <div className="p-4">
      <h2 className="text-2xl font-bold text-gray-100 mb-6">AI Agent Proof Flow Demo</h2>
      <p className="text-gray-400 mb-6">
        This interactive demo simulates a full lifecycle of an AI decision proof: from model registration to batch aggregation.
        Watch the steps unfold and see real API responses.
      </p>

      <div className="flex items-center gap-4 mb-8">
        <button
          onClick={runFullDemo}
          disabled={isDemoRunning}
          className="flex items-center gap-2 px-6 py-3 bg-red-600 hover:bg-red-700 text-white rounded-md text-lg font-medium transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isDemoRunning ? <Loader2 size={24} className="animate-spin" /> : <Play size={24} />}
          {isDemoRunning ? 'Running Demo...' : 'Start Full Demo'}
        </button>
        <button
          onClick={resetDemo}
          disabled={isDemoRunning}
          className="flex items-center gap-2 px-6 py-3 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded-md text-lg font-medium transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <XCircle size={24} />
          Reset
        </button>
      </div>

      {globalError && (
        <div className="p-4 bg-red-900/30 text-red-300 border border-red-700 rounded-md mb-6 flex items-center gap-2">
          <AlertTriangle size={20} />
          <strong>Demo Error:</strong> {globalError}
        </div>
      )}

      <div className="relative flex flex-col items-center justify-center space-y-8 lg:space-y-0 lg:flex-row lg:justify-between lg:space-x-4">
        {steps.map((step, index) => (
          <React.Fragment key={step.id}>
            <div
              className={`relative flex flex-col items-center p-6 rounded-lg border transition-all duration-500 ease-in-out
                ${index <= currentStepIndex && step.status === 'success' ? 'bg-green-900/20 border-green-500' : ''}
                ${index <= currentStepIndex && step.status === 'error' ? 'bg-red-900/20 border-red-500' : ''}
                ${index === currentStepIndex && step.status === 'loading' ? 'bg-blue-900/20 border-blue-500 animate-pulse' : ''}
                ${index > currentStepIndex || step.status === 'idle' ? 'bg-[#1a1a2a] border-[#222235]' : ''}
                w-full lg:w-1/6 min-h-[250px]
              `}
            >
              <div className="flex items-center justify-center w-12 h-12 rounded-full bg-red-600 text-white text-xl font-bold mb-4">
                {step.id}
              </div>
              <div className="text-center">
                <step.icon size={32} className="mx-auto text-red-400 mb-2" />
                <h3 className="text-lg font-semibold text-gray-100 mb-2">{step.name}</h3>
                <p className="text-sm text-gray-400">{step.description}</p>
              </div>

              {step.status === 'loading' && (
                <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-70 rounded-lg">
                  <Loader2 size={32} className="animate-spin text-blue-400" />
                </div>
              )}
              {step.status === 'success' && (
                <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-70 rounded-lg">
                  <CheckCircle size={32} className="text-green-500" />
                </div>
              )}
              {step.status === 'error' && (
                <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-70 rounded-lg">
                  <XCircle size={32} className="text-red-500" />
                </div>
              )}
            </div>
            {index < steps.length - 1 && (
              <div className="relative lg:w-1/12 flex items-center justify-center">
                <ArrowRight
                  size={32}
                  className={`text-gray-500 transition-colors duration-500
                    ${index < currentStepIndex && steps[index].status === 'success' ? 'text-green-500' : ''}
                    ${index === currentStepIndex && steps[index].status === 'loading' ? 'text-blue-500 animate-pulse' : ''}
                  `}
                />
                <div className="absolute w-0.5 h-16 lg:h-0.5 lg:w-16 bg-[#222235] lg:top-1/2 lg:-translate-y-1/2 -z-10"></div>
              </div>
            )}
          </React.Fragment>
        ))}
      </div>

      <div className="mt-12 space-y-6">
        <h3 className="text-xl font-bold text-gray-100">API Responses:</h3>
        {steps.map((step) => (
          <div key={`response-${step.id}`} className="bg-[#1a1a2a] p-4 rounded-lg border border-[#222235]">
            <h4 className="text-lg font-semibold text-gray-200 mb-2">
              Step {step.id}: {step.name}
            </h4>
            {step.status === 'loading' && (
              <p className="text-blue-400 flex items-center gap-2">
                <Loader2 size={18} className="animate-spin" /> Executing...
              </p>
            )}
            {step.status === 'success' && (
              <div className="space-y-2">
                <p className="text-green-400 flex items-center gap-2">
                  <CheckCircle size={18} /> Success!
                </p>
                <pre className="bg-[#0b0b10] p-3 rounded-md font-mono text-sm overflow-x-auto border border-[#222235]">
                  {JSON.stringify(step.response, null, 2)}
                </pre>
              </div>
            )}
            {step.status === 'error' && (
              <div className="space-y-2">
                <p className="text-red-400 flex items-center gap-2">
                  <XCircle size={18} /> Error:
                </p>
                <pre className="bg-[#0b0b10] p-3 rounded-md font-mono text-sm overflow-x-auto border border-[#222235] text-red-300">
                  {step.error}
                </pre>
              </div>
            )}
            {step.status === 'idle' && (
              <p className="text-gray-500">Awaiting execution...</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default AgentDemo;
