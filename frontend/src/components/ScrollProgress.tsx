import { useState, useEffect, useCallback } from 'react'
import { ArrowUp } from 'lucide-react'

export default function ScrollProgress() {
  const [progress, setProgress] = useState(0)
  const [visible, setVisible] = useState(false)

  const handleScroll = useCallback(() => {
    const h = document.documentElement
    const pct = h.scrollTop / (h.scrollHeight - h.clientHeight)
    setProgress(Math.min(pct * 100, 100))
    setVisible(h.scrollTop > 400)
  }, [])

  useEffect(() => {
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [handleScroll])

  const scrollTop = () => window.scrollTo({ top: 0, behavior: 'smooth' })

  const r = 20, c = 2 * Math.PI * r
  const offset = c - (progress / 100) * c

  return (
    <>
      {/* Top bar */}
      <div className="fixed top-0 left-0 w-full h-[2px] z-[60]" aria-hidden="true">
        <div className="h-full bg-cp-red transition-[width] duration-150" style={{ width: `${progress}%` }} />
      </div>
      {/* Scroll-to-top */}
      <button
        onClick={scrollTop}
        aria-label="Scroll to top"
        className={`fixed bottom-6 right-6 z-50 w-12 h-12 rounded-full bg-cp-card border border-cp-border flex items-center justify-center cursor-pointer transition-all duration-300 hover:border-cp-red group ${visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4 pointer-events-none'}`}
      >
        <svg className="absolute inset-0 w-12 h-12 -rotate-90" viewBox="0 0 48 48" aria-hidden="true">
          <circle cx="24" cy="24" r={r} fill="none" stroke="rgba(229,57,53,0.2)" strokeWidth="2" />
          <circle cx="24" cy="24" r={r} fill="none" stroke="#E53935" strokeWidth="2"
            strokeDasharray={c} strokeDashoffset={offset} strokeLinecap="round"
            className="transition-[stroke-dashoffset] duration-150" />
        </svg>
        <ArrowUp size={18} className="text-cp-gray group-hover:text-cp-red transition-colors duration-200" />
      </button>
    </>
  )
}
