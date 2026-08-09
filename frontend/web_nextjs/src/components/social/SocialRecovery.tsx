/**
 * TigerWallet - Social Recovery Component
 * Guardian-based social recovery for wallet access
 * 
 * Features:
 * - Add/remove guardians
 * - Guardian confirmation tracking
 * - Recovery request workflow
 * - Time-lock anti-scam protection
 */

import React, { useState, useCallback } from 'react';
import { ethers } from 'ethers';

type GuardianType = 'address' | 'email' | 'phone';

interface Guardian {
  id: string;
  address: string;
  type: GuardianType;
  confirmed: boolean;
  invitedAt: string;
  confirmedAt?: string;
}

interface RecoveryRequest {
  id: string;
  newOwnerAddress: string;
  status: 'pending' | 'approved' | 'executing' | 'completed' | 'cancelled';
  thresholdMet: boolean;
  signatures: GuardianSignature[];
  delayEnd: string;
  createdAt: string;
}

interface GuardianSignature {
  guardianId: string;
  signature: string;
  signedAt: string;
}

interface SocialRecoveryState {
  isLoading: boolean;
  error: string | null;
  guardians: Guardian[];
  threshold: number;
  activeRecovery: RecoveryRequest | null;
  walletAddress: string;
}

interface SocialRecoveryConfig {
  apiUrl: string;
  chainId: number;
}

export function useSocialRecovery(config: SocialRecoveryConfig) {
  const [state, setState] = useState<SocialRecoveryState>({
    isLoading: false,
    error: null,
    guardians: [],
    threshold: 2,
    activeRecovery: null,
    walletAddress: '',
  });

  // Load guardians
  const loadGuardians = useCallback(async (walletAddress: string): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await fetch(`${config.apiUrl}/api/v1/wallets/${walletAddress}`);
      
      if (!response.ok) {
        throw new Error('Failed to load guardians');
      }

      const data = await response.json();

      setState(prev => ({
        ...prev,
        isLoading: false,
        guardians: data.guardians || [],
        threshold: data.threshold || 2,
        walletAddress,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl]);

  // Add guardian
  const addGuardian = useCallback(async (
    guardianAddress: string,
    guardianType: GuardianType = 'address'
  ): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await fetch(
        `${config.apiUrl}/api/v1/wallets/${state.walletAddress}/guardians`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            guardian: {
              address: guardianAddress,
              type: guardianType,
            },
          }),
        }
      );

      if (!response.ok) {
        throw new Error('Failed to add guardian');
      }

      const data = await response.json();

      setState(prev => ({
        ...prev,
        isLoading: false,
        guardians: data.guardians,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl, state.walletAddress]);

  // Remove guardian
  const removeGuardian = useCallback(async (guardianId: string): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await fetch(
        `${config.apiUrl}/api/v1/wallets/${state.walletAddress}/guardians/${guardianId}`,
        { method: 'DELETE' }
      );

      if (!response.ok) {
        throw new Error('Failed to remove guardian');
      }

      const data = await response.json();

      setState(prev => ({
        ...prev,
        isLoading: false,
        guardians: data.guardians,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl, state.walletAddress]);

  // Initiate recovery
  const initiateRecovery = useCallback(async (newOwnerAddress: string): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await fetch(
        `${config.apiUrl}/api/v1/wallets/${state.walletAddress}/recovery`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            newOwnerAddress,
          }),
        }
      );

      if (!response.ok) {
        throw new Error('Failed to initiate recovery');
      }

      const data = await response.json();

      setState(prev => ({
        ...prev,
        isLoading: false,
        activeRecovery: data,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl, state.walletAddress]);

  // Sign recovery (for guardians)
  const signRecovery = useCallback(async (
    requestId: string,
    guardianId: string
  ): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      // Sign message
      const message = `Approve recovery request ${requestId}`;
      const signature = await (window as any).ethereum?.request({
        method: 'personal_sign',
        params: [message, (window as any).ethereum.selectedAddress],
      });

      const response = await fetch(
        `${config.apiUrl}/api/v1/recovery/${requestId}/sign`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            guardianId,
            signature,
          }),
        }
      );

      if (!response.ok) {
        throw new Error('Failed to sign recovery');
      }

      const data = await response.json();

      setState(prev => ({
        ...prev,
        isLoading: false,
        activeRecovery: data,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl]);

  // Execute recovery (after delay)
  const executeRecovery = useCallback(async (requestId: string): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await fetch(
        `${config.apiUrl}/api/v1/recovery/${requestId}/execute`,
        { method: 'POST' }
      );

      if (!response.ok) {
        throw new Error('Failed to execute recovery');
      }

      const data = await response.json();

      setState(prev => ({
        ...prev,
        isLoading: false,
        activeRecovery: null,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl]);

  // Cancel recovery
  const cancelRecovery = useCallback(async (requestId: string): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const response = await fetch(
        `${config.apiUrl}/api/v1/recovery/${requestId}/cancel`,
        { method: 'POST' }
      );

      if (!response.ok) {
        throw new Error('Failed to cancel recovery');
      }

      setState(prev => ({
        ...prev,
        isLoading: false,
        activeRecovery: null,
      }));
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: error.message,
      }));
    }
  }, [config.apiUrl]);

  // Update threshold
  const updateThreshold = useCallback(async (newThreshold: number): Promise<void> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    // In production, would update on smart contract
    setState(prev => ({
      ...prev,
      isLoading: false,
      threshold: newThreshold,
    }));
  }, []);

  return {
    ...state,
    loadGuardians,
    addGuardian,
    removeGuardian,
    initiateRecovery,
    signRecovery,
    executeRecovery,
    cancelRecovery,
    updateThreshold,
  };
}

