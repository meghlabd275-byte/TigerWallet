// NFTs Page — view ERC-721 NFTs and transfer them on-chain.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';

const CHAIN_OPTIONS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' },
  { id: 10, label: 'Optimism' },
  { id: 8453, label: 'Base' },
];

interface NFT { contract: string; token_id: string; name?: string; image?: string; }

export default function NFTs() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [address, setAddress] = useState('');
  const [chainId, setChainId] = useState(1);
  const [nfts, setNfts] = useState<NFT[]>([]);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [transferTarget, setTransferTarget] = useState<NFT | null>(null);
  const [toAddress, setToAddress] = useState('');
  const [password, setPassword] = useState('');
  const [walletId, setWalletId] = useState('');

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) {
        setAddress(data.wallets[0].address);
        setChainId(data.wallets[0].chain_id);
        setWalletId(data.wallets[0].id);
      }
    }).catch(() => {});
  }, []);

  const loadNFTs = async () => {
    if (!address) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.getNFTs(address, chainId);
      const list = (data as { nfts?: NFT[] }).nfts || [];
      setNfts(list);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load NFTs');
      setNfts([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (address) loadNFTs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [address, chainId]);

  const handleTransfer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!transferTarget) return;
    setError('');
    setBusy(true);
    try {
      await api.transferNFT({
        walletId,
        password,
        to: toAddress,
        tokenId: transferTarget.token_id,
        contractAddress: transferTarget.contract,
        chainId,
      });
      setTransferTarget(null);
      setToAddress('');
      setPassword('');
      loadNFTs();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Transfer failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="nfts-page">
      <h1>My NFTs</h1>
      {error && <div className="error">{error}</div>}
      <div className="filter-row">
        <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
          {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
        </select>
        <input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="0x…" className="address-input" />
      </div>
      {loading ? <p>Loading…</p> : nfts.length === 0 ? (
        <p>No NFTs found for this address.</p>
      ) : (
        <div className="nfts-grid">
          {nfts.map((nft, i) => (
            <div key={`${nft.contract}-${nft.token_id}-${i}`} className="nft-card">
              {nft.image ? <img src={nft.image} alt={nft.name || ''} className="nft-img" /> : <div className="nft-placeholder">🖼️</div>}
              <h4>{nft.name || `#${nft.token_id}`}</h4>
              <p className="mono small">{nft.contract.slice(0, 10)}…</p>
              <p className="mono small">Token {nft.token_id}</p>
              <button onClick={() => setTransferTarget(nft)}>Send</button>
            </div>
          ))}
        </div>
      )}
      {transferTarget && (
        <div className="modal-backdrop">
          <form className="modal" onSubmit={handleTransfer}>
            <h3>Transfer NFT #{transferTarget.token_id}</h3>
            {error && <div className="error">{error}</div>}
            <div className="form-group">
              <label>To Address</label>
              <input value={toAddress} onChange={(e) => setToAddress(e.target.value)} required placeholder="0x…" />
            </div>
            <div className="form-group">
              <label>Wallet Password</label>
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
            </div>
            <button type="submit" className="primary-btn" disabled={busy}>{busy ? 'Transferring…' : 'Transfer'}</button>
            <button type="button" className="link-btn" onClick={() => setTransferTarget(null)}>Cancel</button>
          </form>
        </div>
      )}
    </div>
  );
}
