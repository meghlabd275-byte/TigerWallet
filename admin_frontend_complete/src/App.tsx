import React, { useState, useEffect, createContext, useContext } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { ThemeProvider, createTheme, CssBaseline, Box, Drawer, AppBar, Toolbar, List, ListItem, ListItemIcon, ListItemText, ListItemButton, Typography, IconButton, Avatar, Menu, MenuItem, Divider, Badge, Chip, Button, TextField, InputAdornment, CircularProgress, Alert, Snackbar, Card, CardContent, Grid, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Tabs, Tab, Dialog, DialogTitle, DialogContent, DialogActions, FormControl, InputLabel, Select, Switch, FormControlLabel, Checkbox, LinearProgress, Tooltip, Pagination } from '@mui/material';
import { MuiDatePicker } from '@mui/x-date-pickers/DatePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';

// Icons
import DashboardIcon from '@mui/icons-material/Dashboard';
import PeopleIcon from '@mui/icons-material/People';
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import CurrencyBitcoinIcon from '@mui/icons-material/CurrencyBitcoin';
import PaymentIcon from '@mui/icons-material/Payment';
import SettingsIcon from '@mui/icons-material/Settings';
import SecurityIcon from '@mui/icons-material/Security';
import NotificationsIcon from '@mui/icons-material/Notifications';
import AssessmentIcon from '@mui/icons-material/Assessment';
import BusinessIcon from '@mui/icons-material/Business';
import ApiIcon from '@mui/icons-material/Api';
import CloudIcon from '@mui/icons-material/Cloud';
import HistoryIcon from '@mui/icons-material/History';
import BlockIcon from '@mui/icons-material/Block';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import VisibilityIcon from '@mui/icons-material/Visibility';
import AddIcon from '@mui/icons-material/Add';
import SearchIcon from '@mui/icons-material/Search';
import MenuIcon from '@mui/icons-material/Menu';
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import LogoutIcon from '@mui/icons-material/Logout';
import PersonIcon from '@mui/icons-material/Person';

// Theme
const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#ff6b35' },
    secondary: { main: '#00d4aa' },
    background: { default: '#0a0a0f', paper: '#141419' },
    text: { primary: '#ffffff', secondary: '#a0a0a0' },
  },
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
    h1: { fontSize: '2rem', fontWeight: 700 },
    h2: { fontSize: '1.5rem', fontWeight: 600 },
    h3: { fontSize: '1.25rem', fontWeight: 600 },
  },
  components: {
    MuiButton: { styleOverrides: { root: { textTransform: 'none', borderRadius: 8 } } },
    MuiCard: { styleOverrides: { root: { borderRadius: 12, border: '1px solid #2a2a35' } } },
    MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
  },
});

// Types
interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  avatar?: string;
  tenantId?: string;
}

interface Tenant {
  id: string;
  name: string;
  slug: string;
  email: string;
  status: 'active' | 'suspended' | 'trial' | 'terminated';
  plan: string;
  createdAt: string;
  trialEndsAt?: string;
}

interface Plan {
  id: string;
  name: string;
  tier: string;
  priceMonthly: number;
  priceYearly: number;
  features: string[];
  apiQuotaMonthly: number;
  maxUsers: number;
  maxWallets: number;
  maxBots: number;
}

interface Subscription {
  id: string;
  tenantId: string;
  planId: string;
  status: string;
  currentPeriodStart: string;
  currentPeriodEnd: string;
  plan?: Plan;
}

interface Usage {
  tenantId: string;
  totalApiCalls: number;
  storageUsedGB: number;
  activeUsers: number;
  activeWallets: number;
  activeBots: number;
  overageApiCalls: number;
}

interface Invoice {
  id: string;
  invoiceNumber: string;
  amount: number;
  amountDue: number;
  status: string;
  dueDate: string;
  paidAt?: string;
}

interface APIKey {
  id: string;
  name: string;
  key: string;
  permissions: string[];
  rateLimit: number;
  isActive: boolean;
  lastUsedAt?: string;
  createdAt: string;
}

interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  chain: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: string;
}

interface Wallet {
  id: string;
  address: string;
  type: string;
  chain: string;
  balance: string;
  tokens: number;
  createdAt: string;
}

