import React, { useState, useEffect, useCallback } from 'react';
import {
  Container, Grid, Card, Button, TextField, Select, MenuItem, FormControl, InputLabel,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TablePagination,
  Dialog, DialogTitle, DialogContent, DialogActions, Chip, Typography, Box, Avatar,
  IconButton, CircularProgress, Alert, Snackbar, InputAdornment
} from '@mui/material';
import {
  Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon, Search as SearchIcon,
  Visibility as ViewIcon, Block as BlockIcon, PlayArrow as ResumeIcon,
  Pause as SuspendIcon, Warning as HaltIcon, DarkMode, LightMode, ShowChart
} from '@mui/icons-material';
import { api, TradingPair, PaginatedResponse, Blockchain } from '../services/api';
import { useTheme } from '../../context/ThemeContext';

const TradingPairsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [pairs, setPairs] = useState<TradingPair[]>([]);
  const [blockchains, setBlockchains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPair, setSelectedPair] = useState<TradingPair | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | 'view'>('create');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [total, setTotal] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [snackbar, setSnackbar] = useState<{open: boolean; message: string; severity: 'success' | 'error'}>({
    open: false, message: '', severity: 'success'
  });

  const [formData, setFormData] = useState({
    baseToken: '',
    quoteToken: '',
    chainId: 1,
    pairAddress: '',
    fee: 0.1,
    minTrade: 0.001,
    maxTrade: 1000000
  });

  const fetchPairs = useCallback(async () => {
    try {
      setLoading(true);
      const response: PaginatedResponse<TradingPair> = await api.getTradingPairs({
        page: page + 1,
        pageSize: rowsPerPage,
        query: searchQuery,
        status: statusFilter || undefined
      });
      setPairs(response.data);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch trading pairs:', error);
      setSnackbar({ open: true, message: 'Failed to fetch trading pairs', severity: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, searchQuery, statusFilter]);

  const fetchBlockchains = useCallback(async () => {
    try {
      const data = await api.getBlockchains();
      setBlockchains(data);
    } catch (error) {
      console.error('Failed to fetch blockchains:', error);
    }
  }, []);

  useEffect(() => {
    fetchPairs();
    fetchBlockchains();
  }, [fetchPairs, fetchBlockchains]);

  const handleCreatePair = () => {
    setSelectedPair(null);
    setFormData({
      baseToken: '',
      quoteToken: '',
      chainId: 1,
      pairAddress: '',
      fee: 0.1,
      minTrade: 0.001,
      maxTrade: 1000000
    });
    setDialogMode('create');
    setOpenDialog(true);
  };

  const handleEditPair = (pair: TradingPair) => {
    setSelectedPair(pair);
    setFormData({
      baseToken: pair.baseToken,
      quoteToken: pair.quoteToken,
      chainId: pair.chainId,
      pairAddress: pair.pairAddress || '',
      fee: pair.fee,
      minTrade: pair.minTrade,
      maxTrade: pair.maxTrade
    });
    setDialogMode('edit');
    setOpenDialog(true);
  };

  const handleViewPair = (pair: TradingPair) => {
    setSelectedPair(pair);
    setDialogMode('view');
    setOpenDialog(true);
  };

  const handleDeletePair = async (pairId: string) => {
    if (!confirm('Are you sure you want to delete this trading pair?')) return;
    try {
      await api.deleteTradingPair(pairId);
      setSnackbar({ open: true, message: 'Trading pair deleted successfully', severity: 'success' });
      fetchPairs();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to delete trading pair', severity: 'error' });
    }
  };

  const handleSuspendPair = async (pairId: string) => {
    try {
      await api.suspendTradingPair(pairId);
      setSnackbar({ open: true, message: 'Trading pair suspended', severity: 'success' });
      fetchPairs();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to suspend trading pair', severity: 'error' });
    }
  };

  const handleResumePair = async (pairId: string) => {
    try {
      await api.resumeTradingPair(pairId);
      setSnackbar({ open: true, message: 'Trading pair resumed', severity: 'success' });
      fetchPairs();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to resume trading pair', severity: 'error' });
    }
  };

  const handleHaltPair = async (pairId: string) => {
    try {
      await api.haltTradingPair(pairId);
      setSnackbar({ open: true, message: 'Trading pair halted', severity: 'success' });
      fetchPairs();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to halt trading pair', severity: 'error' });
    }
  };

  const handleSubmit = async () => {
    try {
      if (dialogMode === 'create') {
        await api.createTradingPair(formData);
        setSnackbar({ open: true, message: 'Trading pair created successfully', severity: 'success' });
      } else if (dialogMode === 'edit' && selectedPair) {
        await api.updateTradingPair(selectedPair.id, formData);
        setSnackbar({ open: true, message: 'Trading pair updated successfully', severity: 'success' });
      }
      setOpenDialog(false);
      fetchPairs();
    } catch (error) {
      setSnackbar({ open: true, message: 'Operation failed', severity: 'error' });
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'success';
      case 'suspended': return 'warning';
      case 'halted': return 'error';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ 
      minHeight: '100vh', 
      bgcolor: theme === 'dark' ? 'var(--bg-primary)' : 'var(--bg-primary)',
      color: theme === 'dark' ? 'var(--text-primary)' : 'var(--text-primary)',
      transition: 'background-color 0.3s, color 0.3s'
    }}>
      <Container maxWidth="xl" sx={{ py: 4 }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Typography variant="h4" fontWeight="bold">
            Trading Pairs Management
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <IconButton onClick={toggleTheme} color="primary">
              {theme === 'dark' ? <LightMode /> : <DarkMode />}
            </IconButton>
            <Button variant="contained" startIcon={<AddIcon />} onClick={handleCreatePair}>
              Create Pair
            </Button>
          </Box>
        </Box>

        {/* Filters */}
        <Card sx={{ p: 2, mb: 3 }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                placeholder="Search pairs (e.g., BTC/ETH)..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon color="action" />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
            <Grid item xs={12} md={3}>
              <FormControl fullWidth>
                <InputLabel>Status</InputLabel>
                <Select
                  value={statusFilter}
                  label="Status"
                  onChange={(e) => setStatusFilter(e.target.value)}
                >
                  <MenuItem value="">All</MenuItem>
                  <MenuItem value="active">Active</MenuItem>
                  <MenuItem value="suspended">Suspended</MenuItem>
                  <MenuItem value="halted">Halted</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </Card>

        {/* Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="primary">{total}</Typography>
              <Typography variant="body2" color="text.secondary">Total Pairs</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="success.main">{pairs.filter(p => p.status === 'active').length}</Typography>
              <Typography variant="body2" color="text.secondary">Active</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="info.main">
                ${pairs.reduce((sum, p) => sum + p.liquidity, 0).toLocaleString()}
              </Typography>
              <Typography variant="body2" color="text.secondary">Total Liquidity</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="warning.main">
                {pairs.reduce((sum, p) => sum + p.fee, 0).toFixed(3)}%
              </Typography>
              <Typography variant="body2" color="text.secondary">Total Fees</Typography>
            </Card>
          </Grid>
        </Grid>

        {/* Table */}
        <Card>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Pair</TableCell>
                    <TableCell>Chain</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Fee (%)</TableCell>
                    <TableCell>Min Trade</TableCell>
                    <TableCell>Max Trade</TableCell>
                    <TableCell>Liquidity</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {pairs.map((pair) => (
                    <TableRow key={pair.id} hover>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                          <ShowChart color="primary" />
                          <Typography fontWeight="medium">
                            {pair.baseToken}/{pair.quoteToken}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={blockchains.find(b => b.id === pair.chainId)?.symbol || `Chain ${pair.chainId}`} 
                          size="small" 
                        />
                      </TableCell>
                      <TableCell>
                        <Chip label={pair.status} color={getStatusColor(pair.status) as any} size="small" />
                      </TableCell>
                      <TableCell>{pair.fee}%</TableCell>
                      <TableCell>{pair.minTrade}</TableCell>
                      <TableCell>{pair.maxTrade.toLocaleString()}</TableCell>
                      <TableCell>${pair.liquidity.toLocaleString()}</TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={() => handleViewPair(pair)}>
                          <ViewIcon />
                        </IconButton>
                        <IconButton size="small" onClick={() => handleEditPair(pair)}>
                          <EditIcon />
                        </IconButton>
                        {pair.status === 'active' ? (
                          <IconButton size="small" color="warning" onClick={() => handleSuspendPair(pair.id)}>
                            <SuspendIcon />
                          </IconButton>
                        ) : (
                          <IconButton size="small" color="success" onClick={() => handleResumePair(pair.id)}>
                            <ResumeIcon />
                          </IconButton>
                        )}
                        <IconButton size="small" color="error" onClick={() => handleHaltPair(pair.id)}>
                          <HaltIcon />
                        </IconButton>
                        <IconButton size="small" color="error" onClick={() => handleDeletePair(pair.id)}>
                          <DeleteIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
          <TablePagination
            component="div"
            count={total}
            page={page}
            onPageChange={(event, newPage) => setPage(newPage)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(event) => {
              setRowsPerPage(parseInt(event.target.value, 10));
              setPage(0);
            }}
          />
        </Card>

        {/* Dialog */}
        <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="sm" fullWidth>
          <DialogTitle>
            {dialogMode === 'create' && 'Create Trading Pair'}
            {dialogMode === 'edit' && 'Edit Trading Pair'}
            {dialogMode === 'view' && 'Trading Pair Details'}
          </DialogTitle>
          <DialogContent>
            {dialogMode === 'view' && selectedPair ? (
              <Box sx={{ pt: 2 }}>
                <Grid container spacing={2}>
                  <Grid item xs={12}>
                    <Typography><strong>Pair:</strong> {selectedPair.baseToken}/{selectedPair.quoteToken}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Chain:</strong> {blockchains.find(b => b.id === selectedPair.chainId)?.name || `Chain ${selectedPair.chainId}`}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Status:</strong> {selectedPair.status}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Pair Address:</strong> {selectedPair.pairAddress || 'N/A'}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Fee:</strong> {selectedPair.fee}%</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Min Trade:</strong> {selectedPair.minTrade}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Max Trade:</strong> {selectedPair.maxTrade.toLocaleString()}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Liquidity:</strong> ${selectedPair.liquidity.toLocaleString()}</Typography>
                  </Grid>
                </Grid>
              </Box>
            ) : (
              <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Base Token"
                      fullWidth
                      value={formData.baseToken}
                      onChange={(e) => setFormData({ ...formData, baseToken: e.target.value.toUpperCase() })}
                      required
                      placeholder="BTC"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Quote Token"
                      fullWidth
                      value={formData.quoteToken}
                      onChange={(e) => setFormData({ ...formData, quoteToken: e.target.value.toUpperCase() })}
                      required
                      placeholder="USDT"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <FormControl fullWidth>
                      <InputLabel>Blockchain</InputLabel>
                      <Select
                        value={formData.chainId}
                        label="Blockchain"
                        onChange={(e) => setFormData({ ...formData, chainId: e.target.value as number })}
                      >
                        {blockchains.map((bc) => (
                          <MenuItem key={bc.id} value={bc.id}>{bc.name} ({bc.symbol})</MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Pair Address"
                      fullWidth
                      value={formData.pairAddress}
                      onChange={(e) => setFormData({ ...formData, pairAddress: e.target.value })}
                      placeholder="0x..."
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      label="Fee (%)"
                      type="number"
                      fullWidth
                      value={formData.fee}
                      onChange={(e) => setFormData({ ...formData, fee: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      label="Min Trade"
                      type="number"
                      fullWidth
                      value={formData.minTrade}
                      onChange={(e) => setFormData({ ...formData, minTrade: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      label="Max Trade"
                      type="number"
                      fullWidth
                      value={formData.maxTrade}
                      onChange={(e) => setFormData({ ...formData, maxTrade: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                </Grid>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
            {dialogMode !== 'view' && (
              <Button onClick={handleSubmit} variant="contained">
                {dialogMode === 'create' ? 'Create' : 'Update'}
              </Button>
            )}
          </DialogActions>
        </Dialog>

        <Snackbar
          open={snackbar.open}
          autoHideDuration={6000}
          onClose={() => setSnackbar(p => ({ ...p, open: false }))}
        >
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default TradingPairsPage;
