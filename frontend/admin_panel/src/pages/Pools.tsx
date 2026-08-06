import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Card, CardContent
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, Pool
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Pool {
  id: number;
  name: string;
  token_a: string;
  token_b: string;
  chain: string;
  tvl: number;
  volume_24h: number;
  apy: number;
  is_active: boolean;
}

interface PoolsProps {
  darkMode?: boolean;
}

const Pools: React.FC<PoolsProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newPool, setNewPool] = useState({
    name: '', token_a: '', token_b: '', chain: '', is_active: true
  });

  const { data: poolsData, isLoading, refetch } = useQuery({
    queryKey: ['pools'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/pools');
      if (!response.ok) throw new Error('Failed to fetch pools');
      return response.json();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (pool: typeof newPool) => {
      const response = await fetch('/api/v1/admin/pools', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(pool)
      });
      if (!response.ok) throw new Error('Failed to create pool');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pools'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: 'Pool added!', severity: 'success' });
    },
  });

  const pools = poolsData?.data || [];
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Liquidity Pools</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateDialogOpen(true)}>Add Pool</Button>
            <IconButton onClick={() => refetch()}><Refresh /></IconButton>
          </Box>
        </Box>

        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total Pools</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>{pools.length}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total TVL</Typography>
              <Typography variant="h4" sx={{ color: '#f97316' }}>$5.2M</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>24h Volume</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>$1.8M</Typography>
            </CardContent></Card>
          </Grid>
        </Grid>

        <Paper sx={{ bgcolor: cardBg }}>
          {isLoading ? <LinearProgress /> : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Pool Name</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Token A</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Token B</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Chain</TableCell>
                    <TableCell sx={{ color: textSecondary }}>TVL</TableCell>
                    <TableCell sx={{ color: textSecondary }}>24h Volume</TableCell>
                    <TableCell sx={{ color: textSecondary }}>APY</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {pools.map((pool: Pool) => (
                    <TableRow key={pool.id}>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>{pool.name}</TableCell>
                      <TableCell><Chip label={pool.token_a} size="small" /></TableCell>
                      <TableCell><Chip label={pool.token_b} size="small" /></TableCell>
                      <TableCell sx={{ color: textPrimary }}>{pool.chain}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>${pool.tvl?.toLocaleString()}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>${pool.volume_24h?.toLocaleString()}</TableCell>
                      <TableCell sx={{ color: 'success.main', fontWeight: 'bold' }}>{pool.apy?.toFixed(2)}%</TableCell>
                      <TableCell><Chip label={pool.is_active ? 'Active' : 'Inactive'} size="small" color={pool.is_active ? 'success' : 'default'} /></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Add Pool</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}><TextField fullWidth label="Pool Name" value={newPool.name} onChange={(e) => setNewPool({...newPool, name: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Token A" value={newPool.token_a} onChange={(e) => setNewPool({...newPool, token_a: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Token B" value={newPool.token_b} onChange={(e) => setNewPool({...newPool, token_b: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="Chain" value={newPool.chain} onChange={(e) => setNewPool({...newPool, chain: e.target.value})} /></Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newPool)}>Create</Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Pools;
