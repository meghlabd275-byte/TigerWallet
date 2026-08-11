'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Chip, IconButton, Select, MenuItem, FormControl, InputLabel,
  Dialog, DialogTitle, DialogContent, DialogActions, Tabs, Tab,
  LinearProgress, Snackbar, Alert, CircularProgress, Tooltip,
  Slider, InputAdornment, Divider
} from '@mui/material';
import {
  Pool, Agriculture, TrendingUp, Add, Remove, Lock, LockOpen,
  Refresh, ShowChart, Info, Warning, CheckCircle, AccessTime,
  AccountBalance, Favorite
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';

// ============================================================================
// Types & Interfaces
// ============================================================================

interface FarmPool {
  id: string;
  pair: string;
  pairIcon: string;
  protocol: string;
  chainId: number;
  chainName: string;
  totalStaked: number;
  tvl: number;
  apr: number;
  apy: number;
  rewardToken: string;
  rewardPerDay: number;
  minStake: number;
  lockPeriod: number;
  isActive: boolean;
  allocPoint: number;
}

interface UserStake {
  poolId: string;
  amount: string;
  pendingRewards: string;
  lastClaimTime: number;
  unlockTime?: number;
  isLocked: boolean;
}

interface StakeRequest {
  poolId: string;
  amount: string;
  lockPeriod?: number;
}

interface UnstakeRequest {
  poolId: string;
  amount: string;
}

interface ClaimRequest {
  poolId: string;
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
};

const LOCK_PERIODS = [
  { days: 0, label: 'No Lock', multiplier: 1 },
  { days: 7, label: '7 Days', multiplier: 1.5 },
  { days: 30, label: '30 Days', multiplier: 2 },
  { days: 90, label: '90 Days', multiplier: 3 },
  { days: 180, label: '180 Days', multiplier: 4 },
  { days: 365, label: '1 Year', multiplier: 5 },
];

// ============================================================================
// Utility Functions
// ============================================================================

function formatNumber(num: number, decimals: number = 2): string {
  if (num >= 1e9) return (num / 1e9).toFixed(decimals) + 'B';
  if (num >= 1e6) return (num / 1e6).toFixed(decimals) + 'M';
  if (num >= 1e3) return (num / 1e3).toFixed(decimals) + 'K';
  return num.toFixed(decimals);
}

function formatUSD(amount: number): string {
  if (amount >= 1e9) return `$${(amount / 1e9).toFixed(2)}B`;
  if (amount >= 1e6) return `$${(amount / 1e6).toFixed(2)}M`;
  if (amount >= 1e3) return `$${(amount / 1e3).toFixed(2)}K`;
  return `$${amount.toFixed(2)}`;
}

function formatPercent(value: number): string {
  return `${value.toFixed(2)}%`;
}

function formatTokens(amount: string, decimals: number = 18): string {
  if (!amount || amount === '0') return '0';
  try {
    const num = Number(amount) / Math.pow(10, decimals);
    if (num < 0.0001) return '<0.0001';
    return num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 6 });
  } catch {
    return '0';
  }
}

function timeUntil(unlockTime: number): string {
  const now = Date.now();
  const diff = unlockTime - now;
  if (diff <= 0) return 'LockOpened';
  const days = Math.floor(diff / (24 * 60 * 60 * 1000));
  const hours = Math.floor((diff % (24 * 60 * 60 * 1000)) / (60 * 60 * 1000));
  return `${days}d ${hours}h`;
}

// ============================================================================
// ============================================================================
// Main Farming Page Component
// ============================================================================

