import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Switch, Card, CardContent
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, Hub
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface DEX {
  id: number;
  name: string;
  router_address: string;
  factory_address: string;
  tokens: string[];
  chains: string[];
  is_active: boolean;
  volume_24h: number;
}

interface DEXsProps {
  darkMode?: boolean;
}

const DEXs: React.FC<DEXsProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newDEX, setNewDEX] = useState({
    name: '', router_address: '', factory_address: '', tokens: '', chains: '', is_active: true
  });

  const { data: dexData, isLoading, refetch } = useQuery({
    queryKey: ['dexs'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/dexs');
      if (!response.ok) throw new Error('Failed to fetch DEXes');
      return response.json();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (dex: typeof newDEX) => {
      const response = await fetch('/api/v1/admin/dexs', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(dex)
      });
      if (!response.ok) throw new Error('Failed to create DEX');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dexs'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: 'DEX added!', severity: 'success' });
    },
  });

  const dexes = dexData?.data || [];
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Decentralized Exchanges</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateDialogOpen(true)}>Add DEX</Button>
            <IconButton onClick={() => refetch()}><Refresh /></IconButton>
          </Box>
        </Box>

        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total DEXes</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>{dexes.length}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>24h Volume</Typography>
              <Typography variant="h4" sx={{ color: '#f97316' }}>$2.4M</Typography>
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
                    <TableCell sx={{ color: textSecondary }}>Router</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Factory</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Chains</TableCell>
                    <TableCell sx={{ color: textSecondary }}>24h Volume</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {dexes.map((dex: DEX) => (
                    <TableRow key={dex.id}>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>{dex.name}</TableCell>
                      <TableCell sx={{ color: textPrimary, fontFamily: 'monospace', fontSize: 12 }}>
                        {dex.router_address?.substring(0, 10)}...
                      </TableCell>
                      <TableCell sx={{ color: textPrimary, fontFamily: 'monospace', fontSize: 12 }}>
                        {dex.factory_address?.substring(0, 10)}...
                      </TableCell>
                      <TableCell>{dex.chains?.map((c: string) => <Chip key={c} label={c} size="small" sx={{ mr: 0.5 }} />)}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>${dex.volume_24h?.toLocaleString()}</TableCell>
                      <TableCell><Chip label={dex.is_active ? 'Active' : 'Inactive'} size="small" color={dex.is_active ? 'success' : 'default'} /></TableCell>
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
          <DialogTitle sx={{ bgcolor: cardBg }}>Add DEX</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}><TextField fullWidth label="Name" value={newDEX.name} onChange={(e) => setNewDEX({...newDEX, name: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="Router Address" value={newDEX.router_address} onChange={(e) => setNewDEX({...newDEX, router_address: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="Factory Address" value={newDEX.factory_address} onChange={(e) => setNewDEX({...newDEX, factory_address: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="Supported Chains (comma separated)" value={newDEX.chains} onChange={(e) => setNewDEX({...newDEX, chains: e.target.value})} /></Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newDEX)}>Create</Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default DEXs;
