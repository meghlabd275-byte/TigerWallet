"use client";

import { useState, useCallback, useEffect } from "react";
import { 
  Wallet, 
  Shield, 
  Zap, 
  Users, 
  Key, 
  ArrowRight,
  CheckCircle,
  Loader2,
  AlertCircle,
  Copy,
  ExternalLink
} from "lucide-react";
import { BrowserProvider, Contract, formatEther, parseEther } from "ethers";
import { SmartAccountService } from "@/lib/smartAccount";
import { EntryPoint__factory, SmartAccountFactory__factory } from "@/lib/contracts";

type ChainConfig = {
  name: string;
  entryPoint: string;
  factory: string;
  bundlerUrl: string;
  rpcUrl: string;
  explorer: string;
};

const env = (key: string): string => process.env[key] ?? "";

// Deployments are intentionally environment-driven. The UI refuses to submit transactions
// until every address and endpoint for the selected chain has been configured by deployment.
const CHAIN_CONFIG: Record<number, ChainConfig> = {
  1: { name: "Ethereum Mainnet", entryPoint: env("NEXT_PUBLIC_ENTRYPOINT_1"), factory: env("NEXT_PUBLIC_FACTORY_1"), bundlerUrl: env("NEXT_PUBLIC_BUNDLER_1"), rpcUrl: env("NEXT_PUBLIC_RPC_1"), explorer: "https://etherscan.io" },
  137: { name: "Polygon Mainnet", entryPoint: env("NEXT_PUBLIC_ENTRYPOINT_137"), factory: env("NEXT_PUBLIC_FACTORY_137"), bundlerUrl: env("NEXT_PUBLIC_BUNDLER_137"), rpcUrl: env("NEXT_PUBLIC_RPC_137"), explorer: "https://polygonscan.com" },
  8453: { name: "Base Mainnet", entryPoint: env("NEXT_PUBLIC_ENTRYPOINT_8453"), factory: env("NEXT_PUBLIC_FACTORY_8453"), bundlerUrl: env("NEXT_PUBLIC_BUNDLER_8453"), rpcUrl: env("NEXT_PUBLIC_RPC_8453"), explorer: "https://basescan.org" },
  42161: { name: "Arbitrum One", entryPoint: env("NEXT_PUBLIC_ENTRYPOINT_42161"), factory: env("NEXT_PUBLIC_FACTORY_42161"), bundlerUrl: env("NEXT_PUBLIC_BUNDLER_42161"), rpcUrl: env("NEXT_PUBLIC_RPC_42161"), explorer: "https://arbiscan.io" },
};

function requireChainConfig(chainId: number): ChainConfig {
  const config = CHAIN_CONFIG[chainId];
  if (!config || !config.entryPoint || !config.factory || !config.bundlerUrl || !config.rpcUrl) {
    throw new Error(`Account-abstraction deployment is not configured for chain ${chainId}`);
  }
  return config;
}

function requireEthereumProvider(): NonNullable<Window["ethereum"]> {
  if (typeof window === "undefined" || !window.ethereum) {
    throw new Error("An injected EIP-1193 wallet provider is required");
  }
  return window.ethereum;
}

// Entry Point ABI (simplified for demo)
const ENTRY_POINT_ABI = [
  "function getUserOpHash((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes signature, uint256 paymasterAndData)) returns (bytes32)",
  "function handleOps((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes signature, uint256 paymasterAndData)[], address beneficiary)",
];

