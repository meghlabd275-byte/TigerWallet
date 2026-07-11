"use client";

import { useState, useCallback } from "react";
import { 
  Wallet, 
  Bitcoin, 
  Send, 
  RefreshCw, 
  Copy, 
  CheckCircle,
  AlertCircle,
  Loader2,
  ArrowRight,
  Flame,
  Hash,
  Image,
  ExternalLink,
  Layers,
  Shield
} from "lucide-react";

// Ordinals Types
interface Ordinal {
  id: string;
  number: number;
  collection: string;
  name: string;
  inscription: string;
  contentType: string;
  contentUrl?: string;
  sale?: {
    price: number;
    currency: string;
  };
}

interface BRC20Token {
  tick: string;
  balance: number;
  available: number;
  transferable: number;
}

interface UTXO {
  txid: string;
  vout: number;
  amount: number;
  ordinal?: Ordinal;
}

// Bitcoin Network
const BITCOIN_NETWORKS = {
  mainnet: {
    name: "Bitcoin Mainnet",
    rpcUrl: "https://btc.central.tigerwallet.io",
    explorer: "https://mempool.space",
    network: "bitcoin",
  },
  signet: {
    name: "Bitcoin Signet",
    rpcUrl: "https://signet.tigerwallet.io",
    explorer: "https://mempool.space/signet",
    network: "signet",
  },
  testnet: {
    name: "Bitcoin Testnet",
    rpcUrl: "https://testnet.tigerwallet.io",
    explorer: "https://mempool.space/testnet",
    network: "testnet",
  },
};

