'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, IconButton, Select, MenuItem, FormControl, InputLabel,
  Dialog, DialogTitle, DialogContent, DialogActions, Tabs, Tab,
  LinearProgress, Snackbar, Alert, CircularProgress,
  Slider, InputAdornment, Divider, ToggleButton, ToggleButtonGroup
} from '@mui/material';
import {
  Schedule, TrendingUp, TrendingDown, Add, Remove, PlayArrow,
  Pause, Delete, Refresh, ShowChart, AccessTime, CheckCircle
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';
import { api } from '@/lib/api/client';

// ============================================================================
// Types & Interfaces
// ============================================================================

interface TWAPOrder {
  id: string;
  userAddress: string;
  tokenIn: string;
  tokenOut: string;
  totalAmount: string;
  filledAmount: string;
  numOrders: number;
  completedOrders: number;
  intervalMinutes: number;
  startTime: number;
  endTime: number;
  nextExecutionTime: number;
  priceType: 'market' | 'limit';
  limitPrice?: number;
  slippageBps: number;
  status: 'active' | 'paused' | 'completed' | 'cancelled';
  createdAt: number;
  lastExecutionTime?: number;
  lastExecutionPrice?: number;
}

interface DCAOrder {
  id: string;
  userAddress: string;
  tokenIn: string;
  tokenOut: string;
  amountPerOrder: string;
  totalAmount: string;
  filledAmount: string;
  frequency: 'daily' | 'weekly' | 'monthly';
  dayOfWeek?: number;
  dayOfMonth?: number;
  hourOfDay: number;
  nextExecutionTime: number;
  status: 'active' | 'paused' | 'completed' | 'cancelled';
  createdAt: number;
  totalOrders: number;
  completedOrders: number;
}

interface OrderExecution {
  id: string;
  orderId: string;
  orderType: 'twap' | 'dca';
  tokenIn: string;
  tokenOut: string;
  amountIn: string;
  amountOut: string;
  price: number;
  timestamp: number;
  txHash: string;
  status: 'success' | 'failed' | 'pending';
}

// ============================================================================
// Constants
// ============================================================================

const TOKENS = [
  { symbol: 'ETH', name: 'Ethereum', icon: '🔷', price: 2450 },
  { symbol: 'BTC', name: 'Bitcoin', icon: '₿', price: 62500 },
  { symbol: 'USDC', name: 'USD Coin', icon: '💵', price: 1 },
  { symbol: 'USDT', name: 'Tether', icon: '💰', price: 1 },
  { symbol: 'LINK', name: 'Chainlink', icon: '🔗', price: 18.5 },
  { symbol: 'UNI', name: 'Uniswap', icon: '🦄', price: 12.5 },
  { symbol: 'AAVE', name: 'Aave', icon: '👻', price: 285 },
  { symbol: 'MATIC', name: 'Polygon', icon: '🟣', price: 0.85 },
];

const FREQUENCIES = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
];

const INTERVALS = [
  { value: 15, label: '15 minutes' },
  { value: 30, label: '30 minutes' },
  { value: 60, label: '1 hour' },
  { value: 240, label: '4 hours' },
  { value: 1440, label: 'Daily' },
];

const DAYS_OF_WEEK = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

// ============================================================================
// Utility Functions
// ============================================================================

