import React from 'react';
import SectionIntro from './SectionIntro';
import { Link as LinkIcon, FileText, Code, Shield, Swords, Box, Layers, Brain, ExternalLink } from 'lucide-react';

interface ContractInfo {
  name: string;
  address: string | null;
  purpose: string;
  icon: React.ElementType;
  deployed: boolean;
  lines: number;
  deployDate?: string;
}

const CONTRACTS: ContractInfo[] = [
  {
    name: 'proof-registry',
    address: '96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708',
    purpose: 'Immutable on-chain store for all proof metadata — hashes, Merkle roots, timestamps, and verification status.',
    icon: FileText,
    deployed: true,
    lines: 251,
    deployDate: '2026-06-29',
  },
  {
    name: 'verifier-gate',
    address: 'a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3',
    purpose: 'Gateway contract for Merkle inclusion verification — checks proof existence and validity via cross-contract calls.',
    icon: Shield,
    deployed: true,
    lines: 143,
    deployDate: '2026-06-29',
  },
  {
    name: 'defi-mock',
    address: 'fe0c45f67c8cd99f0bda0047399a113588870ec0d79d9102f44107303f0b39ef',
    purpose: 'KYC-gated DeFi vault — demonstrates proof-based access control for financial operations via cross-contract verification.',
    icon: Code,
    deployed: true,
    lines: 202,
    deployDate: '2026-07-07',
  },
  {
    name: 'stake-slashing',
    address: '1ad1b3d94be631532d6daf3a195fafc9dfe8a16504e87d87784d51089b983d52',
    purpose: 'Economic penalty contract — 20% CSPR slash on revoked proofs with permissionless bounty for reporters. Cross-contract call to proof-registry.',
    icon: Swords,
    deployed: true,
    lines: 273,
    deployDate: '2026-07-07',
  },
  {
    name: 'proof-of-inference',
    address: null,
    purpose: 'Full inference proof contract — records model hash, input/output commitments, and verification result on-chain for each AI decision.',
    icon: Brain,
    deployed: false,
    lines: 498,
  },
  {
    name: 'model-registry',
    address: null,
    purpose: 'On-chain model versioning registry — tracks model hashes, ownership, and version history for provenance auditing.',
    icon: Box,
    deployed: false,
    lines: 372,
  },
  {
    name: 'proof-aggregation',
    address: null,
    purpose: 'Batch aggregation contract — stores Merkle roots of aggregated proof batches for gas-efficient on-chain verification.',
    icon: Layers,
    deployed: false,
    lines: 179,
  },
];

const EXPLORER_BASE_URL = 'https://testnet.cspr.live/contract/';
const GITHUB_BASE_URL = 'https://github.com/anna-stolbovskaja/CasperProver/tree/main/contracts/';

const Contracts: React.FC = () => {
  const deployed = CONTRACTS.filter(c => c.deployed);
  const written = CONTRACTS.filter(c => !c.deployed);

  return (
    <div className="p-4">
      <SectionIntro
        title="Smart Contracts"
        description="7 Rust/Wasm smart contracts built for CasperProver: 4 deployed on Casper testnet (proof_registry, verifier_gate, defi_mock, stake_slashing) and 3 written but not yet deployed (proof-of-inference, model-registry, proof-aggregation). Each contract is verified on-chain with real deploy hashes — click to view on Casper Explorer."
        dataSource="Real smart contracts deployed on Casper testnet. Deploy hashes and contract hashes verified via CSPR.cloud API."
        badge="On-chain verified"
        badgeColor="green"
      />
      <h2 className="text-2xl font-bold text-gray-100 mb-2">CasperProver Contracts</h2>
      <p className="text-gray-400 mb-6">
        {deployed.length} deployed on Casper testnet · {written.length} written and ready for mainnet · {CONTRACTS.reduce((s, c) => s + c.lines, 0).toLocaleString()} lines of Rust
      </p>

      {/* Deployed contracts */}
      <h3 className="text-lg font-semibold text-green-400 mb-4 flex items-center gap-2">
        <span className="w-2 h-2 rounded-full bg-green-400" /> Deployed on Testnet
      </h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        {deployed.map(c => (
          <div key={c.name} className="bg-[#1a1a2a] p-5 rounded-lg border border-[#222235] shadow-md">
            <div className="flex items-center gap-3 mb-3">
              {React.createElement(c.icon, { size: 24, className: 'text-red-500' })}
              <h3 className="text-lg font-semibold text-gray-100">{c.name}</h3>
              <span className="ml-auto text-xs text-green-400/70 border border-green-500/20 px-2 py-0.5 rounded">
                {c.deployDate}
              </span>
            </div>
            <p className="text-gray-400 text-sm mb-3">{c.purpose}</p>
            <div className="text-xs text-gray-500 font-mono mb-3 break-all">
              {c.address?.slice(0, 16)}...{c.address?.slice(-8)}
            </div>
            <div className="flex items-center gap-3">
              <a
                href={`${EXPLORER_BASE_URL}${c.address}`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white rounded-md text-xs font-medium transition-colors"
              >
                <LinkIcon size={14} /> CasperLive
              </a>
              <a
                href={`${GITHUB_BASE_URL}${c.name}`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-3 py-1.5 border border-gray-700 hover:border-gray-500 text-gray-300 rounded-md text-xs font-medium transition-colors"
              >
                <ExternalLink size={14} /> Source ({c.lines} lines)
              </a>
            </div>
          </div>
        ))}
      </div>

      {/* Written contracts */}
      <h3 className="text-lg font-semibold text-yellow-400 mb-4 flex items-center gap-2">
        <span className="w-2 h-2 rounded-full bg-yellow-400" /> Written — Ready for Mainnet
      </h3>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
        {written.map(c => (
          <div key={c.name} className="bg-[#1a1a2a] p-5 rounded-lg border border-[#222235]/60 shadow-md">
            <div className="flex items-center gap-3 mb-3">
              {React.createElement(c.icon, { size: 24, className: 'text-yellow-500' })}
              <h3 className="text-lg font-semibold text-gray-100">{c.name}</h3>
            </div>
            <p className="text-gray-400 text-sm mb-3">{c.purpose}</p>
            <a
              href={`${GITHUB_BASE_URL}${c.name}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 px-3 py-1.5 border border-gray-700 hover:border-gray-500 text-gray-300 rounded-md text-xs font-medium transition-colors"
            >
              <ExternalLink size={14} /> Source ({c.lines} lines)
            </a>
          </div>
        ))}
      </div>

      {/* Additional info */}
      <div className="bg-[#1a1a2a] p-5 rounded-lg border border-[#222235] shadow-md">
        <h3 className="text-lg font-semibold text-gray-100 mb-3">Architecture Notes</h3>
        <ul className="text-gray-400 text-sm space-y-2">
          <li>• All contracts are written in Rust for Casper 2.x (CEP-18 compatible)</li>
          <li>• <strong>stake-slashing</strong> uses cross-contract calls to read proof state from <strong>proof-registry</strong></li>
          <li>• <strong>defi-mock</strong> demonstrates KYC gating via cross-contract verification against stored proofs</li>
          <li>• <strong>stake-slashing-session</strong> — helper session contract for initiating stake/slash operations</li>
          <li>• Integration tests cover all entry points (<code>contracts/tests/</code>)</li>
          <li>• 248+ testnet transactions across contract deploys and entry-point calls</li>
        </ul>
      </div>
    </div>
  );
};

export default Contracts;
