import { Link } from 'react-router-dom'

export default function NotFound() {
  return (
    <section className="min-h-screen flex items-center justify-center relative">
      <div className="absolute inset-0 bg-cp-black" />
      <div className="absolute inset-0 opacity-[0.04]" style={{ backgroundImage: 'radial-gradient(circle at 1px 1px, rgba(255,50,50,0.3) 1px, transparent 0)', backgroundSize: '40px 40px' }} />
      <div className="relative z-10 text-center px-6">
        <img src="/images/mascot/pose4.webp" alt="" className="w-32 mx-auto mb-6 drop-shadow-[0_0_20px_rgba(255,50,50,0.3)]" />
        <h1 className="text-6xl font-extrabold text-cp-red mb-4">404</h1>
        <p className="text-xl text-gray-400 mb-2">proof not found</p>
        <p className="text-sm text-gray-500 mb-8">no merkle path leads here. verification impossible.</p>
        <Link to="/" className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-cp-red text-white font-semibold hover:scale-[1.02] transition-transform">
          back to root
        </Link>
      </div>
    </section>
  )
}
