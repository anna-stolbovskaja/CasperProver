import { ArrowRight, FileText } from 'lucide-react'
import { useEffect, useRef } from 'react'

export default function Hero() {
  const mascotRef = useRef<HTMLImageElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (!mascotRef.current) return
      const x = (e.clientX / window.innerWidth - 0.5) * 8
      const y = (e.clientY / window.innerHeight - 0.5) * 8
      mascotRef.current.style.transform = `translate(${x}px, ${y}px)`
    }
    window.addEventListener('mousemove', handler)
    return () => window.removeEventListener('mousemove', handler)
  }, [])

  return (
    <section id="home" className="relative min-h-screen flex items-center overflow-hidden pt-16">
      {/* Dark grid bg */}
      <div className="absolute inset-0 bg-cp-black" />
      <div className="absolute inset-0 opacity-[0.06]" aria-hidden="true"
        style={{ backgroundImage: 'radial-gradient(circle at 1px 1px, rgba(255,50,50,0.3) 1px, transparent 0)', backgroundSize: '40px 40px' }} />

      {/* Red glow */}
      <div className="absolute top-1/4 right-1/3 w-[500px] h-[500px] bg-red-600/8 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-1/4 left-1/4 w-[300px] h-[300px] bg-red-700/5 rounded-full blur-[100px] pointer-events-none" />

      <div className="cp-section relative z-10 grid lg:grid-cols-2 gap-8 items-center py-20">
        {/* Left: Text */}
        <div className="flex flex-col items-start">
          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold leading-[1.1] mb-6">
            Provable AI.<br />
            <span className="text-cp-red">On Casper.</span>
          </h1>

          <p className="text-base sm:text-lg text-gray-400 mb-8 max-w-lg leading-relaxed">
            CasperProver creates cryptographic proofs of AI agent decisions &mdash; verifiable, private, and trustless. No black boxes. Just proof.
          </p>

          <div className="flex flex-wrap gap-4 mb-10">
            <a href="/app" className="group inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-cp-red text-white font-semibold shadow-lg shadow-cp-red/20 hover:shadow-cp-red/40 hover:scale-[1.02] transition-all">
              Start Building <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
            </a>
            <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-white/10 text-white/90 font-medium hover:bg-white/5 transition-colors">
              <FileText className="w-4 h-4" /> Read Docs
            </a>
          </div>

          <div className="flex items-center gap-3 text-sm text-gray-500">
            <span>Built on</span>
            <img src="/images/casper-logo.png" alt="Casper" className="w-5 h-5" />
            <img src="/images/casper-wordmark-white.png" alt="Casper Network" className="h-4 opacity-50" />
          </div>
        </div>

        {/* Right: Mascot with floating labels */}
        <div className="relative flex justify-center lg:justify-end">
          {/* Red aura behind mascot */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div className="w-64 h-64 sm:w-80 sm:h-80 rounded-full bg-cp-red/15 blur-[60px] animate-pulse-slow" />
          </div>

          <img
            ref={mascotRef}
            src="/images/mascot/hero.webp"
            alt="CasperProver Spirit"
            className="relative z-10 w-56 sm:w-72 lg:w-[380px] drop-shadow-[0_0_40px_rgba(255,50,50,0.3)] transition-transform duration-300 ease-out"
            loading="eager"
          />

          {/* Floating labels */}
          <div className="absolute top-[15%] left-0 sm:left-[5%] z-20 px-3 py-1.5 rounded-lg bg-cp-red/10 border border-cp-red/20 backdrop-blur-sm text-xs animate-float">
            <span className="text-cp-red font-mono font-bold">INPUT</span>
            <p className="text-gray-400 text-[10px] font-mono mt-0.5">"kyc_id": "****"</p>
          </div>
          <div className="absolute top-[40%] right-0 sm:right-[5%] z-20 px-3 py-1.5 rounded-lg bg-green-900/20 border border-green-500/20 backdrop-blur-sm text-xs animate-float-delay">
            <span className="text-green-400 font-bold">ZERO-KNOWLEDGE PROOF</span>
            <p className="text-gray-400 text-[10px] font-mono mt-0.5">0x5f3a...b8e21 ✓</p>
          </div>
          <div className="absolute bottom-[20%] left-[10%] z-20 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 backdrop-blur-sm text-xs animate-float-slow">
            <span className="text-gray-300 font-bold">⬡ DECISION</span>
            <p className="text-green-400 text-[10px] mt-0.5">✓ APPROVED</p>
          </div>
        </div>
      </div>

      {/* Feature bar */}
      <div className="absolute bottom-0 inset-x-0 z-10">
        <div className="cp-section">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pb-8">
            {[
              { icon: '🔐', title: 'Verifiable', desc: 'Cryptographic proofs you can verify on-chain.' },
              { icon: '🔒', title: 'Private', desc: 'Zero-knowledge proofs protect sensitive data.' },
              { icon: '⚡', title: 'Trustless', desc: 'No intermediaries. Just math and consensus.' },
              { icon: '🏗️', title: 'On Casper', desc: 'Built natively for speed, security, and scalability.' },
            ].map((f, i) => (
              <div key={i} className="group p-4 rounded-xl border border-white/5 bg-white/[0.02] backdrop-blur-sm hover:border-cp-red/30 hover:bg-cp-red/5 transition-all">
                <span className="text-2xl mb-2 block">{f.icon}</span>
                <h3 className="text-sm font-semibold text-white mb-1">{f.title}</h3>
                <p className="text-xs text-gray-500 leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
