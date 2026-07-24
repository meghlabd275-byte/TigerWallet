'use client';

import React, { useState, useEffect, useCallback } from 'react';

interface Order {
  id: string;
  type: 'buy' | 'sell';
  price: number;
  amount: number;
  total: number;
  filled: number;
  status: 'open' | 'partial' | 'filled' | 'cancelled';
  timestamp: number;
}

interface Market {
  symbol: string;
  base: string;
  quote: string;
  price: number;
  change24h: number;
  volume24h: number;
  high24h: number;
  low24h: number;
  bids: { price: number; amount: number }[];
  asks: { price: number; amount: number }[];
}

const MARKETS: Market[] = [
  { symbol: 'TGR/USDT', base: 'TGR', quote: 'USDT', price: 0.25, change24h: 5.2, volume24h: 1250000, high24h: 0.27, low24h: 0.23, bids: [], asks: [] },
  { symbol: 'RUSD/USDT', base: 'RUSD', quote: 'USDT', price: 1.00, change24h: 0.05, volume24h: 850000, high24h: 1.01, low24h: 0.99, bids: [], asks: [] },
  { symbol: 'ETH/USDT', base: 'ETH', quote: 'USDT', price: 3500, change24h: -2.1, volume24h: 25000000, high24h: 3600, low24h: 3450, bids: [], asks: [] },
  { symbol: 'BTC/USDT', base: 'BTC', quote: 'USDT', price: 65000, change24h: 1.8, volume24h: 45000000, high24h: 66000, low24h: 64000, bids: [], asks: [] },
  { symbol: 'SOL/USDT', base: 'SOL', quote: 'USDT', price: 145, change24h: 3.5, volume24h: 8500000, high24h: 150, low24h: 140, bids: [], asks: [] },
];

// Generate realistic order book
const generateOrderBook = (midPrice: number): { bids: { price: number; amount: number }[]; asks: { price: number; amount: number }[] } => {
  const bids = [];
  const asks = [];
  
  for (let i = 0; i < 15; i++) {
    bids.push({
      price: midPrice * (1 - 0.0001 * (i + 1) - Math.random() * 0.0002),
      amount: Math.random() * 10 + 1,
    });
    asks.push({
      price: midPrice * (1 + 0.0001 * (i + 1) + Math.random() * 0.0002),
      amount: Math.random() * 10 + 1,
    });
  }
  
  return { bids, asks };
};

