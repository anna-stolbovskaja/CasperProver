import { useState, useEffect, useCallback } from 'react'
import { Shield, Hash, GitBranch, Wallet, LogOut, Play, Loader2, CheckCircle, XCircle, List, BarChart3 } from 'lucide-react'
import { createProof, getProofs, getHealth } from '../lib/api'
import type { ProofRecord } from '../lib/api'
import { connectWallet, disconnectWallet, shortKey } from '../lib/wallet'
import type { WalletState } from '../lib/wallet'

export default function Dashboard() {
  const [proofs, setProofs] = useState<ProofRecord[]>([])
  const [wallet, setWallet] = useState<WalletState>({ connected: false, publicKey: null, accountHash: null, simulated: false })
  const [chain, setChain] = useState('connecting...')
  const [agent, setAgent] = useState('agent-alpha')
  const [input, setInput] = useState('loan_decision_42')
  const [model, setModel] = useState('gpt-4o')
  const [output, setOutput] = useState('approved')
  const [loading, setLoading] = useState(false)
  const [selected, setSelected] = useState<ProofRecord | null>(null)

  useEffect(() => {
    getHealth().then(h => setChain(h.status === 'ok' ? 'casper-test' : 'offline')).catch(() => setChain('offline'))
  }, [])

  const handleConnect = async () => {
    if (wallet.connected) { setWallet(disconnectWallet()); setProofs([]); return }
    const state = await connectWallet()
    setWallet(state)
    try { const p = await getProofs(); setProofs(p) } catch {}
  }

  const handleCreate = async () => {
    if (loading) return
    setLoading(true)
    try {
      const proof = await createProof({ agent, input, output, model })
      setProofs(prev => [proof, ...prev])
      setSelected(proof)
    } catch {}
    setLoading(false)
  }

  const valid = proofs.filter(p => p.valid && !p.revoked).length

  return (
    <div className="min-h-screen bg-cp-black pt-20">
      <div className="cp-section py-8">
        {/* Header */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-extrabold text-white">Proof Dashboard</h1>
            <p className="text-gray-500 text-sm mt-1">Generate and inspect cryptographic proofs</p>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-cp-card border border-cp-border text-xs">
              <span className={`w-2 h-2 rounded-full ${chain === 'casper-test' ? 'bg-green-500' : 'bg-red-500'}`} />
              <span className="text-gray-400 font-mono">{chain}</span>
            </div>
            <button onClick={handleConnect} className={`inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold transition-all ${
              wallet.connected ? 'bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20' : 'bg-red-600 text-white hover:bg-red-500'
            }`}>
              {wallet.connected ? <LogOut className="w-4 h-4" /> : <Wallet className="w-4 h-4" />}
              {wallet.connected ? shortKey(wallet.publicKey || '') : 'Connect Wallet'}
            </button>
          </div>
        </div>

        {!wallet.connected ? (
          <div className="bg-cp-card rounded-2xl border border-cp-border p-16 text-center">
            <Wallet className="w-12 h-12 text-gray-700 mx-auto mb-4" />
            <h2 className="text-xl font-bold text-white mb-2">Connect wallet to begin</h2>
            <p className="text-gray-500 mb-6">Connect a Casper wallet to generate and manage proofs.</p>
            <button onClick={handleConnect} className="inline-flex items-center gap-2 px-8 py-3 bg-red-600 text-white font-semibold rounded-xl hover:bg-red-500 transition-colors">
              <Wallet className="w-4 h-4" /> Connect Wallet
            </button>
          </div>
        ) : (
          <>
            {/* Stats */}
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="bg-cp-card rounded-xl border border-cp-border p-5">
                <List className="w-5 h-5 text-gray-600 mb-2" />
                <p className="text-2xl font-bold text-white">{proofs.length}</p>
                <p className="text-xs text-gray-500">Total Proofs</p>
              </div>
              <div className="bg-cp-card rounded-xl border border-cp-border p-5">
                <CheckCircle className="w-5 h-5 text-green-500/60 mb-2" />
                <p className="text-2xl font-bold text-white">{valid}</p>
                <p className="text-xs text-gray-500">Valid</p>
              </div>
              <div className="bg-cp-card rounded-xl border border-cp-border p-5">
                <BarChart3 className="w-5 h-5 text-red-500/60 mb-2" />
                <p className="text-2xl font-bold text-white">{proofs.length > 0 ? proofs[0]?.merkle_path?.length || 0 : 0}</p>
                <p className="text-xs text-gray-500">Merkle Depth</p>
              </div>
            </div>

            <div className="grid lg:grid-cols-5 gap-6">
              {/* Create proof */}
              <div className="lg:col-span-2 bg-cp-card rounded-2xl border border-cp-border p-6">
                <h3 className="text-white font-bold mb-4 flex items-center gap-2"><Shield className="w-4 h-4 text-red-400" /> New Proof</h3>
                <div className="space-y-3">
                  {[
                    { l: 'Agent', v: agent, s: setAgent },
                    { l: 'Input', v: input, s: setInput },
                    { l: 'Model', v: model, s: setModel },
                    { l: 'Output', v: output, s: setOutput },
                  ].map((f, i) => (
                    <div key={i}>
                      <label className="text-xs text-gray-600 font-mono mb-1 block">{f.l}</label>
                      <input value={f.v} onChange={e => f.s(e.target.value)} className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50" />
                    </div>
                  ))}
                  <button onClick={handleCreate} disabled={loading} className="w-full mt-2 inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-red-600 text-white text-sm font-semibold rounded-lg hover:bg-red-500 disabled:opacity-50 transition-all">
                    {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                    {loading ? 'Generating...' : 'Generate Proof'}
                  </button>
                </div>
              </div>

              {/* Proof detail / list */}
              <div className="lg:col-span-3 bg-cp-card rounded-2xl border border-cp-border p-6">
                <h3 className="text-white font-bold mb-4 flex items-center gap-2"><Hash className="w-4 h-4 text-red-400" /> Proofs</h3>
                {selected ? (
                  <div className="space-y-2">
                    <button onClick={() => setSelected(null)} className="text-xs text-gray-500 hover:text-gray-300 mb-2">&larr; Back to list</button>
                    <div className="bg-black/40 rounded-xl p-4 font-mono text-xs space-y-1.5 overflow-x-auto">
                      <p className="text-gray-400">id: <span className="text-white">{selected.id}</span></p>
                      <p className="text-gray-400">agent: <span className="text-white">{selected.agent}</span></p>
                      <p className="text-gray-400">proof_hash: <span className="text-red-400 break-all">{selected.proof_hash}</span></p>
                      <p className="text-gray-400">input_hash: <span className="text-orange-300 break-all">{selected.input_hash}</span></p>
                      <p className="text-gray-400">output_hash: <span className="text-orange-300 break-all">{selected.output_hash}</span></p>
                      <p className="text-gray-400">model_hash: <span className="text-yellow-300 break-all">{selected.model_hash}</span></p>
                      <p className="text-gray-400">merkle_root: <span className="text-green-400 break-all">{selected.merkle_root}</span></p>
                      <p className="text-gray-400">leaf_index: <span className="text-white">{selected.leaf_index}</span></p>
                      <p className="text-gray-400">valid: <span className={selected.valid ? 'text-green-400' : 'text-red-400'}>{String(selected.valid)}</span></p>
                      <p className="text-gray-400">revoked: <span className="text-white">{String(selected.revoked)}</span></p>
                      <p className="text-gray-400">merkle_path: <span className="text-gray-500">[{selected.merkle_path?.length || 0} nodes]</span></p>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-2 max-h-80 overflow-y-auto">
                    {proofs.length === 0 ? (
                      <p className="text-gray-600 text-sm text-center py-8">No proofs yet. Generate one to get started.</p>
                    ) : proofs.map((p, i) => (
                      <button key={i} onClick={() => setSelected(p)} className="w-full flex items-center justify-between p-3 rounded-lg bg-black/20 hover:bg-black/40 transition-colors text-left">
                        <div className="min-w-0">
                          <p className="font-mono text-xs text-gray-300 truncate">{p.proof_hash}</p>
                          <p className="text-xs text-gray-600">{p.agent} / {p.use_case || 'general'}</p>
                        </div>
                        <div className="flex items-center gap-2 shrink-0 ml-3">
                          {p.valid ? <CheckCircle className="w-4 h-4 text-green-500" /> : <XCircle className="w-4 h-4 text-red-500" />}
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
