// TigerWallet Admin - System Status Page
// Monitor system services and metrics

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface SystemService {
  name: string;
  status: string;
  uptime: string;
  latency: string;
  lastCheck: string;
  cpu?: number;
  memory?: number;
  requestsPerSecond?: number;
  errorRate?: string;
}

interface SystemMetrics {
  totalUsers: number;
  activeUsers24h: number;
  totalTransactions: number;
  transactionVolume24h: string;
  uptime: string;
  apiLatency: string;
}

const SystemPage: React.FC = () => {
  const [services, setServices] = useState<SystemService[]>([]);
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [restarting, setRestarting] = useState<string | null>(null);

  useEffect(() => {
    loadSystemData();
    const interval = setInterval(loadSystemData, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  const loadSystemData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [statusData, metricsData] = await Promise.all([
        adminApi.getSystemStatus().catch(() => []),
        adminApi.getSystemMetrics().catch(() => null),
      ]);

      setServices(Array.isArray(statusData) ? statusData : []);
      setMetrics(metricsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load system data');
    } finally {
      setLoading(false);
    }
  };

  const handleRestartService = async (serviceName: string) => {
    if (!confirm(`Are you sure you want to restart ${serviceName}?`)) return;
    
    try {
      setRestarting(serviceName);
      await adminApi.restartService(serviceName);
      loadSystemData();
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to restart ${serviceName}`);
    } finally {
      setRestarting(null);
    }
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'running': return 'var(--color-success)';
      case 'degraded': return 'var(--color-warning)';
      case 'error': case 'stopped': return 'var(--color-error)';
      default: return 'var(--text-tertiary)';
    }
  };

  const getStatusBadgeClass = (status: string): string => {
    switch (status) {
      case 'running': return 'badge-success';
      case 'degraded': return 'badge-warning';
      case 'error': case 'stopped': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  const getProgressBarColor = (value: number): string => {
    if (value >= 90) return 'var(--color-error)';
    if (value >= 70) return 'var(--color-warning)';
    return 'var(--color-success)';
  };

  if (loading && services.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="loader"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
          System Status
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Monitor platform services and system health
        </p>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {/* System Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="stat-card">
          <div className="stat-label">Total Users</div>
          <div className="stat-value">{metrics?.totalUsers?.toLocaleString() || '0'}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Active Users (24h)</div>
          <div className="stat-value">{metrics?.activeUsers24h?.toLocaleString() || '0'}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">24h Transactions</div>
          <div className="stat-value">{metrics?.totalTransactions?.toLocaleString() || '0'}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">System Uptime</div>
          <div className="stat-value">{metrics?.uptime || '99.9%'}</div>
        </div>
      </div>

      {/* Services Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
        {services.map((service, index) => (
          <div key={index} className="card">
            <div className="card-header flex justify-between items-center">
              <div className="flex items-center gap-2">
                <span
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: getStatusColor(service.status) }}
                ></span>
                <h3 className="font-semibold" style={{ color: 'var(--text-primary)' }}>
                  {service.name}
                </h3>
              </div>
              <span className={`badge ${getStatusBadgeClass(service.status)}`}>
                {service.status}
              </span>
            </div>
            <div className="card-body">
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span style={{ color: 'var(--text-tertiary)' }}>Uptime</span>
                  <span style={{ color: 'var(--text-primary)' }}>{service.uptime}</span>
                </div>
                <div className="flex justify-between">
                  <span style={{ color: 'var(--text-tertiary)' }}>Latency</span>
                  <span style={{ color: 'var(--text-primary)' }}>{service.latency}</span>
                </div>
                <div className="flex justify-between">
                  <span style={{ color: 'var(--text-tertiary)' }}>Last Check</span>
                  <span style={{ color: 'var(--text-secondary)' }}>
                    {new Date(service.lastCheck).toLocaleTimeString()}
                  </span>
                </div>
                
                {service.cpu !== undefined && (
                  <div>
                    <div className="flex justify-between mb-1">
                      <span style={{ color: 'var(--text-tertiary)' }}>CPU</span>
                      <span style={{ color: 'var(--text-primary)' }}>{service.cpu}%</span>
                    </div>
                    <div className="w-full h-2 rounded-full" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
                      <div
                        className="h-2 rounded-full"
                        style={{
                          width: `${service.cpu}%`,
                          backgroundColor: getProgressBarColor(service.cpu),
                        }}
                      ></div>
                    </div>
                  </div>
                )}
                
                {service.memory !== undefined && (
                  <div>
                    <div className="flex justify-between mb-1">
                      <span style={{ color: 'var(--text-tertiary)' }}>Memory</span>
                      <span style={{ color: 'var(--text-primary)' }}>{service.memory}%</span>
                    </div>
                    <div className="w-full h-2 rounded-full" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
                      <div
                        className="h-2 rounded-full"
                        style={{
                          width: `${service.memory}%`,
                          backgroundColor: getProgressBarColor(service.memory),
                        }}
                      ></div>
                    </div>
                  </div>
                )}
                
                {service.requestsPerSecond !== undefined && (
                  <div className="flex justify-between">
                    <span style={{ color: 'var(--text-tertiary)' }}>RPS</span>
                    <span style={{ color: 'var(--text-primary)' }}>{service.requestsPerSecond}</span>
                  </div>
                )}
                
                {service.errorRate && (
                  <div className="flex justify-between">
                    <span style={{ color: 'var(--text-tertiary)' }}>Error Rate</span>
                    <span style={{ 
                      color: parseFloat(service.errorRate) > 1 ? 'var(--color-error)' : 'var(--text-primary)'
                    }}>
                      {service.errorRate}
                    </span>
                  </div>
                )}
              </div>
            </div>
            <div className="card-footer">
              <button
                className="btn btn-sm btn-outline w-full"
                onClick={() => handleRestartService(service.name)}
                disabled={restarting === service.name}
              >
                {restarting === service.name ? (
                  <>
                    <span className="spinner mr-2"></span>
                    Restarting...
                  </>
                ) : (
                  'Restart Service'
                )}
              </button>
            </div>
          </div>
        ))}

        {services.length === 0 && (
          <div className="col-span-full text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
            No services available
          </div>
        )}
      </div>

      {/* Last Updated */}
      <div className="text-center" style={{ color: 'var(--text-tertiary)' }}>
        Last updated: {new Date().toLocaleString()}
        <button 
          className="btn btn-sm btn-outline ml-4"
          onClick={loadSystemData}
        >
          Refresh
        </button>
      </div>
    </div>
  );
};

export default SystemPage;
