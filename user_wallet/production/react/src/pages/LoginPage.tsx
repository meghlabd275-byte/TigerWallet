/**
 * Login Page - Create or Import Wallet
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';

function LoginPage() {
  const navigate = useNavigate();
  const { login, isLoading } = useAuth();
  const { createWallet, importFromMnemonic } = useWallet();
  const { theme, toggleTheme } = useTheme();
  
  const [mode, setMode] = useState<'login' | 'create' | 'import'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [walletPassword, setWalletPassword] = useState('');
  const [selectedChain, setSelectedChain] = useState('ethereum');
  const [step, setStep] = useState(1);
  const [error, setError] = useState('');

  const chains = [
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB' },
    { id: 'arbitrum', name: 'Arbitrum', symbol: 'ARB' },
    { id: 'solana', name: 'Solana', symbol: 'SOL' },
  ];

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await login({ email, password });
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Login failed');
    }
  };

  const handleCreateWallet = async () => {
    setError('');
    // The canonical wallet-api backend generates a REAL BIP-39 mnemonic
    // (CSPRNG entropy + checksum) when POST /wallets is called without a
    // mnemonic. The flow below requests creation first, then displays the
    // backend-generated mnemonic for backup confirmation. The client NEVER
    // fabricates a mnemonic itself.
    try {
      const newWallet = await createWallet(undefined, walletPassword, selectedChain as any);
      setMnemonic(newWallet.mnemonic || '');
      setStep(2);
    } catch (err: any) {
      setError(err.message || 'Failed to create wallet');
    }
  };

  const handleConfirmCreate = async () => {
    setError('');
    // The wallet was already created (with a backend-generated mnemonic) in
    // handleCreateWallet. Here the user has confirmed they backed up the
    // mnemonic, so we just authenticate and proceed.
    try {
      await login({ email, password });
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Failed to login');
    }
  };

  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await importFromMnemonic(mnemonic, walletPassword, selectedChain as any);
      await login({ email, password });
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Failed to import wallet');
    }
  };

  return (
    <div className={`min-h-screen flex items-center justify-center ${theme === 'dark' ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="w-full max-w-md p-8">
        {/* Theme Toggle */}
        <button 
          onClick={toggleTheme}
          className={`absolute top-4 right-4 p-2 rounded-lg ${theme === 'dark' ? 'bg-gray-800 text-amber-500' : 'bg-gray-200 text-gray-700'}`}
        >
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>

        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-amber-500">TigerWallet</h1>
          <p className={`mt-2 ${theme === 'dark' ? 'text-gray-400' : 'text-gray-600'}`}>Enterprise Web3 Wallet</p>
        </div>

        <div className={`flex mb-6 rounded-lg p-1 ${theme === 'dark' ? 'bg-gray-800' : 'bg-gray-200'}`}>
          <button onClick={() => setMode('login')} className={`flex-1 py-2 rounded-md text-sm font-medium ${mode === 'login' ? 'bg-amber-500 text-black' : 'text-gray-400'}`}>Login</button>
          <button onClick={() => { setMode('create'); setStep(1); }} className={`flex-1 py-2 rounded-md text-sm font-medium ${mode === 'create' ? 'bg-amber-500 text-black' : 'text-gray-400'}`}>Create</button>
          <button onClick={() => setMode('import')} className={`flex-1 py-2 rounded-md text-sm font-medium ${mode === 'import' ? 'bg-amber-500 text-black' : 'text-gray-400'}`}>Import</button>
        </div>

        {error && <div className="mb-4 p-3 bg-red-500/20 border border-red-500 rounded-lg text-red-400 text-sm">{error}</div>}

        {mode === 'login' && (
          <form onSubmit={handleLogin} className="space-y-4">
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" className={`w-full px-4 py-3 border rounded-lg ${theme === 'dark' ? 'bg-gray-800 border-gray-700 text-white' : 'bg-white border-gray-300 text-gray-900'}`} required />
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Password" className={`w-full px-4 py-3 border rounded-lg ${theme === 'dark' ? 'bg-gray-800 border-gray-700 text-white' : 'bg-white border-gray-300 text-gray-900'}`} required />
            <button type="submit" disabled={isLoading} className="w-full py-3 bg-amber-500 text-black font-semibold rounded-lg">{isLoading ? 'Loading...' : 'Login'}</button>
          </form>
        )}

        {mode === 'create' && step === 1 && (
          <div className="space-y-4">
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white" required />
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Password" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white" required />
            <select value={selectedChain} onChange={(e) => setSelectedChain(e.target.value)} className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white">
              {chains.map(c => <option key={c.id} value={c.id}>{c.name} ({c.symbol})</option>)}
            </select>
            <button onClick={handleCreateWallet} className="w-full py-3 bg-amber-500 text-black font-semibold rounded-lg">Generate Recovery Phrase</button>
          </div>
        )}

        {mode === 'create' && step === 2 && (
          <div className="space-y-4">
            <div className="p-4 bg-amber-500/10 border border-amber-500 rounded-lg">
              <p className="text-amber-500 text-sm font-medium">Save your recovery phrase!</p>
            </div>
            <div className="grid grid-cols-3 gap-2 p-4 bg-gray-800 rounded-lg">
              {mnemonic.split(' ').map((word, i) => (
                <div key={i} className="flex items-center gap-2"><span className="text-gray-500 text-xs">{i+1}.</span><span className="text-white font-mono text-sm">{word}</span></div>
              ))}
            </div>
            <input type="password" value={walletPassword} onChange={(e) => setWalletPassword(e.target.value)} placeholder="Confirm Wallet Password" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white" required />
            <div className="flex gap-3">
              <button onClick={() => setStep(1)} className="flex-1 py-3 bg-gray-700 text-white rounded-lg">Back</button>
              <button onClick={handleConfirmCreate} disabled={isLoading} className="flex-1 py-3 bg-amber-500 text-black font-semibold rounded-lg">Confirm</button>
            </div>
          </div>
        )}

        {mode === 'import' && (
          <form onSubmit={handleImport} className="space-y-4">
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white" required />
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Password" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white" required />
            <textarea value={mnemonic} onChange={(e) => setMnemonic(e.target.value)} placeholder="Recovery phrase (12 or 24 words)" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white h-24" required />
            <input type="password" value={walletPassword} onChange={(e) => setWalletPassword(e.target.value)} placeholder="Wallet Password" className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white" required />
            <select value={selectedChain} onChange={(e) => setSelectedChain(e.target.value)} className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white">
              {chains.map(c => <option key={c.id} value={c.id}>{c.name} ({c.symbol})</option>)}
            </select>
            <button type="submit" disabled={isLoading} className="w-full py-3 bg-amber-500 text-black font-semibold rounded-lg">{isLoading ? 'Importing...' : 'Import Wallet'}</button>
          </form>
        )}
      </div>
    </div>
  );
}

export default LoginPage;
