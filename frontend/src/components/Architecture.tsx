import { ExternalLink } from 'lucide-react'

const CONTRACTS = [
  { name: 'proof-registry', hash: '96e97c4d564fe737', deploy: 'd64299b651750b69' },
  { name: 'verifier-gate', hash: 'a37f9cde9dbdc5bb', deploy: 'c1320d182c032367' },
  { name: 'defi-mock', hash: 'b9b11a976af20b4b', deploy: '6ed38d8dc4c55908' },
]

const STACK = ['Go 1.22', 'Rust', 'casper-contract 5.1', 'Merkle Trees', 'SHA-256', 'React', 'TypeScript']

export default function Architecture() {
  return (
    <section id="architecture" className="py-16 sm:py-24 relative">
      {/* Decorative transparent mascot */}
      <img src="/images/mascot-gradient-t.png" alt="" aria-hidden="true"
        className="absolute bottom-0 right-0 w-72 opacity-[0.04] pointer-events-none select-none translate-x-1/4" loading="lazy" />

      <div className="cp-section">
        <div className="text-center mb-10">
          <h2 className="text-2xl sm:text-3xl font-bold mb-3">Architecture</h2>
          <p className="text-cp-gray max-w-lg mx-auto">
            Three smart contracts on Casper testnet. Real deploys, real verification.
          </p>
        </div>

        {/* Contracts table */}
        <div className="max-w-3xl mx-auto mb-10 overflow-x-auto">
          <table className="w-full text-sm" role="table">
            <thead>
              <tr className="border-b border-cp-border">
                <th className="text-left py-3 px-4 text-cp-gray font-medium text-xs uppercase tracking-wider">Contract</th>
                <th className="text-left py-3 px-4 text-cp-gray font-medium text-xs uppercase tracking-wider">Hash</th>
                <th className="text-left py-3 px-4 text-cp-gray font-medium text-xs uppercase tracking-wider">Deploy TX</th>
              </tr>
            </thead>
            <tbody>
              {CONTRACTS.map(c => (
                <tr key={c.name} className="border-b border-cp-border/50 hover:bg-cp-red/[0.02] transition-colors">
                  <td className="py-3 px-4">
                    <span className="font-mono text-cp-red text-xs">{c.name}</span>
                  </td>
                  <td className="py-3 px-4">
                    <a href={`https://testnet.cspr.live/contract/${c.hash}`} target="_blank" rel="noopener noreferrer"
                      className="font-mono text-xs text-cp-gray hover:text-white transition-colors inline-flex items-center gap-1 cursor-pointer">
                      {c.hash}... <ExternalLink size={10} />
                    </a>
                  </td>
                  <td className="py-3 px-4">
                    <a href={`https://testnet.cspr.live/deploy/${c.deploy}`} target="_blank" rel="noopener noreferrer"
                      className="font-mono text-xs text-cp-gray hover:text-white transition-colors inline-flex items-center gap-1 cursor-pointer">
                      {c.deploy}... <ExternalLink size={10} />
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Architecture diagram */}
        <div className="max-w-3xl mx-auto cp-card !p-6 mb-10">
          <div className="flex flex-wrap items-center justify-center gap-2 sm:gap-3 text-xs font-mono">
            {['AI Agent', '→', 'CasperProver Engine', '→', 'Casper RPC', '→'].map((s, i) => (
              <span key={i} className={s === '→' ? 'text-cp-red' : 'bg-cp-black px-3 py-1.5 rounded border border-cp-border text-cp-gray'}>{s}</span>
            ))}
            <div className="flex flex-col gap-1">
              {CONTRACTS.map(c => (
                <span key={c.name} className="bg-cp-red/10 border border-cp-red/20 px-3 py-1 rounded text-cp-red text-[10px]">{c.name}</span>
              ))}
            </div>
          </div>
        </div>

        {/* Tech stack pills */}
        <div className="flex flex-wrap justify-center gap-2">
          {STACK.map(s => (
            <span key={s} className="px-3 py-1.5 rounded-full bg-cp-card border border-cp-border text-xs text-cp-gray">{s}</span>
          ))}
        </div>
      </div>
    </section>
  )
}
