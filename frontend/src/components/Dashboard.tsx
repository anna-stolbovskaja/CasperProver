import { useState, useRef } from 'react'
import { Play, Check, Loader2, Search, Clock, Hash, ShieldCheck } from 'lucide-react'

interface Proof {
  id: string
  agent: string
  rootHash: string
  status: 'valid' | 'revoked'
  created: string
  deployTx: string
}

interface LogLine { time: string; text: string; type: 'info' | 'success' | 'hash' }

function fakeHash(s: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = Math.imul(h, 0x01000193) }
  return (h >>> 0).toString(16).padStart(8, '0') + ((h ^ 0xdeadbeef) >>> 0).toString(16).padStart(8, '0')
}

const DEMO_PROOFS: Proof[] = [
  { id: 'proof-001', agent: 'kyc-agent-v2', rootHash: '8f4a2b3c9d1e5f70', status: 'valid', created: '2026-06-29T12:00:00Z', deployTx: 'd64299b6' },
  { id: 'proof-002', agent: 'credit-scorer', rootHash: 'a1b2c3d4e5f6a7b8', status: 'valid', created: '2026-06-29T11:30:00Z', deployTx: 'c1320d18' },
  { id: 'proof-003', agent: 'trade-bot-alpha', rootHash: 'deadbeefcafebabe', status: 'revoked', created: '2026-06-28T18:00:00Z', deployTx: '6ed38d8d' },
]

