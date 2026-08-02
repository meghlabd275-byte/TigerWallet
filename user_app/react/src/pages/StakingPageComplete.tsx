/**
 * TigerWallet - Staking Page
 * Complete staking functionality for web app
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../stores/ThemeStore';

interface Validator {
  id: string;
  name: string;
  apr: number;
  tvl: string;
  risk: 'low' | 'medium' | 'high';
  minStake: string;
}

interface StakingPosition {
  id: string;
  validator: string;
  amount: string;
  reward: string;
  unlockTime: number;
  status: 'active' | 'unlocking' | 'unlocked';
}

const StakingPage: React.FC = () => {
  const { theme } = useTheme();
  
  const [selectedChain, setSelectedChain] = useState('ethereum');
  const [validators, setValidators] = useState<Validator[]>([]);
  const [positions, setPositions] = useState<StakingPosition[]>([]);
  const [selectedValidator, setSelectedValidator] = useState<Validator | null>(null);
  const [stakeAmount, setStakeAmount] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const chains = [
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC' },
    { id: 'solana', name: 'Solana', symbol: 'SOL' },
    { id: 'cosmos', name: 'Cosmos', symbol: 'ATOM' },
  ];

  const loadValidators = useCallback(async () => {
    setIsLoading(true);
    try {
      // Mock validators data
      const mockValidators: Validator[] = {
        ethereum: [
          { id: 'lido', name: 'Lido', apr: 4.2, tvl: '$15B', risk: 'low', minStake: '0.01' },
          { id: 'rocketpool', name: 'Rocket Pool', apr: 3.8, tvl: '$1.5B', risk: 'low', minStake: '0.01' },
          { id: 'coinbase', name: 'Coinbase Staking', apr: 3.5, tvl: '$8B', risk: 'low', minStake: '0.01' },
        ],
        polygon: [
          { id: 'ankr', name: 'Ankr', apr: 5.2, tvl: '$500M', risk: 'medium', minStake: '10' },
          { id: 'stader', name: 'Stader', apr: 4.8, tvl: '$300M', risk: 'medium', minStake: '10' },
        ],
        solana: [
          { id: 'marinade', name: 'Marinade Finance', apr: 6.5, tvl: '$400M', risk: 'low', minStake: '1' },
          { id: 'jpool', name: 'JPool', apr: 6.2, tvl: '$200M', risk: 'medium', minStake: '1' },
        ],
        cosmos: [
          { id: 'cosmoshub', name: 'Cosmos Hub', apr: 12, tvl: '$1B', risk: 'low', minStake: '1' },
          { id: 'osmosis', name: 'Osmosis', apr: 15, tvl: '$500M', risk: 'medium', minStake: '1' },
        ],
      };
      setValidators(mockValidators[selectedChain as keyof typeof mockValidators] || []);
    } finally {
      setIsLoading(false);
    }
  }, [selectedChain]);

  useEffect(() => {
    loadValidators();
  }, [loadValidators]);

  const handleStake = useCallback(async () => {
    if (!selectedValidator || !stakeAmount) return;
    
    setIsLoading(true);
    try {
      // Simulate staking
      alert(`Staking ${stakeAmount} to ${selectedValidator.name}`);
      setStakeAmount('');
    } finally {
      setIsLoading(false);
    }
  }, [selectedValidator, stakeAmount]);

  const handleUnstake = useCallback(async (positionId: string) => {
    if (!confirm('Are you sure you want to unstake?')) return;
    
    setIsLoading(true);
    try {
      // Simulate unstaking
      alert(`Unstaking position ${positionId}`);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const handleClaim = useCallback(async (positionId: string) => {
    setIsLoading(true);
    try {
      // Simulate claiming
      alert(`Claiming rewards for position ${positionId}`);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'low': return '#22c55e';
      case 'medium': return '#f59e0b';
      case 'high': return '#ef4444';
      default: return '#888';
    }
  };

  const selectedChainData = chains.find(c => c.id === selectedChain);

  return (
    <div className={`staking-page ${theme}`}>
      <div className="page-header">
        <h1>📈 Staking</h1>
        <p>Earn rewards by staking your tokens</p>
      </div>

      {/* Chain Selector */}
      <div className="chain-selector">
        {chains.map(chain => (
          <button
            key={chain.id}
            className={`chain-tab ${selectedChain === chain.id ? 'active' : ''}`}
            onClick={() => setSelectedChain(chain.id)}
          >
            <span className="chain-symbol">{chain.symbol}</span>
            <span className="chain-name">{chain.name}</span>
          </button>
        ))}
      </div>

      {/* Current Positions */}
      {positions.length > 0 && (
        <div className="section">
          <h2>Your Staking Positions</h2>
          <div className="positions-list">
            {positions.map(pos => (
              <div key={pos.id} className="position-card">
                <div className="position-info">
                  <div className="validator-name">{pos.validator}</div>
                  <div className="position-amount">
                    <span className="label">Staked:</span>
                    <span className="value">{pos.amount} {selectedChainData?.symbol}</span>
                  </div>
                  <div className="position-reward">
                    <span className="label">Reward:</span>
                    <span className="value">{pos.reward} {selectedChainData?.symbol}</span>
                  </div>
                  <div className={`position-status ${pos.status}`}>
                    {pos.status}
                  </div>
                </div>
                <div className="position-actions">
                  <button 
                    className="btn btn-primary"
                    onClick={() => handleClaim(pos.id)}
                    disabled={isLoading}
                  >
                    Claim
                  </button>
                  <button 
                    className="btn btn-secondary"
                    onClick={() => handleUnstake(pos.id)}
                    disabled={isLoading}
                  >
                    Unstake
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Validators */}
      <div className="section">
        <h2>Select Validator</h2>
        
        {isLoading ? (
          <div className="loading">Loading validators...</div>
        ) : (
          <div className="validators-list">
            {validators.map(validator => (
              <div 
                key={validator.id} 
                className={`validator-card ${selectedValidator?.id === validator.id ? 'selected' : ''}`}
                onClick={() => setSelectedValidator(validator)}
              >
                <div className="validator-header">
                  <div className="validator-name">{validator.name}</div>
                  <div className="validator-risk" style={{ color: getRiskColor(validator.risk) }}>
                    {validator.risk} risk
                  </div>
                </div>
                <div className="validator-stats">
                  <div className="stat">
                    <span className="label">APR</span>
                    <span className="value">{validator.apr}%</span>
                  </div>
                  <div className="stat">
                    <span className="label">TVL</span>
                    <span className="value">{validator.tvl}</span>
                  </div>
                  <div className="stat">
                    <span className="label">Min</span>
                    <span className="value">{validator.minStake} {selectedChainData?.symbol}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Stake Form */}
      {selectedValidator && (
        <div className="section">
          <h2>Stake {selectedChainData?.symbol}</h2>
          <div className="stake-form">
            <div className="form-group">
              <label>Amount</label>
              <input
                type="number"
                value={stakeAmount}
                onChange={(e) => setStakeAmount(e.target.value)}
                placeholder={`Min: ${selectedValidator.minStake}`}
                className="form-input"
              />
              <div className="quick-amounts">
                <button onClick={() => setStakeAmount('10')}>10</button>
                <button onClick={() => setStakeAmount('50')}>50</button>
                <button onClick={() => setStakeAmount('100')}>100</button>
                <button onClick={() => setStakeAmount('500')}>500</button>
              </div>
            </div>
            
            <div className="stake-summary">
              <div className="summary-row">
                <span>Validator</span>
                <span>{selectedValidator.name}</span>
              </div>
              <div className="summary-row">
                <span>Est. Annual Reward</span>
                <span>{(parseFloat(stakeAmount || '0') * selectedValidator.apr / 100).toFixed(4)} {selectedChainData?.symbol}</span>
              </div>
              <div className="summary-row">
                <span>Est. Daily Reward</span>
                <span>{(parseFloat(stakeAmount || '0') * selectedValidator.apr / 100 / 365).toFixed(6)} {selectedChainData?.symbol}</span>
              </div>
            </div>

            <button
              className="btn btn-primary btn-large"
              onClick={handleStake}
              disabled={isLoading || !stakeAmount}
            >
              {isLoading ? 'Staking...' : 'Stake Now'}
            </button>
          </div>
        </div>
      )}

      <style>{`
        .staking-page {
          padding: 20px;
          max-width: 900px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 24px;
        }

        .page-header h1 {
          font-size: 28px;
          margin-bottom: 8px;
        }

        .chain-selector {
          display: flex;
          gap: 8px;
          margin-bottom: 24px;
          overflow-x: auto;
        }

        .chain-tab {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 12px 20px;
          background: var(--card-bg, #1e1e2e);
          border: 2px solid transparent;
          border-radius: 12px;
          cursor: pointer;
          transition: all 0.2s;
        }

        .chain-tab:hover {
          border-color: var(--primary-color, #6c5ce7);
        }

        .chain-tab.active {
          border-color: var(--primary-color, #6c5ce7);
          background: var(--primary-color, #6c5ce7)22;
        }

        .chain-symbol {
          font-weight: 700;
          font-size: 18px;
        }

        .section {
          background: var(--card-bg, #1e1e2e);
          border-radius: 12px;
          padding: 20px;
          margin-bottom: 20px;
        }

        .section h2 {
          font-size: 20px;
          margin-bottom: 16px;
        }

        .validators-list {
          display: grid;
          gap: 12px;
        }

        .validator-card {
          background: var(--input-bg, #2a2a3e);
          border: 2px solid transparent;
          border-radius: 12px;
          padding: 16px;
          cursor: pointer;
          transition: all 0.2s;
        }

        .validator-card:hover {
          border-color: var(--primary-color, #6c5ce7);
        }

        .validator-card.selected {
          border-color: var(--primary-color, #6c5ce7);
        }

        .validator-header {
          display: flex;
          justify-content: space-between;
          margin-bottom: 12px;
        }

        .validator-name {
          font-size: 18px;
          font-weight: 600;
        }

        .validator-risk {
          font-size: 12px;
          font-weight: 600;
          text-transform: uppercase;
        }

        .validator-stats {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 12px;
        }

        .stat {
          text-align: center;
        }

        .stat .label {
          display: block;
          font-size: 12px;
          color: var(--text-muted, #888);
          margin-bottom: 4px;
        }

        .stat .value {
          font-size: 16px;
          font-weight: 600;
        }

        .stake-form {
          max-width: 400px;
        }

        .form-group {
          margin-bottom: 20px;
        }

        .form-group label {
          display: block;
          margin-bottom: 8px;
          font-weight: 600;
        }

        .form-input {
          width: 100%;
          padding: 12px;
          background: var(--input-bg, #2a2a3e);
          border: 1px solid var(--border-color, #333);
          border-radius: 8px;
          color: var(--text-primary, #fff);
          font-size: 16px;
        }

        .quick-amounts {
          display: flex;
          gap: 8px;
          margin-top: 8px;
        }

        .quick-amounts button {
          flex: 1;
          padding: 8px;
          background: var(--input-bg, #2a2a3e);
          border: 1px solid var(--border-color, #333);
          border-radius: 6px;
          color: var(--text-secondary, #ccc);
          cursor: pointer;
        }

        .stake-summary {
          background: var(--info-bg, #2a2a4e);
          padding: 16px;
          border-radius: 8px;
          margin-bottom: 20px;
        }

        .summary-row {
          display: flex;
          justify-content: space-between;
          padding: 8px 0;
          border-bottom: 1px solid var(--border-color, #333);
        }

        .summary-row:last-child {
          border-bottom: none;
        }

        .loading {
          text-align: center;
          padding: 40px;
          color: var(--text-muted, #888);
        }
      `}</style>
    </div>
  );
};

export default StakingPage;
