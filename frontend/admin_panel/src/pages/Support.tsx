import React, { useState } from 'react';
import { 
  Box, Typography, Container, Grid, Paper, TextField, Button, 
  Chip, Dialog, DialogTitle, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Select, MenuItem, FormControl, InputLabel, IconButton, Card, CardContent,
  LinearProgress, Tabs, Tab, Snackbar, Alert, InputAdornment
} from '@mui/material';
import { 
  Add, Edit, Delete, Send, Visibility, CheckCircle, Warning, 
  Schedule, Person, Flag, Refresh
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

interface Ticket {
  id: number;
  ticket_id: string;
  user_id: number;
  user_name: string;
  subject: string;
  description: string;
  category: string;
  priority: string;
  status: string;
  assigned_to: string;
  sla_first_response_by: string;
  sla_resolution_by: string;
  first_response_at: string;
  resolved_at: string;
  created_at: string;
}

interface TicketMessage {
  id: number;
  ticket_id: string;
  sender_name: string;
  sender_type: string;
  message: string;
  is_internal: boolean;
  created_at: string;
}

interface SupportProps {
  darkMode?: boolean;
}

const Support: React.FC<SupportProps> = ({ darkMode }) => {
  const queryClient = useQueryClient();
  const [tabValue, setTabValue] = useState(0);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [messageDialogOpen, setMessageDialogOpen] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newMessage, setNewMessage] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [filterStatus, setFilterStatus] = useState('all');
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' as 'success' | 'error' });

  // New ticket form
  const [newTicket, setNewTicket] = useState({
    user_id: 1,
    user_email: '',
    user_name: '',
    subject: '',
    description: '',
    category: 'general',
    priority: 'medium',
  });

  // Fetch tickets
  const { data: ticketsData, isLoading: ticketsLoading, refetch: refetchTickets } = useQuery({
    queryKey: ['supportTickets', filterStatus],
    queryFn: async () => {
      const statusParam = filterStatus !== 'all' ? `&status=${filterStatus}` : '';
      const response = await fetch(`/api/v1/admin/support/tickets?${statusParam}`);
      if (!response.ok) throw new Error('Failed to fetch tickets');
      return response.json();
    },
  });

  // Fetch single ticket with messages
  const { data: ticketDetail, isLoading: ticketLoading } = useQuery({
    queryKey: ['ticketDetail', selectedTicket?.id],
    queryFn: async () => {
      if (!selectedTicket) return null;
      const response = await fetch(`/api/v1/admin/support/tickets/${selectedTicket.id}`);
      if (!response.ok) throw new Error('Failed to fetch ticket details');
      return response.json();
    },
    enabled: !!selectedTicket,
  });

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['supportStats'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/support/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      return response.json();
    },
  });

  // Create ticket mutation
  const createTicketMutation = useMutation({
    mutationFn: async (ticket: typeof newTicket) => {
      const response = await fetch('/api/v1/admin/support/tickets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(ticket),
      });
      if (!response.ok) throw new Error('Failed to create ticket');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['supportTickets'] });
      setCreateDialogOpen(false);
      setNewTicket({
        user_id: 1, user_email: '', user_name: '', subject: '', 
        description: '', category: 'general', priority: 'medium'
      });
      setSnackbar({ open: true, message: 'Ticket created successfully!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to create ticket', severity: 'error' });
    },
  });

  // Add message mutation
  const addMessageMutation = useMutation({
    mutationFn: async (data: { content: string; is_internal: boolean }) => {
      const response = await fetch(`/api/v1/admin/support/tickets/${selectedTicket?.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!response.ok) throw new Error('Failed to add message');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ticketDetail', selectedTicket?.id] });
      queryClient.invalidateQueries({ queryKey: ['supportTickets'] });
      setMessageDialogOpen(false);
      setNewMessage('');
      setSnackbar({ open: true, message: 'Message sent!', severity: 'success' });
    },
    onError: () => {
      setSnackbar({ open: true, message: 'Failed to send message', severity: 'error' });
    },
  });

  // Update ticket status mutation
  const updateTicketMutation = useMutation({
    mutationFn: async (data: { status: string; priority?: string }) => {
      const response = await fetch(`/api/v1/admin/support/tickets/${selectedTicket?.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!response.ok) throw new Error('Failed to update ticket');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ticketDetail', selectedTicket?.id] });
      queryClient.invalidateQueries({ queryKey: ['supportTickets'] });
      setSnackbar({ open: true, message: 'Ticket updated!', severity: 'success' });
    },
  });

  const tickets = ticketsData?.data || [];
  const messages = ticketDetail?.messages || [];

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent': return 'error';
      case 'high': return 'warning';
      case 'medium': return 'info';
      default: return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'open': return 'error';
      case 'in_progress': return 'warning';
      case 'resolved': return 'success';
      case 'closed': return 'default';
      default: return 'default';
    }
  };

  const isSLABreached = (ticket: Ticket) => {
    if (!ticket.sla_resolution_by) return false;
    return new Date(ticket.sla_resolution_by) < new Date() && 
           ticket.status !== 'resolved' && ticket.status !== 'closed';
  };

  const cardBg = darkMode ? '#1a1a1a' : '#fff';
  const textPrimary = darkMode ? '#fff' : '#000';
  const textSecondary = darkMode ? '#aaa' : '#666';

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
            Support Center
          </Typography>
          <Button 
            variant="contained" 
            startIcon={<Add />}
            onClick={() => setCreateDialogOpen(true)}
          >
            New Ticket
          </Button>
        </Box>

        {/* Stats Cards */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Open Tickets</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats?.open_tickets || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>In Progress</Typography>
                <Typography variant="h4" sx={{ color: '#f97316' }}>{stats?.in_progress_tickets || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>SLA Breached</Typography>
                <Typography variant="h4" sx={{ color: 'error.main' }}>{stats?.breached_sla || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="body2" sx={{ color: textSecondary }}>Avg Response</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>
                  {(stats?.avg_response_time || 0).toFixed(1)}h
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Filter Tabs */}
        <Paper sx={{ mb: 2, bgcolor: cardBg }}>
          <Tabs 
            value={tabValue} 
            onChange={(_, v) => setTabValue(v)}
            sx={{ 
              '& .MuiTab-root': { color: textSecondary },
              '& .Mui-selected': { color: '#f97316' }
            }}
          >
            <Tab label="All Tickets" />
            <Tab label="My Tickets" />
            <Tab label="Unassigned" />
          </Tabs>
        </Paper>

        {/* Filter */}
        <Box sx={{ mb: 2, display: 'flex', gap: 2 }}>
          <FormControl size="small" sx={{ minWidth: 150 }}>
            <InputLabel>Status</InputLabel>
            <Select
              value={filterStatus}
              label="Status"
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <MenuItem value="all">All</MenuItem>
              <MenuItem value="open">Open</MenuItem>
              <MenuItem value="in_progress">In Progress</MenuItem>
              <MenuItem value="resolved">Resolved</MenuItem>
              <MenuItem value="closed">Closed</MenuItem>
            </Select>
          </FormControl>
          <Button 
            startIcon={<Refresh />} 
            onClick={() => refetchTickets()}
          >
            Refresh
          </Button>
        </Box>

        {/* Tickets Table */}
        <Paper sx={{ bgcolor: cardBg }}>
          {ticketsLoading ? (
            <LinearProgress />
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: textSecondary }}>Ticket ID</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Subject</TableCell>
                    <TableCell sx={{ color: textSecondary }}>User</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Category</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Priority</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                    <TableCell sx={{ color: textSecondary }}>SLA</TableCell>
                    <TableCell sx={{ color: textSecondary }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {tickets.map((ticket: Ticket) => (
                    <TableRow key={ticket.id} hover>
                      <TableCell sx={{ color: textPrimary }}>{ticket.ticket_id}</TableCell>
                      <TableCell sx={{ color: textPrimary, maxWidth: 200 }}>
                        <Typography noWrap>{ticket.subject}</Typography>
                      </TableCell>
                      <TableCell sx={{ color: textPrimary }}>{ticket.user_name}</TableCell>
                      <TableCell sx={{ color: textPrimary }}>
                        <Chip label={ticket.category} size="small" />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={ticket.priority} 
                          size="small" 
                          color={getPriorityColor(ticket.priority)} 
                        />
                      </TableCell>
                      <TableCell>
                        <Chip 
                          label={ticket.status.replace('_', ' ')} 
                          size="small" 
                          color={getStatusColor(ticket.status)} 
                        />
                      </TableCell>
                      <TableCell>
                        {isSLABreached(ticket) ? (
                          <Chip icon={<Warning />} label="Breached" color="error" size="small" />
                        ) : (
                          <Chip icon={<Schedule />} label="On Track" color="success" size="small" />
                        )}
                      </TableCell>
                      <TableCell>
                        <IconButton 
                          size="small" 
                          onClick={() => setSelectedTicket(ticket)}
                        >
                          <Visibility />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Paper>

        {/* Ticket Detail Dialog */}
        <Dialog 
          open={!!selectedTicket} 
          onClose={() => setSelectedTicket(null)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle sx={{ bgcolor: cardBg, borderBottom: `1px solid ${darkMode ? '#333' : '#eee'}` }}>
            {selectedTicket?.subject}
          </DialogTitle>
          <DialogContent sx={{ bgcolor: cardBg }}>
            {ticketLoading ? (
              <LinearProgress />
            ) : (
              <Box>
                <Box sx={{ mb: 2, display: 'flex', gap: 1 }}>
                  <Chip label={selectedTicket?.priority} color={getPriorityColor(selectedTicket?.priority || '')} />
                  <Chip label={selectedTicket?.status?.replace('_', ' ')} color={getStatusColor(selectedTicket?.status || '')} />
                  {isSLABreached(selectedTicket!) && (
                    <Chip icon={<Warning />} label="SLA Breached" color="error" />
                  )}
                </Box>

                <Typography variant="body2" sx={{ color: textSecondary, mb: 1 }}>
                  Created: {selectedTicket?.created_at}
                </Typography>

                <Divider sx={{ my: 2 }} />

                {/* Messages */}
                {messages.map((msg: TicketMessage, idx: number) => (
                  <Box 
                    key={idx} 
                    sx={{ 
                      mb: 2, 
                      p: 2, 
                      borderRadius: 1,
                      bgcolor: msg.sender_type === 'admin' ? (darkMode ? '#222' : '#f5f5f5') : (darkMode ? '#1a1a1a' : '#fff'),
                      border: msg.is_internal ? '1px solid orange' : 'none',
                    }}
                  >
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                      <Typography variant="subtitle2" sx={{ color: textPrimary }}>
                        {msg.sender_name} ({msg.sender_type})
                      </Typography>
                      {msg.is_internal && <Chip label="Internal" size="small" color="warning" />}
                    </Box>
                    <Typography variant="body2" sx={{ color: textPrimary }}>{msg.message}</Typography>
                    <Typography variant="caption" sx={{ color: textSecondary }}>
                      {msg.created_at}
                    </Typography>
                  </Box>
                ))}

                {/* Quick Actions */}
                <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
                  <Button 
                    variant="contained" 
                    startIcon={<Send />}
                    onClick={() => setMessageDialogOpen(true)}
                  >
                    Reply
                  </Button>
                  <Button 
                    variant="outlined"
                    onClick={() => updateTicketMutation.mutate({ status: 'in_progress' })}
                  >
                    Start Work
                  </Button>
                  <Button 
                    variant="outlined" 
                    color="success"
                    startIcon={<CheckCircle />}
                    onClick={() => updateTicketMutation.mutate({ status: 'resolved' })}
                  >
                    Resolve
                  </Button>
                </Box>
              </Box>
            )}
          </DialogContent>
          <DialogActions sx={{ bgcolor: cardBg, borderTop: `1px solid ${darkMode ? '#333' : '#eee'}` }}>
            <Button onClick={() => setSelectedTicket(null)}>Close</Button>
          </DialogActions>
        </Dialog>

        {/* Reply Dialog */}
        <Dialog open={messageDialogOpen} onClose={() => setMessageDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>Reply to Ticket</DialogTitle>
          <DialogContent>
            <TextField
              fullWidth
              multiline
              rows={4}
              label="Message"
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              sx={{ mt: 2 }}
            />
            <FormControlLabel
              control={
                <Switch 
                  checked={isInternal} 
                  onChange={(e) => setIsInternal(e.target.checked)} 
                />
              }
              label="Internal Note (not visible to user)"
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setMessageDialogOpen(false)}>Cancel</Button>
            <Button 
              variant="contained" 
              onClick={() => addMessageMutation.mutate({ content: newMessage, is_internal: isInternal })}
              disabled={!newMessage.trim()}
            >
              Send
            </Button>
          </DialogActions>
        </Dialog>

        {/* Create Ticket Dialog */}
        <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>Create New Ticket</DialogTitle>
          <DialogContent>
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="User Email"
                  value={newTicket.user_email}
                  onChange={(e) => setNewTicket({...newTicket, user_email: e.target.value})}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Subject"
                  value={newTicket.subject}
                  onChange={(e) => setNewTicket({...newTicket, subject: e.target.value})}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  multiline
                  rows={3}
                  label="Description"
                  value={newTicket.description}
                  onChange={(e) => setNewTicket({...newTicket, description: e.target.value})}
                />
              </Grid>
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Category</InputLabel>
                  <Select
                    value={newTicket.category}
                    label="Category"
                    onChange={(e) => setNewTicket({...newTicket, category: e.target.value})}
                  >
                    <MenuItem value="general">General</MenuItem>
                    <MenuItem value="technical">Technical</MenuItem>
                    <MenuItem value="billing">Billing</MenuItem>
                    <MenuItem value="kyc">KYC</MenuItem>
                    <MenuItem value="withdrawal">Withdrawal</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Priority</InputLabel>
                  <Select
                    value={newTicket.priority}
                    label="Priority"
                    onChange={(e) => setNewTicket({...newTicket, priority: e.target.value})}
                  >
                    <MenuItem value="low">Low</MenuItem>
                    <MenuItem value="medium">Medium</MenuItem>
                    <MenuItem value="high">High</MenuItem>
                    <MenuItem value="urgent">Urgent</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
            <Button 
              variant="contained" 
              onClick={() => createTicketMutation.mutate(newTicket)}
              disabled={!newTicket.subject || !newTicket.description}
            >
              Create Ticket
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

export default Support;
