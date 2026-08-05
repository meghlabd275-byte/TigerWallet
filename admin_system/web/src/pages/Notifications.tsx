/**
 * Notifications - Admin System
 */
import React, { useEffect, useState } from 'react';
import { adminSystemApi } from '../services/api';

export default function Notifications() {
  const [notifications, setNotifications] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadNotifications(); }, []);

  const loadNotifications = async () => {
    try {
      const data = await adminSystemApi.getNotifications();
      setNotifications(data.notifications || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleMarkRead = async (id: string) => {
    try { await adminSystemApi.markNotificationRead(id); loadNotifications(); } catch (error) { console.error('Failed:', error); }
  };

  const handleMarkAllRead = async () => {
    try { await adminSystemApi.markAllNotificationsRead(); loadNotifications(); } catch (error) { console.error('Failed:', error); }
  };

  const handleDelete = async (id: string) => {
    try { await adminSystemApi.deleteNotification(id); loadNotifications(); } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Notifications</h1>
        <button onClick={handleMarkAllRead} className="text-blue-600 hover:underline">Mark All Read</button>
      </div>
      <div className="space-y-4">
        {notifications.map((n) => (
          <div key={n.id} className={`bg-white dark:bg-gray-800 p-4 rounded-lg shadow ${!n.is_read ? 'border-l-4 border-blue-500' : ''}`}>
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold">{n.title}</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{n.message}</p>
                <p className="text-xs text-gray-500 mt-2">{new Date(n.created_at).toLocaleString()}</p>
              </div>
              <div className="space-x-2">
                {!n.is_read && <button onClick={() => handleMarkRead(n.id)} className="text-blue-600 hover:underline text-sm">Mark Read</button>}
                <button onClick={() => handleDelete(n.id)} className="text-red-600 hover:underline text-sm">Delete</button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
