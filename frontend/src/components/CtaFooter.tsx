import { ArrowRight, FileText } from 'lucide-react'
import { Link } from 'react-router-dom'

export default function CtaFooter() {
  return (
    <section className="py-12 sm:py-16">
      <div className="cp-section">
        <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-cp-card to-cp-card border border-cp-border">
          {/* Glow */}
          <div className="absolute inset-0 bg-gradient-to-r from-cp-red/5 to-transparent pointer-events-none" aria-hidden="true" />

          <div className="relative flex flex-col sm:flex-row items-center justify-between gap-6 p-8 sm:p-10">
            <div className="flex items-center gap-4">
              <img src="/images/logo.png" alt="" width={48} height={48} className="rounded-full shrink-0" />
              <div>
                <h2 className="text-xl sm:text-2xl font-bold">Ready to make your AI accountable?</h2>
                <p className="text-cp-gray text-sm mt-1">
                  Join the future of verifiable AI on <span className="cp-gradient-text font-semibold">Casper Network</span>.
                </p>
              </div>
            </div>
            <div className="flex gap-3 shrink-0">
              <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noopener noreferrer" className="cp-btn-outline">
                Explore Docs <FileText size={16} />
              </a>
              <Link to="/app" className="cp-btn-primary">
                Start Building <ArrowRight size={18} />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
