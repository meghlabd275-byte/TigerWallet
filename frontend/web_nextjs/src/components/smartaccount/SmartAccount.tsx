/**
 * TigerWallet - Smart Account (ERC-4337) React hook and components.
 *
 * Talks to the TigerWallet bundler (account_abstraction/go_api) over its REST
 * facade and to the deployed EntryPoint / TigerWalletAccountFactory over the
 * injected EIP-1193 provider. Every figure shown comes from the chain or from
 * the bundler; nothing is held only in React state.
 *
 * Three things this file used to do that it no longer does:
 *
 *  - addOwner / removeOwner only appended to a React array. TigerWalletAccount
 *    has a single `owner` (transferOwnership is onlyOwner), so an owner list was
 *    never real. Ownership transfer is now an actual transaction and the owner
 *    is read from the account.
 *  - addSessionKey / removeSessionKey only appended to a React array. No
 *    session-key contract exists in this repository, so these were removed
 *    rather than left looking functional.
 *  - The EntryPoint ABI was the v0.6 shape (separate callGasLimit,
 *    verificationGasLimit, maxFeePerGas fields). The EntryPoint in this
 *    repository is v0.7, which packs those into `accountGasLimits` and
 *    `gasFees`, so every call built here would have reverted.
 *
 * The user operation is signed with the EIP-191 personal-sign prefix over the
 * EntryPoint's userOpHash, which is what TigerWalletAccount._validateSignature
 * recovers against.
 */

import React, { useState, useCallback, useEffect } from 'react';
import { ethers } from 'ethers';

/** EntryPoint v0.7. getUserOpHash takes the packed struct. */
const ENTRYPOINT_ABI = [
  'function getUserOpHash((address sender, uint256 nonce, bytes initCode, bytes callData, bytes32 accountGasLimits, uint256 preVerificationGas, bytes32 gasFees, bytes paymasterAndData, bytes signature) userOp) view returns (bytes32)',
  'function getNonce(address sender, uint192 key) view returns (uint256)',
  'function getDeposit(address account) view returns (uint256)',
];

/** TigerWalletAccountFactory. `key` is the factory's salt input. */
const FACTORY_ABI = [
  'function createAccount(address owner, uint256 key) returns (address account)',
  'function getAccountAddress(address owner, uint256 key) view returns (address)',
];

/** TigerWalletAccount. */
const SMART_ACCOUNT_ABI = [
  'function owner() view returns (address)',
  'function transferOwnership(address newOwner)',
  'function executeFromOwner(address target, uint256 value, bytes data) returns (bool ok, bytes result)',
  'event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)',
];

/** BaseAccount.executeBatch takes an array of Call{target,value,data}. */
const EXECUTE_BATCH_ABI = [
  'function executeBatch((address target, uint256 value, bytes data)[] calls)',
];

/**
 * Packs two uint128 values into a bytes32, high half first, which is the layout
 * EntryPoint v0.7 expects for accountGasLimits (verificationGasLimit,
 * callGasLimit) and gasFees (maxPriorityFeePerGas, maxFeePerGas).
 */
function packUints(high: ethers.BigNumber, low: ethers.BigNumber): string {
  return ethers.utils.hexZeroPad(
    ethers.utils.hexConcat([
      ethers.utils.hexZeroPad(high.toHexString(), 16),
      ethers.utils.hexZeroPad(low.toHexString(), 16),
    ]),
    32
  );
}

/**
 * A user operation in the EntryPoint v0.7 packed layout, with the two uint128
 * pairs expressed as their named components for readability.
 */
export interface PackedUserOperation {
  sender: string;
  nonce: ethers.BigNumber;
  initCode: string;
  callData: string;
  verificationGasLimit: ethers.BigNumber;
  callGasLimit: ethers.BigNumber;
  preVerificationGas: ethers.BigNumber;
  maxPriorityFeePerGas: ethers.BigNumber;
  maxFeePerGas: ethers.BigNumber;
  paymasterAndData: string;
  signature: string;
}

