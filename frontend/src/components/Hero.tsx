import { useEffect, useState, useRef } from 'react'

export default function Hero() {
  const [typed, setTyped] = useState('')
  const fullText = 'casper-prover verify --model gpt-4 --input "loan_decision_42" --anchor testnet'
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    let i = 0
    const interval = setInterval(() => {
      if (i <= fullText.length) {
        setTyped(fullText.slice(0, i))
        i++
      } else {
        clearInterval(interval)
      }
    }, 40)
    return () => clearInterval(interval)
  }, [])

  // Fire particles
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    canvas.width = canvas.offsetWidth * 2
    canvas.height = canvas.offsetHeight * 2
    ctx.scale(2, 2)

    const particles: {x:number, y:number, vx:number, vy:number, life:number, size:number}[] = []
    const W = canvas.offsetWidth, H = canvas.offsetHeight
    const cx = W * 0.65, cy = H * 0.6

    let raf: number
    const animate = () => {
      ctx.clearRect(0, 0, W, H)

      if (Math.random() < 0.3) {
        particles.push({
          x: cx + (Math.random() - 0.5) * 60,
          y: cy + Math.random() * 20,
          vx: (Math.random() - 0.5) * 0.5,
          vy: -(1 + Math.random() * 2),
          life: 1,
          size: 1 + Math.random() * 3,
        })
      }

      for (let i = particles.length - 1; i >= 0; i--) {
        const p = particles[i]
        p.x += p.vx
        p.y += p.vy
        p.life -= 0.015
        if (p.life <= 0) { particles.splice(i, 1); continue }
        ctx.beginPath()
        ctx.arc(p.x, p.y, p.size * p.life, 0, Math.PI * 2)
        const r = 229 + Math.random() * 26
        const g = 57 * p.life
        ctx.fillStyle = `rgba(${r},${g},20,${p.life * 0.7})`
        ctx.fill()
      }

      raf = requestAnimationFrame(animate)
    }
    animate()
    return () => cancelAnimationFrame(raf)
  }, [])

  return (
    <section id="home" className="relative min-h-screen flex items-center overflow-hidden">
      {/* Grid background */}
      <div className="absolute inset-0 opacity-[0.03]" style={{
        backgroundImage: 'linear-gradient(#E53935 1px, transparent 1px), linear-gradient(90deg, #E53935 1px, transparent 1px)',
        backgroundSize: '60px 60px',
      }} />

      {/* Scan line */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="w-full h-px bg-gradient-to-r from-transparent via-red-500/30 to-transparent animate-scan" />
      </div>

      {/* Fire particles canvas */}
      <canvas ref={canvasRef} className="absolute inset-0 w-full h-full pointer-events-none z-10" />

      {/* Mascot - large, slightly right of center (mobile: visible) */}
      <div className="absolute right-[5%] sm:right-[8%] bottom-0 z-[5] w-[50%] sm:w-[40%] lg:w-[35%] max-w-lg">
        <img
          src="/images/mascot/hero.webp"
          alt="CasperProver Spirit"
          className="w-full animate-fire-glow"
          loading="eager"
        />
      </div>

      <div className="cp-section relative z-20 py-32">
        <div className="max-w-2xl">
          {/* Terminal badge */}
          <div className="inline-flex items-center gap-2 px-3 py-1.5 mb-8 rounded-md bg-red-500/10 border border-red-500/20 font-mono text-xs text-red-400">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse" />
            ZERO-KNOWLEDGE PROOFS FOR AI
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold text-white leading-[1.08] mb-6 tracking-tight">
            Your AI made<br />a decision.<br />
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-red-500 to-orange-400">Prove it.</span>
          </h1>

          <p className="text-lg text-gray-400 mb-10 leading-relaxed max-w-lg">
            CasperProver generates cryptographic proofs for AI agent decisions and anchors them on Casper Network. Verifiable. Private. Trustless.
          </p>

          {/* Terminal command */}
          <div className="bg-black/60 backdrop-blur rounded-xl border border-gray-800 p-5 mb-10 max-w-xl">
            <div className="flex items-center gap-2 mb-3">
              <div className="w-2.5 h-2.5 rounded-full bg-red-500" />
              <div className="w-2.5 h-2.5 rounded-full bg-yellow-500" />
              <div className="w-2.5 h-2.5 rounded-full bg-green-500" />
              <span className="text-gray-600 text-xs ml-2 font-mono">terminal</span>
            </div>
            <div className="font-mono text-sm">
              <span className="text-gray-500">$ </span>
              <span className="text-green-400">{typed}</span>
              <span className="animate-pulse text-red-400">_</span>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <a href="/app" className="inline-flex items-center gap-2 px-7 py-3.5 rounded-xl bg-red-600 text-white font-semibold hover:bg-red-500 transition-all shadow-lg shadow-red-600/20">
              Generate Proof
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
            </a>
            <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 px-7 py-3.5 rounded-xl border border-gray-700 text-gray-300 font-medium hover:border-gray-500 hover:text-white transition-colors">
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
              Source Code
            </a>
          </div>
        </div>
      </div>
    </section>
  )
}
