import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Switch, FormControlLabel, IconButton, Snackbar, Alert, LinearProgress, Card, CardContent
} from '@mui/material';
import { 
  Add, Block, CheckCircle, Refresh, VerifiedUser, Warning, Shield
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface IPRule {
  id: number;
  ip_address: string;
  cidr: string;
  description: string;
  is_whitelist: boolean;
  is_active: boolean;
  created_at: string;
}

interface SecurityProps {
  darkMode?: boolean;
}

const Security: React.FC<SecurityProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  const [newRule, setNewRule] = useState({
    ip_address: '', cidr: '', description: '', is_whitelist: true, is_active: true
  });

  const { data: ipRulesData, isLoading, refetch } = useQuery({
    queryKey: ['ipRules'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/security/ip-rules');
      if (!response.ok) throw new Error('Failed to fetch IP rules');
      return response.json();
    },
  });

  const { data: stats } = useQuery({
    queryKey: ['securityStats'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/security/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      return response.json();
    },
  });

  const createMutation = useMutation({
    mutationFn: async (rule: typeof newRule) => {
      const response = await fetch('/api/v1/admin/security/ip-rules', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(rule)
      });
      if (!response.ok) throw new Error('Failed to create rule');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ipRules'] });
      setCreateDialogOpen(false);
      setNewRule({ ip_address: '', cidr: '', description: '', is_whitelist: true, is_active: true });
      setSnackbar({ open: true, message: 'IP rule added!', severity: 'success' });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async (rule: IPRule) => {
      const response = await fetch(`/api/v1/admin/security/ip-rules/${rule.id}`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: !rule.is_active })
      });
      if (!response.ok) throw new Error('Failed to update rule');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ipRules'] });
    },
  });

  const ipRules = ipRulesData?.data || [];
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5', color: textPrimary }}>
      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, color: textPrimary }}>Security Settings</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" startIcon={<Add />} onClick={() => setCreateDialogOpen(true)}>Add IP Rule</Button>
            <IconButton onClick={() => refetch()}><Refresh /></IconButton>
          </Box>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Shield sx={{ color: '#f97316', fontSize: 32 }} />
                <Box>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Whitelist IPs</Typography>
                  <Typography variant="h5" sx={{ color: textPrimary }}>{ipRules.filter((r: IPRule) => r.is_whitelist && r.is_active).length}</Typography>
                </Box>
              </Box>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Block sx={{ color: 'error.main', fontSize: 32 }} />
                <Box>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Blocked IPs</Typography>
                  <Typography variant="h5" sx={{ color: textPrimary }}>{ipRules.filter((r: IPRule) => !r.is_whitelist && r.is_active).length}</Typography>
                </Box>
              </Box>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Warning sx={{ color: 'warning.main', fontSize: 32 }} />
                <Box>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Failed Logins (24h)</Typography>
                  <Typography variant="h5" sx={{ color: 'warning.main' }}>{stats?.failed_logins_24h || 0}</Typography>
                </Box>
              </Box>
            </CardContent></Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}><CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <CheckCircle sx={{ color: 'success.main', fontSize: 32 }} />
                <Box>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Active Sessions</Typography>
                  <Typography variant="h5" sx={{ color: 'success.main' }}>{stats?.active_sessions || 0}</Typography>
                </Box>
              </Box>
            </CardContent></Card>
          </Grid>
        </Grid>

        {/* Security Settings */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} md={6}>
            <Paper sx={{ bgcolor: cardBg, p: 2 }}>
              <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>Rate Limiting</Typography>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                <Typography sx={{ color: textPrimary }}>Enable Rate Limiting</Typography>
                <Switch defaultChecked />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                <Typography sx={{ color: textPrimary }}>Max Requests per Minute</Typography>
                <Chip label="60" />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                <Typography sx={{ color: textPrimary }}>Block Duration (minutes)</Typography>
                <Chip label="15" />
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12} md={6}>
            <Paper sx={{ bgcolor: cardBg, p: 2 }}>
              <Typography variant="h6" sx={{ color: textPrimary, mb: 2 }}>2FA Settings</Typography>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                <Typography sx={{ color: textPrimary }}>Require 2FA for Admins</Typography>
                <Switch defaultChecked />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                <Typography sx={{ color: textPrimary }}>Require 2FA for Users</Typography>
                <Switch />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1 }}>
                <Typography sx={{ color: textPrimary }}>Backup Codes Enabled</Typography>
                <Switch defaultChecked />
              </Box>
            </Paper>
          </Grid>
        </Grid>

        {/* IP Rules Table */}
        <Paper sx={{ bgcolor: cardBg }}>
          <Typography variant="h6" sx={{ color: textPrimary, p: 2 }}>IP Access Rules</Typography>
          {isLoading ? <LinearProgress /> : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>IP/CIDR</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Description</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Type</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {ipRules.map((rule: IPRule) => (
                    <TableRow key={rule.id}>
                      <TableCell sx={{ color: textPrimary, fontFamily: 'monospace', fontWeight: 'bold' }}>
                        {rule.cidr || rule.ip_address}
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>{rule.description}</TableCell>
                      <TableCell>
                        <Chip 
                          label={rule.is_whitelist ? 'Whitelist' : 'Blacklist'} 
                          size="small" 
                          color={rule.is_whitelist ? 'success' : 'error'} 
                        />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={rule.is_active ? 'Active' : 'Inactive'} 
                          size="small" 
                          color={rule.is_active ? 'success' : 'default'} 
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton size="small" onClick={() => toggleMutation.mutate(rule)}>
                          <Switch checked={rule.is_active} size="small" />
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
          <DialogTitle sx={{ bgcolor: cardBg }}>Add IP Rule</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}><TextField fullWidth label="IP Address or CIDR" value={newRule.ip_address || newRule.cidr} onChange={(e) => setNewRule({...newRule, ip_address: e.target.value, cidr: e.target.value})} /></Grid>
              <Grid item xs={12}><TextField fullWidth label="Description" value={newRule.description} onChange={(e) => setNewRule({...newRule, description: e.target.value})} /></Grid>
              <Grid item xs={12}>
                <FormControlLabel control={<Switch checked={newRule.is_whitelist} onChange={(e) => setNewRule({...newRule, is_whitelist: e.target.checked})} />} label="Whitelist (allow)" />
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={() => createMutation.mutate(newRule)}>Create</Button>
          </DialogActions>
        </Dialog>

        <Snackbar open={snackbar.open} autoHideDuration={6000} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
        </Snackbar>
      </Container>
    </Box>
  );
};

export default Security;
