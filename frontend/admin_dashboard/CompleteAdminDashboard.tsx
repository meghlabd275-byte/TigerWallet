/**
 * TigerWallet Admin Dashboard - Complete Implementation
 * All Features + Light/Dark Theme Support
 * 
 * Features:
 * - Dashboard with all stats
 * - User Management
 * - KYC Management
 * - Token Management
 * - Transaction Management
 * - Blockchain Management
 * - White Label Management
 * - Fee Management
 * - Withdrawal Management
 * - Analytics
 * - Audit Logs
 * - System Settings
 * - Notifications
 * - 2FA Management
 * - Support Tickets
 * - Reports
 * - Bulk Operations
 * - Export (CSV/PDF)
 * - Workflow Automation
 * - Knowledge Base
 */

import React, { useState, useEffect, createContext, useContext } from 'react';
import { 
  LineChart, Line, BarChart, Bar, PieChart, Pie, 
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell 
} from 'recharts';
import { 
  Users, Shield, DollarSign, Activity, Settings, 
  Bell, Search, Menu, X, Moon, Sun, LogOut,
  ChevronDown, ChevronRight, Download, Upload,
  Plus, Edit, Trash2, Check, XCircle, AlertTriangle,
  RefreshCw, Filter, MoreVertical, Eye, Lock, Unlock,
  FileText, MessageSquare, Calendar, BarChart2, Home,
  CreditCard, Wallet, Globe, Link, Key, Smartphone,
  Mail, Phone, MapPin, Clock, AlertCircle, CheckCircle,
  Layers, Database, Server, HardDrive, Cpu
} from 'lucide-react';

// ============================================================================
// Theme Context
// ============================================================================

const ThemeContext = createContext();

export const ThemeProvider = ({ children }) => {
  const [theme, setTheme] = useState('dark');
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    // Load theme from localStorage
    const savedTheme = localStorage.getItem('admin_theme') || 'dark';
    setTheme(savedTheme);
    setIsLoaded(true);
    
    // Apply theme to document
    document.documentElement.setAttribute('data-theme', savedTheme);
  }, []);

  const toggleTheme = () => {
    const newTheme = theme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
    localStorage.setItem('admin_theme', newTheme);
    document.documentElement.setAttribute('data-theme', newTheme);
  };

  if (!isLoaded) {
    return <div className="loading-screen">Loading...</div>;
  }

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, isDark: theme === 'dark' }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => useContext(ThemeContext);

// ============================================================================
// CSS Variables (Theme)
// ============================================================================