function formatAddress(address: string, chars: number = 4): string {
  return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`;
}

function formatAmount(amount: string, decimals: number = 18): string {
  if (!amount || amount === '0') return '0';
  try {
    const num = Number(amount) / Math.pow(10, decimals);
    if (num < 0.0001) return '<0.0001';
    return num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 6 });
  } catch {
    return '0';
  }
}

function formatUSD(amount: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount);
}

function formatDateTime(timestamp: number): string {
  return new Date(timestamp).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
}

function timeUntil(timestamp: number): string {
  const diff = timestamp - Date.now();
  if (diff <= 0) return 'Now';
  const hours = Math.floor(diff / 3600000);
  const minutes = Math.floor((diff % 3600000) / 60000);
  if (hours > 24) return `${Math.floor(hours / 24)}d ${hours % 24}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

// ============================================================================
// Normalizers (map backend payloads to local types)
// ============================================================================

function normalizeTwap(o: any): TWAPOrder {
  return {
    id: String(o.id ?? ''),
    userAddress: String(o.userAddress ?? o.user_address ?? ''),
    tokenIn: String(o.tokenIn ?? o.token_in ?? ''),
    tokenOut: String(o.tokenOut ?? o.token_out ?? ''),
    totalAmount: String(o.totalAmount ?? o.total_amount ?? '0'),
    filledAmount: String(o.filledAmount ?? o.filled_amount ?? '0'),
    numOrders: Number(o.numOrders ?? o.num_orders ?? 0),
    completedOrders: Number(o.completedOrders ?? o.completed_orders ?? 0),
    intervalMinutes: Number(o.intervalMinutes ?? o.interval_minutes ?? 0),
    startTime: Number(o.startTime ?? o.start_time ?? 0),
    endTime: Number(o.endTime ?? o.end_time ?? 0),
    nextExecutionTime: Number(o.nextExecutionTime ?? o.next_execution_time ?? 0),
    priceType: (o.priceType ?? o.price_type ?? 'market') as TWAPOrder['priceType'],
    limitPrice: o.limitPrice != null ? Number(o.limitPrice) : o.limit_price != null ? Number(o.limit_price) : undefined,
    slippageBps: Number(o.slippageBps ?? o.slippage_bps ?? 0),
    status: (o.status ?? 'active') as TWAPOrder['status'],
    createdAt: Number(o.createdAt ?? o.created_at ?? 0),
    lastExecutionTime: o.lastExecutionTime != null ? Number(o.lastExecutionTime) : o.last_execution_time != null ? Number(o.last_execution_time) : undefined,
    lastExecutionPrice: o.lastExecutionPrice != null ? Number(o.lastExecutionPrice) : o.last_execution_price != null ? Number(o.last_execution_price) : undefined,
  };
}

function normalizeDca(o: any): DCAOrder {
  return {
    id: String(o.id ?? ''),
    userAddress: String(o.userAddress ?? o.user_address ?? ''),
    tokenIn: String(o.tokenIn ?? o.token_in ?? ''),
    tokenOut: String(o.tokenOut ?? o.token_out ?? ''),
    amountPerOrder: String(o.amountPerOrder ?? o.amount_per_order ?? '0'),
    totalAmount: String(o.totalAmount ?? o.total_amount ?? '0'),
    filledAmount: String(o.filledAmount ?? o.filled_amount ?? '0'),
    frequency: (o.frequency ?? 'weekly') as DCAOrder['frequency'],
    dayOfWeek: o.dayOfWeek != null ? Number(o.dayOfWeek) : o.day_of_week != null ? Number(o.day_of_week) : undefined,
    dayOfMonth: o.dayOfMonth != null ? Number(o.dayOfMonth) : o.day_of_month != null ? Number(o.day_of_month) : undefined,
    hourOfDay: Number(o.hourOfDay ?? o.hour_of_day ?? 0),
    nextExecutionTime: Number(o.nextExecutionTime ?? o.next_execution_time ?? 0),
    status: (o.status ?? 'active') as DCAOrder['status'],
    createdAt: Number(o.createdAt ?? o.created_at ?? 0),
    totalOrders: Number(o.totalOrders ?? o.total_orders ?? 0),
    completedOrders: Number(o.completedOrders ?? o.completed_orders ?? 0),
  };
}

function normalizeExec(e: any): OrderExecution {
  return {
    id: String(e.id ?? ''),
    orderId: String(e.orderId ?? e.order_id ?? ''),
    orderType: (e.orderType ?? e.order_type ?? 'twap') as OrderExecution['orderType'],
    tokenIn: String(e.tokenIn ?? e.token_in ?? ''),
    tokenOut: String(e.tokenOut ?? e.token_out ?? ''),
    amountIn: String(e.amountIn ?? e.amount_in ?? '0'),
    amountOut: String(e.amountOut ?? e.amount_out ?? '0'),
    price: Number(e.price ?? 0),
    timestamp: Number(e.timestamp ?? 0),
    txHash: String(e.txHash ?? e.tx_hash ?? ''),
    status: (e.status ?? 'pending') as OrderExecution['status'],
  };
}

// ============================================================================
// Main TWAP/DCA Page Component
// ============================================================================

export default function TWAPPage() {
  // State
  const [twapOrders, setTwapOrders] = useState<TWAPOrder[]>([]);
  const [dcaOrders, setDcaOrders] = useState<DCAOrder[]>([]);
  const [executionHistory, setExecutionHistory] = useState<OrderExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState(0);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [orderType, setOrderType] = useState<'twap' | 'dca'>('twap');

  // Create form state
  const [tokenIn, setTokenIn] = useState('USDC');
  const [tokenOut, setTokenOut] = useState('ETH');
  const [totalAmount, setTotalAmount] = useState('');
  const [numOrders, setNumOrders] = useState(10);
  const [intervalMinutes, setIntervalMinutes] = useState(60);
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly'>('weekly');
  const [dayOfWeek, setDayOfWeek] = useState(1);
  const [dayOfMonth, setDayOfMonth] = useState(1);
  const [hourOfDay, setHourOfDay] = useState(9);
  const [priceType, setPriceType] = useState<'market' | 'limit'>('market');
  const [limitPrice, setLimitPrice] = useState('');
  const [slippageBps, setSlippageBps] = useState(50);
  const [creating, setCreating] = useState(false);

  // Snackbar
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success' as 'success' | 'error' | 'info'
  });

  // ============================================================================
  // Data Loading
  // ============================================================================

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.getTwapOrders();
      const data = (res.data || {}) as any;
      const twapList: TWAPOrder[] = Array.isArray(data.twapOrders)
        ? data.twapOrders.map((o: any) => normalizeTwap(o))
        : Array.isArray(data) ? data.map((o: any) => normalizeTwap(o)) : [];
      const dcaList: DCAOrder[] = Array.isArray(data.dcaOrders)
        ? data.dcaOrders.map((o: any) => normalizeDca(o))
        : [];
      const execList: OrderExecution[] = Array.isArray(data.executions)
        ? data.executions.map((e: any) => normalizeExec(e))
        : [];

      setTwapOrders(twapList);
      setDcaOrders(dcaList);
      setExecutionHistory(execList);
    } catch (error: any) {
      setSnackbar({
        open: true,
        message: error?.response?.data?.error || error?.message || 'Failed to load orders',
        severity: 'error',
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // ============================================================================
  // Order Management
  // ============================================================================

  const handleCreateOrder = async () => {
    if (!totalAmount || parseFloat(totalAmount) <= 0) {
      setSnackbar({ open: true, message: 'Please enter a valid amount', severity: 'error' });
      return;
    }

    setCreating(true);
    try {
      const payload: any = {
        tokenIn,
        tokenOut,
        totalAmount,
        numOrders,
        priceType,
        limitPrice: priceType === 'limit' ? limitPrice : undefined,
        slippageBps,
      };

      if (orderType === 'twap') {
        payload.intervalMinutes = intervalMinutes;
        payload.type = 'twap';
        const res = await api.createTwapOrder(payload);
        const created = res.data;
        if (created) {
          setTwapOrders(prev => [...prev, normalizeTwap(created)]);
        }
      } else {
        payload.frequency = frequency;
        payload.dayOfWeek = frequency === 'weekly' ? dayOfWeek : undefined;
        payload.dayOfMonth = frequency === 'monthly' ? dayOfMonth : undefined;
        payload.hourOfDay = hourOfDay;
        payload.amountPerOrder = (parseFloat(totalAmount) / numOrders).toString();
        payload.totalOrders = numOrders;
        payload.type = 'dca';
        const res = await api.createTwapOrder(payload);
        const created = res.data;
        if (created) {
          setDcaOrders(prev => [...prev, normalizeDca(created)]);
        }
      }

      setShowCreateDialog(false);
      setTotalAmount('');
      setSnackbar({ open: true, message: `${orderType.toUpperCase()} order created successfully!`, severity: 'success' });
    } catch (error: any) {
      setSnackbar({
        open: true,
        message: error?.response?.data?.error || error?.message || 'Failed to create order',
        severity: 'error',
      });
    } finally {
      setCreating(false);
    }
  };

  const handlePauseResume = async (orderId: string, type: 'twap' | 'dca', currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'paused' : 'active';
    
    try {
      await api.updateTwapOrder(orderId, { status: newStatus });
      if (type === 'twap') {
        setTwapOrders(prev => prev.map(o =>
          o.id === orderId ? { ...o, status: newStatus as TWAPOrder['status'] } : o
        ));
      } else {
        setDcaOrders(prev => prev.map(o =>
          o.id === orderId ? { ...o, status: newStatus as DCAOrder['status'] } : o
        ));
      }
      
      setSnackbar({ open: true, message: `Order ${newStatus}`, severity: 'success' });
    } catch (error: any) {
      setSnackbar({
        open: true,
        message: error?.response?.data?.error || error?.message || 'Failed to update order',
        severity: 'error',
      });
    }
  };

  const handleCancel = async (orderId: string, type: 'twap' | 'dca') => {
    try {
      await api.cancelTwapOrder(orderId);
      if (type === 'twap') {
        setTwapOrders(prev => prev.map(o =>
          o.id === orderId ? { ...o, status: 'cancelled' as TWAPOrder['status'] } : o
        ));
      } else {
        setDcaOrders(prev => prev.map(o =>
          o.id === orderId ? { ...o, status: 'cancelled' as DCAOrder['status'] } : o
        ));
      }
      
      setSnackbar({ open: true, message: 'Order cancelled', severity: 'success' });
    } catch (error: any) {
      setSnackbar({
        open: true,
        message: error?.response?.data?.error || error?.message || 'Failed to cancel order',
        severity: 'error',
      });
    }
  };

  // ============================================================================
  // Statistics
  // ============================================================================

  const totalTwapVolume = twapOrders.reduce((sum, o) => sum + parseFloat(o.totalAmount), 0);
  const totalDCAVolume = dcaOrders.reduce((sum, o) => sum + parseFloat(o.totalAmount), 0);
  const totalFilledVolume = [...twapOrders, ...dcaOrders].reduce((sum, o) => sum + parseFloat(o.filledAmount), 0);
  const activeOrders = twapOrders.filter(o => o.status === 'active').length + dcaOrders.filter(o => o.status === 'active').length;

  // ============================================================================
  // Main Render
  // ============================================================================

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'var(--bg-primary)', p: 3 }}>
      <Box sx={{ maxWidth: 1400, mx: 'auto' }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Box>
            <Typography variant="h4" sx={{ color: 'white', fontWeight: 'bold' }}>
              📅 TWAP & DCA Orders
            </Typography>
            <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mt: 1 }}>
              Schedule recurring orders and split large orders over time
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="outlined"
              startIcon={<Refresh />}
              onClick={loadData}
              sx={{ borderColor: 'var(--bg-tertiary)', color: 'white' }}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              startIcon={<Add />}
              onClick={() => setShowCreateDialog(true)}
              sx={{ bgcolor: '#00d4aa', color: 'black' }}
            >
              Create Order
            </Button>
          </Box>
        </Box>

        {/* Stats */}
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, mb: 4 }}>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Total TWAP Volume</Typography>
              <Typography variant="h5" sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
                {formatUSD(totalTwapVolume)}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Total DCA Volume</Typography>
              <Typography variant="h5" sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
                {formatUSD(totalDCAVolume)}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Filled Volume</Typography>
              <Typography variant="h5" sx={{ color: '#ff9800', fontWeight: 'bold' }}>
                {formatUSD(totalFilledVolume)}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Active Orders</Typography>
              <Typography variant="h5" sx={{ color: 'white', fontWeight: 'bold' }}>
                {activeOrders}
              </Typography>
            </CardContent>
          </Card>
        </Box>

        {/* Tabs */}
        <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3, mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={(_, v) => setActiveTab(v)}
            sx={{
              borderBottom: '1px solid #2a2a3e',
              '& .MuiTab-root': { color: 'var(--text-secondary)' },
              '& .Mui-selected': { color: '#00d4aa' },
            }}
          >
            <Tab label={`TWAP Orders (${twapOrders.length})`} />
            <Tab label={`DCA Orders (${dcaOrders.length})`} />
            <Tab label="Execution History" />
          </Tabs>

          <CardContent sx={{ p: 3 }}>
            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
                <CircularProgress sx={{ color: '#00d4aa' }} />
              </Box>
            ) : activeTab === 0 ? (
              /* TWAP Orders */
              <Box>
                {twapOrders.length === 0 ? (
                  <Box sx={{ textAlign: 'center', py: 5 }}>
                    <Typography sx={{ color: 'var(--text-secondary)', mb: 2 }}>No TWAP orders yet</Typography>
                    <Button
                      variant="contained"
                      startIcon={<Add />}
                      onClick={() => { setOrderType('twap'); setShowCreateDialog(true); }}
                      sx={{ bgcolor: '#00d4aa', color: 'black' }}
                    >
                      Create Your First TWAP
                    </Button>
                  </Box>
                ) : (
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Pair</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Progress</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Total Amount</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Filled</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Interval</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Next Execution</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Status</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {twapOrders.map(order => (
                          <TableRow key={order.id} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{order.tokenIn}</Typography>
                                <TrendingDown sx={{ color: 'var(--text-secondary)', fontSize: 16 }} />
                                <Typography sx={{ color: '#ff9800', fontWeight: 'bold' }}>{order.tokenOut}</Typography>
                              </Box>
                            </TableCell>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <LinearProgress
                                  variant="determinate"
                                  value={(parseFloat(order.filledAmount) / parseFloat(order.totalAmount)) * 100}
                                  sx={{ width: 80, height: 6, borderRadius: 3, bgcolor: 'var(--bg-secondary)', '& .MuiLinearProgress-bar': { bgcolor: '#00d4aa' } }}
                                />
                                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                                  {order.completedOrders}/{order.numOrders}
                                </Typography>
                              </Box>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: 'white' }}>{formatUSD(parseFloat(order.totalAmount))}</Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: '#00d4aa' }}>{formatUSD(parseFloat(order.filledAmount))}</Typography>
                            </TableCell>
                            <TableCell>
                              <Chip label={`Every ${INTERVALS.find(i => i.value === order.intervalMinutes)?.label}`} size="small" sx={{ bgcolor: 'var(--bg-secondary)' }} />
                            </TableCell>
                            <TableCell>
                              <Typography sx={{ color: order.status === 'active' ? '#00d4aa' : 'var(--text-secondary)' }}>
                                {timeUntil(order.nextExecutionTime)}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={order.status}
                                size="small"
                                sx={{
                                  bgcolor: order.status === 'active' ? '#00d4aa20' : '#ff980020',
                                  color: order.status === 'active' ? '#00d4aa' : '#ff9800',
                                }}
                              />
                            </TableCell>
                            <TableCell align="right">
                              <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                                <IconButton
                                  size="small"
                                  onClick={() => handlePauseResume(order.id, 'twap', order.status)}
                                  sx={{ color: order.status === 'active' ? '#ff9800' : '#00d4aa' }}
                                >
                                  {order.status === 'active' ? <Pause fontSize="small" /> : <PlayArrow fontSize="small" />}
                                </IconButton>
                                <IconButton
                                  size="small"
                                  onClick={() => handleCancel(order.id, 'twap')}
                                  sx={{ color: '#ff5722' }}
                                >
                                  <Delete fontSize="small" />
                                </IconButton>
                              </Box>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Box>
            ) : activeTab === 1 ? (
              /* DCA Orders */
              <Box>
                {dcaOrders.length === 0 ? (
                  <Box sx={{ textAlign: 'center', py: 5 }}>
                    <Typography sx={{ color: 'var(--text-secondary)', mb: 2 }}>No DCA orders yet</Typography>
                    <Button
                      variant="contained"
                      startIcon={<Add />}
                      onClick={() => { setOrderType('dca'); setShowCreateDialog(true); }}
                      sx={{ bgcolor: '#00d4aa', color: 'black' }}
                    >
                      Create Your First DCA
                    </Button>
                  </Box>
                ) : (
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Pair</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Frequency</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Per Order</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Total</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Filled</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Next Execution</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Status</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {dcaOrders.map(order => (
                          <TableRow key={order.id} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{order.tokenIn}</Typography>
                                <TrendingDown sx={{ color: 'var(--text-secondary)', fontSize: 16 }} />
                                <Typography sx={{ color: '#ff9800', fontWeight: 'bold' }}>{order.tokenOut}</Typography>
                              </Box>
                            </TableCell>
                            <TableCell>
                              <Chip label={FREQUENCIES.find(f => f.value === order.frequency)?.label || order.frequency} size="small" sx={{ bgcolor: 'var(--bg-secondary)' }} />
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: 'white' }}>{formatUSD(parseFloat(order.amountPerOrder))}</Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: 'white' }}>{formatUSD(parseFloat(order.totalAmount))}</Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: '#00d4aa' }}>{formatUSD(parseFloat(order.filledAmount))}</Typography>
                            </TableCell>
                            <TableCell>
                              <Typography sx={{ color: order.status === 'active' ? '#00d4aa' : 'var(--text-secondary)' }}>
                                {formatDateTime(order.nextExecutionTime)}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={order.status}
                                size="small"
                                sx={{
                                  bgcolor: order.status === 'active' ? '#00d4aa20' : '#ff980020',
                                  color: order.status === 'active' ? '#00d4aa' : '#ff9800',
                                }}
                              />
                            </TableCell>
                            <TableCell align="right">
                              <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                                <IconButton
                                  size="small"
                                  onClick={() => handlePauseResume(order.id, 'dca', order.status)}
                                  sx={{ color: order.status === 'active' ? '#ff9800' : '#00d4aa' }}
                                >
                                  {order.status === 'active' ? <Pause fontSize="small" /> : <PlayArrow fontSize="small" />}
                                </IconButton>
                                <IconButton
                                  size="small"
                                  onClick={() => handleCancel(order.id, 'dca')}
                                  sx={{ color: '#ff5722' }}
                                >
                                  <Delete fontSize="small" />
                                </IconButton>
                              </Box>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Box>
            ) : (
              /* Execution History */
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Time</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Type</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Pair</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Amount In</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Amount Out</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Price</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Status</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {executionHistory.map(exec => (
                      <TableRow key={exec.id} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                        <TableCell>
                          <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                            {formatDateTime(exec.timestamp)}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip label={exec.orderType.toUpperCase()} size="small" sx={{ bgcolor: 'var(--bg-secondary)', textTransform: 'uppercase', fontSize: '0.65rem' }} />
                        </TableCell>
                        <TableCell>
                          <Typography sx={{ color: '#00d4aa' }}>{exec.tokenIn}</Typography>
                          <Typography sx={{ color: 'var(--text-secondary)', fontSize: '0.75rem' }}>→ {exec.tokenOut}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>{exec.amountIn}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: '#00d4aa' }}>{exec.amountOut}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>${exec.price.toLocaleString()}</Typography>
                        </TableCell>
                        <TableCell>
                          <Chip
                            icon={exec.status === 'success' ? <CheckCircle sx={{ fontSize: 14 }} /> : undefined}
                            label={exec.status}
                            size="small"
                            sx={{
                              bgcolor: exec.status === 'success' ? '#00d4aa20' : exec.status === 'pending' ? '#ff980020' : '#ff572220',
                              color: exec.status === 'success' ? '#00d4aa' : exec.status === 'pending' ? '#ff9800' : '#ff5722',
                            }}
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </CardContent>
        </Card>
      </Box>

      {/* Create Order Dialog */}
      <Dialog
        open={showCreateDialog}
        onClose={() => setShowCreateDialog(false)}
        maxWidth="md"
        fullWidth
        PaperProps={{ sx: { bgcolor: 'var(--bg-primary)', backgroundImage: 'none' } }}
      >
        <DialogTitle sx={{ color: 'white', display: 'flex', justifyContent: 'space-between' }}>
          Create {orderType.toUpperCase()} Order
          <IconButton onClick={() => setShowCreateDialog(false)} sx={{ color: 'white' }}>✕</IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 2 }}>
            {/* Order Type Toggle */}
            <Box sx={{ mb: 3 }}>
              <ToggleButtonGroup
                value={orderType}
                exclusive
                onChange={(_, v) => v && setOrderType(v)}
                fullWidth
                sx={{ '& .MuiToggleButton-root': { color: 'var(--text-secondary)', borderColor: 'var(--bg-tertiary)' }, '& .Mui-selected': { bgcolor: '#00d4aa', color: 'black' } }}
              >
                <ToggleButton value="twap">TWAP (Time-Weighted)</ToggleButton>
                <ToggleButton value="dca">DCA (Dollar-Cost Avg)</ToggleButton>
              </ToggleButtonGroup>
            </Box>

            {/* Token Selection */}
            <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 3 }}>
              <FormControl fullWidth size="small">
                <InputLabel sx={{ color: 'var(--text-secondary)' }}>From Token</InputLabel>
                <Select
                  value={tokenIn}
                  onChange={(e) => setTokenIn(e.target.value)}
                  label="From Token"
                  sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                >
                  {TOKENS.map(t => (
                    <MenuItem key={t.symbol} value={t.symbol}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <span>{t.icon}</span>
                        <span>{t.symbol}</span>
                      </Box>
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControl fullWidth size="small">
                <InputLabel sx={{ color: 'var(--text-secondary)' }}>To Token</InputLabel>
                <Select
                  value={tokenOut}
                  onChange={(e) => setTokenOut(e.target.value)}
                  label="To Token"
                  sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                >
                  {TOKENS.map(t => (
                    <MenuItem key={t.symbol} value={t.symbol}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <span>{t.icon}</span>
                        <span>{t.symbol}</span>
                      </Box>
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>

            {/* Amount */}
            <TextField
              fullWidth
              type="number"
              label="Total Amount"
              value={totalAmount}
              onChange={(e) => setTotalAmount(e.target.value)}
              InputProps={{
                startAdornment: <InputAdornment position="start" sx={{ color: 'var(--text-secondary)' }}>$</InputAdornment>,
              }}
              sx={{ mb: 3, '& .MuiInputLabel-root': { color: 'var(--text-secondary)' }, '& input': { color: 'white' }, '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } } }}
            />

            {/* TWAP Settings */}
            {orderType === 'twap' ? (
              <Box>
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 3 }}>
                  <FormControl fullWidth size="small">
                    <InputLabel sx={{ color: 'var(--text-secondary)' }}>Number of Orders</InputLabel>
                    <Select
                      value={numOrders}
                      onChange={(e) => setNumOrders(e.target.value as number)}
                      label="Number of Orders"
                      sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                    >
                      {[5, 10, 20, 50, 100].map(n => (
                        <MenuItem key={n} value={n}>{n} orders</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl fullWidth size="small">
                    <InputLabel sx={{ color: 'var(--text-secondary)' }}>Interval</InputLabel>
                    <Select
                      value={intervalMinutes}
                      onChange={(e) => setIntervalMinutes(e.target.value as number)}
                      label="Interval"
                      sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                    >
                      {INTERVALS.map(int => (
                        <MenuItem key={int.value} value={int.value}>{int.label}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>

                <FormControl fullWidth size="small" sx={{ mb: 3 }}>
                  <InputLabel sx={{ color: 'var(--text-secondary)' }}>Price Type</InputLabel>
                  <Select
                    value={priceType}
                    onChange={(e) => setPriceType(e.target.value as 'market' | 'limit')}
                    label="Price Type"
                    sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                  >
                    <MenuItem value="market">Market Price</MenuItem>
                    <MenuItem value="limit">Limit Price</MenuItem>
                  </Select>
                </FormControl>

                {priceType === 'limit' && (
                  <TextField
                    fullWidth
                    type="number"
                    label="Limit Price"
                    value={limitPrice}
                    onChange={(e) => setLimitPrice(e.target.value)}
                    sx={{ mb: 3, '& .MuiInputLabel-root': { color: 'var(--text-secondary)' }, '& input': { color: 'white' }, '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } } }}
                  />
                )}
              </Box>
            ) : (
              /* DCA Settings */
              <Box>
                <FormControl fullWidth size="small" sx={{ mb: 3 }}>
                  <InputLabel sx={{ color: 'var(--text-secondary)' }}>Frequency</InputLabel>
                  <Select
                    value={frequency}
                    onChange={(e) => setFrequency(e.target.value as 'daily' | 'weekly' | 'monthly')}
                    label="Frequency"
                    sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                  >
                    {FREQUENCIES.map(f => (
                      <MenuItem key={f.value} value={f.value}>{f.label}</MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 3 }}>
                  {frequency === 'weekly' && (
                    <FormControl fullWidth size="small">
                      <InputLabel sx={{ color: 'var(--text-secondary)' }}>Day of Week</InputLabel>
                      <Select
                        value={dayOfWeek}
                        onChange={(e) => setDayOfWeek(e.target.value as number)}
                        label="Day of Week"
                        sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                      >
                        {DAYS_OF_WEEK.map((day, i) => (
                          <MenuItem key={i} value={i}>{day}</MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  )}
                  {frequency === 'monthly' && (
                    <FormControl fullWidth size="small">
                      <InputLabel sx={{ color: 'var(--text-secondary)' }}>Day of Month</InputLabel>
                      <Select
                        value={dayOfMonth}
                        onChange={(e) => setDayOfMonth(e.target.value as number)}
                        label="Day of Month"
                        sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                      >
                        {Array.from({ length: 28 }, (_, i) => (
                          <MenuItem key={i + 1} value={i + 1}>{i + 1}</MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  )}
                  <FormControl fullWidth size="small">
                    <InputLabel sx={{ color: 'var(--text-secondary)' }}>Time of Day</InputLabel>
                    <Select
                      value={hourOfDay}
                      onChange={(e) => setHourOfDay(e.target.value as number)}
                      label="Time of Day"
                      sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                    >
                      {Array.from({ length: 24 }, (_, i) => (
                        <MenuItem key={i} value={i}>{i.toString().padStart(2, '0')}:00</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>
              </Box>
            )}

            {/* Slippage */}
            <Box sx={{ mb: 3 }}>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)', mb: 1, display: 'block' }}>
                Slippage Tolerance: {slippageBps / 100}%
              </Typography>
              <Slider
                value={slippageBps}
                onChange={(_, v) => setSlippageBps(v as number)}
                min={10}
                max={500}
                step={10}
                marks={[{ value: 50, label: '0.5%' }, { value: 100, label: '1%' }, { value: 500, label: '5%' }]}
                sx={{ color: '#00d4aa' }}
              />
            </Box>

            {/* Summary */}
            <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2 }}>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                {orderType === 'twap' ? (
                  <>Your order will be split into {numOrders} orders over {INTERVALS.find(i => i.value === intervalMinutes)?.label?.toLowerCase()}.</>
                ) : (
                  <>You will buy approximately {(parseFloat(totalAmount) / (orderType === 'dca' ? numOrders : 1)).toFixed(2)} {tokenOut} per order on a {frequency} basis.</>
                )}
              </Typography>
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 3 }}>
          <Button onClick={() => setShowCreateDialog(false)} sx={{ color: 'var(--text-secondary)' }}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleCreateOrder}
            disabled={creating || !totalAmount}
            sx={{ bgcolor: '#00d4aa', color: 'black' }}
          >
            {creating ? <CircularProgress size={20} sx={{ color: 'black' }} /> : `Create ${orderType.toUpperCase()}`}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={5000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ bgcolor: snackbar.severity === 'success' ? '#1b5e20' : snackbar.severity === 'error' ? '#b71c1c' : '#1a237e' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
}