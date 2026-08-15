import React, { useState, useEffect, useCallback } from 'react';
import {
  Container, Grid, Card, TextField, Table, TableBody, TableCell, TableContainer, TableHead,
  TableRow, TablePagination, Chip, Typography, Box, IconButton, InputAdornment, CircularProgress
} from '@mui/material';
import {
  Search as SearchIcon, DarkMode, LightMode, Security as AuditIcon,
  AccessTime as TimeIcon, Computer as IPCIcon
} from '@mui/icons-material';
import { api, AuditLog, PaginatedResponse } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const AuditLogsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [total, setTotal] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      const response: PaginatedResponse<AuditLog> = await api.getAuditLogs({
        page: page + 1,
        pageSize: rowsPerPage,
        query: searchQuery
      });
      setLogs(response.data);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch audit logs:', error);
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, searchQuery]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const getActionColor = (action: string) => {
    if (action.includes('CREATE')) return 'success';
    if (action.includes('UPDATE')) return 'info';
    if (action.includes('DELETE')) return 'error';
    if (action.includes('LOGIN') || action.includes('LOGOUT')) return 'primary';
    return 'default';
  };

  const getStatusColor = (status: string) => {
    return status === 'success' ? 'success' : 'error';
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
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <AuditIcon fontSize="large" color="primary" />
            <Typography variant="h4" fontWeight="bold">
              Audit Logs
            </Typography>
          </Box>
          <IconButton onClick={toggleTheme} color="primary">
            {theme === 'dark' ? <LightMode /> : <DarkMode />}
          </IconButton>
        </Box>

        {/* Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="primary">{total}</Typography>
              <Typography variant="body2" color="text.secondary">Total Logs</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="success.main">
                {logs.filter(l => l.status === 'success').length}
              </Typography>
              <Typography variant="body2" color="text.secondary">Successful</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="error.main">
                {logs.filter(l => l.status === 'failed').length}
              </Typography>
              <Typography variant="body2" color="text.secondary">Failed</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="info.main">
                {new Set(logs.map(l => l.adminId)).size}
              </Typography>
              <Typography variant="body2" color="text.secondary">Active Admins</Typography>
            </Card>
          </Grid>
        </Grid>

        {/* Search */}
        <Card sx={{ p: 2, mb: 3 }}>
          <TextField
            fullWidth
            placeholder="Search by action, resource type, or admin..."
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
                    <TableCell>Timestamp</TableCell>
                    <TableCell>Action</TableCell>
                    <TableCell>Resource Type</TableCell>
                    <TableCell>Admin ID</TableCell>
                    <TableCell>IP Address</TableCell>
                    <TableCell>Status</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id} hover>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <TimeIcon fontSize="small" color="action" />
                          {new Date(log.createdAt).toLocaleString()}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={log.action} 
                          color={getActionColor(log.action) as any} 
                          size="small" 
                        />
                      </TableCell>
                      <TableCell>{log.resourceType}</TableCell>
                      <TableCell>
                        {log.adminId ? log.adminId.substring(0, 8) + '...' : 'System'}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <IPCIcon fontSize="small" color="action" />
                          {log.ipAddress || 'N/A'}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={log.status} 
                          color={getStatusColor(log.status) as any} 
                          size="small" 
                        />
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
            onPageChange={(_event, newPage) => setPage(newPage)}
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

export default AuditLogsPage;
