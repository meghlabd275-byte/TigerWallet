/**
 * TigerWallet White Label Admin - Main Application
 * Production-ready with full functionality and theme support
 */

import React, { useState, useEffect } from 'react';
import { 
  Container, Box, AppBar, Toolbar, Typography, Drawer, List, ListItem, 
  ListItemButton, ListItemIcon, ListItemText, IconButton, useTheme as useMUITheme,
  createTheme, ThemeProvider as MUIThemeProvider, CssBaseline, Divider,
  Chip, Avatar, Menu, MenuItem, Badge
} from '@mui/material';
import {
  Menu as MenuIcon, Dashboard, People, Business, Store, Settings,
  Notifications, Security, Brightness4, Brightness7, ChevronLeft,
  AccountTree, Poll, Payment, Layers, SwapHoriz, Hexagon
} from '@mui/icons-material';
import { useTheme, themeColors } from './context/ThemeContext';
import WhiteLabelDashboard from './pages/WhiteLabelDashboard';
import AdminManagement from './pages/AdminManagement';
import ProductManagement from './pages/ProductManagement';
import TradingPairsPage from './pages/TradingPairsPage';
import BlockchainManagement from './pages/BlockchainManagement';
import AuditLogsPage from './pages/AuditLogsPage';
import NotificationsPage from './pages/NotificationsPage';
import SettingsPage from './pages/SettingsPage';
import { api } from './services/api';

const drawerWidth = 260;

interface NavItem {
  label: string;
  icon: React.ReactNode;
  component: React.ReactNode;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', icon: <Dashboard />, component: <WhiteLabelDashboard /> },
  { label: 'Admins', icon: <People />, component: <AdminManagement /> },
  { label: 'Products', icon: <Store />, component: <ProductManagement /> },
  { label: 'Trading Pairs', icon: <SwapHoriz />, component: <TradingPairsPage /> },
  { label: 'Blockchains', icon: <Hexagon />, component: <BlockchainManagement /> },
  { label: 'Audit Logs', icon: <Security />, component: <AuditLogsPage /> },
  { label: 'Notifications', icon: <Notifications />, component: <NotificationsPage /> },
  { label: 'Settings', icon: <Settings />, component: <SettingsPage /> },
];

