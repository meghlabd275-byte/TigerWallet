'use client';

import React, { useState, useEffect } from 'react';
import { useWallet } from '../wallet';

// ================================================================================
// Types
// ================================================================================

interface HardwareWallet {
  id: string;
  type: 'ledger' | 'trezor' | 'keystone' | 'airgap' | 'coldcard' | 'bitbox02';
  name: string;
  model: string;
  connected: boolean;
  firmwareVersion: string;
  derivationPath: string;
  addresses: string[];
}

interface TransactionRequest {
  id: string;
  to: string;
  value: string;
  data: string;
  chainId: number;
  status: 'pending' | 'signed' | 'broadcast' | 'confirmed' | 'rejected';
  timestamp: number;
}

interface DeviceStatus {
  batteryLevel?: number;
  isLocked: boolean;
  isInitialized: boolean;
  hasSeed: boolean;
}

// Hardware wallet providers
const HARDWARE_WALLETS: Omit<HardwareWallet, 'id' | 'connected' | 'firmwareVersion' | 'addresses'>[] = [
  { type: 'ledger', name: 'Ledger', model: 'Nano X / S Plus' },
  { type: 'ledger', name: 'Ledger', model: 'Stax' },
  { type: 'trezor', name: 'Trezor', model: 'Model T' },
  { type: 'trezor', name: 'Trezor', model: 'Model One' },
  { type: 'keystone', name: 'Keystone', model: 'Pro' },
  { type: 'airgap', name: 'AirGap', model: 'Vault + Wallet' },
  { type: 'coldcard', name: 'ColdCard', model: 'Mk4' },
  { type: 'bitbox02', name: 'BitBox02', model: 'Multi-Signature' },
];

// ================================================================================
// API Service
// ================================================================================

const API_BASE_URL = process.env.NEXT_PUBLIC_HARDWARE_API_URL || 'http://localhost:8082/api/v1';

class HardwareWalletService {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // Device connection
  async connectDevice(deviceType: string): Promise<HardwareWallet> {
    return this.request('/devices/connect', {
      method: 'POST',
      body: JSON.stringify({ deviceType }),
    });
  }

  async disconnectDevice(deviceId: string): Promise<void> {
    await this.request(`/devices/${deviceId}/disconnect`, { method: 'POST' });
  }

  async getConnectedDevices(): Promise<HardwareWallet[]> {
    return this.request('/devices');
  }

  async getDeviceStatus(deviceId: string): Promise<DeviceStatus> {
    return this.request(`/devices/${deviceId}/status`);
  }

  // Address derivation
  async deriveAddresses(deviceId: string, paths: string[]): Promise<string[]> {
    return this.request(`/devices/${deviceId}/derive`, {
      method: 'POST',
      body: JSON.stringify({ paths }),
    });
  }

  // Signing
  async signTransaction(deviceId: string, transaction: any): Promise<{ signature: string }> {
    return this.request(`/devices/${deviceId}/sign`, {
      method: 'POST',
      body: JSON.stringify(transaction),
    });
  }

  async getPendingTransactions(deviceId: string): Promise<TransactionRequest[]> {
    return this.request(`/devices/${deviceId}/transactions`);
  }

  async broadcastTransaction(deviceId: string, txId: string): Promise<{ hash: string }> {
    return this.request(`/devices/${deviceId}/transactions/${txId}/broadcast`, { method: 'POST' });
  }

  async signMessage(deviceId: string, message: string, derivationPath: string): Promise<{ signature: string }> {
    return this.request(`/devices/${deviceId}/sign-message`, {
      method: 'POST',
      body: JSON.stringify({ message, derivationPath }),
    });
  }

  // Firmware
  async checkFirmwareUpdate(deviceId: string): Promise<{ available: boolean; version?: string }> {
    return this.request(`/devices/${deviceId}/firmware`);
  }

  async updateFirmware(deviceId: string): Promise<void> {
    await this.request(`/devices/${deviceId}/firmware`, { method: 'POST' });
  }
}

const hardwareService = new HardwareWalletService();

