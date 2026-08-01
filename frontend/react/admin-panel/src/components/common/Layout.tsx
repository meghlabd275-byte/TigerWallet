// TigerWallet Layout Component - Admin Panel Layout with Theme Support

import React, { ReactNode } from 'react';
import Header from './Header';
import Sidebar from './Sidebar';
import { useTheme } from './ThemeProvider';
import './Layout.css';

interface LayoutProps {
  children: ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const { theme } = useTheme();
  const [sidebarOpen, setSidebarOpen] = React.useState(true);

  return (
    <div className={`layout layout--${theme}`}>
      <Sidebar isOpen={sidebarOpen} onToggle={() => setSidebarOpen(!sidebarOpen)} />
      <div className={`layout__main ${sidebarOpen ? 'layout__main--sidebar-open' : ''}`}>
        <Header onMenuClick={() => setSidebarOpen(!sidebarOpen)} />
        <main className="layout__content">
          {children}
        </main>
      </div>
    </div>
  );
};

export default Layout;