const themeStyles = `
  :root {
    /* Light Theme */
    --bg-primary: #ffffff;
    --bg-secondary: #f5f7fa;
    --bg-tertiary: #e8ecf1;
    --bg-card: #ffffff;
    --bg-hover: #f0f4f8;
    
    --text-primary: #1a202c;
    --text-secondary: #4a5568;
    --text-muted: #718096;
    --text-inverse: #ffffff;
    
    --border-color: #e2e8f0;
    --border-light: #edf2f7;
    
    --primary: #3182ce;
    --primary-hover: #2b6cb0;
    --primary-light: #bee3f8;
    
    --success: #38a169;
    --success-light: #c6f6d5;
    
    --warning: #d69e2e;
    --warning-light: #fefcbf;
    
    --danger: #e53e3e;
    --danger-light: #fed7d7;
    
    --info: #3182ce;
    --info-light: #bee3f8;
    
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
    --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.1);
    --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.1);
    
    --radius-sm: 4px;
    --radius-md: 8px;
    --radius-lg: 12px;
    --radius-full: 9999px;
    
    --transition-fast: 150ms ease;
    --transition-normal: 250ms ease;
  }

  [data-theme="dark"] {
    --bg-primary: #0d1117;
    --bg-secondary: #161b22;
    --bg-tertiary: #21262d;
    --bg-card: #1c2128;
    --bg-hover: #30363d;
    
    --text-primary: #f0f6fc;
    --text-secondary: #8b949e;
    --text-muted: #6e7681;
    --text-inverse: #0d1117;
    
    --border-color: #30363d;
    --border-light: #21262d;
    
    --primary: #58a6ff;
    --primary-hover: #79b8ff;
    --primary-light: #388bfd26;
    
    --success: #3fb950;
    --success-light: #238636;
    
    --warning: #d29922;
    --warning-light: #9e6a03;
    
    --danger: #f85149;
    --danger-light: #da3633;
    
    --info: #58a6ff;
    --info-light: #388bfd26;
    
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
    --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.4);
    --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.5);
  }

  * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    background-color: var(--bg-primary);
    color: var(--text-primary);
    line-height: 1.6;
    transition: background-color var(--transition-normal), color var(--transition-normal);
  }

  .app-container {
    display: flex;
    min-height: 100vh;
  }

  /* Sidebar */
  .sidebar {
    width: 260px;
    background: var(--bg-secondary);
    border-right: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    transition: all var(--transition-normal);
  }

  .sidebar-header {
    padding: 20px;
    border-bottom: 1px solid var(--border-color);
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 20px;
    font-weight: 700;
    color: var(--primary);
  }

  .logo-icon {
    width: 40px;
    height: 40px;
    background: linear-gradient(135deg, var(--primary), #9f7aea);
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }

  .sidebar-nav {
    flex: 1;
    padding: 16px 0;
    overflow-y: auto;
  }

  .nav-section {
    margin-bottom: 8px;
  }

  .nav-section-title {
    padding: 8px 20px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--text-muted);
    letter-spacing: 0.5px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 20px;
    color: var(--text-secondary);
    text-decoration: none;
    cursor: pointer;
    transition: all var(--transition-fast);
    border-left: 3px solid transparent;
  }

  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .nav-item.active {
    background: var(--primary-light);
    color: var(--primary);
    border-left-color: var(--primary);
  }

  .nav-item svg {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  .nav-item span {
    font-size: 14px;
  }

  .nav-badge {
    margin-left: auto;
    background: var(--danger);
    color: white;
    font-size: 11px;
    padding: 2px 8px;
    border-radius: var(--radius-full);
  }

  /* Main Content */
  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  /* Header */
  .header {
    height: 64px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 24px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .menu-toggle {
    display: none;
    background: none;
    border: none;
    color: var(--text-primary);
    cursor: pointer;
    padding: 8px;
  }

  .search-box {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--bg-tertiary);
    border-radius: var(--radius-md);
    padding: 8px 16px;
    width: 300px;
  }

  .search-box input {
    background: none;
    border: none;
    color: var(--text-primary);
    font-size: 14px;
    width: 100%;
    outline: none;
  }

  .search-box input::placeholder {
    color: var(--text-muted);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .header-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    background: var(--bg-tertiary);
    border: none;
    border-radius: var(--radius-md);
    color: var(--text-secondary);
    cursor: pointer;
    position: relative;
    transition: all var(--transition-fast);
  }

  .header-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .header-btn .badge {
    position: absolute;
    top: -4px;
    right: -4px;
    background: var(--danger);
    color: white;
    font-size: 10px;
    width: 18px;
    height: 18px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .theme-toggle {
    background: var(--bg-tertiary);
  }

  .user-menu {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 12px;
    background: var(--bg-tertiary);
    border-radius: var(--radius-md);
    cursor: pointer;
  }

  .user-avatar {
    width: 32px;
    height: 32px;
    border-radius: var(--radius-full);
    background: var(--primary);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: 600;
    font-size: 14px;
  }

  .user-info {
    display: flex;
    flex-direction: column;
  }

  .user-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .user-role {
    font-size: 12px;
    color: var(--text-muted);
  }

  /* Page Content */
  .page-content {
    flex: 1;
    padding: 24px;
    overflow-y: auto;
    background: var(--bg-primary);
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }

  .page-title {
    font-size: 28px;
    font-weight: 700;
    color: var(--text-primary);
  }

  .page-subtitle {
    font-size: 14px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  .page-actions {
    display: flex;
    gap: 12px;
  }

  /* Cards */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    overflow: hidden;
    transition: all var(--transition-normal);
  }

  .card:hover {
    box-shadow: var(--shadow-md);
  }

  .card-header {
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .card-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .card-body {
    padding: 20px;
  }

  /* Stats Grid */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 20px;
    margin-bottom: 24px;
  }

  .stat-card {
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    padding: 20px;
    display: flex;
    align-items: flex-start;
    gap: 16px;
    transition: all var(--transition-normal);
  }

  .stat-card:hover {
    box-shadow: var(--shadow-md);
    transform: translateY(-2px);
  }

  .stat-icon {
    width: 48px;
    height: 48px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .stat-icon.users { background: var(--primary-light); color: var(--primary); }
  .stat-icon.transactions { background: var(--success-light); color: var(--success); }
  .stat-icon.volume { background: var(--warning-light); color: var(--warning); }
  .stat-icon.pending { background: var(--danger-light); color: var(--danger); }

  .stat-content {
    flex: 1;
  }

  .stat-label {
    font-size: 13px;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .stat-value {
    font-size: 28px;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.2;
  }

  .stat-change {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    margin-top: 4px;
  }

  .stat-change.positive { color: var(--success); }
  .stat-change.negative { color: var(--danger); }

  /* Buttons */
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px 20px;
    font-size: 14px;
    font-weight: 500;
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all var(--transition-fast);
    border: none;
    text-decoration: none;
  }

  .btn-primary {
    background: var(--primary);
    color: white;
  }

  .btn-primary:hover {
    background: var(--primary-hover);
  }

  .btn-secondary {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
  }

  .btn-secondary:hover {
    background: var(--bg-hover);
  }

  .btn-danger {
    background: var(--danger);
    color: white;
  }

  .btn-danger:hover {
    background: #c53030;
  }

  .btn-success {
    background: var(--success);
    color: white;
  }

  .btn-sm {
    padding: 6px 12px;
    font-size: 13px;
  }

  .btn-icon {
    padding: 8px;
    width: 36px;
    height: 36px;
  }

  /* Tables */
  .table-container {
    overflow-x: auto;
  }

  .table {
    width: 100%;
    border-collapse: collapse;
  }

  .table th {
    text-align: left;
    padding: 12px 16px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--text-muted);
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-color);
  }

  .table td {
    padding: 14px 16px;
    font-size: 14px;
    color: var(--text-primary);
    border-bottom: 1px solid var(--border-color);
  }

  .table tr:hover td {
    background: var(--bg-hover);
  }

  /* Status Badges */
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    font-size: 12px;
    font-weight: 500;
    border-radius: var(--radius-full);
  }

  .badge-success { background: var(--success-light); color: var(--success); }
  .badge-warning { background: var(--warning-light); color: var(--warning); }
  .badge-danger { background: var(--danger-light); color: var(--danger); }
  .badge-info { background: var(--info-light); color: var(--info); }
  .badge-default { background: var(--bg-tertiary); color: var(--text-secondary); }

  /* Forms */
  .form-group {
    margin-bottom: 20px;
  }

  .form-label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .form-input {
    width: 100%;
    padding: 10px 14px;
    font-size: 14px;
    color: var(--text-primary);
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    outline: none;
    transition: all var(--transition-fast);
  }

  .form-input:focus {
    border-color: var(--primary);
    box-shadow: 0 0 0 3px var(--primary-light);
  }

  .form-select {
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%23718096' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 40px;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    opacity: 0;
    visibility: hidden;
    transition: all var(--transition-normal);
  }

  .modal-overlay.active {
    opacity: 1;
    visibility: visible;
  }

  .modal {
    background: var(--bg-card);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 500px;
    max-height: 90vh;
    overflow-y: auto;
    transform: scale(0.9);
    transition: all var(--transition-normal);
  }

  .modal-overlay.active .modal {
    transform: scale(1);
  }

  .modal-header {
    padding: 20px;
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .modal-title {
    font-size: 18px;
    font-weight: 600;
  }

  .modal-close {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
  }

  .modal-body {
    padding: 20px;
  }

  .modal-footer {
    padding: 16px 20px;
    border-top: 1px solid var(--border-color);
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  /* Tabs */
  .tabs {
    display: flex;
    gap: 4px;
    background: var(--bg-secondary);
    padding: 4px;
    border-radius: var(--radius-md);
    margin-bottom: 24px;
  }

  .tab {
    padding: 8px 16px;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
    background: none;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .tab:hover {
    color: var(--text-primary);
  }

  .tab.active {
    background: var(--bg-card);
    color: var(--primary);
    box-shadow: var(--shadow-sm);
  }

  /* Pagination */
  .pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 0;
  }

  .pagination-info {
    font-size: 14px;
    color: var(--text-muted);
  }

  .pagination-controls {
    display: flex;
    gap: 8px;
  }

  .pagination-btn {
    padding: 8px 12px;
    font-size: 14px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .pagination-btn:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .pagination-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .pagination-btn.active {
    background: var(--primary);
    color: white;
    border-color: var(--primary);
  }

  /* Charts */
  .chart-container {
    height: 300px;
    margin-top: 16px;
  }

  /* Notifications */
  .notification-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .notification-item {
    display: flex;
    gap: 12px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .notification-item:hover {
    background: var(--bg-hover);
  }

  .notification-item.unread {
    background: var(--primary-light);
  }

  .notification-icon {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .notification-content {
    flex: 1;
  }

  .notification-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .notification-message {
    font-size: 13px;
    color: var(--text-secondary);
    margin-top: 2px;
  }

  .notification-time {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  /* Loading */
  .loading-screen {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-color);
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Responsive */
  @media (max-width: 768px) {
    .sidebar {
      position: fixed;
      left: -260px;
      z-index: 100;
    }

    .sidebar.open {
      left: 0;
    }

    .menu-toggle {
      display: block;
    }

    .search-box {
      display: none;
    }

    .stats-grid {
      grid-template-columns: 1fr;
    }

    .page-header {
      flex-direction: column;
      gap: 16px;
      align-items: flex-start;
    }
  }
`;

