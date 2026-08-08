'use client';

import React, { useState, useEffect } from 'react';
import { useWallet } from '../wallet';
import { useTheme } from '../components/ThemeProvider';
import { parseEther, parseUnits, formatEther } from 'viem';

// ================================================================================
// Types & Interfaces
// ================================================================================

interface UserOperation {
  sender: string;
  nonce: string;
  initCode: string;
  callData: string;
  callGasLimit: string;
  verificationGasLimit: string;
  preVerificationGas: string;
  maxFeePerGas: string;
  maxPriorityFeePerGas: string;
  paymasterAndData: string;
  signature: string;
}

interface GasEstimate {
  callGasLimit: string;
  verificationGasLimit: string;
  preVerificationGas: string;
  maxFeePerGas: string;
  maxPriorityFeePerGas: string;
  gasPrice: string;
}

interface SmartAccount {
  address: string;
  owner: string;
  factory: string;
  nonce: string;
  isDeployed: boolean;
  balance: string;
}

interface PaymasterSponsorship {
  enabled: boolean;
  sponsorAddress: string;
  validUntil: number;
  signature: string;
}

interface BundleStatus {
  bundleHash: string;
  userOpHashes: string[];
  status: 'pending' | 'submitted' | 'confirmed' | 'failed';
  gasUsed?: string;
  transactionHash?: string;
  blockNumber?: number;
  error?: string;
}

// ================================================================================
// API Service
// ================================================================================

const API_BASE_URL = process.env.NEXT_PUBLIC_AA_API_URL || 'http://localhost:8080/v1';

class AccountAbstractionAPI {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  async getEntryPoints(chainId: string): Promise<string[]> {
    return this.request(`/chains/${chainId}/entry-points`);
  }

  async estimateGas(userOp: Partial<UserOperation>): Promise<GasEstimate> {
    return this.request('/rpc/eth_estimateGas', {
      method: 'POST',
      body: JSON.stringify(userOp),
    });
  }

  async sendUserOperation(userOp: UserOperation): Promise<{ hash: string }> {
    return this.request('/rpc/eth_sendUserOperation', {
      method: 'POST',
      body: JSON.stringify(userOp),
    });
  }

  async getUserOperationReceipt(hash: string): Promise<any> {
    return this.request(`/rpc/eth_getUserOperationReceipt/${hash}`);
  }

  async createSmartWallet(owner: string, salt?: string): Promise<{ address: string }> {
    return this.request('/wallet', {
      method: 'POST',
      body: JSON.stringify({ owner, salt }),
    });
  }

  async getSmartWalletInfo(sender: string): Promise<SmartAccount> {
    return this.request(`/wallet/${sender}`);
  }

  async getPaymasterSponsorship(userOp: Partial<UserOperation>): Promise<PaymasterSponsorship> {
    return this.request('/paymaster/sponsorship', {
      method: 'POST',
      body: JSON.stringify(userOp),
    });
  }
}

const aaAPI = new AccountAbstractionAPI();

// ================================================================================
// Main Component
// ================================================================================

