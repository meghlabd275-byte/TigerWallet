import React, { useState, createContext, useContext, useEffect } from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import {
  AppBar, Toolbar, Typography, Drawer, List, ListItem, ListItemIcon, ListItemText,
  Box, IconButton, Avatar, Menu, MenuItem, Divider, useTheme, Badge,
  CssBaseline, ThemeProvider, createTheme
} from '@mui/material'
import {
  Dashboard, People, Pool, Hub, Receipt, ShowChart, SmartToy,
  AccountTree, HubOutlined, AttachMoney, Security, Analytics, Settings,
  Menu as MenuIcon, Notifications, Brightness7, Brightness4, DarkMode, LightMode,
  Support
} from '@mui/icons-material'

// Theme Context
interface ThemeContextType {
  darkMode: boolean;
  toggleTheme: () => void;
}

export const ThemeContext = createContext<ThemeContextType>({
  darkMode: false,
  toggleTheme: () => {}
});

export const useThemeContext = () => useContext(ThemeContext);
export { lightTheme, darkTheme };

const drawerWidth = 260

const menuItems = [
  { text: 'Dashboard', icon: <Dashboard />, path: '/dashboard' },
  { text: 'Users', icon: <People />, path: '/users' },
  { text: 'Pools', icon: <Pool />, path: '/pools' },
  { text: 'Bridges', icon: <Hub />, path: '/bridges' },
  { text: 'Transactions', icon: <Receipt />, path: '/transactions' },
  { text: 'Market Maker', icon: <ShowChart />, path: '/market-maker' },
  { text: 'Bots', icon: <SmartToy />, path: '/bots' },
  { text: 'Chains', icon: <AccountTree />, path: '/chains' },
  { text: 'DEXs', icon: <HubOutlined />, path: '/dexs' },
  { text: 'Fees', icon: <AttachMoney />, path: '/fees' },
  { text: 'Treasury', icon: <AttachMoney />, path: '/treasury' },
  { text: 'Support', icon: <Support />, path: '/support' },
  { text: 'Security', icon: <Security />, path: '/security' },
  { text: 'Analytics', icon: <Analytics />, path: '/analytics' },
  { text: 'Settings', icon: <Settings />, path: '/settings' },
]

const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#f97316' },
    secondary: { main: '#1e293b' },
    background: { default: '#f5f5f5', paper: '#ffffff' },
  },
  typography: { fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif' },
  components: {
    MuiAppBar: { styleOverrides: { root: { backgroundColor: '#ffffff', color: '#000' } } },
    MuiDrawer: { styleOverrides: { paper: { backgroundColor: '#ffffff' } } },
  }
});

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#f97316' },
    secondary: { main: '#1e293b' },
    background: { default: '#0a0a0a', paper: '#1a1a1a' },
  },
  typography: { fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif' },
  components: {
    MuiAppBar: { styleOverrides: { root: { backgroundColor: '#1a1a1a', color: '#fff' } } },
    MuiDrawer: { styleOverrides: { paper: { backgroundColor: '#1a1a1a' } } },
  }
});

const Layout: React.FC = () => {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
  const [darkMode, setDarkMode] = useState(() => {
    const saved = localStorage.getItem('admin_dark_mode');
    return saved ? JSON.parse(saved) : false;
  })
  const theme = useTheme()
  const navigate = useNavigate()

  useEffect(() => {
    localStorage.setItem('admin_dark_mode', JSON.stringify(darkMode));
  }, [darkMode]);

  const toggleTheme = () => setDarkMode(!darkMode)

  const handleDrawerToggle = () => setMobileOpen(!mobileOpen)
  const handleProfileMenuOpen = (event: React.MouseEvent<HTMLElement>) => setAnchorEl(event.currentTarget)
  const handleMenuClose = () => setAnchorEl(null)

  const drawer = (
    <div>
      <Toolbar sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Box component="span" sx={{ fontSize: 28, fontWeight: 'bold', color: 'primary.main' }}>🐯</Box>
        <Typography variant="h6" noWrap>TigerSwap</Typography>
      </Toolbar>
      <Divider />
      <List>
        {menuItems.map((item) => (
          <ListItem button key={item.text} onClick={() => navigate(item.path)} sx={{ '&:hover': { bgcolor: 'action.hover' } }}>
            <ListItemIcon sx={{ minWidth: 40, color: darkMode ? '#f97316' : '#f97316' }}>{item.icon}</ListItemIcon>
            <ListItemText primary={item.text} />
          </ListItem>
        ))}
      </List>
    </div>
  )

  const themeValue = { darkMode, toggleTheme }

  return (
    <ThemeContext.Provider value={themeValue}>
      <ThemeProvider theme={darkMode ? darkTheme : lightTheme}>
        <CssBaseline />
        <Box sx={{ display: 'flex' }}>
          <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }} elevation={0}>
            <Toolbar>
              <IconButton color="inherit" edge="start" onClick={handleDrawerToggle} sx={{ mr: 2 }}>
                <MenuIcon />
              </IconButton>
              <Typography variant="h6" noWrap sx={{ flexGrow: 1 }}>Admin Panel</Typography>
              <IconButton color="inherit" onClick={toggleTheme} sx={{ mr: 1 }}>
                {darkMode ? <Brightness7 /> : <Brightness4 />}
              </IconButton>
              <IconButton color="inherit"><Badge badgeContent={4} color="error"><Notifications /></Badge></IconButton>
              <IconButton color="inherit" onClick={handleProfileMenuOpen}>
                <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main' }}>A</Avatar>
              </IconButton>
            </Toolbar>
          </AppBar>
          <Box component="nav" sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}>
            <Drawer variant="temporary" open={mobileOpen} onClose={handleDrawerToggle} sx={{ 
              '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth, bgcolor: darkMode ? '#1a1a1a' : '#fff' } 
            }}>
              {drawer}
            </Drawer>
            <Drawer variant="permanent" sx={{ 
              display: { xs: 'none', md: 'block' }, 
              '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth, bgcolor: darkMode ? '#1a1a1a' : '#fff' } 
            }} open>
              {drawer}
            </Drawer>
          </Box>
          <Box component="main" sx={{ flexGrow: 1, p: 3, width: { md: `calc(100% - ${drawerWidth}px)` }, mt: 8 }}>
            <Outlet context={{ darkMode, toggleTheme }} />
          </Box>
          <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleMenuClose}>
            <MenuItem onClick={handleMenuClose}>Profile</MenuItem>
            <MenuItem onClick={handleMenuClose}>Logout</MenuItem>
          </Menu>
        </Box>
      </ThemeProvider>
    </ThemeContext.Provider>
  )
}

export default Layout