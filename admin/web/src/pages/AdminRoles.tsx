/**
 * TigerWallet Admin - RBAC Admin Roles Management Page
 * Roles CRUD + Permissions CRUD + assign/revoke roles to admins + effective permissions
 * (mirrors /api/v1/roles, /api/v1/permissions, /api/v1/admins/:id/roles, /api/v1/admins/:id/permissions)
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { adminRolesAPI, adminApi } from '../services/api';

interface Role {
  id: string;
  name: string;
  description: string;
  permissions: string[];
  adminCount: number;
  createdAt: string;
}

interface Permission {
  id: string;
  name: string;
  resource: string;
  action: string;
  description: string;
  createdAt: string;
}

interface AdminSummary {
  id: string;
  email: string;
  username: string;
  role: string;
  status: string;
}

type Tab = 'roles' | 'permissions' | 'assignments';

export const AdminRolesPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [tab, setTab] = useState<Tab>('roles');

  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [admins, setAdmins] = useState<AdminSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Role form
  const [showRoleForm, setShowRoleForm] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [roleForm, setRoleForm] = useState({ name: '', description: '', permissions: '' });

  // Permission form
  const [showPermForm, setShowPermForm] = useState(false);
  const [editingPerm, setEditingPerm] = useState<Permission | null>(null);
  const [permForm, setPermForm] = useState({ name: '', resource: '', action: '', description: '' });

  // Assignment panel
  const [selectedAdminId, setSelectedAdminId] = useState<string>('');
  const [effectivePerms, setEffectivePerms] = useState<string[] | null>(null);
  const [assignRoleId, setAssignRoleId] = useState('');

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  useEffect(() => { loadAll(); }, []);

  const loadAll = async () => {
    try {
      setLoading(true); setError(null);
      const [rolesRes, permsRes, adminsRes] = await Promise.all([
        adminRolesAPI.getRoles(),
        adminRolesAPI.getPermissions(),
        adminApi.listAdmins(),
      ]);
      setRoles(rolesRes.data || []);
      setPermissions(permsRes.data || []);
      setAdmins((adminsRes.data as AdminSummary[]) || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load RBAC data');
    } finally {
      setLoading(false);
    }
  };

  // ---- Roles ----
  const resetRoleForm = () => { setRoleForm({ name: '', description: '', permissions: '' }); setEditingRole(null); };
  const openCreateRole = () => { resetRoleForm(); setShowRoleForm(true); };
  const openEditRole = (r: Role) => { setEditingRole(r); setRoleForm({ name: r.name, description: r.description, permissions: Array.isArray(r.permissions) ? r.permissions.join(', ') : '' }); setShowRoleForm(true); };

  const handleRoleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        name: roleForm.name,
        description: roleForm.description,
        permissions: roleForm.permissions.split(',').map((p) => p.trim()).filter(Boolean),
      };
      if (editingRole) await adminRolesAPI.updateRole(editingRole.id, payload); else await adminRolesAPI.createRole(payload);
      setShowRoleForm(false); resetRoleForm(); loadAll();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save role'); }
  };

  const handleDeleteRole = async (id: string) => { if (!confirm('Delete this role?')) return; try { await adminRolesAPI.deleteRole(id); loadAll(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete role'); } };

  // ---- Permissions ----
  const resetPermForm = () => { setPermForm({ name: '', resource: '', action: '', description: '' }); setEditingPerm(null); };
  const openCreatePerm = () => { resetPermForm(); setShowPermForm(true); };
  const openEditPerm = (p: Permission) => { setEditingPerm(p); setPermForm({ name: p.name, resource: p.resource, action: p.action, description: p.description }); setShowPermForm(true); };

  const handlePermSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editingPerm) await adminRolesAPI.updatePermission(editingPerm.id, permForm); else await adminRolesAPI.createPermission(permForm);
      setShowPermForm(false); resetPermForm(); loadAll();
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to save permission'); }
  };

  const handleDeletePerm = async (id: string) => { if (!confirm('Delete this permission?')) return; try { await adminRolesAPI.deletePermission(id); loadAll(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to delete permission'); } };

  // ---- Assignments ----
  const loadEffectivePerms = async (adminId: string) => {
    setSelectedAdminId(adminId);
    setEffectivePerms(null);
    if (!adminId) return;
    try {
      const res = await adminRolesAPI.getEffectivePermissions(adminId);
      const data = res.data as any[];
      setEffectivePerms(Array.isArray(data) ? data.map((p) => (typeof p === 'string' ? p : p.name || p.id)) : []);
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed to load effective permissions'); }
  };

  const handleAssignRole = async () => {
    if (!selectedAdminId || !assignRoleId) return;
    try { await adminRolesAPI.assignRole(selectedAdminId, { roleId: assignRoleId }); setAssignRoleId(''); loadEffectivePerms(selectedAdminId); loadAll(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to assign role'); }
  };

  const handleRevokeRole = async (roleId: string) => {
    if (!selectedAdminId) return;
    if (!confirm('Revoke this role from the admin?')) return;
    try { await adminRolesAPI.revokeRole(selectedAdminId, roleId); loadEffectivePerms(selectedAdminId); loadAll(); } catch (err) { setError(err instanceof Error ? err.message : 'Failed to revoke role'); }
  };

  const tabBtnStyle = (active: boolean): React.CSSProperties => ({
    padding: '8px 16px',
    border: '1px solid',
    borderColor: active ? 'var(--color-primary)' : colors.border,
    backgroundColor: active ? 'var(--color-primary)' : 'transparent',
    color: active ? '#ffffff' : colors.text,
    borderRadius: '6px',
    cursor: 'pointer',
  });

  return (
    <div className="p-6" style={{ color: colors.text }}>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>RBAC - Roles & Permissions</h1>
        <button className="btn btn-secondary" onClick={toggleTheme}>{isDark ? '☀️ Light' : '🌙 Dark'}</button>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      <div className="flex gap-2 mb-6">
        <button style={tabBtnStyle(tab === 'roles')} onClick={() => setTab('roles')}>Roles</button>
        <button style={tabBtnStyle(tab === 'permissions')} onClick={() => setTab('permissions')}>Permissions</button>
        <button style={tabBtnStyle(tab === 'assignments')} onClick={() => setTab('assignments')}>Admin Assignments</button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : (
        <>
          {/* ROLES TAB */}
          {tab === 'roles' && (
            <div>
              <div className="flex justify-end mb-4"><button className="btn btn-primary" onClick={openCreateRole}>+ New Role</button></div>

              {showRoleForm && (
                <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
                  <div className="card-header"><h2 style={{ color: colors.text }}>{editingRole ? 'Edit Role' : 'New Role'}</h2></div>
                  <div className="card-body">
                    <form onSubmit={handleRoleSubmit}>
                      <div className="form-group"><label className="form-label">Name</label><input className="form-input" value={roleForm.name} onChange={(e) => setRoleForm({ ...roleForm, name: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                      <div className="form-group"><label className="form-label">Description</label><input className="form-input" value={roleForm.description} onChange={(e) => setRoleForm({ ...roleForm, description: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                      <div className="form-group"><label className="form-label">Permissions (comma-separated names/IDs)</label><input className="form-input" value={roleForm.permissions} onChange={(e) => setRoleForm({ ...roleForm, permissions: e.target.value })} placeholder="e.g. users.read, users.write" style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                      <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editingRole ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowRoleForm(false); resetRoleForm(); }}>Cancel</button></div>
                    </form>
                  </div>
                </div>
              )}

              <div className="card" style={{ backgroundColor: colors.bgCard }}>
                <div className="card-body p-0">
                  {roles.length === 0 ? (
                    <div className="text-center py-8" style={{ color: colors.textSecondary }}>No roles found</div>
                  ) : (
                    <table className="table">
                      <thead><tr><th>Name</th><th>Description</th><th>Permissions</th><th>Admins</th><th>Created</th><th>Actions</th></tr></thead>
                      <tbody>
                        {roles.map((r) => (
                          <tr key={r.id}>
                            <td style={{ color: colors.text }}>{r.name}</td>
                            <td style={{ color: colors.textSecondary }}>{r.description}</td>
                            <td style={{ color: colors.textSecondary }}>{Array.isArray(r.permissions) ? r.permissions.length : 0}</td>
                            <td style={{ color: colors.textSecondary }}>{r.adminCount}</td>
                            <td style={{ color: colors.textSecondary }}>{new Date(r.createdAt).toLocaleDateString()}</td>
                            <td><div className="flex gap-2"><button className="btn btn-sm btn-outline" onClick={() => openEditRole(r)}>Edit</button><button className="btn btn-sm btn-danger" onClick={() => handleDeleteRole(r.id)}>Delete</button></div></td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* PERMISSIONS TAB */}
          {tab === 'permissions' && (
            <div>
              <div className="flex justify-end mb-4"><button className="btn btn-primary" onClick={openCreatePerm}>+ New Permission</button></div>

              {showPermForm && (
                <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
                  <div className="card-header"><h2 style={{ color: colors.text }}>{editingPerm ? 'Edit Permission' : 'New Permission'}</h2></div>
                  <div className="card-body">
                    <form onSubmit={handlePermSubmit}>
                      <div className="flex gap-4" style={{ flexWrap: 'wrap' }}>
                        <div className="form-group"><label className="form-label">Name</label><input className="form-input" value={permForm.name} onChange={(e) => setPermForm({ ...permForm, name: e.target.value })} required placeholder="e.g. users.read" style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                        <div className="form-group"><label className="form-label">Resource</label><input className="form-input" value={permForm.resource} onChange={(e) => setPermForm({ ...permForm, resource: e.target.value })} required placeholder="e.g. users" style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                        <div className="form-group"><label className="form-label">Action</label><input className="form-input" value={permForm.action} onChange={(e) => setPermForm({ ...permForm, action: e.target.value })} required placeholder="e.g. read" style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                        <div className="form-group" style={{ flexBasis: '100%' }}><label className="form-label">Description</label><input className="form-input" value={permForm.description} onChange={(e) => setPermForm({ ...permForm, description: e.target.value })} required style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }} /></div>
                      </div>
                      <div className="flex gap-2 mt-4"><button type="submit" className="btn btn-primary">{editingPerm ? 'Update' : 'Create'}</button><button type="button" className="btn btn-secondary" onClick={() => { setShowPermForm(false); resetPermForm(); }}>Cancel</button></div>
                    </form>
                  </div>
                </div>
              )}

              <div className="card" style={{ backgroundColor: colors.bgCard }}>
                <div className="card-body p-0">
                  {permissions.length === 0 ? (
                    <div className="text-center py-8" style={{ color: colors.textSecondary }}>No permissions found</div>
                  ) : (
                    <table className="table">
                      <thead><tr><th>Name</th><th>Resource</th><th>Action</th><th>Description</th><th>Created</th><th>Actions</th></tr></thead>
                      <tbody>
                        {permissions.map((p) => (
                          <tr key={p.id}>
                            <td style={{ color: colors.text }}>{p.name}</td>
                            <td style={{ color: colors.textSecondary }}>{p.resource}</td>
                            <td style={{ color: colors.textSecondary }}>{p.action}</td>
                            <td style={{ color: colors.textSecondary }}>{p.description}</td>
                            <td style={{ color: colors.textSecondary }}>{new Date(p.createdAt).toLocaleDateString()}</td>
                            <td><div className="flex gap-2"><button className="btn btn-sm btn-outline" onClick={() => openEditPerm(p)}>Edit</button><button className="btn btn-sm btn-danger" onClick={() => handleDeletePerm(p.id)}>Delete</button></div></td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* ASSIGNMENTS TAB */}
          {tab === 'assignments' && (
            <div>
              <div className="card mb-6" style={{ backgroundColor: colors.bgCard, border: `1px solid ${colors.border}` }}>
                <div className="card-header"><h2 style={{ color: colors.text }}>Select Admin</h2></div>
                <div className="card-body">
                  <div className="form-group">
                    <label className="form-label">Admin</label>
                    <select className="form-select" value={selectedAdminId} onChange={(e) => loadEffectivePerms(e.target.value)} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}>
                      <option value="">-- Select an admin --</option>
                      {admins.map((a) => <option key={a.id} value={a.id}>{a.username} ({a.email})</option>)}
                    </select>
                  </div>

                  {selectedAdminId && (
                    <div className="flex gap-4 mt-4" style={{ flexWrap: 'wrap', alignItems: 'flex-end' }}>
                      <div className="form-group" style={{ minWidth: '240px' }}>
                        <label className="form-label">Assign Role</label>
                        <select className="form-select" value={assignRoleId} onChange={(e) => setAssignRoleId(e.target.value)} style={{ backgroundColor: 'var(--bg-secondary)', color: colors.text }}>
                          <option value="">-- Select a role --</option>
                          {roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
                        </select>
                      </div>
                      <button className="btn btn-primary" onClick={handleAssignRole} disabled={!assignRoleId}>Assign Role</button>
                    </div>
                  )}
                </div>
              </div>

              {selectedAdminId && (
                <div className="card" style={{ backgroundColor: colors.bgCard }}>
                  <div className="card-header"><h2 style={{ color: colors.text }}>Effective Permissions</h2></div>
                  <div className="card-body">
                    {effectivePerms === null ? (
                      <div className="loader"></div>
                    ) : effectivePerms.length === 0 ? (
                      <div style={{ color: colors.textSecondary }}>No effective permissions for this admin.</div>
                    ) : (
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        {effectivePerms.map((perm, idx) => (
                          <span key={idx} className="badge badge-info" style={{ marginBottom: '4px' }}>{perm}</span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {selectedAdminId && (
                <div className="card mt-6" style={{ backgroundColor: colors.bgCard }}>
                  <div className="card-header"><h2 style={{ color: colors.text }}>Assigned Roles</h2></div>
                  <div className="card-body p-0">
                    <table className="table">
                      <thead><tr><th>Role</th><th>Description</th><th>Actions</th></tr></thead>
                      <tbody>
                        {roles.length === 0 ? (
                          <tr><td colSpan={3} className="text-center py-8" style={{ color: colors.textSecondary }}>No roles available</td></tr>
                        ) : (
                          roles.map((r) => (
                            <tr key={r.id}>
                              <td style={{ color: colors.text }}>{r.name}</td>
                              <td style={{ color: colors.textSecondary }}>{r.description}</td>
                              <td><button className="btn btn-sm btn-danger" onClick={() => handleRevokeRole(r.id)}>Revoke</button></td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default AdminRolesPage;
