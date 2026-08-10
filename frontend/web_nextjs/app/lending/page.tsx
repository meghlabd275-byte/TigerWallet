'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { 
  Box, Typography, Card, CardContent, Button, TextField, 
  Chip, CircularProgress, Alert, IconButton, InputAdornment,
  Slider, Divider, Stack, Switch, FormControlLabel, Dialog,
  DialogTitle, DialogContent, List, ListItem, ListItemText,
  Tabs, Tab, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, LinearProgress, Tooltip
} from '@mui/material';
import {
  AccountBalance, TrendingUp, TrendingDown, ShowChart,
  ArrowForward, ArrowBack, Refresh, Warning, Info,
  MonetizationOn, SwapVert, WarningAmber, CheckCircle
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';

// ============================================================================
// Types
// ============================================================================

interface Market {
  id: number;
  asset_address: string;
  asset_symbol: string;
  asset_name: string;
  asset_decimals: number;
  total_supply: string;
  total_borrows: string;
  supply_apy: number;
  borrow_apy: number;
  utilization_rate: number;
  ltv: number;
  liquidation_threshold: number;
  liquidation_bonus: number;
  is_active: boolean;
  chain_id: number;
}

interface UserSupply {
  id: number;
  user_address: string;
  market_id: number;
  asset_address: string;
  balance: string;
  balance_usd: number;
  accrued_rewards: string;
  apy: number;
}

interface UserBorrow {
  id: number;
  user_address: string;
  market_id: number;
  asset_address: string;
  balance: string;
  balance_usd: number;
  accrued_interest: string;
  apy: number;
}

interface UserPosition {
  supplies: UserSupply[];
  borrows: UserBorrow[];
  collateral_usd: number;
  borrows_usd: number;
  health_factor: number;
  net_apy: number;
}

interface SupplyRequest {
  user_address: string;
  asset_address: string;
  amount: string;
  chain_id: number;
}

interface SupplyResponse {
  success: boolean;
  transaction_hash?: string;
  new_balance: string;
  new_balance_usd: number;
  apy: number;
  error?: string;
}

// ============================================================================
// Constants
// ============================================================================

// Same-origin API base: the Next.js app proxies /api/v1/lending/* to the
// go/lending_service backend (see app/api/v1/lending/ routes).
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

const DEFAULT_MARKETS: Market[] = [
  { id: 1, asset_address: '0x0000000000000000000000000000000000000000', asset_symbol: 'ETH', asset_name: 'Ethereum', asset_decimals: 18, total_supply: '0', total_borrows: '0', supply_apy: 3.5, borrow_apy: 5.2, utilization_rate: 0.65, ltv: 0.80, liquidation_threshold: 0.85, liquidation_bonus: 0.05, is_active: true, chain_id: 1 },
  { id: 2, asset_address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', asset_symbol: 'USDT', asset_name: 'Tether USD', asset_decimals: 6, total_supply: '0', total_borrows: '0', supply_apy: 4.2, borrow_apy: 5.8, utilization_rate: 0.72, ltv: 0.90, liquidation_threshold: 0.95, liquidation_bonus: 0.02, is_active: true, chain_id: 1 },
  { id: 3, asset_address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', asset_symbol: 'USDC', asset_name: 'USD Coin', asset_decimals: 6, total_supply: '0', total_borrows: '0', supply_apy: 4.0, borrow_apy: 5.5, utilization_rate: 0.68, ltv: 0.90, liquidation_threshold: 0.95, liquidation_bonus: 0.02, is_active: true, chain_id: 1 },
  { id: 4, asset_address: '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', asset_symbol: 'WBTC', asset_name: 'Wrapped Bitcoin', asset_decimals: 8, total_supply: '0', total_borrows: '0', supply_apy: 1.8, borrow_apy: 3.5, utilization_rate: 0.45, ltv: 0.70, liquidation_threshold: 0.80, liquidation_bonus: 0.05, is_active: true, chain_id: 1 },
];

// ============================================================================
// Utility Functions
// ============================================================================

function formatUSD(amount: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(amount);
}

function formatPercent(value: number): string {
  return `${value.toFixed(2)}%`;
}

function formatAddress(address: string, chars: number = 4): string {
  if (!address) return '';
  return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`;
}

function shortenNumber(num: number): string {
  if (num >= 1e9) return (num / 1e9).toFixed(2) + 'B';
  if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M';
  if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K';
  return num.toFixed(2);
}

// ============================================================================
// Main Component
// ============================================================================

export default function LendingPage() {
  const { isDark } = useTheme();
  const [activeTab, setActiveTab] = useState(0);
  const [markets, setMarkets] = useState<Market[]>(DEFAULT_MARKETS);
  const [userPosition, setUserPosition] = useState<UserPosition | null>(null);
  const [selectedMarket, setSelectedMarket] = useState<Market | null>(null);
  const [amount, setAmount] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [walletAddress, setWalletAddress] = useState<string>('');
  const [operation, setOperation] = useState<'supply' | 'borrow'>('supply');
  const [dialogOpen, setDialogOpen] = useState(false);

  // Simulated wallet address (in production, connect to wallet)
  useEffect(() => {
    const savedWallet = localStorage.getItem('tigerwallet_address');
    if (savedWallet) {
      setWalletAddress(savedWallet);
    }
  }, []);

  // Fetch markets
  const fetchMarkets = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/lending/markets`);
      if (response.ok) {
        const data = await response.json();
        if (data.markets && data.markets.length > 0) {
          setMarkets(data.markets);
        }
      }
    } catch (err) {
      console.log('Using default markets');
    }
  }, []);

  // Fetch user position
  const fetchUserPosition = useCallback(async () => {
    if (!walletAddress) return;
    try {
      const response = await fetch(
        `${API_BASE}/api/v1/lending/position?user_address=${walletAddress}&chain_id=1`
      );
      if (response.ok) {
        const data = await response.json();
        setUserPosition(data);
      }
    } catch (err) {
      console.log('No position data');
    }
  }, [walletAddress]);

  useEffect(() => {
    fetchMarkets();
    fetchUserPosition();
  }, [fetchMarkets, fetchUserPosition]);

  // Handle supply
  const handleSupply = async () => {
    if (!selectedMarket || !amount || !walletAddress) return;
    
    setIsLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const request: SupplyRequest = {
        user_address: walletAddress,
        asset_address: selectedMarket.asset_address,
        amount: amount,
        chain_id: selectedMarket.chain_id,
      };

      const response = await fetch(`${API_BASE}/api/v1/lending/supply`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });

      const data: SupplyResponse = await response.json();

      if (data.success) {
        if (data.transaction_hash) {
          setSuccess(`Successfully supplied ${amount} ${selectedMarket.asset_symbol}. Tx: ${data.transaction_hash.slice(0, 10)}...`);
        } else {
          // The lending service prepared a real Aave V3 supply transaction
          // (to/data/chain_id). It must be signed and broadcast via the
          // wallet_api (POST /api/v1/send with wallet_id + password).
          setSuccess(`Supply transaction prepared for ${amount} ${selectedMarket.asset_symbol}. Submit it from your wallet to complete.`);
        }
        setAmount('');
        fetchUserPosition();
        fetchMarkets();
      } else {
        setError(data.error || 'Supply failed');
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Supply failed: lending service unavailable';
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  };

  // Handle borrow
  const handleBorrow = async () => {
    if (!selectedMarket || !amount || !walletAddress) return;
    
    setIsLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const request: SupplyRequest = {
        user_address: walletAddress,
        asset_address: selectedMarket.asset_address,
        amount: amount,
        chain_id: selectedMarket.chain_id,
      };

      const response = await fetch(`${API_BASE}/api/v1/lending/borrow`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });

      const data = await response.json();

      if (data.success) {
        if (data.transaction_hash) {
          setSuccess(`Successfully borrowed ${amount} ${selectedMarket.asset_symbol}. Tx: ${data.transaction_hash.slice(0, 10)}...`);
        } else {
          // The lending service prepared a real Aave V3 borrow transaction
          // (to/data/chain_id). It must be signed and broadcast via the
          // wallet_api (POST /api/v1/send with wallet_id + password).
          setSuccess(`Borrow transaction prepared for ${amount} ${selectedMarket.asset_symbol}. Submit it from your wallet to complete.`);
        }
        setAmount('');
        fetchUserPosition();
      } else {
        setError(data.error || 'Borrow failed');
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Borrow failed: lending service unavailable';
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  };

  // Open dialog for supply/borrow
  const openOperationDialog = (market: Market, op: 'supply' | 'borrow') => {
    setSelectedMarket(market);
    setOperation(op);
    setDialogOpen(true);
  };

  // Health factor color
  const getHealthFactorColor = (hf: number): string => {
    if (hf >= 2) return '#4caf50';
    if (hf >= 1.5) return '#ff9800';
    if (hf >= 1) return '#f44336';
    return '#d32f2f';
  };

  // Set wallet address (simplified)
  const handleConnectWallet = () => {
    const address = '0x742d35Cc6634C0532925a3b844Bc9e7595f5eA1E'; // Demo address
    localStorage.setItem('tigerwallet_address', address);
    setWalletAddress(address);
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900' : 'bg-slate-50'} text-${isDark ? 'white' : 'gray-900'}`}>
      {/* Header */}
      <header className={`${isDark ? 'bg-slate-800' : 'bg-white'} border-b ${isDark ? 'border-slate-700' : 'border-slate-200'} p-4`}>
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <a href="/" className="text-2xl">🐯</a>
            <h1 className="text-xl font-bold">Lending & Borrowing</h1>
          </div>
          <div className="flex items-center gap-4">
            {walletAddress ? (
              <Chip 
                label={formatAddress(walletAddress)} 
                onDelete={() => { localStorage.removeItem('tigerwallet_address'); setWalletAddress(''); }}
                className={isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'}
              />
            ) : (
              <Button 
                variant="contained" 
                onClick={handleConnectWallet}
                className="bg-orange-500 hover:bg-orange-600"
              >
                Connect Wallet
              </Button>
            )}
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto p-6">
        {/* User Position Summary */}
        {walletAddress && userPosition && (
          <Card className={`mb-6 ${isDark ? 'bg-slate-800' : 'bg-white'}`}>
            <CardContent>
              <Typography variant="h6" className="mb-4">Your Position</Typography>
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className={`p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-green-50'}`}>
                  <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>Total Supplied</Typography>
                  <Typography variant="h5" className="text-green-600">{formatUSD(userPosition.collateral_usd)}</Typography>
                </div>
                <div className={`p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-red-50'}`}>
                  <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>Total Borrowed</Typography>
                  <Typography variant="h5" className="text-red-600">{formatUSD(userPosition.borrows_usd)}</Typography>
                </div>
                <div className={`p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-blue-50'}`}>
                  <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>Net APY</Typography>
                  <Typography variant="h5" className={userPosition.net_apy >= 0 ? 'text-green-600' : 'text-red-600'}>
                    {formatPercent(userPosition.net_apy)}
                  </Typography>
                </div>
                <div className={`p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-yellow-50'}`}>
                  <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>Health Factor</Typography>
                  <Typography variant="h5" style={{ color: getHealthFactorColor(userPosition.health_factor) }}>
                    {userPosition.health_factor.toFixed(2)}
                  </Typography>
                </div>
              </div>
              
              {userPosition.health_factor < 1.5 && (
                <Alert severity="warning" className="mt-4">
                  <WarningAmber /> Your health factor is low. Please add collateral or repay loans to avoid liquidation.
                </Alert>
              )}
            </CardContent>
          </Card>
        )}

        {/* Tabs */}
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} className="mb-4">
          <Tab label="Markets" />
          <Tab label="Your Supplies" />
          <Tab label="Your Borrows" />
        </Tabs>

        {/* Markets Tab */}
        {activeTab === 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {markets.map(market => (
              <Card key={market.id} className={isDark ? 'bg-slate-800' : 'bg-white'}>
                <CardContent>
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-2">
                      <MonetizationOn className="text-orange-500" />
                      <div>
                        <Typography variant="h6">{market.asset_symbol}</Typography>
                        <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>
                          {market.asset_name}
                        </Typography>
                      </div>
                    </div>
                    <Chip label={`Chain: ${market.chain_id}`} size="small" />
                  </div>

                  <div className="grid grid-cols-2 gap-4 mb-4">
                    <div>
                      <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>Supply APY</Typography>
                      <Typography variant="h6" className="text-green-600 flex items-center">
                        <TrendingUp fontSize="small" className="mr-1" />
                        {formatPercent(market.supply_apy)}
                      </Typography>
                    </div>
                    <div>
                      <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>Borrow APY</Typography>
                      <Typography variant="h6" className="text-red-600 flex items-center">
                        <TrendingDown fontSize="small" className="mr-1" />
                        {formatPercent(market.borrow_apy)}
                      </Typography>
                    </div>
                  </div>

                  <div className="mb-4">
                    <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>
                      Utilization: {formatPercent(market.utilization_rate * 100)}
                    </Typography>
                    <LinearProgress 
                      variant="determinate" 
                      value={market.utilization_rate * 100} 
                      className="mt-1 h-2 rounded"
                      sx={{ 
                        backgroundColor: isDark ? '#334155' : '#e2e8f0',
                        '& .MuiLinearProgress-bar': {
                          backgroundColor: market.utilization_rate > 0.8 ? '#f44336' : '#4caf50'
                        }
                      }}
                    />
                  </div>

                  <div className="flex gap-2">
                    <Button 
                      variant="contained" 
                      color="success"
                      startIcon={<TrendingUp />}
                      onClick={() => openOperationDialog(market, 'supply')}
                      disabled={!walletAddress}
                      className="flex-1"
                    >
                      Supply
                    </Button>
                    <Button 
                      variant="contained" 
                      color="error"
                      startIcon={<TrendingDown />}
                      onClick={() => openOperationDialog(market, 'borrow')}
                      disabled={!walletAddress}
                      className="flex-1"
                    >
                      Borrow
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {/* Supplies Tab */}
        {activeTab === 1 && (
          <Card className={isDark ? 'bg-slate-800' : 'bg-white'}>
            <CardContent>
              {userPosition && userPosition.supplies.length > 0 ? (
                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell>Asset</TableCell>
                        <TableCell>Balance</TableCell>
                        <TableCell>Value (USD)</TableCell>
                        <TableCell>APY</TableCell>
                        <TableCell>Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {userPosition.supplies.map(supply => (
                        <TableRow key={supply.id}>
                          <TableCell>{supply.asset_address.slice(0, 10)}...</TableCell>
                          <TableCell>{supply.balance}</TableCell>
                          <TableCell>{formatUSD(supply.balance_usd)}</TableCell>
                          <TableCell className="text-green-600">{formatPercent(supply.apy)}</TableCell>
                          <TableCell>
                            <Button size="small" color="warning">Withdraw</Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography className={`text-center py-8 ${isDark ? "text-slate-400" : "text-gray-500"}`}>
                  No supplies yet. Start by supplying assets to a market.
                </Typography>
              )}
            </CardContent>
          </Card>
        )}

        {/* Borrows Tab */}
        {activeTab === 2 && (
          <Card className={isDark ? 'bg-slate-800' : 'bg-white'}>
            <CardContent>
              {userPosition && userPosition.borrows.length > 0 ? (
                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell>Asset</TableCell>
                        <TableCell>Balance</TableCell>
                        <TableCell>Value (USD)</TableCell>
                        <TableCell>APY</TableCell>
                        <TableCell>Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {userPosition.borrows.map(borrow => (
                        <TableRow key={borrow.id}>
                          <TableCell>{borrow.asset_address.slice(0, 10)}...</TableCell>
                          <TableCell>{borrow.balance}</TableCell>
                          <TableCell>{formatUSD(borrow.balance_usd)}</TableCell>
                          <TableCell className="text-red-600">{formatPercent(borrow.apy)}</TableCell>
                          <TableCell>
                            <Button size="small" color="primary">Repay</Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography className={`text-center py-8 ${isDark ? "text-slate-400" : "text-gray-500"}`}>
                  No active borrows.
                </Typography>
              )}
            </CardContent>
          </Card>
        )}

        {/* Error/Success Messages */}
        {error && (
          <Alert severity="error" className="mt-4" onClose={() => setError(null)}>
            {error}
          </Alert>
        )}
        {success && (
          <Alert severity="success" className="mt-4" onClose={() => setSuccess(null)}>
            <CheckCircle className="mr-2" />
            {success}
          </Alert>
        )}
      </div>

      {/* Supply/Borrow Dialog */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {operation === 'supply' ? 'Supply' : 'Borrow'} {selectedMarket?.asset_symbol}
        </DialogTitle>
        <DialogContent>
          <div className="py-4">
            <Typography variant="body2" className={`mb-4 ${isDark ? "text-slate-400" : "text-gray-500"}`}>
              {operation === 'supply' 
                ? `Supply ${selectedMarket?.asset_name} to earn interest. You can withdraw anytime.`
                : `Borrow ${selectedMarket?.asset_name} against your collateral. Maintain a healthy health factor.`
              }
            </Typography>
            
            <TextField
              fullWidth
              label="Amount"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              type="number"
              placeholder="0.00"
              className="mb-4"
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <Button 
                      size="small" 
                      onClick={() => setAmount('100')}
                    >
                      MAX
                    </Button>
                  </InputAdornment>
                ),
              }}
            />

            {operation === 'borrow' && userPosition && (
              <div className={`p-3 rounded-lg mb-4 ${isDark ? 'bg-slate-700' : 'bg-yellow-50'}`}>
                <Typography variant="caption" className={isDark ? 'text-slate-400' : 'text-gray-500'}>
                  Max borrow: {formatUSD(userPosition.collateral_usd * (selectedMarket?.ltv || 0))}
                </Typography>
              </div>
            )}

            <div className="flex gap-2">
              <Button 
                variant="contained"
                color={operation === 'supply' ? 'success' : 'error'}
                onClick={operation === 'supply' ? handleSupply : handleBorrow}
                disabled={isLoading || !amount}
                fullWidth
              >
                {isLoading ? <CircularProgress size={24} /> : (operation === 'supply' ? 'Supply' : 'Borrow')}
              </Button>
              <Button 
                variant="outlined"
                onClick={() => setDialogOpen(false)}
                fullWidth
              >
                Cancel
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