// Guardian Card Component
interface GuardianCardProps {
  guardian: Guardian;
  onRemove: (id: string) => void;
}

function GuardianCard({ guardian, onRemove }: GuardianCardProps) {
  return (
    <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
      <div className="flex items-center gap-3">
        <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
          guardian.confirmed ? 'bg-green-100 text-green-600' : 'bg-yellow-100 text-yellow-600'
        }`}>
          {guardian.confirmed ? (
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          ) : (
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          )}
        </div>
        <div>
          <p className="font-medium text-gray-900">
            {guardian.type === 'address' 
              ? `${guardian.address.slice(0, 6)}...${guardian.address.slice(-4)}`
              : guardian.address
            }
          </p>
          <p className="text-sm text-gray-500">
            {guardian.confirmed ? 'Confirmed' : 'Pending confirmation'}
          </p>
        </div>
      </div>
      <button
        onClick={() => onRemove(guardian.id)}
        className="text-red-500 hover:text-red-700 p-2"
      >
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      </button>
    </div>
  );
}

// Recovery Status Card Component
interface RecoveryStatusCardProps {
  recovery: RecoveryRequest;
  onSign: () => void;
  onExecute: () => void;
  onCancel: () => void;
  canSign: boolean;
  canExecute: boolean;
}

function RecoveryStatusCard({ 
  recovery, 
  onSign, 
  onExecute, 
  onCancel, 
  canSign, 
  canExecute 
}: RecoveryStatusCardProps) {
  const statusColors = {
    pending: 'bg-yellow-100 text-yellow-800',
    approved: 'bg-blue-100 text-blue-800',
    executing: 'bg-purple-100 text-purple-800',
    completed: 'bg-green-100 text-green-800',
    cancelled: 'bg-red-100 text-red-800',
  };

  const delayEnd = new Date(recovery.delayEnd);
  const now = new Date();
  const canExecuteNow = canExecute && now >= delayEnd;

  return (
    <div className="bg-white border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold">Recovery Request</h3>
        <span className={`px-3 py-1 rounded-full text-sm font-medium ${statusColors[recovery.status]}`}>
          {recovery.status.charAt(0).toUpperCase() + recovery.status.slice(1)}
        </span>
      </div>

      <div className="space-y-3 mb-6">
        <div className="flex justify-between text-sm">
          <span className="text-gray-500">New Owner:</span>
          <span className="font-mono">
            {recovery.newOwnerAddress.slice(0, 6)}...{recovery.newOwnerAddress.slice(-4)}
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-gray-500">Signatures:</span>
          <span>{recovery.signatures.length} confirmed</span>
        </div>
        {recovery.status === 'pending' && (
          <div className="flex justify-between text-sm">
            <span className="text-gray-500">Available to execute:</span>
            <span className={canExecuteNow ? 'text-green-600' : 'text-gray-600'}>
              {delayEnd.toLocaleString()}
            </span>
          </div>
        )}
      </div>

      <div className="flex gap-3">
        {recovery.status === 'pending' && canSign && !recovery.thresholdMet && (
          <button
            onClick={onSign}
            className="flex-1 bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700"
          >
            Sign Recovery
          </button>
        )}
        {recovery.status === 'approved' && canExecuteNow && (
          <button
            onClick={onExecute}
            className="flex-1 bg-green-600 text-white py-2 px-4 rounded-lg hover:bg-green-700"
          >
            Execute Recovery
          </button>
        )}
        {recovery.status === 'pending' && (
          <button
            onClick={onCancel}
            className="flex-1 bg-gray-200 text-gray-700 py-2 px-4 rounded-lg hover:bg-gray-300"
          >
            Cancel
          </button>
        )}
      </div>

      {recovery.status === 'pending' && !canExecuteNow && (
        <p className="text-xs text-gray-500 mt-3 text-center">
          24-hour delay for security. This protects against scam attacks.
        </p>
      )}
    </div>
  );
}

// Main Social Recovery Component
interface SocialRecoveryPanelProps {
  walletAddress: string;
}

export function SocialRecoveryPanel({ walletAddress }: SocialRecoveryPanelProps) {
  const [newGuardian, setNewGuardian] = useState('');
  const [newOwner, setNewOwner] = useState('');

  const {
    isLoading,
    error,
    guardians,
    threshold,
    activeRecovery,
    loadGuardians,
    addGuardian,
    removeGuardian,
    initiateRecovery,
    signRecovery,
    executeRecovery,
    cancelRecovery,
  } = useSocialRecovery({
    apiUrl: process.env.NEXT_PUBLIC_SOCIAL_RECOVERY_API || 'http://localhost:8443',
    chainId: parseInt(process.env.NEXT_PUBLIC_CHAIN_ID || '1'),
  });

  // Load on mount
  React.useEffect(() => {
    if (walletAddress) {
      loadGuardians(walletAddress);
    }
  }, [walletAddress, loadGuardians]);

  const handleAddGuardian = async () => {
    if (!newGuardian) return;
    await addGuardian(newGuardian);
    setNewGuardian('');
  };

  const handleInitiateRecovery = async () => {
    if (!newOwner) return;
    await initiateRecovery(newOwner);
    setNewOwner('');
  };

  const confirmedCount = guardians.filter(g => g.confirmed).length;

  return (
    <div className="max-w-2xl mx-auto p-6">
      <h2 className="text-2xl font-bold mb-6">Social Recovery</h2>

      {error && (
        <div className="bg-red-50 text-red-700 p-4 rounded-lg mb-6">
          {error}
        </div>
      )}

      {/* Active Recovery */}
      {activeRecovery && (
        <div className="mb-6">
          <RecoveryStatusCard
            recovery={activeRecovery}
            onSign={() => signRecovery(activeRecovery.id, 'current-user')}
            onExecute={() => executeRecovery(activeRecovery.id)}
            onCancel={() => cancelRecovery(activeRecovery.id)}
            canSign={false}
            canExecute={false}
          />
        </div>
      )}

      {/* Guardians List */}
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">Guardians</h3>
          <span className="text-sm text-gray-500">
            {confirmedCount}/{threshold} confirmed
          </span>
        </div>

        {guardians.length === 0 ? (
          <p className="text-gray-500 text-center py-4">
            No guardians added yet
          </p>
        ) : (
          <div className="space-y-3">
            {guardians.map(guardian => (
              <GuardianCard
                key={guardian.id}
                guardian={guardian}
                onRemove={removeGuardian}
              />
            ))}
          </div>
        )}

        {/* Add Guardian */}
        <div className="mt-4 pt-4 border-t">
          <div className="flex gap-2">
            <input
              type="text"
              value={newGuardian}
              onChange={(e) => setNewGuardian(e.target.value)}
              placeholder="Guardian address or email"
              className="flex-1 px-4 py-2 border rounded-lg"
            />
            <button
              onClick={handleAddGuardian}
              disabled={isLoading || !newGuardian}
              className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              Add
            </button>
          </div>
        </div>
      </div>

      {/* Initiate Recovery */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold mb-4">Initiate Recovery</h3>
        <p className="text-gray-600 text-sm mb-4">
          If you've lost access to your wallet, you can initiate a recovery request.
          Guardians will have 24 hours to review before execution.
        </p>
        
        <div className="flex gap-2">
          <input
            type="text"
            value={newOwner}
            onChange={(e) => setNewOwner(e.target.value)}
            placeholder="New owner address"
            className="flex-1 px-4 py-2 border rounded-lg"
          />
          <button
            onClick={handleInitiateRecovery}
            disabled={isLoading || !newOwner || guardians.length < threshold}
            className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 disabled:opacity-50"
          >
            Start Recovery
          </button>
        </div>
        
        {guardians.length < threshold && (
          <p className="text-xs text-gray-500 mt-2">
            Need at least {threshold} guardians to initiate recovery
          </p>
        )}
      </div>
    </div>
  );
}

export default SocialRecoveryPanel;
