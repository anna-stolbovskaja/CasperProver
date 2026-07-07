import React from 'react';
import ErrorBoundary from './ErrorBoundary';
import { NavLink, Outlet } from 'react-router-dom';
import { useWallet } from '../../lib/CsprClickProvider';

const tabs = [
  { name: 'Overview', path: 'overview' },
  { name: 'Proofs', path: 'proofs' },
  { name: 'Models', path: 'models' },
  { name: 'Aggregation', path: 'aggregation' },
  { name: 'ZK Proofs', path: 'zk-proofs' },
  { name: 'PQ Crypto', path: 'pq-crypto' },
  { name: 'Contracts', path: 'contracts' },
  { name: 'Agent Demo', path: 'agent-demo' },
  { name: 'Playground', path: 'playground' },
  { name: 'KYC', path: 'kyc' },
];

const LabLayout: React.FC = () => {
  const { publicKey, connected: isConnected, signIn } = useWallet();

  return (
    <div className="min-h-screen bg-[#0b0b10] text-gray-100 font-sans">
      <div className="container mx-auto px-4 py-6">
        <nav className="flex items-center justify-between border-b border-[#222235] mb-6 overflow-x-auto">
          <div className="flex space-x-6">
            {tabs.map((tab) => (
              <NavLink
                key={tab.name}
                to={tab.path}
                className={({ isActive }) =>
                  `py-3 text-lg font-medium transition-colors duration-200 whitespace-nowrap ${
                    isActive
                      ? 'text-red-500 border-b-2 border-red-500'
                      : 'text-gray-400 hover:text-gray-200'
                  }`
                }
                end={tab.path === 'overview'}
              >
                {tab.name}
              </NavLink>
            ))}
          </div>
          {isConnected ? (
            <div className="flex items-center gap-2 px-4 py-2 bg-green-900/30 border border-green-700/50 text-green-400 rounded-md text-sm font-mono ml-4 flex-shrink-0">
              <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
              {publicKey ? `${publicKey.slice(0, 8)}...${publicKey.slice(-6)}` : 'Connected'}
            </div>
          ) : (
            <button
              onClick={() => signIn()}
              className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md text-sm font-medium transition-colors duration-200 ml-4 flex-shrink-0"
            >
              Connect Wallet
            </button>
          )}
        </nav>

        <main className="p-4 bg-[#13131d] border border-[#222235] rounded-lg shadow-lg min-h-[calc(100vh-200px)]">
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>
    </div>
  );
};

export default LabLayout;
