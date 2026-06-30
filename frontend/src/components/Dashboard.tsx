import { useState, useEffect, useCallback } from 'react'
import {
  Shield, Hash, Wallet, LogOut, Play, Loader2, CheckCircle, XCircle,
  BarChart3, ExternalLink, Search, ChevronLeft, ChevronRight, Filter, Eye, Layers, FileText
} from 'lucide-react'
import { createProof, getProofs, getHealth, getStats, verifyProof, exportProof } from '../lib/api'
import type { ProofRecord, StatsResponse, HealthResponse } from '../lib/api'
import { connectWallet, disconnectWallet, shortKey } from '../lib/wallet'
import type { WalletState } from '../lib/wallet'

const REGISTRY = '96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708'
const EXPLORER = 'https://testnet.cspr.live/contract/'

const MODELS = ['gpt-4o', 'llama-3.1-70b', 'claude-3.5-sonnet', 'mistral-large', 'kyc-model-v2.1', 'lending-llm-v3.0', 'claims-assessor-v1.4']
const USE_CASES = [
  { value: '', label: 'General' },
  { value: 'kyc', label: 'KYC / Identity' },
  { value: 'loan', label: 'Loan Decisioning' },
  { value: 'insurance', label: 'Insurance Claims' },
  { value: 'aml_screening', label: 'AML Screening' },
  { value: 'content_moderation', label: 'Content Moderation' },
  { value: 'medical_screening', label: 'Medical Screening' },
]

const DEMO_SCENARIOS = [
  {
    title: 'KYC / Identity Verification',
    desc: 'Bank verifies customer identity via AI, generates proof, DeFi protocol grants access based on verified proof.',
    agent: 'kyc-verifier-v2',
    input: '{"user_id":"alice_0x3f","doc_type":"passport","country":"DE","issued":"2022-03-15"}',
    output: '{"verified":true,"confidence":0.97,"risk_score":12,"flags":[]}',
    model: 'kyc-model-v2.1',
    useCase: 'kyc',
    before: 'AI says "verified". No proof. Protocol trusts blindly or rejects.',
    after: 'Proof P-X: input/output/model hashed, Merkle root anchored on-chain. Auditable forever.',
  },
  {
    title: 'Loan Approval Audit',
    desc: 'AI approves a loan. Regulator can verify the exact model, inputs, and decision — without accessing raw PII.',
    agent: 'loan-decisioning-bot',
    input: '{"applicant":"bob_0x7a","income":85000,"debt_ratio":0.23,"credit_score":742,"requested":25000}',
    output: '{"decision":"approved","limit":25000,"rate":5.4,"term_months":36}',
    model: 'lending-llm-v3.0',
    useCase: 'loan',
    before: 'AI says "approved". Borrower has no proof of fair treatment. Regulator cannot audit.',
    after: 'Cryptographic proof: exact inputs + model + output. Merkle path verifiable by anyone.',
  },
  {
    title: 'Insurance Claim Assessment',
    desc: 'AI evaluates damage claim. Proof ensures the insurer cannot retroactively change the AI assessment.',
    agent: 'insurance-claims-ai',
    input: '{"claim_id":"CLM-2847","type":"auto","damage_photos":3,"police_report":true,"amount_claimed":12500}',
    output: '{"assessment":"valid","approved_amount":11800,"deductible":500,"payout":11300}',
    model: 'claims-assessor-v1.4',
    useCase: 'insurance',
    before: 'Insurer says AI assessed $8,000. Claimant disputes. No verifiable record.',
    after: 'On-chain proof: AI assessed $11,300 payout. Tamper-evident. Court-admissible.',
  },
]

const STEPS = ['Hashing inputs', 'Building Merkle tree', 'Generating proof', 'Anchoring on-chain', 'Complete']

type Tab = 'overview' | 'generate' | 'proofs' | 'demo' | 'verify'

