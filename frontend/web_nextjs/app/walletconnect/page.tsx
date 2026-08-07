'use client';

import React, { useState, useCallback } from 'react';

interface WalletConnectSession {
  id: string;
  name: string;
  icon?: string;
  url: string;
  chains: number[];
  methods: string[];
  peerId: string;
  connectedAt: number;
}

interface DApp {
  id: string;
  name: string;
  url: string;
  icon: string;
  description: string;
  category: string;
  chains: number[];
}

const RECOMMENDED_DAPPS: DApp[] = [
  { id: '1', name: 'Uniswap', url: 'https://app.uniswap.org', icon: '🦄', description: 'DEX Aggregator', category: 'DeFi', chains: [1, 56, 42161] },
  { id: '2', name: 'OpenSea', url: 'https://opensea.io', icon: '🌊', description: 'NFT Marketplace', category: 'NFT', chains: [1, 137, 56] },
  { id: '3', name: 'Aave', url: 'https://app.aave.com', icon: '👻', description: 'Lending Protocol', category: 'DeFi', chains: [1, 137, 56] },
  { id: '4', name: 'PancakeSwap', url: 'https://pancakeswap.finance', icon: '🥞', description: 'DEX', category: 'DeFi', chains: [56] },
  { id: '5', name: 'Lido', url: 'https://lido.fi', icon: '💧', description: 'Liquid Staking', category: 'DeFi', chains: [1] },
];

const WC_METHODS = ['eth_requestAccounts', 'eth_sendTransaction', 'eth_sign', 'personal_sign', 'eth_signTypedData', 'eth_accounts', 'eth_blockNumber', 'eth_call', 'eth_estimateGas', 'web3_clientVersion'];

export default function WalletConnect() {
  const [sessions, setSessions] = useState<WalletConnectSession[]>([]);
  const [showQR, setShowQR] = useState(false);
  const [qrUri, setQrUri] = useState('');
  const [connecting, setConnecting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [selectedDapp, setSelectedDapp] = useState<DApp | null>(null);
  const [customUri, setCustomUri] = useState('');

  const handleConnect = useCallback(async (_dapp?: DApp) => {
    setConnecting(false);
    setShowQR(false);
    setMessage({ type: 'error', text: 'WalletConnect transport is not configured. No session or QR URI was created.' });
  }, []);

  const handleCustomConnect = useCallback(async () => {
    if (!customUri.startsWith('wc:')) {
      setMessage({ type: 'error', text: 'Invalid WalletConnect URI' });
      return;
    }
    setConnecting(false);
    setShowQR(false);
    setMessage({ type: 'error', text: 'WalletConnect transport is not configured. The supplied URI was not connected.' });
  }, [customUri]);

  const handleDisconnect = useCallback(async (sessionId: string) => {
    await new Promise(resolve => setTimeout(resolve, 500));
    setSessions(prev => prev.filter(s => s.id !== sessionId));
    setMessage({ type: 'success', text: 'Disconnected from dApp' });
  }, []);

  const formatTime = (timestamp: number): string => { const diff = Date.now() - timestamp; const hours = Math.floor(diff / 3600000); if (hours < 1) return 'Just now'; if (hours < 24) return `${hours}h ago`; return new Date(timestamp).toLocaleDateString(); };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">WalletConnect v2</h1></div>
            <nav className="flex gap-4"><a href="/wallet" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Wallet</a></nav>
          </div>
        </div>
      </header>
      {message && <div className="fixed top-20 right-4 z-50"><div className={`px-6 py-3 rounded-lg shadow-lg ${message.type === 'success' ? 'bg-green-500' : 'bg-red-500'} text-white`}>{message.text}</div></div>}
      {showQR && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md w-full mx-4">
            <div className="text-center mb-4"><div className="text-4xl mb-2">📱</div><h3 className="text-xl font-semibold">Scan QR Code</h3><p className="text-slate-500 text-sm">{selectedDapp ? `Connect to ${selectedDapp.name}` : 'Scan QR from dApp'}</p></div>
            <div className="bg-white p-4 rounded-lg mb-4"><div className="w-48 h-48 mx-auto bg-slate-200 dark:bg-slate-700 rounded-lg flex items-center justify-center"><div className="text-center"><div className="text-6xl mb-2">⬛</div><p className="text-xs text-slate-500">QR Code</p></div></div></div>
            <p className="text-xs text-slate-500 text-center mb-4">Or copy the URI below:</p>
            <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3 mb-4"><code className="text-xs break-all">{qrUri.slice(0, 50)}...</code></div>
            <button onClick={() => setShowQR(false)} className="w-full bg-slate-200 dark:bg-slate-700 py-2 rounded-lg">Cancel</button>
          </div>
        </div>
      )}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-8">
          <h2 className="text-xl font-semibold mb-4">Connected dApps ({sessions.length})</h2>
          {sessions.length === 0 ? <div className="text-center py-8"><div className="text-6xl mb-4">🔌</div><h3 className="text-lg font-semibold mb-2">No Connected dApps</h3><p className="text-slate-500">Connect to dApps using WalletConnect v2</p></div> : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {sessions.map((session) => (
                <div key={session.id} className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-3"><div className="flex items-center gap-3"><div className="text-2xl">{session.icon}</div><div><div className="font-semibold">{session.name}</div><div className="text-xs text-slate-500">Connected {formatTime(session.connectedAt)}</div></div></div><div className="w-2 h-2 bg-green-500 rounded-full"></div></div>
                  <button onClick={() => handleDisconnect(session.id)} className="w-full bg-red-100 dark:bg-red-900/30 text-red-600 py-2 rounded-lg text-sm">Disconnect</button>
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-8">
          <h2 className="text-xl font-semibold mb-4">Connect Manually</h2>
          <div className="flex gap-4"><input type="text" value={customUri} onChange={(e) => setCustomUri(e.target.value)} placeholder="Paste WalletConnect URI (wc:...)" className="flex-1 bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3" /><button onClick={handleCustomConnect} disabled={connecting || !customUri} className="bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white px-6 py-3 rounded-lg">{connecting ? 'Connecting...' : 'Connect'}</button></div>
        </div>
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-4">Recommended dApps</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {RECOMMENDED_DAPPS.map((dapp) => (
              <div key={dapp.id} className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4 hover:shadow-lg transition-shadow">
                <div className="flex items-center gap-3 mb-3"><div className="text-3xl">{dapp.icon}</div><div><div className="font-semibold">{dapp.name}</div><div className="text-xs text-slate-500">{dapp.description}</div></div></div>
                <div className="flex gap-2"><button onClick={() => handleConnect(dapp)} disabled={connecting} className="flex-1 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-2 rounded-lg text-sm">Connect</button><a href={dapp.url} target="_blank" rel="noopener noreferrer" className="px-4 py-2 bg-slate-200 dark:bg-slate-600 rounded-lg text-sm">Visit</a></div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
