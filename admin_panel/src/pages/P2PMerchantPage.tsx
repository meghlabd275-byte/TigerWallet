// P2P Merchant Management Admin Page
// Production-ready with API integration

import React, { useState, useEffect, useCallback } from 'react';
import './P2PMerchantPage.css';
import { 
  getMerchants, 
  getMerchantStats, 
  getTiers, 
  getPendingCollateral,
  approveCollateral as apiApproveCollateral,
  releaseCollateral as apiReleaseCollateral,
  updateMerchantStatus,
  updateAdStatus,
  getAdvertisements,
  getTopMerchants,
  defaultTiers,
  COLLATERAL_REQUIREMENTS,
  P2PMerchant,
  MerchantTier,
  MerchantAdvertisement,
  CollateralDeposit,
  MerchantStats
} from '../services/p2pMerchantService';

const P2PMerchantPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'merchants' | 'advertisements' | 'tiers' | 'analytics' | 'settings'>('merchants');
  const [merchants, setMerchants] = useState<P2PMerchant[]>([]);
  const [advertisements, setAdvertisements] = useState<MerchantAdvertisement[]>([]);
  const [tiers, setTiers] = useState<MerchantTier[]>(defaultTiers);
  const [pendingCollateral, setPendingCollateral] = useState<CollateralDeposit[]>([]);
  const [stats, setStats] = useState<MerchantStats | null>(null);
  const [topByVolume, setTopByVolume] = useState<P2PMerchant[]>([]);
  const [topByTrades, setTopByTrades] = useState<P2PMerchant[]>([]);
  
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [filterVerification, setFilterVerification] = useState('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  const [selectedMerchant, setSelectedMerchant] = useState<P2PMerchant | null>(null);
  const [showMerchantModal, setShowMerchantModal] = useState(false);

  // Load data
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Load tiers (use default if API fails)
      try {
        const tiersData = await getTiers();
        setTiers(tiersData);
      } catch {
        setTiers(defaultTiers);
      }

      // Load merchants
      const merchantsData = await getMerchants({
        status: filterStatus !== 'all' ? filterStatus : undefined,
        verification: filterVerification !== 'all' ? filterVerification : undefined,
        search: searchTerm || undefined,
      });
      setMerchants(merchantsData.merchants);

      // Load advertisements
      const adsData = await getAdvertisements();
      setAdvertisements(adsData.ads);

      // Load stats
      const statsData = await getMerchantStats();
      setStats(statsData);

      // Load pending collateral
      const collateralData = await getPendingCollateral();
      setPendingCollateral(collateralData);

      // Load top merchants
      const topVolume = await getTopMerchants('volume', 10);
      const topTrades = await getTopMerchants('trades', 10);
      setTopByVolume(topVolume);
      setTopByTrades(topTrades);

    } catch (err: any) {
      setError(err.message || 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, [filterStatus, filterVerification, searchTerm]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Handlers
  const handleSuspendMerchant = async (merchantId: string) => {
    try {
      const merchant = merchants.find(m => m.id === merchantId);
      if (!merchant) return;
      
      const newStatus = merchant.status === 'suspended' ? 'active' : 'suspended';
      await updateMerchantStatus(merchantId, newStatus);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to update merchant status');
    }
  };

  const handleBanMerchant = async (merchantId: string) => {
    try {
      await updateMerchantStatus(merchantId, 'banned');
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to ban merchant');
    }
  };

  const handleApproveCollateral = async (depositId: string) => {
    try {
      await apiApproveCollateral(depositId);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to approve collateral');
    }
  };

  const handleReleaseCollateral = async (merchantId: string) => {
    try {
      await apiReleaseCollateral(merchantId);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to release collateral');
    }
  };

  const handleToggleAd = async (adId: string) => {
    try {
      const ad = advertisements.find(a => a.id === adId);
      if (!ad) return;
      
      const newStatus = ad.status === 'active' ? 'paused' : 'active';
      await updateAdStatus(adId, newStatus);
      await loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to update advertisement');
    }
  };

  const getTierInfo = (tierId: string): MerchantTier | undefined => {
    return tiers.find(t => t.id === tierId);
  };

  const getTierColor = (tierId: string) => {
    const tier = getTierInfo(tierId);
    return tier?.color || '#6c757d';
  };

  const getVerificationColor = (level: string) => {
    const colors: Record<string, string> = {
      unverified: '#6c757d',
      email: '#17a2b8',
      phone: '#ffc107',
      kyc: '#28a745',
      advanced: '#4361ee',
    };
    return colors[level] || '#6c757d';
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: '#28a745',
      suspended: '#ffc107',
      banned: '#dc3545',
      paused: '#6c757d',
      closed: '#6c757d',
    };
    return colors[status] || '#6c757d';
  };

  const filteredMerchants = merchants;
  const filteredAds = advertisements;

  if (loading && !merchants.length) {
    return (
      <div className="p2p-merchant-page loading">
        <div className="loading-spinner">Loading...</div>
      </div>
    );
  }

  return (
    <div className="p2p-merchant-page">
      <div className="page-header">
        <h1>P2P Merchant Management</h1>
        <div className="header-actions">
          <button className="refresh-btn" onClick={loadData}>Refresh</button>
          <button className="export-btn">Export Data</button>
        </div>
      </div>

      {error && (
        <div className="error-banner">
          <span>{error}</span>
          <button onClick={() => setError(null)}>×</button>
        </div>
      )}

      <div className="stats-cards">
        <div className="stat-card">
          <div className="stat-icon">🏪</div>
          <div className="stat-info">
            <span className="stat-value">{stats?.totalMerchants || 0}</span>
            <span className="stat-label">Total Merchants</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">✅</div>
          <div className="stat-info">
            <span className="stat-value">{stats?.activeMerchants || 0}</span>
            <span className="stat-label">Active Merchants</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">🛡️</div>
          <div className="stat-info">
            <span className="stat-value">{stats?.verifiedMerchants || 0}</span>
            <span className="stat-label">Verified</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-info">
            <span className="stat-value">${((stats?.totalVolume || 0) / 1000000).toFixed(2)}M</span>
            <span className="stat-label">Total Volume</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">⭐</div>
          <div className="stat-info">
            <span className="stat-value">{(stats?.avgRating || 0).toFixed(2)}</span>
            <span className="stat-label">Avg Rating</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">📢</div>
          <div className="stat-info">
            <span className="stat-value">{stats?.activeAds || 0}</span>
            <span className="stat-label">Active Ads</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">🔒</div>
          <div className="stat-info">
            <span className="stat-value">${(stats?.totalCollateral || 0).toLocaleString()}</span>
            <span className="stat-label">Total Collateral</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon">⏳</div>
          <div className="stat-info">
            <span className="stat-value">{stats?.pendingCollateral || 0}</span>
            <span className="stat-label">Pending Collateral</span>
          </div>
        </div>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'merchants' ? 'active' : ''} 
          onClick={() => setActiveTab('merchants')}
        >
          🏪 Merchants ({merchants.length})
        </button>
        <button 
          className={activeTab === 'advertisements' ? 'active' : ''} 
          onClick={() => setActiveTab('advertisements')}
        >
          📢 Advertisements ({advertisements.length})
        </button>
        <button 
          className={activeTab === 'tiers' ? 'active' : ''} 
          onClick={() => setActiveTab('tiers')}
        >
          🏆 Tiers & Collateral
        </button>
        <button 
          className={activeTab === 'analytics' ? 'active' : ''} 
          onClick={() => setActiveTab('analytics')}
        >
          📊 Analytics
        </button>
        <button 
          className={activeTab === 'settings' ? 'active' : ''} 
          onClick={() => setActiveTab('settings')}
        >
          ⚙️ Settings
        </button>
      </div>

      {activeTab === 'merchants' && (
        <div className="merchants-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search merchants..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
            <select 
              value={filterStatus} 
              onChange={(e) => setFilterStatus(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Status</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="banned">Banned</option>
            </select>
            <select 
              value={filterVerification} 
              onChange={(e) => setFilterVerification(e.target.value)}
              className="filter-select"
            >
              <option value="all">All Verification</option>
              <option value="unverified">Unverified</option>
              <option value="email">Email</option>
              <option value="phone">Phone</option>
              <option value="kyc">KYC</option>
              <option value="advanced">Advanced</option>
            </select>
          </div>

          <div className="merchants-table">
            <table>
              <thead>
                <tr>
                  <th>Merchant</th>
                  <th>Tier</th>
                  <th>Collateral</th>
                  <th>Verification</th>
                  <th>Trades</th>
                  <th>Volume</th>
                  <th>Rating</th>
                  <th>Completion</th>
                  <th>Currency</th>
                  <th>Limits</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredMerchants.slice(0, 50).map(merchant => (
                  <tr key={merchant.id}>
                    <td>
                      <div className="merchant-info">
                        <span className="username">{merchant.username}</span>
                        <span className="user-id">{merchant.userId}</span>
                      </div>
                    </td>
                    <td>
                      <span 
                        className="tier-badge"
                        style={{ 
                          backgroundColor: getTierColor(merchant.tier),
                          color: merchant.tier === 'gold' || merchant.tier === 'silver' ? '#333' : '#fff'
                        }}
                      >
                        {merchant.tier.toUpperCase()}
                      </span>
                    </td>
                    <td>
                      <div className="collateral-info">
                        <span className={`collateral-status ${merchant.collateralStatus}`}>
                          {merchant.collateralStatus === 'none' ? '❌ None' : 
                           merchant.collateralStatus === 'pending' ? '⏳ Pending' :
                           merchant.collateralStatus === 'deposited' ? `✅ ${merchant.collateralAmount} ${merchant.collateralToken}` :
                           '↩️ Released'}
                        </span>
                      </div>
                    </td>
                    <td>
                      <span 
                        className="verification-badge"
                        style={{ backgroundColor: getVerificationColor(merchant.verificationLevel) }}
                      >
                        {merchant.verificationLevel}
                      </span>
                    </td>
                    <td>{merchant.totalTrades.toLocaleString()}</td>
                    <td>${(merchant.totalVolume / 1000).toFixed(1)}K</td>
                    <td>
                      <span className="rating">⭐ {merchant.rating.toFixed(2)}</span>
                    </td>
                    <td>{merchant.completionRate.toFixed(1)}%</td>
                    <td>{merchant.currency}</td>
                    <td>
                      <span className="limits">
                        ${merchant.limits.minOrder} - ${merchant.limits.maxOrder.toLocaleString()}
                      </span>
                    </td>
                    <td>
                      <span 
                        className="status-badge"
                        style={{ backgroundColor: getStatusColor(merchant.status) }}
                      >
                        {merchant.status}
                      </span>
                    </td>
                    <td>
                      <div className="actions">
                        <button 
                          className="action-btn view"
                          onClick={() => { setSelectedMerchant(merchant); setShowMerchantModal(true); }}
                        >
                          View
                        </button>
                        {merchant.status !== 'banned' && (
                          <button 
                            className="action-btn suspend"
                            onClick={() => handleSuspendMerchant(merchant.id)}
                          >
                            {merchant.status === 'suspended' ? 'Unfreeze' : 'Suspend'}
                          </button>
                        )}
                        {merchant.status !== 'banned' && (
                          <button 
                            className="action-btn ban"
                            onClick={() => handleBanMerchant(merchant.id)}
                          >
                            Ban
                          </button>
                        )}
                        {merchant.collateralStatus === 'pending' && (
                          <button 
                            className="action-btn approve"
                            onClick={() => handleApproveCollateral(merchant.id)}
                          >
                            Approve
                          </button>
                        )}
                        {merchant.collateralStatus === 'deposited' && (
                          <button 
                            className="action-btn release"
                            onClick={() => handleReleaseCollateral(merchant.id)}
                          >
                            Release
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'advertisements' && (
        <div className="advertisements-section">
          <div className="filters">
            <input
              type="text"
              placeholder="Search advertisements..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>

          <div className="ads-table">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Merchant</th>
                  <th>Type</th>
                  <th>Crypto</th>
                  <th>Fiat</th>
                  <th>Price</th>
                  <th>Premium</th>
                  <th>Limits</th>
                  <th>Payment</th>
                  <th>Orders</th>
                  <th>Completion</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filteredAds.map(ad => {
                  const merchant = merchants.find(m => m.id === ad.merchantId);
                  return (
                    <tr key={ad.id}>
                      <td className="ad-id">{ad.id}</td>
                      <td>{merchant?.username || 'Unknown'}</td>
                      <td>
                        <span className={`type-badge ${ad.type}`}>
                          {ad.type.toUpperCase()}
                        </span>
                      </td>
                      <td>{ad.crypto}</td>
                      <td>{ad.fiat}</td>
                      <td className="price">${ad.price.toLocaleString()}</td>
                      <td className={ad.premium >= 0 ? 'positive' : 'negative'}>
                        {ad.premium >= 0 ? '+' : ''}{ad.premium}%
                      </td>
                      <td>${ad.limits.min} - ${ad.limits.max.toLocaleString()}</td>
                      <td>{ad.paymentMethods.join(', ')}</td>
                      <td>{ad.ordersCount}</td>
                      <td>{ad.completionRate.toFixed(1)}%</td>
                      <td>
                        <span 
                          className="status-badge"
                          style={{ backgroundColor: getStatusColor(ad.status) }}
                        >
                          {ad.status}
                        </span>
                      </td>
                      <td>
                        <div className="actions">
                          <button 
                            className="action-btn toggle"
                            onClick={() => handleToggleAd(ad.id)}
                          >
                            {ad.status === 'active' ? 'Pause' : 'Activate'}
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'tiers' && (
        <div className="tiers-section">
          <div className="tier-info-banner">
            <h2>🏆 Merchant Tier & Collateral System</h2>
            <p>Merchants must deposit collateral in USDT or USDC to access higher tiers and trading limits.</p>
            <div className="collateral-requirements">
              <span className="requirement">💰 Minimum Collateral: ${COLLATERAL_REQUIREMENTS.minCollateral} USDT/USDC</span>
              <span className="requirement">💎 Maximum Collateral: ${COLLATERAL_REQUIREMENTS.maxCollateral} USDT/USDC</span>
            </div>
          </div>

          <div className="tiers-grid">
            {tiers.map(tier => {
              const tierMerchants = merchants.filter(m => m.tier === tier.id);
              return (
                <div key={tier.id} className="tier-card" style={{ borderTopColor: tier.color }}>
                  <div className="tier-header" style={{ backgroundColor: tier.color }}>
                    <h3>{tier.name}</h3>
                    <span className="tier-price">${tier.collateralAmount} USDT/USDC</span>
                  </div>
                  <div className="tier-body">
                    <div className="tier-stat">
                      <span className="label">Merchants</span>
                      <span className="value">{tierMerchants.length}</span>
                    </div>
                    <div className="tier-stat">
                      <span className="label">Max Order</span>
                      <span className="value">${tier.maxOrderLimit.toLocaleString()}</span>
                    </div>
                    <div className="tier-stat">
                      <span className="label">Daily Volume</span>
                      <span className="value">${tier.maxDailyVolume.toLocaleString()}</span>
                    </div>
                    <div className="tier-features">
                      <span className="label">Features:</span>
                      <ul>
                        {tier.features.map((feature, idx) => (
                          <li key={idx}>✓ {feature}</li>
                        ))}
                      </ul>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="pending-collateral-section">
            <h3>⏳ Pending Collateral Approvals</h3>
            <div className="pending-list">
              {pendingCollateral.map(deposit => (
                <div key={deposit.id} className="pending-item">
                  <div className="pending-info">
                    <span className="username">{deposit.merchantName}</span>
                    <span className="tier-badge" style={{ backgroundColor: getTierColor(deposit.tier) }}>
                      {deposit.tier.toUpperCase()}
                    </span>
                  </div>
                  <div className="pending-details">
                    <span className="amount">{deposit.amount} {deposit.token}</span>
                    <span className="tier-level">{getTierInfo(deposit.tier)?.name} Tier</span>
                  </div>
                  <div className="pending-actions">
                    <button 
                      className="action-btn approve"
                      onClick={() => handleApproveCollateral(deposit.id)}
                    >
                      ✓ Approve
                    </button>
                    <button className="action-btn reject">
                      ✗ Reject
                    </button>
                  </div>
                </div>
              ))}
              {pendingCollateral.length === 0 && (
                <p className="no-pending">No pending collateral deposits</p>
              )}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'analytics' && (
        <div className="analytics-section">
          <div className="analytics-grid">
            <div className="analytics-card">
              <h3>Top Merchants by Volume</h3>
              <div className="top-list">
                {topByVolume.map((m, idx) => (
                  <div key={m.id} className="top-item">
                    <span className="rank">#{idx + 1}</span>
                    <span className="name">{m.username}</span>
                    <span className="volume">${(m.totalVolume / 1000000).toFixed(2)}M</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="analytics-card">
              <h3>Top Merchants by Trades</h3>
              <div className="top-list">
                {topByTrades.map((m, idx) => (
                  <div key={m.id} className="top-item">
                    <span className="rank">#{idx + 1}</span>
                    <span className="name">{m.username}</span>
                    <span className="volume">{m.totalTrades.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="analytics-card">
              <h3>Verification Distribution</h3>
              <div className="distribution">
                {['unverified', 'email', 'phone', 'kyc', 'advanced'].map(level => {
                  const count = merchants.filter(m => m.verificationLevel === level).length;
                  const pct = (count / merchants.length * 100).toFixed(1);
                  return (
                    <div key={level} className="dist-item">
                      <span className="label">{level}</span>
                      <div className="bar-container">
                        <div 
                          className="bar" 
                          style={{ 
                            width: `${pct}%`,
                            backgroundColor: getVerificationColor(level)
                          }} 
                        />
                      </div>
                      <span className="value">{count} ({pct}%)</span>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="settings-section">
          <div className="settings-group">
            <h3>Merchant Requirements</h3>
            <div className="setting-item">
              <label>
                <input type="checkbox" defaultChecked />
                Require Email Verification
              </label>
            </div>
            <div className="setting-item">
              <label>
                <input type="checkbox" defaultChecked />
                Require Phone Verification
              </label>
            </div>
            <div className="setting-item">
              <label>
                <input type="checkbox" defaultChecked />
                Require KYC for Trading
              </label>
            </div>
          </div>

          <div className="settings-group">
            <h3>Trading Limits</h3>
            <div className="setting-item">
              <label>Minimum Order Amount (USD)</label>
              <input type="number" defaultValue={10} />
            </div>
            <div className="setting-item">
              <label>Maximum Order Amount (USD)</label>
              <input type="number" defaultValue={100000} />
            </div>
            <div className="setting-item">
              <label>Merchant Daily Limit (USD)</label>
              <input type="number" defaultValue={500000} />
            </div>
          </div>

          <div className="settings-group">
            <h3>Collateral Settings</h3>
            <div className="setting-item">
              <label>Minimum Collateral (USDT/USDC)</label>
              <input type="number" defaultValue={COLLATERAL_REQUIREMENTS.minCollateral} />
            </div>
            <div className="setting-item">
              <label>Maximum Collateral (USDT/USDC)</label>
              <input type="number" defaultValue={COLLATERAL_REQUIREMENTS.maxCollateral} />
            </div>
          </div>

          <div className="settings-group">
            <h3>Payment Methods</h3>
            <div className="payment-methods-grid">
              {['Bank Transfer', 'PayPal', 'Venmo', 'Cash Deposit', 'SEPA', 'AliPay', 'WeChat Pay', 'Crypto'].map(method => (
                <label key={method} className="payment-item">
                  <input type="checkbox" defaultChecked />
                  {method}
                </label>
              ))}
            </div>
          </div>

          <div className="settings-actions">
            <button className="save-btn">Save Settings</button>
          </div>
        </div>
      )}

      {showMerchantModal && selectedMerchant && (
        <div className="modal-overlay" onClick={() => setShowMerchantModal(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <h2>Merchant Details - {selectedMerchant.username}</h2>
            <div className="merchant-details">
              <div className="detail-row">
                <label>User ID:</label>
                <span>{selectedMerchant.userId}</span>
              </div>
              <div className="detail-row">
                <label>Email:</label>
                <span>{selectedMerchant.email}</span>
              </div>
              <div className="detail-row">
                <label>Phone:</label>
                <span>{selectedMerchant.phone}</span>
              </div>
              <div className="detail-row">
                <label>Verification:</label>
                <span 
                  className="verification-badge"
                  style={{ backgroundColor: getVerificationColor(selectedMerchant.verificationLevel) }}
                >
                  {selectedMerchant.verificationLevel}
                </span>
              </div>
              <div className="detail-row">
                <label>Status:</label>
                <span 
                  className="status-badge"
                  style={{ backgroundColor: getStatusColor(selectedMerchant.status) }}
                >
                  {selectedMerchant.status}
                </span>
              </div>
              <div className="detail-row">
                <label>Tier:</label>
                <span 
                  className="tier-badge"
                  style={{ backgroundColor: getTierColor(selectedMerchant.tier) }}
                >
                  {selectedMerchant.tier.toUpperCase()}
                </span>
              </div>
              <div className="detail-row">
                <label>Collateral:</label>
                <span>{selectedMerchant.collateralAmount} {selectedMerchant.collateralToken}</span>
              </div>
              <div className="detail-row">
                <label>Total Trades:</label>
                <span>{selectedMerchant.totalTrades.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Total Volume:</label>
                <span>${selectedMerchant.totalVolume.toLocaleString()}</span>
              </div>
              <div className="detail-row">
                <label>Rating:</label>
                <span>⭐ {selectedMerchant.rating.toFixed(2)}</span>
              </div>
              <div className="detail-row">
                <label>Completion Rate:</label>
                <span>{selectedMerchant.completionRate.toFixed(1)}%</span>
              </div>
              <div className="detail-row">
                <label>Member Since:</label>
                <span>{selectedMerchant.createdAt}</span>
              </div>
            </div>
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowMerchantModal(false)}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default P2PMerchantPage;
