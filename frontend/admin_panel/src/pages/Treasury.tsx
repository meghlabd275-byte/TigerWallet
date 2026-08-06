import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Card, CardContent
} from '@mui/material';
import { 
  Add, Refresh, AccountBalanceWallet, TrendingUp, TrendingDown
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface TreasuryAsset {
  id: number;
  token: string;
  chain: string;
  balance: number;
  value_usd: number;
  wallet_type: string;
}

interface TreasuryProps {
  darkMode?: boolean;
}

const Treasury: React.FC<TreasuryProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });

  // Fetch treasury data
  const { data: treasuryData, isLoading, refetch } = useQuery({
    queryKey: ['treasury'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/treasury');
      if (!response.ok) throw new Error('Failed to fetch treasury');
      return response.json();
    },
  });

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['treasuryStats'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/treasury/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      return response.json();
    },
  });

  const assets = treasuryData?.data || [];
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ 
      minHeight: '100vh',
      bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5',
      color: textPrimary,
      transition: 'all 0.3s ease'
    }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>
            Treasury Management
          </Typography>
          <IconButton onClick={() => refetch()}>
            <Refresh />
          </IconButton>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <AccountBalanceWallet sx={{ color: '#f97316', fontSize: 32 }} />
                  <Box>
                    <Typography variant="body2" sx={{ color: textSecondary }}>Total Balance</Typography>
                    <Typography variant="h5" sx={{ color: textPrimary, fontWeight: 'bold' }}>
                      ${(stats?.total_balance || 0).toLocaleString()}
                    </Typography>
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <TrendingUp sx={{ color: 'success.main', fontSize: 32 }} />
                  <Box>
                    <Typography variant="body2" sx={{ color: textSecondary }}>24h Inflow</Typography>
                    <Typography variant="h5" sx={{ color: 'success.main', fontWeight: 'bold' }}>
                      +${(stats?.inflow_24h || 0).toLocaleString()}
                    </Typography>
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <TrendingDown sx={{ color: 'error.main', fontSize: 32 }} />
                  <Box>
                    <Typography variant="body2" sx={{ color: textSecondary }}>24h Outflow</Typography>
                    <Typography variant="h5" sx={{ color: 'error.main', fontWeight: 'bold' }}>
                      -${(stats?.outflow_24h || 0).toLocaleString()}
                    </Typography>
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <AccountBalanceWallet sx={{ color: 'info.main', fontSize: 32 }} />
                  <Box>
                    <Typography variant="body2" sx={{ color: textSecondary }}>Wallet Count</Typography>
                    <Typography variant="h5" sx={{ color: textPrimary, fontWeight: 'bold' }}>
                      {stats?.wallet_count || 0}
                    </Typography>
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Wallet Breakdown */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} md={8}>
            <Paper sx={{ bgcolor: cardBg, p: 2 }}>
              <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Wallet Breakdown</Typography>
              {isLoading ? (
                <LinearProgress />
              ) : (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ color: textSecondary }}>Token</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Chain</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Type</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Balance</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Value (USD)</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {assets.map((asset: TreasuryAsset) => (
                        <TableRow key={asset.id}>
                          <TableCell><Chip label={asset.token} size="small" /></TableCell>
                          <TableCell sx={{ color: textPrimary }}>{asset.chain}</TableCell>
                          <TableCell sx={{ color: textPrimary }}>{asset.wallet_type}</TableCell>
                          <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>
                            {asset.balance?.toFixed(4)}
                          </TableCell>
                          <TableCell sx={{ color: textPrimary }}>
                            ${asset.value_usd?.toLocaleString()}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </Paper>
          </Grid>
          <Grid item xs={12} md={4}>
            <Paper sx={{ bgcolor: cardBg, p: 2 }}>
              <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Quick Actions</Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                <Button variant="outlined" fullWidth>Distribute Funds</Button>
                <Button variant="outlined" fullWidth>Top Up Wallets</Button>
                <Button variant="outlined" fullWidth>Generate Report</Button>
                <Button variant="outlined" fullWidth>View Transactions</Button>
              </Box>
            </Paper>
          </Grid>
        </Grid>

        {/* Distribution Chart Placeholder */}
        <Paper sx={{ bgcolor: cardBg, p: 2 }}>
          <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Asset Distribution</Typography>
          <Box sx={{ 
            height: 200, 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            bgcolor: darkMode ? '#222' : '#f5f5f5',
            borderRadius: 1
          }}>
            <Typography sx={{ color: textSecondary }}>
              Chart visualization - BTC: 45%, ETH: 30%, USDT: 15%, Other: 10%
            </Typography>
          </Box>
        </Paper>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Treasury;
