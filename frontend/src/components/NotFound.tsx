export default function NotFound() {
  return (
    <div className="min-h-screen bg-cp-black flex items-center justify-center px-4">
      <div className="text-center max-w-md">
        <img src="/images/mascot/pose4.webp" alt="" className="w-40 mx-auto mb-6 animate-fire-glow" />
        <p className="text-red-500 font-mono text-sm mb-2">PROOF_NOT_FOUND</p>
        <h1 className="text-6xl font-extrabold text-white mb-3">404</h1>
        <p className="text-gray-500 mb-8">This path has no valid Merkle proof.</p>
        <a href="/" className="inline-flex items-center gap-2 px-6 py-3 bg-red-600 text-white font-semibold rounded-xl hover:bg-red-500 transition-colors">Return Home</a>
      </div>
    </div>
  )
}
