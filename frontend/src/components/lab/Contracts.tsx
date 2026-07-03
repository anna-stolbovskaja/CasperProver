import React from 'react';
import { Link } from 'react-router-dom';
import { Link as LinkIcon, FileText, Code, Shield } from 'lucide-react';

const CONTRACTS = {
  'Proof Registry': '7f71b026b79014c5b82e2db4c2e8cf7fa23c2a23e7e06247d4078c12ac28e5d6',
  'Verifier Gate': '09c1f7f4ff8b6b8e2fb16c2b52b38a0d3d1e3c4f5a6b7c8d9e0f1a2b3c4d5e6f',
  'DeFi Mock': 'a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef12345678',
};

const EXPLORER_BASE_URL = 'https://testnet.cspr.live/contract/';

const Contracts: React.FC = () => {
  return (
    <div className="p-4">
      <h2 className="text-2xl font-bold text-gray-100 mb-6">CasperProver Contracts</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {Object.entries(CONTRACTS).map(([name, address]) => (
          <div key={name} className="bg-[#1a1a2a] p-6 rounded-lg border border-[#222235] shadow-md flex flex-col justify-between">
            <div>
              <div className="flex items-center gap-3 mb-3">
                {name === 'Proof Registry' && <FileText size={28} className="text-red-500" />}
                {name === 'Verifier Gate' && <Shield size={28} className="text-red-500" />}
                {name === 'DeFi Mock' && <Code size={28} className="text-red-500" />}
                <h3 className="text-xl font-semibold text-gray-100">{name}</h3>
              </div>
              <p className="text-gray-400 mb-3">
                Address: <span className="font-mono text-red-300 break-all">{address}</span>
              </p>
              <p className="text-gray-500 text-sm mb-4">
                {name === 'Proof Registry' && 'Stores and manages all ZK proof metadata on the Casper blockchain.'}
                {name === 'Verifier Gate' && 'Acts as a gateway for on-chain verification of ZK proofs.'}
                {name === 'DeFi Mock' && 'A mock DeFi contract for testing interactions with ZK proofs.'}
              </p>
            </div>
            <a
              href={`${EXPLORER_BASE_URL}${address}`}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-4 inline-flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md transition-colors duration-200 text-sm font-medium"
            >
              <LinkIcon size={18} />
              View on CasperLive
            </a>
          </div>
        ))}
      </div>

      <div className="mt-8 p-6 bg-[#1a1a2a] rounded-lg border border-[#222235] shadow-md">
        <h3 className="text-xl font-semibold text-gray-100 mb-4">About Casper Network</h3>
        <p className="text-gray-400 leading-relaxed">
          The Casper Network is a live proof-of-stake blockchain optimized for enterprise and developer adoption.
          It features upgradeable contracts, predictable gas fees, and WebAssembly (Wasm) support, making it an
          ideal platform for sophisticated applications like CasperProver. Our smart contracts are deployed
          on the Casper testnet for development and testing purposes.
        </p>
        <a
          href="https://casper.network/"
          target="_blank"
          rel="noopener noreferrer"
          className="mt-4 inline-flex items-center gap-2 text-red-500 hover:text-red-400 transition-colors duration-200"
        >
          Learn more about Casper <LinkIcon size={16} />
        </a>
      </div>
    </div>
  );
};

export default Contracts;
