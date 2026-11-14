import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Login() {
  const { login, register, error } = useAuth();
  const navigate = useNavigate();
  const [mode, setMode] = useState('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [email, setEmail] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      if (mode === 'login') await login(username, password);
      else await register(username, password, email || undefined);
      navigate('/dashboard');
    } catch (_) {
      // the context surfaces the backend message through `error`
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <h1>TigerBots</h1>
      <h2>{mode === 'login' ? 'Sign in' : 'Create account'}</h2>
      <form onSubmit={submit}>
        <label>
          Username
          <input value={username} onChange={(e) => setUsername(e.target.value)} required autoFocus />
        </label>
        <label>
          Password
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </label>
        {mode === 'register' && (
          <label>
            Email (optional)
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </label>
        )}
        {error && <p className="error-banner">{error}</p>}
        <button type="submit" disabled={busy}>{busy ? 'Working…' : (mode === 'login' ? 'Sign in' : 'Register')}</button>
      </form>
      <button className="link-button" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
        {mode === 'login' ? 'Need an account? Register' : 'Already registered? Sign in'}
      </button>
    </div>
  );
}
