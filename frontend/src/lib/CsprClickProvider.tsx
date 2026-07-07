/**
 * CSPR.click SDK integration — CDN-based pattern.
 *
 * Loads the CSPR.click SDK from CDN, hides its default top-bar via CSS,
 * and exposes a React context for custom wallet UI. The sign-in modal
 * itself still renders from the SDK (popup/iframe).
 *
 * Reference: @make-software/csprclick-core-types v2.1.0
 */
import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from 'react'
import { CONTENT_MODE } from '@make-software/csprclick-core-types'
import type {
  AccountType,
  CsprClickInitOptions,
  ICSPRClickSDK,
} from '@make-software/csprclick-core-types'
import type { ClickUIOptions } from '@make-software/csprclick-core-types/clickui'
import { CSPR_CLICK_APP_ID } from './wallet'

declare global {
  interface Window {
    clickUIOptions: ClickUIOptions
    clickSDKOptions: CsprClickInitOptions
    csprclick?: ICSPRClickSDK
  }
}

/* ---------- SDK bootstrap options ---------- */

window.clickUIOptions = {
  uiContainer: 'csprclick-ui',
  rootAppElement: '#root',
  showTopBar: false,
  defaultTheme: 'dark',
  accountMenuItems: ['CopyHashMenuItem'],
}

window.clickSDKOptions = {
  appName: 'CasperProver',
  appId: CSPR_CLICK_APP_ID,
  providers: ['casper-wallet', 'ledger', 'metamask-snap', 'csprclick-w3a-google'],
  contentMode: CONTENT_MODE.IFRAME,
  chainName: 'casper-test',
}

/* ---------- context ---------- */

interface WalletCtx {
  connected: boolean
  publicKey: string | null
  accountHash: string | null
  provider: string | null
  clickRef: ICSPRClickSDK | undefined
  ready: boolean
  signIn: () => void
  signOut: () => void
}

const WalletContext = createContext<WalletCtx>({
  connected: false,
  publicKey: null,
  accountHash: null,
  provider: null,
  clickRef: undefined,
  ready: false,
  signIn: () => {},
  signOut: () => {},
})

export const useWallet = () => useContext(WalletContext)

/* ---------- provider ---------- */

type AccountChangedEvent = { account?: AccountType }

export default function CsprClickWrapper({ children }: { children: ReactNode }) {
  const [connectedAccount, setConnectedAccount] = useState<AccountType | undefined>()
  const [clickRef, setClickRef] = useState<ICSPRClickSDK | undefined>()
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const checkActive = async (ref: ICSPRClickSDK) => {
      try {
        const account = await ref.getActiveAccountAsync({ withBalance: false })
        setConnectedAccount(account?.public_key ? account : undefined)
      } catch {
        setConnectedAccount(undefined)
      }
    }

    const handleAccountChanged = (event: AccountChangedEvent) => {
      setConnectedAccount(event.account?.public_key ? event.account : undefined)
    }

    const handleSdkLoaded = () => {
      const ref = window.csprclick
      if (!ref) return
      setClickRef(ref)
      setReady(true)
      ref.on('csprclick:signed_in', handleAccountChanged)
      ref.on('csprclick:switched_account', handleAccountChanged)
      ref.on('csprclick:unsolicited_account_change', handleAccountChanged)
      ref.on('csprclick:signed_out', () => setConnectedAccount(undefined))
      ref.on('csprclick:disconnected', () => setConnectedAccount(undefined))
      checkActive(ref)
    }

    window.addEventListener('csprclick:loaded', handleSdkLoaded)
    if (window.csprclick) handleSdkLoaded()

    // Load the SDK script from CDN if not already present
    if (!document.querySelector('script#csprclick-client')) {
      const script = document.createElement('script')
      script.src = 'https://cdn.cspr.click/ui/v2.1.0/csprclick-client-2.1.0.js'
      script.id = 'csprclick-client'
      script.async = true
      document.head.appendChild(script)
    }

    return () => window.removeEventListener('csprclick:loaded', handleSdkLoaded)
  }, [])

  const signIn = useCallback(() => {
    clickRef?.signIn()
  }, [clickRef])

  const signOut = useCallback(() => {
    clickRef?.signOut()
  }, [clickRef])

  const connected = !!connectedAccount?.public_key

  return (
    <WalletContext.Provider
      value={{
        connected,
        publicKey: connectedAccount?.public_key ?? null,
        accountHash: (connectedAccount as any)?.account_hash ?? null,
        provider: (connectedAccount as any)?.provider ?? null,
        clickRef,
        ready,
        signIn,
        signOut,
      }}
    >
      {children}
    </WalletContext.Provider>
  )
}
