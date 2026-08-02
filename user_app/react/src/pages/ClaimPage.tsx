// Claim Page - Claim Airdrops, Rewards, and Bonuses

import React, { useState } from 'react';
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

const ClaimPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'available' | 'history'>('available');
  const [claimableRewards, setClaimableRewards] = useState<ClaimableReward[]>([
    {
      id: '1',
      type: 'airdrop',
      title: 'TigerWallet Launch Airdrop',
      description: 'Welcome bonus for early users',
      amount: 100,
      token: 'TIGER',
      expiresAt: Date.now() + 7 * 24 * 60 * 60 * 1000,
      status: 'claimable',
      source: 'TigerWallet',
      icon: '🎁'
    },
    {
      id: '2',
      type: 'bonus',
      title: 'First Deposit Bonus',
      description: 'Get 20% bonus on your first deposit',
      amount: 50,
      token: 'USDT',
      expiresAt: Date.now() + 30 * 24 * 60 * 60 * 1000,
      status: 'claimable',
      source: 'Promotion',
      icon: '💰'
    },
    {
      id: '3',
      type: 'reward',
      title: 'Trading Competition Reward',
      description: 'You placed 3rd in weekly trading competition',
      amount: 250,
      token: 'USDT',
      expiresAt: Date.now() + 14 * 24 * 60 * 60 * 1000,
      status: 'claimable',
      source: 'Competition',
      icon: '🏆'
    },
    {
      id: '4',
      type: 'rebate',
      title: 'Trading Fee Rebate',
      description: 'Your weekly trading fee rebate',
      amount: 15.5,
      token: 'USDT',
      expiresAt: Date.now() + 3 * 24 * 60 * 60 * 1000,
      status: 'claimable',
      source: 'Fee Rebate',
      icon: '🔄'
    },
    {
      id: '5',
      type: 'cashback',
      title: 'Cashback Reward',
      description: 'Your monthly cashback from swaps',
      amount: 8.25,
      token: 'USDT',
      expiresAt: Date.now() + 60 * 24 * 60 * 60 * 1000,
      status: 'pending',
      source: 'Swap Cashback',
      icon: '💵'
    },
    {
      id: '6',
      type: 'airdrop',
      title: 'Partner Airdrop - ChainX',
      description: 'Exclusive airdrop from ChainX partnership',
      amount: 500,
      token: 'CX',
      expiresAt: Date.now() - 2 * 24 * 60 * 60 * 1000,
      status: 'expired',
      source: 'ChainX',
      icon: '🪂'
    }
  ]);

  const [claimHistory, setClaimHistory] = useState<ClaimHistory[]>([
    {
      id: '1',
      type: 'bonus',
      title: 'Sign-up Bonus',
      amount: 10,
      token: 'USDT',
      claimedAt: Date.now() - 10 * 24 * 60 * 60 * 1000,
      txHash: '0x1234...abcd'
    },
    {
      id: '2',
      type: 'reward',
      title: 'Referral Bonus',
      amount: 25,
      token: 'USDT',
      claimedAt: Date.now() - 5 * 24 * 60 * 60 * 1000,
      txHash: '0x5678...efgh'
    }
  ]);

  const [claimingId, setClaimingId] = useState<string | null>(null);
  const [claimSuccess, setClaimSuccess] = useState<{show: boolean; reward?: ClaimableReward}>({show: false});

  const availableRewards = claimableRewards.filter(r => r.status === 'claimable');
  const pendingRewards = claimableRewards.filter(r => r.status === 'pending');
  const totalClaimable = availableRewards.reduce((sum, r) => sum + r.amount, 0);

  const handleClaim = async (reward: ClaimableReward) => {
    setClaimingId(reward.id);
    
    // Simulate claiming process
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    // Update reward status
    setClaimableRewards(claimableRewards.map(r => 
      r.id === reward.id ? { ...r, status: 'claimed' as const } : r
    ));

    // Add to history
    setClaimHistory([{
      id: Date.now().toString(),
      type: reward.type,
      title: reward.title,
      amount: reward.amount,
      token: reward.token,
      claimedAt: Date.now(),
      txHash: `0x${Math.random().toString(16).slice(2, 10)}...${Math.random().toString(16).slice(2, 6)}`
    }, ...claimHistory]);

    setClaimingId(null);
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
