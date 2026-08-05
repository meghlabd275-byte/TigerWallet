/**
 * Settings Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Settings() {
  const { theme, setTheme } = useTheme();
  const [config, setConfig] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadConfig(); }, []);

  const loadConfig = async () => {
    try {
      const data = await whiteLabelAdminApi.getConfig();
      setConfig(data);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
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
            <button onClick={() => setTheme('light')} className={`px-4 py-2 rounded ${theme === 'light' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700'}`}>Light</button>
            <button onClick={() => setTheme('dark')} className={`px-4 py-2 rounded ${theme === 'dark' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700'}`}>Dark</button>
          </div>
        </div>
      </div>
    </div>
  );
}
