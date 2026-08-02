import React, { useState } from 'react';
import Header from '../components/Header';
import Sidebar from '../components/Sidebar';

// NFT Gallery Page - Complete
const NFTsPage = () => {
  const [selectedTab, setSelectedTab] = useState('collectibles');
  const [searchQuery, setSearchQuery] = useState('');

  const nfts = [
    { id: 1, name: 'Bored Ape #1234', collection: 'Bored Ape Yacht Club', image: '🦍', price: '45.5 ETH' },
    { id: 2, name: 'CryptoPunk #5678', collection: 'CryptoPunks', image: '👾', price: '32.0 ETH' },
    { id: 3, name: 'Azuki #9012', collection: 'Azuki', image: '🥷', price: '15.2 ETH' },
    { id: 4, name: 'Doodle #3456', collection: 'Doodles', image: '🎨', price: '3.5 ETH' },
    { id: 5, name: 'Moonbird #7890', collection: 'Moonbirds', image: '🐦', price: '8.1 ETH' },
    { id: 6, name: 'Pudgy #2345', collection: 'Pudgy Penguins', image: '🐧', price: '2.8 ETH' },
  ];

  const activities = [
    { id: 1, type: 'sent', name: 'Bored Ape #1234', time: '2 hours ago', amount: '-1 NFT' },
    { id: 2, type: 'received', name: 'CryptoPunk #5678', time: '1 day ago', amount: '+1 NFT' },
    { id: 3, type: 'listed', name: 'Azuki #9012', time: '2 days ago', amount: '2.5 ETH' },
  ];

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
              {nfts.map((nft) => (
                <div key={nft.id} className="nft-card">
                  <div className="nft-image">{nft.image}</div>
                  <div className="nft-info">
                    <div className="nft-name">{nft.name}</div>
                    <div className="nft-collection">{nft.collection}</div>
                    <div className="nft-price">{nft.price}</div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {selectedTab === 'activity' && (
            <div className="activity-list">
              {activities.map((activity) => (
                <div key={activity.id} className="activity-item">
                  <div className="activity-icon">
                    {activity.type === 'sent' ? '📤' : activity.type === 'received' ? '📥' : '📋'}
                  </div>
                  <div className="activity-info">
                    <div className="activity-name">
                      {activity.type === 'sent' ? 'Sent' : activity.type === 'received' ? 'Received' : 'Listed'} {activity.name}
                    </div>
                    <div className="activity-time">{activity.time}</div>
                  </div>
                  <div className="activity-amount">{activity.amount}</div>
                </div>
              ))}
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
