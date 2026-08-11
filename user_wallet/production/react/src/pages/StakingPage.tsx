import React, { useState, useEffect, useCallback } from 'react';
import Header from '../components/Header';
import Sidebar from '../components/Sidebar';
import { useWallet } from '../contexts/WalletContext';
import { WalletService } from '../services/WalletService';

interface StakingAsset {
  symbol: string;
  apy?: number;
  staked?: string;
  reward?: string;
}

// Staking Page - Complete
const StakingPage = () => {
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());
  const [selectedTab, setSelectedTab] = useState('stake');
  const [stakeAmount, setStakeAmount] = useState('');
  const [selectedPool, setSelectedPool] = useState('');
  const [isStaking, setIsStaking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [password, setPassword] = useState('');
  const [assets, setAssets] = useState<StakingAsset[]>([]);
  const [isLoadingAssets, setIsLoadingAssets] = useState(false);

  const chainId = activeWallet?.chain?.chainId ?? 1;

  const loadAssets = useCallback(async () => {
    setIsLoadingAssets(true);
    setError(null);
    try {
      const data = (await walletService.getStakingPositions(activeWallet?.id ?? '', chainId)) as StakingAsset[];
      const mapped = (data && data.length ? data : []).map((a) => ({
        symbol: String(a.symbol ?? ''),
        apy: typeof a.apy === 'number' ? a.apy : undefined,
        staked: a.staked ? String(a.staked) : '0',
        reward: a.reward ? String(a.reward) : '0',
      }));
      setAssets(mapped);
      if (mapped.length > 0 && !selectedPool) setSelectedPool(mapped[0].symbol);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load staking assets');
      setAssets([]);
    } finally {
      setIsLoadingAssets(false);
    }
  }, [walletService, activeWallet?.id, chainId, selectedPool]);

  useEffect(() => {
    loadAssets();
  }, [loadAssets]);

  const handleStake = async () => {
    if (!stakeAmount || parseFloat(stakeAmount) <= 0) {
      setError('Please enter a valid amount');
      return;
    }
    if (!activeWallet) { setError('No active wallet'); return; }
    if (!password) { setError('Wallet password is required to stake'); return; }
    if (!selectedPool) { setError('Select a pool first'); return; }
    setIsStaking(true);
    setError(null);
    try {
      const result = await walletService.stake(
        activeWallet.id,
        selectedPool,
        stakeAmount,
        undefined,
        password,
        chainId,
      );
      if (!result.txHash) {
        setError('Stake submitted but no transaction hash was returned by the backend');
      } else {
        setStakeAmount('');
        setPassword('');
        await loadAssets();
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Stake failed');
    } finally {
      setIsStaking(false);
    }
  };

  const handleClaim = async (poolName: string) => {
    if (!activeWallet) { setError('No active wallet'); return; }
    if (!password) { setError('Wallet password is required to claim'); return; }
    setError(null);
    try {
      await walletService.claimRewards(activeWallet.id, poolName, password, chainId);
      await loadAssets();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Claim failed');
    }
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
                  {isLoadingAssets ? (
                    <div className="pool-name">Loading pools…</div>
                  ) : assets.length === 0 ? (
                    <div className="pool-name">No staking assets available</div>
                  ) : (
                    assets.map((pool) => (
                      <div
                        key={pool.symbol}
                        className={`pool-card ${selectedPool === pool.symbol ? 'active' : ''}`}
                        onClick={() => setSelectedPool(pool.symbol)}
                      >
                        <div className="pool-name">{pool.symbol}</div>
                        <div className="pool-apy">APY: {pool.apy !== undefined ? `${pool.apy}%` : '—'}</div>
                      </div>
                    ))
                  )}
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
                  <span className="token">{selectedPool}</span>
                </div>
              </div>

              {/* Wallet password (required by backend /staking/stake) */}
              <div className="card">
                <h3>Wallet Password</h3>
                <div className="amount-input">
                  <input
                    type="password"
                    placeholder="Wallet password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </div>
              </div>

              {error && (
                <div className="card" style={{ color: '#dc2626' }}>{error}</div>
              )}

              {/* Stake Button */}
              <button
                className={`stake-btn ${isStaking ? 'loading' : ''}`}
                onClick={handleStake}
                disabled={isStaking || !selectedPool || !password}
              >
                {isStaking ? 'Staking...' : 'Stake'}
              </button>
            </div>
          )}

          {/* Earn Tab */}
          {selectedTab === 'earn' && (
            <div className="earn-content">
              {isLoadingAssets ? (
                <div className="earn-card"><div className="pool-name">Loading…</div></div>
              ) : assets.length === 0 ? (
                <div className="earn-card"><div className="pool-name">No staking positions</div></div>
              ) : (
                assets.map((pool) => (
                  <div key={pool.symbol} className="earn-card">
                    <div className="earn-header">
                      <div>
                        <div className="pool-name">{pool.symbol}</div>
                        <div className="pool-apy">APY: {pool.apy !== undefined ? `${pool.apy}%` : '—'}</div>
                      </div>
                      <div className="earn-right">
                        <div className="staked-label">Staked</div>
                        <div className="staked-value">{pool.staked ?? '0'}</div>
                      </div>
                    </div>
                    <div className="earn-footer">
                      <div>
                        <div className="reward-label">Pending Reward</div>
                        <div className="reward-value">{pool.reward ?? '0'}</div>
                      </div>
                      <button className="claim-btn" onClick={() => handleClaim(pool.symbol)}>
                        Claim
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {/* Pools Tab */}
          {selectedTab === 'pools' && (
            <div className="pools-content">
              {isLoadingAssets ? (
                <div className="pool-detail-card"><div className="pool-name">Loading…</div></div>
              ) : assets.length === 0 ? (
                <div className="pool-detail-card"><div className="pool-name">No pools available</div></div>
              ) : (
                assets.map((pool) => (
                  <div key={pool.symbol} className="pool-detail-card">
                    <div className="pool-detail-header">
                      <div className="pool-name">{pool.symbol}</div>
                      <div className="pool-apy-large">
                        <span className="apy-value">{pool.apy !== undefined ? `${pool.apy}%` : '—'}</span>
                        <span className="apy-label">APY</span>
                      </div>
                    </div>
                    <div className="total-staked">Total Staked: {pool.staked ?? '0'}</div>
                    <button className="stake-pool-btn" onClick={() => { setSelectedTab('stake'); setSelectedPool(pool.symbol); }}>Stake</button>
                  </div>
                ))
              )}
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
