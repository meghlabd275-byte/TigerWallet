/**
 * TigerWallet Super Admin - Audit Logs Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function AuditLogs() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [action, setAction] = useState('');
  const [resourceType, setResourceType] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getAuditLogs({
        action: action || undefined,
        resource_type: resourceType || undefined,
      });
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Audit Logs</h1>

      <div className="card mb-4"><div className="card-body">
        <div className="flex gap-3">
          <div className="form-group flex-1"><label className="text-secondary">Action</label><input className="input w-full" value={action} onChange={(e) => setAction(e.target.value)} placeholder="e.g. create, update, delete" /></div>
          <div className="form-group flex-1"><label className="text-secondary">Resource Type</label><input className="input w-full" value={resourceType} onChange={(e) => setResourceType(e.target.value)} placeholder="e.g. user, token" /></div>
          <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-primary" onClick={load}>Filter</button></div>
        </div>
      </div></div>

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No audit logs found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Time</th><th>Admin</th><th>Action</th><th>Resource</th><th>Status</th><th>IP</th></tr></thead>
            <tbody>
              {items.map((l) => (
                <tr key={l.id}>
                  <td className="text-secondary">{l.created_at ? new Date(l.created_at).toLocaleString() : '-'}</td>
                  <td className="text-primary">{l.admin_email || l.admin_id}</td>
                  <td className="text-primary">{l.action}</td>
                  <td className="text-secondary">{l.resource_type}{l.resource_id ? ` / ${l.resource_id}` : ''}</td>
                  <td><span className={`badge ${l.status === 'success' ? 'badge-success' : 'badge-error'}`}>{l.status}</span></td>
                  <td className="text-secondary">{l.ip_address}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}
    </div>
  );
}
