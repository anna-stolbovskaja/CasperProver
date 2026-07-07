import React from 'react';
import ErrorBoundary from './ErrorBoundary';
import { NavLink, Outlet } from 'react-router-dom';
import { useWallet } from '../../lib/CsprClickProvider';
import { shortKey } from '../../lib/wallet';
import { Wallet, FlaskConical, Radio } from 'lucide-react';

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
  const { publicKey, connected: isConnected, signIn, signOut } = useWallet();

  return (
    <div className="min-h-screen bg-[#0b0b10] text-gray-100 font-sans">
      {/* Sticky top bar: wallet + mode indicator */}
      <div className="sticky top-0 z-40 bg-[#0b0b10]/95 backdrop-blur border-b border-[#222235]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          {/* Top row: logo + wallet */}
          <div className="flex items-center justify-between h-14">
            <a href="/" className="flex items-center gap-2">
              <img src="/images/logo.webp" alt="CasperProver" className="h-6 w-auto" />
              <span className="font-bold text-white text-base hidden sm:block">CasperProver</span>
            </a>

            <div className="flex items-center gap-3">
              {/* Demo / Testnet mode badge */}
              {isConnected ? (
                <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-green-900/30 border border-green-700/40 text-green-400 text-xs font-medium">
                  <Radio className="w-3 h-3" />
                  Testnet
                </span>
              ) : (
                <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-yellow-900/30 border border-yellow-700/40 text-yellow-400 text-xs font-medium">
                  <FlaskConical className="w-3 h-3" />
                  Demo Mode
                </span>
              )}

              {/* Wallet button */}
              {isConnected ? (
                <button
                  onClick={() => signOut()}
                  className="flex items-center gap-2 px-3 py-1.5 bg-green-900/30 border border-green-700/50 text-green-400 rounded-lg text-sm font-mono hover:bg-green-900/50 transition-colors"
                >
                  <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
                  {publicKey ? shortKey(publicKey) : 'Connected'}
                </button>
              ) : (
                <button
                  onClick={() => signIn()}
                  className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm font-medium transition-colors"
                >
                  <Wallet className="w-4 h-4" />
                  Connect Wallet
                </button>
              )}
            </div>
          </div>

          {/* Tab row */}
          <nav className="flex overflow-x-auto -mb-px scrollbar-none">
            {tabs.map((tab) => (
              <NavLink
                key={tab.name}
                to={tab.path}
                className={({ isActive }) =>
                  `py-2.5 px-3 text-sm font-medium transition-colors duration-200 whitespace-nowrap border-b-2 ${
                    isActive
                      ? 'text-red-500 border-red-500'
                      : 'text-gray-400 border-transparent hover:text-gray-200 hover:border-gray-600'
                  }`
                }
                end={tab.path === 'overview'}
              >
                {tab.name}
              </NavLink>
            ))}
          </nav>
        </div>
      </div>

      {/* Demo mode banner (below sticky header, only when no wallet) */}
      {!isConnected && (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-2">
          <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-yellow-900/20 border border-yellow-700/30 text-yellow-300 text-xs">
            <FlaskConical className="w-3.5 h-3.5 shrink-0" />
            <span>
              <strong>Demo mode</strong> — actions use emulated data. Connect a Casper wallet to interact with testnet.
            </span>
          </div>
        </div>
      )}

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
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