export default function Dashboard() {
  const [tab, setTab] = useState<Tab>('overview')
  const [proofs, setProofs] = useState<ProofRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [stats, setStats] = useState<StatsResponse | null>(null)
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [wallet, setWallet] = useState<WalletState>({ connected: false, publicKey: null, accountHash: null, simulated: false })
  const [selected, setSelected] = useState<ProofRecord | null>(null)

  // generate form
  const [agent, setAgent] = useState('agent-alpha')
  const [input, setInput] = useState('')
  const [model, setModel] = useState('gpt-4o')
  const [output, setOutput] = useState('')
  const [useCase, setUseCase] = useState('')
  const [mode, setMode] = useState<'local' | 'anchored'>('local')
  const [loading, setLoading] = useState(false)
  const [step, setStep] = useState(-1)
  const [genResult, setGenResult] = useState<ProofRecord | null>(null)

  // verify form
  const [verifyId, setVerifyId] = useState('')
  const [verifyInput, setVerifyInput] = useState('')
  const [verifyOutput, setVerifyOutput] = useState('')
  const [verifyModel, setVerifyModel] = useState('')
  const [verifyResult, setVerifyResult] = useState<Record<string, unknown> | null>(null)
  const [verifying, setVerifying] = useState(false)

  // filter
  const [filterAgent, setFilterAgent] = useState('')
  const [filterMode, setFilterMode] = useState('')
  const LIMIT = 20

  const loadProofs = useCallback(async () => {
    try {
      const r = await getProofs({ page, limit: LIMIT, agent: filterAgent || undefined, mode: filterMode || undefined })
      setProofs(r.proofs || [])
      setTotal(r.total)
    } catch { /* offline */ }
  }, [page, filterAgent, filterMode])

  useEffect(() => { loadProofs() }, [loadProofs])
  useEffect(() => {
    getStats().then(setStats).catch(() => {})
    getHealth().then(setHealth).catch(() => {})
  }, [])

  const handleConnect = async () => {
    if (wallet.connected) { setWallet(disconnectWallet()); return }
    const state = await connectWallet()
    setWallet(state)
  }

  const handleGenerate = async () => {
    if (loading || !agent.trim() || !input.trim() || !output.trim()) return
    setLoading(true)
    setGenResult(null)
    setStep(0)

    const start = performance.now()
    try {
      const proof = await createProof(
        { agent: agent.trim(), input: input.trim(), output: output.trim(), model, use_case: useCase, mode },
        wallet.publicKey || undefined,
      )
      const elapsed = performance.now() - start
      // step through quickly based on real time
      const stepDelay = Math.max(80, elapsed / 4)
      for (let i = 1; i <= 4; i++) {
        await new Promise(r => setTimeout(r, stepDelay))
        setStep(i)
      }
      setGenResult(proof)
      loadProofs()
      getStats().then(setStats).catch(() => {})
    } catch { setStep(-1) }
    setLoading(false)
  }

  const handleVerify = async () => {
    if (verifying || !verifyId.trim()) return
    setVerifying(true)
    setVerifyResult(null)
    try {
      const r = await verifyProof({
        proof_id: verifyId.trim(),
        input: verifyInput.trim() || undefined,
        output: verifyOutput.trim() || undefined,
        model: verifyModel.trim() || undefined,
      })
      setVerifyResult(r as unknown as Record<string, unknown>)
    } catch (e) {
      setVerifyResult({ error: String(e) })
    }
    setVerifying(false)
  }

  const handleDemoGenerate = async (scenario: typeof DEMO_SCENARIOS[0]) => {
    setTab('generate')
    setAgent(scenario.agent)
    setInput(scenario.input)
    setOutput(scenario.output)
    setModel(scenario.model)
    setUseCase(scenario.useCase)
    setMode('anchored')
    setGenResult(null)
    setStep(-1)
  }

  const handleExport = async (id: string) => {
    try {
      const data = await exportProof(id)
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `proof-${id}.json`
      a.click()
    } catch { /* */ }
  }

  const totalPages = Math.ceil(total / LIMIT)

  const NAV_LINKS = [
    { label: 'Home', href: '/#home' },
    { label: 'Features', href: '/#features' },
    { label: 'Pipeline', href: '/#pipeline' },
    { label: 'Demo', href: '/#demo' },
    { label: 'SDK', href: '/#sdk' },
    { label: 'FAQ', href: '/#faq' },
  ]

  return (
    <div className="min-h-screen bg-cp-black">
      {/* Global Navbar handles nav — no duplicate nav here */}
      <div className="cp-section py-8 pt-24">
        {/* Header */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6">
          <div>
            <h1 className="text-2xl font-extrabold text-white">Proof Dashboard</h1>
            <p className="text-gray-500 text-sm mt-1">Generate, verify, and inspect cryptographic proofs for AI decisions</p>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-cp-card border border-cp-border text-xs">
              <span className={`w-2 h-2 rounded-full ${health?.status === 'ok' || wallet.connected ? 'bg-green-500' : 'bg-red-500'}`} />
              <span className="text-gray-400 font-mono">{health?.chain || (wallet.connected ? 'casper-test' : 'offline')}</span>
            </div>
            <button onClick={handleConnect} className={`inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold transition-all ${
              wallet.connected ? 'bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20' : 'bg-red-600 text-white hover:bg-red-500'
            }`}>
              {wallet.connected ? <LogOut className="w-4 h-4" /> : <Wallet className="w-4 h-4" />}
              {wallet.connected ? shortKey(wallet.publicKey || '') : 'Connect Wallet'}
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 mb-6 overflow-x-auto pb-1">
          {([
            ['overview', 'Overview', BarChart3, true],
            ['generate', 'Generate', Shield, false],
            ['proofs', 'Proofs', Layers, true],
            ['demo', 'Use Cases', Eye, true],
            ['verify', 'Verify', Search, false],
          ] as const).map(([key, label, Icon, isDemo]) => (
            <button
              key={key}
              onClick={() => setTab(key as Tab)}
              className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors whitespace-nowrap ${
                tab === key ? 'bg-red-500/10 text-red-400 border border-red-500/20' : 'text-gray-500 hover:text-gray-300 hover:bg-white/5'
              }`}
            >
              <Icon className="w-4 h-4" /> {label}
              {isDemo && <span className="px-1 py-0.5 rounded text-[9px] font-bold tracking-wide bg-amber-500/20 text-amber-400 border border-amber-500/30">DEMO</span>}
            </button>
          ))}
        </div>

        {/* Demo notice */}
        {(['overview', 'proofs', 'demo'] as Tab[]).includes(tab) && (
          <div className="mb-4 px-4 py-3 bg-amber-500/5 border border-amber-500/20 rounded-xl flex items-start gap-3">
            <span className="mt-0.5 px-1.5 py-0.5 rounded text-[9px] font-bold tracking-wide bg-amber-500/20 text-amber-400 border border-amber-500/30 shrink-0">DEMO</span>
            <p className="text-xs text-amber-300/70 leading-relaxed">
              <span className="font-semibold text-amber-300">Testnet Demo Environment.</span> This tab shows pre-seeded proof records to demonstrate CasperProver’s capabilities. These are realistic AI decision proofs — the same cryptographic structure anchored on the Casper testnet in production. Use the Generate and Verify tabs to create and validate real on-chain proofs.
            </p>
          </div>
        )}

        {/* === OVERVIEW TAB === */}
        {tab === 'overview' && (
          <div className="space-y-6">
            {/* Stats grid */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              {[
                { label: 'Total Proofs', value: stats?.total_proofs ?? 0, icon: Layers },
                { label: 'Valid', value: stats?.valid_proofs ?? 0, icon: CheckCircle },
                { label: 'Agents', value: stats?.unique_agents ?? 0, icon: Shield },
                { label: 'Avg Gen', value: `${(stats?.avg_generation_ms ?? 0).toFixed(1)}ms`, icon: BarChart3 },
              ].map((s, i) => (
                <div key={i} className="bg-cp-card rounded-xl border border-cp-border p-4">
                  <s.icon className="w-4 h-4 text-gray-600 mb-2" />
                  <p className="text-xl font-bold text-white">{s.value}</p>
                  <p className="text-xs text-gray-500">{s.label}</p>
                </div>
              ))}
            </div>

            {/* Use-case breakdown */}
            {stats?.use_cases && Object.keys(stats.use_cases).length > 0 && (
              <div className="bg-cp-card rounded-xl border border-cp-border p-5">
                <h3 className="text-white font-bold mb-3 text-sm">Proofs by Use Case</h3>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {Object.entries(stats.use_cases).map(([k, v]) => (
                    <div key={k} className="flex items-center justify-between px-3 py-2 rounded-lg bg-black/30">
                      <span className="text-xs text-gray-400 font-mono">{k}</span>
                      <span className="text-sm font-bold text-white">{v}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Contracts */}
            {health?.contracts && (
              <div className="bg-cp-card rounded-xl border border-cp-border p-5">
                <h3 className="text-white font-bold mb-3 text-sm">Deployed Contracts</h3>
                <div className="space-y-2">
                  {Object.entries(health.contracts).map(([name, hash]) => (
                    <a key={name} href={`${EXPLORER}${hash}`} target="_blank" rel="noreferrer"
                      className="flex items-center justify-between px-3 py-2 rounded-lg bg-black/30 hover:bg-black/50 transition-colors group">
                      <span className="text-xs text-gray-400 font-mono">{name.replace('_', '-')}</span>
                      <span className="flex items-center gap-2">
                        <span className="text-xs text-gray-600 font-mono hidden sm:inline">{hash.slice(0, 12)}...</span>
                        <ExternalLink className="w-3 h-3 text-gray-600 group-hover:text-red-400" />
                      </span>
                    </a>
                  ))}
                </div>
              </div>
            )}

            {/* Recent proofs preview */}
            <div className="bg-cp-card rounded-xl border border-cp-border p-5">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-white font-bold text-sm">Recent Proofs</h3>
                <button onClick={() => setTab('proofs')} className="text-xs text-red-400 hover:text-red-300">View all</button>
              </div>
              <div className="space-y-2">
                {proofs.slice(0, 5).map(p => (
                  <button key={p.id} onClick={() => { setSelected(p); setTab('proofs') }}
                    className="w-full flex items-center justify-between p-3 rounded-lg bg-black/20 hover:bg-black/40 transition-colors text-left">
                    <div className="min-w-0">
                      <p className="font-mono text-xs text-gray-300 truncate">{p.id} — {p.proof_hash.slice(0, 24)}...</p>
                      <p className="text-[10px] text-gray-600">{p.agent} / {p.use_case || 'general'} / {p.generation_ms}ms</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0 ml-3">
                      {p.mode === 'anchored' && <span className="text-[9px] px-1.5 py-0.5 rounded bg-red-500/10 text-red-400 border border-red-500/20">anchored</span>}
                      {p.valid ? <CheckCircle className="w-4 h-4 text-green-500" /> : <XCircle className="w-4 h-4 text-red-500" />}
                    </div>
                  </button>
                ))}
                {proofs.length === 0 && <p className="text-gray-600 text-sm text-center py-4">No proofs yet.</p>}
              </div>
            </div>
          </div>
        )}

        {/* === GENERATE TAB === */}
        {tab === 'generate' && (
          <div className="grid lg:grid-cols-2 gap-6">
            <div className="bg-cp-card rounded-2xl border border-cp-border p-6">
              <h3 className="text-white font-bold mb-4 flex items-center gap-2"><Shield className="w-4 h-4 text-red-400" /> New Proof</h3>

              {/* Mode toggle */}
              <div className="flex gap-2 mb-4">
                {(['local', 'anchored'] as const).map(m => (
                  <button key={m} onClick={() => setMode(m)} className={`px-3 py-1.5 rounded-lg text-xs font-mono transition-colors ${
                    mode === m ? 'bg-red-500/10 text-red-400 border border-red-500/20' : 'text-gray-500 hover:text-gray-300 bg-black/20'
                  }`}>
                    {m === 'local' ? 'Local (fast)' : 'Anchored (on-chain)'}
                  </button>
                ))}
              </div>

              <div className="space-y-3">
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Agent <span className="text-gray-700">alphanumeric, max 64 chars</span></label>
                  <input value={agent} onChange={e => { if (/^[a-zA-Z0-9_-]{0,64}$/.test(e.target.value)) setAgent(e.target.value) }}
                    placeholder="e.g. loan-bot-v2" className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50" />
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Model</label>
                  <select value={model} onChange={e => setModel(e.target.value)}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50">
                    {MODELS.map(m => <option key={m} value={m}>{m}</option>)}
                  </select>
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Use Case</label>
                  <select value={useCase} onChange={e => setUseCase(e.target.value)}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50">
                    {USE_CASES.map(u => <option key={u.value} value={u.value}>{u.label}</option>)}
                  </select>
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Input <span className="text-gray-700">the data sent to the model (max 10KB)</span></label>
                  <textarea value={input} onChange={e => { if (e.target.value.length <= 10240) setInput(e.target.value) }}
                    placeholder='e.g. {"applicant":"bob","income":85000,"credit_score":742}' rows={3}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50 resize-none" />
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Output <span className="text-gray-700">the model's response (max 10KB)</span></label>
                  <textarea value={output} onChange={e => { if (e.target.value.length <= 10240) setOutput(e.target.value) }}
                    placeholder='e.g. {"decision":"approved","limit":25000}' rows={3}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50 resize-none" />
                </div>
                <button onClick={handleGenerate} disabled={loading || !agent.trim() || !input.trim() || !output.trim()}
                  className="w-full mt-2 inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-red-600 text-white text-sm font-semibold rounded-lg hover:bg-red-500 disabled:opacity-40 disabled:cursor-not-allowed transition-all">
                  {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                  {loading ? 'Processing...' : 'Generate Proof'}
                </button>
              </div>
            </div>

            {/* Generation progress + result */}
            <div className="bg-cp-card rounded-2xl border border-cp-border p-6">
              <h3 className="text-white font-bold mb-4 flex items-center gap-2"><Hash className="w-4 h-4 text-red-400" /> Result</h3>

              {step >= 0 && (
                <div className="space-y-2 mb-4">
                  {STEPS.map((s, i) => (
                    <div key={i} className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-all duration-300 ${
                      i < step ? 'bg-green-500/5' : i === step ? 'bg-red-500/5' : 'opacity-30'
                    }`}>
                      {i < step ? (
                        <CheckCircle className="w-4 h-4 text-green-500 shrink-0" />
                      ) : i === step ? (
                        <Loader2 className="w-4 h-4 text-red-400 animate-spin shrink-0" />
                      ) : (
                        <div className="w-4 h-4 rounded-full border border-gray-700 shrink-0" />
                      )}
                      <span className={`text-sm font-mono ${i <= step ? 'text-gray-300' : 'text-gray-600'}`}>{s}</span>
                      {i < step && genResult && i === 4 && (
                        <span className="ml-auto text-[10px] text-gray-500 font-mono">{genResult.generation_ms}ms</span>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {genResult ? (
                <div className="space-y-2">
                  <div className="bg-black/40 rounded-xl p-4 font-mono text-xs space-y-1.5 overflow-x-auto">
                    <p className="text-gray-400">id: <span className="text-white">{genResult.id}</span></p>
                    <p className="text-gray-400">agent: <span className="text-white">{genResult.agent}</span></p>
                    <p className="text-gray-400">proof_hash: <span className="text-red-400 break-all">{genResult.proof_hash}</span></p>
                    <p className="text-gray-400">merkle_root: <span className="text-green-400 break-all">{genResult.merkle_root}</span></p>
                    <p className="text-gray-400">mode: <span className="text-white">{genResult.mode}</span></p>
                    <p className="text-gray-400">generation_ms: <span className="text-yellow-300">{genResult.generation_ms}</span></p>
                    {genResult.deploy_hash && (
                      <p className="text-gray-400">deploy_hash: <span className="text-orange-300 break-all">{genResult.deploy_hash}</span></p>
                    )}
                    <p className="text-gray-400">valid: <span className="text-green-400">{String(genResult.valid)}</span></p>
                  </div>
                  <div className="flex gap-2">
                    <a href={`${EXPLORER}${REGISTRY}`} target="_blank" rel="noreferrer"
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-500/10 text-red-400 text-xs font-mono hover:bg-red-500/20 transition-colors">
                      <ExternalLink className="w-3 h-3" /> View on Casper
                    </a>
                    <button onClick={() => handleExport(genResult.id)}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 text-xs font-mono hover:bg-white/10 transition-colors">
                      <FileText className="w-3 h-3" /> Export JSON
                    </button>
                  </div>
                </div>
              ) : step < 0 ? (
                <div className="text-center py-12">
                  <Shield className="w-10 h-10 text-gray-700 mx-auto mb-3" />
                  <p className="text-gray-500 text-sm">Fill the form and click Generate.</p>
                  <p className="text-gray-600 text-xs mt-1">Or try a use-case from the Demo tab.</p>
                </div>
              ) : null}
            </div>
          </div>
        )}

        {/* === PROOFS TAB === */}
        {tab === 'proofs' && (
          <div className="space-y-4">
            {/* Filters */}
            <div className="flex flex-wrap gap-3 items-center">
              <div className="flex items-center gap-2">
                <Filter className="w-4 h-4 text-gray-500" />
                <input value={filterAgent} onChange={e => { setFilterAgent(e.target.value); setPage(1) }}
                  placeholder="Filter by agent..." className="px-3 py-1.5 bg-cp-card border border-cp-border rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50 w-48" />
              </div>
              <select value={filterMode} onChange={e => { setFilterMode(e.target.value); setPage(1) }}
                className="px-3 py-1.5 bg-cp-card border border-cp-border rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50">
                <option value="">All modes</option>
                <option value="local">Local</option>
                <option value="anchored">Anchored</option>
              </select>
              <span className="text-xs text-gray-500 ml-auto">{total} proofs total</span>
            </div>

            {selected ? (
              <div className="bg-cp-card rounded-2xl border border-cp-border p-6">
                <button onClick={() => setSelected(null)} className="text-xs text-gray-500 hover:text-gray-300 mb-3 flex items-center gap-1">
                  <ChevronLeft className="w-3 h-3" /> Back to list
                </button>
                <div className="bg-black/40 rounded-xl p-5 font-mono text-xs space-y-1.5 overflow-x-auto">
                  {Object.entries(selected).map(([k, v]) => {
                    if (k === 'merkle_path') return <p key={k} className="text-gray-400">{k}: <span className="text-gray-500">[{(v as string[])?.length || 0} nodes]</span></p>
                    const color = k.includes('hash') ? 'text-red-400' : k === 'merkle_root' ? 'text-green-400' : k === 'valid' ? (v ? 'text-green-400' : 'text-red-400') : 'text-white'
                    return <p key={k} className="text-gray-400">{k}: <span className={`${color} break-all`}>{String(v)}</span></p>
                  })}
                </div>
                <div className="flex gap-2 mt-4">
                  <a href={`${EXPLORER}${selected.deploy_hash || REGISTRY}`} target="_blank" rel="noreferrer"
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-500/10 text-red-400 text-xs font-mono hover:bg-red-500/20 transition-colors">
                    <ExternalLink className="w-3 h-3" /> Explorer
                  </a>
                  <button onClick={() => handleExport(selected.id)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 text-xs font-mono hover:bg-white/10 transition-colors">
                    <FileText className="w-3 h-3" /> Export
                  </button>
                </div>
              </div>
            ) : (
              <>
                <div className="space-y-2">
                  {proofs.map(p => (
                    <button key={p.id} onClick={() => setSelected(p)}
                      className="w-full flex items-center justify-between p-3 rounded-lg bg-cp-card border border-cp-border hover:border-red-500/20 transition-colors text-left">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-xs text-white">{p.id}</span>
                          {p.mode === 'anchored' && <span className="text-[9px] px-1.5 py-0.5 rounded bg-red-500/10 text-red-400 border border-red-500/20">anchored</span>}
                          {p.use_case && <span className="text-[9px] px-1.5 py-0.5 rounded bg-white/5 text-gray-500">{p.use_case}</span>}
                        </div>
                        <p className="font-mono text-[10px] text-gray-600 truncate mt-0.5">{p.proof_hash}</p>
                        <p className="text-[10px] text-gray-600">{p.agent} / {p.generation_ms}ms / {new Date(p.timestamp * 1000).toLocaleString()}</p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0 ml-3">
                        {p.valid ? <CheckCircle className="w-4 h-4 text-green-500" /> : <XCircle className="w-4 h-4 text-red-500" />}
                      </div>
                    </button>
                  ))}
                  {proofs.length === 0 && <p className="text-gray-600 text-sm text-center py-8">No proofs match your filters.</p>}
                </div>
                {totalPages > 1 && (
                  <div className="flex items-center justify-center gap-2 pt-2">
                    <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}
                      className="p-2 rounded-lg bg-cp-card border border-cp-border text-gray-400 disabled:opacity-30 hover:text-white transition-colors">
                      <ChevronLeft className="w-4 h-4" />
                    </button>
                    <span className="text-xs text-gray-500 font-mono px-3">{page} / {totalPages}</span>
                    <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages}
                      className="p-2 rounded-lg bg-cp-card border border-cp-border text-gray-400 disabled:opacity-30 hover:text-white transition-colors">
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        )}

        {/* === DEMO / USE CASES TAB === */}
        {tab === 'demo' && (
          <div className="space-y-6">
            <div className="bg-cp-card rounded-xl border border-cp-border p-5">
              <h3 className="text-white font-bold mb-2">Why CasperProver?</h3>
              <p className="text-gray-400 text-sm leading-relaxed">
                AI agents make high-stakes decisions — approving loans, verifying identities, assessing insurance claims. Today, these decisions are black boxes. CasperProver creates tamper-evident, on-chain verifiable proofs that bind an AI's input, model, and output together. Anyone can verify. Nobody can alter.
              </p>
            </div>

            {DEMO_SCENARIOS.map((s, i) => (
              <div key={i} className="bg-cp-card rounded-2xl border border-cp-border p-6">
                <div className="flex items-start justify-between gap-4 mb-4">
                  <div>
                    <h3 className="text-white font-bold text-lg">{s.title}</h3>
                    <p className="text-gray-400 text-sm mt-1">{s.desc}</p>
                  </div>
                  <button onClick={() => handleDemoGenerate(s)}
                    className="shrink-0 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-semibold hover:bg-red-500 transition-colors">
                    <Play className="w-3 h-3" /> Try it
                  </button>
                </div>

                <div className="grid sm:grid-cols-2 gap-4">
                  <div className="rounded-xl bg-black/30 border border-gray-800 p-4">
                    <p className="text-[10px] font-mono text-red-500 tracking-widest mb-2">WITHOUT PROOF</p>
                    <p className="text-sm text-gray-400">{s.before}</p>
                  </div>
                  <div className="rounded-xl bg-green-500/5 border border-green-500/15 p-4">
                    <p className="text-[10px] font-mono text-green-500 tracking-widest mb-2">WITH CASPERPROVER</p>
                    <p className="text-sm text-gray-300">{s.after}</p>
                  </div>
                </div>

                <div className="mt-3 bg-black/20 rounded-lg p-3 font-mono text-[10px] text-gray-600 overflow-x-auto">
                  <span className="text-gray-500">input:</span> {s.input.slice(0, 80)}...
                </div>
              </div>
            ))}
          </div>
        )}

        {/* === VERIFY TAB === */}
        {tab === 'verify' && (
          <div className="max-w-2xl mx-auto">
            <div className="bg-cp-card rounded-2xl border border-cp-border p-6">
              <h3 className="text-white font-bold mb-4 flex items-center gap-2"><Search className="w-4 h-4 text-red-400" /> Verify a Proof</h3>
              <p className="text-gray-500 text-sm mb-4">Enter a proof ID to check its status. Optionally provide the original input, output, and model to perform full cryptographic verification.</p>
              <div className="space-y-3">
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Proof ID <span className="text-red-400">*</span></label>
                  <input value={verifyId} onChange={e => setVerifyId(e.target.value)} placeholder="e.g. P-1"
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50" />
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Original Input <span className="text-gray-700">(optional, for full verification)</span></label>
                  <textarea value={verifyInput} onChange={e => setVerifyInput(e.target.value)} rows={2}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50 resize-none" />
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Original Output <span className="text-gray-700">(optional)</span></label>
                  <textarea value={verifyOutput} onChange={e => setVerifyOutput(e.target.value)} rows={2}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50 resize-none" />
                </div>
                <div>
                  <label className="text-xs text-gray-600 font-mono mb-1 block">Model <span className="text-gray-700">(optional)</span></label>
                  <input value={verifyModel} onChange={e => setVerifyModel(e.target.value)}
                    className="w-full px-3 py-2 bg-black/40 border border-gray-800 rounded-lg text-sm text-white font-mono focus:outline-none focus:border-red-500/50" />
                </div>
                <button onClick={handleVerify} disabled={verifying || !verifyId.trim()}
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-red-600 text-white text-sm font-semibold rounded-lg hover:bg-red-500 disabled:opacity-40 transition-all">
                  {verifying ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
                  {verifying ? 'Verifying...' : 'Verify'}
                </button>
              </div>

              {verifyResult && (
                <div className="mt-4 bg-black/40 rounded-xl p-4 font-mono text-xs space-y-1.5">
                  {Object.entries(verifyResult).map(([k, v]) => {
                    if (typeof v === 'object' && v !== null) {
                      return (
                        <div key={k}>
                          <p className="text-gray-400 mb-1">{k}:</p>
                          {Object.entries(v as Record<string, boolean>).map(([ck, cv]) => (
                            <p key={ck} className="text-gray-400 pl-4">{ck}: <span className={cv ? 'text-green-400' : 'text-red-400'}>{String(cv)}</span></p>
                          ))}
                        </div>
                      )
                    }
                    const color = k === 'verified' ? (v ? 'text-green-400' : 'text-red-400') : k === 'valid' ? (v ? 'text-green-400' : 'text-red-400') : k === 'error' ? 'text-red-400' : 'text-white'
                    return <p key={k} className="text-gray-400">{k}: <span className={color}>{String(v)}</span></p>
                  })}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
