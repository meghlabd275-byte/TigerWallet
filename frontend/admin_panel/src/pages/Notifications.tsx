import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Select, MenuItem, FormControl, InputLabel, Card, CardContent,
  Switch, FormControlLabel, IconButton, Snackbar, Alert, LinearProgress,
  Avatar
} from '@mui/material';
import { 
  Add, Send, Refresh, Email, Sms, PushNotifications, Broadcast
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Notification {
  id: number;
  title: string;
  message: string;
  type: string;
  priority: string;
  is_read: boolean;
  created_at: string;
}

interface NotificationsProps {
  darkMode?: boolean;
}

const Notifications: React.FC<NotificationsProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  
  const [notification, setNotification] = useState({
    user_id: 0,
    title: '',
    message: '',
    type: 'info',
    priority: 'normal',
    send_email: false,
    send_sms: false,
    send_push: false,
  });

  // Fetch notifications
  const { data: notificationsData, isLoading, refetch } = useQuery({
    queryKey: ['notifications'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/notifications');
      if (!response.ok) throw new Error('Failed to fetch notifications');
      return response.json();
    },
  });

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['notificationStats'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/notifications/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      return response.json();
    },
  });

  // Send notification mutation
  const sendMutation = useMutation({
    mutationFn: async (notif: typeof notification) => {
      const response = await fetch('/api/v1/admin/notifications', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(notif),
      });
      if (!response.ok) throw new Error('Failed to send notification');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['notificationStats'] });
      setCreateDialogOpen(false);
      setNotification({
        user_id: 0, title: '', message: '', type: 'info', 
        priority: 'normal', send_email: false, send_sms: false, send_push: false
      });
      setSnackbar({ open: true, message: 'Notification sent successfully!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to send notification', severity: 'error' });
    },
  });

  // Broadcast mutation
  const broadcastMutation = useMutation({
    mutationFn: async (notif: typeof notification) => {
      const response = await fetch('/api/v1/admin/notifications/broadcast', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: notif.title,
          message: notif.message,
          type: notif.type,
        }),
      });
      if (!response.ok) throw new Error('Failed to broadcast');
      return response.json();
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: `Sent to ${data.notified} users!`, severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to broadcast', severity: 'error' });
    },
  });

  // Mark as read mutation
  const markReadMutation = useMutation({
    mutationFn: async (id: number) => {
      const response = await fetch(`/api/v1/admin/notifications/${id}/read`, {
        method: 'PUT',
      });
      if (!response.ok) throw new Error('Failed to mark as read');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['notificationStats'] });
    },
  });

  // Mark all read mutation
  const markAllReadMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch('/api/v1/admin/notifications/read-all', {
        method: 'PUT',
      });
      if (!response.ok) throw new Error('Failed to mark all as read');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['notificationStats'] });
      setSnackbar({ open: true, message: 'All notifications marked as read!', severity: 'success' });
    },
  });

  const notifications = notificationsData?.data || [];
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'success': return 'success';
      case 'warning': return 'warning';
      case 'error': return 'error';
      default: return 'info';
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
            Notifications
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button 
              variant="outlined"
              onClick={() => markAllReadMutation.mutate()}
            >
              Mark All Read
            </Button>
            <Button 
              variant="contained" 
              startIcon={<Send />}
              onClick={() => setCreateDialogOpen(true)}
            >
              Send Notification
            </Button>
          </Box>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Avatar sx={{ bgcolor: 'info.main' }}><Email /></Avatar>
                <Box>
                  <Typography variant="h5" sx={{ color: textPrimary }}>{stats?.total_notifications || 0}</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Total</Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Avatar sx={{ bgcolor: 'warning.main' }}><Email /></Avatar>
                <Box>
                  <Typography variant="h5" sx={{ color: textPrimary }}>{stats?.unread_count || 0}</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Unread</Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Avatar sx={{ bgcolor: 'success.main' }}><Broadcast /></Avatar>
                <Box>
                  <Typography variant="h5" sx={{ color: textPrimary }}>{stats?.today_count || 0}</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Today</Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Avatar sx={{ bgcolor: 'error.main' }}><Email /></Avatar>
                <Box>
                  <Typography variant="h5" sx={{ color: textPrimary }}>{stats?.error_count || 0}</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>Errors</Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Quick Actions */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} md={4}>
            <Card 
              sx={{ bgcolor: cardBg, cursor: 'pointer', '&:hover': { opacity: 0.9 } }}
              onClick={() => {
                setNotification({
                  ...notification,
                  title: 'System Update',
                  message: 'We will be performing scheduled maintenance.',
                  type: 'info',
                  send_email: true,
                  send_push: true,
                });
                setCreateDialogOpen(true);
              }}
            >
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Email sx={{ color: '#f97316', fontSize: 40 }} />
                <Box>
                  <Typography variant="h6" sx={{ color: textPrimary }}>Email Campaign</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    Send to all users
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card 
              sx={{ bgcolor: cardBg, cursor: 'pointer', '&:hover': { opacity: 0.9 } }}
              onClick={() => {
                setNotification({
                  ...notification,
                  title: 'Security Alert',
                  message: 'Unusual login detected.',
                  type: 'warning',
                  send_push: true,
                });
                setCreateDialogOpen(true);
              }}
            >
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <PushNotifications sx={{ color: '#f97316', fontSize: 40 }} />
                <Box>
                  <Typography variant="h6" sx={{ color: textPrimary }}>Push Notification</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    Mobile app users
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Sms sx={{ color: '#f97316', fontSize: 40 }} />
                <Box>
                  <Typography variant="h6" sx={{ color: textPrimary }}>SMS Campaign</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    Coming soon
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Notifications Table */}
        <Paper sx={{ bgcolor: cardBg }}>
          {isLoading ? (
            <LinearProgress />
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Title</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Message</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Type</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Priority</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Created</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {notifications.map((notif: Notification) => (
                    <TableRow 
                      key={notif.id}
                      onClick={() => !notif.is_read && markReadMutation.mutate(notif.id)}
                      sx={{ cursor: 'pointer', bgcolor: notif.is_read ? 'transparent' : (darkMode ? '#1a1a1a' : '#f5f5f5') }}
                    >
                      <TableCell>
                        {notif.is_read ? (
                          <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'grey.400' }} />
                        ) : (
                          <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: '#f97316' }} />
                        )}
                      </TableCell>
                      <TableCell sx={{ color: textPrimary, fontWeight: notif.is_read ? 'normal' : 'bold' }}>
                        {notif.title}
                      </TableCell>
                      <TableCell sx={{ color: textPrimary, maxWidth: 300 }}>
                        <Typography noWrap>{notif.message}</Typography>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={notif.type} 
                          size="small" 
                          color={getTypeColor(notif.type)} 
                        />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={notif.priority} 
                          size="small" 
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        {new Date(notif.created_at).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        {/* Send Dialog */}
        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Send Notification</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Title"
                  value={notification.title}
                  onChange={(e) => setNotification({...notification, title: e.target.value})}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  multiline
                  rows={3}
                  label="Message"
                  value={notification.message}
                  onChange={(e) => setNotification({...notification, message: e.target.value})}
                />
              </Grid>
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Type</InputLabel>
                  <Select
                    value={notification.type}
                    label="Type"
                    onChange={(e) => setNotification({...notification, type: e.target.value})}
                  >
                    <MenuItem value="info">Info</MenuItem>
                    <MenuItem value="success">Success</MenuItem>
                    <MenuItem value="warning">Warning</MenuItem>
                    <MenuItem value="error">Error</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Priority</InputLabel>
                  <Select
                    value={notification.priority}
                    label="Priority"
                    onChange={(e) => setNotification({...notification, priority: e.target.value})}
                  >
                    <MenuItem value="low">Low</MenuItem>
                    <MenuItem value="normal">Normal</MenuItem>
                    <MenuItem value="high">High</MenuItem>
                    <MenuItem value="urgent">Urgent</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ color: textSecondary, mb: 1 }}>
                  Send via:
                </Typography>
                <FormControlLabel
                  control={
                    <Switch 
                      checked={notification.send_email}
                      onChange={(e) => setNotification({...notification, send_email: e.target.checked})}
                    />
                  }
                  label="Email"
                />
                <FormControlLabel
                  control={
                    <Switch 
                      checked={notification.send_push}
                      onChange={(e) => setNotification({...notification, send_push: e.target.checked})}
                    />
                  }
                  label="Push Notification"
                />
                <FormControlLabel
                  control={
                    <Switch 
                      checked={notification.send_sms}
                      onChange={(e) => setNotification({...notification, send_sms: e.target.checked})}
                    />
                  }
                  label="SMS"
                />
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button 
              variant="outlined"
              onClick={() => broadcastMutation.mutate(notification)}
              disabled={!notification.title || !notification.message}
            >
              Broadcast All
            </Button>
            <Button 
              variant="contained" 
              onClick={() => sendMutation.mutate(notification)}
              disabled={!notification.title || !notification.message}
            >
              Send
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

export default Notifications;
