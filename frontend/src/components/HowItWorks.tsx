import { Brain, Layers, ShieldCheck, PackageOpen } from 'lucide-react'

const STEPS = [
  {
    n: 1,
    icon: Brain,
    title: 'AI Agent',
    subtitle: 'Makes Decision',
    desc: 'AI processes input and generates output.',
  },
  {
    n: 2,
    icon: Layers,
    title: 'CasperProver',
    subtitle: 'Captures I/O',
    desc: 'Captures what was received and returned.',
  },
  {
    n: 3,
    icon: ShieldCheck,
    title: 'Proof',
    subtitle: 'Generated',
    desc: 'Creates a Merkle-anchored proof of correctness.',
  },
  {
    n: 4,
    icon: PackageOpen,
    title: 'On-Chain',
    subtitle: 'Verification',
    desc: 'Anyone can verify the proof without seeing the data.',
  },
]

export default function HowItWorks() {
  return (
    <section id="how-it-works" className="py-16 sm:py-24">
      <div className="cp-section">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Left: How It Works */}
          <div className="cp-card !p-8">
            <h2 className="text-2xl sm:text-3xl font-bold mb-8">
              How <span className="cp-gradient-text">CasperProver</span> Works
            </h2>

            {/* Steps */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-8">
              {STEPS.map((s, i) => (
                <div key={s.n} className="text-center relative">
                  {/* Connector line */}
                  {i < 3 && (
                    <div className="hidden sm:block absolute top-5 left-[60%] w-[80%] border-t border-dashed border-cp-red/30" aria-hidden="true" />
                  )}
                  {/* Number badge */}
                  <div className="w-10 h-10 mx-auto rounded-full border-2 border-cp-red/40 flex items-center justify-center text-cp-red font-bold text-sm mb-3 relative z-10 bg-cp-card">
                    {s.n}
                  </div>
                  <div className="cp-icon-circle mx-auto mb-2">
                    <s.icon size={20} className="text-cp-red" />
                  </div>
                  <h4 className="font-semibold text-sm text-white">{s.title}</h4>
                  <p className="text-xs text-cp-gray mt-0.5">{s.subtitle}</p>
                </div>
              ))}
            </div>

            {/* Descriptions */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              {STEPS.map(s => (
                <p key={s.n} className="text-xs text-cp-gray-dark leading-relaxed text-center">{s.desc}</p>
              ))}
            </div>

            <p className="text-sm text-center mt-6">
              <span className="cp-gradient-text font-semibold">Privacy</span>
              <span className="text-cp-gray"> by design. Verifiability by default.</span>
            </p>
          </div>

          {/* Right: Built for Real-World AI */}
          <div className="cp-card !p-8 relative overflow-hidden">
            <h2 className="text-2xl sm:text-3xl font-bold mb-8">
              Built for Real-World <span className="cp-gradient-text">AI</span>
            </h2>

            <div className="space-y-5">
              {[
                { icon: '🔐', title: 'KYC & Identity', desc: 'Private identity verification' },
                { icon: '📊', title: 'Credit Scoring', desc: 'Verifiable credit assessments' },
                { icon: '📈', title: 'Trading Agents', desc: 'Provable trading decisions' },
                { icon: '🛡️', title: 'Risk Management', desc: 'Audit-ready AI operations' },
              ].map(u => (
                <div key={u.title} className="flex items-center gap-4 group">
                  <div className="cp-icon-circle group-hover:border-cp-red/40 transition-colors duration-200">
                    <span className="text-lg" role="img" aria-label={u.title}>{u.icon}</span>
                  </div>
                  <div>
                    <h4 className="font-semibold text-white">{u.title}</h4>
                    <p className="text-sm text-cp-gray">{u.desc}</p>
                  </div>
                </div>
              ))}
            </div>

            {/* Decorative mascot */}
            <img
              src="/images/mascot-blue-t.png"
              alt=""
              className="absolute -bottom-8 -right-8 w-48 opacity-10 pointer-events-none select-none"
              aria-hidden="true"
              loading="lazy"
            />

            {/* Decorative chart line */}
            <div className="absolute bottom-6 right-6 opacity-20" aria-hidden="true">
              <svg width="120" height="60" viewBox="0 0 120 60" fill="none">
                <polyline points="0,50 20,40 40,45 60,25 80,30 100,10 120,15" stroke="#E53935" strokeWidth="2" fill="none" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
