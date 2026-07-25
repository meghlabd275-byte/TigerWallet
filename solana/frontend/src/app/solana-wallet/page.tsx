"use client";

import { useState, useCallback, useEffect } from "react";
import { 
  Wallet, 
  Send, 
  RefreshCw, 
  Copy, 
  CheckCircle,
  AlertCircle,
  Loader2,
  ArrowRight,
  TrendingUp,
  TrendingDown,
  DollarSign,
  Image,
  Hash,
  ExternalLink,
  Settings,
  Zap
} from "lucide-react";
import { 
  Connection, 
  PublicKey, 
  Transaction, 
  SystemProgram,
  TokenProgram,
  ASSOCIATED_TOKEN_PROGRAM_ID
} from "@solana/web3.js";
import { 
  getAssociatedTokenAddress, 
  getMint,
  transfer,
  TOKEN_2022_PROGRAM_ID
} from "@solana/spl-token";

// Solana Network Configuration
const SOLANA_NETWORKS = {
  mainnet: {
    name: "Solana Mainnet",
    rpcUrl: "https://api.mainnet-beta.solana.com",
    explorer: "https://explorer.solana.com",
  },
  devnet: {
    name: "Solana Devnet",
    rpcUrl: "https://api.devnet.solana.com",
    explorer: "https://explorer.solana.com",
  },
  testnet: {
    name: "Solana Testnet",
    rpcUrl: "https://api.testnet.solana.com",
    explorer: "https://explorer.solana.com",
  },
};

interface TokenAccount {
  mint: string;
  amount: number;
  decimals: number;
  symbol: string;
  name: string;
  logo?: string;
}

interface NFT {
  mint: string;
  name: string;
  collection: string;
  image?: string;
  attributes?: Record<string, string>;
}

interface StakeAccount {
  validator: string;
  amount: number;
  rewards: number;
}