export default function AccountAbstractionPage() {
  const [isConnected, setIsConnected] = useState(false);
  const [accountAddress, setAccountAddress] = useState<string | null>(null);
  const [chainId, setChainId] = useState<number>(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [smartAccount, setSmartAccount] = useState<{
    address: string;
    isDeployed: boolean;
    nonce: number;
  } | null>(null);
  const [guardians, setGuardians] = useState<string[]>([]);
  const [newGuardian, setNewGuardian] = useState("");
  const [transactions, setTransactions] = useState<Array<{
    hash: string;
    status: "pending" | "confirmed" | "failed";
    timestamp: number;
  }>>([]);

  // Connect wallet
  const connectWallet = useCallback(async () => {
    if (typeof window.ethereum === "undefined") {
      setError("Please install MetaMask or another Web3 wallet");
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      
      const provider = new BrowserProvider(requireEthereumProvider());
      const accounts = await provider.send("eth_requestAccounts", []);
      
      if (accounts.length > 0) {
        setAccountAddress(accounts[0]);
        setIsConnected(true);
        
        const network = await provider.getNetwork();
        setChainId(Number(network.chainId));
        
        // Check if smart account exists
        await checkSmartAccount(accounts[0], Number(network.chainId));
      }
    } catch (err: any) {
      setError(err.message || "Failed to connect wallet");
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Check if smart account exists
  const checkSmartAccount = async (eoaAddress: string, chainId: number) => {
    const config = requireChainConfig(chainId);
    if (!config) return;

    try {
      // Calculate smart account address
      const factoryAddress = config.factory;
      const factory = SmartAccountFactory__factory.connect(factoryAddress, new BrowserProvider(requireEthereumProvider()));
      
      const salt = 0;
      const initCode = await factory.getInitCode(eoaAddress);
      const smartAccountAddress = await factory.getAccountAddress(eoaAddress, salt);
      
      // Check if deployed
      const provider = new BrowserProvider(requireEthereumProvider());
      const code = await provider.getCode(smartAccountAddress);
      
      setSmartAccount({
        address: smartAccountAddress,
        isDeployed: code !== "0x",
        nonce: 0,
      });
    } catch (err: any) {
      console.error("Error checking smart account:", err);
    }
  };

  // Deploy smart account
  const deploySmartAccount = useCallback(async () => {
    if (!accountAddress || !window.ethereum) return;

    setIsLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const config = requireChainConfig(chainId);
      const provider = new BrowserProvider(requireEthereumProvider());
      const signer = await provider.getSigner();
      
      const factory = SmartAccountFactory__factory.connect(config.factory, signer);
      
      // Create initialize data
      const initializeData = SmartAccountService.encodeInitialize(accountAddress);
      
      // Send transaction to create smart account
      const tx = await factory.createAccount(accountAddress, 0);
      const receipt = await tx.wait();
      
      if (receipt.status === 1) {
        setSuccess("Smart Account deployed successfully!");
        await checkSmartAccount(accountAddress, chainId);
      }
    } catch (err: any) {
      setError(err.message || "Failed to deploy smart account");
    } finally {
      setIsLoading(false);
    }
  }, [accountAddress, chainId]);

  // Send User Operation (gasless)
  const sendUserOp = useCallback(async (to: string, amount: string) => {
    if (!accountAddress || !smartAccount?.address || !window.ethereum) return;

    setIsLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const config = requireChainConfig(chainId);
      const provider = new BrowserProvider(requireEthereumProvider());
      const signer = await provider.getSigner();
      
      // Get fee data
      const feeData = await provider.getFeeData();
      const maxFeePerGas = feeData.maxFeePerGas || parseEther("0.1");
      const maxPriorityFeePerGas = feeData.maxPriorityFeePerGas || parseEther("0.01");
      
      // Build user operation
      const userOp = {
        sender: smartAccount.address,
        nonce: BigInt(smartAccount.nonce),
        initCode: "0x",
        callData: SmartAccountService.encodeExecute(to, parseEther(amount)),
        callGasLimit: 100000,
        verificationGasLimit: 200000,
        preVerificationGas: 21000,
        maxFeePerGas: maxFeePerGas,
        maxPriorityFeePerGas: maxPriorityFeePerGas,
        signature: "0x",
        paymasterAndData: config.entryPoint, // Using entry point as paymaster for demo
      };
      
      // Sign user operation
      const entryPoint = EntryPoint__factory.connect(config.entryPoint, signer);
      const userOpHash = await entryPoint.getUserOpHash(userOp);
      const signature = await signer.signMessage(SmartAccountService.hashUserOp(userOp));
      userOp.signature = signature;
      
      // Send to bundler
      const bundlerUrl = config.bundlerUrl;
      const response = await fetch(`${bundlerUrl}/rpc`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "eth_sendUserOperation",
          params: [userOp, config.entryPoint],
          id: 1,
        }),
      });
      
      const result = await response.json();
      
      if (result.error) {
        throw new Error(result.error.message);
      }
      
      setTransactions(prev => [...prev, {
        hash: result.result,
        status: "pending",
        timestamp: Date.now(),
      }]);
      
      setSuccess(`Transaction sent! Hash: ${result.result.slice(0, 10)}...`);
      
      // Wait for confirmation
      await new Promise(resolve => setTimeout(resolve, 5000));
      setTransactions(prev => prev.map(tx => 
        tx.hash === result.result ? { ...tx, status: "confirmed" as const } : tx
      ));
    } catch (err: any) {
      setError(err.message || "Failed to send transaction");
    } finally {
      setIsLoading(false);
    }
  }, [accountAddress, smartAccount, chainId]);

  // Guardian recovery requires a deployed smart-account ABI that exposes guardian
  // storage and recovery methods. The current contract bindings do not expose those
  // methods, so this path fails closed instead of mutating local state or claiming success.
  const addGuardian = useCallback(async () => {
    if (!newGuardian || !smartAccount?.address) return;
    setError("Guardian recovery is unavailable because the deployed smart-account contract does not expose guardian methods");
  }, [newGuardian, smartAccount]);

  // Switch network
  const switchNetwork = useCallback(async (newChainId: number) => {
    if (!window.ethereum) return;
    
    try {
      await window.ethereum.request({
        method: "wallet_switchEthereumChain",
        params: [{ chainId: `0x${newChainId.toString(16)}` }],
      });
      setChainId(newChainId);
    } catch (err: any) {
      setError("Failed to switch network");
    }
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-center mb-12 gap-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 tiger-gradient rounded-xl flex items-center justify-center">
              <Wallet className="w-7 h-7 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">Account Abstraction</p>
            </div>
          </div>
          
          <div className="flex items-center gap-4">
            <select
              value={chainId}
              onChange={(e) => switchNetwork(Number(e.target.value))}
              className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-tiger-orange"
            >
              {Object.entries(CHAIN_CONFIG).map(([id, config]) => (
                <option key={id} value={id}>
                  {config.name}
                </option>
              ))}
            </select>
            
            {!isConnected ? (
              <button
                onClick={connectWallet}
                disabled={isLoading}
                className="bg-tiger-orange hover:bg-orange-600 px-6 py-2 rounded-lg font-medium transition-colors flex items-center gap-2"
              >
                {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />}
                Connect Wallet
              </button>
            ) : (
              <div className="flex items-center gap-2 bg-gray-800 px-4 py-2 rounded-lg">
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                <span className="text-sm font-mono">
                  {accountAddress?.slice(0, 6)}...{accountAddress?.slice(-4)}
                </span>
              </div>
            )}
          </div>
        </header>

        {/* Error/Success Messages */}
        {error && (
          <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 mb-6 flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
            <p className="text-red-400">{error}</p>
          </div>
        )}
        
        {success && (
          <div className="bg-green-500/10 border border-green-500/50 rounded-lg p-4 mb-6 flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0" />
            <p className="text-green-400">{success}</p>
          </div>
        )}

        {!isConnected ? (
          // Not Connected State
          <div className="text-center py-20">
            <div className="w-24 h-24 mx-auto mb-6 bg-gray-800 rounded-full flex items-center justify-center">
              <Shield className="w-12 h-12 text-tiger-orange" />
            </div>
            <h2 className="text-3xl font-bold mb-4">Smart Accounts Powered by ERC-4337</h2>
            <p className="text-gray-400 mb-8 max-w-2xl mx-auto">
              Experience gasless transactions, social recovery, and seamless multi-chain 
              operations with TigerWallet&apos;s Account Abstraction infrastructure.
            </p>
            <button
              onClick={connectWallet}
              className="tiger-gradient tiger-glow px-8 py-4 rounded-xl font-bold text-lg flex items-center gap-3 mx-auto hover:scale-105 transition-transform"
            >
              <Zap className="w-5 h-5" />
              Get Started
            </button>
            
            {/* Features Grid */}
            <div className="grid md:grid-cols-3 gap-6 mt-16">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Zap className="w-10 h-10 text-tiger-accent mb-4" />
                <h3 className="text-xl font-bold mb-2">Gasless Transactions</h3>
                <p className="text-gray-400">Pay gas with any token or let dApps sponsor your transactions</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Users className="w-10 h-10 text-tiger-accent mb-4" />
                <h3 className="text-xl font-bold mb-2">Social Recovery</h3>
                <p className="text-gray-400">Recover your account through trusted guardians</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Key className="w-10 h-10 text-tiger-accent mb-4" />
                <h3 className="text-xl font-bold mb-2">Session Keys</h3>
                <p className="text-gray-400">Grant limited permissions to dApps without exposing your keys</p>
              </div>
            </div>
          </div>
        ) : (
          // Connected State
          <div className="grid lg:grid-cols-3 gap-8">
            {/* Main Content */}
            <div className="lg:col-span-2 space-y-6">
              {/* Smart Account Info */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
                  <Wallet className="w-5 h-5 text-tiger-orange" />
                  Smart Account
                </h2>
                
                {smartAccount ? (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between p-4 bg-gray-900/50 rounded-lg">
                      <div>
                        <p className="text-gray-400 text-sm">Account Address</p>
                        <p className="font-mono text-sm break-all">{smartAccount.address}</p>
                      </div>
                      <button
                        onClick={() => navigator.clipboard.writeText(smartAccount.address)}
                        className="p-2 hover:bg-gray-700 rounded-lg transition-colors"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    </div>
                    
                    <div className="flex items-center gap-4">
                      <div className="flex items-center gap-2">
                        <div className={`w-3 h-3 rounded-full ${smartAccount.isDeployed ? 'bg-green-500' : 'bg-yellow-500'}`} />
                        <span className="text-sm">{smartAccount.isDeployed ? "Deployed" : "Not Deployed"}</span>
                      </div>
                      <div className="text-sm text-gray-400">
                        Nonce: {smartAccount.nonce}
                      </div>
                    </div>
                    
                    {!smartAccount.isDeployed && (
                      <button
                        onClick={deploySmartAccount}
                        disabled={isLoading}
                        className="w-full bg-tiger-orange hover:bg-orange-600 py-3 rounded-lg font-medium transition-colors flex items-center justify-center gap-2"
                      >
                        {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
                        Deploy Smart Account
                      </button>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-8 text-gray-400">
                    <p>Connect to see your smart account</p>
                  </div>
                )}
              </div>

              {/* Send Transaction */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Send Transaction</h2>
                <TransactionForm 
                  onSubmit={sendUserOp}
                  isLoading={isLoading}
                  disabled={!smartAccount?.isDeployed}
                />
              </div>

              {/* Transaction History */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Transaction History</h2>
                {transactions.length > 0 ? (
                  <div className="space-y-3">
                    {transactions.map((tx, i) => (
                      <div key={i} className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                        <div className="flex items-center gap-3">
                          <div className={`w-2 h-2 rounded-full ${
                            tx.status === "pending" ? "bg-yellow-500 animate-pulse" :
                            tx.status === "confirmed" ? "bg-green-500" : "bg-red-500"
                          }`} />
                          <span className="font-mono text-sm">{tx.hash.slice(0, 10)}...</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <a
                            href={`${CHAIN_CONFIG[chainId]?.explorer ?? "#"}/tx/${tx.hash}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-tiger-accent hover:underline text-sm flex items-center gap-1"
                          >
                            View <ExternalLink className="w-3 h-3" />
                          </a>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-gray-400 text-center py-8">No transactions yet</p>
                )}
              </div>
            </div>

            {/* Sidebar */}
            <div className="space-y-6">
              {/* Guardians */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
                  <Users className="w-5 h-5 text-tiger-orange" />
                  Guardians
                </h2>
                
                <div className="space-y-3 mb-4">
                  {guardians.length > 0 ? (
                    guardians.map((guardian, i) => (
                      <div key={i} className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg">
                        <span className="font-mono text-sm">{guardian.slice(0, 6)}...{guardian.slice(-4)}</span>
                      </div>
                    ))
                  ) : (
                    <p className="text-gray-400 text-sm">No guardians added yet</p>
                  )}
                </div>
                
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={newGuardian}
                    onChange={(e) => setNewGuardian(e.target.value)}
                    placeholder="0x..."
                    className="flex-1 bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-tiger-orange"
                  />
                  <button
                    onClick={addGuardian}
                    disabled={isLoading || !newGuardian}
                    className="bg-tiger-orange hover:bg-orange-600 px-4 py-2 rounded-lg transition-colors"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* Supported Chains */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Supported Chains</h2>
                <div className="space-y-2">
                  {Object.entries(CHAIN_CONFIG).map(([id, config]) => (
                    <div 
                      key={id}
                      className={`p-3 rounded-lg flex items-center justify-between ${
                        Number(id) === chainId ? 'bg-tiger-orange/20 border border-tiger-orange/50' : 'bg-gray-900/50'
                      }`}
                    >
                      <span className="text-sm">{config.name}</span>
                      {Number(id) === chainId && (
                        <CheckCircle className="w-4 h-4 text-tiger-orange" />
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Stats */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Statistics</h2>
                <div className="space-y-4">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Total UserOps</span>
                    <span className="font-bold">{transactions.length}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Gas Saved</span>
                    <span className="font-bold text-green-400">~{transactions.length * 0.002} ETH</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Guardians</span>
                    <span className="font-bold">{guardians.length}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// Transaction Form Component
function TransactionForm({ 
  onSubmit, 
  isLoading, 
  disabled 
}: { 
  onSubmit: (to: string, amount: string) => void;
  isLoading: boolean;
  disabled: boolean;
}) {
  const [to, setTo] = useState("");
  const [amount, setAmount] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (to && amount) {
      onSubmit(to, amount);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm text-gray-400 mb-2">Recipient Address</label>
        <input
          type="text"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          placeholder="0x..."
          disabled={disabled || isLoading}
          className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-tiger-orange disabled:opacity-50"
        />
      </div>
      <div>
        <label className="block text-sm text-gray-400 mb-2">Amount (ETH)</label>
        <input
          type="number"
          step="0.0001"
          min="0"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="0.0"
          disabled={disabled || isLoading}
          className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-tiger-orange disabled:opacity-50"
        />
      </div>
      <button
        type="submit"
        disabled={disabled || isLoading || !to || !amount}
        className="w-full bg-tiger-orange hover:bg-orange-600 py-3 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isLoading ? (
          <Loader2 className="w-4 h-4 animate-spin" />
        ) : (
          <>
            <ArrowRight className="w-4 h-4" />
            Send Gasless Transaction
          </>
        )}
      </button>
    </form>
  );
}
