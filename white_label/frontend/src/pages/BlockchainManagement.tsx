import React, { useState, useEffect, useCallback } from 'react';
import {
  Container, Grid, Card, Button, Chip, Typography, Box, IconButton,
  CircularProgress, Alert, Snackbar, Avatar
} from '@mui/material';
import {
  DarkMode, LightMode,
  CheckCircle as EnableIcon, Block as DisableIcon, Security as ChainIcon
} from '@mui/icons-material';
import { api, Blockchain } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const BlockchainManagement: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [blockchains, setBlockchains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(true);
  const [snackbar, setSnackbar] = useState<{open: boolean; message: string; severity: 'success' | 'error'}>({
    open: false, message: '', severity: 'success'
  });

  const fetchBlockchains = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api.getBlockchains();
      setBlockchains(data);
    } catch (error) {
      console.error('Failed to fetch blockchains:', error);
      setSnackbar({ open: true, message: 'Failed to fetch blockchains', severity: 'error' });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchBlockchains();
  }, [fetchBlockchains]);

  const handleEnableBlockchain = async (id: number) => {
    try {
      await api.enableBlockchain(id);
      setSnackbar({ open: true, message: 'Blockchain enabled', severity: 'success' });
      fetchBlockchains();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to enable blockchain', severity: 'error' });
    }
  };

  const handleDisableBlockchain = async (id: number) => {
    try {
      await api.disableBlockchain(id);
      setSnackbar({ open: true, message: 'Blockchain disabled', severity: 'success' });
      fetchBlockchains();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to disable blockchain', severity: 'error' });
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'evm': return 'primary';
      case 'solana': return 'success';
      case 'bitcoin': return 'warning';
      case 'cosmos': return 'info';
      default: return 'default';
    }
  };

  const enabledCount = blockchains.filter(b => b.status === 'enabled').length;

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
            Blockchain Management
          </Typography>
          <IconButton onClick={toggleTheme} color="primary">
            {theme === 'dark' ? <LightMode /> : <DarkMode />}
          </IconButton>
        </Box>

        {/* Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="primary">{blockchains.length}</Typography>
              <Typography variant="body2" color="text.secondary">Total Blockchains</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="success.main">{enabledCount}</Typography>
              <Typography variant="body2" color="text.secondary">Enabled</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="info.main">
                {blockchains.filter(b => b.category === 'evm').length}
              </Typography>
              <Typography variant="body2" color="text.secondary">EVM Chains</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="warning.main">
                {blockchains.filter(b => b.isDefault).length}
              </Typography>
              <Typography variant="body2" color="text.secondary">Default</Typography>
            </Card>
          </Grid>
        </Grid>

        {/* Blockchain Grid */}
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <Grid container spacing={3}>
            {blockchains.map((blockchain) => (
              <Grid item xs={12} sm={6} md={4} key={blockchain.id}>
                <Card sx={{ p: 3, height: '100%' }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
                      <Avatar sx={{ bgcolor: 'primary.main', width: 48, height: 48 }}>
                        <ChainIcon />
                      </Avatar>
                      <Box>
                        <Typography variant="h6" fontWeight="bold">{blockchain.name}</Typography>
                        <Typography variant="body2" color="text.secondary">{blockchain.symbol}</Typography>
                      </Box>
                    </Box>
                    <Chip 
                      label={blockchain.status} 
                      color={blockchain.status === 'enabled' ? 'success' : 'error'} 
                      size="small" 
                    />
                  </Box>
                  
                  <Box sx={{ mb: 2 }}>
                    <Chip 
                      label={blockchain.category.toUpperCase()} 
                      color={getCategoryColor(blockchain.category) as any} 
                      size="small" 
                      sx={{ mr: 1 }}
                    />
                    {blockchain.isDefault && (
                      <Chip label="Default" color="primary" size="small" />
                    )}
                  </Box>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    RPCs: {blockchain.rpcUrls?.length || 0} | Explorers: {blockchain.explorerUrls?.length || 0}
                  </Typography>

                  <Box sx={{ display: 'flex', gap: 1 }}>
                    {blockchain.status === 'enabled' ? (
                      <Button
                        size="small"
                        variant="outlined"
                        color="error"
                        startIcon={<DisableIcon />}
                        onClick={() => handleDisableBlockchain(blockchain.id)}
                      >
                        Disable
                      </Button>
                    ) : (
                      <Button
                        size="small"
                        variant="outlined"
                        color="success"
                        startIcon={<EnableIcon />}
                        onClick={() => handleEnableBlockchain(blockchain.id)}
                      >
                        Enable
                      </Button>
                    )}
                  </Box>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}

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

export default BlockchainManagement;