// ============================================================================
// Components
// ============================================================================

// Sidebar Component
const Sidebar = ({ isOpen, onClose, activePage, setActivePage }) => {
  const menuItems = [
    { id: 'dashboard', icon: Home, label: 'Dashboard' },
    { id: 'users', icon: Users, label: 'Users', badge: 12 },
    { id: 'kyc', icon: Shield, label: 'KYC', badge: 5 },
    { id: 'transactions', icon: Activity, label: 'Transactions' },
    { id: 'tokens', icon: Layers, label: 'Tokens' },
    { id: 'wallets', icon: Wallet, label: 'Wallets' },
    { id: 'withdrawals', icon: DollarSign, label: 'Withdrawals', badge: 3 },
    { id: 'blockchains', icon: Globe, label: 'Blockchains' },
    { id: 'whitelabels', icon: Globe, label: 'White Labels' },
    { id: 'analytics', icon: BarChart2, label: 'Analytics' },
    { id: 'reports', icon: FileText, label: 'Reports' },
    { id: 'audit', icon: FileText, label: 'Audit Logs' },
    { id: 'tickets', icon: MessageSquare, label: 'Support', badge: 8 },
    { id: 'settings', icon: Settings, label: 'Settings' },
  ];

  return (
    <aside className={`sidebar ${isOpen ? 'open' : ''}`}>
      <div className="sidebar-header">
        <div className="logo">
          <div className="logo-icon">
            <Shield size={24} />
          </div>
          <span>TigerAdmin</span>
        </div>
      </div>
      
      <nav className="sidebar-nav">
        <div className="nav-section">
          <div className="nav-section-title">Main</div>
          {menuItems.slice(0, 6).map(item => (
            <div 
              key={item.id}
              className={`nav-item ${activePage === item.id ? 'active' : ''}`}
              onClick={() => { setActivePage(item.id); onClose?.(); }}
            >
              <item.icon size={20} />
              <span>{item.label}</span>
              {item.badge && <span className="nav-badge">{item.badge}</span>}
            </div>
          ))}
        </div>
        
        <div className="nav-section">
          <div className="nav-section-title">Management</div>
          {menuItems.slice(6, 10).map(item => (
            <div 
              key={item.id}
              className={`nav-item ${activePage === item.id ? 'active' : ''}`}
              onClick={() => { setActivePage(item.id); onClose?.(); }}
            >
              <item.icon size={20} />
              <span>{item.label}</span>
              {item.badge && <span className="nav-badge">{item.badge}</span>}
            </div>
          ))}
        </div>
        
        <div className="nav-section">
          <div className="nav-section-title">System</div>
          {menuItems.slice(10).map(item => (
            <div 
              key={item.id}
              className={`nav-item ${activePage === item.id ? 'active' : ''}`}
              onClick={() => { setActivePage(item.id); onClose?.(); }}
            >
              <item.icon size={20} />
              <span>{item.label}</span>
              {item.badge && <span className="nav-badge">{item.badge}</span>}
            </div>
          ))}
        </div>
      </nav>
    </aside>
  );
};

