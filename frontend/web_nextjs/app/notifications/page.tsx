'use client';

import React, { useState } from 'react';
import { useWallet } from '../wallet';

// ================================================================================
// Types
// ================================================================================

interface NotificationPreferences {
  priceAlerts: boolean;
  transactionAlerts: boolean;
  airdropAlerts: boolean;
  gasAlerts: boolean;
  portfolioAlerts: boolean;
  marketingEmails: boolean;
  pushNotifications: boolean;
  emailNotifications: boolean;
  smsNotifications: boolean;
  minPriceChange: number;
  gasPriceThreshold: number;
}

interface Notification {
  id: string;
  type: 'price' | 'transaction' | 'airdrop' | 'gas' | 'portfolio' | 'system';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  data?: any;
}

interface PriceAlert {
  id: string;
  token: string;
  condition: 'above' | 'below';
  targetPrice: number;
  currentPrice: number;
  triggered: boolean;
}

// ================================================================================
// Main Component
// ================================================================================

export default function NotificationsPage() {
  const { address, isConnected } = useWallet();
  
  // Preferences
  const [preferences, setPreferences] = useState<NotificationPreferences>({
    priceAlerts: true,
    transactionAlerts: true,
    airdropAlerts: true,
    gasAlerts: true,
    portfolioAlerts: true,
    marketingEmails: false,
    pushNotifications: true,
    emailNotifications: true,
    smsNotifications: false,
    minPriceChange: 5,
    gasPriceThreshold: 50,
  });
  
  // Notifications
  const [notifications, setNotifications] = useState<Notification[]>([
    {
      id: '1',
      type: 'price',
      title: 'ETH Price Alert',
      message: 'Ethereum has increased by 5% in the last hour',
      timestamp: Date.now() - 3600000,
      read: false,
    },
    {
      id: '2',
      type: 'transaction',
      title: 'Transaction Confirmed',
      message: 'Your transaction of 0.5 ETH has been confirmed',
      timestamp: Date.now() - 7200000,
      read: false,
    },
    {
      id: '3',
      type: 'airdrop',
      title: 'New Airdrop Detected',
      message: 'You may be eligible for a new token airdrop',
      timestamp: Date.now() - 86400000,
      read: true,
    },
  ]);
  
  // Price alerts
  const [priceAlerts, setPriceAlerts] = useState<PriceAlert[]>([
    { id: '1', token: 'ETH', condition: 'above', targetPrice: 3500, currentPrice: 3200, triggered: false },
    { id: '2', token: 'BTC', condition: 'below', targetPrice: 60000, currentPrice: 65000, triggered: false },
  ]);
  
  // UI state
  const [activeTab, setActiveTab] = useState<'all' | 'alerts' | 'transactions'>('all');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // New alert form
  const [newAlertToken, setNewAlertToken] = useState('ETH');
  const [newAlertCondition, setNewAlertCondition] = useState<'above' | 'below'>('above');
  const [newAlertPrice, setNewAlertPrice] = useState('');

  const markAsRead = (id: string) => {
    setNotifications(prev => prev.map(n => 
      n.id === id ? { ...n, read: true } : n
    ));
  };

  const markAllAsRead = () => {
    setNotifications(prev => prev.map(n => ({ ...n, read: true })));
    setSuccess('All notifications marked as read');
  };

  const deleteNotification = (id: string) => {
    setNotifications(prev => prev.filter(n => n.id !== id));
  };

  const createPriceAlert = () => {
    if (!newAlertPrice) return;
    
    const alert: PriceAlert = {
      id: `alert_${Date.now()}`,
      token: newAlertToken,
      condition: newAlertCondition,
      targetPrice: parseFloat(newAlertPrice),
      currentPrice: newAlertToken === 'ETH' ? 3200 : newAlertToken === 'BTC' ? 65000 : 100,
      triggered: false,
    };
    
    setPriceAlerts(prev => [...prev, alert]);
    setNewAlertPrice('');
    setSuccess(`Price alert created for ${newAlertToken}`);
  };

  const deletePriceAlert = (id: string) => {
    setPriceAlerts(prev => prev.filter(a => a.id !== id));
  };

  const updatePreference = (key: keyof NotificationPreferences, value: any) => {
    setPreferences(prev => ({ ...prev, [key]: value }));
  };

  const filteredNotifications = notifications.filter(n => {
    if (activeTab === 'alerts') return ['price', 'airdrop', 'gas'].includes(n.type);
    if (activeTab === 'transactions') return n.type === 'transaction';
    return true;
  });

  const formatTime = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return new Date(timestamp).toLocaleDateString();
  };

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'price': return '📈';
      case 'transaction': return '💸';
      case 'airdrop': return '🎁';
      case 'gas': return '⛽';
      case 'portfolio': return '💼';
      default: return '🔔';
    }
  };

  const unreadCount = notifications.filter(n => !n.read).length;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">Notifications</h1>
          <p className="text-slate-400">Manage alerts and notifications</p>
        </div>

        {/* Tabs */}
        <div className="flex gap-4 mb-6">
          {(['all', 'alerts', 'transactions'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                activeTab === tab
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
              {tab === 'all' && unreadCount > 0 && (
                <span className="ml-2 bg-red-500 text-white text-xs px-2 py-0.5 rounded-full">
                  {unreadCount}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Notifications List */}
        <div className="bg-slate-800 rounded-2xl border border-slate-700 mb-6">
          <div className="flex items-center justify-between p-4 border-b border-slate-700">
            <h2 className="text-lg font-semibold text-white">Recent Notifications</h2>
            {unreadCount > 0 && (
              <button
                onClick={markAllAsRead}
                className="text-blue-400 hover:text-blue-300 text-sm"
              >
                Mark all as read
              </button>
            )}
          </div>
          
          {filteredNotifications.length === 0 ? (
            <div className="p-8 text-center">
              <span className="text-4xl mb-4 block">🔔</span>
              <p className="text-slate-400">No notifications</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-700">
              {filteredNotifications.map(notification => (
                <div 
                  key={notification.id}
                  className={`p-4 flex items-start gap-4 hover:bg-slate-700/30 cursor-pointer ${
                    !notification.read ? 'bg-slate-700/20' : ''
                  }`}
                  onClick={() => markAsRead(notification.id)}
                >
                  <span className="text-2xl">{getNotificationIcon(notification.type)}</span>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h3 className="text-white font-medium">{notification.title}</h3>
                      {!notification.read && (
                        <span className="w-2 h-2 bg-blue-500 rounded-full"></span>
                      )}
                    </div>
                    <p className="text-slate-400 text-sm">{notification.message}</p>
                    <p className="text-slate-500 text-xs mt-1">{formatTime(notification.timestamp)}</p>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      deleteNotification(notification.id);
                    }}
                    className="text-slate-500 hover:text-red-400"
                  >
                    ✕
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Price Alerts */}
        <div className="bg-slate-800 rounded-2xl border border-slate-700 mb-6">
          <div className="p-4 border-b border-slate-700">
            <h2 className="text-lg font-semibold text-white">Price Alerts</h2>
          </div>
          
          <div className="p-4">
            <div className="flex gap-4 mb-4">
              <select
                value={newAlertToken}
                onChange={(e) => setNewAlertToken(e.target.value)}
                className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white"
              >
                <option value="ETH">ETH</option>
                <option value="BTC">BTC</option>
                <option value="SOL">SOL</option>
                <option value="MATIC">MATIC</option>
              </select>
              <select
                value={newAlertCondition}
                onChange={(e) => setNewAlertCondition(e.target.value as 'above' | 'below')}
                className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white"
              >
                <option value="above">Above</option>
                <option value="below">Below</option>
              </select>
              <input
                type="number"
                value={newAlertPrice}
                onChange={(e) => setNewAlertPrice(e.target.value)}
                placeholder="Price"
                className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-2 text-white"
              />
              <button
                onClick={createPriceAlert}
                className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg font-medium"
              >
                Add Alert
              </button>
            </div>
            
            <div className="space-y-2">
              {priceAlerts.map(alert => (
                <div key={alert.id} className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
                  <div className="flex items-center gap-3">
                    <span className="text-white font-medium">{alert.token}</span>
                    <span className="text-slate-400">goes</span>
                    <span className={alert.condition === 'above' ? 'text-green-400' : 'text-red-400'}>
                      {alert.condition}
                    </span>
                    <span className="text-white font-mono">${alert.targetPrice}</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-slate-400 text-sm">
                      Current: ${alert.currentPrice}
                    </span>
                    <button
                      onClick={() => deletePriceAlert(alert.id)}
                      className="text-red-400 hover:text-red-300"
                    >
                      ✕
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Notification Preferences */}
        <div className="bg-slate-800 rounded-2xl border border-slate-700">
          <div className="p-4 border-b border-slate-700">
            <h2 className="text-lg font-semibold text-white">Notification Preferences</h2>
          </div>
          
          <div className="p-4 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              {[
                ['priceAlerts', 'Price Alerts'],
                ['transactionAlerts', 'Transaction Alerts'],
                ['airdropAlerts', 'Airdrop Alerts'],
                ['gasAlerts', 'Gas Alerts'],
                ['portfolioAlerts', 'Portfolio Alerts'],
                ['marketingEmails', 'Marketing Emails'],
              ].map(([key, label]) => (
                <label key={key} className="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={preferences[key as keyof NotificationPreferences] as boolean}
                    onChange={(e) => updatePreference(key as keyof NotificationPreferences, e.target.checked)}
                    className="w-5 h-5 rounded bg-slate-700 border-slate-600 text-blue-600 focus:ring-blue-500"
                  />
                  <span className="text-white">{label}</span>
                </label>
              ))}
            </div>
            
            <div className="border-t border-slate-700 pt-4">
              <h3 className="text-white font-medium mb-4">Delivery Methods</h3>
              <div className="grid grid-cols-3 gap-4">
                {[
                  ['pushNotifications', 'Push Notifications'],
                  ['emailNotifications', 'Email'],
                  ['smsNotifications', 'SMS'],
                ].map(([key, label]) => (
                  <label key={key} className="flex items-center gap-3 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={preferences[key as keyof NotificationPreferences] as boolean}
                      onChange={(e) => updatePreference(key as keyof NotificationPreferences, e.target.checked)}
                      className="w-5 h-5 rounded bg-slate-700 border-slate-600 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-white">{label}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Messages */}
        {error && (
          <div className="mt-6 p-4 bg-red-500/10 border border-red-500/30 rounded-xl">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {success && (
          <div className="mt-6 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
            <p className="text-green-400">{success}</p>
          </div>
        )}
      </div>
    </div>
  );
}