/** Encodes a PackedUserOperation into the struct the EntryPoint ABI expects. */
function toAbiUserOp(op: PackedUserOperation) {
  return {
    sender: op.sender,
    nonce: op.nonce,
    initCode: op.initCode,
    callData: op.callData,
    accountGasLimits: packUints(op.verificationGasLimit, op.callGasLimit),
    preVerificationGas: op.preVerificationGas,
    gasFees: packUints(op.maxPriorityFeePerGas, op.maxFeePerGas),
    paymasterAndData: op.paymasterAndData,
    signature: op.signature,
  };
}

/** Encodes BaseAccount.executeBatch calldata for a list of calls. */
function encodeExecuteBatch(
  calls: Array<{ to: string; value?: ethers.BigNumber; data?: string }>
): string {
  const iface = new ethers.utils.Interface(EXECUTE_BATCH_ABI);
  return iface.encodeFunctionData('executeBatch', [
    calls.map((c) => [c.to, c.value ?? ethers.BigNumber.from(0), c.data ?? '0x']),
  ]);
}

/**
 * Converts the bundler's decimal quantity strings into BigNumbers. The bundler
 * reports nonce, balance and gas in decimal, unlike the chain's hex quantities.
 */
function toBigNumber(value: string | undefined): ethers.BigNumber {
  if (value === undefined || value === '') {
    return ethers.BigNumber.from(0);
  }
  return ethers.BigNumber.from(value);
}

export interface SmartAccountConfig {
  /** Deployed EntryPoint v0.7 address. Required; there is no safe default. */
  entryPoint: string;
  /** TigerWalletAccountFactory address. Required; there is no safe default. */
  factory: string;
  /** Bundler REST base, e.g. https://bundler.tigerwallet.io. Required. */
  bundlerUrl: string;
}

interface SmartAccountState {
  address: string | null;
  owner: string | null;
  isDeployed: boolean;
  isConnecting: boolean;
  /** Account balance in wei, as a decimal string. */
  balance: string;
  nonce: string;
  factory: string | null;
}

/** Raised when the hook is used without the configuration it needs. */
export class SmartAccountConfigError extends Error {}

function requireConfig(config: SmartAccountConfig): void {
  if (!config.entryPoint || !ethers.utils.isAddress(config.entryPoint)) {
    throw new SmartAccountConfigError(
      'entryPoint is not a valid address; set NEXT_PUBLIC_ENTRY_POINT'
    );
  }
  if (!config.factory || !ethers.utils.isAddress(config.factory)) {
    throw new SmartAccountConfigError(
      'factory is not a valid address; set NEXT_PUBLIC_SA_FACTORY'
    );
  }
  if (!config.bundlerUrl) {
    throw new SmartAccountConfigError(
      'bundlerUrl is required; set NEXT_PUBLIC_BUNDLER_URL'
    );
  }
}

/**
 * Reads the bundler's error body. The bundler answers failures with a JSON
 * object carrying an `error` string, so surfacing that is more useful than the
 * bare HTTP status.
 */
async function readBundlerError(response: Response): Promise<string> {
  let detail = `${response.status} ${response.statusText}`;
  try {
    const body = await response.json();
    if (body && typeof body.error === 'string') {
      detail = body.error;
    }
  } catch {
    // A non-JSON body leaves the status line as the best available detail.
  }
  return detail;
}

