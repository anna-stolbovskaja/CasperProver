import React from 'react';
import ErrorBoundary from './ErrorBoundary';
import { NavLink, Outlet } from 'react-router-dom';
import { Wallet } from 'lucide-react';

const tabs = [
  { name: 'Overview', path: 'overview' },
  { name: 'Proofs', path: 'proofs' },
  { name: 'Models', path: 'models' },
  { name: 'Aggregation', path: 'aggregation' },
  { name: 'PQ Crypto', path: 'pq-crypto' },
  { name: 'Contracts', path: 'contracts' },
  { name: 'Agent Demo', path: 'agent-demo' },
  { name: 'Playground', path: 'playground' },
  { name: 'KYC', path: 'kyc' },
];

const LabLayout: React.FC = () => {
  return (
    <div className="min-h-screen bg-[#0b0b10] text-gray-100 font-sans">
      <header className="py-4 px-6 border-b border-[#222235] flex items-center justify-between">
        <h1 className="text-2xl font-bold text-red-500">CasperProver Lab</h1>

      </header>

      <div className="container mx-auto px-4 py-6">
        <nav className="flex items-center justify-between border-b border-[#222235] mb-6 overflow-x-auto">
          <div className="flex space-x-6">
            {tabs.map((tab) => (
              <NavLink
                key={tab.name}
                to={tab.path}
                className={({ isActive }) =>
                  `py-3 text-lg font-medium transition-colors duration-200 ${
                    isActive
                      ? 'text-red-500 border-b-2 border-red-500'
                      : 'text-gray-400 hover:text-gray-200'
                  }`
                }
                end={tab.path === 'overview'} // 'end' prop for exact match on the index route
              >
                {tab.name}
              </NavLink>
            ))}
          </div>
          <button className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-md text-sm font-medium transition-colors duration-200 ml-4 flex-shrink-0">
            <Wallet size={18} />
            Connect Wallet
          </button>
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
