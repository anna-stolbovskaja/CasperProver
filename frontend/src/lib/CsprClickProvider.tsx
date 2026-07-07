/**
 * CSPR.click integration wrapper.
 *
 * Wraps the app in ClickProvider and exposes a lightweight React context
 * so any component can read the connected account and call sign/send.
 */

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react'
import { ClickProvider, useClickRef } from '@make-software/csprclick-ui'
import { CONTENT_MODE } from '@make-software/csprclick-core-types'
import { CSPR_CLICK_APP_ID, type WalletState } from './wallet'

/* ---------- context ---------- */

interface WalletCtx extends WalletState {
  signIn: () => void
  signOut: () => void
}

const WalletContext = createContext<WalletCtx>({
  connected: false,
  publicKey: null,
  accountHash: null,
  provider: null,
  signIn: () => {},
  signOut: () => {},
})

export const useWallet = () => useContext(WalletContext)

/* ---------- inner (has access to useClickRef) ---------- */

function WalletBridge({ children }: { children: ReactNode }) {
  const clickRef = useClickRef()
  const [wallet, setWallet] = useState<WalletState>({
    connected: false,
    publicKey: null,
    accountHash: null,
    provider: null,
  })

  useEffect(() => {
    if (!clickRef) return

    const onSignedIn = (evt: { account: { public_key: string; provider: string } }) => {
      setWallet({
        connected: true,
        publicKey: evt.account.public_key,
        accountHash: evt.account.public_key.slice(0, 64),
        provider: evt.account.provider,
      })
    }

    const onSignedOut = () => {
      setWallet({ connected: false, publicKey: null, accountHash: null, provider: null })
    }

    const onSwitched = (evt: { account: { public_key: string; provider: string } }) => {
      setWallet({
        connected: true,
        publicKey: evt.account.public_key,
        accountHash: evt.account.public_key.slice(0, 64),
        provider: evt.account.provider,
      })
    }

    clickRef.on('csprclick:signed_in', onSignedIn)
    clickRef.on('csprclick:switched_account', onSwitched)
    clickRef.on('csprclick:signed_out', onSignedOut)
    clickRef.on('csprclick:disconnected', onSignedOut)

    const existing = clickRef.getActiveAccount?.()
    if (existing?.public_key) {
      setWallet({
        connected: true,
        publicKey: existing.public_key,
        accountHash: existing.public_key.slice(0, 64),
        provider: existing.provider,
      })
    }

    return () => {
      clickRef.off('csprclick:signed_in', onSignedIn)
      clickRef.off('csprclick:switched_account', onSwitched)
      clickRef.off('csprclick:signed_out', onSignedOut)
      clickRef.off('csprclick:disconnected', onSignedOut)
    }
  }, [clickRef])

  const signIn = useCallback(() => {
    clickRef?.signIn()
  }, [clickRef])

  const signOut = useCallback(() => {
    clickRef?.signOut()
  }, [clickRef])

  return (
    <WalletContext.Provider value={{ ...wallet, signIn, signOut }}>
      {children}
    </WalletContext.Provider>
  )
}

/* ---------- outer wrapper ---------- */

export default function CsprClickWrapper({ children }: { children: ReactNode }) {
  return (
    <ClickProvider
      options={{
        appName: 'CasperProver',
        appId: CSPR_CLICK_APP_ID,
        contentMode: CONTENT_MODE.POPUP,
        providers: ['casper-wallet', 'ledger', 'metamask-snap', 'csprclick-w3a-google'],
        chainName: 'casper-test',
        logLevel: 1,
      }}
    >
      <WalletBridge>
        {children}
      </WalletBridge>
    </ClickProvider>
  )
}