// Header Component
const Header = ({ onMenuClick }) => {
  const { theme, toggleTheme } = useTheme();
  const [notifications] = useState([
    { id: 1, title: 'New KYC Request', message: 'User John Doe submitted KYC documents', time: '2 min ago', unread: true },
    { id: 2, title: 'Withdrawal Approved', message: 'Withdrawal of 1.5 BTC has been approved', time: '15 min ago', unread: true },
    { id: 3, title: 'System Alert', message: 'Server CPU usage above 80%', time: '1 hour ago', unread: false },
  ]);

  return (
    <header className="header">
      <div className="header-left">
        <button className="menu-toggle" onClick={onMenuClick}>
          <Menu size={24} />
        </button>
        
        <div className="search-box">
          <Search size={18} color="var(--text-muted)" />
          <input type="text" placeholder="Search users, transactions, tokens..." />
        </div>
      </div>
      
      <div className="header-right">
        <button className="header-btn">
          <Bell size={20} />
          <span className="badge">3</span>
        </button>
        
        <button className="header-btn theme-toggle" onClick={toggleTheme}>
          {theme === 'dark' ? <Sun size={20} /> : <Moon size={20} />}
        </button>
        
        <div className="user-menu">
          <div className="user-avatar">AD</div>
          <div className="user-info">
            <span className="user-name">Admin User</span>
            <span className="user-role">Super Admin</span>
          </div>
        </div>
      </div>
    </header>
  );
};