const App: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [currentPage, setCurrentPage] = useState(0);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  
  // Apply theme to document
  useEffect(() => {
    const colors = theme === 'dark' ? themeColors.dark : themeColors.light;
    document.documentElement.style.setProperty('--bg-primary', colors.background);
    document.documentElement.style.setProperty('--bg-secondary', colors.backgroundSecondary);
    document.documentElement.style.setProperty('--text-primary', colors.text);
    document.documentElement.style.setProperty('--text-secondary', colors.textSecondary);
    document.documentElement.style.setProperty('--border-color', colors.border);
    document.body.style.backgroundColor = colors.background;
    document.body.style.color = colors.text;
  }, [theme]);

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const handleMenu = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const currentComponent = navItems[currentPage]?.component || <WhiteLabelDashboard />;

  const drawer = (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Toolbar sx={{ justifyContent: 'center', py: 2 }}>
        <Typography variant="h5" sx={{ fontWeight: 'bold', color: '#f59e0b' }}>
          🐯 TigerWallet
        </Typography>
      </Toolbar>
      <Divider />
      <List sx={{ flex: 1, px: 1, py: 2 }}>
        {navItems.map((item, index) => (
          <ListItem key={item.label} disablePadding sx={{ mb: 0.5 }}>
            <ListItemButton
              selected={currentPage === index}
              onClick={() => { setCurrentPage(index); setMobileOpen(false); }}
              sx={{
                borderRadius: 2,
                '&.Mui-selected': {
                  backgroundColor: theme === 'dark' ? 'rgba(59, 130, 246, 0.2)' : 'rgba(59, 130, 246, 0.1)',
                  '&:hover': {
                    backgroundColor: theme === 'dark' ? 'rgba(59, 130, 246, 0.3)' : 'rgba(59, 130, 246, 0.2)',
                  }
                }
              }}
            >
              <ListItemIcon sx={{ 
                color: currentPage === index ? '#3b82f6' : theme === 'dark' ? '#94a3b8' : '#64748b',
                minWidth: 40
              }}>
                {item.icon}
              </ListItemIcon>
              <ListItemText 
                primary={item.label}
                primaryTypographyProps={{
                  fontWeight: currentPage === index ? 600 : 400,
                  color: theme === 'dark' ? '#f1f5f9' : '#0f172a'
                }}
              />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <CssBaseline />
      
      {/* App Bar */}
      <AppBar
        position="fixed"
        sx={{
          width: { sm: `calc(100% - ${drawerWidth}px)` },
          ml: { sm: `${drawerWidth}px` },
          bgcolor: theme === 'dark' ? '#1a1a2e' : '#ffffff',
          color: theme === 'dark' ? '#f1f5f9' : '#0f172a',
          boxShadow: theme === 'dark' ? 'none' : '0 1px 3px rgba(0,0,0,0.1)',
          borderBottom: `1px solid ${theme === 'dark' ? '#2d3748' : '#e2e8f0'}`
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label="open drawer"
            edge="start"
            onClick={handleDrawerToggle}
            sx={{ mr: 2, display: { sm: 'none' } }}
          >
            <MenuIcon />
          </IconButton>
          
          <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
            {navItems[currentPage]?.label || 'Dashboard'}
          </Typography>
          
          {/* Theme Toggle */}
          <IconButton 
            onClick={toggleTheme} 
            color="inherit"
            sx={{ mr: 1 }}
          >
            {theme === 'dark' ? <Brightness7 /> : <Brightness4 />}
          </IconButton>
          
          {/* Notifications */}
          <IconButton color="inherit" sx={{ mr: 1 }}>
            <Badge badgeContent={3} color="error">
              <Notifications />
            </Badge>
          </IconButton>
          
          {/* User Menu */}
          <Chip
            avatar={<Avatar sx={{ bgcolor: '#f59e0b' }}>A</Avatar>}
            label="Admin"
            onClick={handleMenu}
            sx={{ 
              bgcolor: theme === 'dark' ? '#2d3748' : '#f1f5f9',
              color: theme === 'dark' ? '#f1f5f9' : '#0f172a',
              cursor: 'pointer',
              '&:hover': {
                bgcolor: theme === 'dark' ? '#374151' : '#e2e8f0'
              }
            }}
          />
          <Menu
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={handleClose}
          >
            <MenuItem onClick={handleClose}>Profile</MenuItem>
            <MenuItem onClick={handleClose}>Account Settings</MenuItem>
            <Divider />
            <MenuItem onClick={handleClose}>Logout</MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>
      
      {/* Drawer */}
      <Box
        component="nav"
        sx={{ width: { sm: drawerWidth }, flexShrink: { sm: 0 } }}
      >
        {/* Mobile Drawer */}
        <Drawer
          variant="temporary"
          open={mobileOpen}
          onClose={handleDrawerToggle}
          ModalProps={{ keepMounted: true }}
          sx={{
            display: { xs: 'block', sm: 'none' },
            '& .MuiDrawer-paper': { 
              boxSizing: 'border-box', 
              width: drawerWidth,
              bgcolor: theme === 'dark' ? '#1a1a2e' : '#ffffff',
              borderRight: `1px solid ${theme === 'dark' ? '#2d3748' : '#e2e8f0'}`
            },
          }}
        >
          {drawer}
        </Drawer>
        
        {/* Desktop Drawer */}
        <Drawer
          variant="permanent"
          sx={{
            display: { xs: 'none', sm: 'block' },
            '& .MuiDrawer-paper': { 
              boxSizing: 'border-box', 
              width: drawerWidth,
              bgcolor: theme === 'dark' ? '#1a1a2e' : '#ffffff',
              borderRight: `1px solid ${theme === 'dark' ? '#2d3748' : '#e2e8f0'}`
            },
          }}
          open
        >
          {drawer}
        </Drawer>
      </Box>
      
      {/* Main Content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          width: { sm: `calc(100% - ${drawerWidth}px)` },
          minHeight: '100vh',
          bgcolor: theme === 'dark' ? '#0f0f23' : '#f8fafc',
          mt: '64px'
        }}
      >
        {currentComponent}
      </Box>
    </Box>
  );
};

export default App;
