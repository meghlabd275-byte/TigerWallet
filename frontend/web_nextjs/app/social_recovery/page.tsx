'use client';

import React, { useState, useEffect, useCallback } from 'react';
import api, { Guardian, RecoveryRequest } from '../../src/lib/api/client';

export default function SocialRecovery() {
  const [guardians, setGuardians] = useState<Guardian[]>([]);
  const [loadingGuardians, setLoadingGuardians] = useState(true);
  const [newGuardianAddress, setNewGuardianAddress] = useState('');
  const [newGuardianName, setNewGuardianName] = useState('');
  const [recoveryRequest, setRecoveryRequest] = useState<RecoveryRequest | null>(null);
  const [showAddGuardian, setShowAddGuardian] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchGuardians = useCallback(async () => {
    setLoadingGuardians(true);
    try {
      const res = await api.getGuardians();
      if (res.success && res.data) {
        setGuardians(res.data);
      } else {
        setGuardians([]);
        if (res.error) {
          setMessage({ type: 'error', text: res.error });
        }
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to load guardians' });
      setGuardians([]);
    } finally {
      setLoadingGuardians(false);
    }
  }, []);

  useEffect(() => {
    fetchGuardians();
  }, [fetchGuardians]);

  // Threshold derived from an active recovery request, if any; otherwise falls back
  // to a simple majority of the current guardian set (min 1).
  const threshold = recoveryRequest?.threshold ?? Math.max(1, Math.ceil(guardians.length / 2));
  const confirmations = recoveryRequest?.confirmations ?? guardians.filter(g => g.confirmed).length;

  const handleAddGuardian = async () => {
    if (!newGuardianAddress || !newGuardianName) {
      setMessage({ type: 'error', text: 'Please fill in all fields' });
      return;
    }

    if (!newGuardianAddress.startsWith('0x') || newGuardianAddress.length !== 42) {
      setMessage({ type: 'error', text: 'Invalid Ethereum address' });
      return;
    }

    setSubmitting(true);
    setMessage(null);
    try {
      const res = await api.addGuardian({ address: newGuardianAddress, name: newGuardianName });
      if (res.success && res.data) {
        setGuardians(prev => [...prev, res.data!]);
        setNewGuardianAddress('');
        setNewGuardianName('');
        setShowAddGuardian(false);
        setMessage({ type: 'success', text: 'Guardian added successfully!' });
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to add guardian' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to add guardian' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveGuardian = async (address: string) => {
    setSubmitting(true);
    setMessage(null);
    try {
      const res = await api.removeGuardian(address);
      if (res.success) {
        setGuardians(prev => prev.filter(g => g.address !== address));
        setMessage({ type: 'success', text: 'Guardian removed' });
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to remove guardian' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to remove guardian' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleInitiateRecovery = async () => {
    if (recoveryRequest) return;
    const newOwner = window.prompt('Enter the new owner address for recovery:');
    if (!newOwner) return;
    if (!newOwner.startsWith('0x') || newOwner.length !== 42) {
      setMessage({ type: 'error', text: 'Invalid new owner address' });
      return;
    }

    setSubmitting(true);
    setMessage(null);
    try {
      const res = await api.initiateRecovery(newOwner);
      if (res.success && res.data) {
        setRecoveryRequest(res.data);
        setMessage({ type: 'success', text: 'Recovery request initiated!' });
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to initiate recovery' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to initiate recovery' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancelRecovery = async () => {
    if (!recoveryRequest) return;
    setSubmitting(true);
    setMessage(null);
    try {
      const res = await api.cancelRecovery(recoveryRequest.id);
      if (res.success) {
        setRecoveryRequest(null);
        setMessage({ type: 'success', text: 'Recovery request cancelled' });
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to cancel recovery' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to cancel recovery' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleConfirmRecovery = async (guardian: Guardian) => {
    if (!recoveryRequest) return;
    setSubmitting(true);
    setMessage(null);
    try {
      const res = await api.confirmRecovery(recoveryRequest.id, guardian.address);
      if (res.success && res.data) {
        setRecoveryRequest(res.data);
        setGuardians(res.data.guardians);
        setMessage({ type: 'success', text: 'Recovery confirmed!' });
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to confirm recovery' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to confirm recovery' });
    } finally {
      setSubmitting(false);
    }
  };

  const formatAddress = (addr: string) => {
    if (!addr) return '';
    return addr.slice(0, 6) + '...' + addr.slice(-4);
  };

  const formatTime = (timestamp: number) => {
    return new Date(timestamp).toLocaleDateString();
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      {/* Header */}
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Social Recovery</h1>
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Wallet</a>
            </nav>
          </div>
        </div>
      </header>

      {/* Message */}
      {message && (
        <div className="max-w-4xl mx-auto px-4 pt-4">
          <div className={`p-3 rounded-lg ${message.type === 'success' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'}`}>
            {message.text}
          </div>
        </div>
      )}

      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Info Card */}
        <div className="bg-blue-50 dark:bg-blue-900 rounded-lg p-6 mb-8">
          <h3 className="font-semibold text-blue-800 dark:text-blue-200 mb-2">🔐 How Social Recovery Works</h3>
          <ul className="text-sm text-blue-700 dark:text-blue-300 space-y-1">
            <li>• Add guardians (friends, family, or devices) who can help recover your wallet</li>
            <li>• Set a threshold - minimum number of guardians needed to recover</li>
            <li>• If you lose access, guardians can sign a message to restore your wallet</li>
            <li>• You remain in full control - guardians cannot access your funds</li>
          </ul>
        </div>

        {/* Guardian Status */}
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 shadow-sm mb-8">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold">Your Guardians</h2>
            <button
              onClick={() => setShowAddGuardian(!showAddGuardian)}
              className="bg-orange-500 hover:bg-orange-600 text-white px-4 py-2 rounded-lg transition-colors"
            >
              + Add Guardian
            </button>
          </div>

          {/* Add Guardian Form */}
          {showAddGuardian && (
            <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4 mb-6">
              <h3 className="font-semibold mb-4">Add New Guardian</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-1">Guardian Name</label>
                  <input
                    type="text"
                    value={newGuardianName}
                    onChange={(e) => setNewGuardianName(e.target.value)}
                    placeholder="e.g., Mom, Best Friend"
                    className="w-full bg-white dark:bg-slate-600 border-0 rounded-lg px-4 py-2"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-1">Guardian Address</label>
                  <input
                    type="text"
                    value={newGuardianAddress}
                    onChange={(e) => setNewGuardianAddress(e.target.value)}
                    placeholder="0x..."
                    className="w-full bg-white dark:bg-slate-600 border-0 rounded-lg px-4 py-2 font-mono"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={handleAddGuardian}
                    disabled={submitting}
                    className="flex-1 bg-green-500 hover:bg-green-600 text-white py-2 rounded-lg transition-colors disabled:opacity-50"
                  >
                    {submitting ? 'Adding...' : 'Add Guardian'}
                  </button>
                  <button
                    onClick={() => setShowAddGuardian(false)}
                    className="px-4 py-2 bg-slate-300 dark:bg-slate-600 rounded-lg transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Threshold Info */}
          <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4 mb-6">
            <div className="flex justify-between items-center">
              <span className="text-slate-600 dark:text-slate-400">Recovery Threshold</span>
              <span className="font-semibold">{confirmations} / {threshold} guardians confirmed</span>
            </div>
            <div className="mt-2 h-2 bg-slate-300 dark:bg-slate-600 rounded-full overflow-hidden">
              <div
                className="h-full bg-orange-500 transition-all"
                style={{ width: `${(confirmations / threshold) * 100}%` }}
              />
            </div>
          </div>

          {/* Guardian List */}
          <div className="space-y-3">
            {loadingGuardians ? (
              <div className="text-center py-8 text-slate-500 dark:text-slate-400">Loading guardians...</div>
            ) : guardians.length === 0 ? (
              <div className="text-center py-8 text-slate-500 dark:text-slate-400">
                No guardians yet. Add trusted contacts to enable wallet recovery.
              </div>
            ) : (
              guardians.map((guardian) => (
                <div key={guardian.address} className="flex items-center justify-between p-4 bg-slate-100 dark:bg-slate-700 rounded-lg">
                  <div className="flex items-center gap-4">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                      guardian.confirmed ? 'bg-green-500' : 'bg-slate-400'
                    } text-white`}>
                      {guardian.confirmed ? '✓' : '?'}
                    </div>
                    <div>
                      <div className="font-semibold">{guardian.name}</div>
                      <div className="text-sm text-slate-500 dark:text-slate-400 font-mono">
                        {formatAddress(guardian.address)}
                      </div>
                      <div className="text-xs text-slate-400">Added {formatTime(guardian.addedAt)}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className={`px-2 py-1 rounded text-xs ${
                      guardian.confirmed 
                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                        : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                    }`}>
                      {guardian.confirmed ? 'Confirmed' : 'Pending'}
                    </span>
                    <button
                      onClick={() => handleRemoveGuardian(guardian.address)}
                      disabled={submitting}
                      className="text-red-500 hover:text-red-400 disabled:opacity-50"
                    >
                      Remove
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Recovery Section */}
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 shadow-sm">
          <h2 className="text-xl font-semibold mb-6">Wallet Recovery</h2>
          
          {!recoveryRequest ? (
            <div>
              <p className="text-slate-500 dark:text-slate-400 mb-4">
                If you've lost access to your wallet, you can initiate a recovery request.
                Your guardians will need to confirm to restore access.
              </p>
              <button
                onClick={handleInitiateRecovery}
                disabled={submitting || guardians.length === 0}
                className="w-full bg-red-500 hover:bg-red-600 text-white py-3 rounded-lg font-semibold transition-colors disabled:opacity-50"
              >
                {submitting ? 'Initiating...' : '🚨 Initiate Wallet Recovery'}
              </button>
            </div>
          ) : (
            <div>
              <div className="bg-yellow-50 dark:bg-yellow-900 border border-yellow-200 dark:border-yellow-700 rounded-lg p-4 mb-4">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-yellow-600 dark:text-yellow-400 text-xl">⚠️</span>
                  <span className="font-semibold text-yellow-800 dark:text-yellow-200">Recovery in Progress</span>
                </div>
                <p className="text-sm text-yellow-700 dark:text-yellow-300">
                  New owner address: {formatAddress(recoveryRequest.newOwner)}
                </p>
              </div>

              <div className="space-y-3 mb-4">
                {guardians.map((guardian) => (
                  <div key={guardian.address} className="flex items-center justify-between p-3 bg-slate-100 dark:bg-slate-700 rounded-lg">
                    <div className="font-mono text-sm">{formatAddress(guardian.address)}</div>
                    {guardian.confirmed ? (
                      <span className="text-green-500">✓ Confirmed</span>
                    ) : (
                      <button
                        onClick={() => handleConfirmRecovery(guardian)}
                        disabled={submitting}
                        className="text-orange-500 hover:text-orange-400 disabled:opacity-50"
                      >
                        {submitting ? 'Confirming...' : 'Confirm'}
                      </button>
                    )}
                  </div>
                ))}
              </div>

              <button
                onClick={handleCancelRecovery}
                disabled={submitting}
                className="w-full bg-slate-300 dark:bg-slate-600 hover:bg-slate-400 dark:hover:bg-slate-500 py-2 rounded-lg transition-colors disabled:opacity-50"
              >
                {submitting ? 'Cancelling...' : 'Cancel Recovery'}
              </button>
            </div>
          )}
        </div>

        {/* Security Tips */}
        <div className="mt-8 bg-slate-100 dark:bg-slate-800 rounded-lg p-6">
          <h3 className="font-semibold mb-4">🛡️ Security Tips</h3>
          <ul className="text-sm text-slate-600 dark:text-slate-400 space-y-2">
            <li>• Choose guardians you trust - they can help recover your wallet</li>
            <li>• Keep at least 3-5 guardians for redundancy</li>
            <li>• Don't give all guardians to the same person</li>
            <li>• Guardians cannot access your funds - only help recover access</li>
            <li>• Update your guardians periodically</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
