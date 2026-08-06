import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Select, MenuItem, FormControl, InputLabel, Card, CardContent,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  LinearProgress
} from '@mui/material';
import { 
  TrendingUp, TrendingDown, People, Receipt, AttachMoney, ShowChart
} from '@mui/icons-material';
import { useQuery } from '@tanstack/react-query';

interface AnalyticsProps {
  darkMode?: boolean;
}

const Analytics: React.FC<AnalyticsProps> = ({ darkMode }) => {
  const [period, setPeriod] = useState('7d');

  const { data: stats, isLoading } = useQuery({
    queryKey: ['analytics', period],
    queryFn: async () => {
      const response = await fetch(`/api/v1/admin/analytics/dashboard?period=${period}`);
      if (!response.ok) throw new Error('Failed to fetch analytics');
      return response.json();
    },
  });

  const { data: volumeData } = useQuery({
    queryKey: ['volumeAnalytics', period],
    queryFn: async () => {
      const response = await fetch(`/api/v1/admin/analytics/volume?period=${period}`);
      if (!response.ok) throw new Error('Failed to fetch volume');
      return response.json();
    },
  });

  const { data: revenueData } = useQuery({
    queryKey: ['revenueAnalytics', period],
    queryFn: async () => {
      const response = await fetch(`/api/v1/admin/analytics/revenue?period=${period}`);
      if (!response.ok) throw new Error('Failed to fetch revenue');
      return response.json();
    },
  });

  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Analytics Dashboard</Typography>
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Period</InputLabel>
            <Select value={period} label="Period" onChange={(e) => setPeriod(e.target.value)}>
              <MenuItem value="24h">24 Hours</MenuItem>
              <MenuItem value="7d">7 Days</MenuItem>
              <MenuItem value="30d">30 Days</MenuItem>
              <MenuItem value="90d">90 Days</MenuItem>
            </Select>
          </FormControl>
        </Box>

        {isLoading ? <LinearProgress /> : (
          <>
            {/* Key Metrics */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
              <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ bgcolor: cardBg }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <People sx={{ color: '#f97316', fontSize: 32 }} />
                      <Box>
                        <Typography variant="body2" sx={{ color: textSecondary }}>Total Users</Typography>
                        <Typography variant="h5" sx={{ color: textPrimary, fontWeight: 'bold' }}>{stats?.total_users?.toLocaleString() || 0}</Typography>
                        <Typography variant="caption" sx={{ color: 'success.main' }}>+{stats?.new_users_today || 0} today</Typography>
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ bgcolor: cardBg }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Receipt sx={{ color: '#f97316', fontSize: 32 }} />
                      <Box>
                        <Typography variant="body2" sx={{ color: textSecondary }}>Transactions</Typography>
                        <Typography variant="h5" sx={{ color: textPrimary, fontWeight: 'bold' }}>{stats?.total_transactions?.toLocaleString() || 0}</Typography>
                        <Typography variant="caption" sx={{ color: 'success.main' }}>+{stats?.today_transactions || 0} today</Typography>
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
                        <Typography variant="body2" sx={{ color: textSecondary }}>Total Volume</Typography>
                        <Typography variant="h5" sx={{ color: textPrimary, fontWeight: 'bold' }}>${(stats?.total_volume || 0).toLocaleString()}</Typography>
                        <Typography variant="caption" sx={{ color: 'success.main' }}>+${(stats?.today_volume || 0).toLocaleString()} today</Typography>
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ bgcolor: cardBg }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <AttachMoney sx={{ color: 'success.main', fontSize: 32 }} />
                      <Box>
                        <Typography variant="body2" sx={{ color: textSecondary }}>Revenue</Typography>
                        <Typography variant="h5" sx={{ color: 'success.main', fontWeight: 'bold' }}>${(stats?.total_revenue || 0).toLocaleString()}</Typography>
                        <Typography variant="caption" sx={{ color: 'success.main' }}>+${(stats?.today_revenue || 0).toLocaleString()} today</Typography>
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>

            {/* Volume by Chain */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
              <Grid item xs={12} md={6}>
                <Paper sx={{ bgcolor: cardBg, p: 2 }}>
                  <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Volume by Chain</Typography>
                  {volumeData?.chain_volumes?.map((chain: { chain: string; volume: number; count: number }) => (
                    <Box key={chain.chain} sx={{ display: 'flex', justifyContent: 'space-between', py: 1, borderBottom: `1px solid ${darkMode ? '#333' : '#eee'}` }}>
                      <Chip label={chain.chain} size="small" />
                      <Typography sx={{ color: textPrimary, fontWeight: 'bold' }}>${chain.volume?.toLocaleString()}</Typography>
                    </Box>
                  ))}
                </Paper>
              </Grid>
              <Grid item xs={12} md={6}>
                <Paper sx={{ bgcolor: cardBg, p: 2 }}>
                  <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Revenue by Type</Typography>
                  {revenueData?.revenue_by_type?.map((item: { type: string; revenue: number }) => (
                    <Box key={item.type} sx={{ display: 'flex', justifyContent: 'space-between', py: 1, borderBottom: `1px solid ${darkMode ? '#333' : '#eee'}` }}>
                      <Typography sx={{ color: textPrimary, textTransform: 'capitalize' }}>{item.type}</Typography>
                      <Typography sx={{ color: 'success.main', fontWeight: 'bold' }}>${item.revenue?.toLocaleString()}</Typography>
                    </Box>
                  ))}
                </Paper>
              </Grid>
            </Grid>

            {/* Top Tokens */}
            <Paper sx={{ bgcolor: cardBg, p: 2 }}>
              <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Top Tokens by Volume</Typography>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: textSecondary }}>Token</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Volume</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Transactions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {volumeData?.token_volumes?.slice(0, 10).map((token: { token: string; volume: number; count: number }) => (
                      <TableRow key={token.token}>
                        <TableCell><Chip label={token.token} size="small" /></TableCell>
                        <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>${token.volume?.toLocaleString()}</TableCell>
                        <TableCell sx={{ color: textPrimary }}>{token.count?.toLocaleString()}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Paper>
          </>
        )}
      </Container>
    </Box>
  );
};

export default Analytics;