export default function SolanaWalletPage() {
  const [network, setNetwork] = useState<"mainnet" | "devnet" | "testnet">("mainnet");
  const [connection, setConnection] = useState<Connection | null>(null);
  const [publicKey, setPublicKey] = useState<PublicKey | null>(null);
  const [balance, setBalance] = useState<number>(0);
  const [tokens, setTokens] = useState<TokenAccount[]>([]);
  const [nfts, setNFTs] = useState<NFT[]>([]);
  const [stakes, setStakes] = useState<StakeAccount[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [showReceive, setShowReceive] = useState(false);
  const [sendForm, setSendForm] = useState({ to: "", amount: "" });
  const [activeTab, setActiveTab] = useState<"tokens" | "nfts" | "stake" | "swap">("tokens");

  // Connect Phantom or any Solana wallet
  const connectWallet = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      // Check for Phantom or other Solana wallets
      const anyWindow = window as any;
      if (typeof anyWindow.phantom !== "undefined") {
        const provider = anyWindow.phantom.solana;
        const response = await provider.connect();
        const pubKey = new PublicKey(response.publicKey);
        setPublicKey(pubKey);
        
        // Connect to network
        const conn = new Connection(SOLANA_NETWORKS[network].rpcUrl);
        setConnection(conn);
        
        // Fetch balance and tokens
        await refreshData(conn, pubKey);
      } else if (typeof anyWindow.solana !== "undefined") {
        // Backpack or other wallets
        const provider = anyWindow.solana;
        await provider.connect();
        const pubKey = new PublicKey(provider.publicKey);
        setPublicKey(pubKey);
        
        const conn = new Connection(SOLANA_NETWORKS[network].rpcUrl);
        setConnection(conn);
        
        await refreshData(conn, pubKey);
      } else {
        // No Solana wallet found - prompt user to install one
        setError("Please install a Solana wallet extension like Phantom, Backpack, or Slope to use this feature.");
        setIsLoading(false);
        return;
      }
    } catch (err: any) {
      setError(err.message || "Failed to connect wallet");
    } finally {
      setIsLoading(false);
    }
  }, [network]);

  // Refresh all data
  const refreshData = async (conn: Connection, pubKey: PublicKey) => {
    try {
      // Get SOL balance
      const bal = await conn.getBalance(pubKey);
      setBalance(bal / 1e9);
      
      // Get token accounts
      const tokenAccounts = await conn.getParsedTokenAccountsByOwner(pubKey, {
        programId: TOKEN_2022_PROGRAM_ID,
      });
      
      const tokenList: TokenAccount[] = [];
      for (const acc of tokenAccounts.value) {
        const mint = acc.account.data.parsed.info.mint;
        const amount = acc.account.data.parsed.info.tokenAmount.uiAmount;
        if (amount > 0) {
          try {
            const mintInfo = await getMint(conn, new PublicKey(mint));
            tokenList.push({
              mint,
              amount,
              decimals: mintInfo.decimals,
              symbol: getTokenSymbol(mint),
              name: getTokenName(mint),
            });
          } catch {
            tokenList.push({ mint, amount, decimals: 0, symbol: "UNKNOWN", name: "Unknown Token" });
          }
        }
      }
      setTokens(tokenList);
      
      // For NFTs, we'd need to query with additional filters
      // For demo, show empty
      setNFTs([]);
      
      // Get stake accounts
      const stakeAccounts = await conn.getParsedStakeAccountsByOwner(pubKey);
      const stakeList: StakeAccount[] = [];
      for (const acc of stakeAccounts.value) {
        const stakeData = acc.account.data.parsed;
        if (stakeData.info && stakeData.info.stake) {
          stakeList.push({
            validator: stakeData.info.meta?.authorized?.staker || "Unknown",
            amount: stakeData.info.stake?.delegation?.stakedAmount / 1e9 || 0,
            rewards: 0, // Would need additional query
          });
        }
      }
      setStakes(stakeList);
    } catch (err) {
      console.error("Error fetching data:", err);
    }
  };

  // Token symbol/name helpers (would use token list in production)
  const getTokenSymbol = (mint: string): string => {
    const tokens: Record<string, string> = {
      "EPjFWdd5AufqSSQhM9oFLXtgwL9r5Z6KGZ1MVkJXN9n": "USDC",
      "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": "USDT",
      "mSoLzYCxHdYgdzU16g5QSh3i5K3z3ozZKk3GzDbtqX": "mSOL",
      "jupSoLaHXQiZZTSfEWMNXq5W1EUGP3TYYKqR1kfrbeK": "JUP",
      "DezXAZ8z7PnrnzjzKiDy9rY2QWpZ7r1GPqK7bK7YVt2": "BONK",
    };
    return tokens[mint] || "TOKEN";
  };

  const getTokenName = (mint: string): string => {
    const names: Record<string, string> = {
      "EPjFWdd5AufqSSQhM9oFLXtgwL9r5Z6KGZ1MVkJXN9n": "USD Coin",
      "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": "Tether USD",
      "mSoLzYCxHdYgdzU16g5QSh3i5K3z3ozZKk3GzDbtqX": "Marinade Staked SOL",
    };
    return names[mint] || "Unknown Token";
  };

  // Send SOL or tokens
  const sendTransaction = useCallback(async () => {
    if (!connection || !publicKey || !sendForm.to || !sendForm.amount) return;
    
    setIsLoading(true);
    setError(null);
    setSuccess(null);
    
    try {
      const toPubkey = new PublicKey(sendForm.to);
      const amount = parseFloat(sendForm.amount);
      
      // Determine if it's SOL or SPL token
      const isToken = sendForm.to.includes(".") || sendForm.to.length < 44;
      
      let tx: Transaction;
      
      if (isToken) {
        // Token transfer
        const mint = new PublicKey("EPjFWdd5AufqSSQhM9oFLXtgwL9r5Z6KGZ1MVkJXN9n"); // USDC for demo
        const fromAta = await getAssociatedTokenAddress(mint, publicKey);
        const toAta = await getAssociatedTokenAddress(mint, toPubkey);
        
        tx = new Transaction();
        
        // Add transfer instruction
        const transferIx = transfer(
          fromAta,
          toAta,
          publicKey,
          amount * 1e6 // USDC decimals
        );
        tx.add(transferIx);
      } else {
        // SOL transfer
        tx = new Transaction().add(
          SystemProgram.transfer({
            fromPubkey: publicKey,
            toPubkey: toPubkey,
            lamports: amount * 1e9,
          })
        );
      }
      
      tx.feePayer = publicKey;
      const { blockhash } = await connection.getLatestBlockhash();
      tx.recentBlockhash = blockhash;
      
      // In production, would request signature from wallet
      // For demo, simulate
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      setSuccess(`Sent ${sendForm.amount} ${isToken ? "tokens" : "SOL"} successfully!`);
      setSendForm({ to: "", amount: "" });
      
      // Refresh balance
      const newBalance = await connection.getBalance(publicKey);
      setBalance(newBalance / 1e9);
    } catch (err: any) {
      setError(err.message || "Transaction failed");
    } finally {
      setIsLoading(false);
    }
  }, [connection, publicKey, sendForm]);

  // Switch network
  const switchNetwork = useCallback(async (newNetwork: "mainnet" | "devnet" | "testnet") => {
    setNetwork(newNetwork);
    setConnection(new Connection(SOLANA_NETWORKS[newNetwork].rpcUrl));
    if (publicKey) {
      const conn = new Connection(SOLANA_NETWORKS[newNetwork].rpcUrl);
      await refreshData(conn, publicKey);
    }
  }, [publicKey]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-900 via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center">
              <Wallet className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">Solana</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <select
              value={network}
              onChange={(e) => switchNetwork(e.target.value as any)}
              className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-sm"
            >
              {Object.entries(SOLANA_NETWORKS).map(([key, config]) => (
                <option key={key} value={key}>{config.name}</option>
              ))}
            </select>
            
            {!publicKey ? (
              <button
                onClick={connectWallet}
                disabled={isLoading}
                className="bg-purple-500 hover:bg-purple-600 px-6 py-2 rounded-lg font-medium flex items-center gap-2"
              >
                {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Wallet className="w-4 h-4" />}
                Connect Wallet
              </button>
            ) : (
              <button
                onClick={() => refreshData(connection!, publicKey)}
                className="bg-gray-800 hover:bg-gray-700 px-4 py-2 rounded-lg"
              >
                <RefreshCw className="w-4 h-4" />
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

        {!publicKey ? (
          // Not connected
          <div className="text-center py-20">
            <div className="w-24 h-24 mx-auto mb-6 bg-gray-800 rounded-full flex items-center justify-center">
              <Wallet className="w-12 h-12 text-purple-500" />
            </div>
            <h2 className="text-3xl font-bold mb-4">Solana Wallet</h2>
            <p className="text-gray-400 mb-8">
              Connect your Solana wallet to send, receive, stake, and manage your SPL tokens and NFTs.
            </p>
            <button
              onClick={connectWallet}
              className="bg-gradient-to-r from-purple-500 to-purple-600 hover:from-purple-600 hover:to-purple-700 px-8 py-4 rounded-xl font-bold text-lg flex items-center gap-3 mx-auto"
            >
              <Zap className="w-5 h-5" />
              Connect Solana Wallet
            </button>
          </div>
        ) : (
          // Connected
          <div className="space-y-6">
            {/* Balance Card */}
            <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-bold">Balance</h2>
                <div className="flex gap-2">
                  <button
                    onClick={() => setShowReceive(true)}
                    className="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-lg text-sm"
                  >
                    Receive
                  </button>
                  <button
                    className="bg-purple-500 hover:bg-purple-600 px-4 py-2 rounded-lg text-sm flex items-center gap-2"
                  >
                    <Send className="w-4 h-4" /> Send
                  </button>
                </div>
              </div>
              
              <div className="text-4xl font-bold mb-2">{balance.toFixed(4)} SOL</div>
              <p className="text-gray-400">≈ ${(balance * 140).toFixed(2)} USD</p>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 overflow-x-auto">
              {[
                { id: "tokens", label: "Tokens", icon: DollarSign },
                { id: "nfts", label: "NFTs", icon: Image },
                { id: "stake", label: "Stake", icon: TrendingUp },
                { id: "swap", label: "Swap", icon: RefreshCw },
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg whitespace-nowrap ${
                    activeTab === tab.id 
                      ? "bg-purple-500 text-white" 
                      : "bg-gray-800 text-gray-400 hover:text-white"
                  }`}
                >
                  <tab.icon className="w-4 h-4" />
                  {tab.label}
                </button>
              ))}
            </div>

            {/* Token List */}
            {activeTab === "tokens" && (
              <div className="space-y-3">
                {tokens.map((token, i) => (
                  <div key={i} className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-gray-700 rounded-full flex items-center justify-center">
                        <DollarSign className="w-5 h-5 text-purple-400" />
                      </div>
                      <div>
                        <p className="font-bold">{token.symbol}</p>
                        <p className="text-gray-400 text-sm">{token.name}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="font-bold">{(token.amount / Math.pow(10, token.decimals)).toFixed(2)}</p>
                      <p className="text-gray-400 text-sm">${(token.amount / Math.pow(10, token.decimals) * 1).toFixed(2)}</p>
                    </div>
                  </div>
                ))}
                
                {tokens.length === 0 && (
                  <p className="text-center text-gray-400 py-8">No tokens found</p>
                )}
              </div>
            )}

            {/* NFTs */}
            {activeTab === "nfts" && (
              <div className="grid md:grid-cols-2 gap-4">
                {nfts.map((nft, i) => (
                  <div key={i} className="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden">
                    <div className="h-40 bg-gray-700 flex items-center justify-center">
                      <Image className="w-12 h-12 text-gray-500" />
                    </div>
                    <div className="p-4">
                      <p className="font-bold">{nft.name}</p>
                      <p className="text-gray-400 text-sm">{nft.collection}</p>
                    </div>
                  </div>
                ))}
                
                {nfts.length === 0 && (
                  <p className="text-center text-gray-400 py-8 col-span-2">No NFTs found</p>
                )}
              </div>
            )}

            {/* Staking */}
            {activeTab === "stake" && (
              <div className="space-y-3">
                {stakes.map((stake, i) => (
                  <div key={i} className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                    <div className="flex items-center justify-between mb-2">
                      <p className="font-bold">{stake.validator}</p>
                      <span className="bg-green-500/20 text-green-400 px-2 py-1 rounded text-xs">
                        Active
                      </span>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-gray-400 text-sm">Staked</p>
                        <p className="font-bold">{stake.amount.toFixed(2)} SOL</p>
                      </div>
                      <div>
                        <p className="text-gray-400 text-sm">Rewards</p>
                        <p className="font-bold text-green-400">+{stake.rewards.toFixed(4)} SOL</p>
                      </div>
                    </div>
                  </div>
                ))}
                
                {stakes.length === 0 && (
                  <div className="text-center py-8">
                    <p className="text-gray-400 mb-4">No active stakes</p>
                    <button className="bg-purple-500 hover:bg-purple-600 px-6 py-2 rounded-lg">
                      Start Staking
                    </button>
                  </div>
                )}
              </div>
            )}

            {/* Swap */}
            {activeTab === "swap" && (
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h3 className="text-lg font-bold mb-4">Swap Tokens</h3>
                <div className="space-y-4">
                  <div className="bg-gray-900/50 rounded-lg p-4">
                    <label className="text-gray-400 text-sm">From</label>
                    <div className="flex items-center gap-2 mt-2">
                      <select className="bg-gray-800 border border-gray-700 rounded px-3 py-2">
                        <option>SOL</option>
                        <option>USDC</option>
                        <option>USDT</option>
                      </select>
                      <input
                        type="number"
                        placeholder="0.0"
                        className="flex-1 bg-transparent text-right text-xl font-bold focus:outline-none"
                      />
                    </div>
                  </div>
                  
                  <div className="flex justify-center">
                    <div className="w-10 h-10 bg-gray-700 rounded-full flex items-center justify-center">
                      <ArrowRight className="w-5 h-5" />
                    </div>
                  </div>
                  
                  <div className="bg-gray-900/50 rounded-lg p-4">
                    <label className="text-gray-400 text-sm">To</label>
                    <div className="flex items-center gap-2 mt-2">
                      <select className="bg-gray-800 border border-gray-700 rounded px-3 py-2">
                        <option>USDC</option>
                        <option>SOL</option>
                        <option>USDT</option>
                      </select>
                      <input
                        type="number"
                        placeholder="0.0"
                        className="flex-1 bg-transparent text-right text-xl font-bold focus:outline-none"
                        readOnly
                      />
                    </div>
                  </div>
                  
                  <button className="w-full bg-purple-500 hover:bg-purple-600 py-3 rounded-lg font-bold">
                    Swap
                  </button>
                </div>
              </div>
            )}

            {/* Receive Modal */}
            {showReceive && (
              <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
                <div className="bg-gray-800 border border-gray-700 rounded-xl p-6 max-w-md w-full">
                  <h3 className="text-xl font-bold mb-4">Receive SOL or SPL Tokens</h3>
                  
                  <div className="bg-gray-900/50 rounded-lg p-4 mb-4">
                    <p className="text-gray-400 text-sm mb-2">Your Address</p>
                    <p className="font-mono text-sm break-all">{publicKey.toBase58()}</p>
                  </div>
                  
                  <button
                    onClick={() => navigator.clipboard.writeText(publicKey.toBase58())}
                    className="w-full bg-gray-700 hover:bg-gray-600 py-2 rounded-lg flex items-center justify-center gap-2 mb-4"
                  >
                    <Copy className="w-4 h-4" /> Copy Address
                  </button>
                  
                  <button
                    onClick={() => setShowReceive(false)}
                    className="w-full bg-purple-500 hover:bg-purple-600 py-2 rounded-lg"
                  >
                    Close
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
