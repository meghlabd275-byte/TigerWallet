'use client';

import React, { useState, useEffect } from 'react';

interface P2PMerchant {
  id: string;
  userId: string;
  username: string;
  avatar: string;
  status: 'PENDING' | 'ACTIVE' | 'SUSPENDED' | 'BANNED';
  collateralAmount: number;
  collateralToken: string;
  totalTrades: number;
  totalVolume: number;
  completionRate: number;
  avgResponseTime: number;
  avgReleaseTime: number;
  rating: number;
  totalReviews: number;
  isVerified: boolean;
  traderLevel: string;
  joinedAt: Date;
}

interface P2PAdvert {
  id: string;
  userId: string;
  username: string;
  avatar: string;
  side: 'BUY' | 'SELL';
  token: string;
  fiatCurrency: string;
  paymentMethod: string;
  price: number;
  minAmount: number;
  maxAmount: number;
  availableAmount: number;
  ordersCompleted: number;
  completionRate: number;
  avgReleaseTime: number;
  isOnline: boolean;
  isMerchant: boolean;
  merchantLevel?: string;
  collateralLocked?: number;
  isVerified: boolean;
  securityScore: number;
}

interface P2POrder {
  id: string;
  advertId: string;
  side: string;
  token: string;
  fiatCurrency: string;
  paymentMethod: string;
  price: number;
  amount: number;
  fiatAmount: number;
  status: string;
  buyerSecurityDeposit?: number;
  sellerSecurityDeposit?: number;
  createdAt: Date;
}

const MERCHANT_LEVELS: Record<string, { collateral: number; color: string; icon: string }> = {
  'NEWBIE': { collateral: 100, color: 'gray', icon: '🌱' },
  'BRONZE': { collateral: 250, color: '#cd7f32', icon: '🥉' },
  'SILVER': { collateral: 500, color: '#c0c0c0', icon: '🥈' },
  'GOLD': { collateral: 1000, color: '#ffd700', icon: '🥇' },
  'PLATINUM': { collateral: 2500, color: '#e5e4e2', icon: '💎' },
  'DIAMOND': { collateral: 5000, color: '#b9f2ff', icon: '👑' },
};

const TOKENS = ['USDT', 'BTC', 'ETH', 'USDC', 'BNB'];
const FIAT_CURRENCIES = ['USD', 'EUR', 'GBP', 'CNY', 'INR'];
const PAYMENT_METHODS = ['Bank Transfer', 'PayPal', 'AliPay', 'WeChat Pay', 'UPI', 'Gift Card'];

const generateAdverts = (): P2PAdvert[] => {
  const users = [
    { username: 'CryptoTrader1', avatar: '🧑‍💼', online: true, merchant: true, level: 'GOLD', collateral: 1000, verified: true, score: 98 },
    { username: 'BitSeller', avatar: '👨‍💻', online: true, merchant: true, level: 'SILVER', collateral: 500, verified: true, score: 95 },
    { username: 'FastTrade', avatar: '⚡', online: false, merchant: false, level: '', collateral: 0, verified: false, score: 85 },
    { username: 'P2PPro', avatar: '🎯', online: true, merchant: true, level: 'PLATINUM', collateral: 2500, verified: true, score: 99 },
    { username: 'SecureDeal', avatar: '🔒', online: true, merchant: true, level: 'DIAMOND', collateral: 5000, verified: true, score: 100 },
    { username: 'NewTrader', avatar: '🌱', online: true, merchant: false, level: '', collateral: 0, verified: false, score: 70 },
  ];
  
  const basePrices: Record<string, number> = { 'USDT': 1, 'BTC': 43250, 'ETH': 2280, 'USDC': 1, 'BNB': 312.5 };
  
  const adverts: P2PAdvert[] = [];
  let id = 0;
  
  for (const user of users) {
    for (const token of TOKENS) {
      for (const fiat of FIAT_CURRENCIES) {
        const priceVariation = (Math.random() * 10 - 5) / 1000;
        const basePrice = basePrices[token] || 10;
        
        adverts.push({
          id: `advert_${id++}`,
          userId: `user_${users.indexOf(user)}`,
          username: user.username,
          avatar: user.avatar,
          side: id % 2 === 0 ? 'BUY' : 'SELL',
          token,
          fiatCurrency: fiat,
          paymentMethod: PAYMENT_METHODS[id % PAYMENT_METHODS.length],
          price: basePrice * (1 + priceVariation),
          minAmount: fiat === 'USD' ? 10 : 100,
          maxAmount: fiat === 'USD' ? 5000 : 50000,
          availableAmount: (id * 0.5 + 1) * basePrice,
          ordersCompleted: 50 + id * 10,
          completionRate: user.merchant ? 95 + (id % 5) : 80 + (id % 15),
          avgReleaseTime: user.merchant ? 2 + (id % 5) : 5 + (id % 10),
          isOnline: user.online,
          isMerchant: user.merchant,
          merchantLevel: user.level,
          collateralLocked: user.collateral,
          isVerified: user.verified,
          securityScore: user.score,
        });
      }
    }
  }
  return adverts;
};

