/**
 * TigerWallet Super Admin - Admin Roles & Permissions (RBAC) Page
 * Structured custom roles + granular permissions + per-admin assignment.
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function AdminRoles() {
  const [roles, setRoles] = useState<any[]>([]);
  const [permissions, setPermissions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const [showRoleForm, setShowRoleForm] = useState(false);
  const [roleForm, setRoleForm] = useState({ name: '', description: '', permissions: '' });

  const [showPermForm, setShowPermForm] = useState(false);
  const [permForm, setPermForm] = useState({ name: '', description: '', category: '' });

  // per-admin role management panel
  const [adminId, setAdminId] = useState('');
  const [effectivePerms, setEffectivePerms] = useState<string[] | null>(null);
  const [assignRoleId, setAssignRoleId] = useState('');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const [rolesRes, permsRes]: any = await Promise.all([
        superAdminApi.getAdminRoles(),
        superAdminApi.getAdminPermissions(),
      ]);
      setRoles(rolesRes.roles || rolesRes.data || rolesRes || []);
      setPermissions(permsRes.permissions || permsRes.data || permsRes || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load roles/permissions');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

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

  const handleCreateRole = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createAdminRole({
        name: roleForm.name,
        description: roleForm.description || undefined,
        permissions: roleForm.permissions ? roleForm.permissions.split(',').map((s) => s.trim()).filter(Boolean) : undefined,
      });
      setShowRoleForm(false);
      setRoleForm({ name: '', description: '', permissions: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create role');
    } finally {
      setActionLoading(false);
    }
  };

  const handleCreatePermission = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createAdminPermission({
        name: permForm.name,
        description: permForm.description || undefined,
        category: permForm.category || undefined,
      });
      setShowPermForm(false);
      setPermForm({ name: '', description: '', category: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create permission');
    } finally {
      setActionLoading(false);
    }
  };

  const handleViewEffective = async () => {
    if (!adminId) return;
    setActionLoading(true);
    try {
      const res: any = await superAdminApi.getAdminEffectivePermissions(adminId);
      setEffectivePerms(res.permissions || []);
    } catch (err: any) {
      alert(err?.message || 'Failed to load effective permissions');
      setEffectivePerms([]);
    } finally {
      setActionLoading(false);
    }
  };

  const renderPerms = (perms: any) => {
    if (!perms) return <span className="text-secondary">—</span>;
    const arr: string[] = Array.isArray(perms) ? perms : (typeof perms === 'string' ? (perms.startsWith('[') ? JSON.parse(perms) : perms.split(',')) : []);
    if (arr.length === 0) return <span className="text-secondary">—</span>;
    return <div className="flex gap-1" style={{ flexWrap: 'wrap' }}>{arr.map((p, i) => <span key={i} className="badge badge-neutral">{p}</span>)}</div>;
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Admin Roles & Permissions</h1>

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : (
        <>
          {/* Roles section */}
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold text-primary">Roles</h2>
            <button className="btn btn-primary" onClick={() => setShowRoleForm((s) => !s)}>
              {showRoleForm ? 'Cancel' : 'New Role'}
            </button>
          </div>

          {showRoleForm && (
            <div className="card mb-6"><div className="card-body">
              <h3 className="text-primary mb-4">Create Role</h3>
              <form onSubmit={handleCreateRole} className="flex flex-col gap-3">
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={roleForm.name} onChange={(e) => setRoleForm({ ...roleForm, name: e.target.value })} required /></div>
                  <div className="form-group flex-1"><label className="text-secondary">Description</label><input className="input w-full" value={roleForm.description} onChange={(e) => setRoleForm({ ...roleForm, description: e.target.value })} /></div>
                </div>
                <div className="form-group"><label className="text-secondary">Permissions (comma-separated)</label><input className="input w-full" placeholder="read:users, write:settings" value={roleForm.permissions} onChange={(e) => setRoleForm({ ...roleForm, permissions: e.target.value })} /></div>
                <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
              </form>
            </div></div>
          )}

          {roles.length === 0 ? (
            <div className="card mb-6"><div className="card-body text-center py-8"><p className="text-secondary">No roles found.</p></div></div>
          ) : (
            <div className="card mb-6"><div className="card-body overflow-x-auto">
              <table className="table"><thead><tr><th>Name</th><th>Description</th><th>Permissions</th><th>System</th><th>Active</th><th>Actions</th></tr></thead>
                <tbody>
                  {roles.map((r) => (
                    <tr key={r.id}>
                      <td className="text-primary">{r.name}</td>
                      <td className="text-secondary">{r.description || '—'}</td>
                      <td style={{ minWidth: 200 }}>{renderPerms(r.permissions)}</td>
                      <td className="text-secondary">{r.is_system ? 'yes' : 'no'}</td>
                      <td><span className={`badge ${r.is_active ? 'badge-success' : 'badge-neutral'}`}>{r.is_active ? 'active' : 'inactive'}</span></td>
                      <td><div className="flex gap-2">
                        {!r.is_system && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.updateAdminRole(r.id, { is_active: !r.is_active }))}>{r.is_active ? 'Deactivate' : 'Activate'}</button>}
                        {!r.is_system && <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this role?')) run(() => superAdminApi.deleteAdminRole(r.id)); }}>Delete</button>}
                      </div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div></div>
          )}

          {/* Permissions section */}
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold text-primary">Permissions</h2>
            <button className="btn btn-primary" onClick={() => setShowPermForm((s) => !s)}>
              {showPermForm ? 'Cancel' : 'New Permission'}
            </button>
          </div>

          {showPermForm && (
            <div className="card mb-6"><div className="card-body">
              <h3 className="text-primary mb-4">Create Permission</h3>
              <form onSubmit={handleCreatePermission} className="flex flex-col gap-3">
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={permForm.name} onChange={(e) => setPermForm({ ...permForm, name: e.target.value })} required /></div>
                  <div className="form-group flex-1"><label className="text-secondary">Category</label><input className="input w-full" value={permForm.category} onChange={(e) => setPermForm({ ...permForm, category: e.target.value })} /></div>
                </div>
                <div className="form-group"><label className="text-secondary">Description</label><input className="input w-full" value={permForm.description} onChange={(e) => setPermForm({ ...permForm, description: e.target.value })} /></div>
                <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
              </form>
            </div></div>
          )}

          {permissions.length === 0 ? (
            <div className="card mb-6"><div className="card-body text-center py-8"><p className="text-secondary">No permissions found.</p></div></div>
          ) : (
            <div className="card mb-6"><div className="card-body overflow-x-auto">
              <table className="table"><thead><tr><th>Name</th><th>Category</th><th>Description</th><th>Active</th></tr></thead>
                <tbody>
                  {permissions.map((p) => (
                    <tr key={p.id}>
                      <td className="text-primary">{p.name}</td>
                      <td className="text-secondary">{p.category}</td>
                      <td className="text-secondary">{p.description || '—'}</td>
                      <td><span className={`badge ${p.is_active ? 'badge-success' : 'badge-neutral'}`}>{p.is_active ? 'active' : 'inactive'}</span></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div></div>
          )}

          {/* Per-admin role assignment panel */}
          <div className="card"><div className="card-body">
            <h2 className="text-xl font-semibold text-primary mb-4">Admin Role Assignments</h2>
            <div className="flex gap-3 mb-4">
              <div className="form-group flex-1"><label className="text-secondary">Admin ID</label><input className="input w-full" value={adminId} onChange={(e) => setAdminId(e.target.value)} placeholder="admin UUID" /></div>
              <div className="form-group flex-1"><label className="text-secondary">Role to assign</label>
                <select className="input w-full" value={assignRoleId} onChange={(e) => setAssignRoleId(e.target.value)}>
                  <option value="">— select role —</option>
                  {roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
                </select>
              </div>
            </div>
            <div className="flex gap-2 mb-4">
              <button className="btn btn-primary" disabled={actionLoading || !adminId || !assignRoleId} onClick={() => run(() => superAdminApi.assignAdminRole(adminId, assignRoleId))}>Assign Role</button>
              <button className="btn btn-secondary" disabled={actionLoading || !adminId || !assignRoleId} onClick={() => run(() => superAdminApi.revokeAdminRole(adminId, assignRoleId))}>Revoke Role</button>
              <button className="btn btn-secondary" disabled={actionLoading || !adminId} onClick={handleViewEffective}>View Effective Permissions</button>
            </div>
            {effectivePerms !== null && (
              <div>
                <h3 className="text-primary mb-2">Effective Permissions for {adminId}</h3>
                {effectivePerms.length === 0 ? (
                  <p className="text-secondary">No permissions granted.</p>
                ) : (
                  <div className="flex gap-1" style={{ flexWrap: 'wrap' }}>
                    {effectivePerms.map((p, i) => <span key={i} className="badge badge-neutral">{p}</span>)}
                  </div>
                )}
              </div>
            )}
          </div></div>
        </>
      )}
    </div>
  );
}
