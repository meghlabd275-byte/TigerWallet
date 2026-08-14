/**
 * TigerWallet Super Admin - Reports Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Reports() {
  const [items, setItems] = useState<any[]>([]);
  const [templates, setTemplates] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [compliance, setCompliance] = useState({ type: 'kyc', startDate: '', endDate: '' });
  const [finance, setFinance] = useState({ type: 'revenue', period: 'monthly' });
  const [report, setReport] = useState({ report_type: 'user', title: '', user_id: '', format: 'pdf' });
  const [generated, setGenerated] = useState<any | null>(null);

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const [fin, tpl]: any = await Promise.all([
        superAdminApi.getFinanceReports(),
        superAdminApi.getReportTemplates().catch(() => ({ templates: [] })),
      ]);
      setItems(fin.data || fin.items || fin || []);
      setTemplates(tpl.templates || tpl.data || tpl || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load reports');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCompliance = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      const r = await superAdminApi.generateComplianceReport(compliance.type, compliance.startDate, compliance.endDate);
      setGenerated(r);
    } catch (err: any) {
      alert(err?.message || 'Failed to generate compliance report');
    } finally {
      setActionLoading(false);
    }
  };

  const handleFinance = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      const r = await superAdminApi.generateFinanceReport(finance.type, finance.period);
      setGenerated(r);
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to generate finance report');
    } finally {
      setActionLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createReport({
        report_type: report.report_type,
        title: report.title,
        user_id: report.user_id,
        format: report.format,
      });
      setReport({ report_type: 'user', title: '', user_id: '', format: 'pdf' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create report');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDownload = async (id: string) => {
    setActionLoading(true);
    try {
      const blob = await superAdminApi.downloadReport(id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `report-${id}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err: any) {
      alert(err?.message || 'Download failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Reports</h1>

      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="card"><div className="card-body">
          <h3 className="text-primary mb-4">Generate Compliance Report</h3>
          <form onSubmit={handleCompliance} className="flex flex-col gap-3">
            <div className="form-group"><label className="text-secondary">Type</label><select className="input w-full" value={compliance.type} onChange={(e) => setCompliance({ ...compliance, type: e.target.value })}><option>kyc</option><option>aml</option><option>transaction</option></select></div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Start Date</label><input className="input w-full" type="date" value={compliance.startDate} onChange={(e) => setCompliance({ ...compliance, startDate: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">End Date</label><input className="input w-full" type="date" value={compliance.endDate} onChange={(e) => setCompliance({ ...compliance, endDate: e.target.value })} required /></div>
            </div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Generate</button>
          </form>
        </div></div>

        <div className="card"><div className="card-body">
          <h3 className="text-primary mb-4">Generate Finance Report</h3>
          <form onSubmit={handleFinance} className="flex flex-col gap-3">
            <div className="form-group"><label className="text-secondary">Type</label><select className="input w-full" value={finance.type} onChange={(e) => setFinance({ ...finance, type: e.target.value })}><option>revenue</option><option>volume</option><option>fees</option></select></div>
            <div className="form-group"><label className="text-secondary">Period</label><select className="input w-full" value={finance.period} onChange={(e) => setFinance({ ...finance, period: e.target.value })}><option>daily</option><option>weekly</option><option>monthly</option><option>yearly</option></select></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Generate</button>
          </form>
        </div></div>
      </div>

      <div className="card mb-6"><div className="card-body">
        <h3 className="text-primary mb-4">Create Report</h3>
        <form onSubmit={handleCreate} className="flex flex-col gap-3">
          <div className="flex gap-3">
            <div className="form-group flex-1"><label className="text-secondary">Title</label><input className="input w-full" value={report.title} onChange={(e) => setReport({ ...report, title: e.target.value })} required /></div>
            <div className="form-group flex-1"><label className="text-secondary">Report Type</label><input className="input w-full" value={report.report_type} onChange={(e) => setReport({ ...report, report_type: e.target.value })} required /></div>
            <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={report.user_id} onChange={(e) => setReport({ ...report, user_id: e.target.value })} required /></div>
            <div className="form-group flex-1"><label className="text-secondary">Format</label><select className="input w-full" value={report.format} onChange={(e) => setReport({ ...report, format: e.target.value })}><option>pdf</option><option>csv</option><option>json</option></select></div>
          </div>
          <button className="btn btn-primary" disabled={actionLoading} type="submit">Create Report</button>
        </form>
      </div></div>

      {generated && (
        <div className="alert alert-info mb-4"><pre className="text-info" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(generated, null, 2)}</pre></div>
      )}

      {templates.length > 0 && (
        <div className="card mb-4"><div className="card-body">
          <h3 className="text-primary mb-2">Report Templates</h3>
          <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
            {templates.map((t, i) => <span key={i} className="badge badge-neutral">{t.name || t.type || JSON.stringify(t)}</span>)}
          </div>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No reports found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>ID</th><th>Type</th><th>Period</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((r) => (
                <tr key={r.id}>
                  <td className="text-secondary">{r.id.slice(0, 8)}...</td>
                  <td className="text-primary">{r.type || r.report_type}</td>
                  <td className="text-secondary">{r.period || '-'}</td>
                  <td><span className="badge badge-info">{r.status || 'generated'}</span></td>
                  <td><button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleDownload(r.id)}>Download</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}
    </div>
  );
}
