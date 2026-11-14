import React, { useEffect, useState } from 'react';
import api, { setApiBaseUrl, getApiBaseUrlForDisplay } from '../services/project-party-api';

export default function Settings() {
  const [serverUrl, setServerUrl] = useState(getApiBaseUrlForDisplay());
  const [status, setStatus] = useState(null);
  const [isError, setIsError] = useState(false);
  const [health, setHealth] = useState(null);

  useEffect(() => {
    let alive = true;
    api.health()
      .then(h => { if (alive) setHealth(h); })
      .catch(() => { if (alive) setHealth(null); });
    return () => { alive = false; };
  }, []);

  const save = async () => {
    let saved;
    try {
      saved = setApiBaseUrl(serverUrl);
    } catch (e) {
      setIsError(true);
      setStatus(e.message);
      return;
    }
    setServerUrl(saved);
    setIsError(false);
    setStatus(`Saved. Using ${saved}`);
    try {
      setHealth(await api.health());
    } catch {
      setHealth(null);
    }
  };

  return (
    <div className="settings-page">
      <h1>Settings</h1>

      <h2>Backend server</h2>
      <p className="muted">
        Point this app at your deployment. A hosted backend is not reachable at
        localhost.
      </p>
      <input
        type="url"
        value={serverUrl}
        spellCheck="false"
        onChange={(e) => setServerUrl(e.target.value)}
        placeholder="https://project-party.tigerwallet.com/api/v1"
      />
      <button className="btn" onClick={save}>Save server</button>
      {status && <p className={isError ? 'error-banner' : 'muted'}>{status}</p>}

      <h2>Backend health</h2>
      {health ? (
        <dl className="detail-list">
          <dt>Status</dt><dd>{health.status || 'ok'}</dd>
          <dt>Version</dt><dd>{health.version || '—'}</dd>
        </dl>
      ) : <p className="empty-state">Backend unreachable.</p>}
    </div>
  );
}