// ================================================================================
// Main Component
// ================================================================================

export default function HardwareWalletPage() {
  const { address, isConnected, chainId } = useWallet();
  
  const [devices, setDevices] = useState<HardwareWallet[]>([]);
  const [selectedDevice, setSelectedDevice] = useState<HardwareWallet | null>(null);
  const [deviceStatus, setDeviceStatus] = useState<DeviceStatus | null>(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // Transaction signing
  const [pendingTransactions, setPendingTransactions] = useState<TransactionRequest[]>([]);
  const [signingInProgress, setSigningInProgress] = useState<string | null>(null);
  
  // Derivation paths
  const [derivationPath, setDerivationPath] = useState("m/44'/60'/0'/0/0");
  const [derivedAddresses, setDerivedAddresses] = useState<string[]>([]);

  useEffect(() => {
    loadConnectedDevices();
  }, []);

  const loadConnectedDevices = async () => {
    try {
      const deviceList = await hardwareService.getConnectedDevices();
      setDevices(deviceList);
    } catch (err) {
      console.error('Failed to load devices:', err);
    }
  };

  const loadPendingTransactions = async (deviceId: string) => {
    try {
      const txs = await hardwareService.getPendingTransactions(deviceId);
      setPendingTransactions(txs);
    } catch (err) {
      console.error('Failed to load pending transactions:', err);
      setPendingTransactions([]);
    }
  };

  const connectDevice = async (deviceType: string) => {
    setIsConnecting(true);
    setError(null);
    
    try {
      const device = await hardwareService.connectDevice(deviceType);
      setDevices(prev => [...prev, device]);
      setSelectedDevice(device);
      setSuccess(`Connected to ${device.name} ${device.model}!`);
      
      // Load device status
      const status = await hardwareService.getDeviceStatus(device.id);
      setDeviceStatus(status);

      // Load pending transactions awaiting signature
      loadPendingTransactions(device.id);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsConnecting(false);
    }
  };

  const disconnectDevice = async (deviceId: string) => {
    setIsLoading(true);
    
    try {
      await hardwareService.disconnectDevice(deviceId);
      setDevices(prev => prev.filter(d => d.id !== deviceId));
      if (selectedDevice?.id === deviceId) {
        setSelectedDevice(null);
        setDeviceStatus(null);
      }
      setSuccess('Device disconnected');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const deriveAddresses = async () => {
    if (!selectedDevice) return;
    
    setIsLoading(true);
    try {
      // Generate multiple derivation paths
      const paths = [];
      for (let i = 0; i < 5; i++) {
        const path = derivationPath.replace(/0\/0$/, `0/${i}`);
        paths.push(path);
      }
      
      const addresses = await hardwareService.deriveAddresses(selectedDevice.id, paths);
      setDerivedAddresses(addresses);
      setSuccess('Addresses derived successfully');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const signTransaction = async (txId: string) => {
    if (!selectedDevice) return;
    
    const tx = pendingTransactions.find(t => t.id === txId);
    if (!tx) {
      setError('Transaction not found');
      return;
    }
    
    setSigningInProgress(txId);
    setError(null);
    
    try {
      // Sign the actual pending transaction using its real fields.
      const result = await hardwareService.signTransaction(selectedDevice.id, {
        to: tx.to,
        value: tx.value,
        data: tx.data,
        chainId: tx.chainId,
      });
      
      setPendingTransactions(prev => prev.map(t => 
        t.id === txId ? { ...t, status: 'signed' as const } : t
      ));
      
      setSuccess(`Transaction signed! Signature: ${result.signature.slice(0, 20)}...`);
    } catch (err: any) {
      setError(err.message);
      setPendingTransactions(prev => prev.map(t => 
        t.id === txId ? { ...t, status: 'rejected' as const } : t
      ));
    } finally {
      setSigningInProgress(null);
    }
  };

  const signMessage = async (message: string) => {
    if (!selectedDevice) return;
    
    setIsLoading(true);
    setError(null);
    
    try {
      const result = await hardwareService.signMessage(
        selectedDevice.id,
        message,
        derivationPath
      );
      
      setSuccess(`Message signed! Signature: ${result.signature.slice(0, 20)}...`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const getDeviceIcon = (type: string): string => {
    switch (type) {
      case 'ledger': return '📊';
      case 'trezor': return '🔐';
      case 'keystone': return '🏭';
      case 'airgap': return '✈️';
      case 'coldcard': return '❄️';
      case 'bitbox02': return '📦';
      default: return '💳';
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'confirmed': return 'bg-green-500/20 text-green-400';
      case 'signed': return 'bg-blue-500/20 text-blue-400';
      case 'broadcast': return 'bg-yellow-500/20 text-yellow-400';
      case 'pending': return 'bg-yellow-500/20 text-yellow-400';
      case 'rejected': return 'bg-red-500/20 text-red-400';
      default: return 'bg-slate-500/20 text-slate-400';
    }
  };

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">Hardware Wallet</h1>
          <p className="text-slate-400">Connect and manage hardware wallets</p>
        </div>

        {/* Connected Devices */}
        {devices.length > 0 && (
          <div className="bg-slate-800/50 rounded-xl p-4 mb-6 border border-slate-700">
            <div className="flex items-center gap-2 mb-4">
              <span className="w-3 h-3 bg-green-500 rounded-full animate-pulse"></span>
              <span className="text-white font-medium">{devices.length} device(s) connected</span>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {devices.map(device => (
                <div 
                  key={device.id}
                  onClick={() => {
                    setSelectedDevice(device);
                    hardwareService.getDeviceStatus(device.id).then(setDeviceStatus);
                  }}
                  className={`p-4 rounded-xl border cursor-pointer transition-all ${
                    selectedDevice?.id === device.id 
                      ? 'border-blue-500 bg-blue-500/10' 
                      : 'border-slate-600 bg-slate-700/30 hover:border-slate-500'
                  }`}
                >
                  <div className="flex items-center gap-3 mb-3">
                    <span className="text-3xl">{getDeviceIcon(device.type)}</span>
                    <div>
                      <p className="text-white font-medium">{device.name}</p>
                      <p className="text-slate-400 text-sm">{device.model}</p>
                    </div>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-green-400 text-sm">Connected</span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        disconnectDevice(device.id);
                      }}
                      className="text-red-400 hover:text-red-300 text-sm"
                    >
                      Disconnect
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Connect New Device */}
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Connect Hardware Wallet</h2>
            
            <div className="grid grid-cols-2 gap-4">
              {HARDWARE_WALLETS.map(wallet => (
                <button
                  key={`${wallet.type}-${wallet.model}`}
                  onClick={() => connectDevice(wallet.type)}
                  disabled={isConnecting}
                  className="flex items-center gap-3 p-4 bg-slate-700/50 hover:bg-slate-700 rounded-xl transition-colors text-left"
                >
                  <span className="text-2xl">{getDeviceIcon(wallet.type)}</span>
                  <div>
                    <p className="text-white font-medium">{wallet.name}</p>
                    <p className="text-slate-400 text-sm">{wallet.model}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Selected Device Info */}
          {selectedDevice && (
            <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
              <h2 className="text-xl font-semibold text-white mb-6">Device Details</h2>
              
              <div className="space-y-4">
                <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                  <span className="text-slate-400">Device ID</span>
                  <span className="text-white font-mono text-sm">{selectedDevice.id.slice(0, 8)}...</span>
                </div>
                <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                  <span className="text-slate-400">Model</span>
                  <span className="text-white">{selectedDevice.model}</span>
                </div>
                <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                  <span className="text-slate-400">Firmware</span>
                  <span className="text-white">{selectedDevice.firmwareVersion || 'Unknown'}</span>
                </div>
                <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                  <span className="text-slate-400">Status</span>
                  <span className="text-green-400">Ready</span>
                </div>
                
                {deviceStatus && (
                  <>
                    <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                      <span className="text-slate-400">Locked</span>
                      <span className={deviceStatus.isLocked ? 'text-yellow-400' : 'text-green-400'}>
                        {deviceStatus.isLocked ? 'Yes' : 'No'}
                      </span>
                    </div>
                    <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                      <span className="text-slate-400">Initialized</span>
                      <span className={deviceStatus.isInitialized ? 'text-green-400' : 'text-red-400'}>
                        {deviceStatus.isInitialized ? 'Yes' : 'No'}
                      </span>
                    </div>
                    {deviceStatus.batteryLevel !== undefined && (
                      <div className="flex justify-between p-3 bg-slate-700/50 rounded-lg">
                        <span className="text-slate-400">Battery</span>
                        <span className="text-white">{deviceStatus.batteryLevel}%</span>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Address Derivation */}
        {selectedDevice && (
          <div className="mt-6 bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Derive Addresses</h2>
            
            <div className="flex gap-4 mb-4">
              <input
                type="text"
                value={derivationPath}
                onChange={(e) => setDerivationPath(e.target.value)}
                placeholder="Derivation path (e.g., m/44'/60'/0'/0/0)"
                className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white font-mono"
              />
              <button
                onClick={deriveAddresses}
                disabled={isLoading}
                className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium"
              >
                {isLoading ? 'Deriving...' : 'Derive'}
              </button>
            </div>

            {derivedAddresses.length > 0 && (
              <div className="space-y-2 mt-4">
                <h3 className="text-white font-medium mb-3">Derived Addresses</h3>
                {derivedAddresses.map((addr, i) => (
                  <div key={i} className="flex items-center justify-between p-3 bg-slate-700/30 rounded-lg">
                    <span className="text-slate-400 text-sm">m/44'/60'/0'/0/{i}</span>
                    <span className="text-white font-mono text-sm">{addr.slice(0, 6)}...{addr.slice(-4)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Transaction Signing */}
        {selectedDevice && (
          <div className="mt-6 bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Sign Transaction</h2>
            
            {pendingTransactions.length > 0 ? (
              <div className="space-y-3">
                {pendingTransactions.map(tx => (
                  <div key={tx.id} className="flex items-center justify-between p-4 bg-slate-700/30 rounded-xl">
                    <div>
                      <p className="text-white font-medium">Transaction</p>
                      <p className="text-slate-400 text-sm">To: {tx.to.slice(0, 6)}...{tx.to.slice(-4)}</p>
                      <p className="text-slate-400 text-sm">Value: {tx.value} ETH</p>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusBadge(tx.status)}`}>
                        {tx.status}
                      </span>
                      {tx.status === 'pending' && (
                        <button
                          onClick={() => signTransaction(tx.id)}
                          disabled={signingInProgress !== null}
                          className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-4 py-2 rounded-lg text-sm"
                        >
                          {signingInProgress === tx.id ? 'Signing...' : 'Sign'}
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <p className="text-slate-400 mb-4">No pending transactions</p>
                <button
                  onClick={() => selectedDevice && loadPendingTransactions(selectedDevice.id)}
                  className="bg-slate-600 hover:bg-slate-500 text-white px-6 py-3 rounded-xl font-medium"
                >
                  Refresh Transactions
                </button>
              </div>
            )}
          </div>
        )}

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

        {/* Supported Features */}
        <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-slate-800/50 rounded-xl p-4 border border-slate-700">
            <h3 className="text-white font-medium mb-2">Multi-Chain Support</h3>
            <p className="text-slate-400 text-sm">Ethereum, Bitcoin, Solana, and 50+ chains</p>
          </div>
          <div className="bg-slate-800/50 rounded-xl p-4 border border-slate-700">
            <h3 className="text-white font-medium mb-2">Secure Signing</h3>
            <p className="text-slate-400 text-sm">Private keys never leave your device</p>
          </div>
          <div className="bg-slate-800/50 rounded-xl p-4 border border-slate-700">
            <h3 className="text-white font-medium mb-2">Firmware Updates</h3>
            <p className="text-slate-400 text-sm">Regular security updates from manufacturers</p>
          </div>
        </div>
      </div>
    </div>
  );
}
