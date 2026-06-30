/**
 * Casper Wallet integration with simulated fallback.
 */

const DEMO_PUBLIC_KEY = '020260dd84fc2f98a96e6a62ad499e0bcf21e7edf0eb1b48ee0fba6fda0d9478af4c'
const DEMO_ACCOUNT_HASH = 'a3f5b4c2d1e6f7089a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f'

export interface WalletState {
  connected: boolean
  publicKey: string | null
  accountHash: string | null
  simulated: boolean
}

export function detectCasperWallet(): boolean {
  return typeof window !== 'undefined' && !!(window as Record<string, unknown>).CasperWalletProvider
}

export async function connectWallet(): Promise<WalletState> {
  if (detectCasperWallet()) {
    try {
      const provider = (window as Record<string, unknown>).CasperWalletProvider as () => {
        requestConnection: () => Promise<boolean>
        getActivePublicKey: () => Promise<string>
      }
      const wallet = provider()
      const ok = await wallet.requestConnection()
      if (ok) {
        const pubKey = await wallet.getActivePublicKey()
        return { connected: true, publicKey: pubKey, accountHash: pubKey.slice(0, 64), simulated: false }
      }
    } catch { /* fall through */ }
  }
  return { connected: true, publicKey: DEMO_PUBLIC_KEY, accountHash: DEMO_ACCOUNT_HASH, simulated: true }
}

export function disconnectWallet(): WalletState {
  return { connected: false, publicKey: null, accountHash: null, simulated: false }
}

export function shortKey(key: string): string {
  if (key.length <= 16) return key
  return key.slice(0, 8) + '...' + key.slice(-6)
}