interface Bot {
  id: string;
  name: string;
  type: string;
  status: 'running' | 'stopped' | 'error';
  pair: string;
  capital: number;
  pnl: number;
  createdAt: string;
}

interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  type: string;
  rpcUrl: string;
  explorerUrl: string;
  isActive: boolean;
  chainId?: number;
}

interface AuditLog {
  id: string;
  userId: string;
  action: string;
  resource: string;
  details: string;
  ipAddress: string;
  timestamp: string;
}

// Context
interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
};

// API Base URL
const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:9000/api/v1';

// API Service
const api = {
  async request(endpoint: string, options: RequestInit = {}) {
    const token = localStorage.getItem('token');
    const headers = {
      'Content-Type': 'application/json',
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options.headers,
    };
    const response = await fetch(`${API_BASE}${endpoint}`, { ...options, headers });
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }
    return response.json();
  },

  // Auth
  login: (email: string, password: string) => 
    api.request('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  register: (email: string, password: string, name: string) =>
    api.request('/auth/register', { method: 'POST', body: JSON.stringify({ email, password, name }) }),

  // Tenants
  getTenants: (params?: { status?: string; limit?: number; offset?: number }) =>
    api.request(`/tenants?${new URLSearchParams(params as any)}`),
  getTenant: (id: string) => api.request(`/tenants/${id}`),
  createTenant: (data: Partial<Tenant>) => api.request('/tenants', { method: 'POST', body: JSON.stringify(data) }),
  updateTenant: (id: string, data: Partial<Tenant>) => api.request(`/tenants/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  updateTenantStatus: (id: string, status: string) => api.request(`/tenants/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),

  // Plans
  getPlans: () => api.request('/public/plans'),
  getPlan: (id: string) => api.request(`/plans/${id}`),

  // Subscriptions
  getSubscription: (tenantId: string) => api.request(`/subscriptions/${tenantId}`),
  upgradeSubscription: (tenantId: string, planId: string, billingCycle: string) =>
    api.request(`/subscriptions/${tenantId}/upgrade`, { method: 'POST', body: JSON.stringify({ plan_id: planId, billing_cycle: billingCycle }) }),
  cancelSubscription: (tenantId: string) => api.request(`/subscriptions/${tenantId}/cancel`, { method: 'POST' }),

  // Usage
  getUsage: (tenantId: string) => api.request(`/usage/${tenantId}`),
  recordUsage: (tenantId: string, apiMethod: string, count: number) =>
    api.request(`/usage/${tenantId}/record`, { method: 'POST', body: JSON.stringify({ api_method: apiMethod, count }) }),
  checkQuota: (tenantId: string, type: string) => api.request(`/usage/${tenantId}/check?type=${type}`),

  // Invoices
  getInvoices: (tenantId: string) => api.request(`/invoices/${tenantId}`),
  getInvoice: (id: string) => api.request(`/invoices/invoice/${id}`),

  // API Keys
  getApiKeys: (tenantId: string) => api.request(`/api-keys?tenant_id=${tenantId}`),
  createApiKey: (data: Partial<APIKey>) => api.request('/api-keys', { method: 'POST', body: JSON.stringify(data) }),
  deleteApiKey: (id: string) => api.request(`/api-keys/${id}`, { method: 'DELETE' }),

  // Wallets
  getWallets: (tenantId: string) => api.request(`/wallets?tenant_id=${tenantId}`),
  getWallet: (id: string) => api.request(`/wallets/${id}`),
  createWallet: (data: Partial<Wallet>) => api.request('/wallets', { method: 'POST', body: JSON.stringify(data) }),

  // Bots
  getBots: (tenantId: string) => api.request(`/bots?tenant_id=${tenantId}`),
  getBot: (id: string) => api.request(`/bots/${id}`),
  createBot: (data: Partial<Bot>) => api.request('/bots', { method: 'POST', body: JSON.stringify(data) }),
  updateBot: (id: string, data: Partial<Bot>) => api.request(`/bots/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteBot: (id: string) => api.request(`/bots/${id}`, { method: 'DELETE' }),

  // Blockchains
  getBlockchains: () => api.request('/blockchains'),
  getBlockchain: (id: string) => api.request(`/blockchains/${id}`),
  createBlockchain: (data: Partial<Blockchain>) => api.request('/blockchains', { method: 'POST', body: JSON.stringify(data) }),
  updateBlockchain: (id: string, data: Partial<Blockchain>) => api.request(`/blockchains/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteBlockchain: (id: string) => api.request(`/blockchains/${id}`, { method: 'DELETE' }),

  // Transactions
  getTransactions: (params?: { tenantId?: string; walletId?: string; limit?: number }) =>
    api.request(`/transactions?${new URLSearchParams(params as any)}`),
  getTransaction: (id: string) => api.request(`/transactions/${id}`),

  // Audit Logs
  getAuditLogs: (tenantId: string, limit?: number) => api.request(`/audit-logs/${tenantId}${limit ? `?limit=${limit}` : ''}`),
};

// Auth Provider
const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (token) {
      // Validate token and get user info
      api.request('/auth/me')
        .then(data => setUser(data.user))
        .catch(() => {
          localStorage.removeItem('token');
          setToken(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [token]);

  const login = async (email: string, password: string) => {
    const data = await api.login(email, password);
    localStorage.setItem('token', data.token);
    setToken(data.token);
    setUser(data.user);
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, token, login, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
};

// Layout
interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [drawerOpen, setDrawerOpen] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const menuItems = [
    { text: 'Dashboard', icon: <DashboardIcon />, path: '/' },
    { text: 'Tenants', icon: <BusinessIcon />, path: '/tenants' },
    { text: 'Users', icon: <PeopleIcon />, path: '/users' },
    { text: 'Wallets', icon: <AccountBalanceWalletIcon />, path: '/wallets' },
    { text: 'Bots', icon: <SmartToyIcon />, path: '/bots' },
    { text: 'Blockchains', icon: <CurrencyBitcoinIcon />, path: '/blockchains' },
    { text: 'Transactions', icon: <PaymentIcon />, path: '/transactions' },
    { text: 'Billing', icon: <PaymentIcon />, path: '/billing' },
    { text: 'API Keys', icon: <ApiIcon />, path: '/api-keys' },
    { text: 'Audit Logs', icon: <HistoryIcon />, path: '/audit-logs' },
    { text: 'Settings', icon: <SettingsIcon />, path: '/settings' },
  ];

  const handleMenu = (event: React.MouseEvent<HTMLElement>) => setAnchorEl(event.currentTarget);
  const handleClose = () => setAnchorEl(null);
  const handleLogout = () => { handleClose(); logout(); navigate('/login'); };

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1, bgcolor: '#141419', borderBottom: '1px solid #2a2a35' }}>
        <Toolbar>
          <IconButton color="inherit" edge="start" onClick={() => setDrawerOpen(!drawerOpen)} sx={{ mr: 2 }}>
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" noWrap sx={{ flexGrow: 1, fontWeight: 700, color: '#ff6b35' }}>
            TigerWallet Admin
          </Typography>
          <Tooltip title="Notifications">
            <IconButton color="inherit">
              <Badge badgeContent={3} color="error">
                <NotificationsIcon />
              </Badge>
            </IconButton>
          </Tooltip>
          <Tooltip title="Account">
            <IconButton onClick={handleMenu} sx={{ ml: 1 }}>
              <Avatar sx={{ width: 32, height: 32, bgcolor: '#ff6b35' }}>
                {user?.name?.charAt(0) || 'A'}
              </Avatar>
            </IconButton>
          </Tooltip>
          <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleClose}>
            <MenuItem disabled>
              <Typography variant="body2">{user?.email}</Typography>
            </MenuItem>
            <Divider />
            <MenuItem onClick={handleLogout}>
              <ListItemIcon><LogoutIcon fontSize="small" /></ListItemIcon>
              Logout
            </MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>
      
      <Drawer
        variant="permanent"
        sx={{
          width: drawerOpen ? 240 : 65,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerOpen ? 240 : 65,
            boxSizing: 'border-box',
            bgcolor: '#141419',
            borderRight: '1px solid #2a2a35',
            transition: 'width 0.2s',
          },
        }}
      >
        <Toolbar />
        <List sx={{ pt: 2 }}>
          {menuItems.map((item) => (
            <ListItem key={item.text} disablePadding>
              <ListItemButton
                onClick={() => navigate(item.path)}
                sx={{ minHeight: 48, px: 2.5, justifyContent: drawerOpen ? 'initial' : 'center' }}
              >
                <ListItemIcon sx={{ minWidth: 0, mr: drawerOpen ? 2 : 'auto', color: '#a0a0a0' }}>
                  {item.icon}
                </ListItemIcon>
                {drawerOpen && <ListItemText primary={item.text} />}
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </Drawer>

      <Box component="main" sx={{ flexGrow: 1, p: 3, mt: 8, bgcolor: '#0a0a0f', minHeight: '100vh' }}>
        {children}
      </Box>
    </Box>
  );
};

// Login Page
const LoginPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(email, password);
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#0a0a0f' }}>
      <Card sx={{ maxWidth: 400, width: '100%', p: 3 }}>
        <Typography variant="h4" align="center" gutterBottom sx={{ color: '#ff6b35', fontWeight: 700 }}>
          TigerWallet Admin
        </Typography>
        <Typography variant="body2" align="center" color="text.secondary" sx={{ mb: 3 }}>
          Sign in to your account
        </Typography>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <form onSubmit={handleSubmit}>
          <TextField
            fullWidth label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)}
            sx={{ mb: 2 }} required
          />
          <TextField
            fullWidth label="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)}
            sx={{ mb: 3 }} required
          />
          <Button fullWidth variant="contained" type="submit" disabled={loading} sx={{ py: 1.5 }}>
            {loading ? <CircularProgress size={24} /> : 'Sign In'}
          </Button>
        </form>
      </Card>
    </Box>
  );
};

