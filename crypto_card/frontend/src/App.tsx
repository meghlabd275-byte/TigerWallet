import React from 'react';
import { ThemeProvider } from './hooks/useTheme';
import CardDashboard from './pages/CardDashboard';

const App: React.FC = () => {
  return (
    <ThemeProvider>
      <CardDashboard />
    </ThemeProvider>
  );
};

export default App;