export default function AccountAbstractionPage() {
  const { address, isConnected, chainId } = useWallet();
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [smartAccount, setSmartAccount] = useState<SmartAccount | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  const [to, setTo] = useState('');
  const [value, setValue] = useState('');
  const [data, setData] = useState('0x');
  
  const [gasEstimate, setGasEstimate] = useState<GasEstimate | null>(null);
  const [usePaymaster, setUsePaymaster] = useState(true);
  const [sponsorship, setSponsorship] = useState<PaymasterSponsorship | null>(null);
  
  const [bundleStatus, setBundleStatus] = useState<BundleStatus | null>(null);
  const [entryPoints, setEntryPoints] = useState<string[]>([]);

  useEffect(() => {
    if (isConnected && chainId) {
      loadEntryPoints();
      if (address) {
        loadSmartWalletInfo(address);
      }
    }
  }, [isConnected, chainId, address]);

  const loadEntryPoints = async () => {
    try {
      const points = await aaAPI.getEntryPoints(chainId?.toString() || '1');
      setEntryPoints(points);
    } catch (err) {
      console.error('Failed to load entry points:', err);
    }
  };

  const loadSmartWalletInfo = async (ownerAddress: string) => {
    setIsLoading(true);
    try {
      const wallet = await aaAPI.getSmartWalletInfo(ownerAddress);
      setSmartAccount(wallet);
    } catch (err) {
      setSmartAccount(null);
    } finally {
      setIsLoading(false);
    }
  };

  const createSmartWallet = async () => {
    if (!address) return;
    
    setIsCreating(true);
    setError(null);
    
    try {
      const result = await aaAPI.createSmartWallet(address);
      setSmartAccount({
        address: result.address,
        owner: address,
        factory: entryPoints[0] || '',
        nonce: '0',
        isDeployed: false,
        balance: '0',
      });
      setSuccess('Smart wallet created successfully!');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsCreating(false);
    }
  };

  const estimateGas = async () => {
    if (!smartAccount || !to) return;
    
    setIsLoading(true);
    try {
      const userOp: Partial<UserOperation> = {
        sender: smartAccount.address,
        nonce: smartAccount.nonce,
        callData: data || '0x',
        callGasLimit: '0',
        verificationGasLimit: '0',
        preVerificationGas: '0',
        maxFeePerGas: '0',
        maxPriorityFeePerGas: '0',
      };

      const estimate = await aaAPI.estimateGas(userOp);
      setGasEstimate(estimate);

      if (usePaymaster) {
        const sponsor = await aaAPI.getPaymasterSponsorship(userOp);
        setSponsorship(sponsor);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const sendTransaction = async () => {
    if (!smartAccount || !to || !gasEstimate) return;
    
    setIsLoading(true);
    setError(null);
    setSuccess(null);
    setBundleStatus(null);

    try {
      const userOp: UserOperation = {
        sender: smartAccount.address,
        nonce: smartAccount.nonce,
        initCode: '0x',
        callData: data || '0x',
        callGasLimit: gasEstimate.callGasLimit,
        verificationGasLimit: gasEstimate.verificationGasLimit,
        preVerificationGas: gasEstimate.preVerificationGas,
        maxFeePerGas: gasEstimate.maxFeePerGas,
        maxPriorityFeePerGas: gasEstimate.maxPriorityFeePerGas,
        paymasterAndData: sponsorship?.enabled ? sponsorship.signature : '0x',
        signature: '0x',
      };

      const result = await aaAPI.sendUserOperation(userOp);
      
      setBundleStatus({
        bundleHash: result.hash,
        userOpHashes: [result.hash],
        status: 'pending',
      });
      
      setSuccess(`Transaction submitted! Hash: ${result.hash}`);
      pollForReceipt(result.hash);
      
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const pollForReceipt = async (hash: string, maxAttempts: number = 30) => {
    for (let i = 0; i < maxAttempts; i++) {
      try {
        const receipt = await aaAPI.getUserOperationReceipt(hash);
        if (receipt) {
          setBundleStatus({
            bundleHash: hash,
            userOpHashes: [hash],
            status: receipt.success ? 'confirmed' : 'failed',
            gasUsed: receipt.gasUsed?.toString(),
            transactionHash: receipt.transactionHash,
            blockNumber: receipt.blockNumber,
          });
          return;
        }
      } catch (err) {}
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  };

  const formatAddress = (addr: string): string => {
    return `${addr?.slice(0, 6)}...${addr?.slice(-4)}`;
  };

  const formatGwei = (wei: string): string => {
    try {
      const gwei = parseUnits(wei || '0', -9);
      return formatEther(gwei);
    } catch {
      return '0';
    }
  };

  if (!isConnected) {
    return (
      <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
        <div className="max-w-4xl mx-auto">
          <h1 className={`text-4xl font-bold mb-8 ${isDark ? 'text-white' : 'text-slate-900'}`}>Account Abstraction</h1>
          <div className={`rounded-2xl p-8 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <p className={`text-lg ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>Please connect your wallet to use Account Abstraction features.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>Account Abstraction</h1>
          <p className={isDark ? "text-slate-400" : "text-slate-500"}>Smart wallet powered by ERC-4337</p>
        </div>

        <div className={`rounded-xl p-4 mb-6 border ${isDark ? 'bg-slate-800/50 border-slate-700' : 'bg-white border-slate-200'}`}>
          <div className="flex items-center justify-between">
            <div>
              <span className={`text-sm ${isDark ? "text-slate-400" : "text-slate-500"}`}>Entry Point:</span>
              <span className={`ml-2 font-mono text-sm ${isDark ? "text-white" : "text-slate-900"}`}>
                {entryPoints[0] || '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              <span className="text-green-400 text-sm">Bundler Active</span>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <div className={`rounded-2xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Smart Wallet</h2>
            
            {smartAccount ? (
              <div className="space-y-4">
                <div className={`flex items-center justify-between p-3 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
                  <span className={isDark ? "text-slate-400" : "text-slate-500"}>Address</span>
                  <span className={`font-mono text-sm ${isDark ? "text-white" : "text-slate-900"}`}>{formatAddress(smartAccount.address)}</span>
                </div>
                <div className={`flex items-center justify-between p-3 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
                  <span className={isDark ? "text-slate-400" : "text-slate-500"}>Nonce</span>
                  <span className={isDark ? "text-white" : "text-slate-900"}>{smartAccount.nonce}</span>
                </div>
                <div className={`flex items-center justify-between p-3 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
                  <span className={isDark ? "text-slate-400" : "text-slate-500"}>Status</span>
                  <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                    smartAccount.isDeployed ? 'bg-green-500/20 text-green-400' : 'bg-yellow-500/20 text-yellow-400'
                  }`}>
                    {smartAccount.isDeployed ? 'Deployed' : 'Not Deployed'}
                  </span>
                </div>
              </div>
            ) : (
              <div className="text-center py-8">
                <p className={`mb-4 ${isDark ? "text-slate-400" : "text-slate-500"}`}>No smart wallet found. Create one to get started.</p>
                <button
                  onClick={createSmartWallet}
                  disabled={isCreating}
                  className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium transition-colors"
                >
                  {isCreating ? 'Creating...' : 'Create Smart Wallet'}
                </button>
              </div>
            )}
          </div>

          <div className={`rounded-2xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Gas Estimation</h2>
            
            {gasEstimate ? (
              <div className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className={isDark ? "text-slate-400" : "text-slate-500"}>Call Gas Limit</span>
                  <span className={isDark ? "text-white" : "text-slate-900"}>{parseInt(gasEstimate.callGasLimit).toLocaleString()}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className={isDark ? "text-slate-400" : "text-slate-500"}>Verification Gas</span>
                  <span className={isDark ? "text-white" : "text-slate-900"}>{parseInt(gasEstimate.verificationGasLimit).toLocaleString()}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className={isDark ? "text-slate-400" : "text-slate-500"}>Max Fee</span>
                  <span className={isDark ? "text-white" : "text-slate-900"}>{formatGwei(gasEstimate.maxFeePerGas)} ETH</span>
                </div>
                
                {sponsorship?.enabled && (
                  <div className="mt-4 p-3 bg-green-500/10 border border-green-500/30 rounded-lg">
                    <div className="flex items-center gap-2 text-green-400">
                      <span className="font-medium">Sponsored by Paymaster</span>
                    </div>
                    <p className="text-green-400/70 text-sm mt-1">No gas fees required!</p>
                  </div>
                )}
              </div>
            ) : (
              <p className={`text-sm ${isDark ? "text-slate-400" : "text-slate-500"}`}>Enter transaction details and estimate gas</p>
            )}
          </div>
        </div>

        <div className={`rounded-2xl p-6 border mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Send Transaction</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
            <div>
              <label className={`block text-sm mb-2 ${isDark ? "text-slate-400" : "text-slate-500"}`}>To Address</label>
              <input
                type="text"
                value={to}
                onChange={(e) => setTo(e.target.value)}
                placeholder="0x..."
                className={`w-full border rounded-lg px-4 py-3 placeholder-slate-500 focus:outline-none focus:border-blue-500 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`}
              />
            </div>
            <div>
              <label className={`block text-sm mb-2 ${isDark ? "text-slate-400" : "text-slate-500"}`}>Amount (ETH)</label>
              <input
                type="text"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="0.0"
                className={`w-full border rounded-lg px-4 py-3 placeholder-slate-500 focus:outline-none focus:border-blue-500 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`}
              />
            </div>
            <div>
              <label className={`block text-sm mb-2 ${isDark ? "text-slate-400" : "text-slate-500"}`}>Data</label>
              <input
                type="text"
                value={data}
                onChange={(e) => setData(e.target.value)}
                placeholder="0x..."
                className={`w-full border rounded-lg px-4 py-3 placeholder-slate-500 focus:outline-none focus:border-blue-500 ${isDark ? 'bg-slate-700 border-slate-600 text-white' : 'bg-white border-slate-300 text-slate-900'}`}
              />
            </div>
          </div>

          <div className="flex items-center gap-3 mb-4">
            <button
              onClick={() => setUsePaymaster(!usePaymaster)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${usePaymaster ? 'bg-blue-600' : 'bg-slate-600'}`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${usePaymaster ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
            <span className={isDark ? "text-slate-300" : "text-slate-700"}>Use Paymaster (Gasless)</span>
          </div>

          <div className="flex gap-4">
            <button
              onClick={estimateGas}
              disabled={!smartAccount || !to || isLoading}
              className="bg-slate-600 hover:bg-slate-500 disabled:bg-slate-700 text-white px-6 py-3 rounded-xl font-medium transition-colors"
            >
              {isLoading ? 'Loading...' : 'Estimate Gas'}
            </button>
            <button
              onClick={sendTransaction}
              disabled={!smartAccount || !gasEstimate || isLoading}
              className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium transition-colors"
            >
              {isLoading ? 'Sending...' : 'Send Transaction'}
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-xl">
            <div className="flex items-center gap-2 text-red-400">
              <span>{error}</span>
            </div>
          </div>
        )}

        {success && (
          <div className="mb-6 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
            <div className="flex items-center gap-2 text-green-400">
              <span>{success}</span>
            </div>
          </div>
        )}

        {bundleStatus && (
          <div className={`rounded-2xl p-6 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Transaction Status</h2>
            <div className="flex items-center gap-4">
              <div className={`w-4 h-4 rounded-full ${
                bundleStatus.status === 'confirmed' ? 'bg-green-500' :
                bundleStatus.status === 'failed' ? 'bg-red-500' :
                bundleStatus.status === 'submitted' ? 'bg-blue-500' :
                'bg-yellow-500 animate-pulse'
              }`}></div>
              <span className={`font-medium capitalize ${isDark ? "text-white" : "text-slate-900"}`}>{bundleStatus.status}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
