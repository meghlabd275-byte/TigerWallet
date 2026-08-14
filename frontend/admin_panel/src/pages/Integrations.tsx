import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Switch, FormControlLabel, IconButton, Card, CardContent,
  Tabs, Tab, Snackbar, Alert, Select, MenuItem, FormControl, InputLabel
} from '@mui/material';
import { 
  Add, Edit, Delete, Refresh, Send, Storage, Cloud, Notifications, Security
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Integration {
  id: number;
  type: string;
  name: string;
  is_active: boolean;
  last_sync_at: string;
  sync_status: string;
  config?: Record<string, string>;
}

interface IntegrationsProps {
  darkMode?: boolean;
}

const Integrations: React.FC<IntegrationsProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [tabValue, setTabValue] = useState(0);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editIntegration, setEditIntegration] = useState<Integration | null>(null);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  
  const [newIntegration, setNewIntegration] = useState({
    type: 'slack',
    name: '',
    config: {} as Record<string, string>,
    is_active: true,
  });

  // Fetch integrations
  const { data: integrationsData, isLoading, refetch } = useQuery({
    queryKey: ['integrations'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/integrations');
      if (!response.ok) throw new Error('Failed to fetch integrations');
      return response.json();
    },
  });

  // Create integration mutation
  const createMutation = useMutation({
    mutationFn: async (integration: typeof newIntegration) => {
      const response = await fetch('/api/v1/admin/integrations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(integration),
      });
      if (!response.ok) throw new Error('Failed to create integration');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations'] });
      setCreateDialogOpen(false);
      setNewIntegration({ type: 'slack', name: '', config: {}, is_active: true });
      setSnackbar({ open: true, message: 'Integration created successfully!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to create integration', severity: 'error' });
    },
  });

  // Toggle integration mutation
  const toggleMutation = useMutation({
    mutationFn: async (integration: Integration) => {
      const response = await fetch(`/api/v1/admin/integrations/${integration.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: integration.name, is_active: !integration.is_active, config: integration.config }),
      });
      if (!response.ok) throw new Error('Failed to update integration');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations'] });
      setSnackbar({ open: true, message: 'Integration updated!', severity: 'success' });
    },
  });

  // Delete integration mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/v1/admin/integrations/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) throw new Error('Failed to delete integration');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations'] });
      setSnackbar({ open: true, message: 'Integration deleted!', severity: 'success' });
    },
  });

  // Test integration mutation
  const testMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/v1/admin/integrations/${id}/test`, {
        method: 'POST',
      });
      if (!response.ok) throw new Error('Failed to test integration');
      return response.json();
    },
    onSuccess: (data) => {
      setSnackbar({ open: true, message: `Test ${data.test_status}!`, severity: data.test_status === 'passed' ? 'success' : 'error' });
    },
  });

  // Send notification mutations
  const slackNotifyMutation = useMutation({
    mutationFn: async (data: { channel: string; text: string }) => {
      const response = await fetch('/api/v1/integrations/slack/notify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!response.ok) throw new Error('Failed to send notification');
      return response.json();
    },
    onSuccess: () => {
      setSnackbar({ open: true, message: 'Slack notification sent!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to send notification', severity: 'error' });
    },
  });

  const integrations = integrationsData?.data || [];
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'slack': return <Notifications />;
      case 'datadog': return <Storage />;
      case 'pagerduty': return <Security />;
      case 'cloudflare': return <Cloud />;
      default: return <Storage />;
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
            Integrations
          </Typography>
          <Button 
            variant="contained" 
            startIcon={<Add />}
            onClick={() => setCreateDialogOpen(true)}
          >
            Add Integration
          </Button>
        </Box>

        {/* Integration Type Tabs */}
        <Paper sx={{ mb: 2, bgcolor: cardBg }}>
          <Tabs 
            value={tabValue} 
            onChange={(_, v) => setTabValue(v)}
            sx={{ 
              '& .MuiTab-root': { color: textSecondary },
              '& .Mui-selected': { color: '#f97316' }
            }}
          >
            <Tab label="All" />
            <Tab label="Slack" />
            <Tab label="PagerDuty" />
            <Tab label="Datadog" />
            <Tab label="Cloudflare" />
          </Tabs>
        </Paper>

        {/* Integration Cards */}
        <Grid container spacing={3}>
          {integrations
            .filter((i: Integration) => tabValue === 0 || i.type.toLowerCase() === ['slack', 'pagerduty', 'datadog', 'cloudflare'][tabValue - 1])
            .map((integration: Integration) => (
            <Grid item xs={12} md={6} key={integration.id}>
              <Card sx={{ bgcolor: cardBg }}>
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Box sx={{ 
                        p: 1, 
                        borderRadius: 1, 
                        bgcolor: darkMode ? '#222' : '#f5f5f5',
                        color: '#f97316'
                      }}>
                        {getTypeIcon(integration.type)}
                      </Box>
                      <Box>
                        <Typography variant="h6" sx={{ color: textPrimary }}>{integration.name}</Typography>
                        <Chip 
                          label={integration.type} 
                          size="small" 
                          sx={{ mt: 0.5 }}
                        />
                      </Box>
                    </Box>
                    <Switch 
                      checked={integration.is_active}
                      onChange={() => toggleMutation.mutate(integration)}
                      color="warning"
                    />
                  </Box>
                  
                  <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
                    <Button 
                      size="small" 
                      variant="outlined"
                      onClick={() => testMutation.mutate(integration.id)}
                    >
                      Test
                    </Button>
                    <Button 
                      size="small" 
                      variant="outlined"
                      onClick={() => setEditIntegration(integration)}
                    >
                      Configure
                    </Button>
                    <Button 
                      size="small" 
                      variant="outlined"
                      color="error"
                      onClick={() => deleteMutation.mutate(integration.id)}
                    >
                      Delete
                    </Button>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>

        {integrations.length === 0 && !isLoading && (
          <Paper sx={{ p: 4, textAlign: 'center', bgcolor: cardBg }}>
            <Typography variant="h6" sx={{ color: textSecondary }}>
              No integrations configured. Add one to get started.
            </Typography>
          </Paper>
        )}

        {/* Create Dialog */}
        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Add Integration</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <FormControl fullWidth>
                  <InputLabel>Type</InputLabel>
                  <Select
                    value={newIntegration.type}
                    label="Type"
                    onChange={(e) => setNewIntegration({...newIntegration, type: e.target.value})}
                  >
                    <MenuItem value="slack">Slack</MenuItem>
                    <MenuItem value="pagerduty">PagerDuty</MenuItem>
                    <MenuItem value="datadog">Datadog</MenuItem>
                    <MenuItem value="cloudflare">Cloudflare</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Name"
                  value={newIntegration.name}
                  onChange={(e) => setNewIntegration({...newIntegration, name: e.target.value})}
                />
              </Grid>
              {newIntegration.type === 'slack' && (
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Webhook URL"
                    type="password"
                    onChange={(e) => setNewIntegration({
                      ...newIntegration, 
                      config: { ...newIntegration.config, webhook_url: e.target.value }
                    })}
                  />
                </Grid>
              )}
              {newIntegration.type === 'pagerduty' && (
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Integration Key"
                    type="password"
                    onChange={(e) => setNewIntegration({
                      ...newIntegration, 
                      config: { ...newIntegration.config, integration_key: e.target.value }
                    })}
                  />
                </Grid>
              )}
              {newIntegration.type === 'datadog' && (
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="API Key"
                    type="password"
                    onChange={(e) => setNewIntegration({
                      ...newIntegration, 
                      config: { ...newIntegration.config, api_key: e.target.value }
                    })}
                  />
                </Grid>
              )}
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button 
              variant="contained" 
              onClick={() => createMutation.mutate(newIntegration)}
              disabled={!newIntegration.name}
            >
              Create
            </Button>
          </DialogActions>
        </Dialog>

        {/* Snackbar */}
        <Snackbar
          open={snackbar.open}
          autoHideDuration={6000}
          onClose={() => setSnackbar({ ...snackbar, open: false })}
        >
          <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
            {snackbar.message}
          </Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Integrations;
