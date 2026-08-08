'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, Chip,
  CircularProgress, Snackbar, Alert, Divider, LinearProgress,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Tabs, Tab, IconButton
} from '@mui/material';
import {
  AccountBalance, Wallet, ShowChart, TrendingUp, TrendingDown,
  SwapHoriz, Pool, History, Refresh
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';
import { api, Portfolio, Transaction } from '@/lib/api/client';

// ============================================================================
// Utility Functions
// ============================================================================

function formatUSD(amount: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount);
}

function formatNumber(num: number): string {
  if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M';
  if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K';
  return num.toFixed(2);
}

function timeAgo(timestamp: number): string {
  const diff = Date.now() - timestamp;
  const hours = Math.floor(diff / 3600000);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function formatAddress(hash: string): string {
  return `${hash.slice(0, 10)}...${hash.slice(-8)}`;
}

// ============================================================================
// Main Portfolio Page
// ============================================================================

export default function PortfolioPage() {
  const [portfolio, setPortfolio] = useState<Portfolio | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState(0);
  const [refreshing, setRefreshing] = useState(false);

  const fetchPortfolio = useCallback(async () => {
    try {
      setError(null);
      setRefreshing(true);
      const res = await api.getPortfolio();
      if (res.success && res.data) {
        setPortfolio(res.data);
      } else {
        setPortfolio({ assets: [], positions: [], transactions: [] });
        if (res.error) setError(res.error);
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to load portfolio');
      setPortfolio({ assets: [], positions: [], transactions: [] });
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchPortfolio();
  }, [fetchPortfolio]);

  const assets = portfolio?.assets ?? [];
  const positions = portfolio?.positions ?? [];
  const transactions = portfolio?.transactions ?? [];

  const totalValue = assets.reduce((sum, a) => sum + (a.value || 0), 0);
  const totalPnL = positions.reduce((sum, p) => sum + (p.pnl || 0), 0);
  const change24h = assets.reduce((sum, a) => sum + (a.value || 0) * ((a.change24h || 0) / 100), 0);

  const EmptyRow = ({ colSpan, label }: { colSpan: number; label: string }) => (
    <TableRow>
      <TableCell colSpan={colSpan} align="center" sx={{ color: '#9ca3af', py: 4 }}>{label}</TableCell>
    </TableRow>
  );

  if (loading) {
    return (
      <Box sx={{ minHeight: '100vh', bgcolor: '#0a0a14', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
        <CircularProgress sx={{ color: '#00d4aa' }} />
      </Box>
    );
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#0a0a14', p: 3 }}>
      <Box sx={{ maxWidth: 1400, mx: 'auto' }}>
        {/* Header */}
        <Box sx={{ mb: 4, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <Box>
            <Typography variant="h4" sx={{ color: 'white', fontWeight: 'bold' }}>
              💼 Portfolio
            </Typography>
            <Typography variant="body2" sx={{ color: '#9ca3af', mt: 1 }}>
              Track your assets, positions, and transaction history
            </Typography>
          </Box>
          <IconButton onClick={fetchPortfolio} disabled={refreshing} sx={{ color: '#00d4aa' }}>
            <Refresh />
          </IconButton>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 3, bgcolor: '#ff572220', color: '#ff5722' }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {refreshing && <LinearProgress sx={{ mb: 2, color: '#00d4aa' }} />}

        {/* Overview Cards */}
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, mb: 4 }}>
          <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3 }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <Wallet sx={{ color: '#00d4aa' }} />
                <Typography variant="caption" sx={{ color: '#9ca3af' }}>Total Value</Typography>
              </Box>
              <Typography variant="h5" sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
                {formatUSD(totalValue)}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3 }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <ShowChart sx={{ color: '#00d4aa' }} />
                <Typography variant="caption" sx={{ color: '#9ca3af' }}>24h Change</Typography>
              </Box>
              <Typography variant="h5" sx={{ color: change24h >= 0 ? '#00d4aa' : '#ff5722', fontWeight: 'bold' }}>
                {change24h >= 0 ? '+' : ''}{formatUSD(change24h)}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3 }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <Pool sx={{ color: '#ff9800' }} />
                <Typography variant="caption" sx={{ color: '#9ca3af' }}>Positions Value</Typography>
              </Box>
              <Typography variant="h5" sx={{ color: 'white', fontWeight: 'bold' }}>
                {formatUSD(positions.reduce((s, p) => s + (p.value || 0), 0))}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3 }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <TrendingUp sx={{ color: '#00d4aa' }} />
                <Typography variant="caption" sx={{ color: '#9ca3af' }}>Total P&L</Typography>
              </Box>
              <Typography variant="h5" sx={{ color: totalPnL >= 0 ? '#00d4aa' : '#ff5722', fontWeight: 'bold' }}>
                {totalPnL >= 0 ? '+' : ''}{formatUSD(totalPnL)}
              </Typography>
            </CardContent>
          </Card>
        </Box>

        {/* Tabs */}
        <Card sx={{ bgcolor: '#1a1a2e', borderRadius: 3, mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={(_, v) => setActiveTab(v)}
            sx={{ borderBottom: '1px solid #2a2a3e', '& .MuiTab-root': { color: '#9ca3af' }, '& .Mui-selected': { color: '#00d4aa' } }}
          >
            <Tab label="Assets" />
            <Tab label="Positions" />
            <Tab label="History" />
          </Tabs>

          <CardContent sx={{ p: 3 }}>
            {activeTab === 0 && (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: '#9ca3af' }}>Asset</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Balance</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Value</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">24h Change</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {assets.length === 0 ? (
                      <EmptyRow colSpan={4} label="No assets found" />
                    ) : (
                      assets.map(asset => (
                        <TableRow key={asset.symbol} sx={{ '&:hover': { bgcolor: '#2a2a3e' } }}>
                          <TableCell>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                              <Typography sx={{ fontSize: 24 }}>{asset.icon || '🪙'}</Typography>
                              <Box>
                                <Typography sx={{ color: 'white', fontWeight: 'bold' }}>{asset.symbol}</Typography>
                                <Typography variant="caption" sx={{ color: '#9ca3af' }}>{asset.name}</Typography>
                              </Box>
                            </Box>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: 'white' }}>{(asset.balance || 0).toLocaleString()}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: '#00d4aa' }}>{formatUSD(asset.value || 0)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: (asset.change24h || 0) >= 0 ? '#00d4aa' : '#ff5722' }}>
                              {(asset.change24h || 0) >= 0 ? '+' : ''}{(asset.change24h || 0).toFixed(2)}%
                            </Typography>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            )}

            {activeTab === 1 && (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: '#9ca3af' }}>Position</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Value</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">APR</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">P&L</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {positions.length === 0 ? (
                      <EmptyRow colSpan={4} label="No positions found" />
                    ) : (
                      positions.map((pos, i) => (
                        <TableRow key={i} sx={{ '&:hover': { bgcolor: '#2a2a3e' } }}>
                          <TableCell>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                              <Typography sx={{ fontSize: 24 }}>{pos.icon || '🧩'}</Typography>
                              <Box>
                                <Typography sx={{ color: 'white', fontWeight: 'bold' }}>{pos.pair}</Typography>
                                <Typography variant="caption" sx={{ color: '#9ca3af' }}>{pos.protocol}</Typography>
                              </Box>
                            </Box>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: '#00d4aa' }}>{formatUSD(pos.value || 0)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Chip label={`${(pos.apr || 0).toFixed(1)}%`} size="small" sx={{ bgcolor: '#00d4aa20', color: '#00d4aa' }} />
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: (pos.pnl || 0) >= 0 ? '#00d4aa' : '#ff5722' }}>
                              {(pos.pnl || 0) >= 0 ? '+' : ''}{formatUSD(pos.pnl || 0)}
                            </Typography>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            )}

            {activeTab === 2 && (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: '#9ca3af' }}>Type</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }}>Details</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Value</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Status</TableCell>
                      <TableCell sx={{ color: '#9ca3af' }} align="right">Time</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {transactions.length === 0 ? (
                      <EmptyRow colSpan={5} label="No transactions found" />
                    ) : (
                      transactions.map(tx => (
                        <TableRow key={tx.hash || tx.id} sx={{ '&:hover': { bgcolor: '#2a2a3e' } }}>
                          <TableCell>
                            <Chip label="Transaction" size="small" sx={{ bgcolor: '#2a2a3e' }} />
                          </TableCell>
                          <TableCell>
                            <Typography sx={{ color: 'white' }}>{tx.value} {tx.from ? `from ${formatAddress(tx.from)}` : ''} → {tx.to ? formatAddress(tx.to) : ''}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: '#00d4aa' }}>{formatUSD(Number(tx.value) || 0)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Chip
                              label={tx.status}
                              size="small"
                              sx={{
                                bgcolor: tx.status === 'confirmed' ? '#00d4aa20' : tx.status === 'pending' ? '#ff980020' : '#ff572220',
                                color: tx.status === 'confirmed' ? '#00d4aa' : tx.status === 'pending' ? '#ff9800' : '#ff5722'
                              }}
                            />
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: '#9ca3af' }}>{timeAgo(tx.timestamp)}</Typography>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </CardContent>
        </Card>
      </Box>
    </Box>
  );
}