// Compliance Page - ProjectParty
import React, { useState } from 'react';
import { api } from '../services/api';

const AUDIT_TYPES = ['security', 'code', 'financial'];

export default function Compliance() {
  const [tokenId, setTokenId] = useState('');
  const [audits, setAudits] = useState<any[]>([]);
  const [kyc, setKyc] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const [auditType, setAuditType] = useState('security');
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchStatus = async (id: string) => {
    if (!id) return;
    setLoading(true);
    setError('');
    setAudits([]);
    setKyc(null);
    try {
      const [a, k] = await Promise.all([
        api.getAuditStatus(id).catch(() => ({ audits: [] })),
        api.getKYCStatus(id).catch(() => null)
      ]);
      setAudits(a.audits || []);
      if (k) setKyc(k);
    } catch (e: any) {
      setError(e.message || 'Failed to load compliance');
    }
    setLoading(false);
  };

  const onLookup = (e: React.FormEvent) => {
    e.preventDefault();
    fetchStatus(tokenId);
  };

  const requestAudit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tokenId) { setMsg({ type: 'error', text: 'Enter a token ID first.' }); return; }
    setSubmitting(true);
    setMsg(null);
    try {
      await api.requestAudit(tokenId, auditType);
      setMsg({ type: 'success', text: 'Audit requested.' });
      fetchStatus(tokenId);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to request audit' });
    }
    setSubmitting(false);
  };

  const submitKyc = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tokenId) { setMsg({ type: 'error', text: 'Enter a token ID first.' }); return; }
    setSubmitting(true);
    setMsg(null);
    try {
      await api.submitKYC(tokenId);
      setMsg({ type: 'success', text: 'KYC submitted.' });
      fetchStatus(tokenId);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to submit KYC' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header"><h1>Compliance</h1></div>
      <p className="subtitle">Request audits, submit KYC, and track compliance status for tokens.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      <section>
        <div className="section-title"><h2>Lookup Token Compliance</h2></div>
        <form onSubmit={onLookup} style={{ display: 'flex', gap: '0.5rem' }}>
          <input value={tokenId} onChange={e => setTokenId(e.target.value)} placeholder="Token ID (UUID)" required />
          <button type="submit" disabled={loading}>{loading ? 'Loading...' : 'Lookup'}</button>
        </form>
      </section>

      <div className="two-col">
        <section>
          <div className="section-title"><h2>Request Audit</h2></div>
          <form onSubmit={requestAudit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Audit Type</label>
                <select value={auditType} onChange={e => setAuditType(e.target.value)}>
                  {AUDIT_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
            </div>
            <button type="submit" disabled={submitting}>{submitting ? 'Requesting...' : 'Request Audit'}</button>
          </form>

          <div style={{ marginTop: '1rem' }}>
            <h3>Audits</h3>
            {audits.length === 0 ? <p className="muted">No data available</p> : (
              <table>
                <thead><tr><th>Type</th><th>Status</th><th>Auditor</th><th>Requested</th></tr></thead>
                <tbody>
                  {audits.map((a: any) => (
                    <tr key={a.id}>
                      <td>{a.audit_type}</td>
                      <td><span className={`badge ${a.status === 'completed' ? 'active' : ''}`}>{a.status}</span></td>
                      <td>{a.auditor || '-'}</td>
                      <td>{new Date(a.requested_at).toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </section>

        <section>
          <div className="section-title"><h2>KYC</h2></div>
          <form onSubmit={submitKyc}>
            <button type="submit" disabled={submitting}>{submitting ? 'Submitting...' : 'Submit KYC'}</button>
          </form>
          <div style={{ marginTop: '1rem' }}>
            {kyc ? (
              <>
                <div className="card-row"><span>Status</span><span><span className={`badge ${kyc.status === 'approved' ? 'active' : ''}`}>{kyc.status}</span></span></div>
                <div className="card-row"><span>Submitted</span><span>{kyc.submitted_at ? new Date(kyc.submitted_at).toLocaleString() : '-'}</span></div>
                <div className="card-row"><span>Expires</span><span>{kyc.expires_at ? new Date(kyc.expires_at).toLocaleString() : '-'}</span></div>
              </>
            ) : <p className="muted">No data available</p>}
          </div>
        </section>
      </div>
    </div>
  );
}
