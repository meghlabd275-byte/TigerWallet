/**
 * TigerWallet Super Admin - Project Teams Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function ProjectTeams() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', description: '' });
  const [selectedTeam, setSelectedTeam] = useState<any | null>(null);
  const [members, setMembers] = useState<any[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [memberForm, setMemberForm] = useState({ user_id: '', role: 'member' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getProjectTeams();
      setItems(res.teams || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load project teams');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const loadMembers = async (team: any) => {
    setSelectedTeam(team);
    setMembersLoading(true);
    try {
      const res: any = await superAdminApi.getProjectTeamMembers(team.id);
      setMembers(res.members || res || []);
    } catch (err: any) {
      alert(err?.message || 'Failed to load members');
      setMembers([]);
    } finally {
      setMembersLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createProjectTeam({ name: form.name, description: form.description || undefined });
      setShowForm(false);
      setForm({ name: '', description: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create project team');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>) => {
    setActionLoading(true);
    try {
      await fn();
      load();
      if (selectedTeam) await loadMembers(selectedTeam);
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTeam) return;
    await run(() => superAdminApi.addProjectTeamMember(selectedTeam.id, { user_id: memberForm.user_id, role: memberForm.role }));
    setMemberForm({ user_id: '', role: 'member' });
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Project Teams</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Team'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Project Team</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="form-group"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
            <div className="form-group"><label className="text-secondary">Description</label><input className="input w-full" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No project teams found.</p></div></div>
      ) : (
        <div className="card mb-6"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Description</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((t) => (
                <tr key={t.id}>
                  <td className="text-primary">{t.name}</td>
                  <td className="text-secondary">{t.description || '-'}</td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => loadMembers(t)}>Members</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this team?')) run(() => superAdminApi.deleteProjectTeam(t.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {selectedTeam && (
        <div className="card mt-4"><div className="card-body">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-primary">Members — {selectedTeam.name}</h3>
            <button className="btn btn-secondary" onClick={() => setSelectedTeam(null)}>Close</button>
          </div>

          <form onSubmit={handleAddMember} className="flex gap-3 mb-4">
            <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={memberForm.user_id} onChange={(e) => setMemberForm({ ...memberForm, user_id: e.target.value })} required /></div>
            <div className="form-group flex-1"><label className="text-secondary">Role</label><select className="input w-full" value={memberForm.role} onChange={(e) => setMemberForm({ ...memberForm, role: e.target.value })}><option>member</option><option>lead</option><option>admin</option></select></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Add Member</button>
          </form>

          {membersLoading ? (
            <div className="flex items-center justify-center p-4"><div className="loader"></div></div>
          ) : members.length === 0 ? (
            <p className="text-secondary">No members found.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="table"><thead><tr><th>User ID</th><th>Role</th><th>Actions</th></tr></thead>
                <tbody>
                  {members.map((m) => (
                    <tr key={m.id}>
                      <td className="text-primary">{m.user_id}</td>
                      <td className="text-secondary">{m.role}</td>
                      <td><button className="btn btn-danger" disabled={actionLoading} onClick={() => run(() => superAdminApi.removeProjectTeamMember(selectedTeam.id, m.id))}>Remove</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div></div>
      )}
    </div>
  );
}
