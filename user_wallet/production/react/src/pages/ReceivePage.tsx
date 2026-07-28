/**
 * Receive Page - Get wallet address to receive tokens
 */

import React from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';

function ReceivePage() {
  const { activeWallet, getAddress } = useWallet();
  const { theme } = useTheme();

  const address = getAddress(activeWallet?.chain as any);

  const copyAddress = () => {
    navigator.clipboard.writeText(address);
  };

  const shareAddress = () => {
    if (navigator.share) {
      navigator.share({
        title: 'My Wallet Address',
        text: address,
      });
    }
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Receive</h1>

      {/* QR Code Placeholder */}
      <div className={`card text-center mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <div className="w-48 h-48 mx-auto mb-4 bg-white rounded-lg flex items-center justify-center">
          <div className="text-6xl">📱</div>
        </div>
        <p className="text-sm opacity-60 mb-4">Scan to receive</p>
        
        <div className="text-xs opacity-60 mb-2">{activeWallet?.chain?.name || 'Ethereum'}</div>
      </div>

      {/* Address */}
      <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-2">Your Address</h3>
        <div className="flex gap-2">
          <input
            type="text"
            value={address}
            readOnly
            className="input flex-1 font-mono text-sm"
          />
          <button onClick={copyAddress} className="btn btn-secondary">
            Copy
          </button>
          <button onClick={shareAddress} className="btn btn-secondary">
            Share
          </button>
        </div>
      </div>

      {/* Networks */}
      <div className={`card mt-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Other Networks</h3>
        <div className="space-y-2">
          {['Bitcoin', 'Solana', 'Polygon', 'BNB Chain'].map(network => (
            <div key={network} className="flex justify-between items-center p-3 bg-amber-500/10 rounded-lg">
              <span>{network}</span>
              <button className="text-amber-500 text-sm font-medium">Get Address</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default ReceivePage;
