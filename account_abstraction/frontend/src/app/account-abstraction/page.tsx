"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  AlertCircle,
  ArrowRight,
  CheckCircle,
  Copy,
  ExternalLink,
  Key,
  Loader2,
  Shield,
  Users,
  Wallet,
  Zap
} from "lucide-react";
import { BrowserProvider, formatEther, getAddress, getBytes, parseEther } from "ethers";
import type { EIP1193Provider } from "@/types/ethereum";

import {
  BundlerService,
  SmartAccountEncoder,
  SmartAccountService,
  type AccountCall
} from "@/lib/smartAccount";
import { accountFactoryContract, type FactoryMode } from "@/lib/contracts";

type ChainConfig = {
  name: string;
  entryPoint: string;
  factory: string;
  bundlerUrl: string;
  rpcUrl: string;
  explorer: string;
  factoryMode: FactoryMode;
  /** Decimals of the chain's native coin, for display. */
  nativeSymbol: string;
};

/**
 * Per-chain deployments.
 *
 * Next.js inlines `process.env.NEXT_PUBLIC_*` only for literal property reads,
 * so each value is named explicitly here. A chain whose addresses are not
 * configured is not offered: the UI cannot submit a user operation without a
 * real EntryPoint, factory and bundler.
 */
function chainConfigs(): Record<number, ChainConfig> {
  const candidates: Array<[number, string, ChainConfig]> = [
    [
      1,
      "Ethereum Mainnet",
      {
        name: "Ethereum Mainnet",
        entryPoint: process.env.NEXT_PUBLIC_ENTRYPOINT_1 ?? "",
        factory: process.env.NEXT_PUBLIC_FACTORY_1 ?? "",
        bundlerUrl: process.env.NEXT_PUBLIC_BUNDLER_1 ?? "",
        rpcUrl: process.env.NEXT_PUBLIC_RPC_1 ?? "",
        explorer: process.env.NEXT_PUBLIC_EXPLORER_1 ?? "https://etherscan.io",
        factoryMode: (process.env.NEXT_PUBLIC_FACTORY_MODE_1 as FactoryMode) ?? "canonical",
        nativeSymbol: "ETH"
      }
    ],
    [
      137,
      "Polygon Mainnet",
      {
        name: "Polygon Mainnet",
        entryPoint: process.env.NEXT_PUBLIC_ENTRYPOINT_137 ?? "",
        factory: process.env.NEXT_PUBLIC_FACTORY_137 ?? "",
        bundlerUrl: process.env.NEXT_PUBLIC_BUNDLER_137 ?? "",
        rpcUrl: process.env.NEXT_PUBLIC_RPC_137 ?? "",
        explorer: process.env.NEXT_PUBLIC_EXPLORER_137 ?? "https://polygonscan.com",
        factoryMode: (process.env.NEXT_PUBLIC_FACTORY_MODE_137 as FactoryMode) ?? "canonical",
        nativeSymbol: "POL"
      }
    ],
    [
      8453,
      "Base Mainnet",
      {
        name: "Base Mainnet",
        entryPoint: process.env.NEXT_PUBLIC_ENTRYPOINT_8453 ?? "",
        factory: process.env.NEXT_PUBLIC_FACTORY_8453 ?? "",
        bundlerUrl: process.env.NEXT_PUBLIC_BUNDLER_8453 ?? "",
        rpcUrl: process.env.NEXT_PUBLIC_RPC_8453 ?? "",
        explorer: process.env.NEXT_PUBLIC_EXPLORER_8453 ?? "https://basescan.org",
        factoryMode: (process.env.NEXT_PUBLIC_FACTORY_MODE_8453 as FactoryMode) ?? "canonical",
        nativeSymbol: "ETH"
      }
    ],
    [
      42161,
      "Arbitrum One",
      {
        name: "Arbitrum One",
        entryPoint: process.env.NEXT_PUBLIC_ENTRYPOINT_42161 ?? "",
        factory: process.env.NEXT_PUBLIC_FACTORY_42161 ?? "",
        bundlerUrl: process.env.NEXT_PUBLIC_BUNDLER_42161 ?? "",
        rpcUrl: process.env.NEXT_PUBLIC_RPC_42161 ?? "",
        explorer: process.env.NEXT_PUBLIC_EXPLORER_42161 ?? "https://arbiscan.io",
        factoryMode: (process.env.NEXT_PUBLIC_FACTORY_MODE_42161 as FactoryMode) ?? "canonical",
        nativeSymbol: "ETH"
      }
    ]
  ];

  const configured: Record<number, ChainConfig> = {};
  for (const [id, , config] of candidates) {
    if (config.entryPoint && config.factory && config.bundlerUrl && config.rpcUrl) {
      configured[id] = config;
    }
  }

  return configured;
}

