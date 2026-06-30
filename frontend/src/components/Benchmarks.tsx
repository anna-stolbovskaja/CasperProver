export default function Benchmarks() {
  const rows = [
    { metric: 'Proof generation time', value: '<50ms', note: 'SHA-256 + Merkle tree' },
    { metric: 'Verification time', value: '<10ms', note: 'Merkle path check' },
    { metric: 'Proof size', value: '~512 bytes', note: 'Hash + path + metadata' },
    { metric: 'GPU required', value: 'None', note: 'Pure cryptographic hashing' },
    { metric: 'False positive rate', value: '0%', note: 'Deterministic verification' },
    { metric: 'On-chain cost', value: '~2.5 CSPR', note: 'Casper testnet deploy' },
  ]

  return (
    <section className="py-24">
      <div className="cp-section">
        <div className="text-center mb-14">
          <p className="text-xs font-mono text-red-500 tracking-widest mb-3">PERFORMANCE</p>
          <h2 className="text-3xl font-extrabold text-white mb-4">
            Numbers that matter.
          </h2>
        </div>

        <div className="max-w-3xl mx-auto bg-cp-card rounded-2xl border border-cp-border overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-cp-border">
                <th className="text-left px-6 py-4 text-xs font-mono text-gray-500 uppercase tracking-wider">Metric</th>
                <th className="text-right px-6 py-4 text-xs font-mono text-gray-500 uppercase tracking-wider">Value</th>
                <th className="text-right px-6 py-4 text-xs font-mono text-gray-500 uppercase tracking-wider hidden sm:table-cell">Note</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => (
                <tr key={i} className="border-b border-cp-border/50 last:border-0 hover:bg-white/[0.02] transition-colors">
                  <td className="px-6 py-4 text-sm text-gray-300">{r.metric}</td>
                  <td className="px-6 py-4 text-sm font-mono text-red-400 text-right font-semibold">{r.value}</td>
                  <td className="px-6 py-4 text-xs text-gray-600 text-right hidden sm:table-cell">{r.note}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