export default function FarmingPage() {
  // State
  const [pools, setPools] = useState<FarmPool[]>([]);
  const [userStakes, setUserStakes] = useState<UserStake[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPool, setSelectedPool] = useState<FarmPool | null>(null);
  const [filterChain, setFilterChain] = useState<number>(0);
  const [sortBy, setSortBy] = useState<'apr' | 'tvl' | 'apy'>('apr');
  const [activeTab, setActiveTab] = useState(0);

  // Staking dialog state
  const [showStakeDialog, setShowStakeDialog] = useState(false);
  const [showUnstakeDialog, setShowUnstakeDialog] = useState(false);
  const [stakeAmount, setStakeAmount] = useState('');
  const [unstakeAmount, setUnstakeAmount] = useState('');
  const [selectedLockPeriod, setSelectedLockPeriod] = useState(0);
  const [staking, setStaking] = useState(false);

  // Snackbar
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'success' as 'success' | 'error' | 'info'
  });

  // ============================================================================
  // Data Loading
  // ============================================================================

  const loadPools = useCallback(async () => {
    setLoading(true);
    try {
      throw new Error('Farming data is unavailable until an authenticated farming API is configured.')
    } catch (error) {
      console.error('Failed to load pools:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPools();
  }, [loadPools]);

  // ============================================================================
  // Staking Functions
  // ============================================================================

  const handleStake = async () => {
    if (!selectedPool || !stakeAmount) return;

    setSnackbar({ open: true, message: 'Staking is unavailable until an authenticated farming contract provider is configured.', severity: 'error' });
  };

  const handleUnstake = async () => {
    if (!selectedPool || !unstakeAmount) return;

    const userStake = userStakes.find(s => s.poolId === selectedPool.id);
    if (userStake?.isLocked) {
      setSnackbar({ open: true, message: 'Cannot unstake while position is locked', severity: 'error' });
      return;
    }

    setSnackbar({ open: true, message: 'Unstaking is unavailable until an authenticated farming contract provider is configured.', severity: 'error' });
  };

  const handleClaim = async (poolId: string) => {
    const stake = userStakes.find(s => s.poolId === poolId);
    if (!stake || parseFloat(stake.pendingRewards) <= 0) return;

    setSnackbar({ open: true, message: 'Reward claims are unavailable until an authenticated farming contract provider is configured.', severity: 'error' });
  };

  // ============================================================================
  // Filtering & Sorting
  // ============================================================================

  const filteredPools = pools
    .filter(pool => filterChain === 0 || pool.chainId === filterChain)
    .sort((a, b) => {
      switch (sortBy) {
        case 'tvl': return b.tvl - a.tvl;
        case 'apy': return b.apy - a.apy;
        default: return b.apr - a.apr;
      }
    });

  const userTvl = userStakes.reduce((sum, s) => {
    const pool = pools.find(p => p.id === s.poolId);
    return sum + (pool ? parseFloat(s.amount) * pool.tvl / pool.totalStaked : 0);
  }, 0);

  const userPendingRewards = userStakes.reduce((sum, s) => sum + parseFloat(s.pendingRewards), 0);

  // ============================================================================
  // Main Render
  // ============================================================================

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'var(--bg-primary)', p: 3 }}>
      <Box sx={{ maxWidth: 1600, mx: 'auto' }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Box>
            <Typography variant="h4" sx={{ color: 'white', fontWeight: 'bold' }}>
              🌾 Yield Farming
            </Typography>
            <Typography variant="body2" sx={{ color: 'var(--text-secondary)', mt: 1 }}>
              Stake LP tokens and earn rewards from multiple protocols
            </Typography>
          </Box>
          <Button
            variant="outlined"
            startIcon={<Refresh />}
            onClick={loadPools}
            sx={{ borderColor: 'var(--bg-tertiary)', color: 'white' }}
          >
            Refresh
          </Button>
        </Box>

        {/* User Stats */}
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2, mb: 4 }}>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Your Staked Value</Typography>
              <Typography variant="h5" sx={{ color: '#00d4aa', fontWeight: 'bold' }}>
                {formatUSD(userTvl)}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Pending Rewards</Typography>
              <Typography variant="h5" sx={{ color: '#ff9800', fontWeight: 'bold' }}>
                {userPendingRewards.toFixed(4)} TIGER
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Total Value Locked</Typography>
              <Typography variant="h5" sx={{ color: 'white', fontWeight: 'bold' }}>
                {formatUSD(pools.reduce((sum, p) => sum + p.tvl, 0))}
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ bgcolor: 'var(--bg-primary)', borderRadius: 3 }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>Active Farms</Typography>
              <Typography variant="h5" sx={{ color: 'white', fontWeight: 'bold' }}>
                {pools.filter(p => p.isActive).length}
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
            <Tab label="All Farms" />
            <Tab label="My Positions" />
            <Tab label="Best APR" />
          </Tabs>

          <CardContent sx={{ p: 3 }}>
            {/* Filters */}
            <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
              <FormControl size="small" sx={{ minWidth: 150 }}>
                <Select
                  value={filterChain}
                  onChange={(e) => setFilterChain(e.target.value as number)}
                  sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                >
                  <MenuItem value={0}>All Chains</MenuItem>
                  {Object.entries(CHAIN_CONFIG).map(([id, config]) => (
                    <MenuItem key={id} value={parseInt(id)}>{config.name}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControl size="small" sx={{ minWidth: 150 }}>
                <Select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as any)}
                  sx={{ color: 'white', '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--bg-tertiary)' } }}
                >
                  <MenuItem value="apr">Sort by APR</MenuItem>
                  <MenuItem value="tvl">Sort by TVL</MenuItem>
                  <MenuItem value="apy">Sort by APY</MenuItem>
                </Select>
              </FormControl>
            </Box>

            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
                <CircularProgress sx={{ color: '#00d4aa' }} />
              </Box>
            ) : activeTab === 0 ? (
              /* All Farms */
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Pool</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }}>Protocol</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">TVL</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">APR</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">APY</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Daily</TableCell>
                      <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {filteredPools.map((pool) => {
                      const userStake = userStakes.find(s => s.poolId === pool.id);
                      return (
                        <TableRow key={pool.id} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                          <TableCell>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Typography sx={{ fontSize: 24 }}>{pool.pairIcon}</Typography>
                              <Box>
                                <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{pool.pair}</Typography>
                                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                                  {pool.chainName}
                                </Typography>
                              </Box>
                            </Box>
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={pool.protocol}
                              size="small"
                              sx={{ bgcolor: 'var(--bg-secondary)', color: 'var(--text-secondary)' }}
                            />
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: 'white' }}>{formatUSD(pool.tvl)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: pool.apr > 30 ? '#00d4aa' : pool.apr > 15 ? '#ff9800' : '#ff4757', fontWeight: 'bold' }}>
                              {formatPercent(pool.apr)}
                            </Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: 'white' }}>{formatPercent(pool.apy)}</Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Typography sx={{ color: 'var(--text-secondary)' }}>
                              {pool.rewardPerDay.toFixed(4)} {pool.rewardToken}
                            </Typography>
                          </TableCell>
                          <TableCell align="right">
                            <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                              <Button
                                size="small"
                                variant="contained"
                                startIcon={<Add />}
                                onClick={() => { setSelectedPool(pool); setShowStakeDialog(true); }}
                                sx={{ bgcolor: '#00d4aa', color: 'black', minWidth: 0, px: 1 }}
                              >
                                Stake
                              </Button>
                              {userStake && (
                                <>
                                  <Button
                                    size="small"
                                    variant="outlined"
                                    startIcon={<Remove />}
                                    onClick={() => { setSelectedPool(pool); setShowUnstakeDialog(true); }}
                                    sx={{ borderColor: 'var(--bg-tertiary)', color: 'white', minWidth: 0, px: 1 }}
                                  >
                                    Unstake
                                  </Button>
                                  {parseFloat(userStake.pendingRewards) > 0 && (
                                    <Button
                                      size="small"
                                      variant="contained"
                                      onClick={() => handleClaim(pool.id)}
                                      sx={{ bgcolor: '#ff9800', color: 'black', minWidth: 0, px: 1 }}
                                    >
                                      Claim
                                    </Button>
                                  )}
                                </>
                              )}
                            </Box>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </TableContainer>
            ) : activeTab === 1 ? (
              /* My Positions */
              <Box>
                {userStakes.length === 0 ? (
                  <Box sx={{ textAlign: 'center', py: 5 }}>
                    <Typography sx={{ color: 'var(--text-secondary)', mb: 2 }}>No active staking positions</Typography>
                    <Button
                      variant="contained"
                      startIcon={<Add />}
                      onClick={() => setActiveTab(0)}
                      sx={{ bgcolor: '#00d4aa', color: 'black' }}
                    >
                      Find a Farm
                    </Button>
                  </Box>
                ) : (
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ color: 'var(--text-secondary)' }}>Pool</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Staked</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Pending Rewards</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Lock Status</TableCell>
                          <TableCell sx={{ color: 'var(--text-secondary)' }} align="right">Actions</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {userStakes.map((stake) => {
                          const pool = pools.find(p => p.id === stake.poolId);
                          if (!pool) return null;
                          return (
                            <TableRow key={stake.poolId} sx={{ '&:hover': { bgcolor: 'var(--bg-secondary)' } }}>
                              <TableCell>
                                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                  <Typography sx={{ fontSize: 24 }}>{pool.pairIcon}</Typography>
                                  <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{pool.pair}</Typography>
                                </Box>
                              </TableCell>
                              <TableCell align="right">
                                <Typography sx={{ color: 'white' }}>
                                  {parseFloat(stake.amount).toFixed(4)} LP
                                </Typography>
                                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                                  ≈ {formatUSD(parseFloat(stake.amount) * pool.tvl / pool.totalStaked)}
                                </Typography>
                              </TableCell>
                              <TableCell align="right">
                                <Typography sx={{ color: '#ff9800', fontWeight: 'bold' }}>
                                  {parseFloat(stake.pendingRewards).toFixed(4)} {pool.rewardToken}
                                </Typography>
                              </TableCell>
                              <TableCell align="right">
                                {stake.isLocked && stake.unlockTime ? (
                                  <Chip
                                    icon={<Lock sx={{ fontSize: 16 }} />}
                                    label={timeUntil(stake.unlockTime)}
                                    size="small"
                                    sx={{ bgcolor: '#ff572220', color: '#ff5722' }}
                                  />
                                ) : (
                                  <Chip
                                    icon={<LockOpen sx={{ fontSize: 16 }} />}
                                    label="LockOpened"
                                    size="small"
                                    sx={{ bgcolor: '#00d4aa20', color: '#00d4aa' }}
                                  />
                                )}
                              </TableCell>
                              <TableCell align="right">
                                <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                                  <Button
                                    size="small"
                                    variant="contained"
                                    disabled={parseFloat(stake.pendingRewards) <= 0}
                                    onClick={() => handleClaim(pool.id)}
                                    sx={{ bgcolor: '#ff9800', color: 'black' }}
                                  >
                                    Claim
                                  </Button>
                                  <Button
                                    size="small"
                                    variant="outlined"
                                    disabled={stake.isLocked}
                                    onClick={() => { setSelectedPool(pool); setShowUnstakeDialog(true); }}
                                    sx={{ borderColor: 'var(--bg-tertiary)', color: 'white' }}
                                  >
                                    Unstake
                                  </Button>
                                </Box>
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
              </Box>
            ) : (
              /* Best APR - Top 5 */
              <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 3 }}>
                {filteredPools.slice(0, 5).map((pool) => (
                  <Card key={pool.id} sx={{ bgcolor: 'var(--bg-secondary)', borderRadius: 3, border: '1px solid #00d4aa30' }}>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography sx={{ fontSize: 32 }}>{pool.pairIcon}</Typography>
                          <Box>
                            <Typography sx={{ color: 'white', fontWeight: 'bold' }}>{pool.pair}</Typography>
                            <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>{pool.chainName}</Typography>
                          </Box>
                        </Box>
                        <Chip label={pool.protocol} size="small" sx={{ bgcolor: 'var(--bg-primary)' }} />
                      </Box>
                      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 2 }}>
                        <Box>
                          <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>APR</Typography>
                          <Typography sx={{ color: '#00d4aa', fontWeight: 'bold', fontSize: 24 }}>
                            {formatPercent(pool.apr)}
                          </Typography>
                        </Box>
                        <Box>
                          <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>TVL</Typography>
                          <Typography sx={{ color: 'white', fontWeight: 'bold' }}>
                            {formatUSD(pool.tvl)}
                          </Typography>
                        </Box>
                      </Box>
                      <Button
                        fullWidth
                        variant="contained"
                        startIcon={<Add />}
                        onClick={() => { setSelectedPool(pool); setShowStakeDialog(true); }}
                        sx={{ bgcolor: '#00d4aa', color: 'black' }}
                      >
                        Stake {pool.pair}
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </Box>
            )}
          </CardContent>
        </Card>
      </Box>

      {/* Stake Dialog */}
      <Dialog
        open={showStakeDialog}
        onClose={() => setShowStakeDialog(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { bgcolor: 'var(--bg-primary)', backgroundImage: 'none' } }}
      >
        <DialogTitle sx={{ color: 'white', display: 'flex', justifyContent: 'space-between' }}>
          Stake {selectedPool?.pair}
          <IconButton onClick={() => setShowStakeDialog(false)} sx={{ color: 'white' }}>
            ✕
          </IconButton>
        </DialogTitle>
        <DialogContent>
          {selectedPool && (
            <Box sx={{ mt: 2 }}>
              <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2, mb: 3 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography sx={{ color: 'var(--text-secondary)' }}>APR</Typography>
                  <Typography sx={{ color: '#00d4aa', fontWeight: 'bold' }}>{formatPercent(selectedPool.apr)}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                  <Typography sx={{ color: 'var(--text-secondary)' }}>Daily Rewards</Typography>
                  <Typography sx={{ color: 'white' }}>{selectedPool.rewardPerDay} {selectedPool.rewardToken}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography sx={{ color: 'var(--text-secondary)' }}>Min Stake</Typography>
                  <Typography sx={{ color: 'white' }}>{selectedPool.minStake} LP</Typography>
                </Box>
              </Box>

              <TextField
                fullWidth
                type="number"
                label="Amount to Stake"
                value={stakeAmount}
                onChange={(e) => setStakeAmount(e.target.value)}
                InputProps={{
                  endAdornment: <InputAdornment position="end" sx={{ color: "var(--text-secondary)" }}>LP</InputAdornment>,
                }}
                sx={{
                  mb: 3,
                  '& .MuiInputLabel-root': { color: 'var(--text-secondary)' },
                  '& input': { color: 'white' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />

              {selectedPool.lockPeriod > 0 && (
                <Box sx={{ mb: 3 }}>
                  <Typography sx={{ color: 'var(--text-secondary)', mb: 2 }}>Lock Period (Higher multiplier)</Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    {LOCK_PERIODS.map(period => (
                      <Chip
                        key={period.days}
                        label={`${period.label} (${period.multiplier}x)`}
                        onClick={() => setSelectedLockPeriod(period.days)}
                        sx={{
                          bgcolor: selectedLockPeriod === period.days ? '#00d4aa' : 'var(--bg-secondary)',
                          color: selectedLockPeriod === period.days ? 'black' : 'white',
                          cursor: 'pointer',
                        }}
                      />
                    ))}
                  </Box>
                </Box>
              )}

              <Box sx={{ bgcolor: 'var(--bg-secondary)', p: 2, borderRadius: 2 }}>
                <Typography variant="caption" sx={{ color: 'var(--text-secondary)' }}>
                  Your stake will earn {selectedLockPeriod > 0 ? LOCK_PERIODS.find(p => p.days === selectedLockPeriod)?.multiplier : 1}x rewards.
                  {selectedLockPeriod > 0 && ' You cannot unstake until the lock period ends.'}
                </Typography>
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 3 }}>
          <Button onClick={() => setShowStakeDialog(false)} sx={{ color: 'var(--text-secondary)' }}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleStake}
            disabled={staking || !stakeAmount || parseFloat(stakeAmount) <= 0}
            sx={{ bgcolor: '#00d4aa', color: 'black' }}
          >
            {staking ? <CircularProgress size={20} sx={{ color: 'black' }} /> : 'Stake'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Unstake Dialog */}
      <Dialog
        open={showUnstakeDialog}
        onClose={() => setShowUnstakeDialog(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { bgcolor: 'var(--bg-primary)', backgroundImage: 'none' } }}
      >
        <DialogTitle sx={{ color: 'white', display: 'flex', justifyContent: 'space-between' }}>
          Unstake {selectedPool?.pair}
          <IconButton onClick={() => setShowUnstakeDialog(false)} sx={{ color: 'white' }}>✕</IconButton>
        </DialogTitle>
        <DialogContent>
          {selectedPool && (
            <Box sx={{ mt: 2 }}>
              <Typography sx={{ color: 'var(--text-secondary)', mb: 3 }}>
                Enter the amount of LP tokens to unstake. You will also claim any pending rewards.
              </Typography>

              <TextField
                fullWidth
                type="number"
                label="Amount to Unstake"
                value={unstakeAmount}
                onChange={(e) => setUnstakeAmount(e.target.value)}
                InputProps={{
                  endAdornment: <InputAdornment position="end" sx={{ color: "var(--text-secondary)" }}>LP</InputAdornment>,
                }}
                sx={{
                  mb: 3,
                  '& .MuiInputLabel-root': { color: 'var(--text-secondary)' },
                  '& input': { color: 'white' },
                  '& .MuiOutlinedInput-root': { '& fieldset': { borderColor: 'var(--bg-tertiary)' } },
                }}
              />

              <Button fullWidth variant="text" sx={{ color: '#00d4aa' }} onClick={() => setUnstakeAmount('MAX')}>
                MAX
              </Button>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 3 }}>
          <Button onClick={() => setShowUnstakeDialog(false)} sx={{ color: 'var(--text-secondary)' }}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleUnstake}
            disabled={staking || !unstakeAmount || parseFloat(unstakeAmount) <= 0}
            sx={{ bgcolor: '#ff5722', color: 'white' }}
          >
            {staking ? <CircularProgress size={20} sx={{ color: 'white' }} /> : 'Unstake'}
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