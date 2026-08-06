import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Switch, FormControlLabel
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, AttachMoney
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface FeeRule {
  id: number;
  name: string;
  type: string;
  token: string;
  chain: string;
  fee_type: string;
  fee_value: number;
  min_amount: number;
  max_amount: number;
  is_active: boolean;
}

interface FeesProps {
  darkMode?: boolean;
}

const Fees: React.FC<FeesProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newFee, setNewFee] = useState({
    name: '',
    type: 'transaction',
    token: 'all',
    chain: 'all',
    fee_type: 'percentage',
    fee_value: 0,
    min_amount: 0,
    max_amount: 0,
    is_active: true,
  });

  // Fetch fee rules
  const { data: feesData, isLoading, refetch } = useQuery({
    queryKey: ['fees'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/fees');
      if (!response.ok) throw new Error('Failed to fetch fees');
      return response.json();
    },
  });

  // Create fee mutation
  const createMutation = useMutation({
    mutationFn: async (fee: typeof newFee) => {
      const response = await fetch('/api/v1/admin/fees', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(fee),
      });
      if (!response.ok) throw new Error('Failed to create fee');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fees'] });
      setCreateDialogOpen(false);
      setNewFee({
        name: '', type: 'transaction', token: 'all', chain: 'all',
        fee_type: 'percentage', fee_value: 0, min_amount: 0, max_amount: 0, is_active: true
      });
      setSnackbar({ open: true, message: 'Fee rule created!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to create fee', severity: 'error' });
    },
  });

  // Toggle fee mutation
  const toggleMutation = useMutation({
    mutationFn: async (fee: FeeRule) => {
      const response = await fetch(`/api/v1/admin/fees/${fee.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: !fee.is_active }),
      });
      if (!response.ok) throw new Error('Failed to update fee');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fees'] });
    },
  });

  // Delete fee mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/v1/admin/fees/${id}`, { method: 'DELETE' });
      if (!response.ok) throw new Error('Failed to delete fee');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fees'] });
      setSnackbar({ open: true, message: 'Fee rule deleted!', severity: 'success' });
    },
  });

  const fees = feesData?.data || [];
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'percentage': return 'primary';
      case 'fixed': return 'secondary';
      case 'tiered': return 'info';
      default: return 'default';
    }
  };

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
            Fee Management
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button 
              variant="contained" 
              startIcon={<Add />}
              onClick={() => setCreateDialogOpen(true)}
            >
              Add Fee Rule
            </Button>
            <IconButton onClick={() => refetch()}>
              <Refresh />
            </IconButton>
          </Box>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Paper sx={{ p: 2, bgcolor: cardBg }}>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total Rules</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>{fees.length}</Typography>
            </Paper>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Paper sx={{ p: 2, bgcolor: cardBg }}>
              <Typography variant="body2" sx={{ color: textSecondary }}>Active Rules</Typography>
              <Typography variant="h4" sx={{ color: 'success.main' }}>
                {fees.filter((f: FeeRule) => f.is_active).length}
              </Typography>
            </Paper>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Paper sx={{ p: 2, bgcolor: cardBg }}>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total Fees Collected</Typography>
              <Typography variant="h4" sx={{ color: '#f97316' }}>$124,567.89</Typography>
            </Paper>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Paper sx={{ p: 2, bgcolor: cardBg }}>
              <Typography variant="body2" sx={{ color: textSecondary }}>Avg Fee Rate</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>0.25%</Typography>
            </Paper>
          </Grid>
        </Grid>

        {/* Fees Table */}
        <Paper sx={{ bgcolor: cardBg }}>
          {isLoading ? (
            <LinearProgress />
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Name</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Type</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Token</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Chain</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Fee Type</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Value</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Range</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {fees.map((fee: FeeRule) => (
                    <TableRow key={fee.id} hover>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>{fee.name}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>{fee.type}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        <Chip label={fee.token} size="small" />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        <Chip label={fee.chain} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Chip label={fee.fee_type} size="small" color={getTypeColor(fee.fee_type)} />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>
                        {fee.fee_type === 'percentage' ? `${fee.fee_value}%` : `$${fee.fee_value}`}
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        ${fee.min_amount} - ${fee.max_amount}
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={fee.is_active ? 'Active' : 'Inactive'} 
                          size="small" 
                          color={fee.is_active ? 'success' : 'default'} 
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton size="small" onClick={() => toggleMutation.mutate(fee)}>
                          <Switch checked={fee.is_active} size="small" />
                        </IconButton>
                        <IconButton size="small" onClick={() => deleteMutation.mutate(fee.id)}>
                          <Delete />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        {/* Create Dialog */}
        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Add Fee Rule</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <TextField fullWidth label="Rule Name" value={newFee.name}
                  onChange={(e) => setNewFee({...newFee, name: e.target.value})} />
              </Grid>
              <Grid item xs={6}>
                <TextField fullWidth label="Fee Value" type="number" value={newFee.fee_value}
                  onChange={(e) => setNewFee({...newFee, fee_value: parseFloat(e.target.value)})} />
              </Grid>
              <Grid item xs={6}>
                <TextField fullWidth label="Fee Type" select value={newFee.fee_type}
                  onChange={(e) => setNewFee({...newFee, fee_type: e.target.value})}
                  SelectProps={{ native: true }}>
                  <option value="percentage">Percentage</option>
                  <option value="fixed">Fixed</option>
                </TextField>
              </Grid>
              <Grid item xs={6}>
                <TextField fullWidth label="Min Amount" type="number" value={newFee.min_amount}
                  onChange={(e) => setNewFee({...newFee, min_amount: parseFloat(e.target.value)})} />
              </Grid>
              <Grid item xs={6}>
                <TextField fullWidth label="Max Amount" type="number" value={newFee.max_amount}
                  onChange={(e) => setNewFee({...newFee, max_amount: parseFloat(e.target.value)})} />
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newFee)} disabled={!newFee.name}>
              Create
            </Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Fees;
