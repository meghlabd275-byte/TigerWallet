// TigerWallet Admin - Fee Configuration Page
// Configure platform fees

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface FeeConfig {
  tradingFee: string;
  withdrawalFee: string;
  withdrawalFeeMin: string;
  depositFee: string;
  networkFee: string;
  makerFee: string;
  takerFee: string;
}

const FeesPage: React.FC = () => {
  const [fees, setFees] = useState<FeeConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [editedFees, setEditedFees] = useState<FeeConfig | null>(null);

  useEffect(() => {
    loadFees();
  }, []);

  const loadFees = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const data = await adminApi.getFeeConfig();
      setFees(data);
      setEditedFees(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load fees');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!editedFees) return;
    
    try {
      setSaving(true);
      setError(null);
      setSuccess(null);
      
      await adminApi.updateFeeConfig(editedFees);
      setFees(editedFees);
      setSuccess('Fees updated successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update fees');
    } finally {
      setSaving(false);
    }
  };

  const handleInputChange = (field: keyof FeeConfig, value: string) => {
    if (!editedFees) return;
    setEditedFees({ ...editedFees, [field]: value });
  };

  if (loading) {
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
          Fee Configuration
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Configure platform trading, withdrawal, and deposit fees
        </p>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {success && (
        <div className="alert alert-success mb-4">
          {success}
        </div>
      )}

      {/* Fee Configuration */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Trading Fees */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Trading Fees</h3>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              <div className="form-group">
                <label className="form-label">Trading Fee (%)</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.tradingFee || ''}
                  onChange={(e) => handleInputChange('tradingFee', e.target.value)}
                  placeholder="0.3"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Default trading fee percentage
                </p>
              </div>
              
              <div className="form-group">
                <label className="form-label">Maker Fee (%)</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.makerFee || ''}
                  onChange={(e) => handleInputChange('makerFee', e.target.value)}
                  placeholder="0.1"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Fee for limit order makers
                </p>
              </div>
              
              <div className="form-group">
                <label className="form-label">Taker Fee (%)</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.takerFee || ''}
                  onChange={(e) => handleInputChange('takerFee', e.target.value)}
                  placeholder="0.2"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Fee for market order takers
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Withdrawal Fees */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Withdrawal Fees</h3>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              <div className="form-group">
                <label className="form-label">Withdrawal Fee</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.withdrawalFee || ''}
                  onChange={(e) => handleInputChange('withdrawalFee', e.target.value)}
                  placeholder="0.0001"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Base withdrawal fee
                </p>
              </div>
              
              <div className="form-group">
                <label className="form-label">Minimum Withdrawal Fee</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.withdrawalFeeMin || ''}
                  onChange={(e) => handleInputChange('withdrawalFeeMin', e.target.value)}
                  placeholder="0.00001"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Minimum withdrawal fee
                </p>
              </div>
              
              <div className="form-group">
                <label className="form-label">Network Fee</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.networkFee || ''}
                  onChange={(e) => handleInputChange('networkFee', e.target.value)}
                  placeholder="0.00001"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Estimated network gas fee
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Deposit Fees */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Deposit Fees</h3>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              <div className="form-group">
                <label className="form-label">Deposit Fee</label>
                <input
                  type="text"
                  className="form-input"
                  value={editedFees?.depositFee || ''}
                  onChange={(e) => handleInputChange('depositFee', e.target.value)}
                  placeholder="0"
                />
                <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                  Fee for deposits (usually 0)
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Fee Preview */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Fee Summary</h3>
          </div>
          <div className="card-body">
            <div className="space-y-3">
              <div className="flex justify-between py-2 border-b" style={{ borderColor: 'var(--border-primary)' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Trading Fee</span>
                <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                  {editedFees?.tradingFee || '0'}%
                </span>
              </div>
              <div className="flex justify-between py-2 border-b" style={{ borderColor: 'var(--border-primary)' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Maker Fee</span>
                <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                  {editedFees?.makerFee || '0'}%
                </span>
              </div>
              <div className="flex justify-between py-2 border-b" style={{ borderColor: 'var(--border-primary)' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Taker Fee</span>
                <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                  {editedFees?.takerFee || '0'}%
                </span>
              </div>
              <div className="flex justify-between py-2 border-b" style={{ borderColor: 'var(--border-primary)' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Withdrawal Fee</span>
                <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                  {editedFees?.withdrawalFee || '0'}
                </span>
              </div>
              <div className="flex justify-between py-2">
                <span style={{ color: 'var(--text-secondary)' }}>Deposit Fee</span>
                <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                  {editedFees?.depositFee || '0'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Save Button */}
      <div className="mt-6 flex justify-end">
        <button
          className="btn btn-primary btn-lg"
          onClick={handleSave}
          disabled={saving}
        >
          {saving ? (
            <>
              <span className="spinner mr-2"></span>
              Saving...
            </>
          ) : (
            'Save Changes'
          )}
        </button>
      </div>
    </div>
  );
};

export default FeesPage;
