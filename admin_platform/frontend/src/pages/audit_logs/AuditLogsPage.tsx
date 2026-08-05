/**
 * TigerWallet Admin Platform - Audit Logs Page
 * Complete audit log viewer with filtering, search, and export
 * Dark/Light theme works everywhere
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Container, Grid, Card, TextField, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, TablePagination, Chip, Typography, IconButton, InputAdornment,
  CircularProgress, Select, MenuItem, FormControl, InputLabel, Button, Dialog,
  DialogTitle, DialogContent, DialogActions, Alert, Snackbar
} from '@mui/material';
import {
  Search as SearchIcon, DarkMode, LightMode, Security as AuditIcon,
  AccessTime as TimeIcon, Computer as IPCIcon, Refresh as RefreshIcon,
  Download as DownloadIcon, FilterList as FilterIcon, Visibility as ViewIcon,
  CheckCircle, Cancel, Warning
} from '@mui/icons-material';
import { useTheme } from '../../contexts/ThemeContext';

// ============================================================================
// Types
// ============================================================================

export interface AuditLog {
  id: number;
  admin_id: number;
  admin_email: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  details?: string;
  ip_address: string;
  user_agent: string;
  status: 'success' | 'failed' | 'pending';
  created_at: string;
}

export interface AuditLogFilters {
  page: number;
  pageSize: number;
  action?: string;
  resource_type?: string;
  status?: string;
  admin_id?: number;
  start_date?: string;
  end_date?: string;
  search?: string;
}

export interface AuditLogStats {
  total: number;
  success: number;
  failed: number;
  pending: number;
}

// ============================================================================
// API Functions
// ============================================================================

const mockAuditLogs: AuditLog[] = [
  { id: 1, admin_id: 1, admin_email: 'superadmin@tigerwallet.com', action: 'USER_LOGIN', resource_type: 'auth', details: 'Successful login', ip_address: '192.168.1.1', user_agent: 'Chrome/120', status: 'success', created_at: '2024-01-15T10:30:00Z' },
  { id: 2, admin_id: 2, admin_email: 'admin@tigerwallet.com', action: 'CREATE_USER', resource_type: 'user', resource_id: '123', details: 'Created new user', ip_address: '192.168.1.2', user_agent: 'Firefox/121', status: 'success', created_at: '2024-01-15T10:25:00Z' },
  { id: 3, admin_id: 1, admin_email: 'superadmin@tigerwallet.com', action: 'UPDATE_FEE', resource_type: 'fee', resource_id: 'fee_eth', details: 'Updated ETH fee to 0.5%', ip_address: '192.168.1.1', user_agent: 'Chrome/120', status: 'success', created_at: '2024-01-15T10:20:00Z' },
  { id: 4, admin_id: 3, admin_email: 'moderator@tigerwallet.com', action: 'DELETE_TOKEN', resource_type: 'token', resource_id: '456', details: 'Token deletion attempt', ip_address: '192.168.1.3', user_agent: 'Safari/17', status: 'failed', created_at: '2024-01-15T10:15:00Z' },
  { id: 5, admin_id: 2, admin_email: 'admin@tigerwallet.com', action: 'APPROVE_KYC', resource_type: 'kyc', resource_id: 'kyc_789', details: 'Approved KYC level 2', ip_address: '192.168.1.2', user_agent: 'Firefox/121', status: 'success', created_at: '2024-01-15T10:10:00Z' },
  { id: 6, admin_id: 1, admin_email: 'superadmin@tigerwallet.com', action: 'SUSPEND_WL', resource_type: 'whitelabel', resource_id: 'wl_123', details: 'Suspended white label', ip_address: '192.168.1.1', user_agent: 'Chrome/120', status: 'success', created_at: '2024-01-15T10:05:00Z' },
  { id: 7, admin_id: 4, admin_email: 'support@tigerwallet.com', action: 'VIEW_USER', resource_type: 'user', resource_id: '999', details: 'Viewed user profile', ip_address: '192.168.1.4', user_agent: 'Edge/120', status: 'success', created_at: '2024-01-15T10:00:00Z' },
  { id: 8, admin_id: 3, admin_email: 'moderator@tigerwallet.com', action: 'EXPORT_DATA', resource_type: 'report', details: 'Exported transaction data', ip_address: '192.168.1.3', user_agent: 'Safari/17', status: 'pending', created_at: '2024-01-15T09:55:00Z' },
  { id: 9, admin_id: 1, admin_email: 'superadmin@tigerwallet.com', action: 'CREATE_ADMIN', resource_type: 'admin', details: 'Created new admin account', ip_address: '192.168.1.1', user_agent: 'Chrome/120', status: 'success', created_at: '2024-01-15T09:50:00Z' },
  { id: 10, admin_id: 2, admin_email: 'admin@tigerwallet.com', action: 'UPDATE_CHAIN', resource_type: 'blockchain', resource_id: 'chain_eth', details: 'Updated Ethereum RPC endpoints', ip_address: '192.168.1.2', user_agent: 'Firefox/121', status: 'success', created_at: '2024-01-15T09:45:00Z' },
];

const fetchAuditLogs = async (filters: AuditLogFilters): Promise<{ data: AuditLog[]; total: number; stats: AuditLogStats }> => {
  await new Promise(resolve => setTimeout(resolve, 500));
  
  let filtered = [...mockAuditLogs];
  
  if (filters.action) {
    filtered = filtered.filter(log => log.action.includes(filters.action!.toUpperCase()));
  }
  if (filters.resource_type) {
    filtered = filtered.filter(log => log.resource_type === filters.resource_type);
  }
  if (filters.status) {
    filtered = filtered.filter(log => log.status === filters.status);
  }
  if (filters.search) {
    const search = filters.search.toLowerCase();
    filtered = filtered.filter(log => 
      log.admin_email.toLowerCase().includes(search) ||
      log.action.toLowerCase().includes(search) ||
      log.details?.toLowerCase().includes(search) ||
      log.ip_address.includes(search)
    );
  }
  
  const total = filtered.length;
  const stats: AuditLogStats = {
    total: mockAuditLogs.length,
    success: mockAuditLogs.filter(l => l.status === 'success').length,
    failed: mockAuditLogs.filter(l => l.status === 'failed').length,
    pending: mockAuditLogs.filter(l => l.status === 'pending').length,
  };
  
  const start = filters.page * filters.pageSize;
  const data = filtered.slice(start, start + filters.pageSize);
  
  return { data, total, stats };
};

// ============================================================================
// Main Component
// ============================================================================

const AuditLogsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const isDark = theme === 'dark';
  
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [stats, setStats] = useState<AuditLogStats>({ total: 0, success: 0, failed: 0, pending: 0 });
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [total, setTotal] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' | 'info' }>({ open: false, message: '', severity: 'info' });

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetchAuditLogs({
        page,
        pageSize: rowsPerPage,
        action: actionFilter || undefined,
        resource_type: resourceFilter || undefined,
        status: statusFilter || undefined,
        search: searchQuery || undefined,
      });
      setLogs(response.data);
      setTotal(response.total);
      setStats(response.stats);
    } catch (error) {
      console.error('Failed to fetch audit logs:', error);
      setSnackbar({ open: true, message: 'Failed to fetch audit logs', severity: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, actionFilter, resourceFilter, statusFilter, searchQuery]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleRefresh = () => {
    fetchLogs();
    setSnackbar({ open: true, message: 'Audit logs refreshed', severity: 'success' });
  };

  const handleExport = (format: 'csv' | 'json' | 'pdf') => {
    let content: string;
    let filename: string;
    let mimeType: string;

    if (format === 'json') {
      content = JSON.stringify(logs, null, 2);
      filename = 'audit_logs.json';
      mimeType = 'application/json';
    } else if (format === 'csv') {
      const headers = ['ID', 'Admin Email', 'Action', 'Resource Type', 'Resource ID', 'Details', 'IP Address', 'Status', 'Created At'];
      const rows = logs.map(log => [
        log.id, log.admin_email, log.action, log.resource_type, log.resource_id || '', 
        log.details || '', log.ip_address, log.status, log.created_at
      ].join(','));
      content = [headers.join(','), ...rows].join('\n');
      filename = 'audit_logs.csv';
      mimeType = 'text/csv';
    } else {
      content = `Audit Logs Export\nGenerated: ${new Date().toISOString()}\n\n${logs.map(log => 
        `ID: ${log.id}\nAdmin: ${log.admin_email}\nAction: ${log.action}\nResource: ${log.resource_type}\nStatus: ${log.status}\nTime: ${log.created_at}\n---`
      ).join('\n')}`;
      filename = 'audit_logs.txt';
      mimeType = 'text/plain';
    }

    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
    
    setSnackbar({ open: true, message: `Exported ${logs.length} records as ${format.toUpperCase()}`, severity: 'success' });
  };

  const handleViewDetails = (log: AuditLog) => {
    setSelectedLog(log);
    setDetailsOpen(true);
  };

  const getActionColor = (action: string) => {
    if (action.includes('CREATE') || action.includes('LOGIN') || action.includes('APPROVE')) return 'success';
    if (action.includes('UPDATE') || action.includes('EDIT')) return 'info';
    if (action.includes('DELETE') || action.includes('SUSPEND') || action.includes('BAN')) return 'error';
    if (action.includes('VIEW') || action.includes('EXPORT')) return 'default';
    return 'default';
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success': return <CheckCircle color="success" />;
      case 'failed': return <Cancel color="error" />;
      case 'pending': return <Warning color="warning" />;
      default: return null;
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  return (
    <Box sx={{ 
      minHeight: '100vh', 
      bgcolor: isDark ? 'background.default' : '#f5f5f5',
      color: isDark ? 'white' : 'text.primary',
      transition: 'all 0.3s'
    }}>
      <Container maxWidth="xl" sx={{ py: 4 }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <AuditIcon sx={{ fontSize: 40, color: 'primary.main' }} />
            <Typography variant="h4" fontWeight="bold">
              Audit Logs
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={handleRefresh}
              sx={{ mr: 1 }}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              startIcon={<DownloadIcon />}
              onClick={() => handleExport('csv')}
            >
              Export CSV
            </Button>
            <IconButton onClick={toggleTheme} color="primary" sx={{ ml: 1 }}>
              {isDark ? <LightMode /> : <DarkMode />}
            </IconButton>
          </Box>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white', transition: 'all 0.3s' }}>
              <Typography variant="body2" color="text.secondary">Total Logs</Typography>
              <Typography variant="h3" fontWeight="bold" color="primary">{stats.total}</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white', transition: 'all 0.3s' }}>
              <Typography variant="body2" color="text.secondary">Successful</Typography>
              <Typography variant="h3" fontWeight="bold" color="success.main">{stats.success}</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white', transition: 'all 0.3s' }}>
              <Typography variant="body2" color="text.secondary">Failed</Typography>
              <Typography variant="h3" fontWeight="bold" color="error.main">{stats.failed}</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white', transition: 'all 0.3s' }}>
              <Typography variant="body2" color="text.secondary">Pending</Typography>
              <Typography variant="h3" fontWeight="bold" color="warning.main">{stats.pending}</Typography>
            </Card>
          </Grid>
        </Grid>

        {/* Filters */}
        <Card sx={{ p: 3, mb: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={3}>
              <TextField
                fullWidth
                label="Search"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                }}
                size="small"
              />
            </Grid>
            <Grid item xs={12} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>Action</InputLabel>
                <Select
                  value={actionFilter}
                  label="Action"
                  onChange={(e) => setActionFilter(e.target.value)}
                >
                  <MenuItem value="">All</MenuItem>
                  <MenuItem value="LOGIN">Login</MenuItem>
                  <MenuItem value="CREATE">Create</MenuItem>
                  <MenuItem value="UPDATE">Update</MenuItem>
                  <MenuItem value="DELETE">Delete</MenuItem>
                  <MenuItem value="VIEW">View</MenuItem>
                  <MenuItem value="EXPORT">Export</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>Resource</InputLabel>
                <Select
                  value={resourceFilter}
                  label="Resource"
                  onChange={(e) => setResourceFilter(e.target.value)}
                >
                  <MenuItem value="">All</MenuItem>
                  <MenuItem value="user">User</MenuItem>
                  <MenuItem value="auth">Auth</MenuItem>
                  <MenuItem value="fee">Fee</MenuItem>
                  <MenuItem value="token">Token</MenuItem>
                  <MenuItem value="kyc">KYC</MenuItem>
                  <MenuItem value="whitelabel">White Label</MenuItem>
                  <MenuItem value="blockchain">Blockchain</MenuItem>
                  <MenuItem value="admin">Admin</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>Status</InputLabel>
                <Select
                  value={statusFilter}
                  label="Status"
                  onChange={(e) => setStatusFilter(e.target.value)}
                >
                  <MenuItem value="">All</MenuItem>
                  <MenuItem value="success">Success</MenuItem>
                  <MenuItem value="failed">Failed</MenuItem>
                  <MenuItem value="pending">Pending</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={3}>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button variant="outlined" onClick={() => handleExport('csv')}>CSV</Button>
                <Button variant="outlined" onClick={() => handleExport('json')}>JSON</Button>
              </Box>
            </Grid>
          </Grid>
        </Card>

        {/* Table */}
        <Card sx={{ bgcolor: isDark ? 'grey.900' : 'white' }}>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>ID</TableCell>
                    <TableCell>Admin</TableCell>
                    <TableCell>Action</TableCell>
                    <TableCell>Resource</TableCell>
                    <TableCell>Details</TableCell>
                    <TableCell>IP Address</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Time</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id} hover>
                      <TableCell>{log.id}</TableCell>
                      <TableCell>{log.admin_email}</TableCell>
                      <TableCell>
                        <Chip 
                          label={log.action} 
                          color={getActionColor(log.action) as any} 
                          size="small" 
                        />
                      </TableCell>
                      <TableCell>{log.resource_type}</TableCell>
                      <TableCell sx={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {log.details || '-'}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          <IPCIcon fontSize="small" color="action" />
                          {log.ip_address}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          {getStatusIcon(log.status)}
                          {log.status}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          <TimeIcon fontSize="small" color="action" />
                          {formatDate(log.created_at)}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <IconButton size="small" onClick={() => handleViewDetails(log)}>
                          <ViewIcon />
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
            onPageChange={(_, newPage) => setPage(newPage)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(e) => {
              setRowsPerPage(parseInt(e.target.value, 10));
              setPage(0);
            }}
            rowsPerPageOptions={[5, 10, 25, 50]}
          />
        </Card>

        {/* Details Dialog */}
        <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle>Audit Log Details</DialogTitle>
          <DialogContent>
            {selectedLog && (
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">ID</Typography>
                  <Typography variant="body1">{selectedLog.id}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">Admin Email</Typography>
                  <Typography variant="body1">{selectedLog.admin_email}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">Action</Typography>
                  <Typography variant="body1">{selectedLog.action}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">Resource Type</Typography>
                  <Typography variant="body1">{selectedLog.resource_type}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">Resource ID</Typography>
                  <Typography variant="body1">{selectedLog.resource_id || '-'}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">Status</Typography>
                  <Chip 
                    label={selectedLog.status} 
                    color={selectedLog.status === 'success' ? 'success' : selectedLog.status === 'failed' ? 'error' : 'warning'}
                    size="small"
                  />
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="caption" color="text.secondary">Details</Typography>
                  <Typography variant="body1">{selectedLog.details || '-'}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">IP Address</Typography>
                  <Typography variant="body1">{selectedLog.ip_address}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">User Agent</Typography>
                  <Typography variant="body1" sx={{ wordBreak: 'break-all' }}>{selectedLog.user_agent}</Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="caption" color="text.secondary">Timestamp</Typography>
                  <Typography variant="body1">{formatDate(selectedLog.created_at)}</Typography>
                </Grid>
              </Grid>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setDetailsOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Snackbar */}
        <Snackbar 
          open={snackbar.open} 
          autoHideDuration={3000} 
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

export default AuditLogsPage;
