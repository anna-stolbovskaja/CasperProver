import { Link } from 'react-router-dom'
import { ArrowLeft, ShieldOff } from 'lucide-react'

export default function NotFound() {
  return (
    <section className="min-h-[80vh] flex items-center justify-center">
      <div className="text-center px-4 animate-fade-in-up">
        {/* Mascot decoration */}
        <div className="relative inline-block mb-6">
          <img src="/images/mascot-blue-t.png" alt="" className="w-32 h-auto opacity-30 mx-auto" loading="lazy" />
          <div className="absolute inset-0 flex items-center justify-center">
            <ShieldOff size={48} className="text-cp-red/60" />
          </div>
        </div>

        <h1 className="text-6xl sm:text-8xl font-extrabold cp-gradient-text mb-4">404</h1>
        <p className="text-xl text-white font-semibold mb-2">Proof Not Found</p>
        <p className="text-cp-gray mb-8 max-w-sm mx-auto">
          This page doesn't exist on the Merkle tree. The verification failed.
        </p>
        <Link to="/" className="cp-btn-primary">
          <ArrowLeft size={18} /> Back to Home
        </Link>
      </div>
    </section>
  )
}
