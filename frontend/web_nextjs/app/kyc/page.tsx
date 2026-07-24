'use client';

import React, { useState } from 'react';

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

export default function KYCPage() {
  const [kycStatus, setKycStatus] = useState<KYCStatus>({
    level: 1,
    status: 'approved',
    documents: [
      { id: '1', type: 'ID Document', status: 'verified', submittedAt: Date.now() - 86400000 * 30 },
      { id: '2', type: 'Selfie', status: 'verified', submittedAt: Date.now() - 86400000 * 30 },
      { id: '3', type: 'Proof of Address', status: 'verified', submittedAt: Date.now() - 86400000 * 25 },
    ],
    limits: {
      deposit: 10000,
      withdraw: 10000,
      trading: 50000,
    },
    verifiedAt: Date.now() - 86400000 * 25,
  });

  const [showModal, setShowModal] = useState(false);
  const [uploadStep, setUploadStep] = useState(0);
  const [loading, setLoading] = useState(false);

  const handleUpload = async (docType: string) => {
    setLoading(true);
    await new Promise(r => setTimeout(r, 1500));
    
    const newDoc = {
      id: Date.now().toString(),
      type: docType,
      status: 'pending' as const,
      submittedAt: Date.now(),
    };
    
    setKycStatus(prev => ({
      ...prev,
      status: 'pending',
      documents: [...prev.documents, newDoc],
    }));
    
    setLoading(false);
    setShowModal(false);
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
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-4xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-2xl">🐯</a>
            <div>
              <h1 className="text-xl font-bold">Identity Verification (KYC)</h1>
              <p className="text-slate-500 text-sm">Verify your identity to unlock higher limits</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-4 py-8">
        {/* Status Card */}
        <div className="bg-white dark:bg-slate-800 rounded-xl p-6 mb-6 border border-slate-200 dark:border-slate-700">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-lg font-semibold">Verification Status</h2>
              <p className="text-slate-500">Your current verification level</p>
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
              <p className="text-slate-500">
                {kycStatus.verifiedAt ? `Verified ${new Date(kycStatus.verifiedAt).toLocaleDateString()}` : 'Verification pending'}
              </p>
            </div>
          </div>

          {/* Limits */}
          <div className="grid grid-cols-3 gap-4 p-4 bg-slate-50 dark:bg-slate-700 rounded-lg">
            <div className="text-center">
              <p className="text-sm text-slate-500">Deposit Limit</p>
              <p className="text-xl font-bold">{currentLimits.deposit === -1 ? 'Unlimited' : `$${currentLimits.deposit.toLocaleString()}`}</p>
            </div>
            <div className="text-center">
              <p className="text-sm text-slate-500">Withdraw Limit</p>
              <p className="text-xl font-bold">{currentLimits.withdraw === -1 ? 'Unlimited' : `$${currentLimits.withdraw.toLocaleString()}`}</p>
            </div>
            <div className="text-center">
              <p className="text-sm text-slate-500">Trading Limit</p>
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
        <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700">
          <h3 className="text-lg font-semibold mb-4">Submitted Documents</h3>
          
          {kycStatus.documents.length === 0 ? (
            <p className="text-slate-500 text-center py-8">No documents submitted yet</p>
          ) : (
            <div className="space-y-3">
              {kycStatus.documents.map(doc => (
                <div key={doc.id} className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-700 rounded-lg">
                  <div>
                    <p className="font-medium">{doc.type}</p>
                    <p className="text-sm text-slate-500">Submitted {new Date(doc.submittedAt).toLocaleDateString()}</p>
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
        <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700 mt-6">
          <h3 className="text-lg font-semibold mb-4">Verification Levels</h3>
          
          <div className="space-y-4">
            {[
              { level: 0, name: 'Unverified', requirements: 'Email verification only', limits: '$1,000' },
              { level: 1, name: 'Basic', requirements: 'ID Document + Selfie', limits: '$10,000' },
              { level: 2, name: 'Intermediate', requirements: 'Proof of Address', limits: '$100,000' },
              { level: 3, name: 'Advanced', requirements: 'Video verification', limits: '$1,000,000' },
              { level: 4, name: 'Ultimate', requirements: 'Business verification', limits: 'Unlimited' },
            ].map(level => (
              <div key={level.level} className={`flex items-center justify-between p-4 rounded-lg ${kycStatus.level === level.level ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800' : 'bg-slate-50 dark:bg-slate-700'}`}>
                <div className="flex items-center gap-4">
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold ${kycStatus.level >= level.level ? 'bg-blue-600 text-white' : 'bg-slate-300'}`}>
                    {level.level + 1}
                  </div>
                  <div>
                    <p className="font-semibold">{level.name}</p>
                    <p className="text-sm text-slate-500">{level.requirements}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-bold">{level.limits}</p>
                  <p className="text-xs text-slate-500">Limits</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Upload Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-xl font-bold mb-4">Submit Documents</h3>
            
            {uploadStep === 0 && (
              <div className="space-y-3">
                <button
                  onClick={() => { setUploadStep(1); }}
                  className="w-full p-4 border-2 border-dashed border-slate-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
                >
                  <p className="font-medium">Government ID</p>
                  <p className="text-sm text-slate-500">Passport, Driver's License, or National ID</p>
                </button>
                <button
                  onClick={() => { setUploadStep(2); }}
                  className="w-full p-4 border-2 border-dashed border-slate-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
                >
                  <p className="font-medium">Proof of Address</p>
                  <p className="text-sm text-slate-500">Bank Statement or Utility Bill (within 3 months)</p>
                </button>
                <button
                  onClick={() => { setUploadStep(3); }}
                  className="w-full p-4 border-2 border-dashed border-slate-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left"
                >
                  <p className="font-medium">Selfie with ID</p>
                  <p className="text-sm text-slate-500">Take a photo holding your ID</p>
                </button>
              </div>
            )}

            {uploadStep > 0 && (
              <div className="text-center">
                <div className="w-20 h-20 mx-auto mb-4 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                <p className="text-lg font-medium mb-2">Uploading Document...</p>
                <p className="text-slate-500">Please wait while we process your document</p>
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
              className="w-full mt-4 py-2 bg-slate-200 dark:bg-slate-700 rounded-lg"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
