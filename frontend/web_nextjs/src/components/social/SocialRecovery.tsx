/**
 * TigerWallet - Social Recovery Component
 * Guardian-based wallet recovery against the TigerWallet social-recovery
 * service (social_recovery/go, REST under /api/v1).
 *
 * Every value here comes from the service; nothing is tracked only in React
 * state. The previous version kept guardians and the threshold in local state
 * and silently mutated them, so a user could "change the threshold" and see it
 * applied while the service still held the old one.
 *
 * The service is addressable by internal wallet id, and separately by chain and
 * address through /wallets/by-address/:chain/:address. Both are exposed here.
 */

import React, { useState, useCallback, useEffect } from 'react';

export type GuardianType = 'address' | 'email' | 'phone' | 'social';

export interface Guardian {
  id: string;
  address: string;
  email?: string;
  phone?: string;
  type: GuardianType;
  confirmed: boolean;
  invitedAt: string;
  confirmedAt?: string;
}

export type RecoveryStatus =
  | 'pending'
  | 'approved'
  | 'executing'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface GuardianSignature {
  guardianId: string;
  signature: string;
  signedAt: string;
  message: string;
}

export interface RecoveryRequest {
  id: string;
  walletId: string;
  newOwnerAddress: string;
  status: RecoveryStatus;
  thresholdMet: boolean;
  signatures: GuardianSignature[];
  delayEnd: string;
  executedAt?: string;
  cancelledAt?: string;
  createdAt: string;
  expiresAt: string;
}

export interface RecoveryWallet {
  id: string;
  address: string;
  chain: string;
  guardians: Guardian[];
  threshold: number;
  status: string;
  recoveryCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface SocialRecoveryConfig {
  /** Base URL of the social-recovery service, without a trailing slash. */
  apiUrl: string;
}

/** Raised when the service rejects a request. */
export class SocialRecoveryError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'SocialRecoveryError';
    this.status = status;
  }
}

/**
 * Performs a request against the service and surfaces the service's own error
 * message. The handlers answer failures with {"error": "..."}, which is far
 * more useful than the bare status line.
 */
