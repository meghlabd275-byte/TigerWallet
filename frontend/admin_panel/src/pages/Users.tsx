import React, { useState, useEffect, useCallback } from 'react'
import { Box, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, TextField, InputAdornment, Button, Dialog, DialogTitle, DialogContent, DialogActions, Chip, Avatar, IconButton, Alert, CircularProgress } from '@mui/material'
import { Search, Visibility, Add, Refresh } from '@mui/icons-material'
import { adminFetch } from '../api'

interface AdminUser {
  id: string
  email: string
  username: string
  role: string
  kyc_status: string
  status: string
  wallet_count: number
  trades: number
  volume: string
  created_at: string
  last_login_at: string | null
}

const shortAddr = (id: string) => (id ? `${id.slice(0, 6)}...${id.slice(-4)}` : '')

const Users: React.FC = () => {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [openDialog, setOpenDialog] = useState(false)
  const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null)

  const loadUsers = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await adminFetch<AdminUser[]>('/api/v1/admin/users')
      setUsers(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users')
      setUsers([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadUsers() }, [loadUsers])

  const filteredUsers = users.filter(user =>
    user.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.id.toLowerCase().includes(searchTerm.toLowerCase())
  )

  return (
    <Box>
      <Typography variant="h4" gutterBottom>User Management</Typography>
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
        <TextField
          placeholder="Search users..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{ startAdornment: <InputAdornment position="start"><Search /></InputAdornment> }}
          sx={{ width: 300 }}
        />
        <Button variant="outlined" startIcon={<Refresh />} onClick={loadUsers} disabled={loading}>Refresh</Button>
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>User</TableCell>
              <TableCell>Email</TableCell>
              <TableCell>Role</TableCell>
              <TableCell>KYC</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Wallets</TableCell>
              <TableCell>Trades</TableCell>
              <TableCell>Volume (30d)</TableCell>
              <TableCell>Created</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow><TableCell colSpan={10} align="center"><CircularProgress size={28} sx={{ my: 4 }} /></TableCell></TableRow>
            ) : filteredUsers.length === 0 ? (
              <TableRow><TableCell colSpan={10} align="center"><Typography sx={{ py: 4 }} color="text.secondary">No users found.</Typography></TableCell></TableRow>
            ) : filteredUsers.map((user) => (
              <TableRow key={user.id}>
                <TableCell>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main' }}>{user.username[0]?.toUpperCase() || '?'}</Avatar>
                    <Box><Typography>{user.username}</Typography><Typography variant="caption" color="text.secondary">{shortAddr(user.id)}</Typography></Box>
                  </Box>
                </TableCell>
                <TableCell>{user.email}</TableCell>
                <TableCell><Chip label={user.role} size="small" /></TableCell>
                <TableCell><Chip label={user.kyc_status || 'unverified'} size="small" variant="outlined" /></TableCell>
                <TableCell><Chip label={user.status} size="small" color={user.status === 'active' ? 'success' : 'default'} /></TableCell>
                <TableCell>{user.wallet_count}</TableCell>
                <TableCell>{user.trades}</TableCell>
                <TableCell>{user.volume}</TableCell>
                <TableCell>{user.created_at ? user.created_at.slice(0, 10) : ''}</TableCell>
                <TableCell>
                  <IconButton size="small" onClick={() => { setSelectedUser(user); setOpenDialog(true) }}><Visibility /></IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={openDialog} onClose={() => setOpenDialog(false)}>
        <DialogTitle>User Details</DialogTitle>
        <DialogContent>
          {selectedUser && (
            <Box sx={{ minWidth: 400 }}>
              <Typography variant="body1"><strong>ID:</strong> {selectedUser.id}</Typography>
              <Typography variant="body1"><strong>Username:</strong> {selectedUser.username}</Typography>
              <Typography variant="body1"><strong>Email:</strong> {selectedUser.email}</Typography>
              <Typography variant="body1"><strong>Role:</strong> {selectedUser.role}</Typography>
              <Typography variant="body1"><strong>KYC:</strong> {selectedUser.kyc_status || 'unverified'}</Typography>
              <Typography variant="body1"><strong>Status:</strong> {selectedUser.status}</Typography>
              <Typography variant="body1"><strong>Wallets:</strong> {selectedUser.wallet_count}</Typography>
              <Typography variant="body1"><strong>Trades:</strong> {selectedUser.trades}</Typography>
              <Typography variant="body1"><strong>Volume (30d):</strong> {selectedUser.volume}</Typography>
              <Typography variant="body1"><strong>Created:</strong> {selectedUser.created_at}</Typography>
              <Typography variant="body1"><strong>Last login:</strong> {selectedUser.last_login_at || 'never'}</Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDialog(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default Users