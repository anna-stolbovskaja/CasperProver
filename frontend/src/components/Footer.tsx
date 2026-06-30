export default function Footer() {
  return (
    <footer className="bg-black border-t border-gray-800/50 py-14">
      <div className="cp-section">
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-8 mb-10">
          <div>
            <div className="flex items-center gap-2 mb-3">
              <img src="/images/logo.webp" alt="CasperProver" className="h-6 w-auto" />
              <span className="font-bold text-white">CasperProver</span>
            </div>
            <p className="text-sm text-gray-500 leading-relaxed">
              Cryptographic proofs for AI agent decisions. Anchored on Casper Network.
            </p>
          </div>
          <div>
            <h4 className="text-gray-300 font-semibold mb-3 text-sm">Product</h4>
            <ul className="space-y-2 text-sm text-gray-500">
              <li><a href="#features" className="hover:text-gray-300 transition-colors">Features</a></li>
              <li><a href="#how" className="hover:text-gray-300 transition-colors">Pipeline</a></li>
              <li><a href="#demo" className="hover:text-gray-300 transition-colors">Live Demo</a></li>
              <li><a href="/app" className="hover:text-gray-300 transition-colors">Dashboard</a></li>
            </ul>
          </div>
          <div>
            <h4 className="text-gray-300 font-semibold mb-3 text-sm">Developers</h4>
            <ul className="space-y-2 text-sm text-gray-500">
              <li><a href="https://github.com/anna-stolbovskaja/CasperProver" target="_blank" rel="noreferrer" className="hover:text-gray-300 transition-colors">GitHub</a></li>
              <li><a href="#sdk" className="hover:text-gray-300 transition-colors">Go SDK</a></li>
              <li><a href="#sdk" className="hover:text-gray-300 transition-colors">MCP Server</a></li>
              <li><a href="#faq" className="hover:text-gray-300 transition-colors">FAQ</a></li>
            </ul>
          </div>
          <div>
            <h4 className="text-gray-300 font-semibold mb-3 text-sm">Built On</h4>
            <div className="flex items-center gap-2 mb-3">
              <img src="/images/casper-logo.png" alt="Casper" className="h-5 w-auto" />
              <span className="text-sm text-gray-400">Casper Network</span>
            </div>
            <p className="text-xs text-gray-600">2 smart contracts on Casper testnet. Proof storage and verification registry.</p>
          </div>
        </div>
        <div className="border-t border-gray-800/50 pt-6 flex flex-col sm:flex-row justify-between items-center gap-3">
          <p className="text-xs text-gray-600">&copy; 2026 CasperProver. MIT License.</p>
          <p className="text-xs text-gray-600">Casper Agentic Buildathon 2026</p>
        </div>
      </div>
    </footer>
  )
}
