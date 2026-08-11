// TigerSwap DApp Browser - Web3 Browser with Wallet Integration

export interface DAppBrowser {
  url: string
  title: string
  favicon: string
  permissions: Permission[]
  isConnected: boolean
  chainId: number
}

export interface Permission {
  type: PermissionType
  granted: boolean
  timestamp: number
}

export type PermissionType = 'wallet' | 'network' | 'assets' | 'transactions' | 'signatures'

export interface WalletConnection {
  address: string
  chainId: number
  connectedAt: number
  dAppUrl: string
}

export interface TransactionRequest {
  id: string
  from: string
  to: string
  value: string
  data: string
  gas: string
  gasPrice: string
  nonce: number
  chainId: number
}

export class DAppBrowserCore {
  private connections: Map<string, WalletConnection> = new Map()
  private permissions: Map<string, Permission[]> = new Map()
  private bookmarks: any[] = []

  async connect(dAppUrl: string): Promise<string> {
    const address = this.generateAddress()
    const connection: WalletConnection = {
      address,
      chainId: 1,
      connectedAt: Date.now(),
      dAppUrl,
    }
    this.connections.set(dAppUrl, connection)
    return address
  }

  async disconnect(dAppUrl: string): Promise<void> {
    this.connections.delete(dAppUrl)
  }

  async requestPermission(dAppUrl: string, permission: PermissionType): Promise<boolean> {
    return true
  }

  private generateAddress(): string {
    // Addresses are derived by the canonical wallet-api backend from a real
    // BIP-39 seed (secp256k1 / BIP-44). This client never fabricates an address.
    throw new Error('Address derivation is performed by the canonical wallet-api backend; client-side fabrication is disabled')
  }
}

export default DAppBrowserCore