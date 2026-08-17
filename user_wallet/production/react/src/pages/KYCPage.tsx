/**
 * KYC Page - Identity verification (required only for P2P trading).
 *
 * Fetches the caller's KYC status from the canonical wallet-api backend
 * (GET /kyc/status) on mount, then exposes a "Start KYC" flow when the
 * user is not yet verified:
 *   1. registerKyc({ full_name, document_type, document_number })
 *   2. submitKyc({ full_name, document_type, document_number })
 *   3. (optional) submitKycDocument(formData) for a verification file
 * Verified users see a green "KYC Verified — P2P trading enabled" banner.
 * All calls go through WalletService; no mock data.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

interface KycStatus {
  status?: string;
  verified?: boolean;
  full_name?: string;
  document_type?: string;
  document_number?: string;
  session_id?: string;
  submitted_at?: string;
}

const DOCUMENT_TYPES = ['passport', 'national_id', 'drivers_license'];

function KYCPage() {
  const { theme } = useTheme();
  const [walletService] = useState(() => new WalletService());

  const [status, setStatus] = useState<KycStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Start KYC form state
  const [fullName, setFullName] = useState('');
  const [documentType, setDocumentType] = useState(DOCUMENT_TYPES[0]);
  const [documentNumber, setDocumentNumber] = useState('');
  const [documentFile, setDocumentFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitMessage, setSubmitMessage] = useState<string | null>(null);

  const loadStatus = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = (await walletService.getKycStatus()) as KycStatus;
      setStatus(data ?? { status: 'not_started' });
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load KYC status');
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, [walletService]);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const isVerified = (s: KycStatus | null) => {
    if (!s) return false;
    if (typeof s.verified === 'boolean') return s.verified;
    const v = String(s.status ?? '').toLowerCase();
    return v === 'verified' || v === 'approved';
  };

  const isPending = (s: KycStatus | null) => {
    if (!s) return false;
    const v = String(s.status ?? '').toLowerCase();
    return v === 'pending' || v === 'in_review' || v === 'review';
  };

  const handleStartKyc = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitMessage(null);

    if (!fullName.trim() || !documentNumber.trim()) {
      setError('Full name and document number are required');
      return;
    }

    setSubmitting(true);
    try {
      // Step 1: register the KYC attempt.
      await walletService.registerKyc({
        full_name: fullName,
        document_type: documentType,
        document_number: documentNumber,
      });

      // Step 2: submit KYC details for review.
      const submitRes: any = await walletService.submitKyc({
        full_name: fullName,
        document_type: documentType,
        document_number: documentNumber,
      });

      // Step 3 (optional): upload a verification document if one was selected.
      if (documentFile) {
        const formData = new FormData();
        formData.append('document', documentFile);
        await walletService.submitKycDocument(formData);
      }

      setSubmitMessage(
        submitRes?.message ||
          submitRes?.status ||
          'KYC submitted for review. Verification is usually completed within a few minutes.'
      );

      // Refresh status after submission.
      await loadStatus();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'KYC submission failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] ?? null;
    setDocumentFile(file);
  };

  const cardClass = `card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`;
  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">KYC Verification</h1>

      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <p className={`text-sm ${theme === 'dark' ? 'text-gray-400' : 'text-gray-600'}`}>
          KYC required only for P2P trading. All other TigerWallet features (Send, Swap,
          Bridge, Staking, NFTs) work without verification.
        </p>
      </div>

      {loading ? (
        <LoadingSpinner label="Loading KYC status..." />
      ) : error && !status ? (
        <div className={`card mb-6 bg-red-500/20 border border-red-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <p className="text-red-500">{error}</p>
          <button onClick={loadStatus} className="btn btn-secondary mt-3">Retry</button>
        </div>
      ) : (
        <>
          {/* Status banner */}
          {isVerified(status) ? (
            <div className={`card mb-6 bg-green-500/20 border border-green-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <h3 className="font-semibold text-green-500 mb-1">
                ✓ KYC Verified — P2P trading enabled
              </h3>
              {status?.full_name && (
                <p className="text-sm opacity-70">{status.full_name}</p>
              )}
              {status?.document_type && (
                <p className="text-sm opacity-70">
                  {status.document_type}
                  {status.document_number ? ` • ${status.document_number}` : ''}
                </p>
              )}
            </div>
          ) : isPending(status) ? (
            <div className={`card mb-6 bg-amber-500/20 border border-amber-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <h3 className="font-semibold text-amber-500 mb-1">
                ⏳ KYC Under Review
              </h3>
              <p className="text-sm opacity-70">
                Your verification is being reviewed. P2P trading will be enabled once approved.
              </p>
              <button onClick={loadStatus} className="btn btn-secondary mt-3">Refresh Status</button>
            </div>
          ) : (
            <div className={`card mb-6 bg-blue-500/20 border border-blue-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <h3 className="font-semibold mb-1">Not Verified</h3>
              <p className="text-sm opacity-70">
                Complete identity verification to enable P2P trading.
              </p>
            </div>
          )}

          {submitMessage && (
            <div className={`card mb-6 bg-green-500/20 border border-green-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <p className="text-green-500 text-sm">{submitMessage}</p>
            </div>
          )}

          {error && (
            <div className={`card mb-6 bg-red-500/20 border border-red-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <p className="text-red-500">{error}</p>
            </div>
          )}

          {/* Start KYC form (only when not verified and not pending) */}
          {!isVerified(status) && !isPending(status) && (
            <form onSubmit={handleStartKyc} className={cardClass}>
              <h3 className="font-semibold mb-4">Start KYC</h3>

              <div className="mb-4">
                <label className="label">Full Name</label>
                <input
                  type="text"
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  placeholder="Jane Doe"
                  className={inputClass}
                  required
                />
              </div>

              <div className="mb-4">
                <label className="label">Document Type</label>
                <select
                  value={documentType}
                  onChange={(e) => setDocumentType(e.target.value)}
                  className={inputClass}
                >
                  {DOCUMENT_TYPES.map((t) => (
                    <option key={t} value={t}>
                      {t.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                    </option>
                  ))}
                </select>
              </div>

              <div className="mb-4">
                <label className="label">Document Number</label>
                <input
                  type="text"
                  value={documentNumber}
                  onChange={(e) => setDocumentNumber(e.target.value)}
                  placeholder="Document number"
                  className={inputClass}
                  required
                />
              </div>

              <div className="mb-6">
                <label className="label">Verification Document (optional)</label>
                <input
                  type="file"
                  accept="image/*,.pdf"
                  onChange={handleFileChange}
                  className={inputClass}
                />
                <p className={`text-xs mt-1 ${theme === 'dark' ? 'text-gray-500' : 'text-gray-500'}`}>
                  Upload a photo or PDF of the selected document (optional).
                </p>
              </div>

              <button
                type="submit"
                disabled={submitting}
                className="btn btn-primary w-full"
              >
                {submitting ? 'Submitting...' : 'Submit KYC'}
              </button>
            </form>
          )}
        </>
      )}
    </div>
  );
}

export default KYCPage;
