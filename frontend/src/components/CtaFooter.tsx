export default function CtaFooter() {
  return (
    <section className="py-20">
      <div className="cp-section">
        <div className="relative bg-gradient-to-br from-cp-card to-black rounded-3xl border border-cp-border p-10 lg:p-14 overflow-hidden">
          {/* Red glow */}
          <div className="absolute -right-20 -bottom-20 w-80 h-80 rounded-full bg-red-500/10 blur-[80px] pointer-events-none" />

          <div className="relative z-10 flex flex-col lg:flex-row items-center gap-10">
            <div className="flex-1">
              <h2 className="text-3xl font-extrabold text-white mb-4">
                Make your AI accountable.
              </h2>
              <p className="text-gray-400 mb-8 max-w-md">
                Generate your first proof in under a minute. No GPU, no complex setup. Just cryptographic certainty.
              </p>
              <div className="flex flex-wrap gap-3">
                <a href="/app" className="inline-flex items-center gap-2 px-8 py-4 bg-red-600 text-white font-semibold rounded-xl hover:bg-red-500 transition-all shadow-lg shadow-red-600/20">
                  Open Dashboard
                </a>
                <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 px-8 py-4 border border-gray-700 text-gray-300 rounded-xl hover:border-gray-500 transition-colors">
                  View Source
                </a>
              </div>
            </div>
            <div className="w-48 shrink-0">
              <img src="/images/mascot/pose3.webp" alt="CasperProver" className="w-full animate-fire-glow" />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
