import React, { useState, useEffect, useCallback } from 'react';
import {
  Container, Grid, Card, Button, TextField, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, TablePagination, Chip, Typography, Box, IconButton, 
  InputAdornment, CircularProgress, Avatar
} from '@mui/material';
import {
  Search as SearchIcon, DarkMode, LightMode, Notifications as NotifIcon,
  CheckCircle, CircleOutlined, Delete as DeleteIcon
} from '@mui/icons-material';
import { api, Notification, PaginatedResponse } from '../services/api';
import { useTheme } from '../../context/ThemeContext';

const NotificationsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [total, setTotal] = useState(0);

  const fetchNotifications = useCallback(async () => {
    try {
      setLoading(true);
      const response: PaginatedResponse<Notification> = await api.getNotifications({
        page: page + 1,
        pageSize: rowsPerPage
      });
      setNotifications(response.data);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch notifications:', error);
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage]);

  useEffect(() => {
    fetchNotifications();
  }, [fetchNotifications]);

  const handleMarkRead = async (id: string) => {
    try {
      await api.markNotificationRead(id);
      fetchNotifications();
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
    }
  };

  const unreadCount = notifications.filter(n => !n.read).length;

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'success': return 'success';
      case 'warning': return 'warning';
      case 'error': return 'error';
      case 'info': return 'info';
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
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <NotifIcon fontSize="large" color="primary" />
            <Typography variant="h4" fontWeight="bold">
              Notifications
            </Typography>
            {unreadCount > 0 && (
              <Chip label={`${unreadCount} unread`} color="error" size="small" />
            )}
          </Box>
          <IconButton onClick={toggleTheme} color="primary">
            {theme === 'dark' ? <LightMode /> : <DarkMode />}
          </IconButton>
        </Box>

        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="primary">{total}</Typography>
              <Typography variant="body2" color="text.secondary">Total</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="error.main">{unreadCount}</Typography>
              <Typography variant="body2" color="text.secondary">Unread</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="success.main">{total - unreadCount}</Typography>
              <Typography variant="body2" color="text.secondary">Read</Typography>
            </Card>
          </Grid>
        </Grid>

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
                    <TableCell>Status</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Title</TableCell>
                    <TableCell>Message</TableCell>
                    <TableCell>Time</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {notifications.map((notif) => (
                    <TableRow key={notif.id} hover sx={{ bgcolor: notif.read ? 'transparent' : 'action.hover' }}>
                      <TableCell>
                        {notif.read ? (
                          <CheckCircle color="success" />
                        ) : (
                          <CircleOutlined color="action" />
                        )}
                      </TableCell>
                      <TableCell>
                        <Chip label={notif.type} color={getTypeColor(notif.type) as any} size="small" />
                      </TableCell>
                      <TableCell>
                        <Typography fontWeight={notif.read ? 'normal' : 'bold'}>
                          {notif.title}
                        </Typography>
                      </TableCell>
                      <TableCell>{notif.message || 'N/A'}</TableCell>
                      <TableCell>{new Date(notif.createdAt).toLocaleString()}</TableCell>
                      <TableCell align="right">
                        {!notif.read && (
                          <Button size="small" onClick={() => handleMarkRead(notif.id)}>
                            Mark Read
                          </Button>
                        )}
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
      </Container>
    </Box>
  );
};

export default NotificationsPage;
