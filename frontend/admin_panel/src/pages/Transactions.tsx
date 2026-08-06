import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Select, MenuItem, FormControl, InputLabel, Card, CardContent,
  IconButton, Snackbar, Alert, LinearProgress, InputAdornment
} from '@mui/material';
import { 
  Search, Refresh, FilterList, Download, Visibility, CheckCircle, Cancel
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Transaction {
  id: number;
  tx_hash: string;
  user_id: number;
  type: string;
  amount: number;
  fee: number;
  token: string;
  chain: string;
  status: string;
  created_at: string;
}

interface TransactionsProps {
  darkMode?: boolean;
}

const Transactions: React.FC<TransactionsProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');
  const [filterChain, setFilterChain] = useState('all');
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });

  // Fetch transactions
  const { data: transactionsData, isLoading, refetch } = useQuery({
    queryKey: ['transactions', filterStatus, filterChain],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (filterStatus !== 'all') params.append('status', filterStatus);
      if (filterChain !== 'all') params.append('chain', filterChain);
      const response = await fetch(`/api/v1/admin/transactions?${params}`);
      if (!response.ok) throw new Error('Failed to fetch transactions');
      return response.json();
    },
  });

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['transactionStats'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/transactions/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      return response.json();
    },
  });

  const transactions = transactionsData?.data || [];
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'pending': return 'warning';
      case 'failed': return 'error';
      case 'processing': return 'info';
      default: return 'default';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'swap': return 'primary';
      case 'transfer': return 'secondary';
      case 'stake': return 'info';
      case 'unstake': return 'warning';
      default: return 'default';
    }
  };

  const filteredTransactions = transactions.filter((tx: Transaction) => 
    searchQuery === '' || 
    tx.tx_hash?.includes(searchQuery) ||
    tx.token?.toLowerCase().includes(searchQuery.toLowerCase())
  );

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
            Transactions
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button 
              variant="outlined" 
              startIcon={<Download />}
            >
              Export
            </Button>
            <IconButton onClick={() => refetch()}>
              <Refresh />
            </IconButton>
          </Box>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Total Transactions</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats?.total_transactions?.toLocaleString() || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Volume (24h)</Typography>
                <Typography variant="h4" sx={{ color: '#f97316' }}>${(stats?.today_volume || 0).toLocaleString()}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Pending</Typography>
                <Typography variant="h4" sx={{ color: 'warning.main' }}>{stats?.pending_count || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Failed</Typography>
                <Typography variant="h4" sx={{ color: 'error.main' }}>{stats?.failed_count || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Filters */}
        <Paper sx={{ p: 2, mb: 2, bgcolor: cardBg }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                size="small"
                placeholder="Search by hash, token..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: <InputAdornment position="start"><Search /></InputAdornment>,
                }}
              />
            </Grid>
            <Grid item xs={6} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>Status</InputLabel>
                <Select
                  value={filterStatus}
                  label="Status"
                  onChange={(e) => setFilterStatus(e.target.value)}
                >
                  <MenuItem value="all">All</MenuItem>
                  <MenuItem value="completed">Completed</MenuItem>
                  <MenuItem value="pending">Pending</MenuItem>
                  <MenuItem value="failed">Failed</MenuItem>
                  <MenuItem value="processing">Processing</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={6} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>Chain</InputLabel>
                <Select
                  value={filterChain}
                  label="Chain"
                  onChange={(e) => setFilterChain(e.target.value)}
                >
                  <MenuItem value="all">All Chains</MenuItem>
                  <MenuItem value="ethereum">Ethereum</MenuItem>
                  <MenuItem value="bsc">BSC</MenuItem>
                  <MenuItem value="polygon">Polygon</MenuItem>
                  <MenuItem value="arbitrum">Arbitrum</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </Paper>

        {/* Transactions Table */}
        <Paper sx={{ bgcolor: cardBg }}>
          {isLoading ? (
            <LinearProgress />
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Tx Hash</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Type</TableCell>
                    <TableCell sx={{ color: textSecondary }}>User</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Amount</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Fee</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Chain</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Date</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredTransactions.slice(0, 50).map((tx: Transaction) => (
                    <TableRow key={tx.id} hover>
                      <TableCell sx={{ color: textPrimary }}>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', maxWidth: 120 }} noWrap>
                          {tx.tx_hash?.substring(0, 10)}...{tx.tx_hash?.substring(tx.tx_hash.length - 8)}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={tx.type} size="small" color={getTypeColor(tx.type)} />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>{tx.user_id}</TableCell>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>
                        {tx.amount?.toFixed(4)} {tx.token}
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        {tx.fee?.toFixed(6)} {tx.token}
                      </TableCell>
                      <TableCell>
                        <Chip label={tx.chain} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={tx.status} 
                          size="small" 
                          color={getStatusColor(tx.status)} 
                        />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        {new Date(tx.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <IconButton size="small">
                          <Visibility />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        {filteredTransactions.length === 0 && !isLoading && (
          <Paper sx={{ p: 4, textAlign: 'center', bgcolor: cardBg }}>
            <Typography variant="h6" sx={{ color: textSecondary }}>
              No transactions found
            </Typography>
          </Paper>
        )}

        {/* Snackbar */}
        <Snackbar
          open={snackbar.open}
          autoHideDuration={6000}
          onClose={() => setSnackbar({ ...snackbar, open: false })}
        >
          <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
            {snackbar.message}
          </Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Transactions;
