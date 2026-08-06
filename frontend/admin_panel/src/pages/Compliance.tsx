import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Select, MenuItem, FormControl, InputLabel, Card, CardContent,
  Tabs, Tab, Snackbar, Alert, LinearProgress, IconButton
} from '@mui/material';
import { 
  Add, Download, Refresh, Warning, CheckCircle, FilePresent, Timeline
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface ComplianceReport {
  id: number;
  report_id: string;
  type: string;
  period_start: string;
  period_end: string;
  status: string;
  file_path: string;
  created_at: string;
}

interface ComplianceProps {
  darkMode?: boolean;
}

const Compliance: React.FC<ComplianceProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [tabValue, setTabValue] = useState(0);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });
  
  const [reportConfig, setReportConfig] = useState({
    type: 'aml',
    period_start: '',
    period_end: '',
    format: 'json',
  });

  // Fetch compliance reports
  const { data: reportsData, isLoading, refetch } = useQuery({
    queryKey: ['complianceReports'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/compliance/reports');
      if (!response.ok) throw new Error('Failed to fetch reports');
      return response.json();
    },
  });

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['complianceStats'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/compliance/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      return response.json();
    },
  });

  // Generate report mutation
  const generateMutation = useMutation({
    mutationFn: async (config: typeof reportConfig) => {
      const response = await fetch('/api/v1/admin/compliance/aml', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      });
      if (!response.ok) throw new Error('Failed to generate report');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['complianceReports'] });
      queryClient.invalidateQueries({ queryKey: ['complianceStats'] });
      setCreateDialogOpen(false);
      setSnackbar({ open: true, message: 'Report generated successfully!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to generate report', severity: 'error' });
    },
  });

  // Generate tax report mutation
  const taxMutation = useMutation({
    mutationFn: async (data: { user_id: number; year: number }) => {
      const response = await fetch('/api/v1/admin/compliance/tax', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!response.ok) throw new Error('Failed to generate tax report');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['complianceReports'] });
      setSnackbar({ open: true, message: 'Tax report generated!', severity: 'success' });
    },
  });

  const reports = reportsData?.data || [];
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'generating': return 'warning';
      case 'pending': return 'info';
      case 'failed': return 'error';
      default: return 'default';
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
            Compliance Center
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button 
              variant="contained" 
              startIcon={<Add />}
              onClick={() => setCreateDialogOpen(true)}
            >
              Generate Report
            </Button>
            <IconButton onClick={() => refetch()}>
              <Refresh />
            </IconButton>
          </Box>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Total Reports</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats?.total_reports || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>AML Reports</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats?.aml_reports || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>High Risk Users</Typography>
                <Typography variant="h4" sx={{ color: 'error.main' }}>{stats?.high_risk_users || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Pending GDPR</Typography>
                <Typography variant="h4" sx={{ color: 'warning.main' }}>{stats?.pending_gdpr || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Report Type Tabs */}
        <Paper sx={{ mb: 2, bgcolor: cardBg }}>
          <Tabs 
            value={tabValue} 
            onChange={(_, v) => setTabValue(v)}
            sx={{ 
              '& .MuiTab-root': { color: textSecondary },
              '& .Mui-selected': { color: '#f97316' }
            }}
          >
            <Tab label="AML Reports" />
            <Tab label="Tax Reports" />
            <Tab label="GDPR Requests" />
          </Tabs>
        </Paper>

        {/* Reports Table */}
        <Paper sx={{ bgcolor: cardBg }}>
          {isLoading ? (
            <LinearProgress />
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Report ID</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Type</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Period</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Generated</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {reports.map((report: ComplianceReport) => (
                    <TableRow key={report.id}>
                      <TableCell sx={{ color: textPrimary }}>{report.report_id}</TableCell>
                      <TableCell>
                        <Chip label={report.type.toUpperCase()} size="small" />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        {report.period_start?.split('T')[0]} - {report.period_end?.split('T')[0]}
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={report.status} 
                          size="small" 
                          color={getStatusColor(report.status)} 
                        />
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        {new Date(report.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell>
                        <IconButton size="small">
                          <Download />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        {/* Quick Actions */}
        <Grid container spacing={2} sx={{ mt: 3 }}>
          <Grid item xs={12} md={4}>
            <Card sx={{ bgcolor: cardBg, cursor: 'pointer', '&:hover': { opacity: 0.9 } }}
              onClick={() => generateMutation.mutate({
                type: 'aml',
                period_start: new Date(Date.now() - 30*24*60*60*1000).toISOString().split('T')[0],
                period_end: new Date().toISOString().split('T')[0],
                format: 'json',
              })}
            >
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Warning sx={{ color: 'warning.main', fontSize: 40 }} />
                <Box>
                  <Typography variant="h6" sx={{ color: textPrimary }}>Generate AML Report</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    Anti-Money Laundering compliance report
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card sx={{ bgcolor: cardBg, cursor: 'pointer', '&:hover': { opacity: 0.9 } }}
              onClick={() => taxMutation.mutate({ user_id: 1, year: new Date().getFullYear() })}
            >
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <FilePresent sx={{ color: 'info.main', fontSize: 40 }} />
                <Box>
                  <Typography variant="h6" sx={{ color: textPrimary }}>Generate Tax Report</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    Annual tax report for user
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card sx={{ bgcolor: cardBg, cursor: 'pointer', '&:hover': { opacity: 0.9 } }}>
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <CheckCircle sx={{ color: 'success.main', fontSize: 40 }} />
                <Box>
                  <Typography variant="h6" sx={{ color: textPrimary }}>GDPR Dashboard</Typography>
                  <Typography variant="body2" sx={{ color: textSecondary }}>
                    Manage data requests
                  </Typography>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Create Report Dialog */}
        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle sx={{ bgcolor: cardBg }}>Generate Compliance Report</DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg, pt: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <FormControl fullWidth>
                  <InputLabel>Report Type</InputLabel>
                  <Select
                    value={reportConfig.type}
                    label="Report Type"
                    onChange={(e) => setReportConfig({...reportConfig, type: e.target.value})}
                  >
                    <MenuItem value="aml">AML Report</MenuItem>
                    <MenuItem value="tax">Tax Report</MenuItem>
                    <MenuItem value="gdpr">GDPR Report</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Start Date"
                  type="date"
                  InputLabelProps={{ shrink: true }}
                  value={reportConfig.period_start}
                  onChange={(e) => setReportConfig({...reportConfig, period_start: e.target.value})}
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="End Date"
                  type="date"
                  InputLabelProps={{ shrink: true }}
                  value={reportConfig.period_end}
                  onChange={(e) => setReportConfig({...reportConfig, period_end: e.target.value})}
                />
              </Grid>
              <Grid item xs={12}>
                <FormControl fullWidth>
                  <InputLabel>Format</InputLabel>
                  <Select
                    value={reportConfig.format}
                    label="Format"
                    onChange={(e) => setReportConfig({...reportConfig, format: e.target.value})}
                  >
                    <MenuItem value="json">JSON</MenuItem>
                    <MenuItem value="pdf">PDF</MenuItem>
                    <MenuItem value="excel">Excel</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg }}>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button 
              variant="contained" 
              onClick={() => generateMutation.mutate(reportConfig)}
              disabled={!reportConfig.period_start || !reportConfig.period_end}
            >
              Generate
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

export default Compliance;
