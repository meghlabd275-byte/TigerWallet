// Header Component
import React from 'react';
import { useTheme } from '../stores/ThemeStore';
import './Header.css';

interface HeaderProps {
  onLogout: () => void;
}

const Header: React.FC<HeaderProps> = ({ onLogout }) => {
  const { theme, toggleTheme } = useTheme();

  return (
    <header className="header">
      <div className="header-left">
        <h2>Welcome Back</h2>
      </div>

      <div className="header-right">
        <button className="header-btn" onClick={toggleTheme} title="Toggle Theme">
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
        
        <button className="header-btn" title="Notifications">
          🔔
        </button>

        <button className="user-btn" onClick={onLogout}>
          <div className="user-avatar">👤</div>
        </button>
      </div>
    </header>
  );
};

export default Header;
