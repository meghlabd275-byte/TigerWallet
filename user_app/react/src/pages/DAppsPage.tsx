// DApps Page
import React, { useState } from 'react';
import './DAppsPage.css';

const DAppsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'discover' | 'bookmarks' | 'history'>('discover');

  const featuredDapps = [
    { id: '1', name: 'Uniswap', category: 'DeFi', icon: '🦄', url: 'https://app.uniswap.org' },
    { id: '2', name: 'Aave', category: 'DeFi', icon: '👻', url: 'https://app.aave.com' },
    { id: '3', name: 'OpenSea', category: 'NFT', icon: '🌊', url: 'https://opensea.io' },
    { id: '4', name: 'OpenSea', category: 'NFT', icon: '🌊', url: 'https://opensea.io' },
    { id: '5', name: 'OpenSea', category: 'NFT', icon: '🌊', url: 'https://opensea.io' },
    { id: '6', name: 'OpenSea', category: 'NFT', icon: '🌊', url: 'https://opensea.io' },
  ];

  const categories = ['All', 'DeFi', 'NFT', 'Games', 'Social', 'Tools'];

  const openDApp = (url: string) => {
    window.open(url, '_blank');
  };

  return (
    <div className="dapps-page">
      <div className="page-header">
        <h1>DApps</h1>
      </div>

      {/* Tabs */}
      <div className="tabs">
        <button
          className={`tab ${activeTab === 'discover' ? 'active' : ''}`}
          onClick={() => setActiveTab('discover')}
        >
          Discover
        </button>
        <button
          className={`tab ${activeTab === 'bookmarks' ? 'active' : ''}`}
          onClick={() => setActiveTab('bookmarks')}
        >
          Bookmarks
        </button>
        <button
          className={`tab ${activeTab === 'history' ? 'active' : ''}`}
          onClick={() => setActiveTab('history')}
        >
          History
        </button>
      </div>

      {activeTab === 'discover' && (
        <div className="discover-section">
          {/* Search */}
          <div className="search-bar">
            <span>🔍</span>
            <input type="text" placeholder="Search DApps..." />
          </div>

          {/* Categories */}
          <div className="categories">
            {categories.map(category => (
              <button key={category} className="category-btn">
                {category}
              </button>
            ))}
          </div>

          {/* Featured */}
          <div className="featured-section">
            <h2>Featured</h2>
            <div className="dapps-grid">
              {featuredDapps.map(dapp => (
                <div
                  key={dapp.id}
                  className="dapp-card"
                  onClick={() => openDApp(dapp.url)}
                >
                  <div className="dapp-icon">{dapp.icon}</div>
                  <div className="dapp-info">
                    <span className="dapp-name">{dapp.name}</span>
                    <span className="dapp-category">{dapp.category}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Popular */}
          <div className="popular-section">
            <h2>Popular</h2>
            <div className="dapps-list">
              {featuredDapps.map(dapp => (
                <div
                  key={dapp.id}
                  className="dapp-row"
                  onClick={() => openDApp(dapp.url)}
                >
                  <div className="dapp-icon">{dapp.icon}</div>
                  <div className="dapp-info">
                    <span className="dapp-name">{dapp.name}</span>
                    <span className="dapp-category">{dapp.category}</span>
                  </div>
                  <button className="visit-btn">Visit</button>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'bookmarks' && (
        <div className="empty-state">
          <span>🔖</span>
          <p>No bookmarks yet</p>
          <p className="hint">Bookmark your favorite DApps for quick access</p>
        </div>
      )}

      {activeTab === 'history' && (
        <div className="empty-state">
          <span>📜</span>
          <p>No browsing history</p>
          <p className="hint">DApps you visit will appear here</p>
        </div>
      )}
    </div>
  );
};

export default DAppsPage;
