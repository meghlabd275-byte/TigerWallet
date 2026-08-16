/**
 * Settings Page - White Label Admin
 */

import React, { useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Settings() {
  const { theme, isDark, setTheme } = useTheme();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [msg, setMsg] = useState('');
  const [err, setErr] = useState('');

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const inputCls = `w-full px-3 py-2 rounded border ${isDark ? 'border-gray-700 bg-gray-700 text-white' : 'border-gray-300 bg-white text-gray-900'}`;

  const handleChangePassword = async () => {
    setMsg(''); setErr('');
    try {
      await whiteLabelAdminApi.changePassword(oldPassword, newPassword);
      setMsg('Password updated.');
      setOldPassword(''); setNewPassword('');
    } catch (e: any) { setErr(e.message || 'Failed to change password'); }
  };

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-6 ${cardText}`}>Settings</h1>

      <div className={`${cardBg} rounded-lg shadow p-6 mb-6`}>
        <h2 className={`text-lg font-semibold mb-4 ${cardText}`}>Appearance</h2>
        <div className="flex items-center justify-between">
          <span className={cardText}>Theme</span>
          <div className="flex space-x-2">
            <button onClick={() => setTheme('light')} className={`px-4 py-2 rounded ${theme === 'light' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800')}`}>Light</button>
            <button onClick={() => setTheme('dark')} className={`px-4 py-2 rounded ${theme === 'dark' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800')}`}>Dark</button>
          </div>
        </div>
      </div>

      <div className={`${cardBg} rounded-lg shadow p-6`}>
        <h2 className={`text-lg font-semibold mb-4 ${cardText}`}>Change Password</h2>
        {msg && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-green-900/50 text-green-200' : 'bg-green-50 text-green-700'}`}>{msg}</div>}
        {err && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{err}</div>}
        <div className="space-y-3 max-w-md">
          <input type="password" placeholder="Current password" value={oldPassword} onChange={e => setOldPassword(e.target.value)} className={inputCls} />
          <input type="password" placeholder="New password" value={newPassword} onChange={e => setNewPassword(e.target.value)} className={inputCls} />
          <button onClick={handleChangePassword} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Update Password</button>
        </div>
      </div>
    </div>
  );
}
