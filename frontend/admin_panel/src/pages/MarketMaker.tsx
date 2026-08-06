import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  IconButton, Snackbar, Alert, LinearProgress, Card, CardContent, Switch
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, ShowChart
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface MarketMakerBot {
  id: number;
  name: string;
  strategy: string;
  pairs: string[];
  status: string;
  volume_24h: number;
  profit_loss: number;
  is_active: boolean;
}

interface MarketMakerProps {
  darkMode?: boolean;
}

const MarketMaker: React.FC<MarketMakerProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newBot, setNewBot] = useState({
    name: '', strategy: 'arb', pairs: '', is_active: true
  });

  const { data: botsData, isLoading, refetch } = useQuery({
    queryKey: ['marketmakers'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/market-makers');
      if (!response.ok) throw new Error('Failed to fetch bots');
      return response.json();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (bot: typeof newBot) => {
      const response = await fetch('/api/v1/admin/market-makers', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(bot)
      });
      if (!response.ok) throw new Error('Failed to create bot');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['marketmakers'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: 'Market Maker added!', severity: 'success' });
    },
  });

  const bots = botsData?.data || [];
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Market Maker Bots</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateDialogOpen(true)}>Add Bot</Button>
            <IconButton onClick={() => refetch()}><Refresh /></IconButton>
          </Box>
        </Box>

        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Active Bots</Typography>
              <Typography variant="h4" sx={{ color: 'success.main' }}>{bots.filter((b: MarketMakerBot) => b.is_active).length}</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>24h Volume</Typography>
              <Typography variant="h4" sx={{ color: '#f97316' }}>$890K</Typography>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Typography variant="body2" sx={{ color: textSecondary }}>Total P/L</Typography>
              <Typography variant="h4" sx={{ color: 'success.main' }}>+$12.5K</Typography>
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
                    <TableCell sx={{ color: textSecondary }}>Strategy</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Trading Pairs</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>24h Volume</TableCell>
                    <TableCell sx={{ color: textSecondary }}>P/L</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Active</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {bots.map((bot: MarketMakerBot) => (
                    <TableRow key={bot.id}>
                      <TableCell sx={{ color: textPrimary, fontWeight: 'bold' }}>{bot.name}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>{bot.strategy}</TableCell>
                      <TableCell>{bot.pairs?.map((p: string) => <Chip key={p} label={p} size="small" sx={{ mr: 0.5 }} />)}</TableCell>
                      <TableCell><Chip label={bot.status} size="small" color={bot.status === 'running' ? 'success' : 'warning'} /></TableCell>
                      <TableCell sx={{ color: textPrimary }}>${bot.volume_24h?.toLocaleString()}</TableCell>
                      <TableCell sx={{ color: bot.profit_loss >= 0 ? 'success.main' : 'error.main', fontWeight: 'bold' }}>
                        {bot.profit_loss >= 0 ? '+' : ''}${bot.profit_loss?.toLocaleString()}
                      </TableCell>
                      <TableCell><Switch checked={bot.is_active} size="small" /></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Add Market Maker</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}><TextField fullWidth label="Bot Name" value={newBot.name} onChange={(e) => setNewBot({...newBot, name: e.target.value})} /></Grid>
              <Grid item xs={12}>
                <TextField fullWidth label="Strategy" select value={newBot.strategy} onChange={(e) => setNewBot({...newBot, strategy: e.target.value})}
                  SelectProps={{ native: true }}>
                  <option value="arb">Arbitrage</option>
                  <option value="mm">Market Making</option>
                  <option value="liq">Liquidity</option>
                </TextField>
              </Grid>
              <Grid item xs={12}><TextField fullWidth label="Trading Pairs (comma separated)" value={newBot.pairs} onChange={(e) => setNewBot({...newBot, pairs: e.target.value})} /></Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newBot)}>Create</Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default MarketMaker;
