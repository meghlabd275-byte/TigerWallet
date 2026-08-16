/**
 * TigerWallet Admin - Partners Management Page
 * CRUD + status + approve/reject for partners (mirrors /api/v1/partners)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { partnersAPI } from '../services/api';

interface Partner {
  id: string;
  name: string;
  email: string;
  type: 'affiliate' | 'referral' | 'institutional' | 'api';
  commissionRate: number;
  referralCode: string;
  totalReferrals: number;
  totalCommission: number;
  status: 'active' | 'paused' | 'suspended' | 'halted' | 'pending' | 'approved' | 'rejected';
  createdAt: string;
}

const STATUS_OPTIONS = ['active', 'paused', 'suspended', 'halted'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'active': case 'approved': return 'badge-success';
    case 'paused': case 'pending': return 'badge-warning';
    case 'suspended': case 'rejected': return 'badge-error';
    case 'halted': return 'badge-neutral';
    default: return 'badge-neutral';
  }
};

export const PartnersPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [partners, setPartners] = useState<Partner[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Partner | null>(null);
  const [rejectingId, setRejectingId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [formData, setFormData] = useState({ name: '', email: '', type: 'affiliate', commissionRate: '10', referralCode: '' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadPartners(); }, []);

  const loadPartners = async () => {
    try { setLoading(true); setError(null); const res = await partnersAPI.getAll(); setPartners(res.data || []); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load partners'); } finally { setLoading(false); }
  };

  const resetForm = () => { setFormData({ name: '', email: '', type: 'affiliate', commissionRate: '10', referralCode: '' }); setEditing(null); };
  const openCreate = () => { resetForm(); setShowForm(true); };
  const openEdit = (p: Partner) => { setEditing(p); setFormData({ name: p.name, email: p.email, type: p.type, commissionRate: String(p.commissionRate), referralCode: p.referralCode }); setShowForm(true); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = { ...formData, commissionRate: Number(formData.commissionRate) };
      if (editing) await partnersAPI.update(editing.id, payload); else await partnersAPI.create(payload);
      setShowForm(false); resetForm(); loadPartners();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save partner'); }
  };

  const handleDelete = async (id: string) => { if (!confirm('Delete this partner?')) return; try { await partnersAPI.delete(id); loadPartners(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete partner'); } };
  const handleStatusChange = async (id: string, status: Status) => { try { await partnersAPI.setStatus(id, status); loadPartners(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to update status'); } };
  const handleApprove = async (id: string) => { try { await partnersAPI.approve(id); loadPartners(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to approve partner'); } };
  const handleReject = async () => { if (!rejectingId) return; try { await partnersAPI.reject(rejectingId, rejectReason); setRejectingId(null); setRejectReason(''); loadPartners(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to reject partner'); } };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Partners Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Partner</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Partner' : 'New Partner'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group"><label className="form-label">Name</label><input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Email</label><input className="form-input" type="email" value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Type</label><select className="form-select" value={formData.type} onChange={(e) => setFormData({ ...formData, type: e.target.value as Partner['type'] })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}><option value="affiliate">Affiliate</option><option value="referral">Referral</option><option value="institutional">Institutional</option><option value="api">API</option></select></div>
                <div className="form-group"><label className="form-label">Commission Rate (%)</label><input className="form-input" type="number" step="0.1" value={formData.commissionRate} onChange={(e) => setFormData({ ...formData, commissionRate: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                <div className="form-group"><label className="form-label">Referral Code</label><input className="form-input" value={formData.referralCode} onChange={(e) => setFormData({ ...formData, referralCode: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
              </div>
              <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button></div>
            </form>
          </div>
        </div>
      )}

      {rejectingId && (
        <div className="modal-overlay" onClick={() => setRejectingId(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} style={{ backgroundColor: colors.bgCard }}>
            <div className="modal-header"><h3 style={{ color: colors.text }}>Reject Partner</h3></div>
            <div className="modal-body"><div className="form-group"><label className="form-label">Reason</label><textarea className="form-textarea" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div></div>
            <div className="modal-footer"><button className="btn btn-danger" onClick={handleReject}>Reject</button><button className="btn btn-secondary" onClick={() => setRejectingId(null)}>Cancel</button></div>
          </div>
        </div>
      )}

      <div className="card" style={{ backgroundColor: colors.bgCard }}>
        <div className="card-body p-0">
          {loading ? (<div className="flex items-center justify-center p-8"><div className="loader"></div></div>) : partners.length === 0 ? (
            <div className="text-center py-8" style={{ color: colors.textSecondary }}>No partners found</div>
          ) : (
            <table className="table">
              <thead><tr><th>Name</th><th>Email</th><th>Type</th><th>Commission</th><th>Referral Code</th><th>Referrals</th><th>Total Comm.</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
              <tbody>
                {partners.map((p) => (
                  <tr key={p.id}>
                    <td style={{ color: colors.text }}>{p.name}</td>
                    <td style={{ color: colors.textSecondary }}>{p.email}</td>
                    <td style={{ color: colors.textSecondary }}>{p.type}</td>
                    <td style={{ color: colors.textSecondary }}>{p.commissionRate}%</td>
                    <td style={{ color: colors.textSecondary }}>{p.referralCode}</td>
                    <td style={{ color: colors.textSecondary }}>{p.totalReferrals}</td>
                    <td style={{ color: colors.textSecondary }}>${p.totalCommission}</td>
                    <td><span className={`badge ${statusBadgeClass(p.status)}`}>{p.status}</span></td>
                    <td style={{ color: colors.textSecondary }}>{new Date(p.createdAt).toLocaleDateString()}</td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        {p.status === 'pending' && (<><button className="btn btn-sm btn-success" onClick={() => handleApprove(p.id)}>Approve</button><button className="btn btn-sm btn-danger" onClick={() => { setRejectingId(p.id); setRejectReason(''); }}>Reject</button></>)}
                        <select className="form-select" style={{ width: 'auto' }} value={p.status} onChange={(e) => handleStatusChange(p.id, e.target.value as Status)}>{STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}</select>
                        <button className="btn btn-sm btn-outline" onClick={() => openEdit(p)}>Edit</button>
                        <button className="btn btn-sm btn-danger" onClick={() => handleDelete(p.id)}>Delete</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
};

export default PartnersPage;
