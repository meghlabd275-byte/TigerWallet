import React, { useState, useEffect, useCallback } from 'react';
import {
  Container, Grid, Card, Button, TextField, Select, MenuItem, FormControl, InputLabel,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TablePagination,
  Dialog, DialogTitle, DialogContent, DialogActions, Chip, Typography, Box, Avatar,
  IconButton, Switch, FormControlLabel, CircularProgress, Alert, Snackbar, InputAdornment
} from '@mui/material';
import {
  Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon, Search as SearchIcon,
  Visibility as ViewIcon, Block as BlockIcon, CheckCircle as ActivateIcon,
  Email as EmailIcon, Security as SecurityIcon, DarkMode, LightMode
} from '@mui/icons-material';
import { api, WhiteLabelAdmin, PaginatedResponse } from '../services/api';
import { useTheme } from '../../context/ThemeContext';

const AdminManagement: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [admins, setAdmins] = useState<WhiteLabelAdmin[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedAdmin, setSelectedAdmin] = useState<WhiteLabelAdmin | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | 'view'>('create');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [total, setTotal] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [snackbar, setSnackbar] = useState<{open: boolean; message: string; severity: 'success' | 'error'}>({
    open: false, message: '', severity: 'success'
  });

  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    role: 'support' as WhiteLabelAdmin['role'],
    clientId: ''
  });

  const fetchAdmins = useCallback(async () => {
    try {
      setLoading(true);
      const response: PaginatedResponse<WhiteLabelAdmin> = await api.getAdmins({
        page: page + 1,
        pageSize: rowsPerPage,
        query: searchQuery
      });
      setAdmins(response.data);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch admins:', error);
      setSnackbar({ open: true, message: 'Failed to fetch admins', severity: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, searchQuery]);

  useEffect(() => {
    fetchAdmins();
  }, [fetchAdmins]);

  const handleCreateAdmin = () => {
    setSelectedAdmin(null);
    setFormData({ name: '', email: '', password: '', role: 'support', clientId: '' });
    setDialogMode('create');
    setOpenDialog(true);
  };

  const handleEditAdmin = (admin: WhiteLabelAdmin) => {
    setSelectedAdmin(admin);
    setFormData({
      name: admin.name,
      email: admin.email,
      password: '',
      role: admin.role,
      clientId: admin.clientId || ''
    });
    setDialogMode('edit');
    setOpenDialog(true);
  };

  const handleViewAdmin = (admin: WhiteLabelAdmin) => {
    setSelectedAdmin(admin);
    setDialogMode('view');
    setOpenDialog(true);
  };

  const handleDeleteAdmin = async (adminId: string) => {
    if (!confirm('Are you sure you want to delete this admin?')) return;
    try {
      await api.deleteAdmin(adminId);
      setSnackbar({ open: true, message: 'Admin deleted successfully', severity: 'success' });
      fetchAdmins();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to delete admin', severity: 'error' });
    }
  };

  const handleSubmit = async () => {
    try {
      if (dialogMode === 'create') {
        await api.createAdmin({
          name: formData.name,
          email: formData.email,
          password: formData.password,
          role: formData.role,
          clientId: formData.clientId || undefined
        });
        setSnackbar({ open: true, message: 'Admin created successfully', severity: 'success' });
      } else if (dialogMode === 'edit' && selectedAdmin) {
        await api.updateAdmin(selectedAdmin.id, {
          name: formData.name,
          role: formData.role
        });
        setSnackbar({ open: true, message: 'Admin updated successfully', severity: 'success' });
      }
      setOpenDialog(false);
      fetchAdmins();
    } catch (error) {
      setSnackbar({ open: true, message: 'Operation failed', severity: 'error' });
    }
  };

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'super_admin': return 'error';
      case 'admin': return 'warning';
      case 'manager': return 'info';
      case 'support': return 'default';
      default: return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    return status === 'active' ? 'success' : 'error';
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
            Admin Management
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <IconButton onClick={toggleTheme} color="primary">
              {theme === 'dark' ? <LightMode /> : <DarkMode />}
            </IconButton>
            <Button variant="contained" startIcon={<AddIcon />} onClick={handleCreateAdmin}>
              Create Admin
            </Button>
          </Box>
        </Box>

        {/* Search */}
        <Card sx={{ p: 2, mb: 3 }}>
          <TextField
            fullWidth
            placeholder="Search admins by name or email..."
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
        </Card>

        {/* Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="primary">{total}</Typography>
              <Typography variant="body2" color="text.secondary">Total Admins</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="success.main">{admins.filter(a => a.status === 'active').length}</Typography>
              <Typography variant="body2" color="text.secondary">Active</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="warning.main">{admins.filter(a => a.role === 'super_admin').length}</Typography>
              <Typography variant="body2" color="text.secondary">Super Admins</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="info.main">{admins.filter(a => a.twoFactorEnabled).length}</Typography>
              <Typography variant="body2" color="text.secondary">2FA Enabled</Typography>
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
                    <TableCell>Admin</TableCell>
                    <TableCell>Email</TableCell>
                    <TableCell>Role</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>2FA</TableCell>
                    <TableCell>Last Login</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {admins.map((admin) => (
                    <TableRow key={admin.id} hover>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                          <Avatar sx={{ bgcolor: 'primary.main' }}>
                            {admin.name[0].toUpperCase()}
                          </Avatar>
                          <Typography fontWeight="medium">{admin.name}</Typography>
                        </Box>
                      </TableCell>
                      <TableCell>{admin.email}</TableCell>
                      <TableCell>
                        <Chip label={admin.role} color={getRoleColor(admin.role) as any} size="small" />
                      </TableCell>
                      <TableCell>
                        <Chip label={admin.status} color={getStatusColor(admin.status) as any} size="small" />
                      </TableCell>
                      <TableCell>
                        {admin.twoFactorEnabled ? (
                          <CheckCircle color="success" />
                        ) : (
                          <BlockIcon color="disabled" />
                        )}
                      </TableCell>
                      <TableCell>
                        {admin.lastLogin ? new Date(admin.lastLogin).toLocaleString() : 'Never'}
                      </TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={() => handleViewAdmin(admin)}>
                          <ViewIcon />
                        </IconButton>
                        <IconButton size="small" onClick={() => handleEditAdmin(admin)}>
                          <EditIcon />
                        </IconButton>
                        <IconButton size="small" color="error" onClick={() => handleDeleteAdmin(admin.id)}>
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
            {dialogMode === 'create' && 'Create New Admin'}
            {dialogMode === 'edit' && 'Edit Admin'}
            {dialogMode === 'view' && 'Admin Details'}
          </DialogTitle>
          <DialogContent>
            {dialogMode === 'view' && selectedAdmin ? (
              <Box sx={{ pt: 2 }}>
                <Grid container spacing={2}>
                  <Grid item xs={12}>
                    <Typography><strong>Name:</strong> {selectedAdmin.name}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Email:</strong> {selectedAdmin.email}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Role:</strong> {selectedAdmin.role}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Status:</strong> {selectedAdmin.status}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>2FA Enabled:</strong> {selectedAdmin.twoFactorEnabled ? 'Yes' : 'No'}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Permissions:</strong> {selectedAdmin.permissions?.join(', ') || 'None'}</Typography>
                  </Grid>
                </Grid>
              </Box>
            ) : (
              <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label="Name"
                  fullWidth
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
                <TextField
                  label="Email"
                  type="email"
                  fullWidth
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  required
                  disabled={dialogMode === 'edit'}
                />
                {dialogMode === 'create' && (
                  <TextField
                    label="Password"
                    type="password"
                    fullWidth
                    value={formData.password}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    required
                  />
                )}
                <FormControl fullWidth>
                  <InputLabel>Role</InputLabel>
                  <Select
                    value={formData.role}
                    label="Role"
                    onChange={(e) => setFormData({ ...formData, role: e.target.value as WhiteLabelAdmin['role'] })}
                  >
                    <MenuItem value="super_admin">Super Admin</MenuItem>
                    <MenuItem value="admin">Admin</MenuItem>
                    <MenuItem value="manager">Manager</MenuItem>
                    <MenuItem value="support">Support</MenuItem>
                  </Select>
                </FormControl>
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

export default AdminManagement;
