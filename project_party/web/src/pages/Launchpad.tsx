// Launchpad Page — WL-ProjectParty. Real backend coverage:
// POST /launchpad (create), GET /launchpad (list), GET /launchpad/:id (get),
// POST /launchpad/:id/participate (participate), GET /launchpad/:id/participations.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const STATUSES = ['', 'upcoming', 'active', 'ended', 'cancelled'];

interface LPForm {
  token_id: string; name: string; description: string;
  start_time: string; end_time: string; total_supply: string;
  price_per_token: string; status: string;
}

const EMPTY: LPForm = {
  token_id: '', name: '', description: '', start_time: '', end_time: '',
  total_supply: '', price_per_token: '', status: 'upcoming'
};

function toRFC3339(local: string) {
  if (!local) return undefined;
  const d = new Date(local);
  if (isNaN(d.getTime())) return undefined;
  return d.toISOString();
}

export default function Launchpad() {
  const [projects, setProjects] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<LPForm>(EMPTY);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [selected, setSelected] = useState<any | null>(null);
  const [participations, setParticipations] = useState<any[] | null>(null);
  const [partLoading, setPartLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.listLaunchpadProjects(statusFilter || undefined);
      setProjects(data.launchpad_projects || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load launchpad projects');
    }
    setLoading(false);
  }, [statusFilter]);

  useEffect(() => { load(); }, [load]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.createLaunchpadProject({
        token_id: form.token_id,
        name: form.name,
        description: form.description || undefined,
        start_time: toRFC3339(form.start_time),
        end_time: toRFC3339(form.end_time),
        total_supply: form.total_supply || undefined,
        price_per_token: form.price_per_token || undefined,
        status: form.status || undefined
      });
      setMsg({ type: 'success', text: 'Launchpad project created.' });
      setForm(EMPTY);
      setShowForm(false);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to create launchpad' });
    }
    setSubmitting(false);
  };

  const view = async (id: string) => {
    setMsg(null);
    try {
      const data = await api.getLaunchpadProject(id);
      setSelected(data);
      setParticipations(null);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to load project' });
    }
  };

  const participate = async (id: string, amount: string) => {
    if (!amount) { setMsg({ type: 'error', text: 'Enter an amount to participate.' }); return; }
    setMsg(null);
    try {
      await api.participateInLaunchpad(id, amount);
      setMsg({ type: 'success', text: 'Participation submitted.' });
      loadParticipations(id);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Participation failed' });
    }
  };

  const loadParticipations = async (id: string) => {
    setPartLoading(true);
    try {
      const data = await api.listParticipations(id);
      setParticipations(data.participations || []);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to load participations' });
    }
    setPartLoading(false);
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Launchpad (IDO / Presale)</h1>
        <div className="row-actions">
          <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}>
            {STATUSES.map(s => <option key={s} value={s}>{s || 'All statuses'}</option>)}
          </select>
          <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Create Project'}</button>
        </div>
      </div>
      <p className="subtitle">Browse launchpad projects, view details, participate, and list participations.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>Create Launchpad Project</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Project Name</label>
                <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Total Supply</label>
                <input value={form.total_supply} onChange={e => setForm({ ...form, total_supply: e.target.value })} placeholder="e.g. 1000000" />
              </div>
              <div className="form-field">
                <label>Price Per Token</label>
                <input value={form.price_per_token} onChange={e => setForm({ ...form, price_per_token: e.target.value })} placeholder="e.g. 0.01" />
              </div>
              <div className="form-field">
                <label>Status</label>
                <select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>
                  {STATUSES.filter(s => s).map(s => <option key={s} value={s}>{s}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Start Time</label>
                <input type="datetime-local" value={form.start_time} onChange={e => setForm({ ...form, start_time: e.target.value })} />
              </div>
              <div className="form-field">
                <label>End Time</label>
                <input type="datetime-local" value={form.end_time} onChange={e => setForm({ ...form, end_time: e.target.value })} />
              </div>
              <div className="form-field" style={{ gridColumn: '1 / -1' }}>
                <label>Description</label>
                <textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Creating…' : 'Create'}</button>
              <button type="button" className="secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {selected && (
        <LaunchpadDetail
          project={selected}
          participations={participations}
          partLoading={partLoading}
          onClose={() => { setSelected(null); setParticipations(null); }}
          onParticipate={(amount) => participate(selected.id, amount)}
          onLoadParticipations={() => loadParticipations(selected.id)}
        />
      )}

      {loading ? (
        <div className="state">Loading launchpad projects…</div>
      ) : projects.length === 0 ? (
        <div className="state">No launchpad projects yet.</div>
      ) : (
        <div className="cards-grid">
          {projects.map((p: any) => (
            <div className="card" key={p.id}>
              <div className="card-row"><span>Name</span><span><strong>{p.name}</strong></span></div>
              <div className="card-row"><span>Status</span><span><span className={`badge ${p.status === 'active' ? 'active' : ''}`}>{p.status}</span></span></div>
              <div className="card-row"><span>Total Supply</span><span>{p.total_supply || '-'}</span></div>
              <div className="card-row"><span>Price / Token</span><span>{p.price_per_token || '-'}</span></div>
              <div className="card-row"><span>Sold</span><span>{p.sold_amount || '-'}</span></div>
              <div className="card-row"><span>Start</span><span>{p.start_time ? new Date(p.start_time).toLocaleString() : '-'}</span></div>
              <div className="card-row"><span>End</span><span>{p.end_time ? new Date(p.end_time).toLocaleString() : '-'}</span></div>
              <div className="card-row"><span>Token</span><span title={p.token_id}>{String(p.token_id).slice(0, 8)}…</span></div>
              <div className="row-actions" style={{ marginTop: '0.6rem' }}>
                <button className="secondary" onClick={() => view(p.id)}>Details</button>
                <ParticipateInline onParticipate={(amt) => participate(p.id, amt)} />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ParticipateInline({ onParticipate }: { onParticipate: (amount: string) => void }) {
  const [amount, setAmount] = useState('');
  return (
    <span className="inline-form">
      <input value={amount} onChange={e => setAmount(e.target.value)} placeholder="amount" style={{ width: 110 }} />
      <button onClick={() => onParticipate(amount)}>Participate</button>
    </span>
  );
}

function LaunchpadDetail({
  project, participations, partLoading, onClose, onParticipate, onLoadParticipations
}: {
  project: any; participations: any[] | null; partLoading: boolean;
  onClose: () => void; onParticipate: (amount: string) => void; onLoadParticipations: () => void;
}) {
  const [amount, setAmount] = useState('');
  return (
    <section>
      <div className="section-title">
        <h2>{project.name}</h2>
        <button className="secondary" onClick={onClose}>Close</button>
      </div>
      <div className="two-col">
        <div>
          <div className="card-row"><span>ID</span><span title={project.id}>{String(project.id).slice(0, 8)}…</span></div>
          <div className="card-row"><span>Status</span><span><span className={`badge ${project.status === 'active' ? 'active' : ''}`}>{project.status}</span></span></div>
          <div className="card-row"><span>Token ID</span><span title={project.token_id}>{String(project.token_id).slice(0, 8)}…</span></div>
          <div className="card-row"><span>Total Supply</span><span>{project.total_supply || '-'}</span></div>
          <div className="card-row"><span>Sold Amount</span><span>{project.sold_amount || '-'}</span></div>
          <div className="card-row"><span>Price / Token</span><span>{project.price_per_token || '-'}</span></div>
          <div className="card-row"><span>Start</span><span>{project.start_time ? new Date(project.start_time).toLocaleString() : '-'}</span></div>
          <div className="card-row"><span>End</span><span>{project.end_time ? new Date(project.end_time).toLocaleString() : '-'}</span></div>
          <div className="card-row"><span>Created</span><span>{project.created_at ? new Date(project.created_at).toLocaleString() : '-'}</span></div>
          {project.description && <p className="muted" style={{ marginTop: '0.5rem' }}>{project.description}</p>}
        </div>
        <div>
          <div className="section-title"><h3>Participate</h3></div>
          <div className="inline-form" style={{ marginBottom: '0.75rem' }}>
            <input value={amount} onChange={e => setAmount(e.target.value)} placeholder="amount" />
            <button onClick={() => onParticipate(amount)}>Submit</button>
          </div>
          <div className="section-title">
            <h3>Participations</h3>
            <button className="secondary" onClick={onLoadParticipations}>Load</button>
          </div>
          {partLoading ? <div className="state">Loading…</div> : participations === null ? (
            <p className="muted">Click “Load” to fetch participations.</p>
          ) : participations.length === 0 ? (
            <p className="muted">No participations recorded.</p>
          ) : (
            <div className="coins-table">
              <table>
                <thead><tr><th>ID</th><th>User</th><th>Amount</th><th>Allocated</th><th>Status</th><th>Created</th></tr></thead>
                <tbody>
                  {participations.map((p: any) => (
                    <tr key={p.id}>
                      <td title={p.id}>{String(p.id).slice(0, 8)}…</td>
                      <td title={p.user_id}>{String(p.user_id).slice(0, 8)}…</td>
                      <td>{p.amount}</td>
                      <td>{p.allocated || '-'}</td>
                      <td><span className={`badge ${p.status === 'completed' ? 'active' : ''}`}>{p.status}</span></td>
                      <td>{p.created_at ? new Date(p.created_at).toLocaleString() : '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
