/**
 * TigerWallet Admin Platform - Main App
 * Production-ready with real API connections
 */

import React, { useState, useEffect } from 'react';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import Layout from './components/Layout';
import { apiClient, authService, DashboardStats, User, UserKYC, Transaction, TradingPair, Blockchain, FeeStructure } from './services/api';

// Dashboard Component
const Dashboard: React.FC = () => {
  const { theme } = useTheme();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDashboard();
  }, []);

  const loadDashboard = async () => {
    try {
      const data = await apiClient.getDashboardStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="spinner"></div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <StatCard title="Total Users" value={stats?.total_users || 0} icon="👥" theme={theme} />
      <StatCard title="Active Users" value={stats?.active_users || 0} icon="✅" theme={theme} />
      <StatCard title="KYC Pending" value={stats?.kyc_pending || 0} icon="🆔" theme={theme} />
      <StatCard title="Total Transactions" value={stats?.total_transactions || 0} icon="💸" theme={theme} />
    </div>
  );
};

const StatCard: React.FC<{ title: string; value: number; icon: string; theme: string }> = ({ title, value, icon, theme }) => (
  <div className={`card ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
    <div className="flex items-center justify-between">
      <div>
        <p className={`text-sm ${theme === 'dark' ? 'text-gray-400' : 'text-gray-500'}`}>{title}</p>
        <p className={`text-2xl font-bold mt-1 ${theme === 'dark' ? 'text-white' : 'text-gray-900'}`}>
          {value.toLocaleString()}
        </p>
      </div>
      <span className="text-3xl">{icon}</span>
    </div>
  </div>
);

// Users Component
const UsersScreen: React.FC = () => {
  const { theme } = useTheme();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);

  useEffect(() => {
    loadUsers();
  }, [page]);

  const loadUsers = async () => {
    try {
      const response = await apiClient.listUsers({ page, limit: 20 });
      setUsers(response.data);
    } catch (error) {
      console.error('Failed to load users:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleBan = async (id: string) => {
    try {
      await apiClient.banUser(id);
      loadUsers();
    } catch (error) {
      console.error('Failed to ban user:', error);
    }
  };

  const handleSuspend = async (id: string) => {
    try {
      await apiClient.suspendUser(id);
      loadUsers();
    } catch (error) {
      console.error('Failed to suspend user:', error);
    }
  };

  const filteredUsers = users.filter(u => 
    u.email.toLowerCase().includes(search.toLowerCase()) ||
    (u.wallet_address?.toLowerCase().includes(search.toLowerCase()))
  );

  return (
    <div>
      <div className="flex gap-4 mb-6">
        <input
          type="text"
          placeholder="Search by email or wallet..."
          className={`input flex-1 ${theme === 'dark' ? 'bg-gray-800 text-white' : 'bg-white text-gray-900'}`}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="spinner"></div>
        </div>
      ) : (
        <div className={`card overflow-hidden ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
          <table className="table">
            <thead>
              <tr>
                <th>Email</th>
                <th>Wallet</th>
                <th>KYC</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map(user => (
                <tr key={user.id}>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{user.email}</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>
                    {user.wallet_address ? `${user.wallet_address.slice(0, 6)}...${user.wallet_address.slice(-4)}` : '-'}
                  </td>
                  <td>
                    <span className={`badge badge-${user.kyc_status === 'approved' ? 'success' : user.kyc_status === 'pending' ? 'warning' : 'default'}`}>
                      {user.kyc_status}
                    </span>
                  </td>
                  <td>
                    <span className={`badge badge-${user.status === 'active' ? 'success' : user.status === 'suspended' ? 'warning' : 'error'}`}>
                      {user.status}
                    </span>
                  </td>
                  <td>
                    <div className="flex gap-2">
                      <button onClick={() => handleSuspend(user.id)} className="btn btn-outline text-xs">Suspend</button>
                      <button onClick={() => handleBan(user.id)} className="btn btn-error text-xs">Ban</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="pagination mt-4">
        <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>Previous</button>
        <span className={`px-4 ${theme === 'dark' ? 'text-white' : 'text-gray-900'}`}>Page {page}</span>
        <button onClick={() => setPage(p => p + 1)}>Next</button>
      </div>
    </div>
  );
};

// KYC Component
const KYCScreen: React.FC = () => {
  const { theme } = useTheme();
  const [requests, setRequests] = useState<UserKYC[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadKYC();
  }, []);

  const loadKYC = async () => {
    try {
      const response = await apiClient.listKYC({ status: 'pending' });
      setRequests(response.data);
    } catch (error) {
      console.error('Failed to load KYC:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      await apiClient.approveKYC(id);
      loadKYC();
    } catch (error) {
      console.error('Failed to approve KYC:', error);
    }
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Enter rejection reason:');
    if (!reason) return;
    try {
      await apiClient.rejectKYC(id, reason);
      loadKYC();
    } catch (error) {
      console.error('Failed to reject KYC:', error);
    }
  };

  return (
    <div>
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="spinner"></div>
        </div>
      ) : (
        <div className={`card ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
          <h2 className={`text-lg font-semibold mb-4 ${theme === 'dark' ? 'text-white' : 'text-gray-900'}`}>
            Pending KYC Requests ({requests.length})
          </h2>
          {requests.length === 0 ? (
            <p className={theme === 'dark' ? 'text-gray-400' : 'text-gray-500'}>No pending requests</p>
          ) : (
            <div className="space-y-4">
              {requests.map(req => (
                <div key={req.id} className={`p-4 rounded-lg ${theme === 'dark' ? 'bg-gray-700' : 'bg-gray-50'}`}>
                  <div className="flex justify-between items-start">
                    <div>
                      <p className={`font-medium ${theme === 'dark' ? 'text-white' : 'text-gray-900'}`}>
                        {req.first_name} {req.last_name}
                      </p>
                      <p className={`text-sm ${theme === 'dark' ? 'text-gray-400' : 'text-gray-500'}`}>
                        Type: {req.kyc_type} | Document: {req.document_type}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <button onClick={() => handleApprove(req.id)} className="btn btn-success">Approve</button>
                      <button onClick={() => handleReject(req.id)} className="btn btn-error">Reject</button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// Transactions Component
const TransactionsScreen: React.FC = () => {
  const { theme } = useTheme();
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [typeFilter, setTypeFilter] = useState('');

  useEffect(() => {
    loadTransactions();
  }, [typeFilter]);

  const loadTransactions = async () => {
    try {
      const response = await apiClient.listTransactions({ type: typeFilter || undefined });
      setTransactions(response.data);
    } catch (error) {
      console.error('Failed to load transactions:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="flex gap-4 mb-6">
        <select
          className={`input w-48 ${theme === 'dark' ? 'bg-gray-800 text-white' : 'bg-white text-gray-900'}`}
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
        >
          <option value="">All Types</option>
          <option value="deposit">Deposit</option>
          <option value="withdrawal">Withdrawal</option>
          <option value="transfer">Transfer</option>
          <option value="swap">Swap</option>
        </select>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="spinner"></div>
        </div>
      ) : (
        <div className={`card overflow-hidden ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Type</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Date</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map(tx => (
                <tr key={tx.id}>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{tx.id.slice(0, 8)}...</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{tx.type}</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{tx.amount}</td>
                  <td>
                    <span className={`badge badge-${tx.status === 'completed' ? 'success' : tx.status === 'pending' ? 'warning' : 'error'}`}>
                      {tx.status}
                    </span>
                  </td>
                  <td className={theme === 'dark' ? 'text-gray-400' : 'text-gray-500'}>
                    {new Date(tx.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

// Trading Pairs Component
const PairsScreen: React.FC = () => {
  const { theme } = useTheme();
  const [pairs, setPairs] = useState<TradingPair[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadPairs();
  }, []);

  const loadPairs = async () => {
    try {
      const response = await apiClient.listPairs({});
      setPairs(response.data);
    } catch (error) {
      console.error('Failed to load pairs:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSuspend = async (id: string) => {
    try {
      await apiClient.suspendPair(id);
      loadPairs();
    } catch (error) {
      console.error('Failed to suspend pair:', error);
    }
  };

  const handleResume = async (id: string) => {
    try {
      await apiClient.resumePair(id);
      loadPairs();
    } catch (error) {
      console.error('Failed to resume pair:', error);
    }
  };

  const handleHalt = async (id: string) => {
    if (!confirm('Emergency halt? This will stop all trading on this pair.')) return;
    try {
      await apiClient.haltPair(id);
      loadPairs();
    } catch (error) {
      console.error('Failed to halt pair:', error);
    }
  };

  return (
    <div>
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="spinner"></div>
        </div>
      ) : (
        <div className={`card overflow-hidden ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
          <table className="table">
            <thead>
              <tr>
                <th>Pair</th>
                <th>Chain</th>
                <th>Maker Fee</th>
                <th>Taker Fee</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {pairs.map(pair => (
                <tr key={pair.id}>
                  <td className={theme === 'dark' ? 'text-white font-medium' : 'text-gray-900 font-medium'}>
                    {pair.base_token}/{pair.quote_token}
                  </td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{pair.chain_id.slice(0, 8)}...</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{pair.maker_fee}</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{pair.taker_fee}</td>
                  <td>
                    <span className={`badge badge-${pair.status === 'active' ? 'success' : pair.status === 'suspended' ? 'warning' : 'error'}`}>
                      {pair.status}
                    </span>
                  </td>
                  <td>
                    <div className="flex gap-2">
                      {pair.status === 'active' ? (
                        <>
                          <button onClick={() => handleSuspend(pair.id)} className="btn btn-outline text-xs">Suspend</button>
                          <button onClick={() => handleHalt(pair.id)} className="btn btn-error text-xs">Halt</button>
                        </>
                      ) : (
                        <button onClick={() => handleResume(pair.id)} className="btn btn-success text-xs">Resume</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

// Blockchains Component
const ChainsScreen: React.FC = () => {
  const { theme } = useTheme();
  const [chains, setChains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadChains();
  }, []);

  const loadChains = async () => {
    try {
      const response = await apiClient.listBlockchains();
      setChains(response.data);
    } catch (error) {
      console.error('Failed to load chains:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleMaintenance = async (id: string, current: boolean) => {
    try {
      await apiClient.setMaintenance(id, !current);
      loadChains();
    } catch (error) {
      console.error('Failed to toggle maintenance:', error);
    }
  };

  const handleToggleActive = async (id: string, current: boolean) => {
    try {
      await apiClient.activateBlockchain(id, !current);
      loadChains();
    } catch (error) {
      console.error('Failed to toggle active:', error);
    }
  };

  return (
    <div>
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="spinner"></div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {chains.map(chain => (
            <div key={chain.id} className={`card ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className={`font-semibold ${theme === 'dark' ? 'text-white' : 'text-gray-900'}`}>
                    {chain.name} ({chain.symbol})
                  </h3>
                  <p className={`text-sm ${theme === 'dark' ? 'text-gray-400' : 'text-gray-500'}`}>
                    Type: {chain.chain_type}
                  </p>
                </div>
                <span className={`badge badge-${chain.is_active ? 'success' : 'error'}`}>
                  {chain.is_active ? 'Active' : 'Inactive'}
                </span>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => handleToggleActive(chain.id, chain.is_active)}
                  className={`btn btn-outline text-xs flex-1`}
                >
                  {chain.is_active ? 'Deactivate' : 'Activate'}
                </button>
                <button
                  onClick={() => handleToggleMaintenance(chain.id, chain.is_maintenance)}
                  className={`btn text-xs flex-1 ${chain.is_maintenance ? 'btn-warning' : 'btn-outline'}`}
                >
                  {chain.is_maintenance ? 'Resume' : 'Maintenance'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Fees Component
const FeesScreen: React.FC = () => {
  const { theme } = useTheme();
  const [fees, setFees] = useState<FeeStructure[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadFees();
  }, []);

  const loadFees = async () => {
    try {
      const response = await apiClient.listFees();
      setFees(response.data);
    } catch (error) {
      console.error('Failed to load fees:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="spinner"></div>
        </div>
      ) : (
        <div className={`card overflow-hidden ${theme === 'dark' ? 'bg-gray-800' : 'bg-white'}`}>
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Maker Fee</th>
                <th>Taker Fee</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {fees.map(fee => (
                <tr key={fee.id}>
                  <td className={theme === 'dark' ? 'text-white font-medium' : 'text-gray-900 font-medium'}>
                    {fee.name}
                  </td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{fee.fee_type}</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{fee.maker_fee}</td>
                  <td className={theme === 'dark' ? 'text-white' : 'text-gray-900'}>{fee.taker_fee}</td>
                  <td>
                    <span className={`badge badge-${fee.is_active ? 'success' : 'error'}`}>
                      {fee.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

// Main App Content
const AppContent: React.FC = () => {
  const [currentPage, setCurrentPage] = useState('Dashboard');
  const { theme } = useTheme();

  const renderPage = () => {
    switch (currentPage) {
      case 'Dashboard': return <Dashboard />;
      case 'Users': return <UsersScreen />;
      case 'KYC': return <KYCScreen />;
      case 'Transactions': return <TransactionsScreen />;
      case 'Trading Pairs': return <PairsScreen />;
      case 'Blockchains': return <ChainsScreen />;
      case 'Fees': return <FeesScreen />;
      default:
        return (
          <div className={`text-center py-12 ${theme === 'dark' ? 'text-gray-400' : 'text-gray-500'}`}>
            <p className="text-4xl mb-4">🚧</p>
            <p>Page "{currentPage}" coming soon...</p>
          </div>
        );
    }
  };

  return (
    <Layout currentPage={currentPage} setCurrentPage={setCurrentPage}>
      {renderPage()}
    </Layout>
  );
};

// Main App with Theme Provider
const App: React.FC = () => {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  );
};

export default App;