// Dashboard
const Dashboard: React.FC = () => {
  const [stats, setStats] = useState({
    totalTenants: 0,
    activeTenants: 0,
    totalWallets: 0,
    totalBots: 0,
    totalTransactions: 0,
    revenue: 0,
  });
  const [recentActivity, setRecentActivity] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // In production, fetch real data
        setStats({
          totalTenants: 156,
          activeTenants: 142,
          totalWallets: 8934,
          totalBots: 567,
          totalTransactions: 45678,
          revenue: 234567,
        });
        setRecentActivity([
          { id: '1', action: 'New tenant registered', tenant: 'Acme Corp', time: '2 min ago' },
          { id: '2', action: 'Subscription upgraded', tenant: 'TechStart', time: '15 min ago' },
          { id: '3', action: 'New wallet created', tenant: 'CryptoKing', time: '30 min ago' },
          { id: '4', action: 'Bot started', tenant: 'TradeBot Inc', time: '1 hour ago' },
          { id: '5', action: 'KYC approved', tenant: 'VerifiedUser', time: '2 hours ago' },
        ]);
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const StatCard: React.FC<{ title: string; value: string | number; icon: React.ReactNode; color: string }> = 
    ({ title, value, icon, color }) => (
      <Card sx={{ height: '100%' }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography variant="body2" color="text.secondary">{title}</Typography>
              <Typography variant="h4" sx={{ fontWeight: 700, color }}>{value}</Typography>
            </Box>
            <Box sx={{ p: 1.5, borderRadius: 2, bgcolor: `${color}20` }}>{icon}</Box>
          </Box>
        </CardContent>
      </Card>
    );

  if (loading) return <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>;

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Dashboard</Typography>
      
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard title="Total Tenants" value={stats.totalTenants} icon={<BusinessIcon />} color="#ff6b35" />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard title="Active Tenants" value={stats.activeTenants} icon={<CheckCircleIcon />} color="#00d4aa" />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard title="Total Wallets" value={stats.totalWallets} icon={<AccountBalanceWalletIcon />} color="#00bcd4" />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard title="Active Bots" value={stats.totalBots} icon={<SmartToyIcon />} color="#9c27b0" />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard title="Transactions" value={stats.totalTransactions.toLocaleString()} icon={<PaymentIcon />} color="#ff9800" />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard title="Revenue" value={`$${stats.revenue.toLocaleString()}`} icon={<CurrencyBitcoinIcon />} color="#4caf50" />
        </Grid>
      </Grid>

      <Grid container spacing={3}>
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>Recent Activity</Typography>
              <List>
                {recentActivity.map((activity) => (
                  <ListItem key={activity.id} sx={{ px: 0 }}>
                    <ListItemIcon><HistoryIcon sx={{ color: '#a0a0a0' }} /></ListItemIcon>
                    <ListItemText 
                      primary={activity.action} 
                      secondary={`${activity.tenant} • ${activity.time}`}
                    />
                  </ListItem>
                ))}
              </List>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={4}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>Quick Actions</Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Button variant="contained" startIcon={<AddIcon />}>Add Tenant</Button>
                <Button variant="outlined" startIcon={<CurrencyBitcoinIcon />}>Add Blockchain</Button>
                <Button variant="outlined" startIcon={<SmartToyIcon />}>Create Bot</Button>
                <Button variant="outlined" startIcon={<ApiIcon />}>Generate API Key</Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

// Tenants Page
const TenantsPage: React.FC = () => {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [openDialog, setOpenDialog] = useState(false);
  const [formData, setFormData] = useState({ name: '', email: '', slug: '' });

  useEffect(() => {
    fetchTenants();
  }, [page, search]);

  const fetchTenants = async () => {
    setLoading(true);
    try {
      const data = await api.getTenants({ limit: 10, offset: page * 10 });
      setTenants(data.tenants || []);
      setTotal(data.total || 0);
    } catch (error) {
      console.error('Failed to fetch tenants:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTenant = async () => {
    try {
      await api.createTenant(formData);
      setOpenDialog(false);
      setFormData({ name: '', email: '', slug: '' });
      fetchTenants();
    } catch (error) {
      console.error('Failed to create tenant:', error);
    }
  };

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1 },
    { field: 'slug', headerName: 'Slug', flex: 1 },
    { field: 'email', headerName: 'Email', flex: 1 },
    { field: 'status', headerName: 'Status', width: 120, 
      renderCell: (params) => (
        <Chip 
          label={params.value} 
          color={params.value === 'active' ? 'success' : params.value === 'trial' ? 'warning' : 'default'}
          size="small" 
        />
      )
    },
    { field: 'plan', headerName: 'Plan', width: 120 },
    { field: 'createdAt', headerName: 'Created', width: 150, 
      renderCell: (params) => new Date(params.value).toLocaleDateString() 
    },
    { field: 'actions', headerName: 'Actions', width: 150,
      renderCell: (params) => (
        <Box>
          <IconButton size="small" onClick={() => {}}><VisibilityIcon /></IconButton>
          <IconButton size="small" onClick={() => {}}><EditIcon /></IconButton>
          <IconButton size="small" color="error" onClick={() => {}}><DeleteIcon /></IconButton>
        </Box>
      )
    },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Tenants</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setOpenDialog(true)}>
          Add Tenant
        </Button>
      </Box>

      <TextField
        fullWidth placeholder="Search tenants..." value={search} onChange={(e) => setSearch(e.target.value)}
        InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment> }}
        sx={{ mb: 3 }}
      />

      <Card sx={{ height: 600 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>
        ) : (
          <DataGrid
            rows={tenants}
            columns={columns}
            pagination
            pageSize={10}
            rowCount={total}
            paginationMode="server"
            onPageChange={(newPage) => setPage(newPage)}
            disableRowSelectionOnClick
          />
        )}
      </Card>

      <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add New Tenant</DialogTitle>
        <DialogContent>
          <TextField fullWidth label="Name" value={formData.name} onChange={(e) => setFormData({...formData, name: e.target.value})} sx={{ mt: 2, mb: 2 }} />
          <TextField fullWidth label="Email" type="email" value={formData.email} onChange={(e) => setFormData({...formData, email: e.target.value})} sx={{ mb: 2 }} />
          <TextField fullWidth label="Slug" value={formData.slug} onChange={(e) => setFormData({...formData, slug: e.target.value})} helperText="Unique identifier for the tenant" />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreateTenant}>Create</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

