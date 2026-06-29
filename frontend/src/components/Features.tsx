import { ShieldCheck, Lock, Zap } from 'lucide-react'

const ICON_FEATURES = [
  { icon: ShieldCheck, title: 'Verifiable', desc: 'Cryptographic proofs you can verify on-chain.' },
  { icon: Lock, title: 'Private', desc: 'Zero-knowledge proofs protect sensitive data.' },
  { icon: Zap, title: 'Trustless', desc: 'No intermediaries. Just math and consensus.' },
]

export default function Features() {
  return (
    <section className="py-4 relative" aria-label="Key features">
      <div className="cp-section">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {ICON_FEATURES.map(f => (
            <div key={f.title} className="cp-card flex items-start gap-4">
              <div className="cp-icon-circle">
                <f.icon size={22} className="text-cp-red" />
              </div>
              <div>
                <h3 className="font-semibold text-white mb-1">{f.title}</h3>
                <p className="text-sm text-cp-gray leading-relaxed">{f.desc}</p>
              </div>
            </div>
          ))}
          {/* On Casper — separate to use img */}
          <div className="cp-card flex items-start gap-4">
            <div className="cp-icon-circle">
              <img src="/images/logo.png" alt="" width={24} height={24} className="rounded-full" />
            </div>
            <div>
              <h3 className="font-semibold text-white mb-1">On Casper</h3>
              <p className="text-sm text-cp-gray leading-relaxed">Built natively for speed, security, and scalability.</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