export function useSmartAccount(config: SmartAccountConfig) {
  const provider = React.useMemo(() => {
    if (typeof window === 'undefined' || !(window as any).ethereum) {
      return null;
    }
    return new ethers.providers.Web3Provider((window as any).ethereum);
  }, []);

  const [state, setState] = useState<SmartAccountState>({
    address: null,
    owner: null,
    isDeployed: false,
    isConnecting: false,
    balance: '0',
    nonce: '0',
    factory: null,
  });

  /**
   * Resolves the counterfactual account for the connected EOA. The address, the
   * deployment state, the balance and the nonce all come from the bundler, which
   * reads them from the chain, so a wrong configuration surfaces as an error
   * rather than as a plausible-looking account.
   */
  const connect = useCallback(async () => {
    if (!provider) {
      throw new SmartAccountConfigError('No EIP-1193 wallet is available');
    }
    requireConfig(config);

    const signer = provider.getSigner();
    const owner = await signer.getAddress();

    setState((prev) => ({ ...prev, isConnecting: true }));

    try {
      const response = await fetch(
        `${config.bundlerUrl}/v1/wallet/${owner}?salt=0`,
        { headers: { Accept: 'application/json' } }
      );
      if (!response.ok) {
        throw new Error(
          `bundler rejected the account lookup: ${await readBundlerError(response)}`
        );
      }

      const view = await response.json();
      if (!view || typeof view.address !== 'string') {
        throw new Error('the bundler returned no account address');
      }

      setState({
        address: view.address,
        owner: typeof view.owner === 'string' ? view.owner : owner,
        isDeployed: Boolean(view.isDeployed),
        isConnecting: false,
        balance: toBigNumber(view.balance).toString(),
        nonce: toBigNumber(view.nonce).toString(),
        factory:
          typeof view.factory === 'string' && view.factory ? view.factory : config.factory,
      });
    } catch (error) {
      setState((prev) => ({ ...prev, isConnecting: false }));
      throw error;
    }
  }, [provider, config]);

  /**
   * Deploys the account by calling the factory from the owner EOA, then
   * re-reads the owner from the chain so the reported state is the chain's, not
   * an optimistic guess. Returns the transaction hash.
   */
  const deploy = useCallback(async (): Promise<string> => {
    if (!provider) {
      throw new SmartAccountConfigError('No EIP-1193 wallet is available');
    }
    requireConfig(config);
    if (!state.address) {
      throw new Error('Connect before deploying');
    }
    if (state.isDeployed) {
      throw new Error('The account is already deployed');
    }

    const signer = provider.getSigner();
    const owner = await signer.getAddress();
    const factory = new ethers.Contract(config.factory, FACTORY_ABI, signer);

    // key 0 is the account the bundler reports for this owner, so the deployed
    // address matches the one shown before deployment.
    const tx = await factory.createAccount(owner, 0);
    const receipt = await tx.wait();

    const account = new ethers.Contract(state.address, SMART_ACCOUNT_ABI, provider);
    const onChainOwner: string = await account.owner();

    setState((prev) => ({ ...prev, isDeployed: true, owner: onChainOwner }));

    return receipt.transactionHash;
  }, [provider, config, state.address, state.isDeployed]);

  /**
   * Transfers ownership of the account. This is a real transaction from the
   * current owner; the contract rejects any other caller. The new owner is read
   * back from the chain.
   */
  const transferOwnership = useCallback(
    async (newOwner: string): Promise<string> => {
      if (!provider || !state.address) {
        throw new Error('Smart account not connected');
      }
      if (!ethers.utils.isAddress(newOwner)) {
        throw new Error('newOwner is not a valid address');
      }
      if (state.owner && newOwner.toLowerCase() === state.owner.toLowerCase()) {
        throw new Error('newOwner is already the owner');
      }

      const account = new ethers.Contract(
        state.address,
        SMART_ACCOUNT_ABI,
        provider.getSigner()
      );
      const tx = await account.transferOwnership(newOwner);
      const receipt = await tx.wait();

      const updated: string = await account.owner();
      setState((prev) => ({ ...prev, owner: updated }));

      return receipt.transactionHash;
    },
    [provider, state.address, state.owner]
  );

  /**
   * Builds, signs and submits a user operation carrying one or more calls.
   *
   * The signature is produced over the EntryPoint's userOpHash with the EIP-191
   * personal-sign prefix, matching TigerWalletAccount._validateSignature.
   * Returns the user operation hash the bundler accepted.
   */
  const sendUserOp = useCallback(
    async (
      calls: Array<{ to: string; value?: ethers.BigNumber; data?: string }>,
      options?: { paymasterAndData?: string }
    ): Promise<string> => {
      if (!provider || !state.address) {
        throw new Error('Smart account not connected');
      }
      requireConfig(config);
      if (calls.length === 0) {
        throw new Error('At least one call is required');
      }

      const signer = provider.getSigner();
      const entryPoint = new ethers.Contract(config.entryPoint, ENTRYPOINT_ABI, provider);

      const nonce: ethers.BigNumber = await entryPoint.getNonce(state.address, 0);
      const callData = encodeExecuteBatch(calls);
      const feeData = await provider.getFeeData();
      const maxFeePerGas = feeData.maxFeePerGas ?? ethers.BigNumber.from(0);
      const maxPriorityFeePerGas = feeData.maxPriorityFeePerGas ?? ethers.BigNumber.from(0);

      const userOp: PackedUserOperation = {
        sender: state.address,
        nonce,
        initCode: '0x',
        callData,
        // Conservative starting limits. The bundler refines them via
        // eth_estimateUserOperationGas before submission when sponsorship is
        // requested through /v1/paymaster/sponsorship.
        verificationGasLimit: ethers.BigNumber.from(150000),
        callGasLimit: ethers.BigNumber.from(100000),
        preVerificationGas: ethers.BigNumber.from(21000),
        maxPriorityFeePerGas,
        maxFeePerGas,
        paymasterAndData: options?.paymasterAndData ?? '0x',
        signature: '0x',
      };

      const userOpHash: string = await entryPoint.getUserOpHash(toAbiUserOp(userOp));

      // signMessage applies the EIP-191 prefix, which is what the account
      // recovers against.
      userOp.signature = await signer.signMessage(ethers.utils.arrayify(userOpHash));

      const response = await fetch(`${config.bundlerUrl}/v1/rpc/eth_sendUserOperation`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(toAbiUserOp(userOp)),
      });
      if (!response.ok) {
        throw new Error(
          `bundler rejected the user operation: ${await readBundlerError(response)}`
        );
      }

      const accepted = await response.json();
      if (typeof accepted === 'string') {
        return accepted;
      }
      if (accepted && typeof accepted.result === 'string') {
        return accepted.result;
      }
      throw new Error('the bundler returned no user operation hash');
    },
    [provider, config, state.address]
  );

  /** Re-reads the account from the bundler. */
  const refresh = useCallback(async () => {
    await connect();
  }, [connect]);

  useEffect(() => {
    if (provider && !state.address && !state.isConnecting) {
      connect().catch(() => {
        // A missing or misconfigured bundler is reported through the hook's
        // thrown errors on explicit calls; auto-connect stays quiet so the
        // component can render its unconfigured state.
      });
    }
  }, [provider, state.address, state.isConnecting, connect]);

  return {
    ...state,
    connect,
    deploy,
    sendUserOp,
    transferOwnership,
    refresh,
  };
}

