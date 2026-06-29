import { Github, Twitter, ExternalLink } from 'lucide-react'

export default function Footer() {
  return (
    <footer className="border-t border-cp-border py-8" role="contentinfo">
      <div className="cp-section flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-cp-gray-dark">
        <div className="flex items-center gap-2">
          <img src="/images/logo.png" alt="" width={20} height={20} className="rounded-full opacity-60" />
          <span>© {new Date().getFullYear()} CasperProver. Built on Casper Network.</span>
        </div>
        <div className="flex items-center gap-4">
          <a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noopener noreferrer"
            className="hover:text-white transition-colors cursor-pointer" aria-label="GitHub">
            <Github size={18} />
          </a>
          <a href="https://testnet.cspr.live" target="_blank" rel="noopener noreferrer"
            className="hover:text-white transition-colors cursor-pointer inline-flex items-center gap-1" aria-label="Casper Testnet Explorer">
            <ExternalLink size={14} /> Explorer
          </a>
        </div>
      </div>
    </footer>
  )
}
