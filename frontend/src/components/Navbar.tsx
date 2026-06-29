import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Menu, X, ExternalLink } from 'lucide-react'

const NAV = [
  { label: 'Home', href: '/#home' },
  { label: 'How it Works', href: '/#how-it-works' },
  { label: 'Use Cases', href: '/#use-cases' },
  { label: 'Docs', href: 'https://github.com/anna-stolbovskaja/CasperProver', external: true },
  { label: 'About', href: '/#architecture' },
]

interface Props {
  mobileOpen: boolean
  setMobileOpen: (v: boolean) => void
}

export default function Navbar({ mobileOpen, setMobileOpen }: Props) {
  const [scrolled, setScrolled] = useState(false)
  const location = useLocation()

  useEffect(() => {
    const fn = () => setScrolled(window.scrollY > 20)
    window.addEventListener('scroll', fn, { passive: true })
    return () => window.removeEventListener('scroll', fn)
  }, [])

  // Handle hash navigation
  const handleNavClick = (href: string) => {
    setMobileOpen(false)
    if (href.startsWith('/#')) {
      const id = href.slice(2)
      if (location.pathname === '/') {
        document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
      } else {
        window.location.href = href
      }
    }
  }

  return (
    <header className={`fixed top-0 left-0 w-full z-50 transition-all duration-300 ${scrolled ? 'bg-cp-black/90 backdrop-blur-md border-b border-cp-border' : 'bg-transparent'}`}>
      <nav className="cp-section flex items-center justify-between h-16 sm:h-18" aria-label="Main navigation">
        {/* Logo */}
        <Link to="/" className="flex items-center gap-2.5 shrink-0" aria-label="CasperProver Home">
          <img src="/images/logo.png" alt="" width={36} height={36} className="rounded-full" />
          <span className="text-lg font-bold tracking-tight">
            Casper<span className="cp-gradient-text">Prover</span>
          </span>
        </Link>

        {/* Desktop nav */}
        <ul className="hidden md:flex items-center gap-1">
          {NAV.map(n => (
            <li key={n.label}>
              {n.external ? (
                <a href={n.href} target="_blank" rel="noopener noreferrer"
                  className="px-3 py-2 text-sm text-cp-gray hover:text-white transition-colors duration-200 cursor-pointer">
                  {n.label}
                </a>
              ) : (
                <button onClick={() => handleNavClick(n.href)}
                  className="px-3 py-2 text-sm text-cp-gray hover:text-white transition-colors duration-200 cursor-pointer">
                  {n.label}
                </button>
              )}
            </li>
          ))}
        </ul>

        {/* Desktop CTA */}
        <div className="hidden md:flex items-center gap-3">
          <Link to="/app" className="cp-btn-outline !py-2 !px-4 !text-sm">
            Launch dApp <ExternalLink size={14} />
          </Link>
          <button className="cp-btn-primary !py-2 !px-4 !text-sm cursor-pointer">
            Connect Wallet
          </button>
        </div>

        {/* Mobile burger */}
        <button
          onClick={() => setMobileOpen(!mobileOpen)}
          className="md:hidden p-2 text-cp-gray hover:text-white transition-colors cursor-pointer"
          aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
          aria-expanded={mobileOpen}
        >
          {mobileOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
      </nav>

      {/* Mobile menu */}
      <div className={`md:hidden transition-all duration-300 overflow-hidden ${mobileOpen ? 'max-h-96 border-b border-cp-border' : 'max-h-0'}`}
        role="menu" aria-hidden={!mobileOpen}>
        <div className="cp-section pb-6 pt-2 flex flex-col gap-1 bg-cp-black/95 backdrop-blur-md">
          {NAV.map(n => (
            n.external ? (
              <a key={n.label} href={n.href} target="_blank" rel="noopener noreferrer"
                className="py-3 px-3 text-cp-gray hover:text-white transition-colors cursor-pointer" role="menuitem">
                {n.label}
              </a>
            ) : (
              <button key={n.label} onClick={() => handleNavClick(n.href)}
                className="py-3 px-3 text-left text-cp-gray hover:text-white transition-colors cursor-pointer" role="menuitem">
                {n.label}
              </button>
            )
          ))}
          <div className="flex gap-3 mt-3 px-3">
            <Link to="/app" onClick={() => setMobileOpen(false)} className="cp-btn-outline !py-2 !px-4 !text-sm flex-1 justify-center">
              Launch dApp
            </Link>
            <button className="cp-btn-primary !py-2 !px-4 !text-sm flex-1 justify-center cursor-pointer">
              Connect Wallet
            </button>
          </div>
        </div>
      </div>
    </header>
  )
}
