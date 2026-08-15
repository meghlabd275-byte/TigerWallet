// Bots Page - Manage Bots
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Bots() {
  const { isDark } = useTheme();
  const [bots, setBots] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({ name: '', strategy: 'grid', network: 'ethereum', trading_pairs: ['BTC/USDT'] });

  useEffect(() => { loadBots(); }, []);

  const loadBots = () => {
    api.getBots().then(data => {
      setBots(data.bots || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.createBot(formData);
      setShowCreate(false);
      loadBots();
    } catch (err) { alert('Failed to create bot'); }
  };

  const handleStart = async (id: string) => {
    try {
      await api.startBot(id);
      loadBots();
    } catch (err) { alert('Failed to start bot'); }
  };

  const handleStop = async (id: string) => {
    try {
      await api.stopBot(id);
      loadBots();
    } catch (err) { alert('Failed to stop bot'); }
  };

  return (
    <div className="bots-page">
      <header className="page-header">
        <h1>My Bots</h1>
        <button className="btn-primary" onClick={() => setShowCreate(!showCreate)}>
          {isDark ? '🌙' : '☀️'} + Create Bot
        </button>
      </header>

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          <input placeholder="Bot Name" value={formData.name} onChange={e => setFormData({...formData, name: e.target.value})} required />
          <select value={formData.strategy} onChange={e => setFormData({...formData, strategy: e.target.value})}>
            <option value="grid">Grid Trading</option>
            <option value="dca">DCA Bot</option>
            <option value="arbitrage">Arbitrage</option>
            <option value="momentum">Momentum</option>
            <option value="ai">AI Trading</option>
          </select>
          <select value={formData.network} onChange={e => setFormData({...formData, network: e.target.value})}>
            <option value="ethereum">Ethereum</option>
            <option value="bsc">BNB Chain</option>
          </select>
          <button type="submit">Create</button>
        </form>
      )}

      {loading ? <p>Loading...</p> : bots.length === 0 ? (
        <p>No bots yet. Create one to start trading!</p>
      ) : (
        <div className="bots-grid">
          {bots.map((bot: any) => (
            <div key={bot.id} className="bot-card">
              <h3>{bot.name}</h3>
              <p>Strategy: {bot.strategy}</p>
              <p>Status: <span className={`status ${bot.status}`}>{bot.status}</span></p>
              <p>Profit: ${bot.total_profit}</p>
              <p>Trades: {bot.total_trades}</p>
              <div className="bot-actions">
                {bot.status === 'running' ? (
                  <button onClick={() => handleStop(bot.id)}>Stop</button>
                ) : (
                  <button onClick={() => handleStart(bot.id)}>Start</button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
