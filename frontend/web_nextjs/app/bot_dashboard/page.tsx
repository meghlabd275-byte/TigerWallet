'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField, Tabs, Tab, Chip,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Paper, IconButton, Dialog, DialogTitle, DialogContent, DialogActions,
  Alert, Select, MenuItem, FormControl, InputLabel, Switch,
  FormControlLabel, Slider, Grid, Avatar, Divider, List, ListItem,
  ListItemText, ListItemIcon, LinearProgress, Tooltip, Badge
} from '@mui/material';
import {
  AccountBalance, Speed, ShowChart, TrendingUp, TrendingDown,
  Settings, Add, PlayArrow, Stop, Pause, Delete, Refresh,
  Build, Security, MonetizationOn, Warning, CheckCircle, Error as ErrorIcon,
  Timeline, People, Store, Dashboard, Notifications, History
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';
import api from '../../src/lib/api/client';

// ============================================================================
// Types & Interfaces
// ============================================================================

interface User {
  id: string;
  address: string;
  name: string;
  role: 'admin' | 'operator' | 'client';
  email?: string;
  createdAt: number;
  isActive: boolean;
}

interface Bot {
  id: string;
  name: string;
  type: 'market_maker' | 'arbitrage' | 'sniper' | 'liquidity' | 'front_run' | 'mev' | 'flash_loan' | 'cross_chain' | 'perp_hedge';
  status: 'running' | 'paused' | 'stopped';
  owner: string;
  profit: number;
  volume: number;
  trades: number;
  winRate: number;
  createdAt: number;
  lastActive: number;
  config: BotConfig;
}

interface BotConfig {
  minInvestment: number;
  maxInvestment: number;
  targetApy: number;
  riskLevel: number;
  maxDailyLoss: number;
  stopLoss: number;
}

interface BotStats {
  totalVolume: number;
  totalProfit: number;
  totalTrades: number;
  successfulTrades: number;
  failedTrades: number;
  winRate: number;
  uptime: number;
}

interface Exchange {
  id: string;
  name: string;
  isActive: boolean;
  minTrade: number;
  maxTrade: number;
  fee: number;
}

interface Transaction {
  id: string;
  botId: string;
  exchange: string;
  type: 'buy' | 'sell';
  amount: number;
  price: number;
  profit: number;
  status: 'success' | 'failed';
  timestamp: number;
}

// ============================================================================
// Constants
// ============================================================================

const BOT_TYPES = [
  { value: 'market_maker', label: 'Market Maker', description: 'Provide liquidity and earn spread', icon: <MonetizationOn /> },
  { value: 'arbitrage', label: 'Arbitrage', description: 'Profit from price differences', icon: <ShowChart /> },
  { value: 'sniper', label: 'Sniper', description: 'Fast trade execution', icon: <Speed /> },
  { value: 'liquidity', label: 'Liquidity', description: 'Deepen order books', icon: <Build /> },
  { value: 'mev', label: 'MEV', description: 'Extract MEV from mempool', icon: <TrendingUp /> },
  { value: 'flash_loan', label: 'Flash Loan', description: 'Risk-free flash loan strategies', icon: <TrendingDown /> },
];

const EXCHANGES = [
  { id: 'uniswap', name: 'Uniswap', isActive: true, minTrade: 100, maxTrade: 1000000, fee: 0.3 },
  { id: 'pancakeswap', name: 'PancakeSwap', isActive: true, minTrade: 50, maxTrade: 500000, fee: 0.25 },
  { id: 'sushiswap', name: 'SushiSwap', isActive: true, minTrade: 100, maxTrade: 500000, fee: 0.3 },
  { id: 'curve', name: 'Curve', isActive: true, minTrade: 1000, maxTrade: 10000000, fee: 0.04 },
  { id: 'dydx', name: 'dYdX', isActive: true, minTrade: 100, maxTrade: 1000000, fee: 0.2 },
];

// ============================================================================
// Component
// ============================================================================

export default function BotDashboard() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  
  // Theme-aware colors
  const bgPrimary = isDark ? '#0f172a' : '#f8fafc';
  const bgSecondary = isDark ? '#1e293b' : '#e2e8f0';
  const bgCard = isDark ? 'rgba(30, 41, 59, 0.8)' : 'rgba(255, 255, 255, 0.9)';
  const textPrimary = isDark ? '#f8fafc' : '#0f172a';
  const textSecondary = isDark ? '#94a3b8' : '#64748b';
  const borderColor = isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)';
  const accentColor = '#f97316';

  // Auth State
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  
  // UI State
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // Data State
  const [users, setUsers] = useState<User[]>([]);
  const [bots, setBots] = useState<Bot[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [exchanges, setExchanges] = useState<Exchange[]>(EXCHANGES);
  
  // Dialogs
  const [createBotDialog, setCreateBotDialog] = useState(false);
  const [createUserDialog, setCreateUserDialog] = useState(false);
  const [confirmDialog, setConfirmDialog] = useState({ open: false, title: '', message: '', action: '' });

  // Bot-platform login form state
  const [loginDialog, setLoginDialog] = useState(false);
  const [loginForm, setLoginForm] = useState({ username: '', password: '' });

  // New Bot Form
  const [newBot, setNewBot] = useState({
    name: '',
    type: 'market_maker' as Bot['type'],
    minInvestment: 1000,
    maxInvestment: 10000,
    targetApy: 20,
    riskLevel: 5,
  });
  
  // New User Form
  const [newUser, setNewUser] = useState({
    name: '',
    email: '',
    role: 'client' as User['role'],
    address: '',
  });
  
  // ============================================================================
  // Effects
  // ============================================================================
  
  useEffect(() => {
    if (isConnected) {
      loadData();
    }
  }, [isConnected]);
  
  // ============================================================================
  // Data Loading
  // ============================================================================
  
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [usersRes, botsRes, txRes] = await Promise.allSettled([
        api.getBotUsers(),
        api.getBots(),
        api.getBotTransactions(),
      ]);

      if (usersRes.status === 'fulfilled' && usersRes.value.data) {
        setUsers(usersRes.value.data);
      } else if (usersRes.status === 'rejected') {
        setError('Failed to load users');
      }

      if (botsRes.status === 'fulfilled' && botsRes.value.data) {
        setBots(botsRes.value.data);
      } else if (botsRes.status === 'rejected') {
        setError('Failed to load bots');
      }

      if (txRes.status === 'fulfilled' && txRes.value.data) {
        setTransactions(txRes.value.data);
      }
      // Transactions are optional; failures are silently ignored to avoid clobbering errors above.

      setSuccess('Data loaded successfully');
    } catch (err: any) {
      setError(err.message || 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, []);
  
  // ============================================================================
  // User Management
  // ============================================================================
  
  const handleConnect = useCallback(async () => {
    // If a bot-platform token is already cached, try to resume the session;
    // otherwise open the login dialog.
    const cached = typeof window !== 'undefined' ? localStorage.getItem('bot_auth_token') : null;
    if (cached) {
      api.setBotPlatformToken(cached);
      setLoading(true);
      setError(null);
      try {
        const res = await api.getCurrentBotUser();
        if (res.success && res.data) {
          setCurrentUser(res.data);
          setIsConnected(true);
          return;
        }
      } catch {
        // fall through to login
      } finally {
        setLoading(false);
      }
    }
    setLoginForm({ username: '', password: '' });
    setLoginDialog(true);
  }, []);

  const handleLogin = useCallback(async () => {
    if (!loginForm.username || !loginForm.password) {
      setError('Username and password are required');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await api.loginBotPlatform(loginForm.username, loginForm.password);
      const res = await api.getCurrentBotUser();
      if (res.success && res.data) {
        setCurrentUser(res.data);
        setIsConnected(true);
        setLoginDialog(false);
        setSuccess('Connected to bot platform');
      } else {
        setError(res.error || 'Authentication succeeded but user profile is unavailable');
      }
    } catch (err: any) {
      const msg = err?.response?.data?.error || err.message || 'Failed to log in to bot platform';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [loginForm]);

  const handleDisconnect = useCallback(() => {
    api.setBotPlatformToken(null);
    setCurrentUser(null);
    setIsConnected(false);
    setBots([]);
    setUsers([]);
    setTransactions([]);
  }, []);
  
  const handleCreateUser = useCallback(async () => {
    if (!newUser.name || !newUser.address) {
      setError('Please fill all fields');
      return;
    }
    
    setLoading(true);
    setError(null);
    try {
      const res = await api.createBotUser(newUser);
      if (res.success && res.data) {
        setUsers([...users, res.data]);
        setSuccess(`User created: ${newUser.name}`);
        setCreateUserDialog(false);
        setNewUser({ name: '', email: '', role: 'client', address: '' });
      } else {
        setError(res.error || 'Failed to create user');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to create user');
    } finally {
      setLoading(false);
    }
  }, [newUser, users]);
  
  const handleDeleteUser = useCallback(async (userId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.deleteBotUser(userId);
      if (res.success) {
        setUsers(users.filter(u => u.id !== userId));
        setSuccess('User deleted');
      } else {
        setError(res.error || 'Failed to delete user');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to delete user');
    } finally {
      setLoading(false);
    }
  }, [users]);
  
  // ============================================================================
  // Bot Management
  // ============================================================================
  
  const handleCreateBot = useCallback(async () => {
    if (!newBot.name) {
      setError('Please enter bot name');
      return;
    }
    
    setLoading(true);
    setError(null);
    try {
      const res = await api.createBot({
        name: newBot.name,
        type: newBot.type,
        minInvestment: newBot.minInvestment,
        maxInvestment: newBot.maxInvestment,
        targetApy: newBot.targetApy,
        riskLevel: newBot.riskLevel,
      });
      if (res.success && res.data) {
        setBots([...bots, res.data]);
        setSuccess(`Bot created: ${newBot.name}`);
        setCreateBotDialog(false);
        setNewBot({ name: '', type: 'market_maker', minInvestment: 1000, maxInvestment: 10000, targetApy: 20, riskLevel: 5 });
      } else {
        setError(res.error || 'Failed to create bot');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to create bot');
    } finally {
      setLoading(false);
    }
  }, [newBot, bots]);
  
  const handleStartBot = useCallback(async (botId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.startBot(botId);
      if (res.success && res.data) {
        setBots(bots.map(b => b.id === botId ? res.data! : b));
        setSuccess('Bot started');
      } else {
        setError(res.error || 'Failed to start bot');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to start bot');
    } finally {
      setLoading(false);
    }
  }, [bots]);
  
  const handleStopBot = useCallback(async (botId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.stopBot(botId);
      if (res.success && res.data) {
        setBots(bots.map(b => b.id === botId ? res.data! : b));
        setSuccess('Bot stopped');
      } else {
        setError(res.error || 'Failed to stop bot');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to stop bot');
    } finally {
      setLoading(false);
    }
  }, [bots]);
  
  const handlePauseBot = useCallback(async (botId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.pauseBot(botId);
      if (res.success && res.data) {
        setBots(bots.map(b => b.id === botId ? res.data! : b));
        setSuccess('Bot paused');
      } else {
        setError(res.error || 'Failed to pause bot');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to pause bot');
    } finally {
      setLoading(false);
    }
  }, [bots]);
  
  const handleDeleteBot = useCallback(async (botId: string) => {
    setConfirmDialog({
      open: true,
      title: 'Delete Bot',
      message: 'Are you sure you want to delete this bot? This action cannot be undone.',
      action: `delete:${botId}`,
    });
  }, []);
  
  const confirmAction = useCallback(async (action: string) => {
    if (action.startsWith('delete:')) {
      const botId = action.split(':')[1];
      setLoading(true);
      setError(null);
      try {
        const res = await api.deleteBot(botId);
        if (res.success) {
          setBots(bots.filter(b => b.id !== botId));
          setSuccess('Bot deleted');
        } else {
          setError(res.error || 'Failed to delete bot');
        }
      } catch (err: any) {
        setError(err.message || 'Failed to delete bot');
      } finally {
        setLoading(false);
        setConfirmDialog({ open: false, title: '', message: '', action: '' });
      }
    }
  }, [bots]);
  
  // ============================================================================
  // Helper Functions
  // ============================================================================
  
  const formatCurrency = (amount: number) => {
    return amount.toLocaleString('en-US', { style: 'currency', currency: 'USD' });
  };
  
  const formatNumber = (num: number) => {
    return num.toLocaleString('en-US');
  };
  
  const formatPercent = (percent: number) => {
    return `${percent.toFixed(1)}%`;
  };
  
  const formatDate = (timestamp: number) => {
    return new Date(timestamp).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };
  
  // ============================================================================
  // Render
  // ============================================================================
  
  if (!isConnected) {
    return (
      <Box sx={{ p: 4, textAlign: 'center', bgcolor: bgPrimary, color: textPrimary, minHeight: '100vh' }}>
        <Typography variant="h4" gutterBottom>Bot Platform Dashboard</Typography>
        <Typography color="text.secondary" sx={{ mb: 4 }}>
          Sign in with your bot-platform credentials to manage trading bots
        </Typography>
        {error && <Alert severity="error" sx={{ mb: 2, maxWidth: 400, mx: 'auto' }}>{error}</Alert>}
        <Button variant="contained" size="large" onClick={handleConnect} disabled={loading}>
          {loading ? 'Connecting...' : 'Sign In'}
        </Button>

        <Dialog open={loginDialog} onClose={() => setLoginDialog(false)} PaperProps={{ sx: { bgcolor: bgCard } }}>
          <DialogTitle sx={{ color: textPrimary }}>Bot Platform Login</DialogTitle>
          <DialogContent>
            {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
            <TextField
              autoFocus fullWidth margin="dense" label="Username"
              value={loginForm.username}
              onChange={(e) => setLoginForm({ ...loginForm, username: e.target.value })}
              sx={{ input: { color: textPrimary }, label: { color: textSecondary } }}
            />
            <TextField
              fullWidth margin="dense" label="Password" type="password"
              value={loginForm.password}
              onChange={(e) => setLoginForm({ ...loginForm, password: e.target.value })}
              onKeyDown={(e) => { if (e.key === 'Enter') handleLogin(); }}
              sx={{ input: { color: textPrimary }, label: { color: textSecondary } }}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setLoginDialog(false)}>Cancel</Button>
            <Button variant="contained" onClick={handleLogin} disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>
          </DialogActions>
        </Dialog>
      </Box>
    );
  }
  
  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Box>
          <Typography variant="h5" fontWeight="bold">Bot Platform</Typography>
          <Typography variant="body2" color="text.secondary">
            Role: {currentUser?.role.toUpperCase()} | Address: {currentUser?.address}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button variant="outlined" startIcon={<Refresh />} onClick={loadData}>Refresh</Button>
          {currentUser?.role === 'admin' && (
            <Button variant="contained" startIcon={<People />} onClick={() => setCreateUserDialog(true)}>
              Add User
            </Button>
          )}
          <Button variant="contained" startIcon={<Add />} onClick={() => setCreateBotDialog(true)}>
            Create Bot
          </Button>
          <Button variant="outlined" color="error" startIcon={<Stop />} onClick={handleDisconnect}>
            Sign Out
          </Button>
        </Box>
      </Box>
      
      {/* Alerts */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess(null)}>
          {success}
        </Alert>
      )}
      
      {/* Loading */}
      {loading && <LinearProgress sx={{ mb: 2 }} />}
      
      {/* Stats Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Total Bots</Typography>
              <Typography variant="h4">{bots.length}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Total Profit</Typography>
              <Typography variant="h4" color="success.main">
                {formatCurrency(bots.reduce((a, b) => a + b.profit, 0))}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Total Volume</Typography>
              <Typography variant="h4">
                {formatCurrency(bots.reduce((a, b) => a + b.volume, 0))}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Active Bots</Typography>
              <Typography variant="h4">
                {bots.filter(b => b.status === 'running').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
      
      {/* Tabs */}
      <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ mb: 3 }}>
        <Tab label="Bots" icon={<Dashboard />} iconPosition="start" />
        <Tab label="Transactions" icon={<History />} iconPosition="start" />
        <Tab label="Exchanges" icon={<Store />} iconPosition="start" />
        {currentUser?.role === 'admin' && (
          <Tab label="Users" icon={<People />} iconPosition="start" />
        )}
        <Tab label="Settings" icon={<Settings />} iconPosition="start" />
      </Tabs>
      
      {/* Bots Tab */}
      {activeTab === 0 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Status</TableCell>
                <TableCell align="right">Profit</TableCell>
                <TableCell align="right">Volume</TableCell>
                <TableCell align="right">Trades</TableCell>
                <TableCell align="right">Win Rate</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {bots.map(bot => (
                <TableRow key={bot.id}>
                  <TableCell>{bot.name}</TableCell>
                  <TableCell>
                    <Chip 
                      label={bot.type.replace('_', ' ')} 
                      size="small" 
                      icon={BOT_TYPES.find(t => t.value === bot.type)?.icon}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip 
                      label={bot.status}
                      size="small"
                      color={bot.status === 'running' ? 'success' : bot.status === 'paused' ? 'warning' : 'default'}
                    />
                  </TableCell>
                  <TableCell align="right" sx={{ color: bot.profit >= 0 ? 'success.main' : 'error.main' }}>
                    {formatCurrency(bot.profit)}
                  </TableCell>
                  <TableCell align="right">{formatCurrency(bot.volume)}</TableCell>
                  <TableCell align="right">{bot.trades}</TableCell>
                  <TableCell align="right">{formatPercent(bot.winRate)}</TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      {bot.status === 'stopped' ? (
                        <IconButton size="small" color="success" onClick={() => handleStartBot(bot.id)}>
                          <PlayArrow />
                        </IconButton>
                      ) : (
                        <IconButton size="small" color="warning" onClick={() => handlePauseBot(bot.id)}>
                          <Pause />
                        </IconButton>
                      )}
                      <IconButton size="small" color="error" onClick={() => handleStopBot(bot.id)}>
                        <Stop />
                      </IconButton>
                      <IconButton size="small" onClick={() => handleDeleteBot(bot.id)}>
                        <Delete />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      
      {/* Transactions Tab */}
      {activeTab === 1 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Time</TableCell>
                <TableCell>Bot</TableCell>
                <TableCell>Exchange</TableCell>
                <TableCell>Type</TableCell>
                <TableCell align="right">Amount</TableCell>
                <TableCell align="right">Price</TableCell>
                <TableCell align="right">Profit</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {transactions.map(tx => (
                <TableRow key={tx.id}>
                  <TableCell>{formatDate(tx.timestamp)}</TableCell>
                  <TableCell>{bots.find(b => b.id === tx.botId)?.name}</TableCell>
                  <TableCell>{tx.exchange}</TableCell>
                  <TableCell sx={{ color: tx.type === 'buy' ? 'success.main' : 'error.main' }}>
                    {tx.type.toUpperCase()}
                  </TableCell>
                  <TableCell align="right">{formatCurrency(tx.amount)}</TableCell>
                  <TableCell align="right">{formatCurrency(tx.price)}</TableCell>
                  <TableCell align="right" sx={{ color: tx.profit >= 0 ? 'success.main' : 'error.main' }}>
                    {formatCurrency(tx.profit)}
                  </TableCell>
                  <TableCell>
                    <Chip 
                      label={tx.status}
                      size="small"
                      color={tx.status === 'success' ? 'success' : 'error'}
                      icon={tx.status === 'success' ? <CheckCircle /> : <ErrorIcon />}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      
      {/* Exchanges Tab */}
      {activeTab === 2 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Status</TableCell>
                <TableCell align="right">Min Trade</TableCell>
                <TableCell align="right">Max Trade</TableCell>
                <TableCell align="right">Fee</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {exchanges.map(ex => (
                <TableRow key={ex.id}>
                  <TableCell>{ex.name}</TableCell>
                  <TableCell>
                    <Chip 
                      label={ex.isActive ? 'Active' : 'Inactive'}
                      size="small"
                      color={ex.isActive ? 'success' : 'default'}
                    />
                  </TableCell>
                  <TableCell align="right">{formatCurrency(ex.minTrade)}</TableCell>
                  <TableCell align="right">{formatCurrency(ex.maxTrade)}</TableCell>
                  <TableCell align="right">{ex.fee}%</TableCell>
                  <TableCell>
                    <FormControlLabel
                      control={
                        <Switch 
                          checked={ex.isActive}
                          onChange={() => {
                            setExchanges(exchanges.map(e => 
                              e.id === ex.id ? { ...e, isActive: !e.isActive } : e
                            ));
                          }}
                        />
                      }
                      label={ex.isActive ? 'Enabled' : 'Disabled'}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      
      {/* Users Tab (Admin Only) */}
      {activeTab === 3 && currentUser?.role === 'admin' && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Address</TableCell>
                <TableCell>Role</TableCell>
                <TableCell>Email</TableCell>
                <TableCell>Created</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map(user => (
                <TableRow key={user.id}>
                  <TableCell>{user.name}</TableCell>
                  <TableCell>{user.address}</TableCell>
                  <TableCell>
                    <Chip 
                      label={user.role}
                      size="small"
                      color={user.role === 'admin' ? 'error' : user.role === 'operator' ? 'warning' : 'primary'}
                    />
                  </TableCell>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>{formatDate(user.createdAt)}</TableCell>
                  <TableCell>
                    <Chip 
                      label={user.isActive ? 'Active' : 'Inactive'}
                      size="small"
                      color={user.isActive ? 'success' : 'error'}
                    />
                  </TableCell>
                  <TableCell>
                    {user.role !== 'admin' && (
                      <IconButton size="small" onClick={() => handleDeleteUser(user.id)}>
                        <Delete />
                      </IconButton>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      
      {/* Settings Tab */}
      {activeTab === 4 && (
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>Platform Settings</Typography>
            <Divider sx={{ my: 2 }} />
            
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <Typography variant="subtitle1" gutterBottom>Protocol Fee</Typography>
                <TextField fullWidth defaultValue="0.5" type="number" size="small" />%
              </Grid>
              <Grid item xs={12} md={6}>
                <Typography variant="subtitle1" gutterBottom>Verification Fee</Typography>
                <TextField fullWidth defaultValue="0.05" type="number" size="small" /> ETH
              </Grid>
              <Grid item xs={12}>
                <FormControlLabel
                  control={<Switch defaultChecked />}
                  label="Require Token Verification"
                />
              </Grid>
              <Grid item xs={12}>
                <FormControlLabel
                  control={<Switch defaultChecked />}
                  label="Require KYC"
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      )}
      
      {/* Create Bot Dialog */}
      <Dialog open={createBotDialog} onClose={() => setCreateBotDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Bot</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              fullWidth
              label="Bot Name"
              value={newBot.name}
              onChange={(e) => setNewBot({ ...newBot, name: e.target.value })}
            />
            <FormControl fullWidth>
              <InputLabel>Bot Type</InputLabel>
              <Select
                value={newBot.type}
                label="Bot Type"
                onChange={(e) => setNewBot({ ...newBot, type: e.target.value as Bot['type'] })}
              >
                {BOT_TYPES.map(bt => (
                  <MenuItem key={bt.value} value={bt.value}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {bt.icon} {bt.label}
                    </Box>
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField
              fullWidth
              label="Min Investment (USD)"
              type="number"
              value={newBot.minInvestment}
              onChange={(e) => setNewBot({ ...newBot, minInvestment: parseInt(e.target.value) })}
            />
            <TextField
              fullWidth
              label="Max Investment (USD)"
              type="number"
              value={newBot.maxInvestment}
              onChange={(e) => setNewBot({ ...newBot, maxInvestment: parseInt(e.target.value) })}
            />
            <TextField
              fullWidth
              label="Target APY (%)"
              type="number"
              value={newBot.targetApy}
              onChange={(e) => setNewBot({ ...newBot, targetApy: parseInt(e.target.value) })}
            />
            <Box>
              <Typography variant="subtitle2" gutterBottom>Risk Level: {newBot.riskLevel}</Typography>
              <Slider
                value={newBot.riskLevel}
                onChange={(_, v) => setNewBot({ ...newBot, riskLevel: v as number })}
                min={1}
                max={10}
                marks={[
                  { value: 1, label: 'Low' },
                  { value: 5, label: 'Medium' },
                  { value: 10, label: 'High' },
                ]}
              />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateBotDialog(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreateBot} disabled={loading}>
            Create Bot
          </Button>
        </DialogActions>
      </Dialog>
      
      {/* Create User Dialog */}
      <Dialog open={createUserDialog} onClose={() => setCreateUserDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add New User</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              fullWidth
              label="Name"
              value={newUser.name}
              onChange={(e) => setNewUser({ ...newUser, name: e.target.value })}
            />
            <TextField
              fullWidth
              label="Email"
              type="email"
              value={newUser.email}
              onChange={(e) => setNewUser({ ...newUser, email: e.target.value })}
            />
            <TextField
              fullWidth
              label="Wallet Address"
              value={newUser.address}
              onChange={(e) => setNewUser({ ...newUser, address: e.target.value })}
            />
            <FormControl fullWidth>
              <InputLabel>Role</InputLabel>
              <Select
                value={newUser.role}
                label="Role"
                onChange={(e) => setNewUser({ ...newUser, role: e.target.value as User['role'] })}
              >
                <MenuItem value="admin">Admin</MenuItem>
                <MenuItem value="operator">Bot Operator</MenuItem>
                <MenuItem value="client">Client</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateUserDialog(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreateUser} disabled={loading}>
            Add User
          </Button>
        </DialogActions>
      </Dialog>
      
      {/* Confirm Dialog */}
      <Dialog open={confirmDialog.open} onClose={() => setConfirmDialog({ open: false, title: '', message: '', action: '' })}>
        <DialogTitle>{confirmDialog.title}</DialogTitle>
        <DialogContent>
          <Typography>{confirmDialog.message}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDialog({ open: false, title: '', message: '', action: '' })}>Cancel</Button>
          <Button variant="contained" color="error" onClick={() => confirmAction(confirmDialog.action)}>
            Confirm
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}