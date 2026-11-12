// Receive Page — show wallet address + QR code for receiving funds.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';
import { drawQr } from '../utils/qr';

export default function Receive() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) setWalletId(data.wallets[0].id);
    }).catch(() => {});
  }, []);

  const selected = wallets.find((w) => w.id === walletId);

  useEffect(() => {
    if (selected) {
      drawQr(document.getElementById('receive-qr') as HTMLCanvasElement | null, selected.address, {
        scale: 6,
        ecc: 'M',
      });
    }
  }, [selected]);

  const copy = () => {
    if (selected) {
      navigator.clipboard?.writeText(selected.address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="receive-page">
      <h1>Receive</h1>
      <div className="form-group">
        <label>Wallet</label>
        <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
          {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)} · Chain {w.chain_id}</option>)}
        </select>
      </div>
      {selected ? (
        <div className="receive-card">
          <canvas id="receive-qr" className="qr-canvas" />
          <p className="wallet-address mono">{selected.address}</p>
          <button className="primary-btn" onClick={copy}>{copied ? '✓ Copied' : '📋 Copy Address'}</button>
          <p className="hint">Send only {selected.chain_id === 1 ? 'Ethereum (ETH/ERC-20)' : `chain ${selected.chain_id}`} assets to this address.</p>
        </div>
      ) : (
        <p>No wallet selected. Create or import one first.</p>
      )}
    </div>
  );
}