// Wallets Page
const WalletsPage: React.FC = () => {
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchWallets = async () => {
      setLoading(true);
      try {
        // In production, fetch real data
        setWallets([
          { id: '1', address: '0x742d35Cc6634C0532925a3b844Bc9e7595f...', type: 'EOA', chain: 'Ethereum', balance: '12.5 ETH', tokens: 5, createdAt: '2024-01-15' },
          { id: '2', address: '0x123d35Cc6634C0532925a3b844Bc9e7595g...', type: 'Contract', chain: 'BNB Chain', balance: '5.2 BNB', tokens: 12, createdAt: '2024-01-20' },
          { id: '3', address: 'Solana123...', type: 'PDA', chain: 'Solana', balance: '100 SOL', tokens: 8, createdAt: '2024-02-01' },
        ]);
      } catch (error) {
        console.error('Failed to fetch wallets:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchWallets();
  }, []);

  const columns: GridColDef[] = [
    { field: 'address', headerName: 'Address', flex: 2 },
    { field: 'type', headerName: 'Type', width: 120 },
    { field: 'chain', headerName: 'Chain', width: 120 },
    { field: 'balance', headerName: 'Balance', width: 150 },
    { field: 'tokens', headerName: 'Tokens', width: 100 },
    { field: 'createdAt', headerName: 'Created', width: 150 },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Wallets</Typography>
        <Button variant="contained" startIcon={<AddIcon />}>Create Wallet</Button>
      </Box>

      <Card sx={{ height: 600 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>
        ) : (
          <DataGrid rows={wallets} columns={columns} disableRowSelectionOnClick />
        )}
      </Card>
    </Box>
  );
};

// Bots Page
const BotsPage: React.FC = () => {
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchBots = async () => {
      setLoading(true);
      try {
        // In production, fetch real data
        setBots([
          { id: '1', name: 'DCA Bot 1', type: 'dca', status: 'running', pair: 'BTC/USDT', capital: 10000, pnl: 5.2, createdAt: '2024-01-10' },
          { id: '2', name: 'Grid Trading', type: 'grid', status: 'running', pair: 'ETH/USDT', capital: 5000, pnl: 3.8, createdAt: '2024-01-15' },
          { id: '3', name: 'Arbitrage Bot', type: 'arbitrage', status: 'stopped', pair: 'BNB/USDT', capital: 20000, pnl: 0, createdAt: '2024-02-01' },
          { id: '4', name: 'Signal Bot', type: 'signal', status: 'error', pair: 'SOL/USDT', capital: 3000, pnl: -2.1, createdAt: '2024-02-10' },
        ]);
      } catch (error) {
        console.error('Failed to fetch bots:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchBots();
  }, []);

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1 },
    { field: 'type', headerName: 'Type', width: 120 },
    { field: 'status', headerName: 'Status', width: 120,
      renderCell: (params) => (
        <Chip 
          label={params.value} 
          color={params.value === 'running' ? 'success' : params.value === 'stopped' ? 'default' : 'error'}
          size="small" 
        />
      )
    },
    { field: 'pair', headerName: 'Pair', width: 120 },
    { field: 'capital', headerName: 'Capital ($)', width: 120, renderCell: (params) => `$${params.value.toLocaleString()}` },
    { field: 'pnl', headerName: 'PnL (%)', width: 100,
      renderCell: (params) => (
        <Typography sx={{ color: params.value >= 0 ? '#00d4aa' : '#f44336', fontWeight: 600 }}>
          {params.value >= 0 ? '+' : ''}{params.value}%
        </Typography>
      )
    },
    { field: 'actions', headerName: 'Actions', width: 150,
      renderCell: (params) => (
        <Box>
          <IconButton size="small"><VisibilityIcon /></IconButton>
          <IconButton size="small"><EditIcon /></IconButton>
          <IconButton size="small" color="error"><DeleteIcon /></IconButton>
        </Box>
      )
    },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Trading Bots</Typography>
        <Button variant="contained" startIcon={<AddIcon />}>Create Bot</Button>
      </Box>

      <Card sx={{ height: 600 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>
        ) : (
          <DataGrid rows={bots} columns={columns} disableRowSelectionOnClick />
        )}
      </Card>
    </Box>
  );
};

