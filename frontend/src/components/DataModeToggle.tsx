import { useState, createContext, useContext } from 'react'
import { FlaskConical, Cpu } from 'lucide-react'

interface DataModeCtx {
  mode: 'demo' | 'live'
  setMode: (m: 'demo' | 'live') => void
}

const DataModeContext = createContext<DataModeCtx>({ mode: 'demo', setMode: () => {} })

export function useDataMode() {
  return useContext(DataModeContext)
}

export function DataModeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setMode] = useState<'demo' | 'live'>('demo')
  return (
    <DataModeContext.Provider value={{ mode, setMode }}>
      {children}
    </DataModeContext.Provider>
  )
}

export default function DataModeToggle() {
  const { mode, setMode } = useDataMode()

  return (
    <div role="radiogroup" aria-label="Data mode" className="inline-flex items-center gap-1 rounded-lg border border-gray-700 bg-gray-900/60 p-0.5 backdrop-blur-sm">
      <button
        onClick={() => setMode('demo')}
        aria-pressed={mode === 'demo'}
        aria-label="Switch to Demo Mode"
        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${
          mode === 'demo'
            ? 'bg-violet-900/50 text-violet-300 shadow-sm'
            : 'text-gray-500 hover:text-gray-300'
        }`}
      >
        <FlaskConical className="w-3 h-3" />
        Demo
      </button>
      <button
        onClick={() => setMode('live')}
        aria-pressed={mode === 'live'}
        aria-label="Switch to Live Mode"
        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${
          mode === 'live'
            ? 'bg-emerald-900/50 text-emerald-300 shadow-sm'
            : 'text-gray-500 hover:text-gray-300'
        }`}
      >
        <Cpu className="w-3 h-3" />
        Live
      </button>
    </div>
  )
}
