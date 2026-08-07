'use client';

import React, { useState } from 'react';

interface Notification {
  id: string;
  type: 'transaction' | 'price' | 'security' | 'system' | 'trade';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  data?: Record<string, string>;
}

const NOTIFICATION_TYPES = [
  { id: 'all', name: 'All', icon: '📬' },
  { id: 'transaction', name: 'Transactions', icon: '💸' },
  { id: 'price', name: 'Price Alerts', icon: '📈' },
  { id: 'trade', name: 'Trading', icon: '🔄' },
  { id: 'security', name: 'Security', icon: '🔒' },
  { id: 'system', name: 'System', icon: '⚙️' },
];

export default function NotificationsCenter() {
  const [notifications] = useState<Notification[]>([]);
  const [filter, setFilter] = useState('all');
  const [settings, setSettings] = useState({
    transaction: true,
    price: true,
    trade: true,
    security: true,
    system: true,
    push: true,
    email: false,
  });

  const filteredNotifications = filter === 'all' 
    ? notifications 
    : notifications.filter(n => n.type === filter);

  const unreadCount = notifications.filter(n => !n.read).length;

  const handleMarkAsRead = (_id: string) => {
    console.error('Notification updates are unavailable until an authenticated notification API is configured.');
  };

  const handleMarkAllAsRead = () => {
    console.error('Notification updates are unavailable until an authenticated notification API is configured.');
  };

  const handleDelete = (_id: string) => {
    console.error('Notification deletion is unavailable until an authenticated notification API is configured.');
  };

  const handleClearAll = () => {
    console.error('Notification clearing is unavailable until an authenticated notification API is configured.');
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'transaction': return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'price': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'trade': return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200';
      case 'security': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'system': return 'bg-slate-100 text-slate-800 dark:bg-slate-700 dark:text-slate-200';
      default: return 'bg-slate-100 text-slate-800';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'transaction': return '💸';
      case 'price': return '📈';
      case 'trade': return '🔄';
      case 'security': return '🔒';
      case 'system': return '⚙️';
      default: return '📢';
    }
  };

  const formatTime = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      {/* Header */}
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Notifications</h1>
              {unreadCount > 0 && (
                <span className="bg-orange-500 text-white text-xs px-2 py-1 rounded-full">
                  {unreadCount} new
                </span>
              )}
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Wallet</a>
            </nav>
          </div>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-6 rounded-lg border border-amber-400 bg-amber-50 px-4 py-3 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200">Live notifications are unavailable until an authenticated notification service is configured.</div>
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar */}
          <div className="lg:col-span-1">
            <div className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm">
              <h3 className="font-semibold mb-4">Filter</h3>
              <div className="space-y-1">
                {NOTIFICATION_TYPES.map((type) => (
                  <button
                    key={type.id}
                    onClick={() => setFilter(type.id)}
                    className={`w-full text-left px-3 py-2 rounded-lg transition-colors ${
                      filter === type.id
                        ? 'bg-orange-100 dark:bg-orange-900 text-orange-700 dark:text-orange-200'
                        : 'hover:bg-slate-100 dark:hover:bg-slate-700'
                    }`}
                  >
                    <span className="mr-2">{type.icon}</span>
                    {type.name}
                  </button>
                ))}
              </div>

              <hr className="my-4 border-slate-200 dark:border-slate-700" />

              <h3 className="font-semibold mb-4">Settings</h3>
              <div className="space-y-3">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.push}
                    onChange={(e) => setSettings({ ...settings, push: e.target.checked })}
                    className="w-4 h-4 rounded"
                  />
                  <span className="text-sm">Push Notifications</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={settings.email}
                    onChange={(e) => setSettings({ ...settings, email: e.target.checked })}
                    className="w-4 h-4 rounded"
                  />
                  <span className="text-sm">Email Notifications</span>
                </label>
              </div>
            </div>
          </div>

          {/* Main Content */}
          <div className="lg:col-span-3">
            {/* Actions */}
            <div className="flex items-center justify-between mb-4">
              <span className="text-slate-500 dark:text-slate-400">
                {filteredNotifications.length} notifications
              </span>
              <div className="flex gap-2">
                <button
                  onClick={handleMarkAllAsRead}
                  className="text-sm text-orange-500 hover:text-orange-400"
                >
                  Mark all as read
                </button>
                <button
                  onClick={handleClearAll}
                  className="text-sm text-red-500 hover:text-red-400"
                >
                  Clear all
                </button>
              </div>
            </div>

            {/* Notification List */}
            <div className="space-y-3">
              {filteredNotifications.length === 0 ? (
                <div className="bg-white dark:bg-slate-800 rounded-lg p-12 text-center">
                  <div className="text-6xl mb-4">🔔</div>
                  <h3 className="text-xl font-semibold mb-2">No Notifications</h3>
                  <p className="text-slate-500 dark:text-slate-400">
                    You're all caught up!
                  </p>
                </div>
              ) : (
                filteredNotifications.map((notification) => (
                  <div
                    key={notification.id}
                    className={`bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm transition-colors ${
                      !notification.read ? 'border-l-4 border-orange-500' : ''
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex items-start gap-4">
                        <div className="text-2xl">{getTypeIcon(notification.type)}</div>
                        <div className="flex-1">
                          <div className="flex items-center gap-2">
                            <h4 className="font-semibold">{notification.title}</h4>
                            <span className={`px-2 py-0.5 rounded text-xs ${getTypeColor(notification.type)}`}>
                              {notification.type}
                            </span>
                          </div>
                          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                            {notification.message}
                          </p>
                          {notification.data && (
                            <div className="mt-2 flex flex-wrap gap-2">
                              {Object.entries(notification.data).map(([key, value]) => (
                                <span key={key} className="text-xs bg-slate-100 dark:bg-slate-700 px-2 py-1 rounded">
                                  {key}: {value}
                                </span>
                              ))}
                            </div>
                          )}
                          <div className="text-xs text-slate-400 mt-2">
                            {formatTime(notification.timestamp)}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {!notification.read && (
                          <span className="w-2 h-2 bg-orange-500 rounded-full" />
                        )}
                        <button
                          onClick={() => handleMarkAsRead(notification.id)}
                          className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                          title="Mark as read"
                        >
                          ✓
                        </button>
                        <button
                          onClick={() => handleDelete(notification.id)}
                          className="text-slate-400 hover:text-red-500"
                          title="Delete"
                        >
                          ✕
                        </button>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