async function request<T>(
  apiUrl: string,
  path: string,
  init?: RequestInit
): Promise<T> {
  const response = await fetch(`${apiUrl}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  });

  if (!response.ok) {
    let detail = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      if (body && typeof body.error === 'string') {
        detail = body.error;
      }
    } catch {
      // A non-JSON body leaves the status line as the best available detail.
    }
    throw new SocialRecoveryError(response.status, detail);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

interface SocialRecoveryState {
  isLoading: boolean;
  error: string | null;
  wallet: RecoveryWallet | null;
  activeRecovery: RecoveryRequest | null;
}

export function useSocialRecovery(config: SocialRecoveryConfig) {
  const { apiUrl } = config;

  const [state, setState] = useState<SocialRecoveryState>({
    isLoading: false,
    error: null,
    wallet: null,
    activeRecovery: null,
  });

  /**
   * Runs an operation, keeps the busy flag honest and stores the failure for the
   * caller to render. Returns the result so callers can chain.
   */
  const run = useCallback(
    async <T,>(action: () => Promise<T>): Promise<T | undefined> => {
      setState((prev) => ({ ...prev, isLoading: true, error: null }));
      try {
        const result = await action();
        setState((prev) => ({ ...prev, isLoading: false }));
        return result;
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        setState((prev) => ({ ...prev, isLoading: false, error: message }));
        return undefined;
      }
    },
    []
  );

  /** Loads a wallet by its internal id. */
  const loadWallet = useCallback(
    (walletId: string) =>
      run(async () => {
        const wallet = await request<RecoveryWallet>(
          apiUrl,
          `/api/v1/wallets/${encodeURIComponent(walletId)}`
        );
        setState((prev) => ({ ...prev, wallet }));
        return wallet;
      }),
    [apiUrl, run]
  );

  /** Loads a wallet by chain and address. */
  const loadWalletByAddress = useCallback(
    (chain: string, address: string) =>
      run(async () => {
        const wallet = await request<RecoveryWallet>(
          apiUrl,
          `/api/v1/wallets/by-address/${encodeURIComponent(chain)}/${encodeURIComponent(address)}`
        );
        setState((prev) => ({ ...prev, wallet }));
        return wallet;
      }),
    [apiUrl, run]
  );

  /** Registers a wallet for social recovery. */
  const createWallet = useCallback(
    (address: string, chain: string, threshold: number) =>
      run(async () => {
        const wallet = await request<RecoveryWallet>(apiUrl, '/api/v1/wallets', {
          method: 'POST',
          body: JSON.stringify({ address, chain, threshold }),
        });
        setState((prev) => ({ ...prev, wallet }));
        return wallet;
      }),
    [apiUrl, run]
  );

  /** Adds a guardian and adopts the wallet the service returns. */
  const addGuardian = useCallback(
    (walletId: string, guardian: { address: string; type: GuardianType }) =>
      run(async () => {
        const wallet = await request<RecoveryWallet>(
          apiUrl,
          `/api/v1/wallets/${encodeURIComponent(walletId)}/guardians`,
          { method: 'POST', body: JSON.stringify({ guardian }) }
        );
        setState((prev) => ({ ...prev, wallet }));
        return wallet;
      }),
    [apiUrl, run]
  );

  /** Removes a guardian. */
  const removeGuardian = useCallback(
    (walletId: string, guardianId: string) =>
      run(async () => {
        const wallet = await request<RecoveryWallet>(
          apiUrl,
          `/api/v1/wallets/${encodeURIComponent(walletId)}/guardians/${encodeURIComponent(guardianId)}`,
          { method: 'DELETE' }
        );
        setState((prev) => ({ ...prev, wallet }));
        return wallet;
      }),
    [apiUrl, run]
  );

  /** Confirms an email or phone guardian with the code the service sent. */
  const verifyGuardian = useCallback(
    (walletId: string, guardianId: string, code: string) =>
      run(async () => {
        const wallet = await request<RecoveryWallet>(
          apiUrl,
          `/api/v1/wallets/${encodeURIComponent(walletId)}/guardians/${encodeURIComponent(guardianId)}/verify`,
          { method: 'POST', body: JSON.stringify({ code }) }
        );
        setState((prev) => ({ ...prev, wallet }));
        return wallet;
      }),
    [apiUrl, run]
  );

  /** Starts a recovery to a new owner address. */
  const initiateRecovery = useCallback(
    (walletId: string, newOwnerAddress: string) =>
      run(async () => {
        const activeRecovery = await request<RecoveryRequest>(
          apiUrl,
          `/api/v1/wallets/${encodeURIComponent(walletId)}/recovery`,
          { method: 'POST', body: JSON.stringify({ newOwnerAddress }) }
        );
        setState((prev) => ({ ...prev, activeRecovery }));
        return activeRecovery;
      }),
    [apiUrl, run]
  );

  /** Loads the in-flight recovery for a wallet, if any. */
  const loadActiveRecovery = useCallback(
    (walletId: string) =>
      run(async () => {
        const activeRecovery = await request<RecoveryRequest>(
          apiUrl,
          `/api/v1/wallets/${encodeURIComponent(walletId)}/recovery`
        );
        setState((prev) => ({ ...prev, activeRecovery }));
        return activeRecovery;
      }),
    [apiUrl, run]
  );

  /** Adds a guardian's approval to a recovery request. */
  const signRecovery = useCallback(
    (requestId: string, guardianId: string, signature: string) =>
      run(async () => {
        const activeRecovery = await request<RecoveryRequest>(
          apiUrl,
          `/api/v1/recovery/${encodeURIComponent(requestId)}/sign`,
          { method: 'POST', body: JSON.stringify({ guardianId, signature }) }
        );
        setState((prev) => ({ ...prev, activeRecovery }));
        return activeRecovery;
      }),
    [apiUrl, run]
  );

  /**
   * Executes a recovery once the threshold is met and the delay has elapsed.
   * The service enforces both; this never forces the transfer.
   */
  const executeRecovery = useCallback(
    (requestId: string) =>
      run(async () => {
        const activeRecovery = await request<RecoveryRequest>(
          apiUrl,
          `/api/v1/recovery/${encodeURIComponent(requestId)}/execute`,
          { method: 'POST' }
        );
        setState((prev) => ({ ...prev, activeRecovery }));
        return activeRecovery;
      }),
    [apiUrl, run]
  );

  /** Cancels a pending recovery, for example after a phishing attempt. */
  const cancelRecovery = useCallback(
    (requestId: string) =>
      run(async () => {
        const activeRecovery = await request<RecoveryRequest>(
          apiUrl,
          `/api/v1/recovery/${encodeURIComponent(requestId)}/cancel`,
          { method: 'POST' }
        );
        setState((prev) => ({ ...prev, activeRecovery }));
        return activeRecovery;
      }),
    [apiUrl, run]
  );

  return {
    ...state,
    loadWallet,
    loadWalletByAddress,
    createWallet,
    addGuardian,
    removeGuardian,
    verifyGuardian,
    initiateRecovery,
    loadActiveRecovery,
    signRecovery,
    executeRecovery,
    cancelRecovery,
  };
}

/** Resolves the service base URL from the public environment. */
export function socialRecoveryConfigFromEnv(): SocialRecoveryConfig {
  return { apiUrl: process.env.NEXT_PUBLIC_SOCIAL_RECOVERY_URL ?? '' };
}

interface SocialRecoveryPanelProps {
  /** Internal wallet id, or omit and supply chain plus address. */
  walletId?: string;
  chain?: string;
  address?: string;
}

/** Guardian management and recovery workflow for a single wallet. */
export function SocialRecoveryPanel({ walletId, chain, address }: SocialRecoveryPanelProps) {
  const recovery = useSocialRecovery(socialRecoveryConfigFromEnv());

  const [guardianAddress, setGuardianAddress] = useState('');
  const [guardianType, setGuardianType] = useState<GuardianType>('address');
  const [verifyCode, setVerifyCode] = useState('');
  const [verifyingGuardianId, setVerifyingGuardianId] = useState<string | null>(null);
  const [newOwner, setNewOwner] = useState('');
  const [approvalGuardianId, setApprovalGuardianId] = useState('');
  const [approvalSignature, setApprovalSignature] = useState('');
  const [newWalletAddress, setNewWalletAddress] = useState('');
  const [newWalletChain, setNewWalletChain] = useState('');
  const [newWalletThreshold, setNewWalletThreshold] = useState('2');
  const [lookupWalletId, setLookupWalletId] = useState('');

  const resolvedId = recovery.wallet?.id ?? walletId;

  useEffect(() => {
    if (walletId) {
      recovery.loadWallet(walletId);
    } else if (chain && address) {
      recovery.loadWalletByAddress(chain, address);
    }
    // Loading is deliberately keyed on the identifiers only; the loader
    // callbacks are recreated whenever their inputs change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [walletId, chain, address]);

  const handleAddGuardian = useCallback(() => {
    if (!resolvedId || !guardianAddress) {
      return;
    }
    recovery.addGuardian(resolvedId, { address: guardianAddress, type: guardianType });
  }, [recovery, resolvedId, guardianAddress, guardianType]);

  const handleVerify = useCallback(() => {
    if (!resolvedId || !verifyingGuardianId || !verifyCode) {
      return;
    }
    recovery.verifyGuardian(resolvedId, verifyingGuardianId, verifyCode).then(() => {
      setVerifyCode('');
      setVerifyingGuardianId(null);
    });
  }, [recovery, resolvedId, verifyingGuardianId, verifyCode]);

  const handleInitiate = useCallback(() => {
    if (!resolvedId || !newOwner) {
      return;
    }
    recovery.initiateRecovery(resolvedId, newOwner);
  }, [recovery, resolvedId, newOwner]);

  const handleApprove = useCallback(() => {
    if (!recovery.activeRecovery || !approvalGuardianId || !approvalSignature) {
      return;
    }
    recovery
      .signRecovery(recovery.activeRecovery.id, approvalGuardianId, approvalSignature)
      .then(() => {
        setApprovalGuardianId('');
        setApprovalSignature('');
      });
  }, [recovery, approvalGuardianId, approvalSignature]);

  const handleCreate = useCallback(() => {
    if (!newWalletAddress || !newWalletChain) {
      return;
    }
    recovery.createWallet(newWalletAddress, newWalletChain, Number(newWalletThreshold));
  }, [recovery, newWalletAddress, newWalletChain, newWalletThreshold]);

  if (!resolvedId) {
    return (
      <div className="bg-white rounded-lg shadow p-6 space-y-3">
        <h2 className="text-xl font-bold">Social recovery</h2>
        <p className="text-sm text-gray-600">
          Register a wallet to protect it, or provide an existing wallet id.
        </p>
        <input
          type="text"
          placeholder="Wallet address"
          value={newWalletAddress}
          onChange={(e) => setNewWalletAddress(e.target.value)}
          className="w-full px-3 py-2 border rounded"
        />
        <input
          type="text"
          placeholder="Chain (for example ethereum)"
          value={newWalletChain}
          onChange={(e) => setNewWalletChain(e.target.value)}
          className="w-full px-3 py-2 border rounded"
        />
        <input
          type="number"
          min={1}
          placeholder="Threshold"
          value={newWalletThreshold}
          onChange={(e) => setNewWalletThreshold(e.target.value)}
          className="w-full px-3 py-2 border rounded"
        />
        <button
          onClick={handleCreate}
          disabled={
            recovery.isLoading ||
            !newWalletAddress ||
            !newWalletChain ||
            Number(newWalletThreshold) < 1
          }
          className="w-full bg-blue-600 text-white py-2 rounded disabled:opacity-50"
        >
          Register wallet
        </button>
        <input
          type="text"
          placeholder="…or an existing wallet id"
          value={lookupWalletId}
          onChange={(e) => setLookupWalletId(e.target.value)}
          className="w-full px-3 py-2 border rounded"
        />
        <button
          onClick={() => lookupWalletId && recovery.loadWallet(lookupWalletId)}
          disabled={recovery.isLoading || !lookupWalletId}
          className="w-full border py-2 rounded disabled:opacity-50"
        >
          Load wallet
        </button>
        {recovery.error && <p className="text-sm text-red-600">{recovery.error}</p>}
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow p-6 space-y-6">
      <h2 className="text-xl font-bold">Social recovery</h2>

      {recovery.wallet && (
        <div className="text-sm text-gray-700">
          <p className="font-mono break-all">{recovery.wallet.address}</p>
          <p>
            {recovery.wallet.chain} · threshold {recovery.wallet.threshold} of{' '}
            {recovery.wallet.guardians.length} · status {recovery.wallet.status}
          </p>
        </div>
      )}

      <div className="space-y-2">
        <h3 className="font-semibold">Guardians</h3>
        {recovery.wallet?.guardians.length ? (
          <ul className="divide-y">
            {recovery.wallet.guardians.map((guardian) => (
              <li key={guardian.id} className="py-2 flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="font-mono text-sm truncate">{guardian.address}</p>
                  <p className="text-xs text-gray-500">
                    {guardian.type} · {guardian.confirmed ? 'confirmed' : 'unconfirmed'}
                  </p>
                </div>
                <div className="flex gap-2 shrink-0">
                  {!guardian.confirmed && (
                    <button
                      onClick={() => setVerifyingGuardianId(guardian.id)}
                      className="text-xs px-2 py-1 border rounded"
                    >
                      Verify
                    </button>
                  )}
                  <button
                    onClick={() => recovery.removeGuardian(resolvedId, guardian.id)}
                    disabled={recovery.isLoading}
                    className="text-xs px-2 py-1 border rounded text-red-600 disabled:opacity-50"
                  >
                    Remove
                  </button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-gray-500">No guardians yet.</p>
        )}
      </div>

      <div className="space-y-2 pt-4 border-t">
        <h3 className="font-semibold">Add a guardian</h3>
        <input
          type="text"
          placeholder="Guardian address"
          value={guardianAddress}
          onChange={(e) => setGuardianAddress(e.target.value)}
          className="w-full px-3 py-2 border rounded"
        />
        <select
          value={guardianType}
          onChange={(e) => setGuardianType(e.target.value as GuardianType)}
          className="w-full px-3 py-2 border rounded"
        >
          <option value="address">Address</option>
          <option value="email">Email</option>
          <option value="phone">Phone</option>
          <option value="social">Social</option>
        </select>
        <button
          onClick={handleAddGuardian}
          disabled={recovery.isLoading || !guardianAddress}
          className="w-full bg-blue-600 text-white py-2 rounded disabled:opacity-50"
        >
          Add guardian
        </button>
      </div>

      {verifyingGuardianId && (
        <div className="space-y-2 pt-4 border-t">
          <h3 className="font-semibold">Confirm guardian</h3>
          <input
            type="text"
            placeholder="Verification code"
            value={verifyCode}
            onChange={(e) => setVerifyCode(e.target.value)}
            className="w-full px-3 py-2 border rounded"
          />
          <div className="flex gap-2">
            <button
              onClick={handleVerify}
              disabled={recovery.isLoading || !verifyCode}
              className="flex-1 bg-blue-600 text-white py-2 rounded disabled:opacity-50"
            >
              Confirm
            </button>
            <button
              onClick={() => {
                setVerifyingGuardianId(null);
                setVerifyCode('');
              }}
              className="flex-1 border py-2 rounded"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      <div className="space-y-2 pt-4 border-t">
        <h3 className="font-semibold">Recover to a new owner</h3>
        {recovery.activeRecovery ? (
          <div className="text-sm space-y-1">
            <p>
              Status {recovery.activeRecovery.status} ·{' '}
              {recovery.activeRecovery.signatures.length} of{' '}
              {recovery.wallet?.threshold ?? 0} approvals
            </p>
            <p className="text-gray-600">New owner {recovery.activeRecovery.newOwnerAddress}</p>
            <p className="text-gray-600">Executable after {recovery.activeRecovery.delayEnd}</p>

            <div className="space-y-2 pt-2">
              <input
                type="text"
                placeholder="Guardian id"
                value={approvalGuardianId}
                onChange={(e) => setApprovalGuardianId(e.target.value)}
                className="w-full px-3 py-2 border rounded"
              />
              <input
                type="text"
                placeholder="Signature"
                value={approvalSignature}
                onChange={(e) => setApprovalSignature(e.target.value)}
                className="w-full px-3 py-2 border rounded"
              />
              <button
                onClick={handleApprove}
                disabled={recovery.isLoading || !approvalGuardianId || !approvalSignature}
                className="w-full bg-blue-600 text-white py-2 rounded disabled:opacity-50"
              >
                Add approval
              </button>
            </div>

            <div className="flex gap-2 pt-2">
              <button
                onClick={() => recovery.executeRecovery(recovery.activeRecovery!.id)}
                disabled={recovery.isLoading || !recovery.activeRecovery.thresholdMet}
                className="flex-1 bg-green-600 text-white py-2 rounded disabled:opacity-50"
              >
                Execute
              </button>
              <button
                onClick={() => recovery.cancelRecovery(recovery.activeRecovery!.id)}
                disabled={recovery.isLoading}
                className="flex-1 border py-2 rounded text-red-600 disabled:opacity-50"
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <>
            <input
              type="text"
              placeholder="New owner address"
              value={newOwner}
              onChange={(e) => setNewOwner(e.target.value)}
              className="w-full px-3 py-2 border rounded"
            />
            <button
              onClick={handleInitiate}
              disabled={recovery.isLoading || !newOwner}
              className="w-full bg-gray-800 text-white py-2 rounded disabled:opacity-50"
            >
              Start recovery
            </button>
          </>
        )}
      </div>

      {recovery.error && <p className="text-sm text-red-600 break-words">{recovery.error}</p>}
    </div>
  );
}

export default SocialRecoveryPanel;
