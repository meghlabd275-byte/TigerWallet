// NFTs Page - Production Ready
import React, { useState, useEffect } from 'react';
import { walletApi, NFT, NFTCollection } from '../services/api';
import { wsService } from '../services/api';
import Header from '../components/Header';
import Sidebar from '../components/Sidebar';

const NFTsPage = () => {
  const [selectedTab, setSelectedTab] = useState('collectibles');
  const [searchQuery, setSearchQuery] = useState('');
  const [nfts, setNfts] = useState<NFT[]>([]);
  const [collections, setCollections] = useState<NFTCollection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedWallet, setSelectedWallet] = useState<string>('');
  const [wallets, setWallets] = useState<any[]>([]);

  useEffect(() => {
    loadWallets();
  }, []);

  useEffect(() => {
    if (selectedWallet) {
      loadNFTs();
    }
  }, [selectedWallet]);

  const loadWallets = async () => {
    try {
      const walletList = await walletApi.getWallets();
      setWallets(walletList);
      if (walletList.length > 0) {
        setSelectedWallet(walletList[0].id);
      }
    } catch (err) {
      console.error('Failed to load wallets:', err);
    }
  };

  const loadNFTs = async () => {
    if (!selectedWallet) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const wallet = wallets.find(w => w.id === selectedWallet);
      if (!wallet) return;
      
      // Fetch NFTs from API
      const nftList = await walletApi.getNFTs(wallet.address, wallet.chain);
      setNfts(nftList);
      
      // Extract unique collections
      const collectionMap = new Map<string, NFTCollection>();
      nftList.forEach((nft: NFT) => {
        if (!collectionMap.has(nft.collectionAddress)) {
          collectionMap.set(nft.collectionAddress, {
            id: nft.collectionAddress,
            name: nft.name.split('#')[0],
            symbol: '',
            address: nft.collectionAddress,
            chain: nft.chain,
            totalSupply: 0,
            floorPrice: nft.price || '0',
            imageUrl: nft.imageUrl
          });
        }
      });
      setCollections(Array.from(collectionMap.values()));
    } catch (err: any) {
      console.error('Failed to load NFTs:', err);
      setError(err.message || 'Failed to load NFTs');
      // Keep empty on error
      setNfts([]);
    } finally {
      setLoading(false);
    }
  };

  const filteredNFTs = nfts.filter(nft => 
    nft.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    nft.collectionAddress.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="app-container">
      <Sidebar />
      <div className="main-content">
        <Header title="NFT Gallery" />
        
        <div className="page-content">
          {/* Wallet Selector */}
          {wallets.length > 0 && (
            <div className="wallet-selector">
              <select 
                value={selectedWallet}
                onChange={(e) => setSelectedWallet(e.target.value)}
              >
                {wallets.map(wallet => (
                  <option key={wallet.id} value={wallet.id}>
                    {wallet.name} ({wallet.chain})
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Error Message */}
          {error && (
            <div className="error-message">{error}</div>
          )}

          {/* Loading State */}
          {loading && (
            <div className="loading">Loading NFTs...</div>
          )}

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
              Collectibles ({filteredNFTs.length})
            </button>
            <button
              className={`tab ${selectedTab === 'collections' ? 'active' : ''}`}
              onClick={() => setSelectedTab('collections')}
            >
              Collections ({collections.length})
            </button>
          </div>

          {/* Content */}
          {!loading && selectedTab === 'collectibles' && (
            <div className="nft-grid">
              {filteredNFTs.length > 0 ? (
                filteredNFTs.map((nft) => (
                  <div key={nft.id || nft.tokenId} className="nft-card">
                    <div className="nft-image">
                      {nft.imageUrl ? (
                        <img src={nft.imageUrl} alt={nft.name} />
                      ) : (
                        '🖼️'
                      )}
                    </div>
                    <div className="nft-info">
                      <div className="nft-name">{nft.name}</div>
                      <div className="nft-collection">{nft.collectionAddress.slice(0, 6)}...</div>
                      {nft.price && (
                        <div className="nft-price">{nft.price}</div>
                      )}
                    </div>
                  </div>
                ))
              ) : (
                <div className="empty-state">
                  <p>No NFTs found</p>
                </div>
              )}
            </div>
          )}

          {!loading && selectedTab === 'collections' && (
            <div className="nft-grid">
              {collections.length > 0 ? (
                collections.map((collection) => (
                  <div key={collection.id} className="nft-card collection-card">
                    <div className="nft-image">
                      {collection.imageUrl ? (
                        <img src={collection.imageUrl} alt={collection.name} />
                      ) : (
                        '🎨'
                      )}
                    </div>
                    <div className="nft-info">
                      <div className="nft-name">{collection.name}</div>
                      <div className="nft-collection">Floor: {collection.floorPrice}</div>
                    </div>
                  </div>
                ))
              ) : (
                <div className="empty-state">
                  <p>No collections found</p>
                </div>
              )}
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
