import { ArrowRight, FileText } from 'lucide-react'

export default function Hero() {
  return (
    <section id="home" className="relative min-h-screen flex items-center overflow-hidden pt-16">
      {/* Background grid */}
      <div className="absolute inset-0 opacity-[0.03]" aria-hidden="true"
        style={{ backgroundImage: 'linear-gradient(rgba(255,255,255,0.1) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.1) 1px, transparent 1px)', backgroundSize: '60px 60px' }} />

      {/* Ambient glow */}
      <div className="absolute top-1/3 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-cp-red/5 rounded-full blur-[120px] pointer-events-none" aria-hidden="true" />

      <div className="cp-section relative z-10 flex flex-col lg:flex-row items-center gap-8 lg:gap-4 py-12 lg:py-0">
        {/* Left content */}
        <div className="flex-1 max-w-xl animate-fade-in-up">
          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold leading-[1.1] tracking-tight mb-6">
            Provable AI.<br />
            <span className="cp-gradient-text">On Casper.</span>
          </h1>
          <p className="text-lg sm:text-xl text-cp-gray leading-relaxed mb-8 max-w-lg">
            CasperProver creates cryptographic proofs of AI
            agent decisions — verifiable, private, and trustless.
            No black boxes. Just proof.
          </p>
          <div className="flex flex-wrap gap-3 mb-10">
            <a href="#live-demo" className="cp-btn-primary">
              Start Building <ArrowRight size={18} />
            </a>
            <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noopener noreferrer" className="cp-btn-outline">
              Read Docs <FileText size={16} />
            </a>
          </div>
          <div className="flex items-center gap-2 text-sm text-cp-gray-dark">
            <span>Built on</span>
            <img src="/images/logo.png" alt="" width={20} height={20} className="rounded-full opacity-80" />
            <span className="text-cp-gray">Casper Network</span>
          </div>
        </div>

        {/* Right: Mascot with effects */}
        <div className="flex-1 flex justify-center items-center relative max-w-lg lg:max-w-xl">
          {/* Smoke / haze layers */}
          <div className="smoke-effect" aria-hidden="true" />
          <div className="smoke-effect smoke-effect-blue" aria-hidden="true" />
          <div className="smoke-effect smoke-effect-purple" aria-hidden="true" />

          {/* Spark particles */}
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="spark" aria-hidden="true"
              style={{
                left: `${20 + Math.random() * 60}%`,
                top: `${30 + Math.random() * 40}%`,
                animationDelay: `${i * 0.3}s`,
                animationDuration: `${1.5 + Math.random() * 1.5}s`,
              }} />
          ))}

          {/* Mascot image with color-shift animation */}
          <img
            src="/images/mascot-gradient-t.png"
            alt="CasperProver mascot"
            width={480}
            height={560}
            className="relative z-10 w-full max-w-[400px] lg:max-w-[480px] h-auto animate-color-shift drop-shadow-2xl select-none"
            loading="eager"
            draggable={false}
          />

          {/* Floating UI cards */}
          <div className="absolute top-[15%] left-0 sm:left-[-5%] z-20 animate-float" aria-hidden="true">
            <div className="bg-cp-card/90 backdrop-blur border border-cp-border rounded-lg px-3 py-2 text-xs">
              <div className="text-cp-red font-mono text-[10px] mb-1">INPUT</div>
              <div className="text-cp-gray font-mono text-[10px]">"kyc_id": "****"</div>
              <div className="text-cp-gray font-mono text-[10px]">"country": "****"</div>
            </div>
          </div>

          <div className="absolute bottom-[30%] left-[2%] sm:left-[-2%] z-20 animate-float-delay" aria-hidden="true">
            <div className="bg-cp-card/90 backdrop-blur border border-cp-border rounded-lg px-3 py-2 text-xs flex items-center gap-2">
              <div className="w-5 h-5 rounded bg-cp-red/20 flex items-center justify-center">
                <span className="text-cp-red text-[10px] font-bold">A</span>
              </div>
              <div>
                <div className="text-white text-[11px] font-medium">DECISION</div>
                <div className="text-green-400 text-[10px] flex items-center gap-1">
                  <svg width="10" height="10" viewBox="0 0 16 16" fill="none"><path d="M3 8l3 3 7-7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
                  APPROVED
                </div>
              </div>
            </div>
          </div>

          <div className="absolute top-[20%] right-0 sm:right-[-3%] z-20 animate-float" style={{ animationDelay: '1s' }} aria-hidden="true">
            <div className="bg-cp-card/90 backdrop-blur border border-cp-border rounded-lg px-3 py-2 text-xs">
              <div className="text-cp-red font-semibold text-[10px] mb-1">ZERO-KNOWLEDGE PROOF</div>
              <div className="text-cp-gray font-mono text-[10px] mb-1">0x5f3a...b8e21</div>
              <div className="text-green-400 text-[10px] flex items-center gap-1">
                <svg width="10" height="10" viewBox="0 0 16 16" fill="none"><path d="M3 8l3 3 7-7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
                Verified
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
