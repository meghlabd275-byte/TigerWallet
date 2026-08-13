'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface Notification {
  id: string;
  type: 'transaction' | 'price' | 'security' | 'system' | 'trade';
  title: string;
  message: string;
  timestamp: number;
  read: boolean;
  data?: Record<string, string>;
}

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const NOTIFICATION_TYPES = [
  { id: 'all', name: 'All', icon: '📬' },
  { id: 'transaction', name: 'Transactions', icon: '💸' },
  { id: 'price', name: 'Price Alerts', icon: '📈' },
  { id: 'trade', name: 'Trading', icon: '🔄' },
  { id: 'security', name: 'Security', icon: '🔒' },
  { id: 'system', name: 'System', icon: '⚙️' },
];

export default function NotificationsCenter() {
  const { isDark } = useTheme();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
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

  const getUserId = (): string => {
    if (typeof window === 'undefined') return 'anonymous';
    try {
      const token = localStorage.getItem('tigerwallet-token');
      if (!token) return 'anonymous';
      const payload = JSON.parse(atob(token.split('.')[1]));
      return payload.user_id || payload.sub || payload.email || 'anonymous';
    } catch {
      return 'anonymous';
    }
  };

  const loadNotifications = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const userId = getUserId();
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`${API_BASE_URL}/api/v1/notifications/users/${encodeURIComponent(userId)}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error(`Failed to load notifications (${res.status})`);
      const data = await res.json();
      const notifs: Notification[] = (data.notifications || []).map((n: Record<string, unknown>) => ({
        id: String(n.id),
        type: (n.type as Notification['type']) || 'system',
        title: String(n.title || ''),
        message: String(n.body || n.message || ''),
        timestamp: n.created_at ? new Date(n.created_at as string).getTime() : Date.now(),
        read: Boolean(n.read),
        data: n.data as Record<string, string> | undefined,
      }));
      setNotifications(notifs);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load notifications');
      setNotifications([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadNotifications();
  }, [loadNotifications]);

  const filteredNotifications = filter === 'all'
    ? notifications
    : notifications.filter(n => n.type === filter);

  const unreadCount = notifications.filter(n => !n.read).length;

  const handleMarkAsRead = async (id: string) => {
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`${API_BASE_URL}/api/v1/notifications/${id}/read`, {
        method: 'PUT',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error('Failed to mark as read');
      setNotifications(prev => prev.map(n => n.id === id ? { ...n, read: true } : n));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to mark as read');
    }
  };

  const handleMarkAllAsRead = async () => {
    try {
      const userId = getUserId();
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`${API_BASE_URL}/api/v1/notifications/users/${encodeURIComponent(userId)}/read`, {
        method: 'PUT',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error('Failed to mark all as read');
      setNotifications(prev => prev.map(n => ({ ...n, read: true })));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to mark all as read');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`${API_BASE_URL}/api/v1/notifications/${id}`, {
        method: 'DELETE',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error('Failed to delete notification');
      setNotifications(prev => prev.filter(n => n.id !== id));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete notification');
    }
  };

  const handleClearAll = async () => {
    try {
      const userId = getUserId();
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`${API_BASE_URL}/api/v1/notifications/users/${encodeURIComponent(userId)}/clear`, {
        method: 'PUT',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error('Failed to clear notifications');
      setNotifications([]);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to clear notifications');
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'transaction': return isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800';
      case 'price': return isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800';
      case 'trade': return isDark ? 'bg-purple-900 text-purple-200' : 'bg-purple-100 text-purple-800';
      case 'security': return isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800';
      case 'system': return isDark ? 'bg-slate-700 text-slate-200' : 'bg-slate-100 text-slate-800';
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
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-slate-50' : 'bg-slate-50 text-slate-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'} border-b`}>
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
              <a href="/wallet" className={`${isDark ? 'text-slate-400' : 'text-slate-600'} hover:text-orange-500`}>Wallet</a>
            </nav>
          </div>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {loading && <div className={`mb-6 rounded-lg border px-4 py-3 ${isDark ? 'border-blue-700 bg-blue-900/30 text-blue-200' : 'border-blue-400 bg-blue-50 text-blue-900'}`}>Loading notifications…</div>}
        {error && <div className={`mb-6 rounded-lg border px-4 py-3 ${isDark ? 'border-red-700 bg-red-900/30 text-red-200' : 'border-red-400 bg-red-50 text-red-900'}`}>{error}</div>}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar */}
          <div className="lg:col-span-1">
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4 shadow-sm`}>
              <h3 className="font-semibold mb-4">Filter</h3>
              <div className="space-y-1">
                {NOTIFICATION_TYPES.map((type) => (
                  <button
                    key={type.id}
                    onClick={() => setFilter(type.id)}
                    className={`w-full text-left px-3 py-2 rounded-lg transition-colors ${
                      filter === type.id
                        ? (isDark ? 'bg-orange-900 text-orange-200' : 'bg-orange-100 text-orange-700')
                        : (isDark ? 'hover:bg-slate-700' : 'hover:bg-slate-100')
                    }`}
                  >
                    <span className="mr-2">{type.icon}</span>
                    {type.name}
                  </button>
                ))}
              </div>

              <hr className={`my-4 ${isDark ? 'border-slate-700' : 'border-slate-200'}`} />

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
              <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>
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
                <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-12 text-center`}>
                  <div className="text-6xl mb-4">🔔</div>
                  <h3 className="text-xl font-semibold mb-2">No Notifications</h3>
                  <p className={isDark ? 'text-slate-400' : 'text-slate-500'}>
                    You're all caught up!
                  </p>
                </div>
              ) : (
                filteredNotifications.map((notification) => (
                  <div
                    key={notification.id}
                    className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4 shadow-sm transition-colors ${
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
                          <p className={`text-sm mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                            {notification.message}
                          </p>
                          {notification.data && (
                            <div className="mt-2 flex flex-wrap gap-2">
                              {Object.entries(notification.data).map(([key, value]) => (
                                <span key={key} className={`text-xs px-2 py-1 rounded ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
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
                          className={isDark ? 'text-slate-400 hover:text-slate-300' : 'text-slate-400 hover:text-slate-600'}
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
