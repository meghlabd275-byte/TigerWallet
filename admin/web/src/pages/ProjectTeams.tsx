/**
 * TigerWallet Admin - Project Teams Management Page
 * CRUD + status control for project teams (mirrors /api/v1/project-teams)
 * Team members management via getMembers/addMember/removeMember
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { projectTeamsAPI } from '../services/api';

interface ProjectTeam {
  id: string;
  team_id: string;
  name: string;
  project_type: string;
  token_symbol: string;
  chain_id: string;
  status: 'active' | 'paused' | 'suspended' | 'halted';
  website: string;
  email: string;
}

interface TeamMember {
  id: string;
  user_id: string;
  name: string;
  email: string;
  role: string;
}

const STATUS_OPTIONS = ['active', 'paused', 'suspended', 'halted'] as const;
type Status = typeof STATUS_OPTIONS[number];

const statusBadgeClass = (status: string): string => {
  switch (status) {
    case 'active': return 'badge-success';
    case 'paused': return 'badge-warning';
    case 'suspended': return 'badge-error';
    case 'halted': return 'badge-neutral';
    default: return 'badge-neutral';
  }
};

export const ProjectTeamsPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [teams, setTeams] = useState<ProjectTeam[]>([]);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [membersLoading, setMembersLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<ProjectTeam | null>(null);
  const [selectedTeam, setSelectedTeam] = useState<ProjectTeam | null>(null);
  const [showMemberForm, setShowMemberForm] = useState(false);
  const [memberFormData, setMemberFormData] = useState({
    user_id: '',
    name: '',
    email: '',
    role: '',
  });
  const [formData, setFormData] = useState({
    team_id: '',
    name: '',
    project_type: '',
    token_symbol: '',
    chain_id: '',
    website: '',
    email: '',
  });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadTeams(); }, []);

  const loadTeams = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await projectTeamsAPI.getAll();
      setTeams(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load project teams');
    } finally {
      setLoading(false);
    }
  };

  const loadMembers = async (teamId: string) => {
    try {
      setMembersLoading(true);
      const res = await projectTeamsAPI.getMembers(teamId);
      setMembers(res.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load team members');
    } finally {
      setMembersLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      team_id: '', name: '', project_type: '', token_symbol: '',
      chain_id: '', website: '', email: '',
    });
    setEditing(null);
  };

  const openCreate = () => { resetForm(); setShowForm(true); };

  const openEdit = (team: ProjectTeam) => {
    setEditing(team);
    setFormData({
      team_id: team.team_id,
      name: team.name,
      project_type: team.project_type,
      token_symbol: team.token_symbol,
      chain_id: team.chain_id,
      website: team.website,
      email: team.email,
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        team_id: formData.team_id,
        name: formData.name,
        project_type: formData.project_type,
        token_symbol: formData.token_symbol,
        chain_id: formData.chain_id,
        website: formData.website,
        email: formData.email,
      };
      if (editing) {
        await projectTeamsAPI.update(editing.id, payload);
      } else {
        await projectTeamsAPI.create(payload);
      }
      setShowForm(false);
      resetForm();
      loadTeams();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save project team');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this project team?')) return;
    try {
      await projectTeamsAPI.delete(id);
      if (selectedTeam && selectedTeam.id === id) {
        setSelectedTeam(null);
        setMembers([]);
      }
      loadTeams();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete project team');
    }
  };

  const handleStatusChange = async (id: string, status: Status) => {
    try {
      await projectTeamsAPI.setStatus(id, status);
      loadTeams();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update status');
    }
  };

  const selectTeam = (team: ProjectTeam) => {
    setSelectedTeam(team);
    loadMembers(team.id);
  };

  // ===== Member management =====
  const resetMemberForm = () => {
    setMemberFormData({ user_id: '', name: '', email: '', role: '' });
  };

  const openAddMember = () => { resetMemberForm(); setShowMemberForm(true); };

  const handleMemberSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTeam) return;
    try {
      const payload = {
        user_id: memberFormData.user_id,
        name: memberFormData.name,
        email: memberFormData.email,
        role: memberFormData.role,
      };
      await projectTeamsAPI.addMember(selectedTeam.id, payload);
      setShowMemberForm(false);
      resetMemberForm();
      loadMembers(selectedTeam.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add member');
    }
  };

  const handleMemberRemove = async (memberId: string) => {
    if (!selectedTeam) return;
    if (!confirm('Remove this member?')) return;
    try {
      await projectTeamsAPI.removeMember(selectedTeam.id, memberId);
      loadMembers(selectedTeam.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove member');
    }
  };

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Project Teams Management</h1>
        <div className="flex gap-2">
          <button className="btn btn-secondary" onClick={toggleTheme}>
            {isDark ? '☀️ Light' : '🌙 Dark'}
          </button>
          <button className="btn btn-primary" onClick={openCreate}>+ New Team</button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {showForm && (
        <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>{editing ? 'Edit Team' : 'New Team'}</h2></div>
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group">
                  <label className="form-label">Team ID</label>
                  <input className="form-input" value={formData.team_id} onChange={(e) => setFormData({ ...formData, team_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Name</label>
                  <input className="form-input" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Project Type</label>
                  <input className="form-input" value={formData.project_type} onChange={(e) => setFormData({ ...formData, project_type: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Token Symbol</label>
                  <input className="form-input" value={formData.token_symbol} onChange={(e) => setFormData({ ...formData, token_symbol: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Chain ID</label>
                  <input className="form-input" value={formData.chain_id} onChange={(e) => setFormData({ ...formData, chain_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Website</label>
                  <input className="form-input" value={formData.website} onChange={(e) => setFormData({ ...formData, website: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Email</label>
                  <input className="form-input" type="email" value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
              </div>
              <div className="flex gap-2 mt-4">
                <button type="submit" className="btn btn-primary">{editing ? 'Update' : 'Create'}</button>
                <button type="button" className="btn btn-secondary" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="flex gap-6" style={{ flexWrap: 'wrap' }}>
        <div className="card" style={{ backgroundColor: colors.bgCard, flex: 2, minWidth: '500px' }}>
          <div className="card-body p-0">
            {loading ? (
              <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
            ) : teams.length === 0 ? (
              <div className="text-center py-8" style={{ color: colors.textSecondary }}>No project teams found</div>
            ) : (
              <table className="table">
                <thead>
                  <tr>
                    <th>Name</th><th>Team ID</th><th>Project Type</th><th>Token</th><th>Chain</th><th>Website</th><th>Email</th><th>Status</th><th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {teams.map((t) => (
                    <tr key={t.id} style={{ cursor: 'pointer', backgroundColor: selectedTeam && selectedTeam.id === t.id ? (isDark ? '#334155' : '#f3f4f6') : 'transparent' }}>
                      <td style={{ color: colors.text }} onClick={() => selectTeam(t)}>{t.name}</td>
                      <td style={{ color: colors.textSecondary }} onClick={() => selectTeam(t)}>{t.team_id}</td>
                      <td style={{ color: colors.textSecondary }} onClick={() => selectTeam(t)}>{t.project_type}</td>
                      <td style={{ color: colors.textSecondary }} onClick={() => selectTeam(t)}>{t.token_symbol}</td>
                      <td style={{ color: colors.textSecondary }} onClick={() => selectTeam(t)}>{t.chain_id}</td>
                      <td style={{ color: colors.textSecondary }} onClick={() => selectTeam(t)}>{t.website}</td>
                      <td style={{ color: colors.textSecondary }} onClick={() => selectTeam(t)}>{t.email}</td>
                      <td><span className={`badge ${statusBadgeClass(t.status)}`}>{t.status}</span></td>
                      <td>
                        <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                          <select className="form-select" style={{ width: 'auto' }} value={t.status} onChange={(e) => handleStatusChange(t.id, e.target.value as Status)}>
                            {STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
                          </select>
                          <button className="btn btn-sm btn-outline" onClick={() => openEdit(t)}>Edit</button>
                          <button className="btn btn-sm btn-danger" onClick={() => handleDelete(t.id)}>Delete</button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        <div className="card" style={{ backgroundColor: colors.bgCard, flex: 1, minWidth: '300px' }}>
          <div className="card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h2 style={{ color: colors.text }}>
              {selectedTeam ? `Members: ${selectedTeam.name}` : 'Team Members'}
            </h2>
            {selectedTeam && <button className="btn btn-sm btn-primary" onClick={openAddMember}>+ Add</button>}
          </div>
          <div className="card-body">
            {!selectedTeam ? (
              <div className="text-center py-8" style={{ color: colors.textSecondary }}>Select a team to view members</div>
            ) : membersLoading ? (
              <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
            ) : members.length === 0 ? (
              <div className="text-center py-8" style={{ color: colors.textSecondary }}>No members found</div>
            ) : (
              <div className="flex flex-col gap-3">
                {members.map((m) => (
                  <div key={m.id} className="flex justify-between items-center p-3 rounded" style={{ border: `1px solid ${colors.border}` }}>
                    <div>
                      <div style={{ color: colors.text, fontWeight: 500 }}>{m.name}</div>
                      <div style={{ color: colors.textSecondary, fontSize: '0.875rem' }}>{m.email}</div>
                      <div style={{ color: colors.textSecondary, fontSize: '0.75rem' }}>Role: {m.role}</div>
                    </div>
                    <button className="btn btn-sm btn-danger" onClick={() => handleMemberRemove(m.id)}>Remove</button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {showMemberForm && selectedTeam && (
        <div className="card mt-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
          <div className="card-header"><h2 style={{ color: colors.text }}>Add Member to {selectedTeam.name}</h2></div>
          <div className="card-body">
            <form onSubmit={handleMemberSubmit}>
              <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                <div className="form-group">
                  <label className="form-label">User ID</label>
                  <input className="form-input" value={memberFormData.user_id} onChange={(e) => setMemberFormData({ ...memberFormData, user_id: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Name</label>
                  <input className="form-input" value={memberFormData.name} onChange={(e) => setMemberFormData({ ...memberFormData, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Email</label>
                  <input className="form-input" type="email" value={memberFormData.email} onChange={(e) => setMemberFormData({ ...memberFormData, email: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
                <div className="form-group">
                  <label className="form-label">Role</label>
                  <input className="form-input" value={memberFormData.role} onChange={(e) => setMemberFormData({ ...memberFormData, role: e.target.value })} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} />
                </div>
              </div>
              <div className="flex gap-2 mt-4">
                <button type="submit" className="btn btn-primary">Add Member</button>
                <button type="button" className="btn btn-secondary" onClick={() => { setShowMemberForm(false); resetMemberForm(); }}>Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default ProjectTeamsPage;