// Blockchains Page
const BlockchainsPage: React.FC = () => {
  const [blockchains, setBlockchains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchBlockchains = async () => {
      setLoading(true);
      try {
        const data = await api.getBlockchains();
        setBlockchains(data.blockchains || []);
      } catch (error) {
        console.error('Failed to fetch blockchains:', error);
        // Fallback data
        setBlockchains([
          { id: '1', name: 'Ethereum', symbol: 'ETH', type: 'evm', rpcUrl: 'https://eth.llamarpc.com', explorerUrl: 'https://etherscan.io', isActive: true, chainId: 1 },
          { id: '2', name: 'BNB Chain', symbol: 'BNB', type: 'evm', rpcUrl: 'https://bsc-dataseed.binance.org', explorerUrl: 'https://bscscan.com', isActive: true, chainId: 56 },
          { id: '3', name: 'Solana', symbol: 'SOL', type: 'non-evm', rpcUrl: 'https://api.mainnet-beta.solana.com', explorerUrl: 'https://explorer.solana.com', isActive: true },
          { id: '4', name: 'Aptos', symbol: 'APT', type: 'non-evm', rpcUrl: 'https://fullnode.mainnet.aptoslabs.com', explorerUrl: 'https://explorer.aptoslabs.com', isActive: true },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchBlockchains();
  }, []);

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1 },
    { field: 'symbol', headerName: 'Symbol', width: 100 },
    { field: 'type', headerName: 'Type', width: 100 },
    { field: 'chainId', headerName: 'Chain ID', width: 100 },
    { field: 'isActive', headerName: 'Status', width: 100,
      renderCell: (params) => (
        <Chip label={params.value ? 'Active' : 'Inactive'} color={params.value ? 'success' : 'default'} size="small" />
      )
    },
    { field: 'actions', headerName: 'Actions', width: 150,
      renderCell: (params) => (
        <Box>
          <IconButton size="small"><EditIcon /></IconButton>
          <IconButton size="small" color="error"><DeleteIcon /></IconButton>
        </Box>
      )
    },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Blockchains</Typography>
        <Button variant="contained" startIcon={<AddIcon />}>Add Blockchain</Button>
      </Box>

      <Card sx={{ height: 600 }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>
        ) : (
          <DataGrid rows={blockchains} columns={columns} disableRowSelectionOnClick />
        )}
      </Card>
    </Box>
  );
};

