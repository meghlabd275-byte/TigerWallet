/**
 * TigerWallet Super Admin - Knowledge Base Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function KnowledgeBase() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ title: '', content: '', category: 'general', tags: '', status: 'draft' });
  const [search, setSearch] = useState('');
  const [results, setResults] = useState<any[] | null>(null);

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getArticles();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load articles');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createArticle({
        title: form.title,
        content: form.content,
        category: form.category,
        tags: form.tags.split(',').map((s) => s.trim()).filter(Boolean),
        status: form.status,
      });
      setShowForm(false);
      setForm({ title: '', content: '', category: 'general', tags: '', status: 'draft' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create article');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>) => {
    setActionLoading(true);
    try {
      await fn();
      load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!search) { setResults(null); return; }
    setActionLoading(true);
    try {
      const r: any = await superAdminApi.searchKnowledgeBase(undefined, search);
      setResults(r.articles || r.data || r.items || []);
    } catch (err: any) {
      alert(err?.message || 'Search failed');
    } finally {
      setActionLoading(false);
    }
  };

  const displayed = results !== null ? results : items;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Knowledge Base</h1>

      <div className="card mb-4"><div className="card-body">
        <div className="flex gap-3">
          <div className="form-group flex-1"><label className="text-secondary">Search</label><input className="input w-full" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search articles" /></div>
          <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-primary" disabled={actionLoading} onClick={handleSearch}>Search</button></div>
          <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-secondary" onClick={() => { setResults(null); setSearch(''); }}>Clear</button></div>
        </div>
      </div></div>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Article'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Article</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Title</label><input className="input w-full" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Category</label><input className="input w-full" value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Status</label><select className="input w-full" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}><option>draft</option><option>published</option></select></div>
            </div>
            <div className="form-group"><label className="text-secondary">Tags (comma-separated)</label><input className="input w-full" value={form.tags} onChange={(e) => setForm({ ...form, tags: e.target.value })} /></div>
            <div className="form-group"><label className="text-secondary">Content</label><textarea className="input w-full" rows={5} value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} required /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : displayed.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">{results !== null ? 'No search results.' : 'No articles found.'}</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Title</th><th>Category</th><th>Status</th><th>Views</th><th>Actions</th></tr></thead>
            <tbody>
              {displayed.map((a) => (
                <tr key={a.id}>
                  <td className="text-primary">{a.title}</td>
                  <td className="text-secondary">{a.category}</td>
                  <td><span className={`badge ${a.status === 'published' ? 'badge-success' : a.status === 'archived' ? 'badge-neutral' : 'badge-warning'}`}>{a.status}</span></td>
                  <td className="text-secondary">{a.views ?? 0}</td>
                  <td><div className="flex gap-2">
                    {a.status !== 'archived' && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.archiveArticle(a.id))}>Archive</button>}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this article?')) run(() => superAdminApi.deleteArticle(a.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}
    </div>
  );
}
