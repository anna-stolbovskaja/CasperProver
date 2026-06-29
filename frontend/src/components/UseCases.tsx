export default function UseCases() {
  return (
    <section id="use-cases" className="py-16 sm:py-24 relative">
      {/* Subtle mascot background decoration */}
      <img src="/images/mascot-blue2-t.png" alt="" aria-hidden="true"
        className="absolute top-0 left-0 w-64 opacity-[0.04] pointer-events-none select-none -translate-x-1/3" loading="lazy" />

      <div className="cp-section">
        <div className="text-center mb-12">
          <h2 className="text-2xl sm:text-3xl font-bold mb-3">
            From Inference to <span className="cp-gradient-text">On-Chain Proof</span>
          </h2>
          <p className="text-cp-gray max-w-lg mx-auto">
            Four on-chain calls. Full audit trail. Zero data exposure.
          </p>
        </div>

        {/* Pipeline visualization */}
        <div className="relative max-w-4xl mx-auto">
          {/* Connector line */}
          <div className="hidden md:block absolute top-1/2 left-[10%] right-[10%] h-px bg-gradient-to-r from-cp-red/50 via-cp-red/30 to-cp-red/50" aria-hidden="true" />

          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 md:gap-6">
            {[
              {
                step: 'Compute',
                contract: null,
                desc: 'AI model runs off-chain. Input/output captured by CasperProver engine.',
                code: 'f(x) = y',
              },
              {
                step: 'Prove',
                contract: null,
                desc: 'SHA-256 hashes of I/O + model built into Merkle tree. Root computed.',
                code: 'root: 0x8f4a...',
              },
              {
                step: 'Anchor',
                contract: 'proof-registry',
                desc: 'Merkle root + metadata submitted to proof-registry contract on Casper.',
                code: 'hash-96e97c...',
              },
              {
                step: 'Gate',
                contract: 'verifier-gate',
                desc: 'DeFi contract queries verifier-gate, checks proof, grants access.',
                code: '✓ verified',
              },
            ].map((s, i) => (
              <div key={s.step} className="cp-card text-center relative z-10">
                <div className="w-8 h-8 mx-auto rounded-full bg-cp-red/20 border border-cp-red/40 flex items-center justify-center text-cp-red font-bold text-sm mb-3">
                  {i + 1}
                </div>
                <h4 className="font-bold text-white mb-1">{s.step}</h4>
                {s.contract && (
                  <span className="inline-block text-[10px] font-mono text-cp-red bg-cp-red/10 px-2 py-0.5 rounded-full mb-2">{s.contract}</span>
                )}
                <p className="text-xs text-cp-gray leading-relaxed mb-3">{s.desc}</p>
                <code className="block text-xs font-mono text-cp-red/80 bg-cp-black/50 rounded px-2 py-1">{s.code}</code>
              </div>
            ))}
          </div>
        </div>

        {/* Privacy note */}
        <p className="text-center text-sm text-cp-gray mt-8">
          The DeFi protocol never sees the KYC data — only that a verified proof exists.
        </p>
      </div>
    </section>
  )
}
