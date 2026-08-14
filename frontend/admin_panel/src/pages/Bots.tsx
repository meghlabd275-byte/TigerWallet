import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Grid, Card, CardContent, Button, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, Chip, IconButton,
  Alert, CircularProgress, Paper, Avatar, Tooltip
} from '@mui/material';
import { PlayArrow, Stop, Refresh, Speed, TrendingUp, TrendingDown, AccountBalanceWallet, ShowChart } from '@mui/icons-material';
import { adminFetch } from '../api';

// Strategy catalog (reference config, not user data) — matches bots_service
// which only accepts grid/dca/arbitrage.
const STRATEGIES: Record<string, string> = {
  grid: 'Grid',
  dca: 'DCA',
  arbitrage: 'Arbitrage',
};

interface Bot {
  id: string;
  user_id: string;
  name: string;
  strategy: string;
  pair: string;
  params: string;
  status: string;
  created_at: number;
  trades: number;
  pnl: string;
  volume: string;
}

const Bots: React.FC = () => {
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionId, setActionId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await adminFetch<{ bots: Bot[]; count: number }>('/api/v1/admin/bots');
      setBots(data?.bots || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load bots');
      setBots([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load() }, [load]);

  const botAction = async (botId: string, action: 'start' | 'stop') => {
    setActionId(botId);
    try {
      await adminFetch(`/api/v1/bots/${botId}/${action}`, { method: 'POST' });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : `${action} failed`);
    } finally {
      setActionId(null);
    }
  };

  const totalPnl = bots.reduce((sum, b) => sum + (parseFloat(b.pnl) || 0), 0);
  const totalVolume = bots.reduce((sum, b) => sum + (parseFloat(b.volume) || 0), 0);
  const totalOrders = bots.reduce((sum, b) => sum + (b.trades || 0), 0);
  const runningBots = bots.filter(b => b.status === 'running').length;

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running': return 'success';
      case 'error': return 'error';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 'bold' }}>Bot Management</Typography>
          <Typography variant="body2" color="text.secondary">
            Monitor all trading bots across users (grid / DCA / arbitrage strategies)
          </Typography>
        </Box>
        <Button variant="outlined" startIcon={<Refresh />} onClick={load} disabled={loading}>Refresh</Button>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>{error}</Alert>}

      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Avatar sx={{ bgcolor: 'primary.main', mr: 2 }}><Speed /></Avatar>
              <Box>
                <Typography variant="body2" color="text.secondary">Running Bots</Typography>
                <Typography variant="h4">{runningBots}/{bots.length}</Typography>
              </Box>
            </Box>
          </CardContent></Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Avatar sx={{ bgcolor: totalPnl >= 0 ? 'success.main' : 'error.main', mr: 2 }}>
                {totalPnl >= 0 ? <TrendingUp /> : <TrendingDown />}
              </Avatar>
              <Box>
                <Typography variant="body2" color="text.secondary">Total PnL</Typography>
                <Typography variant="h4" sx={{ color: totalPnl >= 0 ? 'success.main' : 'error.main' }}>
                  {totalPnl.toLocaleString(undefined, { maximumFractionDigits: 2 })}
                </Typography>
              </Box>
            </Box>
          </CardContent></Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Avatar sx={{ bgcolor: 'info.main', mr: 2 }}><AccountBalanceWallet /></Avatar>
              <Box>
                <Typography variant="body2" color="text.secondary">Total Volume</Typography>
                <Typography variant="h4">{totalVolume.toLocaleString(undefined, { maximumFractionDigits: 2 })}</Typography>
              </Box>
            </Box>
          </CardContent></Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card><CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Avatar sx={{ bgcolor: 'warning.main', mr: 2 }}><ShowChart /></Avatar>
              <Box>
                <Typography variant="body2" color="text.secondary">Total Orders</Typography>
                <Typography variant="h4">{totalOrders}</Typography>
              </Box>
            </Box>
          </CardContent></Card>
        </Grid>
      </Grid>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow sx={{ bgcolor: 'grey[100]' }}>
              <TableCell><strong>Bot</strong></TableCell>
              <TableCell><strong>Owner</strong></TableCell>
              <TableCell><strong>Strategy</strong></TableCell>
              <TableCell><strong>Pair</strong></TableCell>
              <TableCell><strong>Status</strong></TableCell>
              <TableCell><strong>PnL</strong></TableCell>
              <TableCell><strong>Volume</strong></TableCell>
              <TableCell><strong>Orders</strong></TableCell>
              <TableCell><strong>Actions</strong></TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow><TableCell colSpan={9} align="center"><CircularProgress size={28} sx={{ my: 4 }} /></TableCell></TableRow>
            ) : bots.length === 0 ? (
              <TableRow><TableCell colSpan={9} align="center"><Typography sx={{ py: 4 }} color="text.secondary">No bots configured.</Typography></TableCell></TableRow>
            ) : bots.map((bot) => {
              const pnl = parseFloat(bot.pnl) || 0;
              return (
                <TableRow key={bot.id} hover>
                  <TableCell>
                    <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>{bot.name}</Typography>
                    <Typography variant="caption" color="text.secondary">ID: {bot.id.slice(0, 10)}…</Typography>
                  </TableCell>
                  <TableCell><Typography variant="caption">{bot.user_id.slice(0, 10)}…</Typography></TableCell>
                  <TableCell><Chip label={STRATEGIES[bot.strategy] || bot.strategy} size="small" variant="outlined" /></TableCell>
                  <TableCell>{bot.pair}</TableCell>
                  <TableCell><Chip label={bot.status.toUpperCase()} color={getStatusColor(bot.status)} size="small" /></TableCell>
                  <TableCell>
                    <Typography sx={{ color: pnl >= 0 ? 'success.main' : 'error.main', fontWeight: 'bold' }}>
                      {pnl.toLocaleString(undefined, { maximumFractionDigits: 2 })}
                    </Typography>
                  </TableCell>
                  <TableCell>{(parseFloat(bot.volume) || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}</TableCell>
                  <TableCell>{bot.trades}</TableCell>
                  <TableCell>
                    {bot.status === 'running' ? (
                      <Tooltip title="Stop"><span>
                        <IconButton color="error" size="small" disabled={actionId === bot.id} onClick={() => botAction(bot.id, 'stop')}><Stop /></IconButton>
                      </span></Tooltip>
                    ) : (
                      <Tooltip title="Start"><span>
                        <IconButton color="success" size="small" disabled={actionId === bot.id} onClick={() => botAction(bot.id, 'start')}><PlayArrow /></IconButton>
                      </span></Tooltip>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default Bots;
