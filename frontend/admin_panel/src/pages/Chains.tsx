import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Switch, Card, CardContent
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, AccountTree
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Chain {
  id: number;
  name: string;
  symbol: string;
  chain_id: string;
  rpc_url: string;
  explorer_url: string;
  is_active: boolean;
  tx_count: number;
}

interface ChainsProps {
  darkMode?: boolean;
}

const Chains: React.FC<ChainsProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newChain, setNewChain] = useState({
    name: '', symbol: '', chain_id: '', rpc_url: '', explorer_url: '', is_active: true
  });

  const { data: chainsData, isLoading, refetch } = useQuery({
    queryKey: ['chains'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/chains');
      if (!response.ok) throw new Error('Failed to fetch chains');
      return response.json();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (chain: typeof newChain) => {
      const response = await fetch('/api/v1/admin/chains', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(chain)
      });
      if (!response.ok) throw new Error('Failed to create chain');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chains'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: 'Chain added!', severity: 'success' });
    },
  });

  const chains = chainsData?.data || [];
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Blockchain Networks</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateDialogOpen(true)}>Add Chain</Button>
            <IconButton onClick={() => refetch()}><Refresh /></IconButton>
          </Box>
        </Box>

        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total Chains</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>{chains.length}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Active Chains</Typography>
              <Typography variant="h4" sx={{ color: 'success.main' }}>{chains.filter((c: Chain) => c.is_active).length}</Typography>
            </CardContent></Card>
          </Grid>
        </Grid>

        <Paper sx={{ bgcolor: cardBg }}>
          {isLoading ? <LinearProgress /> : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Name</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Symbol</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Chain ID</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Transactions</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {chains.map((chain: Chain) => (
                    <TableRow key={chain.id}>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>{chain.name}</TableCell>
                      <TableCell><Chip label={chain.symbol} size="small" /></TableCell>
                      <TableCell sx={{ color: textPrimary }}>{chain.chain_id}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>{chain.tx_count?.toLocaleString()}</TableCell>
                      <TableCell><Chip label={chain.is_active ? 'Active' : 'Inactive'} size="small" color={chain.is_active ? 'success' : 'default'} /></TableCell>
                      <TableCell>
                        <IconButton size="small"><Edit /></IconButton>
                        <IconButton size="small"><Delete /></IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Add Blockchain</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={6}><TextField fullWidth label="Name" value={newChain.name} onChange={(e) => setNewChain({...newChain, name: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Symbol" value={newChain.symbol} onChange={(e) => setNewChain({...newChain, symbol: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="Chain ID" value={newChain.chain_id} onChange={(e) => setNewChain({...newChain, chain_id: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="RPC URL" value={newChain.rpc_url} onChange={(e) => setNewChain({...newChain, rpc_url: e.target.value})} /></Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newChain)}>Create</Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Chains;
