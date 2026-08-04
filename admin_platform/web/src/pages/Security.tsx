import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { analyticsService } from '../services/api';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line } from 'recharts';

interface SecurityData {
  failedLogins: number;
  activeSessions: number;
  suspiciousIPs: number;
  securityEvents: number;
  blockedIPs: number;
  twoFactorEnabled: number;
  securityScore: number;
}

export default function SecurityDashboard() {
  const { isDark } = useTheme();
  const [data, setData] = useState<SecurityData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadSecurityData();
  }, []);

  const loadSecurityData = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await analyticsService.getSecurityDashboard();
      setData(response.data);
    } catch (err: any) {
      setError(err.message || 'Failed to load security data');
    } finally {
      setLoading(false);
    }
  };

  const loginAttemptsData = [
    { name: '00:00', attempts: 12 },
    { name: '04:00', attempts: 8 },
    { name: '08:00', attempts: 45 },
    { name: '12:00', attempts: 32 },
    { name: '16:00', attempts: 28 },
    { name: '20:00', attempts: 15 },
    { name: '24:00', attempts: 10 },
  ];

  const securityEventsData = [
    { name: 'Mon', events: 5 },
    { name: 'Tue', events: 3 },
    { name: 'Wed', events: 8 },
    { name: 'Thu', events: 6 },
    { name: 'Fri', events: 4 },
    { name: 'Sat', events: 2 },
    { name: 'Sun', events: 1 },
  ];

  if (loading) {
    return (
      <div className={`flex items-center justify-center h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        <div className="text-blue-600 text-xl">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`flex items-center justify-center h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        <div className="text-red-600 text-xl">{error}</div>
      </div>
    );
  }

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Security Dashboard
          </h1>
          <button
            onClick={loadSecurityData}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Refresh
          </button>
        </div>

        {/* Security Score */}
        <div className={`mb-6 p-6 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="flex items-center justify-between">
            <div>
              <h2 className={`text-lg font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Security Score
              </h2>
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                Overall platform security rating
              </p>
            </div>
            <div className="text-center">
              <div className={`text-5xl font-bold ${
                data?.securityScore && data.securityScore >= 90 ? 'text-green-500' : 
                data?.securityScore && data.securityScore >= 70 ? 'text-yellow-500' : 'text-red-500'
              }`}>
                {data?.securityScore || 0}%
              </div>
              <div className={`text-sm mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                {data?.securityScore && data.securityScore >= 90 ? 'Excellent' : 
                 data?.securityScore && data.securityScore >= 70 ? 'Good' : 'Needs Attention'}
              </div>
            </div>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Failed Logins (24h)</p>
            <p className="text-2xl font-bold text-red-500">{data?.failedLogins || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Active Sessions</p>
            <p className="text-2xl font-bold text-blue-500">{data?.activeSessions || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Suspicious IPs</p>
            <p className="text-2xl font-bold text-orange-500">{data?.suspiciousIPs || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Security Events</p>
            <p className="text-2xl font-bold text-yellow-500">{data?.securityEvents || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Blocked IPs</p>
            <p className="text-2xl font-bold text-red-500">{data?.blockedIPs || 0}</p>
          </div>
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>2FA Enabled</p>
            <p className="text-2xl font-bold text-green-500">{data?.twoFactorEnabled || 0}</p>
          </div>
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Login Attempts */}
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Failed Login Attempts (24h)
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={loginAttemptsData}>
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#374151' : '#E5E7EB'} />
                <XAxis dataKey="name" stroke={isDark ? '#9CA3AF' : '#6B7280'} />
                <YAxis stroke={isDark ? '#9CA3AF' : '#6B7280'} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: isDark ? '#1F2937' : '#fff',
                    border: isDark ? '1px solid #374151' : '1px solid #E5E7EB',
                    borderRadius: '8px',
                  }}
                />
                <Bar dataKey="attempts" fill="#EF4444" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Security Events */}
          <div className={`p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Security Events (7 days)
            </h3>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={securityEventsData}>
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#374151' : '#E5E7EB'} />
                <XAxis dataKey="name" stroke={isDark ? '#9CA3AF' : '#6B7280'} />
                <YAxis stroke={isDark ? '#9CA3AF' : '#6B7280'} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: isDark ? '#1F2937' : '#fff',
                    border: isDark ? '1px solid #374151' : '1px solid #E5E7EB',
                    borderRadius: '8px',
                  }}
                />
                <Line type="monotone" dataKey="events" stroke="#F59E0B" strokeWidth={2} dot={{ fill: '#F59E0B' }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Security Actions */}
        <div className={`mt-6 p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Quick Actions
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <button className="px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700">
              Block Suspicious IP
            </button>
            <button className="px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
              Force 2FA Setup
            </button>
            <button className="px-4 py-3 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700">
              View Audit Logs
            </button>
          </div>
        </div>

        {/* Recent Security Alerts */}
        <div className={`mt-6 p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Recent Security Alerts
          </h3>
          <div className={`text-center py-8 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
            {data?.securityEvents || 0} security events in the last 24 hours
          </div>
        </div>
      </div>
    </div>
  );
}
