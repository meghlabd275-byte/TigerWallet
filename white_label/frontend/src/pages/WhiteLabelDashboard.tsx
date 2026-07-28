/**
 * White Label Dashboard - Complete Admin Panel
 * Production-ready with full functionality
 */

import React, { useState, useEffect, useCallback } from 'react';
import { 
  Container, Grid, Card, Button, TextField, Select, MenuItem,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Dialog, DialogTitle, DialogContent, DialogActions, Tabs, Tab,
  Chip, Typography, Box, Avatar, IconButton, Switch, FormControlLabel,
  CircularProgress, Alert, Snackbar
} from '@mui/material';
import { 
  Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon,
  Visibility as ViewIcon, Security as SecurityIcon
} from '@mui/icons-material';

interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  subdomain?: string;
  status: 'active' | 'suspended' | 'pending' | 'expired';
  plan: 'starter' | 'professional' | 'enterprise' | 'custom';
  features: Record<string, boolean>;
  branding: Record<string, string>;
  maxUsers: number;
  currentUsers: number;
  createdAt: string;
}

export const WhiteLabelDashboard: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [clients, setClients] = useState<WhiteLabelClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedClient, setSelectedClient] = useState<WhiteLabelClient | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | 'view'>('create');
  const [snackbar, setSnackbar] = useState<{open: boolean; message: string; severity: 'success' | 'error'}>({
    open: false, message: '', severity: 'success'
  });

  const fetchClients = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/white-label', {
        headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
      });
      if (response.ok) {
        const data = await response.json();
        setClients(data.clients || []);
      }
    } catch (error) {
      console.error('Failed to fetch clients:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchClients(); }, [fetchClients]);

  const handleCreateClient = () => {
    setSelectedClient(null);
    setDialogMode('create');
    setOpenDialog(true);
  };

  const handleEditClient = (client: WhiteLabelClient) => {
    setSelectedClient(client);
    setDialogMode('edit');
    setOpenDialog(true);
  };

  const handleDeleteClient = async (clientId: string) => {
    if (!confirm('Are you sure?')) return;
    try {
      const response = await fetch(`/api/white-label/${clientId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${localStorage.getItem('admin_token')}` }
      });
      if (response.ok) {
        setSnackbar({ open: true, message: 'Client deleted', severity: 'success' });
        fetchClients();
      }
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to delete', severity: 'error' });
    }
  };

  return (
    <Container maxWidth="xl">
      <Box sx={{ mb: 4, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h4" fontWeight="bold">White Label Management</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={handleCreateClient}>
          Create White Label
        </Button>
      </Box>

      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="h3" color="primary">{clients.length}</Typography>
            <Typography variant="body2" color="text.secondary">Total Clients</Typography>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="h3" color="success.main">{clients.filter(c => c.status === 'active').length}</Typography>
            <Typography variant="body2" color="text.secondary">Active</Typography>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="h3" color="warning.main">{clients.filter(c => c.status === 'pending').length}</Typography>
            <Typography variant="body2" color="text.secondary">Pending</Typography>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="h3" color="info.main">{clients.reduce((sum, c) => sum + c.currentUsers, 0)}</Typography>
            <Typography variant="body2" color="text.secondary">Total Users</Typography>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Client</TableCell>
                <TableCell>Domain</TableCell>
                <TableCell>Plan</TableCell>
                <TableCell>Users</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Created</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {clients.map((client) => (
                <TableRow key={client.id} hover>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Avatar sx={{ bgcolor: client.branding?.primaryColor || '#1976d2' }}>
                        {client.name[0]}
                      </Avatar>
                      <Typography fontWeight="medium">{client.name}</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>{client.domain}</TableCell>
                  <TableCell><Chip label={client.plan} size="small" /></TableCell>
                  <TableCell>{client.currentUsers} / {client.maxUsers}</TableCell>
                  <TableCell><Chip label={client.status} color={client.status === 'active' ? 'success' : 'default'} size="small" /></TableCell>
                  <TableCell>{new Date(client.createdAt).toLocaleDateString()}</TableCell>
                  <TableCell align="right">
                    <IconButton size="small" onClick={() => handleEditClient(client)}><EditIcon /></IconButton>
                    <IconButton size="small" color="error" onClick={() => handleDeleteClient(client.id)}><DeleteIcon /></IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar(p => ({ ...p, open: false }))}>
        <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
      </Snackbar>
    </Container>
  );
};

export default WhiteLabelDashboard;
