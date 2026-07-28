// Analytics Page - Complete Implementation

import React, { useState, useEffect } from 'react';
import './AnalyticsPage.css';

interface AnalyticsData {
  totalUsers: number;
  activeUsers: number;
  newUsers: number;
  totalVolume: number;
  dailyVolume: number;
  monthlyVolume: number;
  totalRevenue: number;
  totalTransactions: number;
  avgTransactionValue: number;
}

const AnalyticsPage: React.FC = () => {
  const [data, setData] = useState<AnalyticsData>({
    totalUsers: 125847,
    activeUsers: 45823,
    newUsers: 1234,
    totalVolume: 3847562938,
    dailyVolume: 128374956,
    monthlyVolume: 3847293847,
    totalRevenue: 8472938,
    totalTransactions: 3847291,
    avgTransactionValue: 1000,
  });

  const [timeRange, setTimeRange] = useState('24h');

  const topChains = [
    { name: 'Ethereum', volume: 152384756, percentage: 40 },
    { name: 'BNB Chain', volume: 84729384, percentage: 22 },
    { name: 'Polygon', volume: 48392756, percentage: 13 },
    { name: 'Arbitrum', volume: 29384756, percentage: 8 },
    { name: 'Optimism', volume: 19238475, percentage: 5 },
  ];

  const topTokens = [
    { name: 'USDT', volume: 84729384 },
    { name: 'USDC', volume: 64729384 },
    { name: 'ETH', volume: 48392756 },
    { name: 'BTC', volume: 39284756 },
    { name: 'BNB', volume: 29384756 },
  ];

  const userGrowth = [
    { month: 'Jan', users: 45000 },
    { month: 'Feb', users: 58000 },
    { month: 'Mar', users: 72000 },
    { month: 'Apr', users: 85000 },
    { month: 'May', users: 98000 },
    { month: 'Jun', users: 112000 },
    { month: 'Jul', users: 125847 },
  ];

  return (
    <div className="analytics-page">
      <div className="page-header">
        <div>
          <h1>Analytics</h1>
          <p>Platform analytics and reporting</p>
        </div>
        <div className="time-selector">
          <button
            className={`time-btn ${timeRange === '24h' ? 'active' : ''}`}
            onClick={() => setTimeRange('24h')}
          >
            24H
          </button>
          <button
            className={`time-btn ${timeRange === '7d' ? 'active' : ''}`}
            onClick={() => setTimeRange('7d')}
          >
            7D
          </button>
          <button
            className={`time-btn ${timeRange === '30d' ? 'active' : ''}`}
            onClick={() => setTimeRange('30d')}
          >
            30D
          </button>
          <button
            className={`time-btn ${timeRange === 'all' ? 'active' : ''}`}
            onClick={() => setTimeRange('all')}
          >
            All
          </button>
        </div>
      </div>

      {/* Overview Stats */}
      <div className="stats-grid">
        <div className="stat-card large">
          <span className="stat-label">Total Users</span>
          <span className="stat-value">{data.totalUsers.toLocaleString()}</span>
          <span className="stat-change positive">+{data.newUsers.toLocaleString()} new</span>
        </div>
        <div className="stat-card large">
          <span className="stat-label">Active Users</span>
          <span className="stat-value">{data.activeUsers.toLocaleString()}</span>
          <span className="stat-change positive">+12.5%</span>
        </div>
        <div className="stat-card large">
          <span className="stat-label">Daily Volume</span>
          <span className="stat-value">${data.dailyVolume.toLocaleString()}</span>
          <span className="stat-change positive">+8.3%</span>
        </div>
        <div className="stat-card large">
          <span className="stat-label">Total Revenue</span>
          <span className="stat-value">${data.totalRevenue.toLocaleString()}</span>
          <span className="stat-change positive">+15.2%</span>
        </div>
      </div>

      {/* Charts Grid */}
      <div className="charts-grid">
        {/* Volume by Chain */}
        <div className="chart-card">
          <h3>Volume by Chain</h3>
          <div className="bar-chart">
            {topChains.map((chain, index) => (
              <div key={index} className="bar-item">
                <div className="bar-label">
                  <span>{chain.name}</span>
                  <span>${chain.volume.toLocaleString()}</span>
                </div>
                <div className="bar-track">
                  <div
                    className="bar-fill"
                    style={{ width: `${chain.percentage}%` }}
                  ></div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Top Tokens */}
        <div className="chart-card">
          <h3>Top Tokens by Volume</h3>
          <div className="tokens-list">
            {topTokens.map((token, index) => (
              <div key={index} className="token-row">
                <span className="token-rank">#{index + 1}</span>
                <span className="token-name">{token.name}</span>
                <span className="token-volume">${token.volume.toLocaleString()}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* User Growth */}
      <div className="chart-card full-width">
        <h3>User Growth</h3>
        <div className="line-chart">
          {userGrowth.map((point, index) => (
            <div key={index} className="chart-point">
              <div
                className="point"
                style={{ height: `${(point.users / data.totalUsers) * 100}%` }}
              >
                <span className="tooltip">{point.users.toLocaleString()}</span>
              </div>
              <span className="point-label">{point.month}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Detailed Stats */}
      <div className="stats-detail-grid">
        <div className="detail-card">
          <h4>Transaction Statistics</h4>
          <div className="detail-row">
            <span>Total Transactions</span>
            <span>{data.totalTransactions.toLocaleString()}</span>
          </div>
          <div className="detail-row">
            <span>Avg Transaction Value</span>
            <span>${data.avgTransactionValue.toLocaleString()}</span>
          </div>
          <div className="detail-row">
            <span>Monthly Volume</span>
            <span>${data.monthlyVolume.toLocaleString()}</span>
          </div>
        </div>
        <div className="detail-card">
          <h4>Revenue Breakdown</h4>
          <div className="detail-row">
            <span>Swap Fees</span>
            <span>$4,235,847</span>
          </div>
          <div className="detail-row">
            <span>Withdrawal Fees</span>
            <span>$2,123,847</span>
          </div>
          <div className="detail-row">
            <span>Trading Fees</span>
            <span>$1,108,244</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AnalyticsPage;