// Billing Page
const BillingPage: React.FC = () => {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState(0);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [plansData, invoicesData] = await Promise.all([
          api.getPlans(),
          api.getInvoices('current'), // Would use actual tenant ID
        ]);
        setPlans(plansData.plans || []);
        setInvoices(invoicesData.invoices || []);
      } catch (error) {
        console.error('Failed to fetch billing data:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Billing & Subscription</Typography>
      
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 3 }}>
        <Tab label="Plans" />
        <Tab label="Invoices" />
        <Tab label="Usage" />
      </Tabs>

      {tab === 0 && (
        <Grid container spacing={3}>
          {plans.map((plan) => (
            <Grid item xs={12} sm={6} md={3} key={plan.id}>
              <Card sx={{ height: '100%', border: plan.tier === 'pro' ? '2px solid #ff6b35' : undefined }}>
                <CardContent>
                  <Typography variant="h6">{plan.name}</Typography>
                  <Typography variant="h4" sx={{ my: 2, color: '#ff6b35' }}>
                    ${(plan.priceMonthly / 100).toFixed(0)}<Typography component="span" variant="body2">/mo</Typography>
                  </Typography>
                  <Box sx={{ mb: 2 }}>
                    {plan.features?.slice(0, 4).map((feature, i) => (
                      <Typography key={i} variant="body2" sx={{ display: 'flex', alignItems: 'center', mb: 0.5 }}>
                        <CheckCircleIcon sx={{ fontSize: 16, mr: 1, color: '#00d4aa' }} />
                        {feature}
                      </Typography>
                    ))}
                  </Box>
                  <Button fullWidth variant="outlined">Current Plan</Button>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {tab === 1 && (
        <Card>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Invoice #</TableCell>
                  <TableCell>Amount</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Due Date</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {invoices.map((invoice) => (
                  <TableRow key={invoice.id}>
                    <TableCell>{invoice.invoiceNumber}</TableCell>
                    <TableCell>${(invoice.amount / 100).toFixed(2)}</TableCell>
                    <TableCell>
                      <Chip 
                        label={invoice.status} 
                        color={invoice.status === 'paid' ? 'success' : invoice.status === 'pending' ? 'warning' : 'error'} 
                        size="small" 
                      />
                    </TableCell>
                    <TableCell>{new Date(invoice.dueDate).toLocaleDateString()}</TableCell>
                    <TableCell>
                      <Button size="small">View</Button>
                      <Button size="small">Download</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Card>
      )}

      {tab === 2 && (
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>Current Month Usage</Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <Typography variant="body2" color="text.secondary">API Calls</Typography>
                <Typography variant="h4">45,678 / 500,000</Typography>
                <LinearProgress variant="determinate" value={9} sx={{ mt: 1, height: 8, borderRadius: 4 }} />
              </Grid>
              <Grid item xs={12} sm={6}>
                <Typography variant="body2" color="text.secondary">Storage</Typography>
                <Typography variant="h4">2.5 GB / 100 GB</Typography>
                <LinearProgress variant="determinate" value={2.5} sx={{ mt: 1, height: 8, borderRadius: 4 }} />
              </Grid>
              <Grid item xs={12} sm={6}>
                <Typography variant="body2" color="text.secondary">Active Users</Typography>
                <Typography variant="h4">15 / 25</Typography>
                <LinearProgress variant="determinate" value={60} sx={{ mt: 1, height: 8, borderRadius: 4 }} />
              </Grid>
              <Grid item xs={12} sm={6}>
                <Typography variant="body2" color="text.secondary">Active Bots</Typography>
                <Typography variant="h4">8 / 100</Typography>
                <LinearProgress variant="determinate" value={8} sx={{ mt: 1, height: 8, borderRadius: 4 }} />
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      )}
    </Box>
  );
};

// API Keys Page
const APIKeysPage: React.FC = () => {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchApiKeys = async () => {
      setLoading(true);
      try {
        // In production, fetch real data
        setApiKeys([
          { id: '1', name: 'Production Key', key: 'tw_live_xxxxx...', permissions: ['read', 'write'], rateLimit: 1000, isActive: true, createdAt: '2024-01-15' },
          { id: '2', name: 'Development Key', key: 'tw_test_xxxxx...', permissions: ['read'], rateLimit: 100, isActive: true, createdAt: '2024-01-20' },
        ]);
      } catch (error) {
        console.error('Failed to fetch API keys:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchApiKeys();
  }, []);

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Name', flex: 1 },
    { field: 'key', headerName: 'API Key', flex: 2, renderCell: (params) => params.value?.substring(0, 20) + '...' },
    { field: 'permissions', headerName: 'Permissions', width: 150, renderCell: (params) => params.value?.join(', ') },
    { field: 'rateLimit', headerName: 'Rate Limit/min', width: 130 },
    { field: 'isActive', headerName: 'Status', width: 100,
      renderCell: (params) => (
        <Switch checked={params.value} disabled size="small" />
      )
    },
    { field: 'createdAt', headerName: 'Created', width: 150 },
  ];

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">API Keys</Typography>
        <Button variant="contained" startIcon={<AddIcon />}>Generate Key</Button>
      </Box>

      <Card>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', pt: 4 }}><CircularProgress /></Box>
        ) : (
          <DataGrid rows={apiKeys} columns={columns} disableRowSelectionOnClick />
        )}
      </Card>
    </Box>
  );
};

// Protected Route
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}><CircularProgress /></Box>;
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <Layout>{children}</Layout>;
};

// Main App
const App: React.FC = () => {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <BrowserRouter>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
              <Route path="/tenants" element={<ProtectedRoute><TenantsPage /></ProtectedRoute>} />
              <Route path="/wallets" element={<ProtectedRoute><WalletsPage /></ProtectedRoute>} />
              <Route path="/bots" element={<ProtectedRoute><BotsPage /></ProtectedRoute>} />
              <Route path="/blockchains" element={<ProtectedRoute><BlockchainsPage /></ProtectedRoute>} />
              <Route path="/billing" element={<ProtectedRoute><BillingPage /></ProtectedRoute>} />
              <Route path="/api-keys" element={<ProtectedRoute><APIKeysPage /></ProtectedRoute>} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </LocalizationProvider>
    </ThemeProvider>
  );
};

export default App;
