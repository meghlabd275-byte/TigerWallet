// Feature Management Admin Page
// Manages Red Packets, Claims, Convert System, and Other Features

import React, { useState } from 'react';
import './FeatureManagementPage.css';

interface RedPacketRecord {
  id: string;
  sender: string;
  senderAddress: string;
  token: string;
  amount: number;
  totalCount: number;
  receivedCount: number;
  remainingAmount: number;
  message: string;
  status: 'active' | 'completed' | 'expired';
  createdAt: string;
}

interface ClaimRecord {
  id: string;
  user: string;
  type: 'airdrop' | 'bonus' | 'reward' | 'rebate';
  token: string;
  amount: number;
  status: 'pending' | 'approved' | 'rejected';
  createdAt: string;
}

interface ConvertPair {
  from: string;
  to: string;
  rate: number;
  fee: number;
  enabled: boolean;
}

const FeatureManagementPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'redpacket' | 'claim' | 'convert' | 'settings'>('redpacket');
  
  const [redPackets, setRedPackets] = useState<RedPacketRecord[]>([
    { id: '1', sender: 'User123', senderAddress: '0x1234...5678', token: 'USDT', amount: 1000, totalCount: 10, receivedCount: 7, remainingAmount: 300, message: 'Good luck! 🧧', status: 'active', createdAt: '2024-01-15 10:30:00' },
    { id: '2', sender: 'CryptoKing', senderAddress: '0xabcd...efgh', token: 'USDT', amount: 500, totalCount: 5, receivedCount: 5, remainingAmount: 0, message: 'Happy New Year!', status: 'completed', createdAt: '2024-01-14 15:20:00' },
    { id: '3', sender: 'TraderPro', senderAddress: '0x9876...5432', token: 'ETH', amount: 2, totalCount: 20, receivedCount: 0, remainingAmount: 2, message: 'Claim your rewards!', status: 'active', createdAt: '2024-01-15 09:00:00' },
    { id: '4', sender: 'WhaleAlert', senderAddress: '0xdef0...1234', token: 'BTC', amount: 0.5, totalCount: 50, receivedCount: 50, remainingAmount: 0, message: 'Big airdrop!', status: 'completed', createdAt: '2024-01-10 12:00:00' },
  ]);

  const [claims, setClaims] = useState<ClaimRecord[]>([
    { id: '1', user: 'User001', type: 'airdrop', token: 'TIGER', amount: 100, status: 'approved', createdAt: '2024-01-15 10:00:00' },
    { id: '2', user: 'User002', type: 'bonus', token: 'USDT', amount: 50, status: 'pending', createdAt: '2024-01-15 11:30:00' },
    { id: '3', user: 'User003', type: 'reward', token: 'USDT', amount: 250, status: 'approved', createdAt: '2024-01-14 09:15:00' },
    { id: '4', user: 'User004', type: 'rebate', token: 'USDC', amount: 15.5, status: 'rejected', createdAt: '2024-01-13 16:45:00' },
  ]);

  const [convertPairs, setConvertPairs] = useState<ConvertPair[]>([
    { from: 'BTC', to: 'USDT', rate: 43250, fee: 0.1, enabled: true },
    { from: 'ETH', to: 'USDT', rate: 2280, fee: 0.1, enabled: true },
    { from: 'BNB', to: 'USDT', rate: 312.5, fee: 0.1, enabled: true },
    { from: 'SOL', to: 'USDT', rate: 98.75, fee: 0.1, enabled: true },
    { from: 'USDC', to: 'USDT', rate: 1.0001, fee: 0.05, enabled: true },
    { from: 'USDT', to: 'USDC', rate: 0.9999, fee: 0.05, enabled: true },
    { from: 'DOGE', to: 'USDT', rate: 0.082, fee: 0.15, enabled: false },
  ]);

  const [featureSettings, setFeatureSettings] = useState({
    redPacketEnabled: true,
    redPacketMinAmount: 1,
    redPacketMaxAmount: 100000,
    redPacketMaxCount: 1000,
    claimEnabled: true,
    claimAutoApprove: false,
    claimMinAmount: 1,
    convertEnabled: true,
    convertMinAmount: 1,
    convertMaxAmount: 1000000,
  });

  const handleApproveClaim = (id: string) => {
    setClaims(claims.map(c => 
      c.id === id ? { ...c, status: 'approved' as const } : c
    ));
  };

  const handleRejectClaim = (id: string) => {
    setClaims(claims.map(c => 
      c.id === id ? { ...c, status: 'rejected' as const } : c
    ));
  };

  const handleToggleConvert = (from: string, to: string) => {
    setConvertPairs(convertPairs.map(p => 
      p.from === from && p.to === to ? { ...p, enabled: !p.enabled } : p
    ));
  };

  const handleSettingChange = (key: string, value: any) => {
    setFeatureSettings({ ...featureSettings, [key]: value });
  };

  return (
    <div className="feature-management-page">
      <div className="page-header">
        <h1>Feature Management</h1>
        <button className="save-btn">Save Changes</button>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'redpacket' ? 'active' : ''} 
          onClick={() => setActiveTab('redpacket')}
        >
          🧧 Red Packets ({redPackets.length})
        </button>
        <button 
          className={activeTab === 'claim' ? 'active' : ''} 
          onClick={() => setActiveTab('claim')}
        >
          🎁 Claims ({claims.filter(c => c.status === 'pending').length})
        </button>
        <button 
          className={activeTab === 'convert' ? 'active' : ''} 
          onClick={() => setActiveTab('convert')}
        >
          💱 Convert System
        </button>
        <button 
          className={activeTab === 'settings' ? 'active' : ''} 
          onClick={() => setActiveTab('settings')}
        >
          ⚙️ Settings
        </button>
      </div>

      {activeTab === 'redpacket' && (
        <div className="red-packet-section">
          <div className="section-header">
            <h2>Red Packet Management</h2>
            <div className="stats">
              <span>Active: {redPackets.filter(p => p.status === 'active').length}</span>
              <span>Completed: {redPackets.filter(p => p.status === 'completed').length}</span>
              <span>Expired: {redPackets.filter(p => p.status === 'expired').length}</span>
            </div>
          </div>

          <div className="records-table">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Sender</th>
                  <th>Token</th>
                  <th>Amount</th>
                  <th>Packets</th>
                  <th>Received</th>
                  <th>Message</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {redPackets.map(packet => (
                  <tr key={packet.id}>
                    <td>{packet.id}</td>
                    <td>
                      <div className="sender-info">
                        <span className="username">{packet.sender}</span>
                        <span className="address">{packet.senderAddress}</span>
                      </div>
                    </td>
                    <td>{packet.token}</td>
                    <td>{packet.amount}</td>
                    <td>{packet.totalCount}</td>
                    <td>{packet.receivedCount}/{packet.totalCount}</td>
                    <td className="message">{packet.message}</td>
                    <td>
                      <span className={`status-badge ${packet.status}`}>
                        {packet.status}
                      </span>
                    </td>
                    <td>{packet.createdAt}</td>
                    <td>
                      <button className="action-btn view">View</button>
                      <button className="action-btn delete">Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'claim' && (
        <div className="claim-section">
          <div className="section-header">
            <h2>Claim Management</h2>
            <div className="stats">
              <span>Pending: {claims.filter(c => c.status === 'pending').length}</span>
              <span>Approved: {claims.filter(c => c.status === 'approved').length}</span>
              <span>Rejected: {claims.filter(c => c.status === 'rejected').length}</span>
            </div>
          </div>

          <div className="records-table">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>User</th>
                  <th>Type</th>
                  <th>Token</th>
                  <th>Amount</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {claims.map(claim => (
                  <tr key={claim.id}>
                    <td>{claim.id}</td>
                    <td>{claim.user}</td>
                    <td>
                      <span className={`type-badge ${claim.type}`}>
                        {claim.type}
                      </span>
                    </td>
                    <td>{claim.token}</td>
                    <td>{claim.amount}</td>
                    <td>
                      <span className={`status-badge ${claim.status}`}>
                        {claim.status}
                      </span>
                    </td>
                    <td>{claim.createdAt}</td>
                    <td>
                      {claim.status === 'pending' && (
                        <>
                          <button 
                            className="action-btn approve"
                            onClick={() => handleApproveClaim(claim.id)}
                          >
                            Approve
                          </button>
                          <button 
                            className="action-btn reject"
                            onClick={() => handleRejectClaim(claim.id)}
                          >
                            Reject
                          </button>
                        </>
                      )}
                      {claim.status !== 'pending' && (
                        <span className="processed">Processed</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'convert' && (
        <div className="convert-section">
          <div className="section-header">
            <h2>Convert System Management</h2>
            <button className="add-btn">+ Add Convert Pair</button>
          </div>

          <div className="convert-pairs-grid">
            {convertPairs.map((pair, idx) => (
              <div key={idx} className={`convert-pair-card ${pair.enabled ? 'enabled' : 'disabled'}`}>
                <div className="pair-header">
                  <div className="pair-tokens">
                    <span className="from-token">{pair.from}</span>
                    <span className="arrow">→</span>
                    <span className="to-token">{pair.to}</span>
                  </div>
                  <label className="toggle">
                    <input 
                      type="checkbox" 
                      checked={pair.enabled}
                      onChange={() => handleToggleConvert(pair.from, pair.to)}
                    />
                    <span className="slider"></span>
                  </label>
                </div>
                <div className="pair-details">
                  <div className="detail">
                    <span className="label">Rate</span>
                    <span className="value">1 {pair.from} = {pair.rate} {pair.to}</span>
                  </div>
                  <div className="detail">
                    <span className="label">Fee</span>
                    <span className="value">{pair.fee}%</span>
                  </div>
                </div>
                <div className="pair-actions">
                  <button className="edit-btn">Edit</button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="settings-section">
          <div className="section-header">
            <h2>Feature Settings</h2>
          </div>

          <div className="settings-groups">
            <div className="settings-group">
              <h3>Red Packet Settings</h3>
              <div className="setting-item">
                <label>
                  <input 
                    type="checkbox" 
                    checked={featureSettings.redPacketEnabled}
                    onChange={(e) => handleSettingChange('redPacketEnabled', e.target.checked)}
                  />
                  Enable Red Packets
                </label>
              </div>
              <div className="setting-item">
                <label>Minimum Amount</label>
                <input 
                  type="number" 
                  value={featureSettings.redPacketMinAmount}
                  onChange={(e) => handleSettingChange('redPacketMinAmount', parseFloat(e.target.value))}
                />
              </div>
              <div className="setting-item">
                <label>Maximum Amount</label>
                <input 
                  type="number" 
                  value={featureSettings.redPacketMaxAmount}
                  onChange={(e) => handleSettingChange('redPacketMaxAmount', parseFloat(e.target.value))}
                />
              </div>
              <div className="setting-item">
                <label>Maximum Packets per Red Packet</label>
                <input 
                  type="number" 
                  value={featureSettings.redPacketMaxCount}
                  onChange={(e) => handleSettingChange('redPacketMaxCount', parseInt(e.target.value))}
                />
              </div>
            </div>

            <div className="settings-group">
              <h3>Claim Settings</h3>
              <div className="setting-item">
                <label>
                  <input 
                    type="checkbox" 
                    checked={featureSettings.claimEnabled}
                    onChange={(e) => handleSettingChange('claimEnabled', e.target.checked)}
                  />
                  Enable Claims
                </label>
              </div>
              <div className="setting-item">
                <label>
                  <input 
                    type="checkbox" 
                    checked={featureSettings.claimAutoApprove}
                    onChange={(e) => handleSettingChange('claimAutoApprove', e.target.checked)}
                  />
                  Auto-Approve Claims
                </label>
              </div>
              <div className="setting-item">
                <label>Minimum Claim Amount</label>
                <input 
                  type="number" 
                  value={featureSettings.claimMinAmount}
                  onChange={(e) => handleSettingChange('claimMinAmount', parseFloat(e.target.value))}
                />
              </div>
            </div>

            <div className="settings-group">
              <h3>Convert Settings</h3>
              <div className="setting-item">
                <label>
                  <input 
                    type="checkbox" 
                    checked={featureSettings.convertEnabled}
                    onChange={(e) => handleSettingChange('convertEnabled', e.target.checked)}
                  />
                  Enable Convert
                </label>
              </div>
              <div className="setting-item">
                <label>Minimum Convert Amount</label>
                <input 
                  type="number" 
                  value={featureSettings.convertMinAmount}
                  onChange={(e) => handleSettingChange('convertMinAmount', parseFloat(e.target.value))}
                />
              </div>
              <div className="setting-item">
                <label>Maximum Convert Amount</label>
                <input 
                  type="number" 
                  value={featureSettings.convertMaxAmount}
                  onChange={(e) => handleSettingChange('convertMaxAmount', parseFloat(e.target.value))}
                />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default FeatureManagementPage;
