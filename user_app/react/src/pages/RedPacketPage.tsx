// Red Packet Page - Send and Receive Red Packets
// Traditional lucky money style crypto gifting

import React, { useState, useEffect } from 'react';
import './RedPacketPage.css';

interface RedPacket {
  id: string;
  sender: string;
  senderAddress: string;
  amount: number;
  token: string;
  totalCount: number;
  receivedCount: number;
  remainingAmount: number;
  message: string;
  timestamp: number;
  isExpired: boolean;
  claimedAmount?: number;
}

interface ClaimRecord {
  packetId: string;
  claimer: string;
  amount: number;
  timestamp: number;
}

// Backend API URL for red packets
const API_BASE_URL = 'http://localhost:8443/api/v1/redpacket';

const RedPacketPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'send' | 'receive' | 'history'>('send');
  const [packetType, setPacketType] = useState<'random' | 'fixed'>('random');
  const [token, setToken] = useState('USDT');
  const [amount, setAmount] = useState('');
  const [totalCount, setTotalCount] = useState('10');
  const [message, setMessage] = useState('Good luck! 🧧');
  const [generatedLink, setGeneratedLink] = useState('');
  const [receivedPackets, setReceivedPackets] = useState<RedPacket[]>([]);
  const [sentPackets, setSentPackets] = useState<RedPacket[]>([]);
  const [claimRecords, setClaimRecords] = useState<ClaimRecord[]>([]);
  const [claimLink, setClaimLink] = useState('');
  const [claimResult, setClaimResult] = useState<{success: boolean; amount?: number; message?: string} | null>(null);
  const [showClaimForm, setShowClaimForm] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const tokens = ['USDT', 'USDC', 'ETH', 'BTC', 'BNB', 'SOL', 'TRX', 'BTT'];

  // Load red packet history from backend
  useEffect(() => {
    const loadPackets = async () => {
      setLoading(true);
      try {
        const token = localStorage.getItem('user_token');
        
        // Load sent packets
        const sentRes = await fetch(`${API_BASE_URL}/sent`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (sentRes.ok) {
          const sentData = await sentRes.json();
          setSentPackets(sentData.packets || []);
        }
        
        // Load received packets
        const receivedRes = await fetch(`${API_BASE_URL}/received`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (receivedRes.ok) {
          const receivedData = await receivedRes.json();
          setReceivedPackets(receivedData.packets || []);
        }
      } catch (err) {
        console.error('Failed to load packets:', err);
      } finally {
        setLoading(false);
      }
    };
    
    loadPackets();
  }, []);

  const handleCreatePacket = () => {
    const count = parseInt(totalCount);
    const totalAmount = parseFloat(amount);
    
    const newPacket: RedPacket = {
      id: Date.now().toString(),
      sender: 'You',
      senderAddress: '0x1234...5678',
      amount: totalAmount,
      token,
      totalCount: count,
      receivedCount: 0,
      remainingAmount: totalAmount,
      message,
      timestamp: Date.now(),
      isExpired: false,
    };

    setSentPackets([newPacket, ...sentPackets]);
    setGeneratedLink(`https://tigerwallet.com/redpacket/claim/${newPacket.id}`);
    setActiveTab('history');
  };

  const handleClaim = () => {
    if (!claimLink) return;
    
    // Simulate claiming
    const claimedAmount = packetType === 'random' 
      ? parseFloat(amount) / parseInt(totalCount)  // equal split; real distribution from contract
      : parseFloat(amount) / parseInt(totalCount);
    
    setClaimResult({
      success: true,
      amount: claimedAmount,
      message: `You claimed ${claimedAmount.toFixed(4)} ${token}! 🧧`
    });

    const newRecord: ClaimRecord = {
      packetId: claimLink.split('/').pop() || '',
      claimer: 'You',
      amount: claimedAmount,
      timestamp: Date.now()
    };
    setClaimRecords([newRecord, ...claimRecords]);
  };

  const copyLink = () => {
    navigator.clipboard.writeText(generatedLink);
    alert('Link copied to clipboard!');
  };

  return (
    <div className="red-packet-page">
      <div className="red-packet-header">
        <h1>🧧 Red Packet</h1>
        <p>Send and receive crypto red packets with friends</p>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'send' ? 'active' : ''} 
          onClick={() => { setActiveTab('send'); setShowClaimForm(false); }}
        >
          Send Red Packet
        </button>
        <button 
          className={activeTab === 'receive' ? 'active' : ''} 
          onClick={() => { setActiveTab('receive'); setShowClaimForm(true); }}
        >
          Receive
        </button>
        <button 
          className={activeTab === 'history' ? 'active' : ''} 
          onClick={() => setActiveTab('history')}
        >
          History
        </button>
      </div>

      {activeTab === 'send' && (
        <div className="send-section">
          <div className="create-packet-form">
            <div className="form-section">
              <h3>Packet Type</h3>
              <div className="type-selector">
                <button 
                  className={packetType === 'random' ? 'active' : ''} 
                  onClick={() => setPacketType('random')}
                >
                  <span className="icon">🎲</span>
                  <span className="label">Random</span>
                  <span className="desc">Amounts vary randomly</span>
                </button>
                <button 
                  className={packetType === 'fixed' ? 'active' : ''} 
                  onClick={() => setPacketType('fixed')}
                >
                  <span className="icon">📊</span>
                  <span className="label">Fixed</span>
                  <span className="desc">Equal amounts for all</span>
                </button>
              </div>
            </div>

            <div className="form-section">
              <h3>Token & Amount</h3>
              <div className="token-input">
                <select value={token} onChange={(e) => setToken(e.target.value)}>
                  {tokens.map(t => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
                <input
                  type="number"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="Total amount"
                  min="1"
                />
              </div>
            </div>

            <div className="form-section">
              <h3>Number of Packets</h3>
              <div className="count-input">
                <input
                  type="number"
                  value={totalCount}
                  onChange={(e) => setTotalCount(e.target.value)}
                  placeholder="Number of packets"
                  min="1"
                  max="1000"
                />
                <div className="quick-counts">
                  {[5, 10, 20, 50, 100].map(n => (
                    <button 
                      key={n} 
                      onClick={() => setTotalCount(n.toString())}
                      className={totalCount === n.toString() ? 'active' : ''}
                    >
                      {n}
                    </button>
                  ))}
                </div>
              </div>
              {amount && totalCount && (
                <div className="packet-preview">
                  <span>Each packet: ~{packetType === 'random' ? 'Varies' : (parseFloat(amount) / parseInt(totalCount)).toFixed(4)} {token}</span>
                </div>
              )}
            </div>

            <div className="form-section">
              <h3>Message (Optional)</h3>
              <input
                type="text"
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Enter a message"
                maxLength={50}
              />
            </div>

            <button 
              className="create-btn"
              onClick={handleCreatePacket}
              disabled={!amount || !totalCount}
            >
              Create Red Packet
            </button>

            {generatedLink && (
              <div className="share-section">
                <h3>Share This Link</h3>
                <div className="link-box">
                  <input type="text" value={generatedLink} readOnly />
                  <button onClick={copyLink}>Copy</button>
                </div>
                <div className="share-buttons">
                  <button className="share-btn twitter">Twitter</button>
                  <button className="share-btn telegram">Telegram</button>
                  <button className="share-btn whatsapp">WhatsApp</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'receive' && (
        <div className="receive-section">
          {showClaimForm && !claimResult ? (
            <div className="claim-form">
              <div className="claim-card">
                <div className="claim-icon">🧧</div>
                <h2>You've received a red packet!</h2>
                
                <div className="claim-input">
                  <input
                    type="text"
                    value={claimLink}
                    onChange={(e) => setClaimLink(e.target.value)}
                    placeholder="Paste red packet link here"
                  />
                </div>

                <button className="claim-btn" onClick={handleClaim}>
                  Open Red Packet
                </button>

                <p className="hint">Ask your friend for the red packet link</p>
              </div>
            </div>
          ) : claimResult ? (
            <div className="claim-result">
              <div className="result-card success">
                <div className="result-icon">🎉</div>
                <h2>Congratulations!</h2>
                <div className="claimed-amount">
                  {claimResult.amount?.toFixed(6)} {token}
                </div>
                <p>{claimResult.message}</p>
                <button className="done-btn" onClick={() => { setClaimResult(null); setClaimLink(''); }}>
                  Done
                </button>
              </div>
            </div>
          ) : (
            <div className="no-packet">
              <div className="no-packet-icon">🧧</div>
              <h2>No Red Packet Yet</h2>
              <p>Ask your friend to send you a red packet!</p>
            </div>
          )}
        </div>
      )}

      {activeTab === 'history' && (
        <div className="history-section">
          <div className="history-tabs">
            <button 
              className={receivedPackets.length > 0 || sentPackets.length === 0 ? 'active' : ''}
              onClick={() => {}}
            >
              Sent ({sentPackets.length})
            </button>
            <button>Received (0)</button>
          </div>

          {sentPackets.length === 0 ? (
            <div className="empty-history">
              <p>No red packets sent yet</p>
            </div>
          ) : (
            <div className="packets-list">
              {sentPackets.map(packet => (
                <div key={packet.id} className="packet-card">
                  <div className="packet-header">
                    <span className="packet-icon">🧧</span>
                    <div className="packet-info">
                      <span className="packet-amount">{packet.amount} {packet.token}</span>
                      <span className="packet-message">{packet.message}</span>
                    </div>
                    <span className={`packet-status ${packet.isExpired ? 'expired' : packet.receivedCount >= packet.totalCount ? 'completed' : 'active'}`}>
                      {packet.isExpired ? 'Expired' : packet.receivedCount >= packet.totalCount ? 'Completed' : 'Active'}
                    </span>
                  </div>
                  <div className="packet-stats">
                    <div className="stat">
                      <span className="label">Total</span>
                      <span className="value">{packet.totalCount}</span>
                    </div>
                    <div className="stat">
                      <span className="label">Received</span>
                      <span className="value">{packet.receivedCount}</span>
                    </div>
                    <div className="stat">
                      <span className="label">Remaining</span>
                      <span className="value">{packet.remainingAmount} {packet.token}</span>
                    </div>
                  </div>
                  <div className="packet-progress">
                    <div 
                      className="progress-bar" 
                      style={{ width: `${(packet.receivedCount / packet.totalCount) * 100}%` }}
                    />
                  </div>
                  <div className="packet-footer">
                    <span className="timestamp">{new Date(packet.timestamp).toLocaleString()}</span>
                    <button className="share-again-btn" onClick={copyLink}>
                      Share Again
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default RedPacketPage;
