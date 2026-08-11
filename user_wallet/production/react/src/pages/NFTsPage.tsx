import React, { useState, useEffect, useCallback } from 'react';
import Header from '../components/Header';
import Sidebar from '../components/Sidebar';
import { useWallet } from '../contexts/WalletContext';
import { WalletService } from '../services/WalletService';

interface NFTItem {
  id: string;
  name: string;
  collection: string;
  image: string;
  price: string;
}

// NFT Gallery Page - Complete
const NFTsPage = () => {
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());
  const [selectedTab, setSelectedTab] = useState('collectibles');
  const [searchQuery, setSearchQuery] = useState('');
  const [nfts, setNfts] = useState<NFTItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadNFTs = useCallback(async () => {
    if (!activeWallet) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = (await walletService.getNFTs(activeWallet.id)) as Array<Record<string, unknown>>;
      const mapped: NFTItem[] = (data ?? []).map((n, i) => ({
        id: String(n.id ?? n.token_id ?? i),
        name: String(n.name ?? `NFT #${n.token_id ?? i}`),
        collection: String(n.collection ?? n.contract ?? ''),
        image: String(n.image ?? n.image_url ?? '🖼️'),
        price: n.price ? String(n.price) : '',
      }));
      setNfts(mapped);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load NFTs');
      setNfts([]);
    } finally {
      setIsLoading(false);
    }
  }, [walletService, activeWallet]);

  useEffect(() => {
    loadNFTs();
  }, [loadNFTs]);

  const activities: NFTItem[] = [];

  const filteredNfts = searchQuery
    ? nfts.filter((n) => n.name.toLowerCase().includes(searchQuery.toLowerCase()) || n.collection.toLowerCase().includes(searchQuery.toLowerCase()))
    : nfts;

  return (
    <div className="app-container">
      <Sidebar />
      <div className="main-content">
        <Header title="NFT Gallery" />
        
        <div className="page-content">
          {/* Search Bar */}
          <div className="search-bar">
            <input
              type="text"
              placeholder="Search NFTs..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="search-input"
            />
          </div>

          {/* Tabs */}
          <div className="tabs">
            <button
              className={`tab ${selectedTab === 'collectibles' ? 'active' : ''}`}
              onClick={() => setSelectedTab('collectibles')}
            >
              Collectibles
            </button>
            <button
              className={`tab ${selectedTab === 'activity' ? 'active' : ''}`}
              onClick={() => setSelectedTab('activity')}
            >
              Activity
            </button>
            <button
              className={`tab ${selectedTab === 'opensea' ? 'active' : ''}`}
              onClick={() => setSelectedTab('opensea')}
            >
              OpenSea
            </button>
          </div>

          {/* Content */}
          {selectedTab === 'collectibles' && (
            <div className="nft-grid">
              {isLoading ? (
                <div className="nft-card">Loading NFTs…</div>
              ) : error ? (
                <div className="nft-card" style={{ color: '#dc2626' }}>{error}</div>
              ) : filteredNfts.length === 0 ? (
                <div className="nft-card">No NFTs found</div>
              ) : (
                filteredNfts.map((nft) => (
                  <div key={nft.id} className="nft-card">
                    <div className="nft-image">{nft.image}</div>
                    <div className="nft-info">
                      <div className="nft-name">{nft.name}</div>
                      <div className="nft-collection">{nft.collection}</div>
                      {nft.price && <div className="nft-price">{nft.price}</div>}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {selectedTab === 'activity' && (
            <div className="activity-list">
              {activities.length === 0 ? (
                <div className="activity-item">No recent activity</div>
              ) : (
                activities.map((activity) => (
                  <div key={activity.id} className="activity-item">
                    <div className="activity-icon">
                      {activity.id.startsWith('sent') ? '📤' : activity.id.startsWith('received') ? '📥' : '📋'}
                    </div>
                    <div className="activity-info">
                      <div className="activity-name">{activity.name}</div>
                      <div className="activity-time">{activity.collection}</div>
                    </div>
                    <div className="activity-amount">{activity.price}</div>
                  </div>
                ))
              )}
            </div>
          )}

          {selectedTab === 'opensea' && (
            <div className="opensea-connect">
              <div className="opensea-icon">🌊</div>
              <h3>OpenSea Integration</h3>
              <p>Connect to OpenSea to view and trade your NFTs</p>
              <button className="connect-btn">Connect OpenSea</button>
            </div>
          )}
        </div>
      </div>

      <style>{`
        .page-content {
          padding: 24px;
        }
        .search-bar {
          margin-bottom: 20px;
        }
        .search-input {
          width: 100%;
          padding: 12px 16px;
          border: 1px solid #e2e8f0;
          border-radius: 8px;
          font-size: 14px;
        }
        .tabs {
          display: flex;
          gap: 8px;
          margin-bottom: 24px;
        }
        .tab {
          padding: 10px 20px;
          background: #f1f5f9;
          border: none;
          border-radius: 8px;
          cursor: pointer;
          font-weight: 500;
        }
        .tab.active {
          background: #f97316;
          color: white;
        }
        .nft-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
          gap: 16px;
        }
        .nft-card {
          background: white;
          border-radius: 12px;
          overflow: hidden;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .nft-image {
          height: 180px;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 60px;
        }
        .nft-info {
          padding: 16px;
        }
        .nft-name {
          font-weight: 600;
          margin-bottom: 4px;
        }
        .nft-collection {
          font-size: 12px;
          color: #64748b;
          margin-bottom: 8px;
        }
        .nft-price {
          color: #f97316;
          font-weight: 600;
        }
        .activity-list {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        .activity-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 16px;
          background: white;
          border-radius: 12px;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .activity-icon {
          font-size: 24px;
        }
        .activity-info {
          flex: 1;
        }
        .activity-name {
          font-weight: 500;
        }
        .activity-time {
          font-size: 12px;
          color: #64748b;
        }
        .activity-amount {
          font-weight: 600;
          color: #f97316;
        }
        .opensea-connect {
          text-align: center;
          padding: 60px 20px;
          background: white;
          border-radius: 12px;
        }
        .opensea-icon {
          font-size: 60px;
          margin-bottom: 16px;
        }
        .opensea-connect h3 {
          margin-bottom: 8px;
        }
        .opensea-connect p {
          color: #64748b;
          margin-bottom: 24px;
        }
        .connect-btn {
          padding: 12px 32px;
          background: #f97316;
          color: white;
          border: none;
          border-radius: 8px;
          font-weight: 600;
          cursor: pointer;
        }
      `}</style>
    </div>
  );
};

export default NFTsPage;
