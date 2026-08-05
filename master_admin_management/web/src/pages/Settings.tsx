/**
 * Settings Page
 * System configuration and settings
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Settings() {
  const { theme, setTheme } = useTheme();
  const [config, setConfig] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const data = await masterAdminApi.getConfig();
      setConfig(data);
    } catch (error) {
      console.error('Failed to load config:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await masterAdminApi.updateConfig(config);
      alert('Settings saved successfully');
    } catch (error) {
      console.error('Failed to save config:', error);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Settings</h1>
      
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Appearance</h2>
        <div className="flex items-center justify-between">
          <span>Theme</span>
          <div className="flex space-x-2">
            <button
              onClick={() => setTheme('light')}
              className={`px-4 py-2 rounded ${theme === 'light' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700'}`}
            >
              Light
            </button>
            <button
              onClick={() => setTheme('dark')}
              className={`px-4 py-2 rounded ${theme === 'dark' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700'}`}
            >
              Dark
            </button>
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">System Configuration</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Session Timeout (minutes)</label>
            <input
              type="number"
              value={config?.sessionTimeout || 30}
              onChange={(e) => setConfig({...config, sessionTimeout: parseInt(e.target.value)})}
              className="w-full px-3 py-2 border rounded dark:bg-gray-700"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Max Login Attempts</label>
            <input
              type="number"
              value={config?.maxLoginAttempts || 5}
              onChange={(e) => setConfig({...config, maxLoginAttempts: parseInt(e.target.value)})}
              className="w-full px-3 py-2 border rounded dark:bg-gray-700"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">JWT Secret</label>
            <input
              type="password"
              value={config?.jwtSecret || ''}
              onChange={(e) => setConfig({...config, jwtSecret: e.target.value})}
              className="w-full px-3 py-2 border rounded dark:bg-gray-700"
            />
          </div>
        </div>
      </div>

      <button
        onClick={handleSave}
        disabled={saving}
        className="bg-blue-600 text-white px-6 py-2 rounded hover:bg-blue-700 disabled:opacity-50"
      >
        {saving ? 'Saving...' : 'Save Settings'}
      </button>
    </div>
  );
}