/** Converts a decimal wei string into an ETH display string. */
export function formatWeiAsEther(wei: string): string {
  return ethers.utils.formatEther(toBigNumber(wei));
}

/** Reads the required AA configuration from the public environment. */
export function smartAccountConfigFromEnv(): SmartAccountConfig {
  return {
    entryPoint: process.env.NEXT_PUBLIC_ENTRY_POINT ?? '',
    factory: process.env.NEXT_PUBLIC_SA_FACTORY ?? '',
    bundlerUrl: process.env.NEXT_PUBLIC_BUNDLER_URL ?? '',
  };
}

/**
 * Smart account panel: shows the account the bundler reports for the connected
 * EOA and drives deployment, transfers and batch calls against it.
 */
export function SmartAccountCard() {
  const account = useSmartAccount(smartAccountConfigFromEnv());

  const [to, setTo] = useState('');
  const [value, setValue] = useState('');
  const [newOwner, setNewOwner] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [lastResult, setLastResult] = useState<string | null>(null);

  const run = useCallback(
    async (label: string, action: () => Promise<string>) => {
      setBusy(true);
      setError(null);
      setLastResult(null);
      try {
        const hash = await action();
        setLastResult(`${label}: ${hash}`);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(false);
      }
    },
    []
  );

  const handleSend = useCallback(() => {
    let parsed: ethers.BigNumber;
    try {
      parsed = ethers.utils.parseEther(value);
    } catch {
      setError(`"${value}" is not a valid amount`);
      return;
    }
    run('User operation submitted', () => account.sendUserOp([{ to, value: parsed }]));
  }, [account, to, value, run]);

  if (!account.address) {
    return (
      <div className="bg-white rounded-lg shadow-lg p-6 max-w-md">
        <h2 className="text-2xl font-bold mb-4">Smart Account</h2>
        <button
          onClick={() => run('Account', async () => {
            await account.connect();
            return account.address ?? 'connected';
          })}
          disabled={busy}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {busy ? 'Connecting…' : 'Connect Smart Account'}
        </button>
        {error && <p className="mt-3 text-sm text-red-600 break-words">{error}</p>}
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-lg p-6 max-w-md space-y-4">
      <h2 className="text-2xl font-bold">Smart Account</h2>

      <div>
        <label className="text-sm text-gray-500">Address</label>
        <p className="font-mono text-sm break-all">{account.address}</p>
      </div>

      <div>
        <label className="text-sm text-gray-500">Owner</label>
        <p className="font-mono text-sm break-all">{account.owner ?? '—'}</p>
      </div>

      <div>
        <label className="text-sm text-gray-500">Balance</label>
        <p className="text-lg font-semibold">{formatWeiAsEther(account.balance)} ETH</p>
      </div>

      <div>
        <label className="text-sm text-gray-500">Nonce</label>
        <p className="font-mono text-sm">{account.nonce}</p>
      </div>

      <div>
        <label className="text-sm text-gray-500">Status</label>
        <p className={`font-semibold ${account.isDeployed ? 'text-green-600' : 'text-yellow-600'}`}>
          {account.isDeployed ? 'Deployed' : 'Not deployed'}
        </p>
      </div>

      {!account.isDeployed && (
        <button
          onClick={() => run('Deployment', account.deploy)}
          disabled={busy}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {busy ? 'Deploying…' : 'Deploy Smart Account'}
        </button>
      )}

      {account.isDeployed && (
        <>
          <div className="pt-4 border-t space-y-2">
            <h3 className="font-semibold">Send</h3>
            <input
              type="text"
              placeholder="Recipient address"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="w-full px-3 py-2 border rounded"
            />
            <input
              type="text"
              placeholder="Amount (ETH)"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className="w-full px-3 py-2 border rounded"
            />
            <button
              onClick={handleSend}
              disabled={busy || !ethers.utils.isAddress(to) || !value}
              className="w-full bg-green-600 text-white py-2 px-4 rounded hover:bg-green-700 disabled:opacity-50"
            >
              {busy ? 'Submitting…' : 'Send'}
            </button>
          </div>

          <div className="pt-4 border-t space-y-2">
            <h3 className="font-semibold">Transfer ownership</h3>
            <input
              type="text"
              placeholder="New owner address"
              value={newOwner}
              onChange={(e) => setNewOwner(e.target.value)}
              className="w-full px-3 py-2 border rounded"
            />
            <button
              onClick={() => run('Ownership transfer', () => account.transferOwnership(newOwner))}
              disabled={busy || !ethers.utils.isAddress(newOwner)}
              className="w-full bg-gray-800 text-white py-2 px-4 rounded hover:bg-gray-900 disabled:opacity-50"
            >
              {busy ? 'Submitting…' : 'Transfer ownership'}
            </button>
          </div>
        </>
      )}

      {error && <p className="text-sm text-red-600 break-words">{error}</p>}
      {lastResult && <p className="text-sm text-green-700 break-all">{lastResult}</p>}
    </div>
  );
}

export default SmartAccountCard;
