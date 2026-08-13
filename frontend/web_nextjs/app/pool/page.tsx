'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, IconButton, Select, MenuItem, FormControl, InputLabel,
  Dialog, DialogTitle, DialogContent, DialogActions, Tabs, Tab,
  Slider, InputAdornment, Tooltip, CircularProgress, Snackbar, Alert,
  Divider, Grid, Paper
} from '@mui/material';
import {
  Add, Remove, Visibility, Close, Refresh, ShowChart,
  ContentCopy, TrendingUp, TrendingDown
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';
import {
  api,
  PoolToken as Token,
  LiquidityPool as Pool,
  LiquidityPosition,
} from '@/lib/api/client';

// ============================================================================
// Types & Interfaces
// ============================================================================

interface PoolStats {
  totalTVL: number;
  totalVolume24h: number;
  totalVolume7d: number;
  totalFees24h: number;
  totalPools: number;
  topPools: Pool[];
}

// ============================================================================
// Constants
// ============================================================================

const CHAIN_CONFIG: Record<number, { name: string; explorer: string }> = {
  1: { name: 'Ethereum', explorer: 'https://etherscan.io' },
  56: { name: 'BNB Chain', explorer: 'https://bscscan.com' },
  137: { name: 'Polygon', explorer: 'https://polygonscan.com' },
  42161: { name: 'Arbitrum', explorer: 'https://arbiscan.io' },
  10: { name: 'Optimism', explorer: 'https://optimistic.etherscan.io' },
  8453: { name: 'Base', explorer: 'https://basescan.org' },
};

const DEX_INFO: Record<string, { name: string; logo: string; color: string }> = {
  'uniswap_v2': { name: 'Uniswap V2', logo: '🦄', color: '#FF007A' },
  'uniswap_v3': { name: 'Uniswap V3', logo: '🦄', color: '#FF007A' },
  'sushiswap': { name: 'SushiSwap', logo: '🍣', color: '#FA52A0' },
  'pancakeswap': { name: 'PancakeSwap', logo: '🥞', color: '#633001' },
  'quickswap': { name: 'QuickSwap', logo: '⚡', color: '#6c8fc5' },
};

const FEE_TIERS = [
  { value: 0.01, label: '0.01%', description: 'Stable pairs' },
  { value: 0.05, label: '0.05%', description: 'Stable pairs' },
  { value: 0.3, label: '0.30%', description: 'Traditional' },
  { value: 1, label: '1.00%', description: 'Exotic pairs' },
];

// ============================================================================
// Utility Functions
// ============================================================================

function formatAddress(address: string, chars: number = 4): string {
  return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`;
}

function formatUSD(amount: number): string {
  if (amount >= 1e9) return `$${(amount / 1e9).toFixed(2)}B`;
  if (amount >= 1e6) return `$${(amount / 1e6).toFixed(2)}M`;
  if (amount >= 1e3) return `$${(amount / 1e3).toFixed(2)}K`;
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount);
}

function formatNumber(num: number, decimals: number = 2): string {
  if (num >= 1e9) return (num / 1e9).toFixed(decimals) + 'B';
  if (num >= 1e6) return (num / 1e6).toFixed(decimals) + 'M';
  if (num >= 1e3) return (num / 1e3).toFixed(decimals) + 'K';
  return num.toFixed(decimals);
}

function formatTokenAmount(amount: string, decimals: number = 18): string {
  if (!amount || amount === '0') return '0';
  try {
    const num = Number(amount) / Math.pow(10, decimals);
    if (num < 0.0001) return '<0.0001';
    return num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 6 });
  } catch {
    return '0';
  }
}

function formatPercent(value: number): string {
  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
}

// Main Pool Page Component
// ============================================================================

export default function PoolPage() {
  const { isDark } = useTheme();
  // State
  const [chainId, setChainId] = useState(1);
  const [pools, setPools] = useState<Pool[]>([]);
  const [positions, setPositions] = useState<LiquidityPosition[]>([]);
  const [stats, setStats] = useState<PoolStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPool, setSelectedPool] = useState<Pool | null>(null);
  const [walletAddress] = useState<string | null>(null);

  // UI State
  const [activeTab, setActiveTab] = useState(0);
  const [showCreatePool, setShowCreatePool] = useState(false);
  const [showPoolDetails, setShowPoolDetails] = useState(false);
  const [filterDEX, setFilterDEX] = useState<string>('all');
  const [sortBy, setSortBy] = useState<'tvl' | 'volume' | 'apr'>('tvl');
  const [searchQuery, setSearchQuery] = useState('');

  // Create pool form
  const [token0, setToken0] = useState<string>('');
  const [token1, setToken1] = useState<string>('');
  const [amount0, setAmount0] = useState('');
  const [amount1, setAmount1] = useState('');
  const [feeTier, setFeeTier] = useState(0.3);
  const [priceRangeLow, setPriceRangeLow] = useState(0);
  const [priceRangeHigh, setPriceRangeHigh] = useState(0);
  const [creating, setCreating] = useState(false);

  // Snackbar
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error' | 'info';
  }>({ open: false, message: '', severity: 'info' });

  // ============================================================================
  // Data Loading
  // ============================================================================

  const loadPools = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const poolsRes = await api.getLiquidityPools({ chainId });
      const fetchedPools = poolsRes.success && poolsRes.data ? poolsRes.data : [];
      setPools(fetchedPools);

      if (walletAddress) {
        const posRes = await api.getPoolPositions(walletAddress);
        if (posRes.success && posRes.data) {
          setPositions(posRes.data);
        } else {
          setPositions([]);
        }
      } else {
        setPositions([]);
      }

      const totalTVL = fetchedPools.reduce((sum, p) => sum + p.tvlUSD, 0);
      const totalVolume24h = fetchedPools.reduce((sum, p) => sum + p.volume24h, 0);
      const totalVolume7d = fetchedPools.reduce((sum, p) => sum + p.volume7d, 0);
      const totalFees24h = totalVolume24h * 0.003;

      setStats({
        totalTVL,
        totalVolume24h,
        totalVolume7d,
        totalFees24h,
        totalPools: fetchedPools.length,
        topPools: fetchedPools.slice(0, 5),
      });

      if (!poolsRes.success) {
        setError(poolsRes.error || 'Failed to load pools');
      }
    } catch (err: any) {
      const message = err?.message || 'Failed to load pools';
      setError(message);
      console.error('Failed to load pools:', err);
    } finally {
      setLoading(false);
    }
  }, [chainId, walletAddress]);

  useEffect(() => {
    loadPools();
  }, [loadPools]);

  // ============================================================================
  // Pool Creation
  // ============================================================================

  const handleCreatePool = async () => {
    if (!token0 || !token1 || !amount0 || !amount1) {
      setSnackbar({ open: true, message: 'Please fill in all fields', severity: 'error' });
      return;
    }

    setCreating(true);
    try {
      const res = await api.createLiquidityPool({
        token0,
        token1,
        feeTier,
        amount0,
        amount1,
        chainId,
        priceRangeLow,
        priceRangeHigh,
      });
      if (res.success) {
        setSnackbar({ open: true, message: 'Pool created successfully!', severity: 'success' });
        setShowCreatePool(false);
        setToken0('');
        setToken1('');
        setAmount0('');
        setAmount1('');
        loadPools();
      } else {
        setSnackbar({ open: true, message: res.error || 'Failed to create pool', severity: 'error' });
      }
    } catch (err: any) {
      setSnackbar({ open: true, message: err?.message || 'Failed to create pool', severity: 'error' });
    } finally {
      setCreating(false);
    }
  };

  const handleAddLiquidity = async () => {
    if (!selectedPool || !amount0 || !amount1) return;

    setCreating(true);
    try {
      const res = await api.addLiquidity(selectedPool.id, amount0, amount1);
      if (res.success) {
        setSnackbar({ open: true, message: 'Liquidity added successfully!', severity: 'success' });
        setShowPoolDetails(false);
        loadPools();
      } else {
        setSnackbar({ open: true, message: res.error || 'Failed to add liquidity', severity: 'error' });
      }
    } catch (err: any) {
      setSnackbar({ open: true, message: err?.message || 'Failed to add liquidity', severity: 'error' });
    } finally {
      setCreating(false);
    }
  };

  // ============================================================================
  // Filtering & Sorting
  // ============================================================================

  const filteredPools = pools
    .filter(pool => {
      if (filterDEX !== 'all' && pool.dex !== filterDEX) return false;
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        return (
          pool.token0.symbol.toLowerCase().includes(query) ||
          pool.token1.symbol.toLowerCase().includes(query) ||
          pool.dexName.toLowerCase().includes(query)
        );
      }
      return true;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'volume': return b.volume24h - a.volume24h;
        case 'apr': return b.apr - a.apr;
        default: return b.tvlUSD - a.tvlUSD;
      }
    });

  // ============================================================================
  // Render Helpers
  // ============================================================================

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setSnackbar({ open: true, message: 'Copied to clipboard!', severity: 'success' });
  };

  const renderPoolChip = (pool: Pool) => (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
        {pool.token0.symbol}
      </Typography>
      <Typography sx={{ color: 'var(--text-secondary)' }}>/</Typography>
      <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
        {pool.token1.symbol}
      </Typography>
    </Box>
  );

  const renderDEXChip = (dex: string) => {
    const info = DEX_INFO[dex] || { name: dex, logo: '📦', color: '#666' };
    return (
      <Chip
        label={`${info.logo} ${info.name}`}
        size="small"
        sx={{
          bgcolor: `${info.color}20`,
          color: info.color,
          fontSize: '0.7rem',
        }}
      />
    );
  };

  const renderFeeChip = (fee: number) => (
    <Chip
      label={`${fee}%`}
      size="small"
      sx={{
        bgcolor: fee <= 0.05 ? '#00d4aa20' : fee <= 0.3 ? '#ff980020' : '#ff572220',
        color: fee <= 0.05 ? '#00d4aa' : fee <= 0.3 ? '#ff9800' : '#ff5722',
        fontSize: '0.7rem',
      }}
    />
  );

  // ============================================================================
  // Main Render
  // ============================================================================

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'var(--bg-primary)', color: isDark ? 'white' : '#1a1a2e', p: 3 }}>
      <Box sx={{ maxWidth: 1400, mx: 'auto' }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Box>
            <Typography variant="h4" sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>
              🏊 Liquidity Pools
            </Typography>
            <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mt: 1 }}>
              Provide liquidity and earn fees from trades
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <Select
                value={chainId}
                onChange={(e) => setChainId(e.target.value as number)}
                sx={{ color: isDark ? 'white' : '#1a1a2e', bgcolor: 'var(--bg-primary)', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
              >
                {Object.entries(CHAIN_CONFIG).map(([id, config]) => (
                  <MenuItem key={id} value={parseInt(id)}>{config.name}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button
              variant="outlined"
              startIcon={<Refresh />}
              onClick={loadPools}
              sx={{ borderColor: 'var(--bg-tertiary)', color: isDark ? 'white' : '#1a1a2e' }}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              startIcon={<Add />}
              onClick={() => setShowCreatePool(true)}
              sx={{ bgcolor: '#00d4aa', color: 'black', '&:hover': { bgcolor: '#00b894' } }}
            >
              Create Pool
            </Button>
          </Box>
        </Box>

        {/* Stats Cards */}
        {stats && (
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 2, mb: 4 }}>
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
              <CardContent>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Total TVL</Typography>
                <Typography variant="h5" sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
                  {formatUSD(stats.totalTVL)}
                </Typography>
              </CardContent>
            </Card>
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
              <CardContent>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>24h Volume</Typography>
                <Typography variant="h5" sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>
                  {formatUSD(stats.totalVolume24h)}
                </Typography>
              </CardContent>
            </Card>
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
              <CardContent>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>7d Volume</Typography>
                <Typography variant="h5" sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>
                  {formatUSD(stats.totalVolume7d)}
                </Typography>
              </CardContent>
            </Card>
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
              <CardContent>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>24h Fees</Typography>
                <Typography variant="h5" sx={{ color: '#ff9800', fontWeight: 'bold' }}>
                  {formatUSD(stats.totalFees24h)}
                </Typography>
              </CardContent>
            </Card>
            <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
              <CardContent>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Total Pools</Typography>
                <Typography variant="h5" sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>
                  {stats.totalPools}
                </Typography>
              </CardContent>
            </Card>
          </Box>
        )}

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
            <Tab label="All Pools" />
            <Tab label="My Positions" />
            <Tab label="Analytics" />
          </Tabs>

          <CardContent sx={{ p: 3 }}>
            {/* Filters */}
            <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
              <TextField
                size="small"
                placeholder="Search tokens or DEX..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                sx={{
                  minWidth: 250,
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': {
                    '& fieldset': { borderColor: 'var(--bg-tertiary)' },
                  },
                }}
              />
              <FormControl size="small" sx={{ minWidth: 150 }}>
                <Select
                  value={filterDEX}
                  onChange={(e) => setFilterDEX(e.target.value)}
                  sx={{ color: isDark ? 'white' : '#1a1a2e', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                >
                  <MenuItem value="all">All DEXs</MenuItem>
                  {Object.entries(DEX_INFO).map(([key, info]) => (
                    <MenuItem key={key} value={key}>{info.logo} {info.name}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControl size="small" sx={{ minWidth: 120 }}>
                <Select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as any)}
                  sx={{ color: isDark ? 'white' : '#1a1a2e', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                >
                  <MenuItem value="tvl">Sort by TVL</MenuItem>
                  <MenuItem value="volume">Sort by Volume</MenuItem>
                  <MenuItem value="apr">Sort by APR</MenuItem>
                </Select>
              </FormControl>
            </Box>

            {/* Pool List */}
            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
                <CircularProgress sx={{ color: '#00d4aa' }} />
              </Box>
            ) : error ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 5, gap: 2 }}>
                <Typography sx={{ color: '#ef4444' }}>{error}</Typography>
                <Button
                  variant="outlined"
                  startIcon={<Refresh />}
                  onClick={loadPools}
                  sx={{ borderColor: 'var(--bg-tertiary)', color: isDark ? 'white' : '#1a1a2e' }}
                >
                  Retry
                </Button>
              </Box>
            ) : activeTab === 0 ? (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Pool</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>DEX</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Fee</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">TVL</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Volume (24h)</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">APR</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {filteredPools.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={7} sx={{ textAlign: 'center', color: 'var(--text-secondary)', py: 5 }}>
                          No pools found. Try adjusting your filters or create a new pool.
                        </TableCell>
                      </TableRow>
                    ) : filteredPools.map((pool) => (
                      <TableRow key={pool.id} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            {renderPoolChip(pool)}
                          </Box>
                        </TableCell>
                        <TableCell>{renderDEXChip(pool.dex)}</TableCell>
                        <TableCell>{renderFeeChip(pool.feeTier)}</TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: '#00d4aa' }}>{formatUSD(pool.tvlUSD)}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: isDark ? 'white' : '#1a1a2e' }}>{formatUSD(pool.volume24h)}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: pool.apr > 20 ? '#00d4aa' : pool.apr > 5 ? '#ff9800' : '#ff5722' }}>
                            {formatPercent(pool.apr)}
                          </Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                            <Tooltip title="View Details">
                              <IconButton size="small" onClick={() => { setSelectedPool(pool); setShowPoolDetails(true); }}>
                                <Visibility sx={{ color: 'var(--text-secondary)', fontSize: 18 }} />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Add Liquidity">
                              <IconButton size="small" onClick={() => { setSelectedPool(pool); setShowPoolDetails(true); }}>
                                <Add sx={{ color: '#00d4aa', fontSize: 18 }} />
                              </IconButton>
                            </Tooltip>
                          </Box>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            ) : activeTab === 1 ? (
              <Box>
                {positions.length === 0 ? (
                  <Box sx={{ textAlign: 'center', py: 5 }}>
                    <Typography sx={{ color: 'var(--text-secondary)', mb: 2 }}>No liquidity positions</Typography>
                    <Button
                      variant="contained"
                      startIcon={<Add />}
                      onClick={() => setShowCreatePool(true)}
                      sx={{ bgcolor: '#00d4aa', color: 'black' }}
                    >
                      Create Your First Position
                    </Button>
                  </Box>
                ) : (
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Pool</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Token 0 Amount</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Token 1 Amount</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Total Value</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Share</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Fees Earned</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Status</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {positions.map((pos) => (
                          <TableRow key={pos.id} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                            <TableCell>{renderPoolChip(pos.pool)}</TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: isDark ? 'white' : '#1a1a2e' }}>
                                {parseFloat(pos.token0Amount).toFixed(4)} {pos.pool.token0.symbol}
                              </Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: isDark ? 'white' : '#1a1a2e' }}>
                                {parseFloat(pos.token1Amount).toFixed(4)} {pos.pool.token1.symbol}
                              </Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: '#00d4aa' }}>{formatUSD(pos.totalLiquidity)}</Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: isDark ? 'white' : '#1a1a2e' }}>{pos.poolShare.toFixed(3)}%</Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Typography sx={{ color: '#ff9800' }}>
                                {pos.feesEarned0} {pos.pool.token0.symbol} + {pos.feesEarned1} {pos.pool.token1.symbol}
                              </Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Chip
                                label={pos.isActive ? 'Active' : 'Inactive'}
                                size="small"
                                sx={{
                                  bgcolor: pos.isActive ? '#00d4aa20' : '#ff572220',
                                  color: pos.isActive ? '#00d4aa' : '#ff5722',
                                }}
                              />
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Box>
            ) : (
              <Box sx={{ py: 3 }}>
                {/* Analytics Dashboard — computed from live pool + position data */}
                <Grid container spacing={3} sx={{ mb: 3 }}>
                  <Grid item xs={12} sm={6} md={3}>
                    <Card sx={{ bgcolor: 'var(--bg-secondary)', borderRadius: 2 }}>
                      <CardContent>
                        <Typography sx={{ color: 'var(--text-secondary)', fontSize: '0.75rem', textTransform: 'uppercase' }}>Total Pools</Typography>
                        <Typography sx={{ color: 'var(--text-primary)', fontSize: '1.75rem', fontWeight: 700, mt: 1 }}>
                          {pools.length}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                  <Grid item xs={12} sm={6} md={3}>
                    <Card sx={{ bgcolor: 'var(--bg-secondary)', borderRadius: 2 }}>
                      <CardContent>
                        <Typography sx={{ color: 'var(--text-secondary)', fontSize: '0.75rem', textTransform: 'uppercase' }}>Aggregate TVL</Typography>
                        <Typography sx={{ color: 'var(--text-primary)', fontSize: '1.75rem', fontWeight: 700, mt: 1 }}>
                          ${pools.reduce((sum, p) => sum + (p.tvlUSD || 0), 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                  <Grid item xs={12} sm={6} md={3}>
                    <Card sx={{ bgcolor: 'var(--bg-secondary)', borderRadius: 2 }}>
                      <CardContent>
                        <Typography sx={{ color: 'var(--text-secondary)', fontSize: '0.75rem', textTransform: 'uppercase' }}>24h Volume</Typography>
                        <Typography sx={{ color: 'var(--text-primary)', fontSize: '1.75rem', fontWeight: 700, mt: 1 }}>
                          ${pools.reduce((sum, p) => sum + (p.volume24h || 0), 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                  <Grid item xs={12} sm={6} md={3}>
                    <Card sx={{ bgcolor: 'var(--bg-secondary)', borderRadius: 2 }}>
                      <CardContent>
                        <Typography sx={{ color: 'var(--text-secondary)', fontSize: '0.75rem', textTransform: 'uppercase' }}>Your Positions</Typography>
                        <Typography sx={{ color: 'var(--text-primary)', fontSize: '1.75rem', fontWeight: 700, mt: 1 }}>
                          {positions.length}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                </Grid>

                {/* Top pools by TVL */}
                <Typography sx={{ color: 'var(--text-primary)', fontSize: '1.1rem', fontWeight: 600, mb: 2 }}>
                  Top Pools by TVL
                </Typography>
                {pools.length === 0 ? (
                  <Typography sx={{ color: 'var(--text-secondary)', textAlign: 'center', py: 4 }}>
                    No pool data available. Connect a wallet or select a chain.
                  </Typography>
                ) : (
                  <TableContainer component={Paper} sx={{ bgcolor: 'var(--bg-secondary)' }}>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Pool</TableCell>
                          <TableCell align="right" sx={{ color: 'var(--text-secondary)' }}>TVL</TableCell>
                          <TableCell align="right" sx={{ color: 'var(--text-secondary)' }}>24h Volume</TableCell>
                          <TableCell align="right" sx={{ color: 'var(--text-secondary)' }}>APR</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {[...pools].sort((a, b) => (b.tvlUSD || 0) - (a.tvlUSD || 0)).slice(0, 10).map((p) => (
                          <TableRow key={p.id}>
                            <TableCell sx={{ color: 'var(--text-primary)' }}>
                              {p.token0.symbol} / {p.token1.symbol}
                            </TableCell>
                            <TableCell align="right" sx={{ color: 'var(--text-primary)' }}>
                              ${(p.tvlUSD || 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                            </TableCell>
                            <TableCell align="right" sx={{ color: 'var(--text-primary)' }}>
                              ${(p.volume24h || 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}
                            </TableCell>
                            <TableCell align="right" sx={{ color: '#00d4aa' }}>
                              {(p.apr || 0).toFixed(2)}%
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Box>
            )}
          </CardContent>
        </Card>
      </Box>

      {/* Create Pool Dialog */}
      <Dialog
        open={showCreatePool}
        onClose={() => setShowCreatePool(false)}
        maxWidth="md"
        fullWidth
        PaperProps={{ sx: { bgcolor: 'var(--bg-primary)', backgroundImage: 'none' } }}
      >
        <DialogTitle sx={{ color: isDark ? 'white' : '#1a1a2e', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          Create New Pool
          <IconButton onClick={() => setShowCreatePool(false)} sx={{ color: isDark ? 'white' : '#1a1a2e' }}>
            <Close />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 3, mt: 2 }}>
            <Box>
              <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 1 }}>Token 0</Typography>
              <TextField
                fullWidth
                size="small"
                placeholder="Enter token symbol (e.g., ETH)"
                value={token0}
                onChange={(e) => setToken0(e.target.value.toUpperCase())}
                sx={{
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />
            </Box>
            <Box>
              <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 1 }}>Token 1</Typography>
              <TextField
                fullWidth
                size="small"
                placeholder="Enter token symbol (e.g., USDC)"
                value={token1}
                onChange={(e) => setToken1(e.target.value.toUpperCase())}
                sx={{
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />
            </Box>
            <Box>
              <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 1 }}>Amount Token 0</Typography>
              <TextField
                fullWidth
                size="small"
                type="number"
                placeholder="0.0"
                value={amount0}
                onChange={(e) => setAmount0(e.target.value)}
                InputProps={{
                  endAdornment: token0 ? <InputAdornment position="end" sx={{ color: "var(--text-secondary)" }}>{token0}</InputAdornment> : null,
                }}
                sx={{
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />
            </Box>
            <Box>
              <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 1 }}>Amount Token 1</Typography>
              <TextField
                fullWidth
                size="small"
                type="number"
                placeholder="0.0"
                value={amount1}
                onChange={(e) => setAmount1(e.target.value)}
                InputProps={{
                  endAdornment: token1 ? <InputAdornment position="end" sx={{ color: "var(--text-secondary)" }}>{token1}</InputAdornment> : null,
                }}
                sx={{
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />
            </Box>
          </Box>

          <Box sx={{ mt: 3 }}>
            <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 2 }}>Fee Tier</Typography>
            <Box sx={{ display: 'flex', gap: 2 }}>
              {FEE_TIERS.map((tier) => (
                <Card
                  key={tier.value}
                  sx={{
                    flex: 1,
                    cursor: 'pointer',
                    bgcolor: feeTier === tier.value ? '#00d4aa20' : 'var(--bg-secondary)',
                    borderColor: feeTier === tier.value ? '#00d4aa' : 'transparent',
                    border: '1px solid',
                    '&:hover': { bgcolor: 'var(--bg-tertiary)' },
                  }}
                  onClick={() => setFeeTier(tier.value)}
                >
                  <CardContent sx={{ textAlign: 'center', py: 2 }}>
                    <Typography sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>{tier.label}</Typography>
                    <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>{tier.description}</Typography>
                  </CardContent>
                </Card>
              ))}
            </Box>
          </Box>

          {feeTier > 0.1 && (
            <Box sx={{ mt: 3 }}>
              <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 2 }}>Price Range (Concentrated Liquidity)</Typography>
              <Box sx={{ display: 'flex', gap: 2 }}>
                <TextField
                  fullWidth
                  size="small"
                  type="number"
                  label="Min Price"
                  value={priceRangeLow}
                  onChange={(e) => setPriceRangeLow(parseFloat(e.target.value) || 0)}
                  sx={{
                    '& input': { color: isDark ? 'white' : '#1a1a2e' },
                    '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                    '& .MuiInputLabel-root': { color: 'var(--text-secondary)' },
                  }}
                />
                <TextField
                  fullWidth
                  size="small"
                  type="number"
                  label="Max Price"
                  value={priceRangeHigh}
                  onChange={(e) => setPriceRangeHigh(parseFloat(e.target.value) || 0)}
                  sx={{
                    '& input': { color: isDark ? 'white' : '#1a1a2e' },
                    '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                    '& .MuiInputLabel-root': { color: 'var(--text-secondary)' },
                  }}
                />
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 3 }}>
          <Button onClick={() => setShowCreatePool(false)} sx={{ color: 'var(--text-secondary)' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleCreatePool}
            disabled={creating || !token0 || !token1 || !amount0 || !amount1}
            sx={{ bgcolor: '#00d4aa', color: 'black' }}
          >
            {creating ? <CircularProgress size={20} sx={{ color: 'black' }} /> : 'Create Pool'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Pool Details Dialog */}
      <Dialog
        open={showPoolDetails}
        onClose={() => setShowPoolDetails(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { bgcolor: 'var(--bg-primary)', backgroundImage: 'none' } }}
      >
        <DialogTitle sx={{ color: isDark ? 'white' : '#1a1a2e', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          Pool Details
          <IconButton onClick={() => setShowPoolDetails(false)} sx={{ color: isDark ? 'white' : '#1a1a2e' }}>
            <Close />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          {selectedPool && (
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                {renderPoolChip(selectedPool)}
                {renderDEXChip(selectedPool.dex)}
              </Box>

              <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 3 }}>
                <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2 }}>
                  <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>TVL</Typography>
                  <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{formatUSD(selectedPool.tvlUSD)}</Typography>
                </Box>
                <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2 }}>
                  <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>24h Volume</Typography>
                  <Typography sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>{formatUSD(selectedPool.volume24h)}</Typography>
                </Box>
                <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2 }}>
                  <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>APR</Typography>
                  <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{formatPercent(selectedPool.apr)}</Typography>
                </Box>
                <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2 }}>
                  <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Fee Tier</Typography>
                  <Typography sx={{ color: isDark ? 'white' : '#1a1a2e', fontWeight: 'bold' }}>{selectedPool.feeTier}%</Typography>
                </Box>
              </Box>

              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Pool Address</Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography sx={{ color: isDark ? 'white' : '#1a1a2e', wordBreak: 'break-all', fontSize: '0.85rem' }}>
                    {selectedPool.address}
                  </Typography>
                  <IconButton size="small" onClick={() => copyToClipboard(selectedPool.address)}>
                    <ContentCopy sx={{ color: 'var(--text-secondary)', fontSize: 16 }} />
                  </IconButton>
                </Box>
              </Box>

              <Divider sx={{ borderColor: 'var(--bg-tertiary)', my: 2 }} />

              <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mb: 2 }}>Add Liquidity</Typography>
              <TextField
                fullWidth
                size="small"
                type="number"
                placeholder={`Amount ${selectedPool.token0.symbol}`}
                value={amount0}
                onChange={(e) => setAmount0(e.target.value)}
                sx={{
                  mb: 2,
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />
              <TextField
                fullWidth
                size="small"
                type="number"
                placeholder={`Amount ${selectedPool.token1.symbol}`}
                value={amount1}
                onChange={(e) => setAmount1(e.target.value)}
                sx={{
                  mb: 2,
                  '& input': { color: isDark ? 'white' : '#1a1a2e' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 3 }}>
          <Button onClick={() => setShowPoolDetails(false)} sx={{ color: 'var(--text-secondary)' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleAddLiquidity}
            disabled={creating || !amount0 || !amount1}
            sx={{ bgcolor: '#00d4aa', color: 'black' }}
          >
            {creating ? <CircularProgress size={20} sx={{ color: 'black' }} /> : 'Add Liquidity'}
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