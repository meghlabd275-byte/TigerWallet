// Sidebar Component
import React from 'react';
import { NavLink } from 'react-router-dom';
import './Sidebar.css';

const Sidebar: React.FC = () => {
  const menuItems = [
    { path: '/', icon: '🏠', label: 'Home' },
    { path: '/wallet', icon: '💼', label: 'Wallet' },
    { path: '/send', icon: '📤', label: 'Send' },
    { path: '/receive', icon: '📥', label: 'Receive' },
    { path: '/swap', icon: '🔄', label: 'Swap' },
    { path: '/dapps', icon: '🌐', label: 'DApps' },
    { path: '/settings', icon: '⚙️', label: 'Settings' },
  ];

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="logo">
          <span className="logo-icon">🐯</span>
          <span className="logo-text">TigerWallet</span>
        </div>
      </div>

      <nav className="sidebar-nav">
        <ul className="nav-list">
          {menuItems.map((item) => (
            <li key={item.path}>
              <NavLink
                to={item.path}
                className={({ isActive }) =>
                  `nav-item ${isActive ? 'active' : ''}`
                }
              >
                <span className="nav-icon">{item.icon}</span>
                <span className="nav-label">{item.label}</span>
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      <div className="sidebar-footer">
        <div className="version">v1.0.0</div>
      </div>
    </aside>
  );
};

export default Sidebar;
