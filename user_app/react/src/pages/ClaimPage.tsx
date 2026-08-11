// Claim Page - Claim Airdrops, Rewards, and Bonuses
// Connected to backend claim service

import React, { useState, useEffect, useCallback } from 'react';
import './ClaimPage.css';

interface ClaimableReward {
  id: string;
  type: 'airdrop' | 'reward' | 'bonus' | 'rebate' | 'cashback';
  title: string;
  description: string;
  amount: number;
  token: string;
  expiresAt: number;
  status: 'claimable' | 'pending' | 'claimed' | 'expired';
  source: string;
  icon: string;
}

interface ClaimHistory {
  id: string;
  type: string;
  title: string;
  amount: number;
  token: string;
  claimedAt: number;
  txHash: string;
}

// Reward icons mapping
const REWARD_ICONS: {[key: string]: string} = {
  airdrop: '🎁', reward: '🏆', bonus: '💰', rebate: '🔄', cashback: '💵'
};

const ClaimPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'available' | 'history'>('available');
  const [claimableRewards, setClaimableRewards] = useState<ClaimableReward[]>([]);
  const [claimHistory, setClaimHistory] = useState<ClaimHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [claiming, setClaiming] = useState<string | null>(null);

  // Load claimable rewards from backend
  const loadClaimableRewards = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('user_token');
      const response = await fetch('http://localhost:8443/api/v1/claims/available', {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      
      if (!response.ok) {
        throw new Error('Failed to load rewards');
      }
      
      const data = await response.json();
      
      if (data.rewards && Array.isArray(data.rewards)) {
        const rewards = data.rewards.map((r: any) => ({
          ...r,
          icon: REWARD_ICONS[r.type] || '🎁'
        }));
        setClaimableRewards(rewards);
      } else {
        setClaimableRewards([]);
      }
    } catch (err) {
      console.error('Failed to load claimable rewards:', err);
      setError('Unable to load rewards. Please ensure the backend service is running.');
      setClaimableRewards([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Load claim history from backend
  const loadClaimHistory = useCallback(async () => {
    try {
      const token = localStorage.getItem('user_token');
      const response = await fetch('http://localhost:8443/api/v1/claims/history', {
        headers: token ? { Authorization: `Bearer ${token}` } : {}
      });
      
      if (!response.ok) {
        throw new Error('Failed to load history');
      }
      
      const data = await response.json();
      setClaimHistory(data.history || []);
    } catch (err) {
      console.error('Failed to load claim history:', err);
    }
  }, []);

  useEffect(() => {
    if (activeTab === 'available') {
      loadClaimableRewards();
    } else {
      loadClaimHistory();
    }
  }, [activeTab, loadClaimableRewards, loadClaimHistory]);

  // Handle claiming a reward
  const handleClaim = async (rewardId: string) => {
    setClaiming(rewardId);
    setError(null);
    
    try {
      const token = localStorage.getItem('user_token');
      const response = await fetch(`http://localhost:8443/api/v1/claims/${rewardId}/claim`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        }
      });
      
      const data = await response.json();
      
      if (data.success) {
        // Reload available rewards
        loadClaimableRewards();
        // Add to history
        loadClaimHistory();
        setActiveTab('history');
      } else {
        setError(data.error || 'Claim failed');
      }
    } catch (err) {
      console.error('Claim failed:', err);
      setError('Failed to claim reward. Please try again.');
    } finally {
      setClaiming(null);
    }
  };

  const [claimSuccess, setClaimSuccess] = useState<{show: boolean; reward?: ClaimableReward}>({show: false});

  const availableRewards = claimableRewards.filter(r => r.status === 'claimable');
  const pendingRewards = claimableRewards.filter(r => r.status === 'pending');
  const totalClaimable = availableRewards.reduce((sum, r) => sum + r.amount, 0);

  const handleClaimClick = async (reward: ClaimableReward) => {
    await handleClaim(reward.id);
    setClaimSuccess({show: true, reward});
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'airdrop': return '#9c27b0';
      case 'bonus': return '#ff9800';
      case 'reward': return '#4caf50';
      case 'rebate': return '#2196f3';
      case 'cashback': return '#00bcd4';
      default: return '#666';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'claimable': return 'Claim Now';
      case 'pending': return 'Processing';
      case 'claimed': return 'Claimed';
      case 'expired': return 'Expired';
      default: return status;
    }
  };

  return (
    <div className="claim-page">
      <div className="claim-header">
        <h1>🎁 Claim Rewards</h1>
        <p>Claim your airdrops, rewards, and bonuses</p>
      </div>

      {totalClaimable > 0 && (
        <div className="total-claimable">
          <div className="total-info">
            <span className="label">Total Claimable</span>
            <span className="amount">{totalClaimable.toFixed(2)} USDT</span>
          </div>
          <button className="claim-all-btn">
            Claim All
          </button>
        </div>
      )}

      <div className="tabs">
        <button 
          className={activeTab === 'available' ? 'active' : ''} 
          onClick={() => setActiveTab('available')}
        >
          Available ({availableRewards.length})
        </button>
        <button 
          className={activeTab === 'history' ? 'active' : ''} 
          onClick={() => setActiveTab('history')}
        >
          History ({claimHistory.length})
        </button>
      </div>

      {activeTab === 'available' && (
        <div className="rewards-section">
          {claimableRewards.filter(r => r.status !== 'claimed').length === 0 ? (
            <div className="empty-state">
              <span className="empty-icon">🎁</span>
              <h3>No Rewards Available</h3>
              <p>Check back later for new rewards and airdrops</p>
            </div>
          ) : (
            <>
              {availableRewards.length > 0 && (
                <div className="rewards-group">
                  <h3>Ready to Claim</h3>
                  <div className="rewards-grid">
                    {availableRewards.map(reward => (
                      <div key={reward.id} className="reward-card claimable">
                        <div className="reward-header">
                          <span className="reward-icon">{reward.icon}</span>
                          <span 
                            className="reward-type" 
                            style={{ background: getTypeColor(reward.type) }}
                          >
                            {reward.type}
                          </span>
                        </div>
                        <h4>{reward.title}</h4>
                        <p className="reward-desc">{reward.description}</p>
                        <div className="reward-amount">
                          <span className="amount">{reward.amount}</span>
                          <span className="token">{reward.token}</span>
                        </div>
                        <div className="reward-footer">
                          <span className="expires">
                            Expires: {new Date(reward.expiresAt).toLocaleDateString()}
                          </span>
                          <button 
                            className="claim-btn"
                            onClick={() => handleClaim(reward)}
                            disabled={claimingId !== null}
                          >
                            {claimingId === reward.id ? 'Claiming...' : 'Claim'}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {pendingRewards.length > 0 && (
                <div className="rewards-group">
                  <h3>Processing</h3>
                  <div className="rewards-grid">
                    {pendingRewards.map(reward => (
                      <div key={reward.id} className="reward-card pending">
                        <div className="reward-header">
                          <span className="reward-icon">{reward.icon}</span>
                          <span 
                            className="reward-type" 
                            style={{ background: getTypeColor(reward.type) }}
                          >
                            {reward.type}
                          </span>
                        </div>
                        <h4>{reward.title}</h4>
                        <p className="reward-desc">{reward.description}</p>
                        <div className="reward-amount">
                          <span className="amount">{reward.amount}</span>
                          <span className="token">{reward.token}</span>
                        </div>
                        <div className="reward-footer">
                          <span className="status pending">Processing...</span>
                          <button className="pending-btn" disabled>
                            Pending
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {claimableRewards.filter(r => r.status === 'expired').length > 0 && (
                <div className="rewards-group">
                  <h3>Expired</h3>
                  <div className="rewards-grid">
                    {claimableRewards.filter(r => r.status === 'expired').map(reward => (
                      <div key={reward.id} className="reward-card expired">
                        <div className="reward-header">
                          <span className="reward-icon">{reward.icon}</span>
                          <span 
                            className="reward-type" 
                            style={{ background: '#999' }}
                          >
                            {reward.type}
                          </span>
                        </div>
                        <h4>{reward.title}</h4>
                        <p className="reward-desc">{reward.description}</p>
                        <div className="reward-amount">
                          <span className="amount">{reward.amount}</span>
                          <span className="token">{reward.token}</span>
                        </div>
                        <div className="reward-footer">
                          <span className="status expired">Expired</span>
                          <button className="expired-btn" disabled>
                            Expired
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      )}

      {activeTab === 'history' && (
        <div className="history-section">
          {claimHistory.length === 0 ? (
            <div className="empty-state">
              <span className="empty-icon">📋</span>
              <h3>No Claim History</h3>
              <p>Your claimed rewards will appear here</p>
            </div>
          ) : (
            <div className="history-list">
              {claimHistory.map(record => (
                <div key={record.id} className="history-item">
                  <div className="history-info">
                    <span className="history-icon">
                      {record.type === 'bonus' ? '💰' : record.type === 'reward' ? '🏆' : '🎁'}
                    </span>
                    <div>
                      <h4>{record.title}</h4>
                      <span className="history-date">
                        {new Date(record.claimedAt).toLocaleString()}
                      </span>
                    </div>
                  </div>
                  <div className="history-amount">
                    <span className="amount">+{record.amount}</span>
                    <span className="token">{record.token}</span>
                  </div>
                  <div className="history-tx">
                    <span className="tx-hash">{record.txHash}</span>
                    <span className="view-explorer">View</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {claimSuccess.show && (
        <div className="success-modal" onClick={() => setClaimSuccess({show: false})}>
          <div className="success-content" onClick={e => e.stopPropagation()}>
            <div className="success-icon">🎉</div>
            <h2>Claim Successful!</h2>
            <p>You have received</p>
            <div className="claimed-amount">
              {claimSuccess.reward?.amount} {claimSuccess.reward?.token}
            </div>
            <p className="tx-hash">
              Transaction: 0x{Math.random().toString(16).slice(2, 10)}...{Math.random().toString(16).slice(2, 6)}
            </p>
            <button className="done-btn" onClick={() => setClaimSuccess({show: false})}>
              Done
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default ClaimPage;