export default function OrderbookPage() {
  const [selectedMarket, setSelectedMarket] = useState<Market>(MARKETS[0]);
  const [orderType, setOrderType] = useState<'limit' | 'market'>('limit');
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState('');
  const [amount, setAmount] = useState('');
  const [orders, setOrders] = useState<Order[]>([]);
  const [orderHistory, setOrderHistory] = useState<Order[]>([]);
  const [orderbook, setOrderbook] = useState<{ bids: { price: number; amount: number }[]; asks: { price: number; amount: number }[] }>({ bids: [], asks: [] });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const ob = generateOrderBook(selectedMarket.price);
    setOrderbook(ob);
    
    const interval = setInterval(() => {
      setOrderbook(generateOrderBook(selectedMarket.price * (1 + (Math.random() - 0.5) * 0.001)));
    }, 2000);
    
    return () => clearInterval(interval);
  }, [selectedMarket]);

  const handleSubmitOrder = useCallback(async () => {
    if (!price || !amount) return;
    setLoading(true);
    
    await new Promise(r => setTimeout(r, 500));
    
    const newOrder: Order = {
      id: Date.now().toString(),
      type: side,
      price: parseFloat(price),
      amount: parseFloat(amount),
      total: parseFloat(price) * parseFloat(amount),
      filled: 0,
      status: 'open',
      timestamp: Date.now(),
    };
    
    setOrders(prev => [newOrder, ...prev]);
    setPrice('');
    setAmount('');
    setLoading(false);
  }, [side, price, amount]);

  const handleCancelOrder = useCallback((orderId: string) => {
    setOrders(prev => prev.map(o => o.id === orderId ? { ...o, status: 'cancelled' } : o));
  }, []);

  const totalBidVolume = orderbook.bids.reduce((sum, b) => sum + b.price * b.amount, 0);
  const totalAskVolume = orderbook.asks.reduce((sum, a) => sum + a.price * a.amount, 0);

  return (
    <div className="min-h-screen bg-slate-900 text-white">
      <header className="bg-slate-800 border-b border-slate-700 p-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <a href="/" className="text-2xl">🐯</a>
            <h1 className="text-xl font-bold">Order Book Trading</h1>
          </div>
          <div className="flex items-center gap-4">
            <select 
              value={selectedMarket.symbol}
              onChange={(e) => setSelectedMarket(MARKETS.find(m => m.symbol === e.target.value) || MARKETS[0])}
              className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-2"
            >
              {MARKETS.map(m => (
                <option key={m.symbol} value={m.symbol}>{m.symbol}</option>
              ))}
            </select>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto p-4">
        {/* Market Stats */}
        <div className="grid grid-cols-6 gap-4 mb-4">
          <div className="bg-slate-800 rounded-lg p-4">
            <p className="text-slate-400 text-sm">Last Price</p>
            <p className="text-2xl font-bold">${selectedMarket.price.toLocaleString()}</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-4">
            <p className="text-slate-400 text-sm">24h Change</p>
            <p className={`text-2xl font-bold ${selectedMarket.change24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
              {selectedMarket.change24h >= 0 ? '+' : ''}{selectedMarket.change24h}%
            </p>
          </div>
          <div className="bg-slate-800 rounded-lg p-4">
            <p className="text-slate-400 text-sm">24h High</p>
            <p className="text-xl font-bold">${selectedMarket.high24h.toLocaleString()}</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-4">
            <p className="text-slate-400 text-sm">24h Low</p>
            <p className="text-xl font-bold">${selectedMarket.low24h.toLocaleString()}</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-4">
            <p className="text-slate-400 text-sm">24h Volume</p>
            <p className="text-xl font-bold">${(selectedMarket.volume24h / 1000000).toFixed(2)}M</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-4">
            <p className="text-slate-400 text-sm">Spread</p>
            <p className="text-xl font-bold">{((orderbook.asks[0]?.price || 0) - (orderbook.bids[0]?.price || 0)).toFixed(4)}</p>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-4">
          {/* Order Book */}
          <div className="bg-slate-800 rounded-lg p-4 col-span-2">
            <h3 className="font-bold mb-4">Order Book</h3>
            
            <div className="grid grid-cols-2 gap-4">
              {/* Bids */}
              <div>
                <div className="grid grid-cols-3 text-sm text-slate-400 mb-2">
                  <span>Price</span>
                  <span className="text-right">Amount</span>
                  <span className="text-right">Total</span>
                </div>
                <div className="space-y-0.5">
                  {orderbook.bids.slice(0, 12).map((bid, i) => {
                    const depth = (totalBidVolume - (orderbook.bids.slice(i).reduce((s, b) => s + b.price * b.amount, 0))) / totalBidVolume;
                    return (
                      <div key={i} className="grid grid-cols-3 text-sm relative">
                        <div className="absolute right-0 top-0 bottom-0 bg-green-500/20" style={{ width: `${depth * 100}%` }}></div>
                        <span className="text-green-400 relative z-10">${bid.price.toFixed(4)}</span>
                        <span className="text-right relative z-10">{bid.amount.toFixed(4)}</span>
                        <span className="text-right relative z-10">${(bid.price * bid.amount).toFixed(2)}</span>
                      </div>
                    );
                  })}
                </div>
              </div>
              
              {/* Asks */}
              <div>
                <div className="grid grid-cols-3 text-sm text-slate-400 mb-2">
                  <span>Price</span>
                  <span className="text-right">Amount</span>
                  <span className="text-right">Total</span>
                </div>
                <div className="space-y-0.5">
                  {orderbook.asks.slice(0, 12).map((ask, i) => {
                    const depth = (orderbook.asks.slice(0, i + 1).reduce((s, a) => s + a.price * a.amount, 0)) / totalAskVolume;
                    return (
                      <div key={i} className="grid grid-cols-3 text-sm relative">
                        <div className="absolute right-0 top-0 bottom-0 bg-red-500/20" style={{ width: `${depth * 100}%` }}></div>
                        <span className="text-red-400 relative z-10">${ask.price.toFixed(4)}</span>
                        <span className="text-right relative z-10">{ask.amount.toFixed(4)}</span>
                        <span className="text-right relative z-10">${(ask.price * ask.amount).toFixed(2)}</span>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>

          {/* Trade Form */}
          <div className="bg-slate-800 rounded-lg p-4">
            <h3 className="font-bold mb-4">Place Order</h3>
            
            <div className="flex mb-4 bg-slate-700 rounded-lg p-1">
              <button
                onClick={() => setSide('buy')}
                className={`flex-1 py-2 rounded-lg font-medium ${side === 'buy' ? 'bg-green-600' : ''}`}
              >
                Buy
              </button>
              <button
                onClick={() => setSide('sell')}
                className={`flex-1 py-2 rounded-lg font-medium ${side === 'sell' ? 'bg-red-600' : ''}`}
              >
                Sell
              </button>
            </div>

            <div className="flex mb-4 bg-slate-700 rounded-lg p-1">
              <button
                onClick={() => setOrderType('limit')}
                className={`flex-1 py-2 rounded-lg text-sm ${orderType === 'limit' ? 'bg-slate-600' : ''}`}
              >
                Limit
              </button>
              <button
                onClick={() => setOrderType('market')}
                className={`flex-1 py-2 rounded-lg text-sm ${orderType === 'market' ? 'bg-slate-600' : ''}`}
              >
                Market
              </button>
            </div>

            <div className="space-y-3">
              <div>
                <label className="text-sm text-slate-400">Price (USDT)</label>
                <input
                  type="number"
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                  placeholder={orderType === 'market' ? 'Market Price' : ''}
                  disabled={orderType === 'market'}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg p-3 mt-1"
                />
              </div>
              <div>
                <label className="text-sm text-slate-400">Amount ({selectedMarket.base})</label>
                <input
                  type="number"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="0.00"
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg p-3 mt-1"
                />
              </div>
              <div className="p-3 bg-slate-700 rounded-lg">
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-slate-400">Total</span>
                  <span>${(parseFloat(price || '0') * parseFloat(amount || '0')).toFixed(2)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-400">Fee (0.1%)</span>
                  <span>${((parseFloat(price || '0') * parseFloat(amount || '0')) * 0.001).toFixed(4)}</span>
                </div>
              </div>
              <button
                onClick={handleSubmitOrder}
                disabled={loading || !amount || (orderType === 'limit' && !price)}
                className={`w-full py-3 rounded-lg font-bold ${side === 'buy' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'} disabled:opacity-50`}
              >
                {side === 'buy' ? 'Buy' : 'Sell'} {selectedMarket.base}
              </button>
            </div>
          </div>
        </div>

        {/* Open Orders */}
        <div className="mt-4 bg-slate-800 rounded-lg p-4">
          <h3 className="font-bold mb-4">Open Orders</h3>
          {orders.filter(o => o.status === 'open').length === 0 ? (
            <p className="text-slate-400 text-center py-8">No open orders</p>
          ) : (
            <div className="space-y-2">
              {orders.filter(o => o.status === 'open').map(order => (
                <div key={order.id} className="flex items-center justify-between p-3 bg-slate-700 rounded-lg">
                  <div className="flex items-center gap-4">
                    <span className={`font-bold ${order.type === 'buy' ? 'text-green-400' : 'text-red-400'}`}>
                      {order.type.toUpperCase()}
                    </span>
                    <span>{order.amount} {selectedMarket.base}</span>
                    <span>@ ${order.price.toFixed(4)}</span>
                    <span className="text-slate-400">= ${order.total.toFixed(2)}</span>
                  </div>
                  <button
                    onClick={() => handleCancelOrder(order.id)}
                    className="px-4 py-1 bg-slate-600 rounded hover:bg-slate-500"
                  >
                    Cancel
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
