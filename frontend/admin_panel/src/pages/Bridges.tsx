import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Card, CardContent
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, Hub
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Bridge {
  id: number;
  name: string;
  source_chain: string;
  dest_chain: string;
  token: string;
  fee_percent: number;
  min_amount: number;
  max_amount: number;
  is_active: boolean;
  volume_24h: number;
}

interface BridgesProps {
  darkMode?: boolean;
}

const Bridges: React.FC<BridgesProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newBridge, setNewBridge] = useState({
    name: '', source_chain: '', dest_chain: '', token: '', fee_percent: 0, min_amount: 0, max_amount: 0, is_active: true
  });

  const { data: bridgesData, isLoading, refetch } = useQuery({
    queryKey: ['bridges'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/bridges');
      if (!response.ok) throw new Error('Failed to fetch bridges');
      return response.json();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (bridge: typeof newBridge) => {
      const response = await fetch('/api/v1/admin/bridges', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(bridge)
      });
      if (!response.ok) throw new Error('Failed to create bridge');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bridges'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: 'Bridge added!', severity: 'success' });
    },
  });

  const bridges = bridgesData?.data || [];
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Cross-Chain Bridges</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateDialogOpen(true)}>Add Bridge</Button>
            <IconButton onClick={() => refetch()}><Refresh /></IconButton>
          </Box>
        </Box>

        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total Bridges</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>{bridges.length}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>24h Volume</Typography>
              <Typography variant="h4" sx={{ color: '#f97316' }}>$1.2M</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Supported Chains</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>8</Typography>
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
                    <TableCell sx={{ color: textSecondary }}>Source</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Destination</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Token</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Fee %</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Range</TableCell>
                    <TableCell sx={{ color: textSecondary }}>24h Volume</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {bridges.map((bridge: Bridge) => (
                    <TableRow key={bridge.id}>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>{bridge.name}</TableCell>
                      <TableCell><Chip label={bridge.source_chain} size="small" /></TableCell>
                      <TableCell><Chip label={bridge.dest_chain} size="small" variant="outlined" /></TableCell>
                      <TableCell sx={{ color: textPrimary }}>{bridge.token}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>{bridge.fee_percent}%</TableCell>
                      <TableCell sx={{ color: textPrimary }}>${bridge.min_amount} - ${bridge.max_amount}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>${bridge.volume_24h?.toLocaleString()}</TableCell>
                      <TableCell><Chip label={bridge.is_active ? 'Active' : 'Inactive'} size="small" color={bridge.is_active ? 'success' : 'default'} /></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Add Bridge</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}><TextField fullWidth label="Bridge Name" value={newBridge.name} onChange={(e) => setNewBridge({...newBridge, name: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Source Chain" value={newBridge.source_chain} onChange={(e) => setNewBridge({...newBridge, source_chain: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Destination Chain" value={newBridge.dest_chain} onChange={(e) => setNewBridge({...newBridge, dest_chain: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Token" value={newBridge.token} onChange={(e) => setNewBridge({...newBridge, token: e.target.value})} /></Grid>
              <Grid item xs={6}><TextField fullWidth label="Fee %" type="number" value={newBridge.fee_percent} onChange={(e) => setNewBridge({...newBridge, fee_percent: parseFloat(e.target.value)})} /></Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newBridge)}>Create</Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Bridges;
