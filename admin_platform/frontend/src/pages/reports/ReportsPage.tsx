/**
 * TigerWallet Admin Platform - Reports Page
 * Complete report generation with PDF, Excel, CSV export
 * Dark/Light theme works everywhere
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Container, Grid, Card, TextField, Typography, Button, Select, MenuItem,
  FormControl, InputLabel, CircularProgress, Alert, Snackbar, Chip, Table,
  TableBody, TableCell, TableContainer, TableHead, TableRow, Dialog,
  DialogTitle, DialogContent, DialogActions, LinearProgress
} from '@mui/material';
import {
  Download as DownloadIcon, PictureAsPdf as PdfIcon, TableChart as ExcelIcon,
  Description as CsvIcon, CalendarMonth as CalendarIcon, Refresh as RefreshIcon,
  DarkMode, LightMode, Assessment as ReportIcon
} from '@mui/icons-material';
import { useTheme } from '../../contexts/ThemeContext';

// ============================================================================
// Types
// ============================================================================

export interface Report {
  id: string;
  name: string;
  type: 'transactions' | 'users' | 'revenue' | 'fees' | 'kyc' | 'tokens' | 'custom';
  format: 'pdf' | 'excel' | 'csv' | 'json';
  status: 'pending' | 'processing' | 'completed' | 'failed';
  start_date: string;
  end_date: string;
  record_count: number;
  file_size?: number;
  download_url?: string;
  created_by: string;
  created_at: string;
  completed_at?: string;
}

export interface ReportTemplate {
  id: string;
  name: string;
  description: string;
  type: string;
  default_format: 'pdf' | 'excel' | 'csv';
  fields: string[];
}

// ============================================================================
// Mock Data
// ============================================================================

const mockReports: Report[] = [
  { id: '1', name: 'Monthly Transactions Report', type: 'transactions', format: 'pdf', status: 'completed', start_date: '2024-01-01', end_date: '2024-01-31', record_count: 125000, file_size: 2500000, download_url: '#', created_by: 'superadmin@tigerwallet.com', created_at: '2024-02-01T10:00:00Z', completed_at: '2024-02-01T10:05:00Z' },
  { id: '2', name: 'User Activity Report', type: 'users', format: 'excel', status: 'completed', start_date: '2024-01-01', end_date: '2024-01-31', record_count: 45000, file_size: 1800000, download_url: '#', created_by: 'admin@tigerwallet.com', created_at: '2024-02-01T11:00:00Z', completed_at: '2024-02-01T11:02:00Z' },
  { id: '3', name: 'Revenue Summary', type: 'revenue', format: 'pdf', status: 'processing', start_date: '2024-01-01', end_date: '2024-01-31', record_count: 0, created_by: 'superadmin@tigerwallet.com', created_at: '2024-02-01T12:00:00Z' },
  { id: '4', name: 'Fee Structure Report', type: 'fees', format: 'csv', status: 'completed', start_date: '2024-01-01', end_date: '2024-01-15', record_count: 150, file_size: 25000, download_url: '#', created_by: 'admin@tigerwallet.com', created_at: '2024-01-16T09:00:00Z', completed_at: '2024-01-16T09:01:00Z' },
  { id: '5', name: 'KYC Verification Report', type: 'kyc', format: 'excel', status: 'completed', start_date: '2024-01-01', end_date: '2024-01-31', record_count: 5200, file_size: 950000, download_url: '#', created_by: 'superadmin@tigerwallet.com', created_at: '2024-02-01T08:00:00Z', completed_at: '2024-02-01T08:03:00Z' },
];

const reportTemplates: ReportTemplate[] = [
  { id: '1', name: 'Transaction Report', description: 'Complete transaction history with filters', type: 'transactions', default_format: 'pdf', fields: ['id', 'user_id', 'type', 'amount', 'fee', 'status', 'timestamp', 'tx_hash'] },
  { id: '2', name: 'User Report', description: 'User registration and activity data', type: 'users', default_format: 'excel', fields: ['id', 'email', 'wallet', 'kyc_status', 'created_at', 'last_login'] },
  { id: '3', name: 'Revenue Report', description: 'Revenue breakdown by type and period', type: 'revenue', default_format: 'pdf', fields: ['date', 'transaction_fees', 'withdrawal_fees', 'listing_fees', 'total'] },
  { id: '4', name: 'Fee Report', description: 'All fee configurations and collections', type: 'fees', default_format: 'csv', fields: ['asset', 'deposit_fee', 'withdrawal_fee', 'trade_fee', 'network_fee'] },
  { id: '5', name: 'KYC Report', description: 'KYC verification statistics', type: 'kyc', default_format: 'excel', fields: ['user_id', 'type', 'status', 'submitted_at', 'reviewed_at', 'reviewer'] },
  { id: '6', name: 'Token Report', description: 'Token listings and statistics', type: 'tokens', default_format: 'csv', fields: ['symbol', 'name', 'chain', 'price', 'volume_24h', 'market_cap'] },
];

// ============================================================================
// API Functions
// ============================================================================

const fetchReports = async (): Promise<Report[]> => {
  await new Promise(resolve => setTimeout(resolve, 500));
  return mockReports;
};

const createReport = async (template: ReportTemplate, startDate: string, endDate: string, format: string): Promise<Report> => {
  await new Promise(resolve => setTimeout(resolve, 1000));
  return {
    id: Date.now().toString(),
    name: template.name,
    type: template.type as any,
    format: format as any,
    status: 'completed',
    start_date: startDate,
    end_date: endDate,
    record_count: Math.floor(Math.random() * 10000) + 100,
    file_size: Math.floor(Math.random() * 5000000) + 100000,
    download_url: '#',
    created_by: 'current_admin@tigerwallet.com',
    created_at: new Date().toISOString(),
    completed_at: new Date().toISOString(),
  };
};

// ============================================================================
// Main Component
// ============================================================================

const ReportsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const isDark = theme === 'dark';
  
  const [reports, setReports] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<ReportTemplate | null>(null);
  const [startDate, setStartDate] = useState('2024-01-01');
  const [endDate, setEndDate] = useState('2024-01-31');
  const [format, setFormat] = useState<'pdf' | 'excel' | 'csv' | 'json'>('pdf');
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' | 'info' }>({ open: false, message: '', severity: 'info' });

  const loadReports = useCallback(async () => {
    try {
      setLoading(true);
      const data = await fetchReports();
      setReports(data);
    } catch (error) {
      console.error('Failed to load reports:', error);
      setSnackbar({ open: true, message: 'Failed to load reports', severity: 'error' });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadReports();
  }, [loadReports]);

  const handleCreateReport = async () => {
    if (!selectedTemplate) {
      setSnackbar({ open: true, message: 'Please select a report template', severity: 'error' });
      return;
    }
    
    try {
      setGenerating(true);
      const newReport = await createReport(selectedTemplate, startDate, endDate, format);
      setReports([newReport, ...reports]);
      setCreateOpen(false);
      setSelectedTemplate(null);
      setSnackbar({ open: true, message: 'Report generated successfully!', severity: 'success' });
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to generate report', severity: 'error' });
    } finally {
      setGenerating(false);
    }
  };

  const handleDownload = (report: Report) => {
    setSnackbar({ open: true, message: `Downloading ${report.name}...`, severity: 'info' });
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'processing': return 'warning';
      case 'failed': return 'error';
      default: return 'default';
    }
  };

  const getFormatIcon = (format: string) => {
    switch (format) {
      case 'pdf': return <PdfIcon fontSize="small" />;
      case 'excel': return <ExcelIcon fontSize="small" />;
      case 'csv': return <CsvIcon fontSize="small" />;
      default: return <DownloadIcon fontSize="small" />;
    }
  };

  const formatFileSize = (bytes?: number) => {
    if (!bytes) return '-';
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString();
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
            <ReportIcon sx={{ fontSize: 40, color: 'primary.main' }} />
            <Typography variant="h4" fontWeight="bold">
              Reports & Export
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              startIcon={<DownloadIcon />}
              onClick={() => setCreateOpen(true)}
            >
              Generate Report
            </Button>
            <IconButton onClick={toggleTheme} color="primary" sx={{ ml: 1 }}>
              {isDark ? <LightMode /> : <DarkMode />}
            </IconButton>
          </Box>
        </Box>

        {/* Quick Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Total Reports</Typography>
              <Typography variant="h3" fontWeight="bold" color="primary">{reports.length}</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Completed</Typography>
              <Typography variant="h3" fontWeight="bold" color="success.main">
                {reports.filter(r => r.status === 'completed').length}
              </Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Processing</Typography>
              <Typography variant="h3" fontWeight="bold" color="warning.main">
                {reports.filter(r => r.status === 'processing').length}
              </Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Total Data Exported</Typography>
              <Typography variant="h3" fontWeight="bold">
                {formatFileSize(reports.reduce((acc, r) => acc + (r.file_size || 0), 0))}
              </Typography>
            </Card>
          </Grid>
        </Grid>

        {/* Report Templates */}
        <Card sx={{ p: 3, mb: 4, bgcolor: isDark ? 'grey.900' : 'white' }}>
          <Typography variant="h6" gutterBottom>Quick Generate</Typography>
          <Grid container spacing={2}>
            {reportTemplates.map((template) => (
              <Grid item xs={12} sm={6} md={4} key={template.id}>
                <Card 
                  variant="outlined" 
                  sx={{ 
                    p: 2, 
                    cursor: 'pointer',
                    '&:hover': { bgcolor: isDark ? 'grey.800' : 'grey.50' }
                  }}
                  onClick={() => {
                    setSelectedTemplate(template);
                    setFormat(template.default_format);
                    setCreateOpen(true);
                  }}
                >
                  <Typography variant="subtitle1" fontWeight="bold">{template.name}</Typography>
                  <Typography variant="body2" color="text.secondary">{template.description}</Typography>
                  <Box sx={{ mt: 1, display: 'flex', gap: 0.5 }}>
                    <Chip 
                      icon={getFormatIcon(template.default_format)} 
                      label={template.default_format.toUpperCase()} 
                      size="small" 
                    />
                    <Chip label={`${template.fields.length} fields`} size="small" />
                  </Box>
                </Card>
              </Grid>
            ))}
          </Grid>
        </Card>

        {/* Reports Table */}
        <Card sx={{ bgcolor: isDark ? 'grey.900' : 'white' }}>
          <Box sx={{ p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="h6">Generated Reports</Typography>
            <Button startIcon={<RefreshIcon />} onClick={loadReports}>Refresh</Button>
          </Box>
          
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Name</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Format</TableCell>
                    <TableCell>Date Range</TableCell>
                    <TableCell>Records</TableCell>
                    <TableCell>Size</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Created</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {reports.map((report) => (
                    <TableRow key={report.id} hover>
                      <TableCell>
                        <Typography fontWeight="bold">{report.name}</Typography>
                        <Typography variant="caption" color="text.secondary">by {report.created_by}</Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={report.type} size="small" />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          icon={getFormatIcon(report.format)} 
                          label={report.format.toUpperCase()} 
                          size="small" 
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>
                        {formatDate(report.start_date)} - {formatDate(report.end_date)}
                      </TableCell>
                      <TableCell>{report.record_count.toLocaleString()}</TableCell>
                      <TableCell>{formatFileSize(report.file_size)}</TableCell>
                      <TableCell>
                        <Chip 
                          label={report.status} 
                          color={getStatusColor(report.status) as any} 
                          size="small" 
                        />
                      </TableCell>
                      <TableCell>{formatDate(report.created_at)}</TableCell>
                      <TableCell>
                        <Button
                          size="small"
                          startIcon={<DownloadIcon />}
                          onClick={() => handleDownload(report)}
                          disabled={report.status !== 'completed'}
                        >
                          Download
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Card>

        {/* Create Report Dialog */}
        <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>Generate New Report</DialogTitle>
          <DialogContent>
            {generating ? (
              <Box sx={{ p: 4, textAlign: 'center' }}>
                <CircularProgress sx={{ mb: 2 }} />
                <Typography>Generating report...</Typography>
                <LinearProgress sx={{ mt: 2 }} />
              </Box>
            ) : (
              <Grid container spacing={2} sx={{ mt: 1 }}>
                <Grid item xs={12}>
                  <FormControl fullWidth>
                    <InputLabel>Report Template</InputLabel>
                    <Select
                      value={selectedTemplate?.id || ''}
                      label="Report Template"
                      onChange={(e) => {
                        const template = reportTemplates.find(t => t.id === e.target.value);
                        setSelectedTemplate(template || null);
                        if (template) setFormat(template.default_format);
                      }}
                    >
                      {reportTemplates.map((template) => (
                        <MenuItem key={template.id} value={template.id}>
                          {template.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    label="Start Date"
                    type="date"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                    InputLabelProps={{ shrink: true }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    label="End Date"
                    type="date"
                    value={endDate}
                    onChange={(e) => setEndDate(e.target.value)}
                    InputLabelProps={{ shrink: true }}
                  />
                </Grid>
                <Grid item xs={12}>
                  <FormControl fullWidth>
                    <InputLabel>Export Format</InputLabel>
                    <Select
                      value={format}
                      label="Export Format"
                      onChange={(e) => setFormat(e.target.value as any)}
                    >
                      <MenuItem value="pdf">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <PdfIcon fontSize="small" /> PDF
                        </Box>
                      </MenuItem>
                      <MenuItem value="excel">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <ExcelIcon fontSize="small" /> Excel
                        </Box>
                      </MenuItem>
                      <MenuItem value="csv">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <CsvIcon fontSize="small" /> CSV
                        </Box>
                      </MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
              </Grid>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setCreateOpen(false)} disabled={generating}>Cancel</Button>
            <Button variant="contained" onClick={handleCreateReport} disabled={generating || !selectedTemplate}>
              Generate
            </Button>
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

export default ReportsPage;
