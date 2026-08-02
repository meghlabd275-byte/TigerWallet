import React, { useState } from 'react';
import Header from '../components/Header';
import Sidebar from '../components/Sidebar';

// Staking Page - Complete
const StakingPage = () => {
  const [selectedTab, setSelectedTab] = useState('stake');
  const [stakeAmount, setStakeAmount] = useState('');
  const [selectedPool, setSelectedPool] = useState('ETH 2.0');
  const [isStaking, setIsStaking] = useState(false);

  const pools = [
    { name: 'ETH 2.0', apy: '4.2%', staked: '1.5 ETH', reward: '0.063 ETH' },
    { name: 'BNB', apy: '3.8%', staked: '0 BNB', reward: '0 BNB' },
    { name: 'SOL', apy: '6.5%', staked: '0 SOL', reward: '0 SOL' },
    { name: 'MATIC', apy: '5.2%', staked: '0 MATIC', reward: '0 MATIC' },
  ];

  const handleStake = () => {
    if (!stakeAmount || parseFloat(stakeAmount) <= 0) {
      alert('Please enter valid amount');
      return;
    }
    setIsStaking(true);
    setTimeout(() => {
      setIsStaking(false);
      setStakeAmount('');
      alert('Staked successfully!');
    }, 2000);
  };

  const handleClaim = (poolName: string) => {
    alert(`Claiming rewards from ${poolName}`);
  };

  return (
    <div className="app-container">
      <Sidebar />
      <div className="main-content">
        <Header title="Staking" />
        
        <div className="page-content">
          {/* Tab Selector */}
          <div className="tabs">
            <button
              className={`tab ${selectedTab === 'stake' ? 'active' : ''}`}
              onClick={() => setSelectedTab('stake')}
            >
              Stake
            </button>
            <button
              className={`tab ${selectedTab === 'earn' ? 'active' : ''}`}
              onClick={() => setSelectedTab('earn')}
            >
              Earn
            </button>
            <button
              className={`tab ${selectedTab === 'pools' ? 'active' : ''}`}
              onClick={() => setSelectedTab('pools')}
            >
              Pools
            </button>
          </div>

          {/* Stake Tab */}
          {selectedTab === 'stake' && (
            <div className="stake-content">
              {/* Pool Selector */}
              <div className="card">
                <h3>Select Pool</h3>
                <div className="pool-grid">
                  {pools.map((pool) => (
                    <div
                      key={pool.name}
                      className={`pool-card ${selectedPool === pool.name ? 'active' : ''}`}
                      onClick={() => setSelectedPool(pool.name)}
                    >
                      <div className="pool-name">{pool.name}</div>
                      <div className="pool-apy">APY: {pool.apy}</div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Amount Input */}
              <div className="card">
                <div className="amount-header">
                  <h3>Amount</h3>
                  <button className="max-btn" onClick={() => setStakeAmount('1.0')}>MAX</button>
                </div>
                <div className="amount-input">
                  <input
                    type="number"
                    placeholder="0.0"
                    value={stakeAmount}
                    onChange={(e) => setStakeAmount(e.target.value)}
                  />
                  <span className="token">{selectedPool.split(' ')[0]}</span>
                </div>
              </div>

              {/* Stake Button */}
              <button
                className={`stake-btn ${isStaking ? 'loading' : ''}`}
                onClick={handleStake}
                disabled={isStaking}
              >
                {isStaking ? 'Staking...' : 'Stake'}
              </button>
            </div>
          )}

          {/* Earn Tab */}
          {selectedTab === 'earn' && (
            <div className="earn-content">
              {pools.map((pool) => (
                <div key={pool.name} className="earn-card">
                  <div className="earn-header">
                    <div>
                      <div className="pool-name">{pool.name}</div>
                      <div className="pool-apy">APY: {pool.apy}</div>
                    </div>
                    <div className="earn-right">
                      <div className="staked-label">Staked</div>
                      <div className="staked-value">{pool.staked}</div>
                    </div>
                  </div>
                  <div className="earn-footer">
                    <div>
                      <div className="reward-label">Pending Reward</div>
                      <div className="reward-value">{pool.reward}</div>
                    </div>
                    <button className="claim-btn" onClick={() => handleClaim(pool.name)}>
                      Claim
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pools Tab */}
          {selectedTab === 'pools' && (
            <div className="pools-content">
              {pools.map((pool) => (
                <div key={pool.name} className="pool-detail-card">
                  <div className="pool-detail-header">
                    <div className="pool-name">{pool.name}</div>
                    <div className="pool-apy-large">
                      <span className="apy-value">{pool.apy}</span>
                      <span className="apy-label">APY</span>
                    </div>
                  </div>
                  <div className="total-staked">Total Staked: {pool.staked}</div>
                  <button className="stake-pool-btn">Stake</button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <style>{`
        .page-content {
          padding: 24px;
        }
        .tabs {
          display: flex;
          gap: 8px;
          margin-bottom: 24px;
        }
        .tab {
          padding: 12px 24px;
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
        .card {
          background: white;
          border-radius: 12px;
          padding: 20px;
          margin-bottom: 16px;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .card h3 {
          margin-bottom: 16px;
          font-size: 16px;
          color: #64748b;
        }
        .pool-grid {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 12px;
        }
        .pool-card {
          padding: 16px;
          background: #f8fafc;
          border-radius: 8px;
          text-align: center;
          cursor: pointer;
          border: 2px solid transparent;
          transition: all 0.2s;
        }
        .pool-card.active {
          border-color: #f97316;
          background: #fff7ed;
        }
        .pool-name {
          font-weight: 600;
          margin-bottom: 4px;
        }
        .pool-apy {
          color: #22c55e;
          font-size: 14px;
        }
        .amount-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;
        }
        .max-btn {
          padding: 4px 12px;
          background: #f97316;
          color: white;
          border: none;
          border-radius: 4px;
          cursor: pointer;
        }
        .amount-input {
          display: flex;
          align-items: center;
          gap: 12px;
        }
        .amount-input input {
          flex: 1;
          padding: 12px;
          font-size: 24px;
          border: 1px solid #e2e8f0;
          border-radius: 8px;
        }
        .token {
          font-size: 18px;
          color: #64748b;
        }
        .stake-btn {
          width: 100%;
          padding: 16px;
          background: #f97316;
          color: white;
          border: none;
          border-radius: 12px;
          font-size: 18px;
          font-weight: 600;
          cursor: pointer;
        }
        .stake-btn.loading {
          opacity: 0.6;
        }
        .earn-card, .pool-detail-card {
          background: white;
          border-radius: 12px;
          padding: 20px;
          margin-bottom: 12px;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .earn-header {
          display: flex;
          justify-content: space-between;
          margin-bottom: 16px;
        }
        .earn-right {
          text-align: right;
        }
        .staked-label, .reward-label {
          font-size: 12px;
          color: #64748b;
        }
        .staked-value, .reward-value {
          font-weight: 600;
        }
        .reward-value {
          color: #22c55e;
        }
        .earn-footer {
          display: flex;
          justify-content: space-between;
          align-items: center;
          border-top: 1px solid #e2e8f0;
          padding-top: 16px;
        }
        .claim-btn {
          padding: 8px 20px;
          background: #f97316;
          color: white;
          border: none;
          border-radius: 8px;
          cursor: pointer;
        }
        .pool-detail-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 12px;
        }
        .pool-apy-large {
          text-align: right;
        }
        .apy-value {
          font-size: 28px;
          font-weight: bold;
          color: #22c55e;
        }
        .apy-label {
          font-size: 12px;
          color: #64748b;
          display: block;
        }
        .total-staked {
          font-size: 14px;
          color: #64748b;
          margin-bottom: 16px;
        }
        .stake-pool-btn {
          width: 100%;
          padding: 12px;
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

export default StakingPage;
