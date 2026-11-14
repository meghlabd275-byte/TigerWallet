import React, { useEffect, useState, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import api from '../services/bot-api';

export default function BotDetail() {
  const { id } = useParams();
  const [bot, setBot] = useState(null);
  const [executions, setExecutions] = useState([]);
  const [logs, setLogs] = useState([]);
  const [error, setError] = useState(null);
  const [tab, setTab] = useState('executions');
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    Promise.all([
      api.getBot(id),
      api.listBotExecutions(id).catch(() => ({ executions: [] })),
      api.listBotLogs(id).catch(() => ({ logs: [] })),
    ])
      .then(([b, ex, lg]) => {
        setBot(b && b.bot ? b.bot : b);
        setExecutions((ex && (ex.executions || ex.items)) || []);
        setLogs((lg && (lg.logs || lg.items)) || []);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(load, [load]);

  if (loading) return <div>Loading…</div>;

  return (
    <div className="bot-detail">
      <p><Link to="/bots">← All bots</Link></p>
      <h1>{bot ? bot.name : `Bot ${id}`}</h1>
      {error && <p className="error-banner">{error}</p>}
      {bot && (
        <dl className="detail-list">
          <dt>Type</dt><dd>{bot.bot_type}</dd>
          <dt>Status</dt><dd>{bot.status}</dd>
          <dt>Exchange</dt><dd>{bot.exchange || '—'}</dd>
          <dt>Pair</dt><dd>{bot.pair || '—'}</dd>
        </dl>
      )}

      <div className="tabs">
        <button className={tab === 'executions' ? 'active' : ''} onClick={() => setTab('executions')}>Executions</button>
        <button className={tab === 'logs' ? 'active' : ''} onClick={() => setTab('logs')}>Logs</button>
      </div>

      {tab === 'executions' && (
        executions.length === 0 ? <p className="empty-state">No executions recorded.</p> : (
          <table className="data-table">
            <thead><tr><th>Time</th><th>Action</th><th>Status</th><th>Detail</th></tr></thead>
            <tbody>
              {executions.map((x, i) => (
                <tr key={x.id || i}>
                  <td>{x.created_at || x.timestamp || '—'}</td>
                  <td>{x.action || x.type || '—'}</td>
                  <td>{x.status || '—'}</td>
                  <td>{x.message || x.detail || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      )}

      {tab === 'logs' && (
        logs.length === 0 ? <p className="empty-state">No logs recorded.</p> : (
          <ul className="record-list">
            {logs.map((l, i) => (
              <li key={l.id || i} className="record-item">
                <span className="log-time">{l.created_at || l.timestamp || ''}</span>{' '}
                <span className={`log-level ${l.level || ''}`}>{l.level || 'info'}</span>{' '}
                {l.message || l.msg || ''}
              </li>
            ))}
          </ul>
        )
      )}
    </div>
  );
}