const MerchantBadge = ({ level }: { level?: string }) => {
  if (!level || !MERCHANT_LEVELS[level]) return null;
  const config = MERCHANT_LEVELS[level];
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold"
      style={{ backgroundColor: config.color + '30', color: config.color, border: `1px solid ${config.color}` }}>
      {config.icon} {level}
    </span>
  );
};

const SecurityScore = ({ score }: { score: number }) => {
  const color = score >= 90 ? 'text-green-400' : score >= 70 ? 'text-yellow-400' : 'text-red-400';
  return (
    <div className="flex items-center gap-1">
      <span className={`text-sm font-bold ${color}`}>{score}%</span>
      <div className="w-16 h-2 bg-gray-700 rounded-full overflow-hidden">
        <div className={`h-full ${score >= 90 ? 'bg-green-400' : score >= 70 ? 'bg-yellow-400' : 'bg-red-400'}`} 
          style={{ width: `${score}%` }}></div>
      </div>
    </div>
  );
};

export default function P2PTradingPage() {
  const [adverts, setAdverts] = useState<P2PAdvert[]>([]);
  const [selectedToken, setSelectedToken] = useState('USDT');
  const [selectedFiat, setSelectedFiat] = useState('USD');
  const [selectedSide, setSelectedSide] = useState<'BUY' | 'SELL'>('BUY');
  const [selectedPayment, setSelectedPayment] = useState('All');
  const [activeTab, setActiveTab] = useState<'orders' | 'create' | 'merchant'>('orders');
  const [orders, setOrders] = useState<P2POrder[]>([]);
  const [showMerchantProfile, setShowMerchantProfile] = useState<string | null>(null);

  useEffect(() => { setAdverts(generateAdverts()); }, []);

  const filteredAdverts = adverts.filter(a => 
    a.token === selectedToken && a.fiatCurrency === selectedFiat &&
    a.side === selectedSide && (selectedPayment === 'All' || a.paymentMethod === selectedPayment)
  );

  const handleBuy = (advert: P2PAdvert) => {
    const buyerDeposit = advert.isMerchant ? advert.price * advert.minAmount * 0.02 : advert.price * advert.minAmount * 0.05;
    const sellerDeposit = advert.price * advert.minAmount * 0.03;
    
    const order: P2POrder = {
      id: `order_${Date.now()}`, advertId: advert.id, side: advert.side, token: advert.token,
      fiatCurrency: advert.fiatCurrency, paymentMethod: advert.paymentMethod, price: advert.price,
      amount: advert.minAmount, fiatAmount: advert.minAmount * advert.price, status: 'PENDING',
      buyerSecurityDeposit: buyerDeposit, sellerSecurityDeposit: sellerDeposit, createdAt: new Date(),
    };
    setOrders([order, ...orders]);
    alert(`Order created!\n\n🔒 Security Deposits:\n• Buyer Deposit: ${buyerDeposit.toFixed(4)} ${advert.token}\n• Seller Deposit: ${sellerDeposit.toFixed(4)} ${advert.token}\n\nThese deposits will be held in escrow until the trade is completed.`);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold">P2P Trading</h1>
          <p className="text-gray-400 mt-1">Secure peer-to-peer trading with merchant protection</p>
        </div>

        <div className="bg-gradient-to-r from-blue-900 to-purple-900 rounded-xl p-4 mb-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="text-3xl">🛡️</span>
              <div>
                <div className="font-bold">Protected Trading</div>
                <div className="text-sm text-gray-300">All trades include security deposits to prevent scams</div>
              </div>
            </div>
            <div className="flex gap-6 text-sm">
              <div className="text-center"><div className="font-bold text-green-400">2-5%</div><div className="text-gray-400">Security Deposit</div></div>
              <div className="text-center"><div className="font-bold text-blue-400">$5M</div><div className="text-gray-400">Protection Fund</div></div>
              <div className="text-center"><div className="font-bold text-purple-400">99.5%</div><div className="text-gray-400">Success Rate</div></div>
            </div>
          </div>
        </div>

        <div className="bg-gray-800 rounded-xl p-6 mb-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label className="block text-sm text-gray-400 mb-2">Token</label>
              <select value={selectedToken} onChange={(e) => setSelectedToken(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">
                {TOKENS.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">Fiat Currency</label>
              <select value={selectedFiat} onChange={(e) => setSelectedFiat(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">
                {FIAT_CURRENCIES.map(f => <option key={f} value={f}>{f}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">Payment Method</label>
              <select value={selectedPayment} onChange={(e) => setSelectedPayment(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">
                <option value="All">All Methods</option>
                {PAYMENT_METHODS.map(p => <option key={p} value={p}>{p}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">Type</label>
              <div className="flex space-x-2">
                <button onClick={() => setSelectedSide('BUY')}
                  className={`flex-1 py-3 rounded-lg font-bold ${selectedSide === 'BUY' ? 'bg-green-600' : 'bg-gray-700'}`}>Buy</button>
                <button onClick={() => setSelectedSide('SELL')}
                  className={`flex-1 py-3 rounded-lg font-bold ${selectedSide === 'SELL' ? 'bg-red-600' : 'bg-gray-700'}`}>Sell</button>
              </div>
            </div>
          </div>
        </div>

        <div className="flex space-x-4 mb-6 border-b border-gray-700">
          <button onClick={() => setActiveTab('orders')} className={`pb-3 px-4 font-medium ${activeTab === 'orders' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}>Browse Ads</button>
          <button onClick={() => setActiveTab('create')} className={`pb-3 px-4 font-medium ${activeTab === 'create' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}>Create Ad</button>
          <button onClick={() => setActiveTab('merchant')} className={`pb-3 px-4 font-medium ${activeTab === 'merchant' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}>Merchant Center</button>
        </div>

        {activeTab === 'orders' && (
          <div className="space-y-4">
            {filteredAdverts.map(advert => (
              <div key={advert.id} className="bg-gray-800 rounded-xl p-6">
                <div className="flex justify-between items-start">
                  <div className="flex items-center space-x-4">
                    <div className="w-14 h-14 bg-gray-700 rounded-full flex items-center justify-center text-3xl">{advert.avatar}</div>
                    <div>
                      <div className="font-bold flex items-center gap-2">
                        {advert.username}
                        <MerchantBadge level={advert.merchantLevel} />
                        {advert.isVerified && <span className="text-blue-400">✓</span>}
                      </div>
                      <div className="text-sm text-gray-400">{advert.ordersCompleted} trades • {advert.completionRate}% completion</div>
                      <div className="mt-1"><SecurityScore score={advert.securityScore} /></div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className={`text-xl font-bold ${advert.side === 'BUY' ? 'text-green-400' : 'text-red-400'}`}>{advert.side === 'BUY' ? 'Buy' : 'Sell'} {advert.token}</div>
                    <div className="text-2xl font-bold">{advert.price.toFixed(2)} {advert.fiatCurrency}</div>
                  </div>
                </div>
                {advert.isMerchant && advert.collateralLocked && (
                  <div className="mt-3 bg-green-900/30 border border-green-500/30 rounded-lg p-2 flex items-center gap-2">
                    <span className="text-green-400">🔒</span>
                    <span className="text-sm"><span className="font-bold">{advert.collateralLocked?.toLocaleString()} {advert.token}</span><span className="text-gray-400"> collateral locked in escrow</span></span>
                  </div>
                )}
                <div className="mt-4 flex justify-between items-center">
                  <div className="text-sm text-gray-400">
                    <div>Payment: {advert.paymentMethod}</div>
                    <div>Limits: {advert.minAmount} - {advert.maxAmount} {advert.fiatCurrency}</div>
                    <div>Release: ~{advert.avgReleaseTime} min</div>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => setShowMerchantProfile(advert.userId)} className="px-4 py-2 bg-gray-700 rounded-lg hover:bg-gray-600">👤 Profile</button>
                    <button onClick={() => handleBuy(advert)} className={`px-6 py-3 rounded-lg font-bold ${advert.side === 'BUY' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'}`}>{advert.side === 'BUY' ? 'Buy Now' : 'Sell Now'}</button>
                  </div>
                </div>
              </div>
            ))}
            {filteredAdverts.length === 0 && <div className="text-center py-12 text-gray-400">No ads found</div>}
          </div>
        )}

        {activeTab === 'create' && (
          <div className="bg-gray-800 rounded-xl p-6">
            <h3 className="text-xl font-bold mb-6">Create P2P Advertisement</h3>
            <div className="bg-yellow-900/30 border border-yellow-500/30 rounded-lg p-4 mb-6">
              <div className="flex items-start gap-3">
                <span className="text-2xl">💎</span>
                <div>
                  <div className="font-bold text-yellow-400">Become a Merchant</div>
                  <div className="text-sm text-gray-400">Provide collateral to become verified! Lower fees, higher limits, more trust.</div>
                  <div className="mt-2 flex gap-4 text-sm">
                    <span>🥉 Bronze: $250</span><span>🥈 Silver: $500</span><span>🥇 Gold: $1,000</span><span>💎 Platinum: $2,500</span><span>👑 Diamond: $5,000</span>
                  </div>
                </div>
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div><label className="block text-sm text-gray-400 mb-2">I want to</label><select className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3"><option>Buy</option><option>Sell</option></select></div>
              <div><label className="block text-sm text-gray-400 mb-2">Token</label><select className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">{TOKENS.map(t => <option key={t}>{t}</option>)}</select></div>
              <div><label className="block text-sm text-gray-400 mb-2">Fiat Currency</label><select className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">{FIAT_CURRENCIES.map(f => <option key={f}>{f}</option>)}</select></div>
              <div><label className="block text-sm text-gray-400 mb-2">Price</label><input type="number" className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3" placeholder="0.00" /></div>
              <div><label className="block text-sm text-gray-400 mb-2">Min Amount</label><input type="number" className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3" placeholder="0.00" /></div>
              <div><label className="block text-sm text-gray-400 mb-2">Max Amount</label><input type="number" className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3" placeholder="0.00" /></div>
              <div className="md:col-span-2"><label className="block text-sm text-gray-400 mb-2">Payment Methods</label><div className="grid grid-cols-3 gap-2">{PAYMENT_METHODS.map(p => <label key={p} className="flex items-center space-x-2 bg-gray-700 p-3 rounded-lg cursor-pointer hover:bg-gray-600"><input type="checkbox" /><span>{p}</span></label>)}</div></div>
            </div>
            <button className="w-full bg-blue-600 py-4 rounded-lg font-bold text-lg mt-6 hover:bg-blue-700">Create Advertisement</button>
          </div>
        )}

        {activeTab === 'merchant' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2 bg-gray-800 rounded-xl p-6">
              <h3 className="text-xl font-bold mb-6">Merchant Dashboard</h3>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
                <div className="bg-gray-700 rounded-lg p-4"><div className="text-gray-400 text-sm">Total Trades</div><div className="text-2xl font-bold">1,542</div></div>
                <div className="bg-gray-700 rounded-lg p-4"><div className="text-gray-400 text-sm">Total Volume</div><div className="text-2xl font-bold">$2.5M</div></div>
                <div className="bg-gray-700 rounded-lg p-4"><div className="text-gray-400 text-sm">Completion</div><div className="text-2xl font-bold text-green-400">99.5%</div></div>
                <div className="bg-gray-700 rounded-lg p-4"><div className="text-gray-400 text-sm">Avg Rating</div><div className="text-2xl font-bold text-yellow-400">4.9 ⭐</div></div>
              </div>
              <div className="bg-purple-900/30 border border-purple-500/30 rounded-lg p-4 mb-6">
                <div className="flex items-center justify-between">
                  <div><div className="font-bold text-purple-400">🔒 Collateral Locked</div><div className="text-sm text-gray-400">Your security deposit protects trades</div></div>
                  <div className="text-right"><div className="text-2xl font-bold">2,500 USDT</div><div className="text-sm text-gray-400">≈ $2,500</div></div>
                </div>
              </div>
              <div className="mb-6">
                <div className="flex justify-between mb-2"><span className="font-bold">Trader Level</span><span className="text-yellow-400">💎 PLATINUM</span></div>
                <div className="h-4 bg-gray-700 rounded-full overflow-hidden"><div className="h-full bg-gradient-to-r from-yellow-400 to-purple-500" style={{ width: '75%' }}></div></div>
                <div className="flex justify-between text-xs text-gray-500 mt-1"><span>Silver</span><span>$2,500/2,500</span><span>Platinum</span></div>
              </div>
            </div>
            <div className="space-y-6">
              <div className="bg-gray-800 rounded-xl p-6">
                <h4 className="font-bold mb-4">Merchant Benefits</h4>
                <div className="space-y-3">
                  <div className="flex items-center gap-2"><span className="text-green-400">✓</span><span>Lower trading fees</span></div>
                  <div className="flex items-center gap-2"><span className="text-green-400">✓</span><span>Higher limits</span></div>
                  <div className="flex items-center gap-2"><span className="text-green-400">✓</span><span>Verified badge</span></div>
                  <div className="flex items-center gap-2"><span className="text-green-400">✓</span><span>Priority support</span></div>
                </div>
              </div>
              <div className="bg-gray-800 rounded-xl p-6">
                <h4 className="font-bold mb-4">Security Stats</h4>
                <div className="space-y-3">
                  <div className="flex justify-between"><span className="text-gray-400">Security Score</span><span className="text-green-400 font-bold">99%</span></div>
                  <div className="flex justify-between"><span className="text-gray-400">Disputes Won</span><span className="font-bold">15/17</span></div>
                  <div className="flex justify-between"><span className="text-gray-400">Response Time</span><span className="font-bold">1.2 min</span></div>
                </div>
              </div>
            </div>
          </div>
        )}

        {showMerchantProfile && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-gray-800 rounded-xl p-6 max-w-2xl w-full mx-4">
              <div className="flex justify-between items-start mb-6">
                <div className="flex items-center gap-4">
                  <div className="w-16 h-16 bg-gray-700 rounded-full flex items-center justify-center text-4xl">🎯</div>
                  <div>
                    <div className="flex items-center gap-2"><h3 className="text-xl font-bold">P2PPro</h3><span className="text-blue-400">✓</span><MerchantBadge level="PLATINUM" /></div>
                    <div className="text-gray-400">Member since Jan 2023</div>
                  </div>
                </div>
                <button onClick={() => setShowMerchantProfile(null)} className="text-gray-400 hover:text-white text-2xl">×</button>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
                <div className="bg-gray-700 rounded-lg p-3 text-center"><div className="text-2xl font-bold">1,542</div><div className="text-xs text-gray-400">Trades</div></div>
                <div className="bg-gray-700 rounded-lg p-3 text-center"><div className="text-2xl font-bold">$2.5M</div><div className="text-xs text-gray-400">Volume</div></div>
                <div className="bg-gray-700 rounded-lg p-3 text-center"><div className="text-2xl font-bold text-green-400">99.5%</div><div className="text-xs text-gray-400">Completion</div></div>
                <div className="bg-gray-700 rounded-lg p-3 text-center"><div className="text-2xl font-bold text-yellow-400">4.9 ⭐</div><div className="text-xs text-gray-400">Rating</div></div>
              </div>
              <div className="bg-purple-900/30 border border-purple-500/30 rounded-lg p-4 mb-6">
                <div className="flex items-center gap-3"><span className="text-2xl">🔒</span><div><div className="font-bold">2,500 USDT Collateral</div><div className="text-sm text-gray-400">Secured by locked assets</div></div></div>
              </div>
              <div className="flex gap-3">
                <button className="flex-1 bg-green-600 py-3 rounded-lg font-bold hover:bg-green-700">Start Trade</button>
                <button onClick={() => setShowMerchantProfile(null)} className="px-6 py-3 bg-gray-700 rounded-lg hover:bg-gray-600">Close</button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
