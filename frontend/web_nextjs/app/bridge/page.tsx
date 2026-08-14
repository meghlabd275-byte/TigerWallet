'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  Select, MenuItem, FormControl, InputLabel, Chip,
  CircularProgress, Snackbar, Alert, IconButton,
  Stepper, Step, StepLabel, Divider, InputAdornment
} from '@mui/material';
import {
  SwapHoriz, ArrowForward, AccountBalance, CheckCircle,
  ErrorOutline, AccessTime, Speed, Security, Warning,
  ContentCopy, Refresh, ArrowDropDown
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';

// Same-origin API base: the Next.js app proxies /api/v1/* to the backend
// services (see app/api/v1/_proxy.ts). In the browser this resolves to the
// current host.
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

// ============================================================================
// Types & Interfaces
// ============================================================================

interface Chain {
  id: number;
  name: string;
  icon: string;
  color: string;
  rpcUrl: string;
  explorerUrl: string;
  nativeCurrency: string;
  avgBridgeTime: string;
  gasCost: string;
}

interface BridgeQuote {
  fromChain: number;
  toChain: number;
  token: string;
  amount: string;
  bridgeFee: number;
  networkFee: number;
  estimatedTime: string;
  receivedAmount: string;
  rate: number;
  availableRoutes: BridgeRoute[];
}

interface BridgeRoute {
  id: string;
  name: string;
  logo: string;
  fee: number;
  time: string;
  reliability: number;
  minAmount: number;
  maxAmount: number;
}

interface BridgeTransfer {
  id: string;
  fromChain: number;
  toChain: number;
  token: string;
  amount: string;
  status: 'pending' | 'submitted' | 'confirming' | 'completed' | 'failed';
  sourceTxHash?: string;
  destTxHash?: string;
  confirmations: number;
  requiredConfirmations: number;
  timestamp: number;
}

// ============================================================================
// Constants
// ============================================================================

// Hardcoded chains kept ONLY as an offline fallback; the live list is fetched
// from the canonical GET /api/v1/chains registry on mount (see fetchChains below).
const CHAINS_FALLBACK: Chain[] = [
  { id: 1, name: 'Ethereum', icon: '🔷', color: '#627EEA', rpcUrl: 'https://eth.llamarpc.com', explorerUrl: 'https://etherscan.io', nativeCurrency: 'ETH', avgBridgeTime: '10-15 min', gasCost: '$5-15' },
  { id: 56, name: 'BNB Chain', icon: '🟡', color: '#F3BA2F', rpcUrl: 'https://bsc-dataseed.binance.org', explorerUrl: 'https://bscscan.com', nativeCurrency: 'BNB', avgBridgeTime: '5-10 min', gasCost: '$1-3' },
  { id: 137, name: 'Polygon', icon: '🟣', color: '#8247E5', rpcUrl: 'https://polygon-rpc.com', explorerUrl: 'https://polygonscan.com', nativeCurrency: 'MATIC', avgBridgeTime: '7-12 min', gasCost: '$0.5-2' },
  { id: 42161, name: 'Arbitrum', icon: '🔵', color: '#28A0F0', rpcUrl: 'https://arb1.arbitrum.io/rpc', explorerUrl: 'https://arbiscan.io', nativeCurrency: 'ETH', avgBridgeTime: '10-15 min', gasCost: '$2-5' },
  { id: 10, name: 'Optimism', icon: '🔴', color: '#FF0420', rpcUrl: 'https://mainnet.optimism.io', explorerUrl: 'https://optimistic.etherscan.io', nativeCurrency: 'ETH', avgBridgeTime: '10-15 min', gasCost: '$1-3' },
  { id: 8453, name: 'Base', icon: '🔵', color: '#0052FF', rpcUrl: 'https://mainnet.base.org', explorerUrl: 'https://basescan.org', nativeCurrency: 'ETH', avgBridgeTime: '5-8 min', gasCost: '$0.5-2' },
  { id: 43114, name: 'Avalanche', icon: '🔶', color: '#E84142', rpcUrl: 'https://api.avax.network/ext/bc/C/rpc', explorerUrl: 'https://snowtrace.io', nativeCurrency: 'AVAX', avgBridgeTime: '3-5 min', gasCost: '$1-3' },
];

const TOKENS = [
  { symbol: 'ETH', name: 'Ethereum', decimals: 18, icon: '🔷', isNative: true },
  { symbol: 'USDC', name: 'USD Coin', decimals: 6, icon: '💵', isNative: false },
  { symbol: 'USDT', name: 'Tether', decimals: 6, icon: '💰', isNative: false },
  { symbol: 'DAI', name: 'Dai', decimals: 18, icon: '📎', isNative: false },
  { symbol: 'WBTC', name: 'Wrapped Bitcoin', decimals: 8, icon: '₿', isNative: false },
  { symbol: 'MATIC', name: 'Polygon', decimals: 18, icon: '🟣', isNative: false },
];

const BRIDGE_ROUTES: BridgeRoute[] = [
  { id: 'across', name: 'Across', logo: '🔄', fee: 0.09, time: '1-3 min', reliability: 99.2, minAmount: 10, maxAmount: 1000000 },
  { id: 'stargate', name: 'Stargate', logo: '🌉', fee: 0.06, time: '3-5 min', reliability: 98.8, minAmount: 50, maxAmount: 500000 },
  { id: 'hop', name: 'Hop Exchange', logo: '⚡', fee: 0.04, time: '5-10 min', reliability: 98.5, minAmount: 100, maxAmount: 250000 },
  { id: 'cbridge', name: 'Celer Bridge', logo: '🌐', fee: 0.03, time: '10-20 min', reliability: 97.9, minAmount: 100, maxAmount: 1000000 },
  { id: 'synapse', name: 'Synapse', logo: '🔗', fee: 0.05, time: '5-15 min', reliability: 97.5, minAmount: 50, maxAmount: 250000 },
];

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

function timeAgo(timestamp: number): string {
  const diff = Date.now() - timestamp;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

// ============================================================================
// Bridge quotes, balances, and transfer history must come from authenticated backend providers.
// This page intentionally remains fail-closed until those contracts are configured.

// ============================================================================
// Main Bridge Page Component
// ============================================================================

export default function BridgePage() {
  const { isDark } = useTheme();
  // State
  const [chains, setChains] = useState<Chain[]>(CHAINS_FALLBACK);
  const [fromChain, setFromChain] = useState(1);
  const [toChain, setToChain] = useState(137);
  const [token, setToken] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [quote, setQuote] = useState<BridgeQuote | null>(null);
  const [selectedRoute, setSelectedRoute] = useState('across');
  const [routes, setRoutes] = useState<BridgeRoute[]>([]);
  const [loading, setLoading] = useState(false);
  const [transferHistory, setTransferHistory] = useState<BridgeTransfer[]>([]);
  const [activeTab, setActiveTab] = useState(0);
  const [activeStep, setActiveStep] = useState(0);
  const [walletAddress, setWalletAddress] = useState('');

  // Snackbar
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success' as 'success' | 'error' | 'info'
  });

  // ============================================================================
  // Effects
  // ============================================================================

  useEffect(() => {
    // Fetch the live chain registry from the canonical backend so the chain
    // pickers reflect admin-added chains (no hardcoded-only list). On failure
    // we keep the offline fallback (CHAINS_FALLBACK) rather than fabricate.
    const fetchChains = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/chains`);
        if (response.ok) {
          const data = await response.json();
          if (data.chains && Array.isArray(data.chains) && data.chains.length > 0) {
            // Map the backend ChainConfig schema to the local Chain interface.
            const mapped: Chain[] = data.chains
              .filter((c: { chain_type?: string; is_testnet?: boolean }) => c.chain_type === 'evm' && !c.is_testnet)
              .map((c: { id: number; name: string; symbol: string; rpc_endpoint?: string; explorer_url?: string }) => ({
                id: c.id,
                name: c.name,
                icon: '⛓️',
                color: '#627EEA',
                rpcUrl: c.rpc_endpoint || '',
                explorerUrl: c.explorer_url || '',
                nativeCurrency: c.symbol,
                avgBridgeTime: '5-15 min',
                gasCost: '$1-15',
              }));
            if (mapped.length > 0) setChains(mapped);
          }
        }
      } catch (err) {
        // Backend unreachable: keep CHAINS_FALLBACK (no fabricated chains).
      }
    };

    fetchChains();

    // Fetch routes from API. The bridge service exposes a /routes endpoint
    // listing the registered bridges + their supported chains.
    const fetchRoutes = async () => {
      try {
        const response = await fetch(`${API_BASE}/api/v1/bridge/routes?from_chain=${fromChain}&to_chain=${toChain}`);
        if (response.ok) {
          const data = await response.json();
          if (data.routes && data.routes.length > 0) {
            setRoutes(data.routes);
          }
        }
      } catch (err) {
        // No routes endpoint: leave routes empty (no fabricated routes).
      }
    };

    // Fetch quote from API (go/bridge_service GetQuote is a POST with JSON body).
    const fetchQuote = async () => {
      if (!amount || parseFloat(amount) <= 0) {
        setQuote(null);
        return;
      }
      try {
        const response = await fetch(`${API_BASE}/api/v1/bridge/quote`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            from_chain: String(fromChain),
            to_chain: String(toChain),
            token,
            amount,
          }),
        });
        if (response.ok) {
          const data = await response.json();
          const feeNum = parseFloat(data.fee) || 0;
          const parsedAmount = parseFloat(amount);
          setQuote({
            fromChain,
            toChain,
            token,
            amount: parsedAmount.toString(),
            bridgeFee: feeNum,
            networkFee: 0,
            estimatedTime: `${data.estimated_time || 600}s`,
            receivedAmount: (parsedAmount - feeNum).toString(),
            rate: 1.0,
            availableRoutes: routes,
          });
        } else {
          setQuote(null);
        }
      } catch (err) {
        // Backend unreachable: show no quote rather than fabricate one.
        setQuote(null);
      }
    };

    // Fetch history. The bridge service exposes a per-tx status endpoint but
    // not a bulk /history list, so history stays empty until one is added; we
    // do not fabricate transfer history.
    const fetchHistory = async () => {
      try {
        const authToken = localStorage.getItem('tigerwallet_token');
        const headers: HeadersInit = { 'Content-Type': 'application/json' };
        if (authToken) headers['Authorization'] = `Bearer ${authToken}`;

        const response = await fetch(`${API_BASE}/api/v1/bridge/history`, { headers });
        if (response.ok) {
          const data = await response.json();
          if (data.history) {
            setTransferHistory(data.history);
          }
        }
      } catch (err) {
        // No history endpoint available yet.
      }
    };

    fetchRoutes();
    fetchQuote();
    fetchHistory();
  }, [fromChain, toChain, token, amount, routes]);

  // ============================================================================
  // Handlers
  // ============================================================================

  const handleSwitchChains = () => {
    const temp = fromChain;
    setFromChain(toChain);
    setToChain(temp);
  };

  const handleMaxAmount = () => {
    setSnackbar({ open: true, message: 'Please use your wallet balance', severity: 'info' });
  };

  const handleBridge = async () => {
    if (!walletAddress) {
      setSnackbar({ open: true, message: 'Please connect your wallet first', severity: 'error' });
      return;
    }
    // ---- Client-side input validation (defense in depth) ----
    const amt = parseFloat(amount);
    if (!Number.isFinite(amt) || amt <= 0) {
      setSnackbar({ open: true, message: 'Enter a valid positive amount to bridge', severity: 'error' });
      return;
    }
    if (fromChain === toChain) {
      setSnackbar({ open: true, message: 'Source and destination chains must differ', severity: 'error' });
      return;
    }
    // Validate against the selected route's min/max bounds when available.
    const activeRoute = routes.find((r) => r.id === selectedRoute);
    if (activeRoute) {
      if (amt < activeRoute.minAmount) {
        setSnackbar({
          open: true,
          message: `Amount is below ${activeRoute.name}'s minimum (${activeRoute.minAmount} ${token})`,
          severity: 'error',
        });
        return;
      }
      if (amt > activeRoute.maxAmount) {
        setSnackbar({
          open: true,
          message: `Amount exceeds ${activeRoute.name}'s maximum (${activeRoute.maxAmount} ${token})`,
          severity: 'error',
        });
        return;
      }
    }

    setLoading(true);
    setSnackbar({ open: true, message: 'Processing bridge transfer...', severity: 'info' });

    try {
      const authToken = localStorage.getItem('tigerwallet_token');
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (authToken) headers['Authorization'] = `Bearer ${authToken}`;

      const response = await fetch(`${API_BASE}/api/v1/bridge/transfer`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          // Match go/bridge_service InitiateTransfer field names.
          user_id: walletAddress,
          from_chain: String(fromChain),
          to_chain: String(toChain),
          token,
          amount,
          recipient: walletAddress,
        }),
      });

      const data = await response.json();
      if (data.success && data.tx_id) {
        setSnackbar({ open: true, message: `Bridge initiated! ID: ${data.tx_id}`, severity: 'success' });
        setAmount('');
        setQuote(null);
      } else {
        setSnackbar({ open: true, message: data.error || 'Bridge failed', severity: 'error' });
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Bridge failed';
      setSnackbar({ open: true, message: msg, severity: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const getStatusChip = (status: BridgeTransfer['status']) => {
    switch (status) {
      case 'pending':
        return <Chip label="Pending" size="small" sx={{ bgcolor: '#ff980020', color: '#ff9800' }} />;
      case 'submitted':
        return <Chip label="Submitted" size="small" sx={{ bgcolor: '#00d4ff20', color: '#00d4ff' }} />;
      case 'confirming':
        return <Chip label="Confirming" size="small" sx={{ bgcolor: '#ff980020', color: '#ff9800' }} />;
      case 'completed':
        return <Chip label="Completed" size="small" sx={{ bgcolor: '#00d4aa20', color: '#00d4aa' }} />;
      case 'failed':
        return <Chip label="Failed" size="small" sx={{ bgcolor: '#ff572220', color: '#ff5722' }} />;
    }
  };

  // ============================================================================
  // Main Render
  // ============================================================================

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: isDark ? '#0a0a14' : '#f5f7fa', color: isDark ? 'white' : '#1a1a2e', p: 3 }}>
      <Box sx={{ maxWidth: 1200, mx: 'auto' }}>
        {/* Header */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h4" sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>
            🌉 Cross-Chain Bridge
          </Typography>
          <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mt: 1 }}>
            Transfer assets across 7+ chains with the best rates and fastest routes
          </Typography>
        </Box>

        {/* Features Bar */}
        <Box sx={{ display: 'flex', gap: 3, mb: 4, flexWrap: 'wrap' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Speed sx={{ color: '#00d4aa', fontSize: 20 }} />
            <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Fast Transfers</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Security sx={{ color: '#00d4aa', fontSize: 20 }} />
            <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Secure</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <SwapHoriz sx={{ color: '#00d4aa', fontSize: 20 }} />
            <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>7+ Chains</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CheckCircle sx={{ color: '#00d4aa', fontSize: 20 }} />
            <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Best Routes</Typography>
          </Box>
        </Box>

        {/* Tabs */}
        <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
          <Button
            variant={activeTab === 0 ? 'contained' : 'outlined'}
            onClick={() => setActiveTab(0)}
            sx={{ bgcolor: activeTab === 0 ? '#00d4aa' : 'transparent', color: activeTab === 0 ? 'black' : 'white', borderColor: 'var(--bg-tertiary)' }}
          >
            Bridge
          </Button>
          <Button
            variant={activeTab === 1 ? 'contained' : 'outlined'}
            onClick={() => setActiveTab(1)}
            sx={{ bgcolor: activeTab === 1 ? '#00d4aa' : 'transparent', color: activeTab === 1 ? 'black' : 'white', borderColor: 'var(--bg-tertiary)' }}
          >
            History
          </Button>
        </Box>

        {activeTab === 0 ? (
          /* Bridge Tab */
          <Box sx={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 3 }}>
            {/* Main Bridge Card */}
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
              <CardContent sx={{ p: 4 }}>
                {/* Stepper */}
                <Stepper activeStep={activeStep} sx={{ mb: 4 }}>
                  <Step><StepLabel>Configure</StepLabel></Step>
                  <Step><StepLabel>Transfer</StepLabel></Step>
                  <Step><StepLabel>Complete</StepLabel></Step>
                </Stepper>

                {/* From Chain */}
                <Box sx={{ mb: 3 }}>
                  <Typography sx={{ color: 'var(--text-secondary)', mb: 1 }}>From</Typography>
                  <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
                    <FormControl fullWidth size="small">
                      <Select
                        value={fromChain}
                        onChange={(e) => setFromChain(e.target.value as number)}
                        sx={{ color: isDark ? 'white' : '#1a1a2e', bgcolor: 'var(--bg-secondary)', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                      >
                        {chains.map(chain => (
                          <MenuItem key={chain.id} value={chain.id}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <span>{chain.icon}</span>
                              <span>{chain.name}</span>
                            </Box>
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <TextField
                      size="small"
                      type="number"
                      placeholder="0.0"
                      value={amount}
                      onChange={(e) => setAmount(e.target.value)}
                      InputProps={{
                        endAdornment: (
                          <InputAdornment position="end">
                            <Button size="small" onClick={handleMaxAmount} sx={{ color: '#00d4aa', minWidth: 0 }}>
                              MAX
                            </Button>
                          </InputAdornment>
                        ),
                      }}
                      sx={{ flex: 1, '& input': { color: 'white' }, '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } } }}
                    />
                  </Box>
                </Box>

                {/* Switch Button */}
                <Box sx={{ display: 'flex', justifyContent: 'center', my: 2 }}>
                  <IconButton
                    onClick={handleSwitchChains}
                    sx={{
                      bgcolor: 'var(--bg-secondary)',
                      border: '4px solid #1a1a2e',
                      '&:hover': { bgcolor: 'var(--bg-tertiary)' },
                    }}
                  >
                    <SwapHoriz sx={{ color: '#00d4aa' }} />
                  </IconButton>
                </Box>

                {/* To Chain */}
                <Box sx={{ mb: 3 }}>
                  <Typography sx={{ color: 'var(--text-secondary)', mb: 1 }}>To</Typography>
                  <FormControl fullWidth size="small">
                    <Select
                      value={toChain}
                      onChange={(e) => setToChain(e.target.value as number)}
                      sx={{ color: isDark ? 'white' : '#1a1a2e', bgcolor: 'var(--bg-secondary)', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                    >
                      {chains.filter(c => c.id !== fromChain).map(chain => (
                        <MenuItem key={chain.id} value={chain.id}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <span>{chain.icon}</span>
                            <span>{chain.name}</span>
                          </Box>
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>

                {/* Token Selection */}
                <Box sx={{ mb: 3 }}>
                  <Typography sx={{ color: 'var(--text-secondary)', mb: 1 }}>Token</Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    {TOKENS.map(t => (
                      <Chip
                        key={t.symbol}
                        label={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                            <span>{t.icon}</span>
                            <span>{t.symbol}</span>
                          </Box>
                        }
                        onClick={() => setToken(t.symbol)}
                        sx={{
                          bgcolor: token === t.symbol ? '#00d4aa20' : 'var(--bg-secondary)',
                          color: token === t.symbol ? '#00d4aa' : 'white',
                          cursor: 'pointer',
                        }}
                      />
                    ))}
                  </Box>
                </Box>

                <Divider sx={{ borderColor: 'var(--bg-tertiary)', my: 3 }} />

                {/* Route Selection */}
                <Typography sx={{ color: 'var(--text-secondary)', mb: 2 }}>Select Route</Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mb: 3 }}>
                  {quote?.availableRoutes.slice(0, 3).map(route => (
                    <Box
                      key={route.id}
                      onClick={() => setSelectedRoute(route.id)}
                      sx={{
                        p: 2,
                        borderRadius: 2,
                        bgcolor: selectedRoute === route.id ? '#00d4aa10' : 'var(--bg-secondary)',
                        border: `1px solid ${selectedRoute === route.id ? '#00d4aa' : 'transparent'}`,
                        cursor: 'pointer',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        '&:hover': { bgcolor: 'var(--bg-tertiary)' },
                      }}
                    >
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                        <Typography sx={{ fontSize: 20 }}>{route.logo}</Typography>
                        <Box>
                          <Typography sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>{route.name}</Typography>
                          <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>{route.time}</Typography>
                        </Box>
                      </Box>
                      <Box sx={{ textAlign: 'right' }}>
                        <Typography sx={{ color: '#00d4aa' }}>{(route.fee * 100).toFixed(2)}% fee</Typography>
                        <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>{route.reliability}% success</Typography>
                      </Box>
                    </Box>
                  ))}
                </Box>

                {/* Bridge Button */}
                <Button
                  fullWidth
                  variant="contained"
                  size="large"
                  onClick={handleBridge}
                  disabled={!quote || loading || activeStep > 0}
                  startIcon={loading ? <CircularProgress size={20} sx={{ color: 'black' }} /> : <ArrowForward />}
                  sx={{
                    bgcolor: '#00d4aa',
                    color: 'black',
                    py: 1.5,
                    '&:hover': { bgcolor: '#00b894' },
                    '&:disabled': { bgcolor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' },
                  }}
                >
                  {loading ? 'Processing...' : amount && quote ? `Bridge ${amount} ${token}` : 'Enter Amount'}
                </Button>
              </CardContent>
            </Card>

            {/* Quote Card */}
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3, height: 'fit-content' }}>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ color: 'white', mb: 3 }}>Transfer Details</Typography>

                {quote ? (
                  <Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                      <Typography sx={{ color: 'var(--text-secondary)' }}>You Send</Typography>
                      <Typography sx={{ color: 'white' }}>{formatAmount(amount)} {token}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                      <Typography sx={{ color: 'var(--text-secondary)' }}>Bridge Fee</Typography>
                      <Typography sx={{ color: '#ff9800' }}>{formatUSD(quote.bridgeFee)}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                      <Typography sx={{ color: 'var(--text-secondary)' }}>Network Fee</Typography>
                      <Typography sx={{ color: '#ff9800' }}>{formatUSD(quote.networkFee)}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                      <Typography sx={{ color: 'var(--text-secondary)' }}>Estimated Time</Typography>
                      <Typography sx={{ color: 'white' }}>{quote.estimatedTime}</Typography>
                    </Box>
                    <Divider sx={{ borderColor: 'var(--bg-tertiary)', my: 2 }} />
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                      <Typography sx={{ color: 'var(--text-secondary)' }}>You Receive</Typography>
                      <Typography sx={{ color: '#00d4aa', fontWeight: 'bold', fontSize: 18 }}>
                        {formatAmount(quote.receivedAmount)} {token}
                      </Typography>
                    </Box>

                    <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2, mt: 3 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                        <Speed sx={{ color: '#00d4aa', fontSize: 16 }} />
                        <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Fastest Route</Typography>
                      </Box>
                      <Typography sx={{ color: 'white' }}>
                        {chains.find(c => c.id === fromChain)?.icon} {chains.find(c => c.id === fromChain)?.name}
                        {' → '}
                        {chains.find(c => c.id === toChain)?.icon} {chains.find(c => c.id === toChain)?.name}
                      </Typography>
                    </Box>
                  </Box>
                ) : (
                  <Box sx={{ textAlign: 'center', py: 4 }}>
                    <Typography sx={{ color: 'var(--text-secondary)' }}>Enter an amount to see quote</Typography>
                  </Box>
                )}
              </CardContent>
            </Card>
          </Box>
        ) : (
          /* History Tab */
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" sx={{ color: 'white', mb: 3 }}>Transfer History</Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {transferHistory.map(tx => (
                  <Box
                    key={tx.id}
                    sx={{
                      p: 3,
                      bgcolor: 'var(--bg-secondary)',
                      borderRadius: 2,
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                        {chains.find(c => c.id === tx.fromChain)?.icon}
                        <ArrowForward sx={{ color: 'var(--text-secondary)', fontSize: 16 }} />
                        {chains.find(c => c.id === tx.toChain)?.icon}
                        <Typography sx={{ color: 'white', ml: 1 }}>
                          {tx.amount} {tx.token}
                        </Typography>
                        {getStatusChip(tx.status)}
                      </Box>
                      <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                        {timeAgo(tx.timestamp)}
                        {tx.sourceTxHash && (
                          <> • Source: <a href={`${chains.find(c => c.id === tx.fromChain)?.explorerUrl}/tx/${tx.sourceTxHash}`} target="_blank" rel="noopener" style={{ color: '#00d4aa' }}>{formatAddress(tx.sourceTxHash)}</a></>
                        )}
                      </Typography>
                      {tx.status === 'confirming' && (
                        <Box sx={{ mt: 1 }}>
                          <Typography variant="caption" sx={{ color: '#ff9800' }}>
                            Confirming: {tx.confirmations}/{tx.requiredConfirmations} blocks
                          </Typography>
                        </Box>
                      )}
                    </Box>
                    {tx.destTxHash && (
                      <Button
                        size="small"
                        href={`${chains.find(c => c.id === tx.toChain)?.explorerUrl}/tx/${tx.destTxHash}`}
                        target="_blank"
                        sx={{ color: '#00d4aa' }}
                      >
                        View on Dest
                      </Button>
                    )}
                  </Box>
                ))}
              </Box>
            </CardContent>
          </Card>
        )}

        {/* Supported Chains */}
        <Box sx={{ mt: 4 }}>
          <Typography variant="h6" sx={{ color: 'white', mb: 2 }}>Supported Chains</Typography>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            {chains.map(chain => (
              <Card key={chain.id} sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 2, minWidth: 150 }}>
                <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    <Typography sx={{ fontSize: 24 }}>{chain.icon}</Typography>
                    <Typography sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>{chain.name}</Typography>
                  </Box>
                  <Typography variant="caption" sx={{ color: 'var(--text-secondary)', display: 'block' }}>
                    Avg Time: {chain.avgBridgeTime}
                  </Typography>
                  <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                    Gas: {chain.gasCost}
                  </Typography>
                </CardContent>
              </Card>
            ))}
          </Box>
        </Box>
      </Box>

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