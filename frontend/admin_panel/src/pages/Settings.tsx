import React, { useState, useEffect } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Switch, FormControlLabel, Divider, Alert, CircularProgress,
  Card, CardContent, IconButton, List, ListItem, ListItemText,
  ListItemSecondaryAction, Dialog, DialogTitle, DialogContent,
  DialogActions, Tab, Tabs, Chip, Snackbar, useTheme
} from '@mui/material';
import { 
  Security as SecurityIcon, DarkMode, LightMode, Notifications,
  Email, Sms, PushNotifications, Language, Backup, Delete
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface TwoFactorStatus {
  enabled: boolean;
  methods: string[];
  backup_codes_left: number;
  last_verified_at: number | null;
  trusted_devices: number;
  recovery_codes_left: number;
}

interface SettingsProps {
  darkMode: boolean;
  onThemeToggle: () => void;
}

const Settings: React.FC<SettingsProps> = ({ darkMode, onThemeToggle }) => {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const [tabValue, setTabValue] = useState(0);
  const [twoFactorCode, setTwoFactorCode] = useState('');
  const [setupDialogOpen, setSetupDialogOpen] = useState(false);
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [qrCode, setQrCode] = useState('');
  const [secret, setSecret] = useState('');
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });

  // Fetch 2FA status
  const { data: twoFactorStatus, isLoading: statusLoading } = useQuery<TwoFactorStatus>({
    queryKey: ['twoFactorStatus'],
    queryFn: async () => {
      const response = await fetch('/api/v1/2fa/status/admin/1');
      if (!response.ok) throw new Error('Failed to fetch 2FA status');
      return response.json();
    },
  });

  // Setup 2FA mutation
  const setupMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch('/api/v1/2fa/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: 1, user_type: 'admin' }),
      });
      if (!response.ok) throw new Error('Failed to setup 2FA');
      return response.json();
    },
    onSuccess: (data) => {
      setQrCode(data.qr_code);
      setSecret(data.secret);
      setBackupCodes(data.backup_codes);
      setSetupDialogOpen(true);
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to setup 2FA', severity: 'error' });
    },
  });

  // Enable 2FA mutation
  const enableMutation = useMutation({
    mutationFn: async (code: string) => {
      const response = await fetch('/api/v1/2fa/enable', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: 1, user_type: 'admin', code }),
      });
      if (!response.ok) throw new Error('Failed to enable 2FA');
      return response.json();
    },
    onSuccess: () => {
      setSetupDialogOpen(false);
      setTwoFactorCode('');
      queryClient.invalidateQueries({ queryKey: ['twoFactorStatus'] });
      setSnackbar({ open: true, message: '2FA enabled successfully! Save your backup codes securely.', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Invalid verification code', severity: 'error' });
    },
  });

  // Disable 2FA mutation
  const disableMutation = useMutation({
    mutationFn: async (data: { code: string; password: string }) => {
      const response = await fetch('/api/v1/2fa/disable', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: 1, user_type: 'admin', ...data }),
      });
      if (!response.ok) throw new Error('Failed to disable 2FA');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['twoFactorStatus'] });
      setSnackbar({ open: true, message: '2FA disabled successfully', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Invalid code or password', severity: 'error' });
    },
  });

  // Regenerate backup codes mutation
  const regenerateMutation = useMutation({
    mutationFn: async (password: string) => {
      const response = await fetch('/api/v1/2fa/regenerate-backup-codes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: 1, user_type: 'admin', password }),
      });
      if (!response.ok) throw new Error('Failed to regenerate codes');
      return response.json();
    },
    onSuccess: (data) => {
      setBackupCodes(data.backup_codes);
      setSetupDialogOpen(true);
      setSnackbar({ open: true, message: 'Backup codes regenerated! Save them securely.', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Invalid password', severity: 'error' });
    },
  });

  const handleEnable2FA = () => {
    if (twoFactorCode.length === 6) {
      enableMutation.mutate(twoFactorCode);
    }
  };

  const handleDisable2FA = () => {
    const code = prompt('Enter your 2FA code or backup code:');
    const password = prompt('Enter your password:');
    if (code && password) {
      disableMutation.mutate({ code, password });
    }
  };

  const handleRegenerateCodes = () => {
    const password = prompt('Enter your password to regenerate backup codes:');
    if (password) {
      regenerateMutation.mutate(password);
    }
  };

  return (
    <Box sx={{ 
      minHeight: '100vh',
      bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5',
      color: darkMode ? '#fff' : '#000',
      transition: 'all 0.3s ease'
    }}>
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Typography variant="h4" sx={{ mb: 4, fontWeight: 700 }}>
          Settings
        </Typography>

        <Tabs 
          value={tabValue} 
          onChange={(_, v) => setTabValue(v)}
          sx={{ 
            mb: 3,
            '& .MuiTab-root': { color: darkMode ? '#aaa' : '#666' },
            '& .Mui-selected': { color: '#f97316' }
          }}
        >
          <Tab icon={<SecurityIcon />} label="Security" />
          <Tab icon={<DarkMode />} label="Appearance" />
          <Tab icon={<Notifications />} label="Notifications" />
        </Tabs>

        {/* Security Tab */}
        {tabValue === 0 && (
          <Grid container spacing={3}>
            {/* Two-Factor Authentication */}
            <Grid item xs={12} md={6}>
              <Paper sx={{ 
                p: 3, 
                bgcolor: darkMode ? '#1a1a1a' : '#fff',
                border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
              }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <SecurityIcon sx={{ mr: 2, color: '#f97316' }} />
                  <Typography variant="h6">Two-Factor Authentication</Typography>
                </Box>
                
                {statusLoading ? (
                  <CircularProgress size={24} />
                ) : (
                  <>
                    <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                      <Chip 
                        label={twoFactorStatus?.enabled ? 'Enabled' : 'Disabled'} 
                        color={twoFactorStatus?.enabled ? 'success' : 'default'}
                        sx={{ mr: 2 }}
                      />
                      {twoFactorStatus?.enabled && (
                        <Typography variant="body2" color="text.secondary">
                          {twoFactorStatus.backup_codes_left} backup codes remaining
                        </Typography>
                      )}
                    </Box>

                    {twoFactorStatus?.enabled ? (
                      <Box>
                        <Button 
                          variant="outlined" 
                          color="error" 
                          onClick={handleDisable2FA}
                          startIcon={<Delete />}
                          sx={{ mr: 1 }}
                        >
                          Disable 2FA
                        </Button>
                        <Button 
                          variant="outlined" 
                          onClick={handleRegenerateCodes}
                          startIcon={<Backup />}
                        >
                          Regenerate Codes
                        </Button>
                      </Box>
                    ) : (
                      <Button 
                        variant="contained" 
                        onClick={() => setupMutation.mutate()}
                        disabled={setupMutation.isPending}
                        startIcon={<SecurityIcon />}
                      >
                        Enable 2FA
                      </Button>
                    )}
                  </>
                )}
              </Paper>
            </Grid>

            {/* IP Whitelist */}
            <Grid item xs={12} md={6}>
              <Paper sx={{ 
                p: 3, 
                bgcolor: darkMode ? '#1a1a1a' : '#fff',
                border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
              }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <Language sx={{ mr: 2, color: '#f97316' }} />
                  <Typography variant="h6">IP Whitelist</Typography>
                </Box>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  Restrict access to specific IP addresses
                </Typography>
                <Button variant="outlined" startIcon={<SecurityIcon />}>
                  Manage IPs
                </Button>
              </Paper>
            </Grid>

            {/* Rate Limiting */}
            <Grid item xs={12} md={6}>
              <Paper sx={{ 
                p: 3, 
                bgcolor: darkMode ? '#1a1a1a' : '#fff',
                border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
              }}>
                <Typography variant="h6" sx={{ mb: 2 }}>Rate Limiting</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  Configure API rate limits
                </Typography>
                <Button variant="outlined">Configure Limits</Button>
              </Paper>
            </Grid>
          </Grid>
        )}

        {/* Appearance Tab */}
        {tabValue === 1 && (
          <Paper sx={{ 
            p: 3, 
            bgcolor: darkMode ? '#1a1a1a' : '#fff',
            border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
          }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography variant="h6">Dark Mode</Typography>
                <Typography variant="body2" color="text.secondary">
                  {darkMode ? 'Currently using dark theme' : 'Currently using light theme'}
                </Typography>
              </Box>
              <FormControlLabel
                control={
                  <Switch 
                    checked={darkMode} 
                    onChange={onThemeToggle}
                    color="warning"
                  />
                }
                label={darkMode ? <DarkMode /> : <LightMode />}
              />
            </Box>
          </Paper>
        )}

        {/* Notifications Tab */}
        {tabValue === 2 && (
          <Grid container spacing={3}>
            <Grid item xs={12} md={4}>
              <Paper sx={{ 
                p: 3, 
                bgcolor: darkMode ? '#1a1a1a' : '#fff',
                border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
              }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <Email sx={{ mr: 2, color: '#f97316' }} />
                  <Typography variant="h6">Email Notifications</Typography>
                </Box>
                <FormControlLabel control={<Switch defaultChecked />} label="Enabled" />
              </Paper>
            </Grid>
            <Grid item xs={12} md={4}>
              <Paper sx={{ 
                p: 3, 
                bgcolor: darkMode ? '#1a1a1a' : '#fff',
                border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
              }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <Sms sx={{ mr: 2, color: '#f97316' }} />
                  <Typography variant="h6">SMS Notifications</Typography>
                </Box>
                <FormControlLabel control={<Switch />} label="Enabled" />
              </Paper>
            </Grid>
            <Grid item xs={12} md={4}>
              <Paper sx={{ 
                p: 3, 
                bgcolor: darkMode ? '#1a1a1a' : '#fff',
                border: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
              }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                  <PushNotifications sx={{ mr: 2, color: '#f97316' }} />
                  <Typography variant="h6">Push Notifications</Typography>
                </Box>
                <FormControlLabel control={<Switch />} label="Enabled" />
              </Paper>
            </Grid>
          </Grid>
        )}

        {/* 2FA Setup Dialog */}
        <Dialog 
          open={setupDialogOpen} 
          onClose={() => setSetupDialogOpen(false)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle sx={{ 
            bgcolor: darkMode ? '#1a1a1a' : '#fff',
            borderBottom: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
          }}>
            Setup Two-Factor Authentication
          </DialogTitle>
          <DialogContent sx={{ 
            bgcolor: darkMode ? '#1a1a1a' : '#fff',
            pt: 3
          }}>
            <Alert severity="warning" sx={{ mb: 3 }}>
              Save these backup codes somewhere safe. You'll need them if you lose access to your authenticator.
            </Alert>
            
            {backupCodes.length > 0 && (
              <Paper sx={{ 
                p: 2, 
                bgcolor: darkMode ? '#0a0a0a' : '#f5f5f5',
                mb: 3 
              }}>
                <Grid container spacing={1}>
                  {backupCodes.map((code, idx) => (
                    <Grid item xs={6} key={idx}>
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          fontFamily: 'monospace',
                          bgcolor: darkMode ? '#222' : '#fff',
                          p: 1,
                          borderRadius: 1
                        }}
                      >
                        {code}
                      </Typography>
                    </Grid>
                  ))}
                </Grid>
              </Paper>
            )}

            {secret && (
              <TextField
                fullWidth
                label="Secret Key"
                value={secret}
                InputProps={{ readOnly: true }}
                sx={{ mb: 2 }}
              />
            )}

            {qrCode && !backupCodes.length && (
              <Box sx={{ textAlign: 'center', mb: 3 }}>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  Scan this QR code with your authenticator app
                </Typography>
                <img 
                  src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrCode)}`} 
                  alt="2FA QR Code"
                  style={{ borderRadius: 8 }}
                />
              </Box>
            )}

            {!backupCodes.length && (
              <TextField
                fullWidth
                label="Verification Code"
                value={twoFactorCode}
                onChange={(e) => setTwoFactorCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="Enter 6-digit code"
                sx={{ mb: 2 }}
              />
            )}
          </DialogContent>
          <DialogActions sx={{ 
            bgcolor: darkMode ? '#1a1a1a' : '#fff',
            borderTop: `1px solid ${darkMode ? '#333' : '#e0e0e0'}`
          }}>
            {!backupCodes.length ? (
              <>
                <Button onClick={() => setSetupDialogOpen(false)}>Cancel</Button>
                <Button 
                  onClick={handleEnable2FA} 
                  variant="contained"
                  disabled={twoFactorCode.length !== 6 || enableMutation.isPending}
                >
                  Verify & Enable
                </Button>
              </>
            ) : (
              <Button 
                onClick={() => {
                  setSetupDialogOpen(false);
                  queryClient.invalidateQueries({ queryKey: ['twoFactorStatus'] });
                }} 
                variant="contained"
              >
                I've Saved My Codes
              </Button>
            )}
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

export default Settings;
