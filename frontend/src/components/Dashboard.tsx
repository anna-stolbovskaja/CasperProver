import { useState, useRef, useEffect, useCallback } from 'react'
import { Play, Search, Clock, Hash, ShieldCheck, Wallet, LogOut, Loader2, RefreshCw } from 'lucide-react'
import { createProof, getProofs, getHealth } from '../lib/api'
import type { ProofRecord } from '../lib/api'
import { connectWallet, disconnectWallet, shortKey } from '../lib/wallet'
import type { WalletState } from '../lib/wallet'

interface LogLine { time: string; text: string; type: 'info' | 'success' | 'error' | 'hash' }

function hashTrunc(h: string): string {
  return h.length > 20 ? h.slice(0, 12) + '...' : h
}

export default function Dashboard() {
  const [proofs, setProofs] = useState<ProofRecord[]>([])
  const [input, setInput] = useState('{"user": "alice", "score": 85}')
  const [output, setOutput] = useState('{"approved": true}')
  const [model, setModel] = useState('risk-model-v3')
  const [agent, setAgent] = useState('kyc-agent-v2')
  const [running, setRunning] = useState(false)
  const [log, setLog] = useState<LogLine[]>([])
  const [verifyId, setVerifyId] = useState('')
  const [verifyResult, setVerifyResult] = useState<string | null>(null)
  const [wallet, setWallet] = useState<WalletState>({ connected: false, publicKey: null, accountHash: null, simulated: false })
  const [chainStatus, setChainStatus] = useState<string>('connecting...')
  const [loadingProofs, setLoadingProofs] = useState(false)
  const logRef = useRef<HTMLDivElement>(null)

  const addLine = useCallback((l: LogLine) => {
    setLog(p => [...p, l])
    setTimeout(() => logRef.current?.scrollTo({ top: 9999, behavior: 'smooth' }), 50)
  }, [])

  useEffect(() => {
    getHealth()
      .then(h => setChainStatus(h.status === 'ok' ? `v${h.version}` : 'offline'))
      .catch(() => setChainStatus('offline'))
  }, [])

  const loadProofs = async () => {
    setLoadingProofs(true)
    try {
      const list = await getProofs()
      setProofs(list)
    } catch { /* ignore */ }
    setLoadingProofs(false)
  }

  const handleConnect = async () => {
    if (wallet.connected) {
      setWallet(disconnectWallet())
      setProofs([])
      return
    }
    const state = await connectWallet()
    setWallet(state)
    await loadProofs()
  }

  const handleGenerate = async () => {
    if (running || !wallet.connected) return
    setRunning(true)
    setLog([])

    const ts = () => (performance.now() / 1000).toFixed(3)
    addLine({ time: ts(), text: `Agent: ${agent}`, type: 'info' })
    addLine({ time: ts(), text: `Input: ${input.slice(0, 60)}${input.length > 60 ? '...' : ''}`, type: 'info' })
    addLine({ time: ts(), text: `Output: ${output.slice(0, 60)}${output.length > 60 ? '...' : ''}`, type: 'info' })
    addLine({ time: ts(), text: `Model: ${model}`, type: 'info' })
    addLine({ time: ts(), text: 'Sending to proof engine...', type: 'info' })

    try {
      const proof = await createProof({ agent, input, output, model })

      addLine({ time: ts(), text: `Proof ID: ${proof.id}`, type: 'hash' })
      addLine({ time: ts(), text: `Input hash: 0x${hashTrunc(proof.input_hash)}`, type: 'hash' })
      addLine({ time: ts(), text: `Output hash: 0x${hashTrunc(proof.output_hash)}`, type: 'hash' })
      addLine({ time: ts(), text: `Model hash: 0x${hashTrunc(proof.model_hash)}`, type: 'hash' })
      addLine({ time: ts(), text: `Merkle root: 0x${hashTrunc(proof.merkle_root)}`, type: 'success' })
      addLine({ time: ts(), text: `Path depth: ${proof.merkle_path.length}`, type: 'info' })
      addLine({ time: ts(), text: `Valid: ${proof.valid}`, type: proof.valid ? 'success' : 'error' })

      setProofs(p => [proof, ...p])
    } catch (err) {
      addLine({ time: ts(), text: `Error: ${err instanceof Error ? err.message : String(err)}`, type: 'error' })
    }

    setRunning(false)
  }

  const handleVerify = () => {
    const trimmed = verifyId.trim()
    const found = proofs.find(p => p.id === trimmed || p.merkle_root.startsWith(trimmed) || p.proof_hash.startsWith(trimmed))
    if (found) {
      setVerifyResult(
        found.valid && !found.revoked
          ? `VALID | ${found.id} | Root: 0x${hashTrunc(found.merkle_root)} | Agent: ${found.agent}`
          : `REVOKED | ${found.id} | Proof is no longer valid.`
      )
    } else {
      setVerifyResult('No matching proof found.')
    }
  }

  const validCount = proofs.filter(p => p.valid && !p.revoked).length
  const validRate = proofs.length > 0 ? Math.round((validCount / proofs.length) * 100) : 0

  return (
    <div className="pt-20 pb-16">
      <div className="cp-section">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-bold">CasperProver <span className="cp-gradient-text">Dashboard</span></h1>
            <p className="text-sm text-cp-gray mt-1">Generate, explore, and verify proofs on Casper testnet.</p>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 text-xs">
              <span className={`w-2 h-2 rounded-full ${chainStatus === 'offline' ? 'bg-red-500' : 'bg-green-500 animate-pulse'}`} />
              <span className="text-cp-gray font-mono">{chainStatus}</span>
            </div>
            <button onClick={handleConnect}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer ${
                wallet.connected
                  ? 'bg-cp-red/10 text-cp-red border border-cp-red/30 hover:bg-cp-red/20'
                  : 'bg-cp-red text-white hover:bg-cp-red/90'
              }`}>
              {wallet.connected ? (
                <><LogOut size={12} /> {shortKey(wallet.publicKey || '')}{wallet.simulated ? ' (demo)' : ''}</>
              ) : (
                <><Wallet size={12} /> Connect Wallet</>
              )}
            </button>
          </div>
        </div>

        {!wallet.connected && (
          <div className="cp-card !p-6 text-center mb-8">
            <Wallet size={32} className="mx-auto text-cp-red mb-3" />
            <h3 className="text-lg font-semibold text-white mb-2">Connect Your Wallet</h3>
            <p className="text-sm text-cp-gray mb-4">
              Connect a Casper Wallet to generate and verify proofs.
              {' '}No extension? A demo account will be provided.
            </p>
            <button onClick={handleConnect} className="cp-btn-primary mx-auto !text-sm">
              <Wallet size={14} /> Connect Wallet
            </button>
          </div>
        )}

        {wallet.connected && (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
              {[
                { label: 'Proofs Generated', value: proofs.length, icon: Hash },
                { label: 'Verification Rate', value: `${validRate}%`, icon: ShieldCheck },
                { label: 'Latest', value: proofs.length > 0 ? proofs[0].id : 'none', icon: Clock },
              ].map(s => (
                <div key={s.label} className="cp-card flex items-center gap-4">
                  <div className="cp-icon-circle"><s.icon size={20} className="text-cp-red" /></div>
                  <div>
                    <div className="text-2xl font-bold text-white">{s.value}</div>
                    <div className="text-xs text-cp-gray">{s.label}</div>
                  </div>
                </div>
              ))}
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-8">
              <div className="cp-card !p-5 space-y-3">
                <h3 className="font-semibold text-white flex items-center gap-2"><Play size={16} className="text-cp-red" /> Generate Proof</h3>
                <div>
                  <label htmlFor="d-agent" className="text-xs text-cp-gray mb-1 block">Agent</label>
                  <input id="d-agent" value={agent} onChange={e => setAgent(e.target.value)}
                    className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-xs font-mono text-white focus:border-cp-red/50 focus:outline-none" />
                </div>
                <div>
                  <label htmlFor="d-input" className="text-xs text-cp-gray mb-1 block">Input</label>
                  <textarea id="d-input" value={input} onChange={e => setInput(e.target.value)}
                    className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-xs font-mono text-white resize-none focus:border-cp-red/50 focus:outline-none" rows={2} />
                </div>
                <div>
                  <label htmlFor="d-output" className="text-xs text-cp-gray mb-1 block">Output</label>
                  <textarea id="d-output" value={output} onChange={e => setOutput(e.target.value)}
                    className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-xs font-mono text-white resize-none focus:border-cp-red/50 focus:outline-none" rows={2} />
                </div>
                <div>
                  <label htmlFor="d-model" className="text-xs text-cp-gray mb-1 block">Model</label>
                  <input id="d-model" value={model} onChange={e => setModel(e.target.value)}
                    className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-xs font-mono text-white focus:border-cp-red/50 focus:outline-none" />
                </div>
                <button onClick={handleGenerate} disabled={running}
                  className="cp-btn-primary w-full justify-center !text-sm disabled:opacity-50 disabled:cursor-not-allowed">
                  {running ? <><Loader2 size={14} className="animate-spin" /> Generating...</> : <><Play size={14} /> Generate</>}
                </button>
              </div>

              <div className="cp-card !p-0 overflow-hidden flex flex-col">
                <div className="flex items-center gap-2 px-4 py-2 border-b border-cp-border bg-cp-black/50">
                  <div className="flex gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-cp-red/60" />
                    <span className="w-2 h-2 rounded-full bg-yellow-500/60" />
                    <span className="w-2 h-2 rounded-full bg-green-500/60" />
                  </div>
                  <span className="text-[10px] font-mono text-cp-gray-dark">engine output</span>
                </div>
                <div ref={logRef} className="flex-1 p-3 overflow-y-auto font-mono text-[11px] space-y-0.5 min-h-[240px] max-h-[320px] bg-cp-black">
                  {log.length === 0 && <div className="text-cp-gray-dark"><span className="animate-pulse">_</span> Ready...</div>}
                  {log.map((l, i) => (
                    <div key={i} className="flex gap-2">
                      <span className="text-cp-gray-dark shrink-0">[{l.time}]</span>
                      <span className={l.type === 'success' ? 'text-green-400' : l.type === 'error' ? 'text-red-400' : l.type === 'hash' ? 'text-cp-red/80' : 'text-cp-gray'}>{l.text}</span>
                    </div>
                  ))}
                  {running && <div className="text-cp-gray-dark animate-pulse">_</div>}
                </div>
              </div>
            </div>

            <div className="cp-card !p-5 mb-8">
              <h3 className="font-semibold text-white flex items-center gap-2 mb-3"><Search size={16} className="text-cp-red" /> Verify Proof</h3>
              <div className="flex gap-3">
                <input value={verifyId} onChange={e => setVerifyId(e.target.value)} placeholder="Proof ID or root hash..."
                  className="flex-1 bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-cp-red/50 focus:outline-none" />
                <button onClick={handleVerify} className="cp-btn-primary !text-sm">Verify</button>
              </div>
              {verifyResult && (
                <div className={`mt-3 text-sm font-mono ${verifyResult.startsWith('VALID') ? 'text-green-400' : 'text-red-400'}`}>{verifyResult}</div>
              )}
            </div>

            <div className="cp-card !p-0 overflow-hidden">
              <div className="px-5 py-4 border-b border-cp-border flex items-center justify-between">
                <h3 className="font-semibold text-white">Proof Explorer</h3>
                <button onClick={loadProofs} disabled={loadingProofs}
                  className="text-xs text-cp-gray hover:text-white flex items-center gap-1 cursor-pointer disabled:opacity-50">
                  <RefreshCw size={12} className={loadingProofs ? 'animate-spin' : ''} /> Refresh
                </button>
              </div>
              {proofs.length === 0 ? (
                <div className="p-8 text-center text-sm text-cp-gray">No proofs yet. Generate one above to get started.</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-cp-border text-cp-gray text-xs uppercase tracking-wider">
                        <th className="text-left py-3 px-5 font-medium">ID</th>
                        <th className="text-left py-3 px-5 font-medium">Agent</th>
                        <th className="text-left py-3 px-5 font-medium">Merkle Root</th>
                        <th className="text-left py-3 px-5 font-medium">Status</th>
                        <th className="text-left py-3 px-5 font-medium">Created</th>
                      </tr>
                    </thead>
                    <tbody>
                      {proofs.map(p => (
                        <tr key={p.id} className="border-b border-cp-border/50 hover:bg-cp-red/[0.02] transition-colors">
                          <td className="py-3 px-5 font-mono text-xs text-cp-red">{p.id}</td>
                          <td className="py-3 px-5 text-xs text-cp-gray">{p.agent}</td>
                          <td className="py-3 px-5 font-mono text-xs text-cp-gray">0x{hashTrunc(p.merkle_root)}</td>
                          <td className="py-3 px-5">
                            <span className={`inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full ${
                              p.valid && !p.revoked ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'}`}>
                              {p.valid && !p.revoked ? 'Valid' : 'Revoked'}
                            </span>
                          </td>
                          <td className="py-3 px-5 text-xs text-cp-gray-dark">{new Date(p.timestamp * 1000).toLocaleString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
