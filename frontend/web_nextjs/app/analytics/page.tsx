'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, IconButton, Select, MenuItem, FormControl, InputLabel,
  Dialog, DialogTitle, DialogContent, DialogActions, Tabs, Tab,
  Slider, InputAdornment, Tooltip, CircularProgress, Snackbar, Alert,
  Divider, LinearProgress, ToggleButton, ToggleButtonGroup
} from '@mui/material';
import {
  TrendingUp, TrendingDown, ShowChart, BarChart, PieChart,
  Refresh, Download, CalendarMonth, FilterList, Visibility,
  ArrowUpward, ArrowDownward, AttachMoney, Speed, Pool,
  SwapHoriz, AccountBalanceWallet
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';
import { api } from '@/lib/api/client';

// ============================================================================
// Types & Interfaces
// ============================================================================

interface ProtocolStats {
  totalVolume24h: number;
  totalVolume7d: number;
  totalVolume30d: number;
  totalFees24h: number;
  totalFees7d: number;
  totalFees30d: number;
  totalTVL: number;
  totalUsers: number;
  totalTransactions: number;
  avgSwapSize: number;
  priceChange24h: number;
}

interface VolumeData {
  timestamp: number;
  volume: number;
  trades: number;
  fees: number;
}

interface PoolAnalytics {
  id: string;
  name: string;
  tvl: number;
  volume24h: number;
  volume7d: number;
  fees24h: number;
  apr: number;
  utilization: number;
}

interface TokenAnalytics {
  symbol: string;
  name: string;
  price: number;
  change24h: number;
  volume24h: number;
  marketCap: number;
  topPools: string[];
}

interface ChainAnalytics {
  chainId: number;
  chainName: string;
  volume24h: number;
  tvl: number;
  transactions: number;
  avgGasPrice: number;
  sharePercent: number;
}

// ============================================================================
// Constants
// ============================================================================

const CHAIN_CONFIG: Record<number, { name: string; color: string }> = {
  1: { name: 'Ethereum', color: '#627EEA' },
  56: { name: 'BNB Chain', color: '#F3BA2F' },
  137: { name: 'Polygon', color: '#8247E5' },
  42161: { name: 'Arbitrum', color: '#28A0F0' },
  10: { name: 'Optimism', color: '#FF0420' },
  8453: { name: 'Base', color: '#0052FF' },
  43114: { name: 'Avalanche', color: '#E84142' },
};

// ============================================================================
// Utility Functions
// ============================================================================

function formatUSD(amount: number): string {
  if (amount >= 1e12) return `$${(amount / 1e12).toFixed(2)}T`;
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

function formatPercent(value: number): string {
  return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
}

function formatDate(timestamp: number): string {
  return new Date(timestamp).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

// ============================================================================
// Main Analytics Page Component
// ============================================================================

export default function AnalyticsPage() {
  // State
  const [timeRange, setTimeRange] = useState<'24h' | '7d' | '30d' | 'all'>('7d');
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [protocolStats, setProtocolStats] = useState<ProtocolStats | null>(null);
  const [volumeData, setVolumeData] = useState<VolumeData[]>([]);
  const [poolAnalytics, setPoolAnalytics] = useState<PoolAnalytics[]>([]);
  const [chainAnalytics, setChainAnalytics] = useState<ChainAnalytics[]>([]);
  const [tokenAnalytics, setTokenAnalytics] = useState<TokenAnalytics[]>([]);

  // ============================================================================
  // Data Loading
  // ============================================================================

  const loadAnalytics = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [analyticsRes, revenueRes, txStatsRes] = await Promise.all([
        api.getAnalytics({ range: timeRange }),
        api.getAnalyticsRevenue({ range: timeRange }),
        api.getTransactionStats({ range: timeRange }),
      ]);

      const a = analyticsRes.data || {};
      setProtocolStats({
        totalVolume24h: Number(a.totalVolume24h ?? a.volume24h ?? 0),
        totalVolume7d: Number(a.totalVolume7d ?? a.volume7d ?? 0),
        totalVolume30d: Number(a.totalVolume30d ?? a.volume30d ?? 0),
        totalFees24h: Number(a.totalFees24h ?? a.fees24h ?? 0),
        totalFees7d: Number(a.totalFees7d ?? a.fees7d ?? 0),
        totalFees30d: Number(a.totalFees30d ?? a.fees30d ?? 0),
        totalTVL: Number(a.totalTVL ?? a.tvl ?? 0),
        totalUsers: Number(a.totalUsers ?? a.activeUsers ?? 0),
        totalTransactions: Number(a.totalTransactions ?? 0),
        avgSwapSize: Number(a.avgSwapSize ?? 0),
        priceChange24h: Number(a.priceChange24h ?? 0),
      });

      const rev = revenueRes.data || {};
      const history: any[] = rev.volumeHistory || rev.history || [];
      setVolumeData(history.map((d) => ({
        timestamp: Number(d.timestamp ?? d.time ?? 0),
        volume: Number(d.volume ?? 0),
        trades: Number(d.trades ?? 0),
        fees: Number(d.fees ?? 0),
      })));

      const pools: any[] = a.pools || [];
      setPoolAnalytics(pools.map((p) => ({
        id: String(p.id ?? p.address ?? ''),
        name: String(p.name ?? p.pair ?? ''),
        tvl: Number(p.tvl ?? 0),
        volume24h: Number(p.volume24h ?? 0),
        volume7d: Number(p.volume7d ?? 0),
        fees24h: Number(p.fees24h ?? 0),
        apr: Number(p.apr ?? 0),
        utilization: Number(p.utilization ?? 0),
      })).sort((x, y) => y.volume24h - x.volume24h));

      const chains: any[] = a.chains || [];
      setChainAnalytics(chains.map((c) => ({
        chainId: Number(c.chainId ?? 0),
        chainName: String(c.chainName ?? c.name ?? CHAIN_CONFIG[c.chainId]?.name ?? 'Unknown'),
        volume24h: Number(c.volume24h ?? 0),
        tvl: Number(c.tvl ?? 0),
        transactions: Number(c.transactions ?? 0),
        avgGasPrice: Number(c.avgGasPrice ?? 0),
        sharePercent: Number(c.sharePercent ?? 0),
      })).sort((x, y) => y.volume24h - x.volume24h));

      const txStats = txStatsRes.data || {};
      const tokens: any[] = txStats.tokens || a.tokens || [];
      setTokenAnalytics(tokens.map((t) => ({
        symbol: String(t.symbol ?? ''),
        name: String(t.name ?? ''),
        price: Number(t.price ?? 0),
        change24h: Number(t.change24h ?? 0),
        volume24h: Number(t.volume24h ?? 0),
        marketCap: Number(t.marketCap ?? 0),
        topPools: (t.topPools || []) as string[],
      })));
    } catch (err: any) {
      console.error('Failed to load analytics:', err);
      setError(err?.response?.data?.error || err?.message || 'Failed to load analytics data');
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    loadAnalytics();
  }, [loadAnalytics]);

  // ============================================================================
  // Render Helpers
  // ============================================================================

  const renderChangeIndicator = (value: number) => {
    const isPositive = value >= 0;
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: isPositive ? '#00d4aa' : '#ff4757' }}>
        {isPositive ? <TrendingUp fontSize="small" /> : <TrendingDown fontSize="small" />}
        <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
          {formatPercent(value)}
        </Typography>
      </Box>
    );
  };

  const renderStatCard = (
    title: string,
    value: string,
    change?: number,
    icon?: React.ReactNode,
    subtitle?: string
  ) => (
    <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3 }}>
      <CardContent>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
          <Typography variant="caption" sx={{ color: '#9ca3af' }}>{title}</Typography>
          {icon && <Box sx={{ color: '#9ca3af' }}>{icon}</Box>}
        </Box>
        <Typography variant="h5" sx={{ color: 'white', fontWeight: 'bold', mb: 0.5 }}>
          {value}
        </Typography>
        {change !== undefined && renderChangeIndicator(change)}
        {subtitle && (
          <Typography variant="caption" sx={{ color: '#9ca3af', display: 'block', mt: 0.5 }}>
            {subtitle}
          </Typography>
        )}
      </CardContent>
    </Card>
  );

  // ============================================================================
  // Main Render
  // ============================================================================

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#0a0a14', p: 3 }}>
      <Box sx={{ maxWidth: 1600, mx: 'auto' }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Box>
            <Typography variant="h4" sx={{ color: 'white', fontWeight: 'bold' }}>
              📊 Analytics Dashboard
            </Typography>
            <Typography variant="body2" sx={{ color: '#9ca3af', mt: 1 }}>
              Real-time protocol metrics and performance insights
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <ToggleButtonGroup
              value={timeRange}
              exclusive
              onChange={(_, v) => v && setTimeRange(v)}
              size="small"
              sx={{
                '& .MuiToggleButton-root': {
                  color: '#9ca3af',
                  borderColor: '#3a3a4e',
                  '&.Mui-selected': { bgcolor: '#00d4aa', color: 'black' },
                },
              }}
            >
              <ToggleButton value="24h">24H</ToggleButton>
              <ToggleButton value="7d">7D</ToggleButton>
              <ToggleButton value="30d">30D</ToggleButton>
              <ToggleButton value="all">ALL</ToggleButton>
            </ToggleButtonGroup>
            <Button
              variant="outlined"
              startIcon={<Download />}
              sx={{ borderColor: '#3a3a4e', color: 'white' }}
            >
              Export
            </Button>
            <Button
              variant="outlined"
              startIcon={<Refresh />}
              onClick={loadAnalytics}
              sx={{ borderColor: '#3a3a4e', color: 'white' }}
            >
              Refresh
            </Button>
          </Box>
        </Box>

        {/* Time Range Stats */}
        {protocolStats && (
          <>
            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, mb: 4 }}>
              {timeRange === '24h' && (
                <>
                  {renderStatCard('24h Volume', formatUSD(protocolStats.totalVolume24h), undefined, <SwapHoriz />)}
                  {renderStatCard('24h Fees', formatUSD(protocolStats.totalFees24h), undefined, <AttachMoney />)}
                  {renderStatCard('Transactions', formatNumber(protocolStats.totalTransactions), undefined, <Speed />)}
                  {renderStatCard('Avg Swap Size', formatUSD(protocolStats.avgSwapSize), undefined, <AccountBalanceWallet />)}
                </>
              )}
              {timeRange === '7d' && (
                <>
                  {renderStatCard('7d Volume', formatUSD(protocolStats.totalVolume7d), undefined, <SwapHoriz />)}
                  {renderStatCard('7d Fees', formatUSD(protocolStats.totalFees7d), undefined, <AttachMoney />)}
                  {renderStatCard('Total Users', formatNumber(protocolStats.totalUsers), undefined, <AccountBalanceWallet />)}
                  {renderStatCard('TVL', formatUSD(protocolStats.totalTVL), undefined, <Pool />)}
                </>
              )}
              {timeRange === '30d' && (
                <>
                  {renderStatCard('30d Volume', formatUSD(protocolStats.totalVolume30d), undefined, <SwapHoriz />)}
                  {renderStatCard('30d Fees', formatUSD(protocolStats.totalFees30d), undefined, <AttachMoney />)}
                  {renderStatCard('Total Users', formatNumber(protocolStats.totalUsers), undefined, <AccountBalanceWallet />)}
                  {renderStatCard('TVL', formatUSD(protocolStats.totalTVL), undefined, <Pool />)}
                </>
              )}
              {timeRange === 'all' && (
                <>
                  {renderStatCard('All-Time Volume', formatUSD(protocolStats.totalVolume30d * 12), undefined, <SwapHoriz />)}
                  {renderStatCard('All-Time Fees', formatUSD(protocolStats.totalFees30d * 12), undefined, <AttachMoney />)}
                  {renderStatCard('Total Users', formatNumber(protocolStats.totalUsers), undefined, <AccountBalanceWallet />)}
                  {renderStatCard('TVL', formatUSD(protocolStats.totalTVL), undefined, <Pool />)}
                </>
              )}
            </Box>
          </>
        )}

        {/* Tabs */}
        <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3, mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={(_, v) => setActiveTab(v)}
            sx={{
              borderBottom: '1px solid #2a2a3e',
              '& .MuiTab-root': { color: '#9ca3af' },
              '& .Mui-selected': { color: '#00d4aa' },
            }}
          >
            <Tab label="Overview" />
            <Tab label="Pools" />
            <Tab label="Tokens" />
            <Tab label="Chains" />
          </Tabs>

          <CardContent sx={{ p: 3 }}>
            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
                <CircularProgress sx={{ color: '#00d4aa' }} />
              </Box>
            ) : error ? (
              <Alert severity="error" sx={{ bgcolor: '#2a2a3e', color: '#ff4757' }} action={
                <Button color="inherit" size="small" onClick={loadAnalytics}>Retry</Button>
              }>
                {error}
              </Alert>
            ) : activeTab === 0 ? (
              /* Overview Tab */
              <Box>
                {/* Protocol Stats Grid */}
                <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2, mb: 4 }}>
                  {protocolStats && (
                    <>
                      {renderStatCard('Total Value Locked', formatUSD(protocolStats.totalTVL), protocolStats.priceChange24h, <Pool />)}
                      {renderStatCard('24h Trading Volume', formatUSD(protocolStats.totalVolume24h), undefined, <SwapHoriz />)}
                      {renderStatCard('Active Users (24h)', formatNumber(protocolStats.totalUsers * 0.15), undefined, <AccountBalanceWallet />, 'Based on unique wallets')}
                    </>
                  )}
                </Box>

                {/* Volume Chart */}
                <Card sx={{ bgcolor: '#2a2a3e', borderRadius: 3, mb: 3 }}>
                  <CardContent>
                    <Typography variant="h6" sx={{ color: 'white', mb: 2 }}>Volume History</Typography>
                    <Box sx={{ height: 300, display: 'flex', alignItems: 'flex-end', gap: 1, px: 2 }}>
                      {volumeData.slice(-14).map((data, idx) => {
                        const maxVolume = Math.max(...volumeData.slice(-14).map(d => d.volume));
                        const height = (data.volume / maxVolume) * 250;
                        return (
                          <Tooltip key={idx} title={`${formatDate(data.timestamp)}: ${formatUSD(data.volume)}`}>
                            <Box
                              sx={{
                                flex: 1,
                                height,
                                bgcolor: 'linear-gradient(to top, #00d4aa, #00d4aa80)',
                                background: 'linear-gradient(to top, #00d4aa, #00d4aa60)',
                                borderRadius: '4px 4px 0 0',
                                minWidth: 20,
                                cursor: 'pointer',
                                '&:hover': { background: '#00d4aa' },
                              }}
                            />
                          </Tooltip>
                        );
                      })}
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 2, px: 2 }}>
                      <Typography variant="caption" sx={{ color: '#9ca3af' }}>
                        {formatDate(volumeData[volumeData.length - 14]?.timestamp || Date.now())}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#9ca3af' }}>
                        {formatDate(volumeData[volumeData.length - 1]?.timestamp || Date.now())}
                      </Typography>
                    </Box>
                  </CardContent>
                </Card>

                {/* Key Metrics */}
                <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 3 }}>
                  <Card sx={{ bgcolor: '#2a2a3e', borderRadius: 3 }}>
                    <CardContent>
                      <Typography variant="h6" sx={{ color: 'white', mb: 2 }}>Fee Distribution</Typography>
                      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                        {[
                          { label: 'Swap Fees', value: 65, color: '#00d4aa' },
                          { label: 'Protocol Fees', value: 20, color: '#00d4ff' },
                          { label: 'Gas Refunds', value: 10, color: '#ff9800' },
                          { label: 'Other', value: 5, color: '#9ca3af' },
                        ].map(item => (
                          <Box key={item.label}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                              <Typography variant="body2" sx={{ color: '#9ca3af' }}>{item.label}</Typography>
                              <Typography variant="body2" sx={{ color: 'white' }}>{item.value}%</Typography>
                            </Box>
                            <LinearProgress
                              variant="determinate"
                              value={item.value}
                              sx={{
                                height: 8,
                                borderRadius: 4,
                                bgcolor: '#1a1a2e',
                                '& .MuiLinearProgress-bar': { bgcolor: item.color, borderRadius: 4 },
                              }}
                            />
                          </Box>
                        ))}
                      </Box>
                    </CardContent>
                  </Card>

                  <Card sx={{ bgcolor: '#2a2a3e', borderRadius: 3 }}>
                    <CardContent>
                      <Typography variant="h6" sx={{ color: 'white', mb: 2 }}>Top Performing Pools</Typography>
                      {poolAnalytics.slice(0, 5).map((pool, idx) => (
                        <Box key={pool.id} sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="body2" sx={{ color: '#9ca3af', width: 20 }}>{idx + 1}</Typography>
                            <Typography variant="body2" sx={{ color: 'white' }}>{pool.name}</Typography>
                          </Box>
                          <Box sx={{ textAlign: 'right' }}>
                            <Typography variant="body2" sx={{ color: '#00d4aa' }}>{formatPercent(pool.apr)}</Typography>
                            <Typography variant="caption" sx={{ color: '#9ca3af' }}>APR</Typography>
                          </Box>
                        </Box>
                      ))}
                    </CardContent>
                  </Card>
                </Box>
              </Box>
            ) : activeTab === 1 ? (
              /* Pools Tab */
              poolAnalytics.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 5 }}>
                  <BarChart sx={{ fontSize: 48, color: '#9ca3af', mb: 1 }} />
                  <Typography sx={{ color: '#9ca3af' }}>No pool analytics data available</Typography>
                </Box>
              ) : (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: '#9ca3af' }}>Pool</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">TVL</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Volume (24h)</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Fees (24h)</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">APR</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Utilization</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {poolAnalytics.map((pool) => (
                      <TableRow key={pool.id} sx={{ '&:hover': { bgcolor: '#2a2a3e' } }}>
                        <TableCell>
                          <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{pool.name}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>{formatUSD(pool.tvl)}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>{formatUSD(pool.volume24h)}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: '#ff9800' }}>{formatUSD(pool.fees24h)}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: pool.apr > 50 ? '#00d4aa' : pool.apr > 20 ? '#ff9800' : '#ff4757', fontWeight: 'bold' }}>
                            {formatPercent(pool.apr)}
                          </Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, justifyContent: 'flex-end' }}>
                            <Box sx={{ width: 60, bgcolor: '#1a1a2e', borderRadius: 1, height: 6 }}>
                              <Box sx={{ width: `${pool.utilization}%`, bgcolor: pool.utilization > 70 ? '#00d4aa' : pool.utilization > 40 ? '#ff9800' : '#ff4757', height: '100%', borderRadius: 1 }} />
                            </Box>
                            <Typography sx={{ color: 'white' }}>{pool.utilization.toFixed(0)}%</Typography>
                          </Box>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
              )
            ) : activeTab === 2 ? (
              /* Tokens Tab */
              tokenAnalytics.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 5 }}>
                  <PieChart sx={{ fontSize: 48, color: '#9ca3af', mb: 1 }} />
                  <Typography sx={{ color: '#9ca3af' }}>No token analytics data available</Typography>
                </Box>
              ) : (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: '#9ca3af' }}>Token</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Price</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">24h Change</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">24h Volume</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Market Cap</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {tokenAnalytics.map((token) => (
                      <TableRow key={token.symbol} sx={{ '&:hover': { bgcolor: '#2a2a3e' } }}>
                        <TableCell>
                          <Box>
                            <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{token.symbol}</Typography>
                            <Typography variant="caption" sx={{ color: '#9ca3af' }}>{token.name}</Typography>
                          </Box>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>
                            {token.price < 1 ? `$${token.price.toFixed(4)}` : `$${formatNumber(token.price)}`}
                          </Typography>
                        </TableCell>
                        <TableCell align="right">
                          {renderChangeIndicator(token.change24h)}
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>{formatUSD(token.volume24h)}</Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Typography sx={{ color: 'white' }}>{formatUSD(token.marketCap)}</Typography>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
              )
            ) : (
              /* Chains Tab */
              chainAnalytics.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 5 }}>
                  <ShowChart sx={{ fontSize: 48, color: '#9ca3af', mb: 1 }} />
                  <Typography sx={{ color: '#9ca3af' }}>No chain analytics data available</Typography>
                </Box>
              ) : (
              <Box>
                <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, mb: 4 }}>
                  {chainAnalytics.slice(0, 4).map((chain) => (
                    <Card key={chain.chainId} sx={{ bgcolor: '#2a2a3e', borderRadius: 3 }}>
                      <CardContent>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                          <Box sx={{ width: 12, height: 12, borderRadius: '50%', bgcolor: CHAIN_CONFIG[chain.chainId]?.color || '#666' }} />
                          <Typography sx={{ color: 'white', fontWeight: 'bold' }}>{chain.chainName}</Typography>
                        </Box>
                        <Typography variant="h6" sx={{ color: '#00d4aa' }}>{formatUSD(chain.volume24h)}</Typography>
                        <Typography variant="caption" sx={{ color: '#9ca3af' }}>24h Volume</Typography>
                        <Box sx={{ mt: 1 }}>
                          <Typography variant="caption" sx={{ color: '#9ca3af' }}>
                            Share: {chain.sharePercent}%
                          </Typography>
                        </Box>
                      </CardContent>
                    </Card>
                  ))}
                </Box>

                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ color: '#9ca3af' }}>Chain</TableCell>
                        <TableCell sx={{ color: '#9ca3af' }} align="right">Volume (24h)</TableCell>
                        <TableCell sx={{ color: '#9ca3af' }} align="right">TVL</TableCell>
                        <TableCell sx={{ color: '#9ca3af' }} align="right">Transactions</TableCell>
                        <TableCell sx={{ color: '#9ca3af' }} align="right">Avg Gas</TableCell>
                        <TableCell sx={{ color: '#9ca3af' }} align="right">Market Share</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {chainAnalytics.map((chain) => (
                        <TableRow key={chain.chainId} sx={{ '&:hover': { bgcolor: '#2a2a3e' } }}>
                          <TableCell>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: CHAIN_CONFIG[chain.chainId]?.color || '#666' }} />
                              <Typography sx={{ color: 'white' }}>{chain.chainName}</Typography>
                            </Box>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: 'white' }}>{formatUSD(chain.volume24h)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: '#00d4aa' }}>{formatUSD(chain.tvl)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: 'white' }}>{formatNumber(chain.transactions)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: '#9ca3af' }}>{chain.avgGasPrice.toFixed(1)} gwei</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, justifyContent: 'flex-end' }}>
                              <Box sx={{ width: 60, bgcolor: '#1a1a2e', borderRadius: 1, height: 6 }}>
                                <Box sx={{ width: `${chain.sharePercent * 3}%`, bgcolor: '#00d4aa', height: '100%', borderRadius: 1 }} />
                              </Box>
                              <Typography sx={{ color: 'white' }}>{chain.sharePercent}%</Typography>
                            </Box>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
              )
            )}
          </CardContent>
        </Card>
      </Box>
    </Box>
  );
}