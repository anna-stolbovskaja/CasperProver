import { useState, useRef } from 'react'
import { Play, Check, Loader2 } from 'lucide-react'

interface LogLine {
  time: string
  text: string
  type: 'info' | 'success' | 'hash'
}

// Deterministic SHA-256-like hash (visual demo — actual proof uses Go engine)
function fakeHash(input: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  const hex = (n: number) => (n >>> 0).toString(16).padStart(8, '0')
  return '0x' + hex(h) + hex(h ^ 0xdeadbeef) + hex(h ^ 0xcafebabe) + '...'
}

export default function LiveDemo() {
  const [input, setInput] = useState('{"kyc_id": "user_7291", "country": "DE", "age": 32}')
  const [output, setOutput] = useState('{"risk_score": 42, "status": "approved"}')
  const [model, setModel] = useState('gpt-4-kyc-v2')
  const [running, setRunning] = useState(false)
  const [log, setLog] = useState<LogLine[]>([])
  const [done, setDone] = useState(false)
  const logRef = useRef<HTMLDivElement>(null)

  const addLine = (line: LogLine, delay: number) =>
    new Promise<void>(resolve => {
      setTimeout(() => {
        setLog(prev => [...prev, line])
        logRef.current?.scrollTo({ top: logRef.current.scrollHeight, behavior: 'smooth' })
        resolve()
      }, delay)
    })

  const runDemo = async () => {
    if (running) return
    setRunning(true)
    setDone(false)
    setLog([])

    const inputHash = fakeHash(input)
    const outputHash = fakeHash(output)
    const modelHash = fakeHash(model)
    const rootHash = fakeHash(inputHash + outputHash + modelHash)

    await addLine({ time: '00:00.000', text: 'Hashing input...', type: 'info' }, 200)
    await addLine({ time: '00:00.002', text: `→ ${inputHash}`, type: 'hash' }, 400)
    await addLine({ time: '00:00.003', text: 'Hashing output...', type: 'info' }, 300)
    await addLine({ time: '00:00.005', text: `→ ${outputHash}`, type: 'hash' }, 400)
    await addLine({ time: '00:00.006', text: 'Hashing model identifier...', type: 'info' }, 300)
    await addLine({ time: '00:00.007', text: `→ ${modelHash}`, type: 'hash' }, 400)
    await addLine({ time: '00:00.010', text: 'Building Merkle tree (3 leaves)...', type: 'info' }, 500)
    await addLine({ time: '00:00.012', text: `Root computed: ${rootHash}`, type: 'success' }, 600)
    await addLine({ time: '00:00.015', text: 'Submitting to proof-registry...', type: 'info' }, 500)
    await addLine({ time: '00:02.340', text: `Deploy hash: 0xd642...${Math.random().toString(16).slice(2, 6)}`, type: 'hash' }, 1200)
    await addLine({ time: '00:04.890', text: '✓ Proof stored on-chain', type: 'success' }, 800)
    await addLine({ time: '00:04.895', text: 'Verifying via verifier-gate...', type: 'info' }, 400)
    await addLine({ time: '00:05.120', text: '✓ Verification passed', type: 'success' }, 600)

    setRunning(false)
    setDone(true)
  }

  return (
    <section id="live-demo" className="py-16 sm:py-24">
      <div className="cp-section">
        <div className="text-center mb-10">
          <h2 className="text-2xl sm:text-3xl font-bold mb-3">
            Try It <span className="cp-gradient-text">Live</span>
          </h2>
          <p className="text-cp-gray max-w-md mx-auto">
            Generate a cryptographic proof right now. This demo simulates the CasperProver engine flow.
          </p>
        </div>

        <div className="max-w-3xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Input form */}
          <div className="cp-card !p-5 space-y-4">
            <div>
              <label htmlFor="demo-input" className="block text-xs font-medium text-cp-gray mb-1.5">Computation Input</label>
              <textarea id="demo-input" value={input} onChange={e => setInput(e.target.value)}
                className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-sm font-mono text-white resize-none focus:border-cp-red/50 focus:outline-none transition-colors"
                rows={3} />
            </div>
            <div>
              <label htmlFor="demo-output" className="block text-xs font-medium text-cp-gray mb-1.5">Computation Output</label>
              <textarea id="demo-output" value={output} onChange={e => setOutput(e.target.value)}
                className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-sm font-mono text-white resize-none focus:border-cp-red/50 focus:outline-none transition-colors"
                rows={2} />
            </div>
            <div>
              <label htmlFor="demo-model" className="block text-xs font-medium text-cp-gray mb-1.5">Model Identifier</label>
              <input id="demo-model" value={model} onChange={e => setModel(e.target.value)}
                className="w-full bg-cp-black border border-cp-border rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-cp-red/50 focus:outline-none transition-colors" />
            </div>
            <button onClick={runDemo} disabled={running}
              className="cp-btn-primary w-full justify-center disabled:opacity-50 disabled:cursor-not-allowed">
              {running ? <><Loader2 size={16} className="animate-spin" /> Generating...</> :
               done ? <><Check size={16} /> Run Again</> :
               <><Play size={16} /> Generate Proof</>}
            </button>
          </div>

          {/* Terminal output */}
          <div className="cp-card !p-0 overflow-hidden flex flex-col">
            <div className="flex items-center gap-2 px-4 py-2.5 border-b border-cp-border bg-cp-black/50">
              <div className="flex gap-1.5">
                <span className="w-2.5 h-2.5 rounded-full bg-cp-red/60" />
                <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/60" />
                <span className="w-2.5 h-2.5 rounded-full bg-green-500/60" />
              </div>
              <span className="text-[10px] font-mono text-cp-gray-dark">casperprover engine</span>
            </div>
            <div ref={logRef} className="flex-1 p-4 overflow-y-auto font-mono text-xs space-y-1 min-h-[200px] max-h-[320px] bg-cp-black">
              {log.length === 0 && (
                <div className="text-cp-gray-dark flex items-center gap-1">
                  <span className="animate-pulse">▌</span>
                  Waiting for input...
                </div>
              )}
              {log.map((l, i) => (
                <div key={i} className="flex gap-2">
                  <span className="text-cp-gray-dark shrink-0">[{l.time}]</span>
                  <span className={
                    l.type === 'success' ? 'text-green-400' :
                    l.type === 'hash' ? 'text-cp-red/80' :
                    'text-cp-gray'
                  }>{l.text}</span>
                </div>
              ))}
              {running && (
                <div className="text-cp-gray-dark animate-pulse">▌</div>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
