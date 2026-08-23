import React, { useState, useEffect, createContext, useContext } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { ThemeProvider, createTheme, CssBaseline, Box, AppBar, Toolbar, Typography, Button, Card, CardContent, Grid, TextField, CircularProgress, Alert, Tabs, Tab, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Chip, IconButton, Avatar, Menu, MenuItem, Divider, Drawer, List, ListItem, ListItemIcon, ListItemText, ListItemButton, LinearProgress } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import DashboardIcon from '@mui/icons-material/Dashboard';
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import ReceiptIcon from '@mui/icons-material/Receipt';
import SettingsIcon from '@mui/icons-material/Settings';
import SecurityIcon from '@mui/icons-material/Security';
import LogoutIcon from '@mui/icons-material/Logout';
import MenuIcon from '@mui/icons-material/Menu';
import AddIcon from '@mui/icons-material/Add';
import VisibilityIcon from '@mui/icons-material/Visibility';
import EditIcon from '@mui/icons-material/Edit';
import NotificationsIcon from '@mui/icons-material/Notifications';

const theme = createTheme({
  palette: { mode: 'dark', primary: { main: '#00d4aa' }, secondary: { main: '#ff6b35' },
    background: { default: '#0a0a0f', paper: '#141419' }, text: { primary: '#ffffff', secondary: '#a0a0a0' },
  },
  components: { MuiButton: { styleOverrides: { root: { textTransform: 'none', borderRadius: 8 } } },
    MuiCard: { styleOverrides: { root: { borderRadius: 12, border: '1px solid #2a2a35' } } },
});

interface User { id: string; email: string; name: string; tenantId: string; tenantName: string; }
interface AuthContextType { user: User | null; login: (e: string, p: string) => Promise<void>; logout: () => void; isLoading: boolean; }
const AuthContext = createContext<AuthContextType | null>(null);
const useAuth = () => { const c = useContext(AuthContext); if (!c) throw new Error(); return c; };

const api = {
  async request(endpoint: string, options: RequestInit = {}) {
    const token = localStorage.getItem('wl_token');
    const headers = { 'Content-Type': 'application/json', ...(token && { Authorization: `Bearer ${token}` }), ...options.headers };
    const res = await fetch(`http://localhost:9000/api/v1${endpoint}`, { ...options, headers });
    if (!res.ok) throw new Error('Failed');
    return res.json();
  },
  login: (email: string, pwd: string) => api.request('/auth/login', { method: 'POST', body: JSON.stringify({ email, password: pwd }) }),
};

const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('wl_token'));
  const [isLoading, setIsLoading] = useState(true);
  useEffect(() => { if (token) api.request('/auth/me').then(d => setUser(d.user)).catch(() => { localStorage.removeItem('wl_token'); setToken(null); }).finally(() => setIsLoading(false)); else setIsLoading(false); }, [token]);
  const login = async (email: string, password: string) => { const data = await api.login(email, password); localStorage.setItem('wl_token', data.token); setToken(data.token); setUser(data.user); };
  const logout = () => { localStorage.removeItem('wl_token'); setToken(null); setUser(null); };
  return <AuthContext.Provider value={{ user, login, logout, isLoading }}>{children}</AuthContext.Provider>;
};

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [drawerOpen, setDrawerOpen] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const menuItems = [
    { text: 'Dashboard', icon: <DashboardIcon />, path: '/' },
    { text: 'Wallets', icon: <AccountBalanceWalletIcon />, path: '/wallets' },
    { text: 'Bots', icon: <SmartToyIcon />, path: '/bots' },
    { text: 'Billing', icon: <ReceiptIcon />, path: '/billing' },
    { text: 'Settings', icon: <SettingsIcon />, path: '/settings' },
  ];
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <AppBar position="fixed" sx={{ zIndex: 1201, bgcolor: '#141419', borderBottom: '1px solid #2a2a35' }}>
        <Toolbar>
          <IconButton color="inherit" edge="start" onClick={() => setDrawerOpen(!drawerOpen)} sx={{ mr: 2 }}><MenuIcon /></IconButton>
          <Typography variant="h6" sx={{ flexGrow: 1, fontWeight: 700, color: '#00d4aa' }}>{user?.tenantName || 'Portal'}</Typography>
          <IconButton color="inherit"><Badge badgeContent={2} color="error"><NotificationsIcon /></Badge></IconButton>
          <IconButton onClick={(e) => setAnchorEl(e.currentTarget)} sx={{ ml: 1 }}><Avatar sx={{ width: 32, height: 32, bgcolor: '#00d4aa' }}>{user?.name?.charAt(0)}</Avatar></IconButton>
          <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={() => setAnchorEl(null)}>
            <MenuItem disabled><Typography variant="body2">{user?.email}</Typography></MenuItem><Divider />
            <MenuItem onClick={() => { setAnchorEl(null); logout(); navigate('/login'); }}><ListItemIcon><LogoutIcon /></ListItemIcon>Logout</MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>
      <Drawer variant="permanent" sx={{ width: drawerOpen ? 240 : 65, '& .MuiDrawer-paper': { width: drawerOpen ? 240 : 65, bgcolor: '#141419', borderRight: '1px solid #2a2a35' } }}>
        <Toolbar /><List sx={{ pt: 2 }}>
          {menuItems.map((item) => (
            <ListItem key={item.text} disablePadding>
              <ListItemButton onClick={() => navigate(item.path)} sx={{ minHeight: 48, px: 2.5, justifyContent: drawerOpen ? 'initial' : 'center' }}>
                <ListItemIcon sx={{ minWidth: 0, mr: drawerOpen ? 2 : 'auto' }}>{item.icon}</ListItemIcon>
                {drawerOpen && <ListItemText primary={item.text} />}
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </Drawer>
      <Box component="main" sx={{ flexGrow: 1, p: 3, mt: 8, bgcolor: '#0a0a0f', minHeight: '100vh' }}>{children}</Box>
    </Box>
  );
};

const LoginPage: React.FC = () => {
  const [email, setEmail] = useState(''), [password, setPassword] = useState(''), [error, setError] = useState(''), [loading, setLoading] = useState(false);
  const { login } = useAuth(); const navigate = useNavigate();
  const handleSubmit = async (e: React.FormEvent) => { e.preventDefault(); setError(''); setLoading(true); try { await login(email, password); navigate('/'); } catch (err: any) { setError(err.message); } finally { setLoading(false); } };
  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#0a0a0f' }}>
      <Card sx={{ maxWidth: 400, width: '100%', p: 3 }}>
        <Typography variant="h4" align="center" sx={{ color: '#00d4aa', fontWeight: 700, mb: 2 }}>White Label Portal</Typography>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <form onSubmit={handleSubmit}>
          <TextField fullWidth label="Email" value={email} onChange={(e) => setEmail(e.target.value)} sx={{ mb: 2 }} required />
          <TextField fullWidth label="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} sx={{ mb: 3 }} required />
          <Button fullWidth variant="contained" type="submit" disabled={loading} sx={{ py: 1.5 }}>{loading ? <CircularProgress size={24} /> : 'Sign In'}</Button>
        </form>
      </Card>
    </Box>
  );
};