function requireEthereumProvider(): EIP1193Provider {
  if (typeof window === "undefined" || !window.ethereum) {
    throw new Error("An injected EIP-1193 wallet provider is required");
  }
  return window.ethereum;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "The request failed";
}

/** A transaction the page has submitted, with the status the chain reports. */
type TrackedOperation = {
  hash: string;
  status: "pending" | "confirmed" | "failed";
  /** The block the operation was included in, once mined. */
  blockNumber?: number;
};

export default function AccountAbstractionPage() {
  const configs = useMemo(() => chainConfigs(), []);
  const chainIds = useMemo(
    () => Object.keys(configs).map(Number).sort((a, b) => a - b),
    [configs]
  );

  const [isConnected, setIsConnected] = useState(false);
  const [eoaAddress, setEoaAddress] = useState<string | null>(null);
  const [chainId, setChainId] = useState<number>(chainIds[0] ?? 0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [smartAccount, setSmartAccount] = useState<{
    address: string;
    isDeployed: boolean;
    nonce: bigint;
    deposit: bigint;
  } | null>(null);
  const [operations, setOperations] = useState<TrackedOperation[]>([]);

  const config = configs[chainId];

  const provider = useCallback(
    () => new BrowserProvider(requireEthereumProvider()),
    []
  );

  /**
   * Read the account's address, deployment state, nonce and EntryPoint deposit
   * from the chain.
   */
  const refreshSmartAccount = useCallback(
    async (owner: string, activeChainId: number) => {
      const activeConfig = configs[activeChainId];
      if (!activeConfig) {
        throw new Error(`Account abstraction is not configured for chain ${activeChainId}`);
      }

      const browserProvider = provider();
      const service = new SmartAccountService(browserProvider, activeConfig.factoryMode);

      const address = await service.getAccountAddress(activeConfig.factory, owner, 0n);
      const code = await browserProvider.getCode(address);

      const deployed = code !== "0x";

      // The nonce and the deposit only exist once the account is deployed, and
      // the EntryPoint reports both.
      let nonce = 0n;
      let deposit = 0n;

      if (deployed) {
        const results = await Promise.allSettled([
          service.getNonce(activeConfig.entryPoint, address),
          service.getDeposit(activeConfig.entryPoint, address)
        ]);

        if (results[0].status === "fulfilled") {
          nonce = results[0].value;
        }
        if (results[1].status === "fulfilled") {
          deposit = results[1].value;
        }
      }

      setSmartAccount({ address, isDeployed: deployed, nonce, deposit });
    },
    [configs, provider]
  );

  const connectWallet = useCallback(async () => {
    if (chainIds.length === 0) {
      setError(
        "Account abstraction is not configured. Set NEXT_PUBLIC_ENTRYPOINT_*, NEXT_PUBLIC_FACTORY_*, NEXT_PUBLIC_BUNDLER_* and NEXT_PUBLIC_RPC_*."
      );
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const browserProvider = provider();
      const accounts = (await browserProvider.send("eth_requestAccounts", [])) as string[];

      if (accounts.length === 0) {
        throw new Error("The wallet returned no accounts");
      }

      const network = await browserProvider.getNetwork();
      const activeChainId = Number(network.chainId);

      setEoaAddress(accounts[0]);
      setIsConnected(true);

      if (!configs[activeChainId]) {
        setChainId(chainIds[0]);
        setError(
          `Chain ${activeChainId} is not configured for account abstraction. Switch to a configured network.`
        );
        return;
      }

      setChainId(activeChainId);
      await refreshSmartAccount(accounts[0], activeChainId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setIsLoading(false);
    }
  }, [chainIds, configs, provider, refreshSmartAccount]);

  // Re-read the account when the user switches networks in their wallet.
  useEffect(() => {
    const injected = typeof window !== "undefined" ? window.ethereum : undefined;
    if (!injected?.on) {
      return;
    }

    const onChainChanged = (value: unknown) => {
      const next = Number(value);
      if (configs[next]) {
        setChainId(next);
        if (eoaAddress) {
          refreshSmartAccount(eoaAddress, next).catch((err) => setError(errorMessage(err)));
        }
      }
    };

    injected.on("chainChanged", onChainChanged);
    return () => {
      injected.removeListener?.("chainChanged", onChainChanged);
    };
  }, [configs, eoaAddress, refreshSmartAccount]);

  /**
   * Deploy the smart account.
   *
   * The factory's `createAccount` deploys the account, so this is an ordinary
   * transaction signed by the EOA; no user operation is involved.
   */
  const deploySmartAccount = useCallback(async () => {
    if (!eoaAddress || !config) {
      return;
    }

    setIsLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const browserProvider = provider();
      const signer = await browserProvider.getSigner();
      const service = new SmartAccountService(signer, config.factoryMode);

      // Ask the chain whether the account already exists, so a race with
      // another tab cannot produce a confusing failure.
      const address = await service.getAccountAddress(config.factory, eoaAddress, 0n);
      const code = await browserProvider.getCode(address);
      if (code !== "0x") {
        setSuccess("The smart account is already deployed.");
        await refreshSmartAccount(eoaAddress, chainId);
        return;
      }

      // Both factory flavours expose `createAccount(address,uint256)`, so the
      // shared binding covers them and the signature cannot drift.
      const factory = accountFactoryContract(config.factory, config.factoryMode, signer);

      const tx = await factory.createAccount(getAddress(eoaAddress), 0n);
      const receipt = await tx.wait();

      if (!receipt || receipt.status !== 1) {
        throw new Error("The deployment transaction was not successful");
      }

      setSuccess(`Smart account deployed in transaction ${tx.hash}`);
      await refreshSmartAccount(eoaAddress, chainId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setIsLoading(false);
    }
  }, [chainId, config, eoaAddress, provider, refreshSmartAccount]);

  /**
   * Build, sign and submit a user operation.
   *
   * The gas limits come from the bundler, the hash comes from the EntryPoint and
   * the confirmation comes from the bundler's receipt. Nothing is estimated
   * locally and nothing is assumed to have succeeded.
   */
  const sendUserOp = useCallback(
    async (to: string, amount: string, batch: boolean) => {
      if (!eoaAddress || !smartAccount?.isDeployed || !config) {
        return;
      }

      setIsLoading(true);
      setError(null);
      setSuccess(null);

      try {
        const recipient = getAddress(to);
        const value = parseEther(amount);
        if (value <= 0n) {
          throw new Error("The amount must be greater than zero");
        }

        const browserProvider = provider();
        const signer = await browserProvider.getSigner();
        const service = new SmartAccountService(signer, config.factoryMode);
        const bundler = new BundlerService(config.bundlerUrl, config.entryPoint);

        const call: AccountCall = { to: recipient, value, data: "0x" };
        const callData = batch
          ? SmartAccountEncoder.encodeExecuteBatch([call])
          : SmartAccountEncoder.encodeExecute(recipient, value);

        // A fresh nonce, read from the EntryPoint so two tabs cannot collide.
        const nonce = await service.getNonce(config.entryPoint, smartAccount.address);

        const feeData = await browserProvider.getFeeData();
        if (feeData.maxFeePerGas === null || feeData.maxPriorityFeePerGas === null) {
          throw new Error("The node did not report fee data for this chain");
        }

        const draft = service.buildUserOp({
          entryPoint: config.entryPoint,
          sender: smartAccount.address,
          nonce,
          callData,
          callGasLimit: 0n,
          verificationGasLimit: 0n,
          preVerificationGas: 0n,
          maxFeePerGas: feeData.maxFeePerGas,
          maxPriorityFeePerGas: feeData.maxPriorityFeePerGas
        });

        // Let the bundler fill the gas limits: it simulates the operation
        // against the deployed EntryPoint.
        const estimate = await bundler.estimateUserOperationGas(draft);

        const userOp = service.buildUserOp({
          entryPoint: config.entryPoint,
          sender: smartAccount.address,
          nonce,
          callData,
          callGasLimit: estimate.callGasLimit,
          verificationGasLimit: estimate.verificationGasLimit,
          preVerificationGas: estimate.preVerificationGas,
          maxFeePerGas: feeData.maxFeePerGas,
          maxPriorityFeePerGas: feeData.maxPriorityFeePerGas
        });

        // The hash the EntryPoint will verify against, computed by the
        // EntryPoint itself.
        const userOpHash = await service.getUserOpHash(config.entryPoint, userOp);

        // The account validates an EIP-191 signature over the hash.
        const signature = await signer.signMessage(getBytes(userOpHash));
        userOp.signature = signature;

        const hash = await bundler.sendUserOperation(userOp);

        setOperations((prev) => [{ hash, status: "pending" }, ...prev]);
        setSuccess(`User operation submitted: ${hash}`);

        // Poll the bundler for the receipt. The status reflects what the chain
        // reports, never an assumed outcome.
        for (let attempt = 0; attempt < 30; attempt += 1) {
          await new Promise((resolve) => setTimeout(resolve, 4000));

          const receipt = (await bundler.getUserOperationReceipt(hash)) as {
            success?: boolean;
            receipt?: { blockNumber?: string };
          } | null;

          if (!receipt) {
            continue;
          }

          const blockNumber = receipt.receipt?.blockNumber
            ? Number(BigInt(receipt.receipt.blockNumber))
            : undefined;

          setOperations((prev) =>
            prev.map((op) =>
              op.hash === hash
                ? {
                    ...op,
                    status: receipt.success ? "confirmed" : "failed",
                    blockNumber
                  }
                : op
            )
          );

          if (receipt.success) {
            setSuccess(`User operation confirmed in block ${blockNumber ?? "unknown"}`);
          } else {
            setError(`User operation ${hash} was included but reverted`);
          }

          await refreshSmartAccount(eoaAddress, chainId);
          return;
        }

        setError(
          `No receipt for ${hash} after two minutes. It may still be pending; check the explorer.`
        );
      } catch (err) {
        setError(errorMessage(err));
      } finally {
        setIsLoading(false);
      }
    },
    [chainId, config, eoaAddress, provider, refreshSmartAccount, smartAccount]
  );

  const switchNetwork = useCallback(async (newChainId: number) => {
    const injected = requireEthereumProvider();

    try {
      setError(null);
      await injected.request({
        method: "wallet_switchEthereumChain",
        params: [{ chainId: `0x${newChainId.toString(16)}` }]
      });
      setChainId(newChainId);
      if (eoaAddress) {
        await refreshSmartAccount(eoaAddress, newChainId);
      }
    } catch (err) {
      setError(errorMessage(err));
    }
  }, [eoaAddress, refreshSmartAccount]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-6xl mx-auto">
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
            {chainIds.length > 0 && (
              <select
                value={chainId}
                onChange={(event) => switchNetwork(Number(event.target.value))}
                className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-tiger-orange"
              >
                {chainIds.map((id) => (
                  <option key={id} value={id}>
                    {configs[id].name}
                  </option>
                ))}
              </select>
            )}

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
                <div className="w-2 h-2 bg-green-500 rounded-full" />
                <span className="text-sm font-mono">
                  {eoaAddress?.slice(0, 6)}...{eoaAddress?.slice(-4)}
                </span>
              </div>
            )}
          </div>
        </header>

        {error && (
          <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 mb-6 flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {success && (
          <div className="bg-green-500/10 border border-green-500/50 rounded-lg p-4 mb-6 flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0" />
            <p className="text-green-400 break-all">{success}</p>
          </div>
        )}

        {!isConnected ? (
          <div className="text-center py-20">
            <div className="w-24 h-24 mx-auto mb-6 bg-gray-800 rounded-full flex items-center justify-center">
              <Shield className="w-12 h-12 text-tiger-orange" />
            </div>
            <h2 className="text-3xl font-bold mb-4">Smart Accounts Powered by ERC-4337</h2>
            <p className="text-gray-400 mb-8 max-w-2xl mx-auto">
              Batch calls into one transaction, sponsor gas through a paymaster, and recover
              access through guardians — on the EntryPoint that is deployed for your chain.
            </p>
            <button
              onClick={connectWallet}
              className="tiger-gradient tiger-glow px-8 py-4 rounded-xl font-bold text-lg flex items-center gap-3 mx-auto hover:scale-105 transition-transform"
            >
              <Zap className="w-5 h-5" />
              Get Started
            </button>

            <div className="grid md:grid-cols-3 gap-6 mt-16">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Zap className="w-10 h-10 text-tiger-accent mb-4" />
                <h3 className="text-xl font-bold mb-2">Batched Calls</h3>
                <p className="text-gray-400">Several calls execute in one user operation</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Users className="w-10 h-10 text-tiger-accent mb-4" />
                <h3 className="text-xl font-bold mb-2">Guardian Recovery</h3>
                <p className="text-gray-400">Recover access through the account&apos;s recovery module</p>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <Key className="w-10 h-10 text-tiger-accent mb-4" />
                <h3 className="text-xl font-bold mb-2">Paymaster Sponsorship</h3>
                <p className="text-gray-400">A paymaster can pay gas for the operation</p>
              </div>
            </div>
          </div>
        ) : (
          <div className="grid lg:grid-cols-3 gap-8">
            <div className="lg:col-span-2 space-y-6">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
                  <Wallet className="w-5 h-5 text-tiger-orange" />
                  Smart Account
                </h2>

                {smartAccount ? (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between p-4 bg-gray-900/50 rounded-lg gap-4">
                      <div className="min-w-0">
                        <p className="text-gray-400 text-sm">Account Address</p>
                        <p className="font-mono text-sm break-all">{smartAccount.address}</p>
                      </div>
                      <button
                        onClick={() => navigator.clipboard.writeText(smartAccount.address)}
                        className="p-2 hover:bg-gray-700 rounded-lg transition-colors flex-shrink-0"
                        aria-label="Copy the account address"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    </div>

                    <div className="flex flex-wrap items-center gap-4">
                      <div className="flex items-center gap-2">
                        <div
                          className={`w-3 h-3 rounded-full ${
                            smartAccount.isDeployed ? "bg-green-500" : "bg-yellow-500"
                          }`}
                        />
                        <span className="text-sm">
                          {smartAccount.isDeployed ? "Deployed" : "Not deployed"}
                        </span>
                      </div>
                      <div className="text-sm text-gray-400">
                        Nonce: {smartAccount.nonce.toString()}
                      </div>
                      <div className="text-sm text-gray-400">
                        EntryPoint deposit: {formatEther(smartAccount.deposit)}{" "}
                        {config?.nativeSymbol ?? ""}
                      </div>
                    </div>

                    {!smartAccount.isDeployed && (
                      <button
                        onClick={deploySmartAccount}
                        disabled={isLoading}
                        className="w-full bg-tiger-orange hover:bg-orange-600 py-3 rounded-lg font-medium transition-colors flex items-center justify-center gap-2"
                      >
                        {isLoading ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Zap className="w-4 h-4" />
                        )}
                        Deploy Smart Account
                      </button>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-8 text-gray-400">
                    <p>The account address is read from the chain</p>
                  </div>
                )}
              </div>

              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Send Transaction</h2>
                <TransactionForm
                  onSubmit={sendUserOp}
                  isLoading={isLoading}
                  disabled={!smartAccount?.isDeployed}
                  symbol={config?.nativeSymbol ?? "ETH"}
                />
              </div>

              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">User Operations</h2>
                {operations.length > 0 ? (
                  <div className="space-y-3">
                    {operations.map((op) => (
                      <div
                        key={op.hash}
                        className="flex items-center justify-between p-3 bg-gray-900/50 rounded-lg gap-4"
                      >
                        <div className="flex items-center gap-3 min-w-0">
                          <div
                            className={`w-2 h-2 rounded-full flex-shrink-0 ${
                              op.status === "pending"
                                ? "bg-yellow-500"
                                : op.status === "confirmed"
                                  ? "bg-green-500"
                                  : "bg-red-500"
                            }`}
                          />
                          <span className="font-mono text-sm truncate">{op.hash}</span>
                          {op.blockNumber !== undefined && (
                            <span className="text-xs text-gray-500">block {op.blockNumber}</span>
                          )}
                        </div>
                        {config && (
                          <a
                            href={`${config.explorer}/tx/${op.hash}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-tiger-accent hover:underline text-sm flex items-center gap-1 flex-shrink-0"
                          >
                            View <ExternalLink className="w-3 h-3" />
                          </a>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-gray-400 text-center py-8">No user operations yet</p>
                )}
              </div>
            </div>

            <div className="space-y-6">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
                  <Users className="w-5 h-5 text-tiger-orange" />
                  Recovery
                </h2>
                <p className="text-gray-400 text-sm">
                  Guardian recovery is provided by the account&apos;s recovery module. Configure it
                  from the account contract; this console does not write guardian state.
                </p>
              </div>

              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Configured Chains</h2>
                {chainIds.length > 0 ? (
                  <div className="space-y-2">
                    {chainIds.map((id) => (
                      <div
                        key={id}
                        className={`p-3 rounded-lg flex items-center justify-between ${
                          id === chainId
                            ? "bg-tiger-orange/20 border border-tiger-orange/50"
                            : "bg-gray-900/50"
                        }`}
                      >
                        <span className="text-sm">{configs[id].name}</span>
                        {id === chainId && <CheckCircle className="w-4 h-4 text-tiger-orange" />}
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-gray-400 text-sm">
                    No chain is configured. Set the NEXT_PUBLIC_* deployment variables.
                  </p>
                )}
              </div>

              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <h2 className="text-xl font-bold mb-4">Session</h2>
                <div className="space-y-4">
                  <div className="flex justify-between">
                    <span className="text-gray-400">User operations</span>
                    <span className="font-bold">{operations.length}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">EntryPoint</span>
                    <span className="font-mono text-xs break-all">
                      {config?.entryPoint ?? "—"}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Factory</span>
                    <span className="font-mono text-xs break-all">{config?.factory ?? "—"}</span>
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

function TransactionForm({
  onSubmit,
  isLoading,
  disabled,
  symbol
}: {
  onSubmit: (to: string, amount: string, batch: boolean) => void;
  isLoading: boolean;
  disabled: boolean;
  symbol: string;
}) {
  const [to, setTo] = useState("");
  const [amount, setAmount] = useState("");
  const [batch, setBatch] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    setValidationError(null);

    try {
      getAddress(to);
    } catch {
      setValidationError("Enter a valid recipient address");
      return;
    }

    let parsed: bigint;
    try {
      parsed = parseEther(amount);
    } catch {
      setValidationError("Enter a valid amount");
      return;
    }

    if (parsed <= 0n) {
      setValidationError("The amount must be greater than zero");
      return;
    }

    onSubmit(to, amount, batch);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-red-400 text-sm flex items-center gap-2">
          <AlertCircle className="w-4 h-4" />
          {validationError}
        </p>
      )}

      <div>
        <label className="block text-sm text-gray-400 mb-2">Recipient Address</label>
        <input
          type="text"
          value={to}
          onChange={(event) => setTo(event.target.value)}
          placeholder="0x..."
          disabled={disabled || isLoading}
          className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-tiger-orange disabled:opacity-50"
        />
      </div>

      <div>
        <label className="block text-sm text-gray-400 mb-2">Amount ({symbol})</label>
        <input
          type="text"
          inputMode="decimal"
          value={amount}
          onChange={(event) => setAmount(event.target.value)}
          placeholder="0.0"
          disabled={disabled || isLoading}
          className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-tiger-orange disabled:opacity-50"
        />
      </div>

      <label className="flex items-center gap-2 text-sm text-gray-400">
        <input
          type="checkbox"
          checked={batch}
          onChange={(event) => setBatch(event.target.checked)}
          disabled={disabled || isLoading}
        />
        Route through executeBatch
      </label>

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
            Send User Operation
          </>
        )}
      </button>
    </form>
  );
}
