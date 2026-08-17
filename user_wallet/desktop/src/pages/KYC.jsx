import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const DOCUMENT_TYPES = [
  { value: 'passport', label: 'Passport' },
  { value: 'national_id', label: 'National ID' },
  { value: 'drivers_license', label: "Driver's License" },
];

function KYC() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [form, setForm] = useState({ full_name: '', document_type: 'passport', document_number: '' });
  const [docFile, setDocFile] = useState(null);
  const [busy, setBusy] = useState(false);
  const [info, setInfo] = useState('');

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getKycStatus()
      .then((data) => { if (alive) { setStatus(data); setLoading(false); } })
      .catch((err) => { if (alive) { setError(err.message || 'Failed to load KYC status'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const isVerified = !!(
    status &&
    (status.verified || status.status === 'verified' || status.status === 'approved' ||
      status.kyc_status === 'verified' || status.kyc_status === 'approved')
  );

  const bannerStyle = isVerified
    ? { background: 'rgba(76, 175, 80, 0.15)', borderColor: 'var(--accent)' }
    : { background: isDark ? 'rgba(255, 152, 0, 0.12)' : 'rgba(255, 152, 0, 0.15)', borderColor: '#FF9800' };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    if (!form.full_name.trim() || !form.document_number.trim()) {
      setError('Full name and document number are required');
      return;
    }
    setBusy(true);
    try {
      await api.registerKyc({ full_name: form.full_name.trim() });
      const submitted = await api.submitKyc({
        full_name: form.full_name.trim(),
        document_type: form.document_type,
        document_number: form.document_number.trim(),
      });

      if (docFile) {
        const fd = new FormData();
        fd.append('document', docFile);
        if (submitted && (submitted.session_id || submitted.id)) {
          fd.append('session_id', submitted.session_id || submitted.id);
        }
        try {
          await api.submitKycDocument(fd);
        } catch (docErr) {
          setInfo(`KYC submitted, but document upload failed: ${docErr.message || docErr}`);
        }
      }

      setInfo('KYC submitted successfully. Verification may take a few moments.');
      setForm({ full_name: '', document_type: 'passport', document_number: '' });
      setDocFile(null);
      // Refresh status from the backend.
      const fresh = await api.getKycStatus();
      setStatus(fresh);
    } catch (err) {
      setError(err.message || 'KYC submission failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="kyc-page">
      <header className="page-header">
        <h1>KYC Verification</h1>
      </header>

      <p className="kyc-note" style={{ color: 'var(--text-secondary)' }}>
        KYC required only for P2P trading.
      </p>

      {loading ? (
        <p>Loading...</p>
      ) : error && !status ? (
        <div className="error">{error}</div>
      ) : (
        <>
          <div className="success-banner" style={bannerStyle}>
            <h3 style={{ color: 'var(--accent)' }}>
              {isVerified ? 'KYC Verified — P2P trading enabled' : 'KYC not yet verified'}
            </h3>
            <p style={{ color: 'var(--text-secondary)' }}>
              {isVerified
                ? 'Your identity is verified. You can place P2P orders.'
                : 'Complete KYC below to enable P2P trading.'}
            </p>
          </div>

          {!isVerified && (
            <form className="kyc-form" onSubmit={handleSubmit}>
              {error && <div className="error">{error}</div>}
              {info && <div className="backup-msg">{info}</div>}

              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Full name</label>
              <input
                placeholder="Jane Doe"
                value={form.full_name}
                onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                required
              />

              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Document type</label>
              <select
                value={form.document_type}
                onChange={(e) => setForm({ ...form, document_type: e.target.value })}
              >
                {DOCUMENT_TYPES.map((d) => <option key={d.value} value={d.value}>{d.label}</option>)}
              </select>

              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Document number</label>
              <input
                placeholder="Document number"
                value={form.document_number}
                onChange={(e) => setForm({ ...form, document_number: e.target.value })}
                required
              />

              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Document upload (optional)</label>
              <input
                type="file"
                onChange={(e) => setDocFile(e.target.files && e.target.files[0] ? e.target.files[0] : null)}
              />

              <button type="submit" disabled={busy}>{busy ? 'Submitting…' : 'Start KYC'}</button>
            </form>
          )}
        </>
      )}
    </div>
  );
}

export default KYC;
