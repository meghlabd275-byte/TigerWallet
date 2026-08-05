/**
 * TigerWallet Super Admin - Transactions Page
 */

import React, { useState, useEffect } from 'react';
import superAdminApi from '../services/api';

export default function Transactions() {
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setTimeout(() => setLoading(false), 500);
  }, []);

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Transactions</h1>
      <div className="card">
        <div className="card-body">
          {loading ? (
            <div className="flex items-center justify-center p-8">
              <div className="loader"></div>
            </div>
          ) : (
            <div className="text-center py-8">
              <p className="text-secondary">Transaction management interface</p>
              <p className="text-tertiary mt-2">View, flag, and manage all platform transactions</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
