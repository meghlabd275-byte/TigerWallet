import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function NFTs() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [wallets, setWallets] = useState([]);
  const [walletId, setWalletId] = useState('');
  const [network, setNetwork] = useState('ethereum');
  const [nfts, setNfts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState('');

  const [transfer, setTransfer] = useState(null); // nft being transferred
  const [toAddress, setToAddress] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [info, setInfo] = useState('');

  useEffect(() => {
    let alive = true;
    api.getWallets()
      .then((data) => {
        if (!alive) return;
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0) setWalletId(list[0].id || list[0].wallet_id || '');
        setLoading(false);
      })
      .catch(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const loadNFTs = async () => {
    setError('');
    setInfo('');
    const wallet = wallets.find((w) => (w.id || w.wallet_id) === walletId);
    if (!wallet) { setError('Select a wallet'); return; }
    setFetching(true);
    try {
      const data = await api.getNFTs(wallet.address, CHAIN_IDS[network] || 1);
      setNfts(data.nfts || data.tokens || data.assets || []);
    } catch (err) {
      setError(err.message || 'Failed to load NFTs');
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    if (walletId) loadNFTs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [walletId, network]);

  const doTransfer = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    if (!transfer) return;
    if (!toAddress.trim()) { setError('Recipient address is required'); return; }
    if (password.length < 8) { setError('Wallet password is required (min 8 chars)'); return; }
    setBusy(true);
    try {
      const res = await api.transferNFT({
        walletId,
        password,
        to: toAddress.trim(),
        tokenId: transfer.token_id || transfer.tokenId || transfer.id,
        contractAddress: transfer.contract_address || transfer.contractAddress || transfer.address,
        chainId: CHAIN_IDS[network] || 1,
      });
      const hash = res && (res.hash || res.tx_hash || res.transactionHash || res.txHash);
      setInfo(hash ? `NFT transfer submitted — Tx hash: ${hash}` : 'NFT transfer submitted to the blockchain network.');
      setTransfer(null);
      setToAddress('');
      setPassword('');
    } catch (err) {
      setError(err.message || 'NFT transfer failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>NFTs</h1>
      </header>

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one first to view NFTs.</p>
      ) : (
        <>
          <div className="send-form" style={{ maxWidth: '600px' }}>
            {error && <div className="error">{error}</div>}
            {info && <div className="success-banner"><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}

            <label>Wallet</label>
            <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
              {wallets.map((w, idx) => (
                <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                  {w.label} — {w.address ? w.address.slice(0, 10) : ''}…
                </option>
              ))}
            </select>

            <label>Chain</label>
            <select value={network} onChange={(e) => setNetwork(e.target.value)}>
              {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
            </select>

            <div className="send-actions">
              <button type="button" onClick={loadNFTs} disabled={fetching}>
                {fetching ? 'Loading…' : 'Reload NFTs'}
              </button>
            </div>
          </div>

          {transfer && (
            <form className="import-form" style={{ maxWidth: '600px' }} onSubmit={doTransfer}>
              <h3 style={{ marginBottom: '8px' }}>
                Transfer — {(transfer.name || transfer.contract_address || transfer.contractAddress || 'NFT').toString().slice(0, 40)}
              </h3>
              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Recipient address</label>
              <input placeholder="0x..." value={toAddress} onChange={(e) => setToAddress(e.target.value)} required />
              <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Wallet password</label>
              <input type="password" placeholder="Password (min 8 chars)" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} required />
              <div className="mnemonic-actions">
                <button type="submit" disabled={busy}>{busy ? 'Transferring…' : 'Transfer NFT'}</button>
                <button type="button" className="link-btn" onClick={() => { setTransfer(null); setToAddress(''); setPassword(''); }}>Cancel</button>
              </div>
            </form>
          )}

          {nfts.length === 0 ? (
            <p style={{ marginTop: '16px' }}>No NFTs found for this wallet on this chain.</p>
          ) : (
            <div className="wallets-grid" style={{ marginTop: '20px' }}>
              {nfts.map((nft, idx) => (
                <div key={idx} className="wallet-card">
                  <h3>{nft.name || `Token #${nft.token_id || nft.tokenId || idx}`}</h3>
                  {nft.contract_address || nft.contractAddress || nft.address ? (
                    <p className="address">{nft.contract_address || nft.contractAddress || nft.address}</p>
                  ) : null}
                  {(nft.token_id || nft.tokenId) !== undefined && (
                    <p className="network">Token ID: {nft.token_id || nft.tokenId}</p>
                  )}
                  <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                    <button onClick={() => { setTransfer(nft); setInfo(''); }}>📤 Transfer</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default NFTs;
