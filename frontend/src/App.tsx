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

function Landing() {
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
  const isDashboard = location.pathname === '/dashboard'
  useEffect(() => {
    document.body.style.overflow = mobileOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [mobileOpen])

  return (
    <div className="min-h-screen flex flex-col">
      <ScrollProgress />
      <ScrollToTop />
      {!isDashboard && <Navbar mobileOpen={mobileOpen} setMobileOpen={setMobileOpen} />}
      <main className="flex-1" onClick={mobileOpen ? closeMobile : undefined}>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      {!isDashboard && <Footer />}
    </div>
  )
}
