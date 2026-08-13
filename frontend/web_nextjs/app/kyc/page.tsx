'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data = await response.json();
  return data?.data ?? data;
};

interface KYCStatus {
  level: number;
  status: 'none' | 'pending' | 'approved' | 'rejected';
  documents: {
    id: string;
    type: string;
    status: 'pending' | 'verified' | 'rejected';
    submittedAt: number;
  }[];
  limits: {
    deposit: number;
    withdraw: number;
    trading: number;
  };
  verifiedAt?: number;
}

const DEFAULT_STATUS: KYCStatus = {
  level: 0,
  status: 'none',
  documents: [],
  limits: { deposit: 1000, withdraw: 1000, trading: 2500 },
};

export default function KYCPage() {
  const [kycStatus, setKycStatus] = useState<KYCStatus>(DEFAULT_STATUS);
  const [showModal, setShowModal] = useState(false);
  const [uploadStep, setUploadStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pageLoading, setPageLoading] = useState(true);
  const { isDark } = useTheme();

  // Fetch the real KYC status from the listing_service backend (no hardcoded
  // "approved" seed). The status route requires a user_id; we derive it from
  // the JWT via the backend profile lookup, falling back to the email claim.
  const loadStatus = useCallback(async () => {
    setPageLoading(true);
    setError(null);
    try {
      let userId = '';
      try {
        const profile = await fetchAPI<any>('/user/profile');
        userId = profile?.id || profile?.user_id || profile?.userId || '';
      } catch (err) {
        // profile lookup may fail if not authenticated — fall through to 401 below
      }
      const status = await fetchAPI<KYCStatus>(`/kyc/status?user_id=${encodeURIComponent(userId)}`);
      setKycStatus({
        level: status?.level ?? 0,
        status: status?.status ?? 'none',
        documents: status?.documents ?? [],
        limits: status?.limits ?? DEFAULT_STATUS.limits,
        verifiedAt: status?.verifiedAt,
      });
    } catch (err) {
      setError('Failed to load KYC status. Please try again.');
      setKycStatus(DEFAULT_STATUS);
    } finally {
      setPageLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const handleUpload = async (docType: string) => {
    setLoading(true);
    setError(null);
    try {
      // Submit the document metadata to the real KYC document endpoint.
      // The backend stores the doc + starts async AML verification.
      await fetchAPI('/kyc/submit', {
        method: 'POST',
        body: JSON.stringify({ doc_type: docType, step: uploadStep }),
      });
      // Refresh status so the new pending document appears from the backend.
      await loadStatus();
      setShowModal(false);
      setUploadStep(0);
    } catch (err) {
      setError('Failed to submit document. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const getLevelBenefits = (level: number) => {
    const levels = {
      0: { name: 'Unverified', deposit: 1000, withdraw: 1000, trading: 2500 },
      1: { name: 'Basic', deposit: 10000, withdraw: 10000, trading: 50000 },
      2: { name: 'Intermediate', deposit: 100000, withdraw: 100000, trading: 500000 },
      3: { name: 'Advanced', deposit: 1000000, withdraw: 1000000, trading: 5000000 },
      4: { name: 'Ultimate', deposit: -1, withdraw: -1, trading: -1 },
    };
    return levels[level as keyof typeof levels] || levels[0];
  };

  const currentLimits = getLevelBenefits(kycStatus.level);

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900' : 'bg-slate-50'}`}>
      <header className={`border-b ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
        <div className="max-w-4xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-2xl">🐯</a>
            <div>
              <h1 className="text-xl font-bold">Identity Verification (KYC)</h1>
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Verify your identity to unlock higher limits</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 py-8">
        {pageLoading && (
          <div className={`rounded-xl p-4 mb-6 ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
            <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>Loading verification status...</p>
          </div>
        )}
        {error && (
          <div className="rounded-xl p-4 mb-6 bg-red-100 text-red-800">
            <p>{error}</p>
            <button onClick={loadStatus} className="mt-2 text-sm underline">Retry</button>
          </div>
        )}
        {/* Status Card */}
        <div className={`rounded-xl p-6 mb-6 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-lg font-semibold">Verification Status</h2>
              <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>Your current verification level</p>
            </div>
            <span className={`px-4 py-2 rounded-full text-sm font-medium ${
              kycStatus.status === 'approved' ? 'bg-green-100 text-green-800' :
              kycStatus.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
              'bg-gray-100 text-gray-800'
            }`}>
              {kycStatus.status.toUpperCase()}
            </span>
          </div>

          <div className="flex items-center gap-4 mb-6">
            <div className="w-16 h-16 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white text-2xl font-bold">
              L{kycStatus.level}
            </div>
            <div>
              <p className="text-2xl font-bold">{getLevelBenefits(kycStatus.level).name}</p>
              <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>
                {kycStatus.verifiedAt ? `Verified ${new Date(kycStatus.verifiedAt).toLocaleDateString()}` : 'Verification pending'}
              </p>
            </div>
          </div>

          {/* Limits */}
          <div className={`grid grid-cols-3 gap-4 p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
            <div className="text-center">
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Deposit Limit</p>
              <p className="text-xl font-bold">{currentLimits.deposit === -1 ? 'Unlimited' : `$${currentLimits.deposit.toLocaleString()}`}</p>
            </div>
            <div className="text-center">
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Withdraw Limit</p>
              <p className="text-xl font-bold">{currentLimits.withdraw === -1 ? 'Unlimited' : `$${currentLimits.withdraw.toLocaleString()}`}</p>
            </div>
            <div className="text-center">
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Trading Limit</p>
              <p className="text-xl font-bold">{currentLimits.trading === -1 ? 'Unlimited' : `$${currentLimits.trading.toLocaleString()}`}</p>
            </div>
          </div>
        </div>

        {/* Upgrade CTA */}
        {kycStatus.level < 4 && (
          <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl p-6 mb-6 text-white">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-bold">Upgrade to Level {kycStatus.level + 1}</h3>
                <p className="text-blue-100">Unlock higher limits and features</p>
              </div>
              <button
                onClick={() => setShowModal(true)}
                className="px-6 py-3 bg-white text-blue-600 rounded-lg font-semibold hover:bg-blue-50"
              >
                Upgrade Now
              </button>
            </div>
          </div>
        )}

        {/* Submitted Documents */}
        <div className={`rounded-xl p-6 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
          <h3 className="text-lg font-semibold mb-4">Submitted Documents</h3>
          
          {kycStatus.documents.length === 0 ? (
            <p className={`text-center py-8 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No documents submitted yet</p>
          ) : (
            <div className="space-y-3">
              {kycStatus.documents.map(doc => (
                <div key={doc.id} className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                  <div>
                    <p className="font-medium">{doc.type}</p>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Submitted {new Date(doc.submittedAt).toLocaleDateString()}</p>
                  </div>
                  <span className={`px-3 py-1 rounded-full text-sm ${
                    doc.status === 'verified' ? 'bg-green-100 text-green-800' :
                    doc.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-red-100 text-red-800'
                  }`}>
                    {doc.status.toUpperCase()}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* KYC Levels Info */}
        <div className={`rounded-xl p-6 border mt-6 ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
          <h3 className="text-lg font-semibold mb-4">Verification Levels</h3>
          
          <div className="space-y-4">
            {[
              { level: 0, name: 'Unverified', requirements: 'Email verification only', limits: '$1,000' },
              { level: 1, name: 'Basic', requirements: 'ID Document + Selfie', limits: '$10,000' },
              { level: 2, name: 'Intermediate', requirements: 'Proof of Address', limits: '$100,000' },
              { level: 3, name: 'Advanced', requirements: 'Video verification', limits: '$1,000,000' },
              { level: 4, name: 'Ultimate', requirements: 'Business verification', limits: 'Unlimited' },
            ].map(level => (
              <div key={level.level} className={`flex items-center justify-between p-4 rounded-lg ${kycStatus.level === level.level ? (isDark ? 'bg-blue-900/20 border border-blue-800' : 'bg-blue-50 border border-blue-200') : (isDark ? 'bg-slate-700' : 'bg-slate-50')}`}>
                <div className="flex items-center gap-4">
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold ${kycStatus.level >= level.level ? 'bg-blue-600 text-white' : 'bg-slate-300'}`}>
                    {level.level + 1}
                  </div>
                  <div>
                    <p className="font-semibold">{level.name}</p>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{level.requirements}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-bold">{level.limits}</p>
                  <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Limits</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Upload Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`rounded-xl p-6 max-w-md w-full mx-4 ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
            <h3 className="text-xl font-bold mb-4">Submit Documents</h3>
            
            {uploadStep === 0 && (
              <div className="space-y-3">
                <button
                  onClick={() => { setUploadStep(1); }}
                  className="w-full p-4 border-2 border-dashed border-slate-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
                >
                  <p className="font-medium">Government ID</p>
                  <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Passport, Driver's License, or National ID</p>
                </button>
                <button
                  onClick={() => { setUploadStep(2); }}
                  className="w-full p-4 border-2 border-dashed border-slate-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
                >
                  <p className="font-medium">Proof of Address</p>
                  <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Bank Statement or Utility Bill (within 3 months)</p>
                </button>
                <button
                  onClick={() => { setUploadStep(3); }}
                  className="w-full p-4 border-2 border-dashed border-slate-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
                >
                  <p className="font-medium">Selfie with ID</p>
                  <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Take a photo holding your ID</p>
                </button>
              </div>
            )}

            {uploadStep > 0 && (
              <div className="text-center">
                <div className="w-20 h-20 mx-auto mb-4 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                <p className="text-lg font-medium mb-2">Uploading Document...</p>
                <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>Please wait while we process your document</p>
                {loading && (
                  <div className="mt-4">
                    <button
                      onClick={() => handleUpload(uploadStep === 1 ? 'Government ID' : uploadStep === 2 ? 'Proof of Address' : 'Selfie with ID')}
                      className="px-6 py-2 bg-blue-600 text-white rounded-lg"
                    >
                      Confirm Upload
                    </button>
                  </div>
                )}
              </div>
            )}

            <button
              onClick={() => { setShowModal(false); setUploadStep(0); }}
              className={`w-full mt-4 py-2 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-200'}`}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