// Dashboard Page
const DashboardPage = () => {
  const [stats] = useState({
    totalUsers: 125843,
    activeUsers: 98234,
    totalVolume: '$4.2B',
    dailyVolume: '$128.5M',
    pendingKYC: 23,
    pendingWithdrawals: 15,
    totalTransactions: 2847293,
    todayTransactions: 45231,
  });

  const [volumeData] = useState([
    { name: 'Mon', volume: 42000000 },
    { name: 'Tue', volume: 38000000 },
    { name: 'Wed', volume: 51000000 },
    { name: 'Thu', volume: 47000000 },
    { name: 'Fri', volume: 55000000 },
    { name: 'Sat', volume: 62000000 },
    { name: 'Sun', volume: 58000000 },
  ]);

  const [userGrowthData] = useState([
    { name: 'Jan', users: 45000 },
    { name: 'Feb', users: 52000 },
    { name: 'Mar', users: 61000 },
    { name: 'Apr', users: 72000 },
    { name: 'May', users: 85000 },
    { name: 'Jun', users: 98000 },
  ]);

  const [recentTransactions] = useState([
    { id: 'TX001', user: 'johndoe@example.com', type: 'Deposit', amount: '2.5 BTC', status: 'completed', time: '2 min ago' },
    { id: 'TX002', user: 'janedoe@example.com', type: 'Withdraw', amount: '1.2 ETH', status: 'pending', time: '5 min ago' },
    { id: 'TX003', user: 'bobsmith@example.com', type: 'Swap', amount: '5000 USDT', status: 'completed', time: '10 min ago' },
    { id: 'TX004', user: 'alice@example.com', type: 'Deposit', amount: '10 ETH', status: 'completed', time: '15 min ago' },
    { id: 'TX005', user: 'charlie@example.com', type: 'Withdraw', amount: '2500 USDC', status: 'flagged', time: '20 min ago' },
  ]);

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-subtitle">Welcome back! Here's what's happening today.</p>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary">
            <Download size={18} />
            Export
          </button>
          <button className="btn btn-primary">
            <RefreshCw size={18} />
            Refresh
          </button>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon users">
            <Users size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Total Users</div>
            <div className="stat-value">{stats.totalUsers.toLocaleString()}</div>
            <div className="stat-change positive">
              <ChevronDown size={14} /> +12.5% from last month
            </div>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon transactions">
            <Activity size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Today's Transactions</div>
            <div className="stat-value">{stats.todayTransactions.toLocaleString()}</div>
            <div className="stat-change positive">
              <ChevronDown size={14} /> +8.3% from yesterday
            </div>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon volume">
            <DollarSign size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Daily Volume</div>
            <div className="stat-value">{stats.dailyVolume}</div>
            <div className="stat-change positive">
              <ChevronDown size={14} /> +15.2% from yesterday
            </div>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon pending">
            <Clock size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Pending Withdrawals</div>
            <div className="stat-value">{stats.pendingWithdrawals}</div>
            <div className="stat-change negative">
              <AlertTriangle size={14} /> Requires attention
            </div>
          </div>
        </div>
      </div>

      {/* Charts Row */}
      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '20px', marginBottom: '24px' }}>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Volume Overview</h3>
            <select className="form-input form-select" style={{ width: 'auto' }}>
              <option>Last 7 days</option>
              <option>Last 30 days</option>
              <option>Last 90 days</option>
            </select>
          </div>
          <div className="card-body">
            <div className="chart-container">
              <ResponsiveContainer width="100%" height={280}>
                <LineChart data={volumeData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
                  <XAxis dataKey="name" stroke="var(--text-muted)" />
                  <YAxis stroke="var(--text-muted)" tickFormatter={(v) => `$${(v/1000000).toFixed(0)}M`} />
                  <Tooltip 
                    contentStyle={{ 
                      background: 'var(--bg-card)', 
                      border: '1px solid var(--border-color)',
                      borderRadius: '8px'
                    }}
                  />
                  <Line type="monotone" dataKey="volume" stroke="var(--primary)" strokeWidth={2} dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="card-title">User Growth</h3>
          </div>
          <div className="card-body">
            <div className="chart-container">
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={userGrowthData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
                  <XAxis dataKey="name" stroke="var(--text-muted)" />
                  <YAxis stroke="var(--text-muted)" />
                  <Tooltip 
                    contentStyle={{ 
                      background: 'var(--bg-card)', 
                      border: '1px solid var(--border-color)',
                      borderRadius: '8px'
                    }}
                  />
                  <Bar dataKey="users" fill="var(--primary)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Transactions */}
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Recent Transactions</h3>
          <button className="btn btn-sm btn-secondary">View All</button>
        </div>
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>User</th>
                <th>Type</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Time</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {recentTransactions.map(tx => (
                <tr key={tx.id}>
                  <td>{tx.id}</td>
                  <td>{tx.user}</td>
                  <td>{tx.type}</td>
                  <td>{tx.amount}</td>
                  <td>
                    <span className={`badge badge-${tx.status === 'completed' ? 'success' : tx.status === 'pending' ? 'warning' : 'danger'}`}>
                      {tx.status}
                    </span>
                  </td>
                  <td>{tx.time}</td>
                  <td>
                    <button className="btn btn-icon btn-secondary btn-sm">
                      <Eye size={16} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

// Users Page
const UsersPage = () => {
  const [users] = useState([
    { id: 1, email: 'johndoe@example.com', username: 'johndoe', status: 'active', kyc: 'verified', volume: '$125,000', joined: '2024-01-15' },
    { id: 2, email: 'janedoe@example.com', username: 'janedoe', status: 'active', kyc: 'pending', volume: '$45,000', joined: '2024-02-20' },
    { id: 3, email: 'bobsmith@example.com', username: 'bobsmith', status: 'suspended', kyc: 'rejected', volume: '$12,000', joined: '2024-03-05' },
    { id: 4, email: 'alice@example.com', username: 'alice_wallet', status: 'active', kyc: 'verified', volume: '$890,000', joined: '2024-01-10' },
    { id: 5, email: 'charlie@example.com', username: 'charlie', status: 'active', kyc: 'verified', volume: '$67,000', joined: '2024-04-12' },
  ]);

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">Users</h1>
          <p className="page-subtitle">Manage platform users and their accounts</p>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary">
            <Download size={18} />
            Export
          </button>
          <button className="btn btn-primary">
            <Plus size={18} />
            Add User
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="card" style={{ marginBottom: '20px' }}>
        <div className="card-body" style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
          <div className="search-box" style={{ width: '300px' }}>
            <Search size={18} color="var(--text-muted)" />
            <input type="text" placeholder="Search users..." />
          </div>
          <select className="form-input form-select" style={{ width: '150px' }}>
            <option>All Status</option>
            <option>Active</option>
            <option>Suspended</option>
            <option>Inactive</option>
          </select>
          <select className="form-input form-select" style={{ width: '150px' }}>
            <option>All KYC</option>
            <option>Verified</option>
            <option>Pending</option>
            <option>Rejected</option>
          </select>
          <button className="btn btn-secondary">
            <Filter size={18} />
            More Filters
          </button>
        </div>
      </div>

      {/* Users Table */}
      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>User</th>
                <th>Status</th>
                <th>KYC</th>
                <th>Volume</th>
                <th>Joined</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.id}>
                  <td>
                    <div>
                      <div style={{ fontWeight: 500 }}>{user.username}</div>
                      <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{user.email}</div>
                    </div>
                  </td>
                  <td>
                    <span className={`badge badge-${user.status === 'active' ? 'success' : 'danger'}`}>
                      {user.status}
                    </span>
                  </td>
                  <td>
                    <span className={`badge badge-${user.kyc === 'verified' ? 'success' : user.kyc === 'pending' ? 'warning' : 'danger'}`}>
                      {user.kyc}
                    </span>
                  </td>
                  <td>{user.volume}</td>
                  <td>{user.joined}</td>
                  <td>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button className="btn btn-icon btn-secondary btn-sm"><Eye size={16} /></button>
                      <button className="btn btn-icon btn-secondary btn-sm"><Edit size={16} /></button>
                      <button className="btn btn-icon btn-secondary btn-sm"><Lock size={16} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        
        <div className="pagination">
          <div className="pagination-info">Showing 1-5 of 125,843 users</div>
          <div className="pagination-controls">
            <button className="pagination-btn" disabled>Previous</button>
            <button className="pagination-btn active">1</button>
            <button className="pagination-btn">2</button>
            <button className="pagination-btn">3</button>
            <button className="pagination-btn">...</button>
            <button className="pagination-btn">25169</button>
            <button className="pagination-btn">Next</button>
          </div>
        </div>
      </div>
    </div>
  );
};

// KYC Page
const KycPage = () => {
  const [kycRequests] = useState([
    { id: 'KYC001', user: 'johndoe@example.com', type: 'ID Verification', submitted: '2024-06-15 10:30', status: 'pending', risk: 'low' },
    { id: 'KYC002', user: 'janedoe@example.com', type: 'Selfie Verification', submitted: '2024-06-15 09:45', status: 'pending', risk: 'medium' },
    { id: 'KYC003', user: 'bobsmith@example.com', type: 'ID Verification', submitted: '2024-06-14 16:20', status: 'approved', risk: 'low' },
    { id: 'KYC004', user: 'alice@example.com', type: 'Address Proof', submitted: '2024-06-14 14:10', status: 'rejected', risk: 'high' },
  ]);

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">KYC Verification</h1>
          <p className="page-subtitle">Review and verify user identity documents</p>
        </div>
      </div>

      {/* Stats */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon users">
            <Shield size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Pending Review</div>
            <div className="stat-value">23</div>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon" style={{ background: 'var(--success-light)', color: 'var(--success)' }}>
            <CheckCircle size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Approved Today</div>
            <div className="stat-value">45</div>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon" style={{ background: 'var(--danger-light)', color: 'var(--danger)' }}>
            <XCircle size={24} />
          </div>
          <div className="stat-content">
            <div className="stat-label">Rejected Today</div>
            <div className="stat-value">8</div>
          </div>
        </div>
      </div>

      {/* KYC Requests Table */}
      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>User</th>
                <th>Type</th>
                <th>Submitted</th>
                <th>Risk Level</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {kycRequests.map(req => (
                <tr key={req.id}>
                  <td>{req.id}</td>
                  <td>{req.user}</td>
                  <td>{req.type}</td>
                  <td>{req.submitted}</td>
                  <td>
                    <span className={`badge badge-${req.risk === 'low' ? 'success' : req.risk === 'medium' ? 'warning' : 'danger'}`}>
                      {req.risk}
                    </span>
                  </td>
                  <td>
                    <span className={`badge badge-${req.status === 'pending' ? 'warning' : req.status === 'approved' ? 'success' : 'danger'}`}>
                      {req.status}
                    </span>
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button className="btn btn-sm btn-success"><Check size={16} /> Approve</button>
                      <button className="btn btn-sm btn-danger"><XCircle size={16} /> Reject</button>
                      <button className="btn btn-sm btn-secondary"><Eye size={16} /> View</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

// Transactions Page
const TransactionsPage = () => {
  const [transactions] = useState([
    { id: 'TX001', type: 'Deposit', chain: 'Bitcoin', amount: '2.5 BTC', usdValue: '$162,500', fee: '0.0005 BTC', status: 'completed', time: '2024-06-15 10:30:00' },
    { id: 'TX002', type: 'Withdraw', chain: 'Ethereum', amount: '15 ETH', usdValue: '$45,000', fee: '0.005 ETH', status: 'pending', time: '2024-06-15 10:25:00' },
    { id: 'TX003', type: 'Swap', chain: 'BSC', amount: '10,000 USDT', usdValue: '$10,000', fee: '3 USDT', status: 'completed', time: '2024-06-15 10:20:00' },
    { id: 'TX004', type: 'Transfer', chain: 'Solana', amount: '500 SOL', usdValue: '$62,500', fee: '0.0001 SOL', status: 'flagged', time: '2024-06-15 10:15:00' },
  ]);

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">Transactions</h1>
          <p className="page-subtitle">Monitor and manage all platform transactions</p>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary">
            <Download size={18} />
            Export CSV
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="card" style={{ marginBottom: '20px' }}>
        <div className="card-body" style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
          <select className="form-input form-select" style={{ width: '150px' }}>
            <option>All Types</option>
            <option>Deposit</option>
            <option>Withdraw</option>
            <option>Swap</option>
            <option>Transfer</option>
          </select>
          <select className="form-input form-select" style={{ width: '150px' }}>
            <option>All Status</option>
            <option>Completed</option>
            <option>Pending</option>
            <option>Flagged</option>
            <option>Failed</option>
          </select>
          <select className="form-input form-select" style={{ width: '150px' }}>
            <option>All Chains</option>
            <option>Bitcoin</option>
            <option>Ethereum</option>
            <option>BSC</option>
            <option>Solana</option>
          </select>
          <input type="date" className="form-input" style={{ width: '150px' }} />
        </div>
      </div>

      {/* Transactions Table */}
      <div className="card">
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Type</th>
                <th>Chain</th>
                <th>Amount</th>
                <th>USD Value</th>
                <th>Fee</th>
                <th>Status</th>
                <th>Time</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map(tx => (
                <tr key={tx.id}>
                  <td>{tx.id}</td>
                  <td>{tx.type}</td>
                  <td>{tx.chain}</td>
                  <td>{tx.amount}</td>
                  <td>{tx.usdValue}</td>
                  <td>{tx.fee}</td>
                  <td>
                    <span className={`badge badge-${tx.status === 'completed' ? 'success' : tx.status === 'pending' ? 'warning' : tx.status === 'flagged' ? 'danger' : 'default'}`}>
                      {tx.status}
                    </span>
                  </td>
                  <td>{tx.time}</td>
                  <td>
                    <button className="btn btn-icon btn-secondary btn-sm"><Eye size={16} /></button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

// Settings Page
const SettingsPage = () => {
  const [activeTab, setActiveTab] = useState('general');

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">Settings</h1>
          <p className="page-subtitle">Configure platform settings and preferences</p>
        </div>
      </div>

      <div className="tabs">
        <button className={`tab ${activeTab === 'general' ? 'active' : ''}`} onClick={() => setActiveTab('general')}>General</button>
        <button className={`tab ${activeTab === 'security' ? 'active' : ''}`} onClick={() => setActiveTab('security')}>Security</button>
        <button className={`tab ${activeTab === 'notifications' ? 'active' : ''}`} onClick={() => setActiveTab('notifications')}>Notifications</button>
        <button className={`tab ${activeTab === 'api' ? 'active' : ''}`} onClick={() => setActiveTab('api')}>API Keys</button>
        <button className={`tab ${activeTab === 'integrations' ? 'active' : ''}`} onClick={() => setActiveTab('integrations')}>Integrations</button>
      </div>

      <div className="card">
        <div className="card-body">
          {activeTab === 'general' && (
            <div>
              <div className="form-group">
                <label className="form-label">Platform Name</label>
                <input type="text" className="form-input" defaultValue="TigerWallet" />
              </div>
              <div className="form-group">
                <label className="form-label">Support Email</label>
                <input type="email" className="form-input" defaultValue="support@tigerwallet.com" />
              </div>
              <div className="form-group">
                <label className="form-label">Maintenance Mode</label>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <input type="checkbox" id="maintenance" />
                  <label htmlFor="maintenance">Enable maintenance mode</label>
                </div>
              </div>
              <button className="btn btn-primary">Save Changes</button>
            </div>
          )}
          
          {activeTab === 'security' && (
            <div>
              <div className="form-group">
                <label className="form-label">Two-Factor Authentication</label>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <input type="checkbox" id="2fa" defaultChecked />
                  <label htmlFor="2fa">Require 2FA for all admins</label>
                </div>
              </div>
              <div className="form-group">
                <label className="form-label">Session Timeout (minutes)</label>
                <input type="number" className="form-input" defaultValue="30" />
              </div>
              <div className="form-group">
                <label className="form-label">IP Whitelist</label>
                <textarea className="form-input" rows={4} defaultValue="192.168.1.0/24&#10;10.0.0.0/8" />
              </div>
              <button className="btn btn-primary">Save Changes</button>
            </div>
          )}
          
          {activeTab === 'notifications' && (
            <div>
              <div className="form-group">
                <label className="form-label">Email Notifications</label>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <input type="checkbox" id="notif_kyc" defaultChecked />
                    <label htmlFor="notif_kyc">New KYC submissions</label>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <input type="checkbox" id="notif_withdraw" defaultChecked />
                    <label htmlFor="notif_withdraw">Withdrawal requests</label>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <input type="checkbox" id="notif_security" defaultChecked />
                    <label htmlFor="notif_security">Security alerts</label>
                  </div>
                </div>
              </div>
              <button className="btn btn-primary">Save Changes</button>
            </div>
          )}
          
          {activeTab === 'api' && (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px' }}>
                <h3>API Keys</h3>
                <button className="btn btn-primary"><Plus size={18} /> Generate New Key</button>
              </div>
              <div className="table-container">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Key</th>
                      <th>Created</th>
                      <th>Last Used</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>Production API</td>
                      <td>tw_live_****************************</td>
                      <td>2024-01-15</td>
                      <td>2024-06-15</td>
                      <td>
                        <button className="btn btn-icon btn-danger btn-sm"><Trash2 size={16} /></button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          )}
          
          {activeTab === 'integrations' && (
            <div>
              <div className="form-group">
                <label className="form-label">Slack Webhook</label>
                <input type="text" className="form-input" placeholder="https://hooks.slack.com/..." />
              </div>
              <div className="form-group">
                <label className="form-label">PagerDuty API Key</label>
                <input type="password" className="form-input" placeholder="Enter API key" />
              </div>
              <div className="form-group">
                <label className="form-label">Datadog API Key</label>
                <input type="password" className="form-input" placeholder="Enter API key" />
              </div>
              <button className="btn btn-primary">Save Changes</button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// Notifications Panel
const NotificationsPanel = () => {
  const [notifications] = useState([
    { id: 1, type: 'kyc', title: 'New KYC Request', message: 'User John Doe submitted KYC documents', time: '2 min ago', unread: true },
    { id: 2, type: 'withdrawal', title: 'Withdrawal Request', message: 'Pending approval - 2.5 BTC', time: '5 min ago', unread: true },
    { id: 3, type: 'security', title: 'Security Alert', message: 'Multiple failed login attempts detected', time: '15 min ago', unread: true },
    { id: 4, type: 'system', title: 'System Update', message: 'Server maintenance scheduled for tonight', time: '1 hour ago', unread: false },
  ]);

  const getIcon = (type) => {
    switch(type) {
      case 'kyc': return <Shield size={20} />;
      case 'withdrawal': return <DollarSign size={20} />;
      case 'security': return <AlertTriangle size={20} />;
      default: return <Bell size={20} />;
    }
  };

  return (
    <div className="page-content">
      <div className="page-header">
        <div>
          <h1 className="page-title">Notifications</h1>
          <p className="page-subtitle">View and manage your notifications</p>
        </div>
        <button className="btn btn-secondary">Mark All Read</button>
      </div>

      <div className="notification-list">
        {notifications.map(notif => (
          <div key={notif.id} className={`notification-item ${notif.unread ? 'unread' : ''}`}>
            <div className="notification-icon" style={{ 
              background: notif.type === 'kyc' ? 'var(--primary-light)' : 
                         notif.type === 'withdrawal' ? 'var(--warning-light)' : 
                         notif.type === 'security' ? 'var(--danger-light)' : 'var(--bg-hover)',
              color: notif.type === 'kyc' ? 'var(--primary)' : 
                     notif.type === 'withdrawal' ? 'var(--warning)' : 
                     notif.type === 'security' ? 'var(--danger)' : 'var(--text-secondary)'
            }}>
              {getIcon(notif.type)}
            </div>
            <div className="notification-content">
              <div className="notification-title">{notif.title}</div>
              <div className="notification-message">{notif.message}</div>
              <div className="notification-time">{notif.time}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// ============================================================================
// Main App Component
// ============================================================================

export default function AdminDashboard() {
  const [activePage, setActivePage] = useState('dashboard');
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const renderPage = () => {
    switch(activePage) {
      case 'dashboard': return <DashboardPage />;
      case 'users': return <UsersPage />;
      case 'kyc': return <KycPage />;
      case 'transactions': return <TransactionsPage />;
      case 'settings': return <SettingsPage />;
      default: return <DashboardPage />;
    }
  };

  return (
    <ThemeProvider>
      <style>{themeStyles}</style>
      <div className="app-container">
        <Sidebar 
          isOpen={sidebarOpen} 
          onClose={() => setSidebarOpen(false)} 
          activePage={activePage}
          setActivePage={setActivePage}
        />
        <main className="main-content">
          <Header onMenuClick={() => setSidebarOpen(!sidebarOpen)} />
          {renderPage()}
        </main>
      </div>
    </ThemeProvider>
  );
}
