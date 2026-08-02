/**
 * TigerWallet Admin Panel - Main Application
 * Production-ready with full theme support and API integration
 */

import React, { useState, useEffect, createContext, useContext } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import './styles/index.css';

// Theme Context for light/dark mode
type Theme = 'light' | 'dark';

interface ThemeContextType {
  theme: Theme;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType>({ theme: 'dark', toggleTheme: () => {} });

export const useTheme = () => useContext(ThemeContext);

const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [theme, setTheme] = useState<Theme>(() => {
    const stored = localStorage.getItem('tigerwallet_theme');
    return (stored as Theme) || 'dark';
  });

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('tigerwallet_theme', theme);
  }, [theme]);

  const toggleTheme = () => setTheme(prev => prev === 'dark' ? 'light' : 'dark');

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

// API Base URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

// Auth Context
interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType>({
  isAuthenticated: false,
  token: null,
  login: () => {},
  logout: () => {},
});

export const useAuth = () => useContext(AuthContext);

const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('superadmin_token'));
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(!!token);

  const login = (newToken: string) => {
    localStorage.setItem('superadmin_token', newToken);
    setToken(newToken);
    setIsAuthenticated(true);
  };

  const logout = () => {
    localStorage.removeItem('superadmin_token');
    setToken(null);
    setIsAuthenticated(false);
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

// Pages
import Dashboard from './pages/Dashboard';
import WhiteLabelManagement from './pages/WhiteLabelManagement';
import UserManagement from './pages/UserManagement';
import TokenManagement from './pages/TokenManagement';
import BlockchainManagement from './pages/BlockchainManagement';
import PairManagement from './pages/PairManagement';
import LiquidityManagement from './pages/LiquidityManagement';
import FeeManagement from './pages/FeeManagement';
import Analytics from './pages/Analytics';
import AdminManagement from './pages/AdminManagement';
import APIManagement from './pages/APIManagement';
import KYCMangement from './pages/KYCMangement';
import MasterWallet from './pages/MasterWallet';
import WithdrawalManagement from './pages/WithdrawalManagement';

// Login Page
const LoginPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    try {
      const response = await fetch(`${API_BASE_URL}/super-admin/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        throw new Error('Invalid credentials');
      }

      const data = await response.json();
      login(data.token);
      navigate('/');
    } catch (err) {
      setError('Login failed. Please check your credentials.');
    }
  };

  return (
    <div className="login-page">
      <div className="login-container">
        <div className="login-header">
          <h1>TigerWallet</h1>
          <h2>Super Admin Panel</h2>
        </div>
        <form onSubmit={handleLogin} className="login-form">
          {error && <div className="error-message">{error}</div>}
          <div className="form-group">
            <label>Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="superadmin@tigerwallet.com"
              required
            />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
            />
          </div>
          <button type="submit" className="btn btn-primary">Login</button>
        </form>
      </div>
    </div>
  );
};

// Sidebar
const Sidebar: React.FC = () => {
  const navigate = useNavigate();
  const [activeItem, setActiveItem] = useState('dashboard');

  const menuItems = [
    { id: 'dashboard', label: 'Dashboard', path: '/', icon: '📊' },
    { id: 'users', label: 'User Management', path: '/users', icon: '👥' },
    { id: 'whitelabel', label: 'White Label', path: '/whitelabel', icon: '🏢' },
    { id: 'tokens', label: 'Token Management', path: '/tokens', icon: '🪙' },
    { id: 'blockchains', label: 'Blockchains', path: '/blockchains', icon: '⛓️' },
    { id: 'pairs', label: 'Trading Pairs', path: '/pairs', icon: '🔄' },
    { id: 'liquidity', label: 'Liquidity', path: '/liquidity', icon: '💧' },
    { id: 'fees', label: 'Fee Management', path: '/fees', icon: '💰' },
    { id: 'analytics', label: 'Analytics', path: '/analytics', icon: '📈' },
    { id: 'admins', label: 'Admin Management', path: '/admins', icon: '👤' },
    { id: 'api', label: 'API Management', path: '/api', icon: '🔌' },
    { id: 'kyc', label: 'KYC Management', path: '/kyc', icon: '🛡️' },
    { id: 'masterwallet', label: 'Master Wallet', path: '/master-wallet', icon: '🔐' },
    { id: 'withdrawals', label: 'Withdrawals', path: '/withdrawals', icon: '💸' },
  ];

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <h2>TigerWallet</h2>
        <span className="badge">Admin</span>
      </div>
      <nav className="sidebar-nav">
        {menuItems.map(item => (
          <button
            key={item.id}
            className={`nav-item ${activeItem === item.id ? 'active' : ''}`}
            onClick={() => {
              setActiveItem(item.id);
              navigate(item.path);
            }}
          >
            <span className="nav-icon">{item.icon}</span>
            <span className="nav-label">{item.label}</span>
          </button>
        ))}
      </nav>
    </div>
  );
};

// Header
const Header: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const { logout } = useAuth();
  const navigate = useNavigate();

  return (
    <header className="header">
      <div className="header-left">
        <h1>Super Admin Panel</h1>
      </div>
      <div className="header-right">
        <button onClick={toggleTheme} className="theme-toggle" title="Toggle theme">
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
        <button onClick={() => { logout(); navigate('/login'); }} className="btn btn-secondary">
          Logout
        </button>
      </div>
    </header>
  );
};

// Layout
const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <div className="admin-layout">
      <Sidebar />
      <div className="main-wrapper">
        <Header />
        <main className="main-content">
          {children}
        </main>
      </div>
    </div>
  );
};

// Main App
const App: React.FC = () => {
  return (
    <ThemeProvider>
      <AuthProvider>
        <Router>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/" element={<RequireAuth><Layout><Dashboard /></Layout></RequireAuth>} />
            <Route path="/users" element={<RequireAuth><Layout><UserManagement /></Layout></RequireAuth>} />
            <Route path="/whitelabel" element={<RequireAuth><Layout><WhiteLabelManagement /></Layout></RequireAuth>} />
            <Route path="/tokens" element={<RequireAuth><Layout><TokenManagement /></Layout></RequireAuth>} />
            <Route path="/blockchains" element={<RequireAuth><Layout><BlockchainManagement /></Layout></RequireAuth>} />
            <Route path="/pairs" element={<RequireAuth><Layout><PairManagement /></Layout></RequireAuth>} />
            <Route path="/liquidity" element={<RequireAuth><Layout><LiquidityManagement /></Layout></RequireAuth>} />
            <Route path="/fees" element={<RequireAuth><Layout><FeeManagement /></Layout></RequireAuth>} />
            <Route path="/analytics" element={<RequireAuth><Layout><Analytics /></Layout></RequireAuth>} />
            <Route path="/admins" element={<RequireAuth><Layout><AdminManagement /></Layout></RequireAuth>} />
            <Route path="/api" element={<RequireAuth><Layout><APIManagement /></Layout></RequireAuth>} />
            <Route path="/kyc" element={<RequireAuth><Layout><KYCMangement /></Layout></RequireAuth>} />
            <Route path="/master-wallet" element={<RequireAuth><Layout><MasterWallet /></Layout></RequireAuth>} />
            <Route path="/withdrawals" element={<RequireAuth><Layout><WithdrawalManagement /></Layout></RequireAuth>} />
          </Routes>
        </Router>
      </AuthProvider>
    </ThemeProvider>
  );
};

// Require Auth Component
const RequireAuth: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

export default App;
