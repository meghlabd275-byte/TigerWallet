// DApps Page - Production Ready
import React, { useState, useEffect } from 'react';
import { walletApi, DApp } from '../services/api';
import './DAppsPage.css';

const DAppsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'discover' | 'bookmarks' | 'history'>('discover');
  const [searchQuery, setSearchQuery] = useState('');
  const [dapps, setDapps] = useState<DApp[]>([]);
  const [bookmarks, setBookmarks] = useState<DApp[]>([]);
  const [history, setHistory] = useState<DApp[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedCategory, setSelectedCategory] = useState('All');

  const categories = ['All', 'DeFi', 'NFT', 'Games', 'Social', 'Tools', 'Bridge', 'Staking', 'Exchange'];

  useEffect(() => {
    loadDApps();
    loadBookmarks();
    loadHistory();
  }, []);

  const loadDApps = async () => {
    setLoading(true);
    try {
      const dappList = await walletApi.getDApps();
      setDapps(dappList);
    } catch (err) {
      console.error('Failed to load DApps:', err);
      // Keep empty on error
      setDapps([]);
    } finally {
      setLoading(false);
    }
  };

  const loadBookmarks = () => {
    try {
      const saved = localStorage.getItem('dapp_bookmarks');
      if (saved) {
        setBookmarks(JSON.parse(saved));
      }
    } catch (err) {
      console.error('Failed to load bookmarks:', err);
    }
  };

  const loadHistory = () => {
    try {
      const saved = localStorage.getItem('dapp_history');
      if (saved) {
        setHistory(JSON.parse(saved));
      }
    } catch (err) {
      console.error('Failed to load history:', err);
    }
  };

  const toggleBookmark = (dapp: DApp) => {
    const isBookmarked = bookmarks.some(b => b.id === dapp.id);
    let newBookmarks;
    if (isBookmarked) {
      newBookmarks = bookmarks.filter(b => b.id !== dapp.id);
    } else {
      newBookmarks = [...bookmarks, dapp];
    }
    setBookmarks(newBookmarks);
    localStorage.setItem('dapp_bookmarks', JSON.stringify(newBookmarks));
  };

  const addToHistory = (dapp: DApp) => {
    const newHistory = [dapp, ...history.filter(h => h.id !== dapp.id)].slice(0, 50);
    setHistory(newHistory);
    localStorage.setItem('dapp_history', JSON.stringify(newHistory));
  };

  const filteredDapps = dapps.filter(dapp => {
    const matchesSearch = dapp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      dapp.description?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'All' || dapp.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const openDApp = (dapp: DApp) => {
    addToHistory(dapp);
    window.open(dapp.url, '_blank');
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
          Bookmarks ({bookmarks.length})
        </button>
        <button
          className={`tab ${activeTab === 'history' ? 'active' : ''}`}
          onClick={() => setActiveTab('history')}
        >
          History ({history.length})
        </button>
      </div>

      {activeTab === 'discover' && (
        <div className="discover-section">
          {/* Search */}
          <div className="search-bar">
            <span>🔍</span>
            <input 
              type="text" 
              placeholder="Search DApps..." 
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          {/* Categories */}
          <div className="categories">
            {categories.map(category => (
              <button 
                key={category} 
                className={`category-btn ${selectedCategory === category ? 'active' : ''}`}
                onClick={() => setSelectedCategory(category)}
              >
                {category}
              </button>
            ))}
          </div>

          {/* Loading */}
          {loading && <div className="loading">Loading DApps...</div>}

          {/* Featured - Only show first 6 */}
          {!loading && (
            <div className="featured-section">
              <h2>Featured</h2>
              <div className="dapps-grid">
                {filteredDapps.slice(0, 6).map(dapp => (
                  <div
                    key={dapp.id}
                    className="dapp-card"
                    onClick={() => openDApp(dapp)}
                  >
                    <div className="dapp-icon">{dapp.logoUrl || '🌐'}</div>
                    <div className="dapp-info">
                      <span className="dapp-name">{dapp.name}</span>
                      <span className="dapp-category">{dapp.category}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Popular - Show all filtered */}
          {!loading && filteredDapps.length > 0 && (
            <div className="popular-section">
              <h2>All DApps ({filteredDapps.length})</h2>
              <div className="dapps-list">
                {filteredDapps.map(dapp => (
                  <div
                    key={dapp.id}
                    className="dapp-row"
                  >
                    <div className="dapp-icon" onClick={() => openDApp(dapp)}>
                      {dapp.logoUrl || '🌐'}
                    </div>
                    <div className="dapp-info" onClick={() => openDApp(dapp)}>
                      <span className="dapp-name">{dapp.name}</span>
                      <span className="dapp-category">{dapp.category}</span>
                    </div>
                    <div className="dapp-actions">
                      <button 
                        className="bookmark-btn"
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleBookmark(dapp);
                        }}
                      >
                        {bookmarks.some(b => b.id === dapp.id) ? '⭐' : '☆'}
                      </button>
                      <button className="visit-btn" onClick={() => openDApp(dapp)}>Visit</button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === 'bookmarks' && (
        <div className="bookmarks-section">
          {bookmarks.length > 0 ? (
            <div className="dapps-list">
              {bookmarks.map(dapp => (
                <div key={dapp.id} className="dapp-row">
                  <div className="dapp-icon" onClick={() => openDApp(dapp)}>
                    {dapp.logoUrl || '🌐'}
                  </div>
                  <div className="dapp-info" onClick={() => openDApp(dapp)}>
                    <span className="dapp-name">{dapp.name}</span>
                    <span className="dapp-category">{dapp.category}</span>
                  </div>
                  <div className="dapp-actions">
                    <button 
                      className="bookmark-btn"
                      onClick={() => toggleBookmark(dapp)}
                    >
                      ⭐
                    </button>
                    <button className="visit-btn" onClick={() => openDApp(dapp)}>Visit</button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <span>🔖</span>
              <p>No bookmarks yet</p>
              <p className="hint">Bookmark your favorite DApps for quick access</p>
            </div>
          )}
        </div>
      )}

      {activeTab === 'history' && (
        <div className="history-section">
          {history.length > 0 ? (
            <div className="dapps-list">
              {history.map(dapp => (
                <div key={dapp.id} className="dapp-row">
                  <div className="dapp-icon" onClick={() => openDApp(dapp)}>
                    {dapp.logoUrl || '🌐'}
                  </div>
                  <div className="dapp-info" onClick={() => openDApp(dapp)}>
                    <span className="dapp-name">{dapp.name}</span>
                    <span className="dapp-category">{dapp.category}</span>
                  </div>
                  <button className="visit-btn" onClick={() => openDApp(dapp)}>Visit</button>
                </div>
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <span>📜</span>
              <p>No browsing history</p>
              <p className="hint">DApps you visit will appear here</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default DAppsPage;