const Dashboard: React.FC = () => (
  <Box>
    <Typography variant="h4" gutterBottom>Dashboard</Typography>
    <Grid container spacing={3}>
      <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2" color="text.secondary">API Calls</Typography><Typography variant="h4" sx={{ color: '#00d4aa' }}>45,234</Typography></CardContent></Card></Grid>
      <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2" color="text.secondary">Storage</Typography><Typography variant="h4">2.5 GB</Typography></CardContent></Card></Grid>
      <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2" color="text.secondary">Wallets</Typography><Typography variant="h4">45</Typography></CardContent></Card></Grid>
      <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2" color="text.secondary">Bots</Typography><Typography variant="h4">8</Typography></CardContent></Card></Grid>
    </Grid>
  </Box>
);

const WalletsPage: React.FC = () => {
  const columns: GridColDef[] = [
    { field: 'address', headerName: 'Address', flex: 2 }, { field: 'type', headerName: 'Type', width: 100 },
    { field: 'chain', headerName: 'Chain', width: 120 }, { field: 'balance', headerName: 'Balance', width: 150 },
    { field: 'actions', headerName: 'Actions', width: 120, renderCell: () => <><IconButton size="small"><VisibilityIcon /></IconButton><IconButton size="small"><EditIcon /></IconButton></> },
  ];
  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}><Typography variant="h4">My Wallets</Typography><Button variant="contained" startIcon={<AddIcon />}>Create Wallet</Button></Box>
      <Card sx={{ height: 500 }}><DataGrid rows={[]} columns={columns} disableRowSelectionOnClick /></Card>
    </Box>
  );
};

const BotsPage: React.FC = () => {
  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1 }, { field: 'type', headerName: 'Type', width: 100 },
    { field: 'status', headerName: 'Status', width: 100, renderCell: (p) => <Chip label={p.value} color={p.value === 'running' ? 'success' : 'default'} size="small" /> },
    { field: 'pair', headerName: 'Pair', width: 120 }, { field: 'pnl', headerName: 'PnL', width: 100, renderCell: (p) => <Typography sx={{ color: p.value >= 0 ? '#00d4aa' : '#f44336' }}>{p.value}%</Typography> },
  ];
  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}><Typography variant="h4">My Bots</Typography><Button variant="contained" startIcon={<AddIcon />}>Create Bot</Button></Box>
      <Card sx={{ height: 500 }}><DataGrid rows={[]} columns={columns} disableRowSelectionOnClick /></Card>
    </Box>
  );
};

const BillingPage: React.FC = () => (
  <Box>
    <Typography variant="h4" gutterBottom>Billing</Typography>
    <Card><CardContent><Typography variant="h6">Current Plan: Pro</Typography><Typography>$299/month</Typography></CardContent></Card>
  </Box>
);

const SettingsPage: React.FC = () => (
  <Box>
    <Typography variant="h4" gutterBottom>Settings</Typography>
    <Card><CardContent><TextField fullWidth label="Name" sx={{ mb: 2 }} /><TextField fullWidth label="Email" sx={{ mb: 2 }} /><Button variant="contained">Save</Button></CardContent></Card>
  </Box>
);

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, isLoading } = useAuth();
  if (isLoading) return <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>;
  if (!user) return <Navigate to="/login" replace />;
  return <Layout>{children}</Layout>;
};

const App: React.FC = () => (
  <ThemeProvider theme={theme}><CssBaseline /><BrowserRouter><AuthProvider>
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
      <Route path="/wallets" element={<ProtectedRoute><WalletsPage /></ProtectedRoute>} />
      <Route path="/bots" element={<ProtectedRoute><BotsPage /></ProtectedRoute>} />
      <Route path="/billing" element={<ProtectedRoute><BillingPage /></ProtectedRoute>} />
      <Route path="/settings" element={<ProtectedRoute><SettingsPage /></ProtectedRoute>} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  </AuthProvider></BrowserRouter></ThemeProvider>
);

export default App;
