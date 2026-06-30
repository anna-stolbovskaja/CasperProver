import { useEffect, useRef, useState } from 'react'

const steps = [
  { num: 1, title: 'AI Agent', sub: 'Makes Decision', desc: 'AI processes input and generates output.', icon: '🧠' },
  { num: 2, title: 'CasperProver', sub: 'Captures I/O', desc: 'Captures what was received and returned.', icon: '📦' },
  { num: 3, title: 'Proof', sub: 'Generated', desc: 'Creates a zero-knowledge proof of correctness.', icon: '✓' },
  { num: 4, title: 'On-Chain', sub: 'Verification', desc: 'Anyone can verify the proof without seeing the data.', icon: '🔗' },
]

export default function HowItWorks() {
  const ref = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const obs = new IntersectionObserver(([e]) => { if (e.isIntersecting) setVisible(true) }, { threshold: 0.15 })
    if (ref.current) obs.observe(ref.current)
    return () => obs.disconnect()
  }, [])

  return (
    <section id="how-it-works" className="py-24 relative overflow-hidden" ref={ref}>
      <div className="absolute inset-0 bg-gradient-to-b from-cp-black via-[#0a0a0a] to-cp-black" />

      <div className="cp-section relative z-10">
        <div className="grid lg:grid-cols-2 gap-12">
          {/* left: steps */}
          <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-8">
            <h2 className="text-3xl font-bold mb-2">
              How <span className="text-cp-red">CasperProver</span> Works
            </h2>
            <p className="text-sm text-gray-500 mb-10">Privacy by design. Verifiability by default.</p>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              {steps.map((s, i) => (
                <div
                  key={i}
                  className={`text-center transition-all duration-700 ${visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-6'}`}
                  style={{ transitionDelay: `${i * 120}ms` }}
                >
                  {/* Red circle with number */}
                  <div className="mx-auto w-12 h-12 rounded-full border-2 border-cp-red/40 flex items-center justify-center mb-3 relative">
                    <span className="text-cp-red font-bold">{s.num}</span>
                    {i < 3 && (
                      <div className="hidden sm:block absolute left-full top-1/2 w-[calc(100%-8px)] h-px border-t border-dashed border-cp-red/20" style={{ width: '60px' }} />
                    )}
                  </div>
                  <span className="text-xl mb-1 block">{s.icon}</span>
                  <h3 className="text-sm font-semibold text-white">{s.title}</h3>
                  <p className="text-[10px] text-cp-red">{s.sub}</p>
                  <p className="text-[11px] text-gray-500 mt-1">{s.desc}</p>
                </div>
              ))}
            </div>
          </div>

          {/* right: use cases */}
          <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-8">
            <h2 className="text-3xl font-bold mb-8">
              Built for Real-World <span className="text-cp-red">AI</span>
            </h2>

            <div className="space-y-4">
              {[
                { title: 'KYC & Identity', desc: 'Private identity verification', icon: '👤' },
                { title: 'Credit Scoring', desc: 'Verifiable credit assessments', icon: '📊' },
                { title: 'Trading Agents', desc: 'Provable trading decisions', icon: '📈' },
                { title: 'Risk Management', desc: 'Audit-ready AI operations', icon: '🛡️' },
              ].map((uc, i) => (
                <div
                  key={i}
                  className={`flex items-center gap-4 p-3 rounded-xl hover:bg-white/[0.03] transition-all duration-700 ${visible ? 'opacity-100 translate-x-0' : 'opacity-0 translate-x-6'}`}
                  style={{ transitionDelay: `${i * 100 + 300}ms` }}
                >
                  <div className="w-10 h-10 rounded-lg bg-cp-red/10 flex items-center justify-center text-lg shrink-0">
                    {uc.icon}
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-white">{uc.title}</h3>
                    <p className="text-xs text-gray-500">{uc.desc}</p>
                  </div>
                </div>
              ))}
            </div>

            {/* mascot pose */}
            <div className="mt-6 flex justify-end">
              <img src="/images/mascot/pose3.webp" alt="" className="w-28 opacity-60 drop-shadow-[0_0_20px_rgba(255,50,50,0.2)]" loading="lazy" />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
