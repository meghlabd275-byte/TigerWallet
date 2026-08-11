'use client';

import React, { useState, useEffect, useCallback } from 'react';
import api, { Guardian, RecoveryRequest } from '../../src/lib/api/client';
import { useTheme } from '../components/ThemeProvider';

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

  const { isDark } = useTheme();

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
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-slate-50' : 'bg-slate-50 text-slate-900'}`}>
      {/* Header */}
      <header className={`border-b ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Social Recovery</h1>
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className={`${isDark ? 'text-slate-400' : 'text-slate-600'} hover:text-orange-500`}>Wallet</a>
            </nav>
          </div>
        </div>
      </header>

      {/* Message */}
      {message && (
        <div className="max-w-4xl mx-auto px-4 pt-4">
          <div className={`p-3 rounded-lg ${message.type === 'success' ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800')}`}>
            {message.text}
          </div>
        </div>
      )}

      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Info Card */}
        <div className={`rounded-lg p-6 mb-8 ${isDark ? 'bg-blue-900' : 'bg-blue-50'}`}>
          <h3 className={`font-semibold mb-2 ${isDark ? 'text-blue-200' : 'text-blue-800'}`}>🔐 How Social Recovery Works</h3>
          <ul className={`text-sm space-y-1 ${isDark ? 'text-blue-300' : 'text-blue-700'}`}>
            <li>• Add guardians (friends, family, or devices) who can help recover your wallet</li>
            <li>• Set a threshold - minimum number of guardians needed to recover</li>
            <li>• If you lose access, guardians can sign a message to restore your wallet</li>
            <li>• You remain in full control - guardians cannot access your funds</li>
          </ul>
        </div>

        {/* Guardians Section */}
        <div className={`rounded-lg p-6 shadow-sm mb-8 ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold">Your Guardians</h2>
            <button
              onClick={() => setShowAddGuardian(!showAddGuardian)}
              className="bg-green-500 hover:bg-green-600 text-white px-4 py-2 rounded-lg transition-colors"
            >
              {showAddGuardian ? 'Cancel' : '+ Add Guardian'}
            </button>
          </div>

          {/* Add Guardian Form */}
          {showAddGuardian && (
            <div className={`rounded-lg p-4 mb-6 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
              <div className="flex gap-2 mb-3">
                <div className="flex-1">
                  <label className={`block text-sm mb-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Guardian Name</label>
                  <input
                    type="text"
                    value={newGuardianName}
                    onChange={(e) => setNewGuardianName(e.target.value)}
                    placeholder="e.g., Alice"
                    className={`w-full border-0 rounded-lg px-4 py-2 ${isDark ? 'bg-slate-600' : 'bg-white'}`}
                  />
                </div>
                <div className="flex-1">
                  <label className={`block text-sm mb-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Guardian Address</label>
                  <input
                    type="text"
                    value={newGuardianAddress}
                    onChange={(e) => setNewGuardianAddress(e.target.value)}
                    placeholder="0x..."
                    className={`w-full border-0 rounded-lg px-4 py-2 font-mono ${isDark ? 'bg-slate-600' : 'bg-white'}`}
                  />
                </div>
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
                  className={`px-4 py-2 rounded-lg transition-colors ${isDark ? 'bg-slate-600' : 'bg-slate-300'}`}
                >
                  Cancel
                </button>
              </div>
            </div>
          )}

          {/* Threshold Info */}
          <div className={`rounded-lg p-4 mb-6 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
            <div className="flex justify-between items-center">
              <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Recovery Threshold</span>
              <span className="font-semibold">{confirmations} / {threshold} guardians confirmed</span>
            </div>
            <div className={`mt-2 h-2 rounded-full overflow-hidden ${isDark ? 'bg-slate-600' : 'bg-slate-300'}`}>
              <div
                className="h-full bg-orange-500 transition-all"
                style={{ width: `${(confirmations / threshold) * 100}%` }}
              />
            </div>
          </div>

          {/* Guardian List */}
          <div className="space-y-3">
            {loadingGuardians ? (
              <div className={`text-center py-8 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Loading guardians...</div>
            ) : guardians.length === 0 ? (
              <div className={`text-center py-8 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                No guardians yet. Add trusted contacts to enable wallet recovery.
              </div>
            ) : (
              guardians.map((guardian) => (
                <div key={guardian.address} className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
                  <div className="flex items-center gap-4">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                      guardian.confirmed ? 'bg-green-500' : 'bg-slate-400'
                    } text-white`}>
                      {guardian.confirmed ? '✓' : '?'}
                    </div>
                    <div>
                      <div className="font-semibold">{guardian.name}</div>
                      <div className={`text-sm font-mono ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {formatAddress(guardian.address)}
                      </div>
                      <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Added {formatTime(guardian.addedAt)}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className={`px-2 py-1 rounded text-xs ${
                      guardian.confirmed
                        ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800')
                        : (isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800')
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
        <div className={`rounded-lg p-6 shadow-sm ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
          <h2 className="text-xl font-semibold mb-6">Wallet Recovery</h2>
          
          {!recoveryRequest ? (
            <div>
              <p className={`mb-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
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
              <div className={`border rounded-lg p-4 mb-4 ${isDark ? 'bg-yellow-900 border-yellow-700' : 'bg-yellow-50 border-yellow-200'}`}>
                <div className="flex items-center gap-2 mb-2">
                  <span className={`text-xl ${isDark ? 'text-yellow-400' : 'text-yellow-600'}`}>⚠️</span>
                  <span className={`font-semibold ${isDark ? 'text-yellow-200' : 'text-yellow-800'}`}>Recovery in Progress</span>
                </div>
                <p className={`text-sm ${isDark ? 'text-yellow-300' : 'text-yellow-700'}`}>
                  New owner address: {formatAddress(recoveryRequest.newOwner)}
                </p>
              </div>

              <div className="space-y-3 mb-4">
                {guardians.map((guardian) => (
                  <div key={guardian.address} className={`flex items-center justify-between p-3 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
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
                className={`w-full py-2 rounded-lg transition-colors disabled:opacity-50 ${isDark ? 'bg-slate-600 hover:bg-slate-500' : 'bg-slate-300 hover:bg-slate-400'}`}
              >
                {submitting ? 'Cancelling...' : 'Cancel Recovery'}
              </button>
            </div>
          )}
        </div>

        {/* Security Tips */}
        <div className={`mt-8 rounded-lg p-6 ${isDark ? 'bg-slate-800' : 'bg-slate-100'}`}>
          <h3 className="font-semibold mb-4">🛡️ Security Tips</h3>
          <ul className={`text-sm space-y-2 ${isDark ? 'text-gray-400' : 'text-slate-600'}`}>
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

