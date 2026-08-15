import React, { useState, useEffect, useCallback } from 'react';
import {
  Container, Grid, Card, Button, TextField, Select, MenuItem, FormControl, InputLabel,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TablePagination,
  Dialog, DialogTitle, DialogContent, DialogActions, Chip, Typography, Box, Avatar,
  IconButton, CircularProgress, Alert, Snackbar, InputAdornment
} from '@mui/material';
import {
  Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon, Search as SearchIcon,
  Visibility as ViewIcon, DarkMode, LightMode, AttachMoney as MoneyIcon
} from '@mui/icons-material';
import { api, Product, PaginatedResponse } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const ProductManagement: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | 'view'>('create');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [total, setTotal] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [snackbar, setSnackbar] = useState<{open: boolean; message: string; severity: 'success' | 'error'}>({
    open: false, message: '', severity: 'success'
  });

  const [formData, setFormData] = useState({
    name: '',
    type: 'trading' as Product['type'],
    description: '',
    status: 'enabled' as Product['status'],
    fee: 0,
    minDeposit: 0,
    maxDeposit: 1000000,
    minWithdrawal: 0,
    maxWithdrawal: 1000000,
    features: [] as string[],
    sortOrder: 0
  });

  const fetchProducts = useCallback(async () => {
    try {
      setLoading(true);
      const response: PaginatedResponse<Product> = await api.getProducts({
        page: page + 1,
        pageSize: rowsPerPage,
        query: searchQuery,
        status: statusFilter || undefined
      });
      setProducts(response.data);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch products:', error);
      setSnackbar({ open: true, message: 'Failed to fetch products', severity: 'error' });
    } finally {
      setLoading(false);
    }
  }, [page, rowsPerPage, searchQuery, statusFilter]);

  useEffect(() => {
    fetchProducts();
  }, [fetchProducts]);

  const handleCreateProduct = () => {
    setSelectedProduct(null);
    setFormData({
      name: '',
      type: 'trading',
      description: '',
      status: 'enabled',
      fee: 0,
      minDeposit: 0,
      maxDeposit: 1000000,
      minWithdrawal: 0,
      maxWithdrawal: 1000000,
      features: [],
      sortOrder: 0
    });
    setDialogMode('create');
    setOpenDialog(true);
  };

  const handleEditProduct = (product: Product) => {
    setSelectedProduct(product);
    setFormData({
      name: product.name,
      type: product.type,
      description: product.description || '',
      status: product.status,
      fee: product.fee,
      minDeposit: product.minDeposit,
      maxDeposit: product.maxDeposit,
      minWithdrawal: product.minWithdrawal,
      maxWithdrawal: product.maxWithdrawal,
      features: product.features || [],
      sortOrder: product.sortOrder
    });
    setDialogMode('edit');
    setOpenDialog(true);
  };

  const handleViewProduct = (product: Product) => {
    setSelectedProduct(product);
    setDialogMode('view');
    setOpenDialog(true);
  };

  const handleDeleteProduct = async (productId: string) => {
    if (!confirm('Are you sure you want to delete this product?')) return;
    try {
      await api.deleteProduct(productId);
      setSnackbar({ open: true, message: 'Product deleted successfully', severity: 'success' });
      fetchProducts();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to delete product', severity: 'error' });
    }
  };

  const handleToggleProduct = async (product: Product) => {
    try {
      await api.toggleProduct(product.id);
      setSnackbar({ open: true, message: `Product ${product.status === 'enabled' ? 'disabled' : 'enabled'} successfully`, severity: 'success' });
      fetchProducts();
    } catch (error) {
      setSnackbar({ open: true, message: 'Failed to toggle product', severity: 'error' });
    }
  };

  const handleSubmit = async () => {
    try {
      if (dialogMode === 'create') {
        await api.createProduct(formData);
        setSnackbar({ open: true, message: 'Product created successfully', severity: 'success' });
      } else if (dialogMode === 'edit' && selectedProduct) {
        await api.updateProduct(selectedProduct.id, formData);
        setSnackbar({ open: true, message: 'Product updated successfully', severity: 'success' });
      }
      setOpenDialog(false);
      fetchProducts();
    } catch (error) {
      setSnackbar({ open: true, message: 'Operation failed', severity: 'error' });
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'trading': return 'primary';
      case 'perpetual': return 'secondary';
      case 'staking': return 'success';
      case 'nft': return 'warning';
      case 'wallet': return 'info';
      case 'bridge': return 'error';
      case 'launchpad': return 'default';
      default: return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'enabled': return 'success';
      case 'disabled': return 'error';
      case 'maintenance': return 'warning';
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
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
          <Typography variant="h4" fontWeight="bold">
            Product Management
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <IconButton onClick={toggleTheme} color="primary">
              {theme === 'dark' ? <LightMode /> : <DarkMode />}
            </IconButton>
            <Button variant="contained" startIcon={<AddIcon />} onClick={handleCreateProduct}>
              Create Product
            </Button>
          </Box>
        </Box>

        {/* Filters */}
        <Card sx={{ p: 2, mb: 3 }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                placeholder="Search products by name..."
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
            </Grid>
            <Grid item xs={12} md={3}>
              <FormControl fullWidth>
                <InputLabel>Status</InputLabel>
                <Select
                  value={statusFilter}
                  label="Status"
                  onChange={(e) => setStatusFilter(e.target.value)}
                >
                  <MenuItem value="">All</MenuItem>
                  <MenuItem value="enabled">Enabled</MenuItem>
                  <MenuItem value="disabled">Disabled</MenuItem>
                  <MenuItem value="maintenance">Maintenance</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </Card>

        {/* Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="primary">{total}</Typography>
              <Typography variant="body2" color="text.secondary">Total Products</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="success.main">{products.filter(p => p.status === 'enabled').length}</Typography>
              <Typography variant="body2" color="text.secondary">Enabled</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="info.main">
                {products.reduce((sum, p) => sum + p.fee, 0).toFixed(2)}%
              </Typography>
              <Typography variant="body2" color="text.secondary">Total Fees</Typography>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h3" color="warning.main">
                ${products.reduce((sum, p) => sum + p.maxDeposit, 0).toLocaleString()}
              </Typography>
              <Typography variant="body2" color="text.secondary">Max Deposit</Typography>
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
                    <TableCell>Product</TableCell>
                    <TableCell>Type</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Fee (%)</TableCell>
                    <TableCell>Min Deposit</TableCell>
                    <TableCell>Max Deposit</TableCell>
                    <TableCell>Sort Order</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {products.map((product) => (
                    <TableRow key={product.id} hover>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                          <Avatar sx={{ bgcolor: 'primary.main' }}>
                            {product.name[0].toUpperCase()}
                          </Avatar>
                          <Typography fontWeight="medium">{product.name}</Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label={product.type} color={getTypeColor(product.type) as any} size="small" />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={product.status} 
                          color={getStatusColor(product.status) as any} 
                          size="small" 
                          onClick={() => handleToggleProduct(product)}
                          clickable
                        />
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          <MoneyIcon fontSize="small" color="action" />
                          {product.fee}%
                        </Box>
                      </TableCell>
                      <TableCell>${product.minDeposit.toLocaleString()}</TableCell>
                      <TableCell>${product.maxDeposit.toLocaleString()}</TableCell>
                      <TableCell>{product.sortOrder}</TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={() => handleViewProduct(product)}>
                          <ViewIcon />
                        </IconButton>
                        <IconButton size="small" onClick={() => handleEditProduct(product)}>
                          <EditIcon />
                        </IconButton>
                        <IconButton size="small" color="error" onClick={() => handleDeleteProduct(product.id)}>
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
            onPageChange={(_event, newPage) => setPage(newPage)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(event) => {
              setRowsPerPage(parseInt(event.target.value, 10));
              setPage(0);
            }}
          />
        </Card>

        {/* Dialog */}
        <Dialog open={openDialog} onClose={() => setOpenDialog(false)} maxWidth="md" fullWidth>
          <DialogTitle>
            {dialogMode === 'create' && 'Create New Product'}
            {dialogMode === 'edit' && 'Edit Product'}
            {dialogMode === 'view' && 'Product Details'}
          </DialogTitle>
          <DialogContent>
            {dialogMode === 'view' && selectedProduct ? (
              <Box sx={{ pt: 2 }}>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={6}>
                    <Typography><strong>Name:</strong> {selectedProduct.name}</Typography>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography><strong>Type:</strong> {selectedProduct.type}</Typography>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography><strong>Status:</strong> {selectedProduct.status}</Typography>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography><strong>Fee:</strong> {selectedProduct.fee}%</Typography>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography><strong>Min Deposit:</strong> ${selectedProduct.minDeposit}</Typography>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <Typography><strong>Max Deposit:</strong> ${selectedProduct.maxDeposit}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Description:</strong> {selectedProduct.description || 'N/A'}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography><strong>Features:</strong> {selectedProduct.features?.join(', ') || 'None'}</Typography>
                  </Grid>
                </Grid>
              </Box>
            ) : (
              <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Grid container spacing={2}>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Product Name"
                      fullWidth
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      required
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <FormControl fullWidth>
                      <InputLabel>Type</InputLabel>
                      <Select
                        value={formData.type}
                        label="Type"
                        onChange={(e) => setFormData({ ...formData, type: e.target.value as Product['type'] })}
                      >
                        <MenuItem value="trading">Spot Trading</MenuItem>
                        <MenuItem value="perpetual">Perpetual Trading</MenuItem>
                        <MenuItem value="staking">Staking</MenuItem>
                        <MenuItem value="nft">NFT Marketplace</MenuItem>
                        <MenuItem value="wallet">Wallet</MenuItem>
                        <MenuItem value="bridge">Bridge</MenuItem>
                        <MenuItem value="launchpad">Launchpad</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <FormControl fullWidth>
                      <InputLabel>Status</InputLabel>
                      <Select
                        value={formData.status}
                        label="Status"
                        onChange={(e) => setFormData({ ...formData, status: e.target.value as Product['status'] })}
                      >
                        <MenuItem value="enabled">Enabled</MenuItem>
                        <MenuItem value="disabled">Disabled</MenuItem>
                        <MenuItem value="maintenance">Maintenance</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Fee (%)"
                      type="number"
                      fullWidth
                      value={formData.fee}
                      onChange={(e) => setFormData({ ...formData, fee: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Min Deposit"
                      type="number"
                      fullWidth
                      value={formData.minDeposit}
                      onChange={(e) => setFormData({ ...formData, minDeposit: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Max Deposit"
                      type="number"
                      fullWidth
                      value={formData.maxDeposit}
                      onChange={(e) => setFormData({ ...formData, maxDeposit: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Min Withdrawal"
                      type="number"
                      fullWidth
                      value={formData.minWithdrawal}
                      onChange={(e) => setFormData({ ...formData, minWithdrawal: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Max Withdrawal"
                      type="number"
                      fullWidth
                      value={formData.maxWithdrawal}
                      onChange={(e) => setFormData({ ...formData, maxWithdrawal: parseFloat(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      label="Sort Order"
                      type="number"
                      fullWidth
                      value={formData.sortOrder}
                      onChange={(e) => setFormData({ ...formData, sortOrder: parseInt(e.target.value) || 0 })}
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      label="Description"
                      fullWidth
                      multiline
                      rows={3}
                      value={formData.description}
                      onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    />
                  </Grid>
                </Grid>
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

export default ProductManagement;
