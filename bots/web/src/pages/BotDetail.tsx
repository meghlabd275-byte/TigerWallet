// Bot Detail Page - GET /bots/:id, /bots/:id/executions, /bots/:id/logs
import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api, Bot, BotExecution, BotLog } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

type Tab = 'executions' | 'logs';

export default function BotDetail() {
  const { isDark } = useTheme();
  const { id } = useParams<{ id: string }>();
  const [bot, setBot] = useState<Bot | null>(null);
  const [executions, setExecutions] = useState<BotExecution[]>([]);
  const [logs, setLogs] = useState<BotLog[]>([]);
  const [tab, setTab] = useState<Tab>('executions');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([
      api.getBot(id),
      api.listBotExecutions(id),
      api.listBotLogs(id),
    ]).then(([b, ex, lg]) => {
      setBot(b);
      setExecutions(ex.executions || []);
      setLogs(lg.logs || []);
      setLoading(false);
    }).catch(err => {
      setError(err.message || 'Failed to load bot');
      setLoading(false);
    });
  }, [id]);

  const transition = async (action: 'start' | 'stop' | 'pause') => {
    if (!id) return;
    setBusy(true);
    setError('');
    try {
      const updated = await api[`${action}Bot`](id);
      setBot(updated);
    } catch (err: any) {
      setError(err.message || `Failed to ${action} bot`);
    } finally {
      setBusy(false);
    }
  };

  const handleDelete = async () => {
    if (!id || !window.confirm('Delete this bot?')) return;
    setBusy(true);
    try {
      await api.deleteBot(id);
      window.location.href = '/bots';
    } catch (err: any) {
      setError(err.message || 'Failed to delete bot');
      setBusy(false);
    }
  };

  const fmtDate = (s: string | null) => (s ? new Date(s).toLocaleString() : '—');

  if (loading) return <div className="bot-detail-page">Loading...</div>;
  if (error && !bot) return (
    <div className="bot-detail-page">
      <div className="error">{error}</div>
      <Link to="/bots" className="back-link">← Back to bots</Link>
    </div>
  );
  if (!bot) return null;

  return (
    <div className="bot-detail-page">
      <Link to="/bots" className="back-link">← Back to bots</Link>
      <header className="page-header">
        <h1>{bot.name} <span className="mode-pill">{isDark ? 'Dark' : 'Light'}</span></h1>
        <div className="bot-actions">
          {bot.status === 'running' ? (
            <button disabled={busy} onClick={() => transition('stop')}>Stop</button>
          ) : (
            <button disabled={busy} onClick={() => transition('start')}>Start</button>
          )}
          {bot.status !== 'paused' && (
            <button disabled={busy} onClick={() => transition('pause')}>Pause</button>
          )}
          <button className="btn-danger" disabled={busy} onClick={handleDelete}>Delete</button>
        </div>
      </header>

      {error && <div className="error">{error}</div>}

      <section className="detail-card">
        <div className="detail-grid">
          <div><span className="detail-label">ID</span><code>{bot.id}</code></div>
          <div><span className="detail-label">Type</span><code>{bot.bot_type}</code></div>
          <div><span className="detail-label">Status</span>
            <span className={`status ${bot.status}`}>{bot.status}</span></div>
          <div><span className="detail-label">Exchange</span>{bot.exchange || '—'}</div>
          <div><span className="detail-label">Pair</span>{bot.pair || '—'}</div>
          <div><span className="detail-label">Created</span>{fmtDate(bot.created_at)}</div>
        </div>
        {bot.config && Object.keys(bot.config).length > 0 && (
          <div className="config-block">
            <span className="detail-label">Config</span>
            <pre>{JSON.stringify(bot.config, null, 2)}</pre>
          </div>
        )}
      </section>

      <div className="tabs">
        <button className={tab === 'executions' ? 'active' : ''} onClick={() => setTab('executions')}>
          Executions ({executions.length})
        </button>
        <button className={tab === 'logs' ? 'active' : ''} onClick={() => setTab('logs')}>
          Logs ({logs.length})
        </button>
      </div>

      {tab === 'executions' ? (
        <div className="tab-panel">
          {executions.length === 0 ? (
            <p>No executions recorded yet.</p>
          ) : (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Status</th>
                  <th>PNL</th>
                  <th>Started</th>
                  <th>Ended</th>
                </tr>
              </thead>
              <tbody>
                {executions.map(ex => (
                  <tr key={ex.id}>
                    <td><span className={`status ${ex.status}`}>{ex.status}</span></td>
                    <td className={parseFloat(ex.pnl || '0') >= 0 ? 'profit' : 'loss'}>{ex.pnl}</td>
                    <td>{fmtDate(ex.started_at)}</td>
                    <td>{fmtDate(ex.ended_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      ) : (
        <div className="tab-panel">
          {logs.length === 0 ? (
            <p>No logs recorded yet.</p>
          ) : (
            <ul className="log-list">
              {logs.map(l => (
                <li key={l.id} className={`log-entry log-${l.level}`}>
                  <span className="log-time">{fmtDate(l.created_at)}</span>
                  <span className="log-level">{l.level}</span>
                  <span className="log-message">{l.message}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