export default function BitcoinOrdinalsPage() {
  const [network, setNetwork] = useState<"mainnet" | "signet" | "testnet">("mainnet");
  const [address, setAddress] = useState<string>("");
  const [balance, setBalance] = useState<number>(0);
  const [utxos, setUtxos] = useState<UTXO[]>([]);
  const [ordinals, setOrdinals] = useState<Ordinal[]>([]);
  const [brc20Tokens, setBrc20Tokens] = useState<BRC20Token[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [showReceive, setShowReceive] = useState(false);
  const [activeTab, setActiveTab] = useState<"ordinals" | "brc20" | "utxo">("ordinals");
  const [inscribeForm, setInscribeForm] = useState({
    type: "text",
    content: "",
    recipient: "",
  });
  const [transferForm, setTransferForm] = useState({
    ordinalId: "",
    recipient: "",
  });

  // Connect wallet (mock for demo)
  const connectWallet = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      // Check for Bitcoin wallets
      const anyWindow = window as any;
      
      // Mock wallet for demo
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // Generate or load mock address
      const mockAddress = "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh";
      setAddress(mockAddress);
      setBalance(0.5); // Mock balance in BTC
      setUtxos([
        { txid: "abc123", vout: 0, amount: 0.1 },
        { txid: "def456", vout: 1, amount: 0.4 },
      ]);
      setOrdinals([
        {
          id: "ord1",
          number: 12345,
          collection: "Taproot Wizards",
          name: "Taproot Wizards #123",
          inscription: "abc123def456",
          contentType: "image/png",
          contentUrl: "/ordinals/wizard.png",
        },
        {
          id: "ord2",
          number: 67890,
          collection: "Bitcoin Punks",
          name: "Bitcoin Punk #456",
          inscription: "def789ghi012",
          contentType: "image/png",
        },
        {
          id: "ord3",
          number: 11111,
          collection: "Ordinals",
          name: " inscription #11111",
          inscription: "ins111111",
          contentType: "text/plain",
        },
      ]);
      setBrc20Tokens([
        { tick: "ordi", balance: 1000, available: 800, transferable: 200 },
        { tick: "pepe", balance: 50000, available: 45000, transferable: 5000 },
        { tick: "sats", balance: 1000000, available: 900000, transferable: 100000 },
      ]);
    } catch (err: any) {
      setError(err.message || "Failed to connect wallet");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Switch network
  const switchNetwork = useCallback((newNetwork: "mainnet" | "signet" | "testnet") => {
    setNetwork(newNetwork);
    // Would reconnect to new network
  }, []);

  // Inscribe (create ordinal)
  const inscribe = useCallback(async () => {
    if (!inscribeForm.content || !address) return;
    
    setIsLoading(true);
    setError(null);
    setSuccess(null);
    
    try {
      // Simulate inscription
      await new Promise(resolve => setTimeout(resolve, 3000));
      
      const newOrdinal: Ordinal = {
        id: "ord" + Date.now(),
        number: Math.floor(Math.random() * 100000),
        collection: "Custom",
        name: `${inscribeForm.type} #${Date.now()}`,
        inscription: "ins" + Date.now(),
        contentType: inscribeForm.type === "text" ? "text/plain" : "image/png",
      };
      
      setOrdinals(prev => [newOrdinal, ...prev]);
      setSuccess("Ordinal inscribed successfully!");
      setInscribeForm({ type: "text", content: "", recipient: "" });
    } catch (err: any) {
      setError(err.message || "Inscription failed");
    } finally {
      setIsLoading(false);
    }
  }, [inscribeForm, address]);

  // Transfer ordinal
  const transferOrdinal = useCallback(async () => {
    if (!transferForm.ordinalId || !transferForm.recipient) return;
    
    setIsLoading(true);
    setError(null);
    setSuccess(null);
    
    try {
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      setOrdinals(prev => prev.filter(o => o.id !== transferForm.ordinalId));
      setSuccess("Ordinal transferred successfully!");
      setTransferForm({ ordinalId: "", recipient: "" });
    } catch (err: any) {
      setError(err.message || "Transfer failed");
    } finally {
      setIsLoading(false);
    }
  }, [transferForm]);

  // Transfer BRC20
  const transferBRC20 = useCallback(async (tokenTick: string, amount: number) => {
    setIsLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 2000));
      setSuccess(`Transferred ${amount} ${tokenTick}`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-900 via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-orange-500 to-orange-600 rounded-xl flex items-center justify-center">
              <Bitcoin className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">Bitcoin Ordinals</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <select
              value={network}
              onChange={(e) => switchNetwork(e.target.value as any)}
              className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-sm"
            >
              {Object.entries(BITCOIN_NETWORKS).map(([key, config]) => (
                <option key={key} value={key}>{config.name}</option>
              ))}
            </select>
            
            {!address ? (
              <button
                onClick={connectWallet}
                disabled={isLoading}
                className="bg-orange-500 hover:bg-orange-600 px-6 py-2 rounded-lg font-medium flex items-center gap-2"
              >
                {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Wallet className="w-4 h-4" />}
                Connect Wallet
              </button>
            ) : (
              <button
                onClick={() => setShowReceive(true)}
                className="bg-gray-800 hover:bg-gray-700 px-4 py-2 rounded-lg"
              >
                Receive
              </button>
            )}
          </div>
        </header>

        {/* Error/Success */}
        {error && (
          <div className="mb-6 bg-red-500/10 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-500" />
            <p className="text-red-400">{error}</p>
          </div>
        )}
        
        {success && (
          <div className="mb-6 bg-green-500/10 border border-green-500/50 rounded-lg p-4 flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-500" />
            <p className="text-green-400">{success}</p>
          </div>
        )}

        {!address ? (
          // Not connected
          <div className="text-center py-20">
            <div className="w-24 h-24 mx-auto mb-6 bg-gray-800 rounded-full flex items-center justify-center">
              <Bitcoin className="w-12 h-12 text-orange-500" />
            </div>
            <h2 className="text-3xl font-bold mb-4">Bitcoin Ordinals Wallet</h2>
            <p className="text-gray-400 mb-8 max-w-2xl mx-auto">
              Store, send, and receive Bitcoin Ordinals (BRC-20 tokens, inscriptions, NFTs). 
              Trade ordinals directly from your wallet with full ownership.
            </p>
            <button
              onClick={connectWallet}
              className="bg-gradient-to-r from-orange-500 to-orange-600 hover:from-orange-600 hover:to-orange-700 px-8 py-4 rounded-xl font-bold text-lg flex items-center gap-3 mx-auto"
            >
              <Wallet className="w-5 h-5" />
              Connect Bitcoin Wallet
            </button>

            {/* Features */}
            <div className="grid md:grid-cols-3 gap-6 mt-16 text-left">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Flame className="w-10 h-10 text-orange-400 mb-4" />
                <h3 className="text-xl font-bold mb-2">Ordinals</h3>
                <p className="text-gray-400">Inscribe and trade Bitcoin NFTs with Ordinal protocol</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Layers className="w-10 h-10 text-orange-400 mb-4" />
                <h3 className="text-xl font-bold mb-2">BRC-20</h3>
                <p className="text-gray-400">Trade Bitcoin-native fungible tokens with BRC-20 standard</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Shield className="w-10 h-10 text-orange-400 mb-4" />
                <h3 className="text-xl font-bold mb-2">Secure</h3>
                <p className="text-gray-400">Non-custodial wallet with full control of your keys</p>
              </div>
            </div>
          </div>
        ) : (
          // Connected
          <div className="space-y-6">
            {/* Balance */}
            <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-bold">Bitcoin Balance</h2>
                <button className="bg-orange-500 hover:bg-orange-600 px-4 py-2 rounded-lg text-sm flex items-center gap-2">
                  <Send className="w-4 h-4" /> Send BTC
                </button>
              </div>
              <div className="text-4xl font-bold mb-2">{balance.toFixed(8)} BTC</div>
              <p className="text-gray-400">≈ ${(balance * 65000).toFixed(2)} USD</p>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 overflow-x-auto">
              {[
                { id: "ordinals", label: "Ordinals", icon: Flame },
                { id: "brc20", label: "BRC-20 Tokens", icon: Layers },
                { id: "utxo", label: "UTXO", icon: Hash },
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg whitespace-nowrap ${
                    activeTab === tab.id 
                      ? "bg-orange-500 text-white" 
                      : "bg-gray-800 text-gray-400 hover:text-white"
                  }`}
                >
                  <tab.icon className="w-4 h-4" />
                  {tab.label}
                </button>
              ))}
            </div>

            {/* Ordinals */}
            {activeTab === "ordinals" && (
              <div className="space-y-6">
                {/* Inscribe Form */}
                <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                  <h3 className="text-lg font-bold mb-4">Inscribe New Ordinal</h3>
                  <div className="space-y-4">
                    <div className="flex gap-2">
                      <button
                        onClick={() => setInscribeForm({ ...inscribeForm, type: "text" })}
                        className={`px-4 py-2 rounded-lg ${inscribeForm.type === "text" ? "bg-orange-500" : "bg-gray-700"}`}
                      >
                        Text
                      </button>
                      <button
                        onClick={() => setInscribeForm({ ...inscribeForm, type: "image" })}
                        className={`px-4 py-2 rounded-lg ${inscribeForm.type === "image" ? "bg-orange-500" : "bg-gray-700"}`}
                      >
                        Image
                      </button>
                    </div>
                    <textarea
                      value={inscribeForm.content}
                      onChange={(e) => setInscribeForm({ ...inscribeForm, content: e.target.value })}
                      placeholder={inscribeForm.type === "text" ? "Enter text to inscribe..." : "Enter image URL..."}
                      className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 h-24 focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                    <input
                      type="text"
                      value={inscribeForm.recipient}
                      onChange={(e) => setInscribeForm({ ...inscribeForm, recipient: e.target.value })}
                      placeholder="Recipient address (optional)"
                      className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-orange-500"
                    />
                    <button
                      onClick={inscribe}
                      disabled={isLoading || !inscribeForm.content}
                      className="w-full bg-orange-500 hover:bg-orange-600 py-3 rounded-lg font-bold flex items-center justify-center gap-2"
                    >
                      {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Flame className="w-5 h-5" />}
                      Inscribe (≈ 0.001 BTC)
                    </button>
                  </div>
                </div>

                {/* Ordinals List */}
                <div className="grid md:grid-cols-2 gap-4">
                  {ordinals.map((ordinal) => (
                    <div key={ordinal.id} className="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden">
                      <div className="h-40 bg-gray-700 flex items-center justify-center">
                        {ordinal.contentType.includes("image") ? (
                          <Image className="w-12 h-12 text-gray-500" />
                        ) : (
                          <Hash className="w-12 h-12 text-gray-500" />
                        )}
                      </div>
                      <div className="p-4">
                        <p className="font-bold text-sm">{ordinal.name}</p>
                        <p className="text-gray-400 text-xs mb-2">#{ordinal.number}</p>
                        <p className="text-gray-500 text-xs mb-3">{ordinal.collection}</p>
                        <div className="flex gap-2">
                          <button
                            onClick={() => setTransferForm({ ordinalId: ordinal.id, recipient: "" })}
                            className="flex-1 bg-gray-700 hover:bg-gray-600 py-1.5 rounded text-sm"
                          >
                            Transfer
                          </button>
                          <a
                            href={`${BITCOIN_NETWORKS[network].explorer}/inscription/${ordinal.inscription}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="bg-gray-700 hover:bg-gray-600 py-1.5 px-3 rounded text-sm"
                          >
                            <ExternalLink className="w-4 h-4" />
                          </a>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>

                {ordinals.length === 0 && (
                  <p className="text-center text-gray-400 py-8">No ordinals found</p>
                )}
              </div>
            )}

            {/* BRC-20 Tokens */}
            {activeTab === "brc20" && (
              <div className="space-y-3">
                {brc20Tokens.map((token) => (
                  <div key={token.tick} className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-orange-500/20 rounded-full flex items-center justify-center">
                          <Layers className="w-5 h-5 text-orange-400" />
                        </div>
                        <div>
                          <p className="font-bold">{token.tick.toUpperCase()}</p>
                          <p className="text-gray-400 text-sm">BRC-20</p>
                        </div>
                      </div>
                      <button
                        onClick={() => transferBRC20(token.tick, 10)}
                        className="bg-orange-500 hover:bg-orange-600 px-4 py-2 rounded-lg text-sm"
                      >
                        Transfer
                      </button>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <p className="text-gray-400">Balance</p>
                        <p className="font-bold">{token.balance.toLocaleString()}</p>
                      </div>
                      <div>
                        <p className="text-gray-400">Available</p>
                        <p className="font-bold">{token.available.toLocaleString()}</p>
                      </div>
                      <div>
                        <p className="text-gray-400">Transferable</p>
                        <p className="font-bold">{token.transferable.toLocaleString()}</p>
                      </div>
                    </div>
                  </div>
                ))}

                {brc20Tokens.length === 0 && (
                  <p className="text-center text-gray-400 py-8">No BRC-20 tokens found</p>
                )}
              </div>
            )}

            {/* UTXO */}
            {activeTab === "utxo" && (
              <div className="space-y-3">
                {utxos.map((utxo, i) => (
                  <div key={i} className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 flex items-center justify-between">
                    <div>
                      <p className="font-mono text-sm">{utxo.txid.slice(0, 10)}...</p>
                      <p className="text-gray-400 text-sm">vout: {utxo.vout}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-bold">{utxo.amount} BTC</p>
                      <p className="text-gray-400 text-sm">${(utxo.amount * 65000).toFixed(2)}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Transfer Modal */}
            {transferForm.ordinalId && (
              <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
                <div className="bg-gray-800 border border-gray-700 rounded-xl p-6 max-w-md w-full">
                  <h3 className="text-xl font-bold mb-4">Transfer Ordinal</h3>
                  <input
                    type="text"
                    value={transferForm.recipient}
                    onChange={(e) => setTransferForm({ ...transferForm, recipient: e.target.value })}
                    placeholder="Recipient address"
                    className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 mb-4 focus:outline-none focus:ring-2 focus:ring-orange-500"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={() => setTransferForm({ ordinalId: "", recipient: "" })}
                      className="flex-1 bg-gray-700 hover:bg-gray-600 py-3 rounded-lg"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={transferOrdinal}
                      disabled={isLoading || !transferForm.recipient}
                      className="flex-1 bg-orange-500 hover:bg-orange-600 py-3 rounded-lg font-bold flex items-center justify-center gap-2"
                    >
                      {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Send className="w-5 h-5" />}
                      Transfer
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Receive Modal */}
            {showReceive && (
              <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
                <div className="bg-gray-800 border border-gray-700 rounded-xl p-6 max-w-md w-full">
                  <h3 className="text-xl font-bold mb-4">Receive Bitcoin & Ordinals</h3>
                  <div className="bg-gray-900/50 rounded-lg p-4 mb-4">
                    <p className="text-gray-400 text-sm mb-2">Your Address</p>
                    <p className="font-mono text-sm break-all">{address}</p>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => navigator.clipboard.writeText(address)}
                      className="flex-1 bg-gray-700 hover:bg-gray-600 py-2 rounded-lg flex items-center justify-center gap-2"
                    >
                      <Copy className="w-4 h-4" /> Copy
                    </button>
                    <button
                      onClick={() => setShowReceive(false)}
                      className="flex-1 bg-orange-500 hover:bg-orange-600 py-2 rounded-lg"
                    >
                      Close
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