export default function Dashboard() {
  const [proofs, setProofs] = useState<Proof[]>(DEMO_PROOFS)
  const [input, setInput] = useState('{"user": "alice", "score": 85}')
  const [output, setOutput] = useState('{"approved": true}')
  const [model, setModel] = useState('risk-model-v3')
  const [running, setRunning] = useState(false)
  const [log, setLog] = useState<LogLine[]>([])
  const [verifyId, setVerifyId] = useState('')
  const [verifyResult, setVerifyResult] = useState<string | null>(null)
  const logRef = useRef<HTMLDivElement>(null)

  const addLine = (l: LogLine, d: number) =>
    new Promise<void>(r => setTimeout(() => { setLog(p => [...p, l]); logRef.current?.scrollTo({ top: 9999, behavior: 'smooth' }); r() }, d))

  const generate = async () => {
    if (running) return
    setRunning(true); setLog([])
    const ih = fakeHash(input), oh = fakeHash(output), mh = fakeHash(model), root = fakeHash(ih + oh + mh)
    await addLine({ time: '0.000', text: 'Hashing input...', type: 'info' }, 150)
    await addLine({ time: '0.002', text: `→ 0x${ih}`, type: 'hash' }, 300)
    await addLine({ time: '0.003', text: 'Hashing output...', type: 'info' }, 200)
    await addLine({ time: '0.005', text: `→ 0x${oh}`, type: 'hash' }, 300)
    await addLine({ time: '0.006', text: 'Hashing model...', type: 'info' }, 200)
    await addLine({ time: '0.008', text: `→ 0x${mh}`, type: 'hash' }, 300)
    await addLine({ time: '0.010', text: 'Building Merkle tree...', type: 'info' }, 400)
    await addLine({ time: '0.012', text: `Root: 0x${root}`, type: 'success' }, 500)
    await addLine({ time: '0.015', text: 'Submitting to proof-registry...', type: 'info' }, 400)
    const tx = fakeHash(Date.now().toString())
    await addLine({ time: '2.340', text: `Deploy: 0x${tx.slice(0, 16)}`, type: 'hash' }, 1000)
    await addLine({ time: '4.890', text: '✓ Proof anchored on-chain', type: 'success' }, 600)

    const newProof: Proof = {
      id: `proof-${String(proofs.length + 1).padStart(3, '0')}`,
      agent: model,
      rootHash: root.slice(0, 16),
      status: 'valid',
      created: new Date().toISOString(),
      deployTx: tx.slice(0, 8),
    }
    setProofs(p => [newProof, ...p])
    setRunning(false)
  }

  const verify = () => {
    const found = proofs.find(p => p.id === verifyId || p.rootHash.startsWith(verifyId))
    if (found) {
      setVerifyResult(found.status === 'valid' ? `✓ Proof ${found.id} is VALID. Root: 0x${found.rootHash}` : `✗ Proof ${found.id} is REVOKED.`)
    } else {
      setVerifyResult('✗ No matching proof found.')
    }
  }

  return (
    <div className="pt-20 pb-16">
      <div className="cp-section">
        {/* Header */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-bold">CasperProver <span className="cp-gradient-text">Dashboard</span></h1>
            <p className="text-sm text-cp-gray mt-1">Generate, explore, and verify proofs on Casper testnet.</p>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-cp-gray font-mono">casper-test</span>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
          {[
            { label: 'Proofs Generated', value: proofs.length, icon: Hash },
            { label: 'Verification Rate', value: `${Math.round(proofs.filter(p => p.status === 'valid').length / proofs.length * 100)}%`, icon: ShieldCheck },
            { label: 'Avg Proof Time', value: '4.89s', icon: Clock },
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
          {/* Generate Proof */}
          <div className="cp-card !p-5 space-y-3">
            <h3 className="font-semibold text-white flex items-center gap-2"><Play size={16} className="text-cp-red" /> Generate Proof</h3>
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
            <button onClick={generate} disabled={running}
              className="cp-btn-primary w-full justify-center !text-sm disabled:opacity-50 disabled:cursor-not-allowed">
              {running ? <><Loader2 size={14} className="animate-spin" /> Generating...</> : <><Play size={14} /> Generate</>}
            </button>
          </div>

          {/* Terminal */}
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
              {log.length === 0 && <div className="text-cp-gray-dark"><span className="animate-pulse">▌</span> Ready...</div>}
              {log.map((l, i) => (
                <div key={i} className="flex gap-2">
                  <span className="text-cp-gray-dark shrink-0">[{l.time}]</span>
                  <span className={l.type === 'success' ? 'text-green-400' : l.type === 'hash' ? 'text-cp-red/80' : 'text-cp-gray'}>{l.text}</span>
                </div>
              ))}
              {running && <div className="text-cp-gray-dark animate-pulse">▌</div>}
            </div>
          </div>
        </div>

        {/* Verify */}
        <div className="cp-card !p-5 mb-8">
          <h3 className="font-semibold text-white flex items-center gap-2 mb-3"><Search size={16} className="text-cp-red" /> Verify Proof</h3>
          <div className="flex gap-3">
            <input value={verifyId} onChange={e => setVerifyId(e.target.value)} placeholder="Proof ID or root hash..."
              className="flex-1 bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-cp-red/50 focus:outline-none" />
            <button onClick={verify} className="cp-btn-primary !text-sm">Verify</button>
          </div>
          {verifyResult && (
            <div className={`mt-3 text-sm font-mono ${verifyResult.startsWith('✓') ? 'text-green-400' : 'text-red-400'}`}>{verifyResult}</div>
          )}
        </div>

        {/* Proof table */}
        <div className="cp-card !p-0 overflow-hidden">
          <div className="px-5 py-4 border-b border-cp-border">
            <h3 className="font-semibold text-white">Proof Explorer</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-cp-border text-cp-gray text-xs uppercase tracking-wider">
                  <th className="text-left py-3 px-5 font-medium">ID</th>
                  <th className="text-left py-3 px-5 font-medium">Agent</th>
                  <th className="text-left py-3 px-5 font-medium">Root Hash</th>
                  <th className="text-left py-3 px-5 font-medium">Status</th>
                  <th className="text-left py-3 px-5 font-medium">Created</th>
                </tr>
              </thead>
              <tbody>
                {proofs.map(p => (
                  <tr key={p.id} className="border-b border-cp-border/50 hover:bg-cp-red/[0.02] transition-colors">
                    <td className="py-3 px-5 font-mono text-xs text-cp-red">{p.id}</td>
                    <td className="py-3 px-5 text-xs text-cp-gray">{p.agent}</td>
                    <td className="py-3 px-5 font-mono text-xs text-cp-gray">0x{p.rootHash}</td>
                    <td className="py-3 px-5">
                      <span className={`inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full ${
                        p.status === 'valid' ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'}`}>
                        {p.status === 'valid' ? '● Valid' : '● Revoked'}
                      </span>
                    </td>
                    <td className="py-3 px-5 text-xs text-cp-gray-dark">{new Date(p.created).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
