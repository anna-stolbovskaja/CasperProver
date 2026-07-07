import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ToastProvider } from './lib/toast'
import ToastBridge from './lib/ToastBridge'
import App from './App'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <ToastProvider>
        <ToastBridge />
        <App />
      </ToastProvider>
    </BrowserRouter>
  </StrictMode>
)
