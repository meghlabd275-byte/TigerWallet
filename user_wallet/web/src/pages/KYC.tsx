// KYC Page — view verification status and start/submit KYC for P2P trading.
// Mirrors Staking.tsx (fetch on mount, loading/error/empty states, themed
// classes). KYC is required ONLY for P2P trading.
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

type KycStatus = 'not_submitted' | 'pending' | 'verified' | 'rejected' | 'unknown';

interface KycState {
  status?: string;
  user_id?: string;
  session_id?: string;
  [key: string]: unknown;
}

const STATUS_LABEL: Record<string, string> = {
  not_submitted: 'Not Submitted',
  pending: 'Pending Review',
  verified: 'Verified',
  rejected: 'Rejected',
  unknown: 'Unknown',
};

function normalizeStatus(raw: KycState | null | undefined): KycStatus {
  const s = (raw?.status || '').toLowerCase();
  if (s === 'verified' || s === 'approved') return 'verified';
  if (s === 'pending' || s === 'review') return 'pending';
  if (s === 'rejected' || s === 'declined') return 'rejected';
  if (s === 'not_submitted' || s === 'none' || s === '') return 'not_submitted';
  return 'unknown';
}

export default function KYC() {
  const { user } = useAuth();
  const [kyc, setKyc] = useState<KycState | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [info, setInfo] = useState('');

  // KYC submission form fields.
  const [fullName, setFullName] = useState('');
  const [documentType, setDocumentType] = useState('passport');
  const [documentNumber, setDocumentNumber] = useState('');
  const [file, setFile] = useState<File | null>(null);

  const refresh = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.getKycStatus(user?.id);
      setKyc(data as KycState);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load KYC status');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const status = normalizeStatus(kyc);

  // Start KYC onboarding then submit the form body. Document upload is a
  // separate optional path.
  const startKyc = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setInfo('');
    setBusy(true);
    try {
      await api.registerKyc({});
      const res = await api.submitKyc({
        full_name: fullName,
        document_type: documentType,
        document_number: documentNumber,
      });
      setInfo(typeof res === 'string' ? res : 'KYC submitted for review.');
      if (file) {
        const fd = new FormData();
        fd.append('document', file);
        fd.append('document_type', documentType);
        await api.submitKycDocument(fd);
        setInfo('KYC submitted for review. Document uploaded.');
      }
      await refresh();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'KYC submission failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="kyc-page">
      <header className="page-header">
        <h1>KYC Verification</h1>
      </header>

      {error && <div className="error">{error}</div>}
      {info && <div className="success-banner"><h3>✓ Submitted</h3><p>{info}</p></div>}

      {loading ? (
        <p>Loading...</p>
      ) : (
        <>
          <div className={`kyc-banner ${status}`}>
            <h3>
              {status === 'verified' && '✓ KYC Verified — P2P trading enabled'}
              {status === 'pending' && '⏳ KYC Pending Review'}
              {status === 'rejected' && '✗ KYC Rejected'}
              {status === 'not_submitted' && 'KYC not started'}
              {status === 'unknown' && `Status: ${STATUS_LABEL[status]}`}
            </h3>
            <p>
              {status === 'verified'
                ? 'Your identity is verified. You can trade on the P2P marketplace.'
                : status === 'pending'
                ? 'Your documents are under review. You will be able to trade once approved.'
                : status === 'rejected'
                ? 'Your submission was rejected. Please review and re-submit.'
                : 'Complete verification to enable P2P trading.'}
            </p>
          </div>

          <p className="subtitle">KYC is required only for P2P trading.</p>

          {status !== 'verified' && (
            <div className="kyc-form">
              <h3>Start KYC</h3>
              <form onSubmit={startKyc}>
                <div className="form-group">
                  <label>Full Name</label>
                  <input value={fullName} onChange={(e) => setFullName(e.target.value)} required placeholder="Jane Doe" />
                </div>
                <div className="form-group">
                  <label>Document Type</label>
                  <select value={documentType} onChange={(e) => setDocumentType(e.target.value)}>
                    <option value="passport">Passport</option>
                    <option value="national_id">National ID</option>
                    <option value="drivers_license">Driver's License</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Document Number</label>
                  <input value={documentNumber} onChange={(e) => setDocumentNumber(e.target.value)} required placeholder="AB1234567" />
                </div>
                <div className="form-group">
                  <label>Document Photo (optional)</label>
                  <input type="file" accept="image/*,application/pdf" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
                </div>
                <button type="submit" className="primary-btn" disabled={busy}>
                  {busy ? 'Submitting…' : 'Start KYC'}
                </button>
              </form>
            </div>
          )}

          {kyc && kyc.session_id && (
            <p className="small">Session ID: <span className="mono">{kyc.session_id}</span></p>
          )}
        </>
      )}
    </div>
  );
}
