/**
 * TigerWallet Admin Platform - Batch Operations Page
 * Complete batch operations for users, transactions, tokens
 * Dark/Light theme works everywhere
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Container, Grid, Card, TextField, Typography, Button, Select, MenuItem,
  FormControl, InputLabel, CircularProgress, Alert, Snackbar, Chip, Table,
  TableBody, TableCell, TableContainer, TableHead, TableRow, Dialog,
  DialogTitle, DialogContent, DialogActions, Checkbox, LinearProgress,
  Stepper, Step, StepLabel, IconButton, Tab, Tabs
} from '@mui/material';
import {
  BatchPrediction as BatchIcon, People as UsersIcon, SwapHoriz as TransactionIcon,
  AttachMoney as TokenIcon, CheckCircle, Cancel, Warning, Refresh as RefreshIcon,
  DarkMode, LightMode, Upload as UploadIcon, Download as DownloadIcon,
  Delete as DeleteIcon, Edit as EditIcon, Send as SendIcon
} from '@mui/icons-material';
import { useTheme } from '../../contexts/ThemeContext';

// ============================================================================
// Types
// ============================================================================

interface BatchOperation {
  id: string;
  type: 'user' | 'transaction' | 'token' | 'fee';
  action: string;
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'partial';
  total_count: number;
  success_count: number;
  failed_count: number;
  created_by: string;
  created_at: string;
  completed_at?: string;
  details?: string;
}

interface BatchItem {
  id: string;
  selected: boolean;
  data: Record<string, any>;
  status: 'pending' | 'success' | 'failed';
  error?: string;
}

// ============================================================================
// Mock Data
// ============================================================================

const mockBatchOperations: BatchOperation[] = [
  { id: '1', type: 'user', action: 'SUSPEND', status: 'completed', total_count: 50, success_count: 48, failed_count: 2, created_by: 'admin@tigerwallet.com', created_at: '2024-01-15T10:00:00Z', completed_at: '2024-01-15T10:05:00Z', details: 'Suspended 48 users' },
  { id: '2', type: 'transaction', action: 'APPROVE', status: 'processing', total_count: 100, success_count: 75, failed_count: 0, created_by: 'superadmin@tigerwallet.com', created_at: '2024-01-15T11:00:00Z' },
  { id: '3', type: 'token', action: 'LIST', status: 'completed', total_count: 25, success_count: 25, failed_count: 0, created_by: 'admin@tigerwallet.com', created_at: '2024-01-15T09:00:00Z', completed_at: '2024-01-15T09:02:00Z', details: 'Listed 25 new tokens' },
  { id: '4', type: 'fee', action: 'UPDATE', status: 'failed', total_count: 10, success_count: 0, failed_count: 10, created_by: 'admin@tigerwallet.com', created_at: '2024-01-14T15:00:00Z', completed_at: '2024-01-14T15:01:00Z', details: 'Failed - Invalid fee structure' },
  { id: '5', type: 'user', action: 'KYC_APPROVE', status: 'partial', total_count: 200, success_count: 180, failed_count: 20, created_by: 'superadmin@tigerwallet.com', created_at: '2024-01-14T10:00:00Z', completed_at: '2024-01-14T10:10:00Z', details: '180 approved, 20 requires review' },
];

// ============================================================================
// Main Component
// ============================================================================

const BatchOperationsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const isDark = theme === 'dark';
  
  const [activeTab, setActiveTab] = useState(0);
  const [operations, setOperations] = useState<BatchOperation[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedItems, setSelectedItems] = useState<BatchItem[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [processing, setProcessing] = useState(false);
  const [activeStep, setActiveStep] = useState(0);
  const [batchType, setBatchType] = useState<'user' | 'transaction' | 'token' | 'fee'>('user');
  const [batchAction, setBatchAction] = useState('');
  const [csvData, setCsvData] = useState('');
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' | 'info' }>({ open: false, message: '', severity: 'info' });

  const loadOperations = useCallback(async () => {
    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 500));
    setOperations(mockBatchOperations);
    setLoading(false);
  }, []);

  useEffect(() => {
    loadOperations();
  }, [loadOperations]);

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const text = e.target?.result as string;
        setCsvData(text);
        
        // Parse CSV
        const lines = text.split('\n').filter(line => line.trim());
        const headers = lines[0].split(',').map(h => h.trim());
        
        const items: BatchItem[] = lines.slice(1).map((line, index) => {
          const values = line.split(',');
          const data: Record<string, any> = {};
          headers.forEach((header, i) => {
            data[header] = values[i]?.trim();
          });
          return {
            id: `item-${index}`,
            selected: false,
            data,
            status: 'pending',
          };
        });
        
        setSelectedItems(items);
      };
      reader.readAsText(file);
    }
  };

  const handleToggleSelectAll = () => {
    if (selectedItems.every(item => item.selected)) {
      setSelectedItems(selectedItems.map(item => ({ ...item, selected: false })));
    } else {
      setSelectedItems(selectedItems.map(item => ({ ...item, selected: true })));
    }
  };

  const handleToggleItem = (id: string) => {
    setSelectedItems(selectedItems.map(item => 
      item.id === id ? { ...item, selected: !item.selected } : item
    ));
  };

  const handleProcessBatch = async () => {
    setProcessing(true);
    const selected = selectedItems.filter(item => item.selected);
    
    // Simulate processing
    const processed = selected.map(item => ({
      ...item,
      status: Math.random() > 0.1 ? 'success' : 'failed',
      error: Math.random() > 0.1 ? undefined : 'Processing error',
    }));
    
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    setSelectedItems(selectedItems.map(item => {
      const processedItem = processed.find(p => p.id === item.id);
      return processedItem || item;
    }));
    
    setProcessing(false);
    setActiveStep(2);
    setSnackbar({ open: true, message: 'Batch operation completed!', severity: 'success' });
    
    // Add to operations list
    const newOperation: BatchOperation = {
      id: Date.now().toString(),
      type: batchType,
      action: batchAction,
      status: 'completed',
      total_count: selected.length,
      success_count: processed.filter(p => p.status === 'success').length,
      failed_count: processed.filter(p => p.status === 'failed').length,
      created_by: 'current_admin@tigerwallet.com',
      created_at: new Date().toISOString(),
      completed_at: new Date().toISOString(),
    };
    
    setOperations([newOperation, ...operations]);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'processing': return 'warning';
      case 'failed': return 'error';
      case 'partial': return 'info';
      default: return 'default';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'user': return <UsersIcon />;
      case 'transaction': return <TransactionIcon />;
      case 'token': return <TokenIcon />;
      default: return <BatchIcon />;
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
            <BatchIcon sx={{ fontSize: 40, color: 'primary.main' }} />
            <Typography variant="h4" fontWeight="bold">
              Batch Operations
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              startIcon={<SendIcon />}
              onClick={() => {
                setActiveStep(0);
                setCreateOpen(true);
              }}
            >
              New Batch Operation
            </Button>
            <IconButton onClick={toggleTheme} color="primary">
              {isDark ? <LightMode /> : <DarkMode />}
            </IconButton>
          </Box>
        </Box>

        {/* Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Total Operations</Typography>
              <Typography variant="h3" fontWeight="bold" color="primary">{operations.length}</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Completed</Typography>
              <Typography variant="h3" fontWeight="bold" color="success.main">
                {operations.filter(o => o.status === 'completed').length}
              </Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Processing</Typography>
              <Typography variant="h3" fontWeight="bold" color="warning.main">
                {operations.filter(o => o.status === 'processing').length}
              </Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
              <Typography variant="body2" color="text.secondary">Total Processed</Typography>
              <Typography variant="h3" fontWeight="bold">
                {operations.reduce((acc, o) => acc + o.total_count, 0).toLocaleString()}
              </Typography>
            </Card>
          </Grid>
        </Grid>

        {/* Tabs */}
        <Card sx={{ mb: 3, bgcolor: isDark ? 'grey.900' : 'white' }}>
          <Tabs 
            value={activeTab} 
            onChange={(_, v) => setActiveTab(v)}
            sx={{ borderBottom: 1, borderColor: 'divider' }}
          >
            <Tab label="All Operations" />
            <Tab label="User Operations" />
            <Tab label="Transaction Operations" />
            <Tab label="Token Operations" />
          </Tabs>
        </Card>

        {/* Operations Table */}
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
                    <TableCell>Type</TableCell>
                    <TableCell>Action</TableCell>
                    <TableCell>Progress</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Created By</TableCell>
                    <TableCell>Created At</TableCell>
                    <TableCell>Details</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {operations
                    .filter(op => activeTab === 0 || 
                      (activeTab === 1 && op.type === 'user') ||
                      (activeTab === 2 && op.type === 'transaction') ||
                      (activeTab === 3 && op.type === 'token'))
                    .map((op) => (
                    <TableRow key={op.id} hover>
                      <TableCell>{op.id}</TableCell>
                      <TableCell>
                        <Chip icon={getTypeIcon(op.type)} label={op.type} size="small" />
                      </TableCell>
                      <TableCell>
                        <Chip label={op.action} variant="outlined" size="small" />
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 150 }}>
                          <LinearProgress 
                            variant="determinate" 
                            value={(op.success_count / op.total_count) * 100}
                            sx={{ flex: 1, height: 8, borderRadius: 4 }}
                          />
                          <Typography variant="caption">
                            {op.success_count}/{op.total_count}
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label={op.status} color={getStatusColor(op.status) as any} size="small" />
                      </TableCell>
                      <TableCell>{op.created_by}</TableCell>
                      <TableCell>{formatDate(op.created_at)}</TableCell>
                      <TableCell sx={{ maxWidth: 200 }}>
                        {op.details || '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Card>

        {/* Create Batch Dialog */}
        <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle>Create Batch Operation</DialogTitle>
          <DialogContent>
            <Stepper activeStep={activeStep} sx={{ my: 3 }}>
              <Step>
                <StepLabel>Select Type</StepLabel>
              </Step>
              <Step>
                <StepLabel>Upload Data</StepLabel>
              </Step>
              <Step>
                <StepLabel>Review & Process</StepLabel>
              </Step>
            </Stepper>

            {activeStep === 0 && (
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <FormControl fullWidth>
                    <InputLabel>Operation Type</InputLabel>
                    <Select
                      value={batchType}
                      label="Operation Type"
                      onChange={(e) => setBatchType(e.target.value as any)}
                    >
                      <MenuItem value="user">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <UsersIcon /> User Operations
                        </Box>
                      </MenuItem>
                      <MenuItem value="transaction">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <TransactionIcon /> Transaction Operations
                        </Box>
                      </MenuItem>
                      <MenuItem value="token">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <TokenIcon /> Token Operations
                        </Box>
                      </MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12}>
                  <FormControl fullWidth>
                    <InputLabel>Action</InputLabel>
                    <Select
                      value={batchAction}
                      label="Action"
                      onChange={(e) => setBatchAction(e.target.value)}
                    >
                      {batchType === 'user' && (
                        <>
                          <MenuItem value="SUSPEND">Suspend Users</MenuItem>
                          <MenuItem value="BAN">Ban Users</MenuItem>
                          <MenuItem value="KYC_APPROVE">Approve KYC</MenuItem>
                          <MenuItem value="KYC_REJECT">Reject KYC</MenuItem>
                          <MenuItem value="EMAIL_SEND">Send Email</MenuItem>
                        </>
                      )}
                      {batchType === 'transaction' && (
                        <>
                          <MenuItem value="APPROVE">Approve Transactions</MenuItem>
                          <MenuItem value="REJECT">Reject Transactions</MenuItem>
                          <MenuItem value="CANCEL">Cancel Transactions</MenuItem>
                        </>
                      )}
                      {batchType === 'token' && (
                        <>
                          <MenuItem value="LIST">List Tokens</MenuItem>
                          <MenuItem value="DELIST">Delist Tokens</MenuItem>
                          <MenuItem value="UPDATE">Update Token Info</MenuItem>
                        </>
                      )}
                    </Select>
                  </FormControl>
                </Grid>
              </Grid>
            )}

            {activeStep === 1 && (
              <Box>
                <Box sx={{ mb: 2, display: 'flex', gap: 1 }}>
                  <Button
                    variant="outlined"
                    component="label"
                    startIcon={<UploadIcon />}
                  >
                    Upload CSV
                    <input type="file" hidden accept=".csv" onChange={handleFileUpload} />
                  </Button>
                  <Button
                    variant="outlined"
                    onClick={() => {
                      // Generate sample CSV
                      const sampleData = batchType === 'user' 
                        ? 'user_id,email,action\n1,user1@example.com,suspend\n2,user2@example.com,suspend'
                        : batchType === 'transaction'
                        ? 'tx_id,action\ntx1,approve\ntx2,approve'
                        : 'token_id,symbol,action\n1,ETH,list\n2,BTC,list';
                      setCsvData(sampleData);
                      setSelectedItems([
                        { id: '1', selected: true, data: { id: '1' }, status: 'pending' },
                        { id: '2', selected: true, data: { id: '2' }, status: 'pending' },
                      ]);
                    }}
                  >
                    Use Sample Data
                  </Button>
                </Box>
                
                {selectedItems.length > 0 && (
                  <TableContainer sx={{ maxHeight: 300 }}>
                    <Table size="small" stickyHeader>
                      <TableHead>
                        <TableRow>
                          <TableCell padding="checkbox">
                            <Checkbox
                              checked={selectedItems.every(i => i.selected)}
                              indeterminate={selectedItems.some(i => i.selected) && !selectedItems.every(i => i.selected)}
                              onChange={handleToggleSelectAll}
                            />
                          </TableCell>
                          <TableCell>ID</TableCell>
                          <TableCell>Data</TableCell>
                          <TableCell>Status</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {selectedItems.map((item) => (
                          <TableRow key={item.id}>
                            <TableCell padding="checkbox">
                              <Checkbox
                                checked={item.selected}
                                onChange={() => handleToggleItem(item.id)}
                              />
                            </TableCell>
                            <TableCell>{item.id}</TableCell>
                            <TableCell>{JSON.stringify(item.data)}</TableCell>
                            <TableCell>
                              <Chip 
                                label={item.status} 
                                color={item.status === 'success' ? 'success' : item.status === 'failed' ? 'error' : 'default'}
                                size="small" 
                              />
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                )}
                
                <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                  {selectedItems.filter(i => i.selected).length} of {selectedItems.length} items selected
                </Typography>
              </Box>
            )}

            {activeStep === 2 && (
              <Box>
                <Alert severity="info" sx={{ mb: 2 }}>
                  You are about to perform a batch {batchAction} operation on {selectedItems.filter(i => i.selected).length} items.
                  This action cannot be undone.
                </Alert>
                {processing && (
                  <Box sx={{ mb: 2 }}>
                    <Typography variant="body2" gutterBottom>Processing...</Typography>
                    <LinearProgress />
                  </Box>
                )}
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
            {activeStep > 0 && (
              <Button onClick={() => setActiveStep(activeStep - 1)}>Back</Button>
            )}
            {activeStep < 2 ? (
              <Button 
                variant="contained" 
                onClick={() => setActiveStep(activeStep + 1)}
                disabled={
                  (activeStep === 0 && (!batchType || !batchAction)) ||
                  (activeStep === 1 && selectedItems.filter(i => i.selected).length === 0)
                }
              >
                Next
              </Button>
            ) : (
              <Button 
                variant="contained" 
                color="primary"
                onClick={handleProcessBatch}
                disabled={processing}
                startIcon={<SendIcon />}
              >
                Process Batch
              </Button>
            )}
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

export default BatchOperationsPage;
