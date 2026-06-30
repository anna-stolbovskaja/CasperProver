import { ArrowRight } from 'lucide-react'

export default function CtaFooter() {
  return (
    <section className="py-20 relative overflow-hidden">
      <div className="absolute inset-0 bg-gradient-to-t from-cp-black via-[#0a0808] to-cp-black" />

      <div className="cp-section relative z-10">
        <div className="relative rounded-2xl border border-cp-red/20 bg-gradient-to-r from-cp-red/10 via-red-900/5 to-cp-red/5 p-8 sm:p-12 overflow-hidden">
          <div className="absolute -top-20 -right-20 w-60 h-60 bg-cp-red/15 rounded-full blur-[80px] pointer-events-none" />

          <div className="flex flex-col md:flex-row items-center gap-8">
            <img src="/images/casper-logo.png" alt="" className="w-12 h-12 opacity-60" />

            <div className="flex-1 text-center md:text-left">
              <h2 className="text-2xl sm:text-3xl font-bold text-white mb-2">
                Ready to make your AI accountable?
              </h2>
              <p className="text-gray-400 mb-6">
                Join the future of verifiable AI on <span className="text-cp-red">Casper Network</span>.
              </p>
              <div className="flex flex-wrap gap-4 justify-center md:justify-start">
                <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-white/10 text-white font-medium hover:bg-white/5 transition-colors">
                  Explore Docs
                </a>
                <a href="/app" className="group inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-cp-red text-white font-semibold hover:scale-[1.02] transition-transform shadow-lg shadow-cp-red/25">
                  Start Building <ArrowRight className="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
