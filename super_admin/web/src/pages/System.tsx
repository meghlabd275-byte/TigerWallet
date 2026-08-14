/**
 * TigerWallet Super Admin - System Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function System() {
  const [services, setServices] = useState<any[]>([]);
  const [config, setConfig] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [readiness, setReadiness] = useState<any | null>(null);
  const [configKey, setConfigKey] = useState('');
  const [configValue, setConfigValue] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const [h, c]: any = await Promise.all([
        superAdminApi.getSystemStatus(),
        superAdminApi.getConfig().catch(() => ({})),
      ]);
      setServices(Array.isArray(h) ? h : (h.services || h.data || []));
      setConfig(c || {});
    } catch (err: any) {
      setError(err?.message || 'Failed to load system status');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleRestart = async (name: string) => {
    if (!confirm(`Restart service "${name}"?`)) return;
    setActionLoading(true);
    try {
      await superAdminApi.restartService(name);
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to restart service');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReadiness = async () => {
    setActionLoading(true);
    try {
      const r = await superAdminApi.readinessCheck();
      setReadiness(r);
    } catch (err: any) {
      alert(err?.message || 'Readiness check failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleLiveness = async () => {
    setActionLoading(true);
    try {
      const r = await superAdminApi.healthCheck();
      alert(`Liveness: ${r.status}`);
    } catch (err: any) {
      alert(err?.message || 'Liveness check failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleConfigUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!configKey) return;
    setActionLoading(true);
    try {
      let value: any = configValue;
      try { value = JSON.parse(configValue); } catch { /* keep string */ }
      await superAdminApi.updateConfig({ [configKey]: value });
      setConfigKey('');
      setConfigValue('');
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to update config');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">System</h1>

      <div className="flex gap-2 mb-4">
        <button className="btn btn-primary" disabled={actionLoading} onClick={handleLiveness}>Liveness Check</button>
        <button className="btn btn-primary" disabled={actionLoading} onClick={handleReadiness}>Readiness Check</button>
        <button className="btn btn-secondary" onClick={load}>Refresh</button>
      </div>

      {readiness && (
        <div className="alert alert-info mb-4"><pre className="text-info" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(readiness, null, 2)}</pre></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : (
        <>
          <h2 className="text-xl font-bold text-primary mb-3">Services</h2>
          {services.length === 0 ? (
            <div className="card mb-6"><div className="card-body text-center py-8"><p className="text-secondary">No services found.</p></div></div>
          ) : (
            <div className="card mb-6"><div className="card-body overflow-x-auto">
              <table className="table"><thead><tr><th>Service</th><th>Status</th><th>Uptime</th><th>Version</th><th>Actions</th></tr></thead>
                <tbody>
                  {services.map((s, i) => (
                    <tr key={s.name || i}>
                      <td className="text-primary">{s.name}</td>
                      <td><span className={`badge ${s.status === 'healthy' || s.status === 'up' ? 'badge-success' : 'badge-error'}`}>{s.status}</span></td>
                      <td className="text-secondary">{s.uptime || '-'}</td>
                      <td className="text-secondary">{s.version || '-'}</td>
                      <td><button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleRestart(s.name)}>Restart</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div></div>
          )}

          <h2 className="text-xl font-bold text-primary mb-3">System Configuration</h2>
          <div className="card mb-4"><div className="card-body">
            <form onSubmit={handleConfigUpdate} className="flex gap-3 mb-4">
              <div className="form-group flex-1"><label className="text-secondary">Key</label><input className="input w-full" value={configKey} onChange={(e) => setConfigKey(e.target.value)} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Value (string or JSON)</label><input className="input w-full" value={configValue} onChange={(e) => setConfigValue(e.target.value)} required /></div>
              <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-primary" disabled={actionLoading} type="submit">Update</button></div>
            </form>
            {Object.keys(config).length === 0 ? (
              <p className="text-secondary">No configuration values.</p>
            ) : (
              <pre className="text-secondary" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(config, null, 2)}</pre>
            )}
          </div></div>
        </>
      )}
    </div>
  );
}
