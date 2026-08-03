// TigerWallet Master - Web Application
import React from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './index.css';

// Theme Provider
const ThemeProvider = ({ children }) => {
  const [isDarkMode, setIsDarkMode] = React.useState(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('master_wallet_theme');
      return stored ? stored === 'dark' : true;
    }
    return true;
  });

  React.useEffect(() => {
    document.documentElement.setAttribute('data-theme', isDarkMode ? 'dark' : 'light');
    localStorage.setItem('master_wallet_theme', isDarkMode ? 'dark' : 'light');
  }, [isDarkMode]);

  const toggleTheme = () => setIsDarkMode(!isDarkMode);

  return (
    <ThemeContext.Provider value={{ isDarkMode, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

const ThemeContext = React.createContext({
  isDarkMode: true,
  toggleTheme: () => {}
});

export { ThemeContext, ThemeProvider };

const container = document.getElementById('root');
const root = createRoot(container);
root.render(
  <ThemeProvider>
    <App />
  </ThemeProvider>
);
