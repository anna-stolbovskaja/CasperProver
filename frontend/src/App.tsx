import { Routes, Route, useLocation } from 'react-router-dom'
import { useState, useEffect, useCallback } from 'react'
import Navbar from './components/Navbar'
import Hero from './components/Hero'
import LiveDemo from './components/LiveDemo'
import Features from './components/Features'
import HowItWorks from './components/HowItWorks'
import Benchmarks from './components/Benchmarks'
import SDKSection from './components/SDKSection'
import FAQ from './components/FAQ'
import CtaFooter from './components/CtaFooter'
import Footer from './components/Footer'
import ScrollProgress from './components/ScrollProgress'
import ScrollToTop from './components/ScrollToTop'
import NotFound from './components/NotFound'
import Dashboard from './components/Dashboard'
import { DataModeProvider } from './components/DataModeToggle'

function Landing() {
  useEffect(() => {
    const hash = window.location.hash.slice(1)
    if (hash) {
      // Wait for React to finish rendering all sections, then scroll
      const raf = requestAnimationFrame(() => {
        const el = document.getElementById(hash)
        if (el) el.scrollIntoView({ behavior: 'smooth' })
      })
      return () => cancelAnimationFrame(raf)
    }
  }, [])

  return (
    <>
      <Hero />
      <LiveDemo />
      <Features />
      <HowItWorks />
      <Benchmarks />
      <SDKSection />
      <FAQ />
      <CtaFooter />
    </>
  )
}

export default function App() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const closeMobile = useCallback(() => setMobileOpen(false), [])
  const location = useLocation()
  useEffect(() => {
    document.body.style.overflow = mobileOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [mobileOpen])

  return (
    <DataModeProvider>
    <div className="min-h-screen flex flex-col">
      <ScrollProgress />
      <ScrollToTop />
      <Navbar mobileOpen={mobileOpen} setMobileOpen={setMobileOpen} />
      <main className="flex-1" onClick={mobileOpen ? closeMobile : undefined}>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      <Footer />
    </div>
    </DataModeProvider>
  )
}
