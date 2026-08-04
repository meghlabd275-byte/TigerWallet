import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { ticketService } from '../services/api';

interface Ticket {
  id: number;
  ticket_id: string;
  subject: string;
  description: string;
  category: string;
  priority: string;
  status: string;
  user_id: number;
  created_at: string;
  updated_at: string;
}

interface TicketMessage {
  id: number;
  message: string;
  sender_type: string;
  created_at: string;
}

export default function Tickets() {
  const { isDark } = useTheme();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [messages, setMessages] = useState<TicketMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [filter, setFilter] = useState({ status: '', priority: '', category: '' });
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newTicket, setNewTicket] = useState({ subject: '', description: '', category: 'general', priority: 'medium' });

  useEffect(() => {
    loadTickets();
  }, [filter]);

  const loadTickets = async () => {
    setLoading(true);
    setError('');
    try {
      const params: any = {};
      if (filter.status) params.status = filter.status;
      if (filter.priority) params.priority = filter.priority;
      if (filter.category) params.category = filter.category;
      
      const response = await ticketService.getTickets(params);
      setTickets(response.data);
    } catch (err: any) {
      setError(err.message || 'Failed to load tickets');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectTicket = async (ticket: Ticket) => {
    setSelectedTicket(ticket);
    try {
      const response = await ticketService.getTicketMessages(ticket.id);
      setMessages(response.data);
    } catch (err) {
      console.error('Failed to load messages:', err);
    }
  };

  const handleSendMessage = async () => {
    if (!selectedTicket || !newMessage.trim()) return;
    
    try {
      await ticketService.addMessage(selectedTicket.id, newMessage);
      setNewMessage('');
      loadTickets();
    } catch (err: any) {
      setError(err.message || 'Failed to send message');
    }
  };

  const handleUpdateStatus = async (ticketId: number, status: string) => {
    try {
      await ticketService.updateTicket(ticketId, { status });
      loadTickets();
      if (selectedTicket && selectedTicket.id === ticketId) {
        setSelectedTicket({ ...selectedTicket, status });
      }
    } catch (err: any) {
      setError(err.message || 'Failed to update ticket');
    }
  };

  const handleCreateTicket = async () => {
    try {
      await ticketService.createTicket(newTicket);
      setShowCreateModal(false);
      setNewTicket({ subject: '', description: '', category: 'general', priority: 'medium' });
      loadTickets();
    } catch (err: any) {
      setError(err.message || 'Failed to create ticket');
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent': return 'text-red-600 bg-red-100 dark:bg-red-900';
      case 'high': return 'text-orange-600 bg-orange-100 dark:bg-orange-900';
      case 'medium': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900';
      case 'low': return 'text-green-600 bg-green-100 dark:bg-green-900';
      default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'open': return 'text-blue-600 bg-blue-100 dark:bg-blue-900';
      case 'in_progress': return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900';
      case 'resolved': return 'text-green-600 bg-green-100 dark:bg-green-900';
      case 'closed': return 'text-gray-600 bg-gray-100 dark:bg-gray-900';
      default: return 'text-gray-600 bg-gray-100 dark:bg-gray-900';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Support Tickets
          </h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Create Ticket
          </button>
        </div>

        {/* Filters */}
        <div className={`mb-6 p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="flex gap-4 flex-wrap">
            <select
              value={filter.status}
              onChange={(e) => setFilter({ ...filter, status: e.target.value })}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Status</option>
              <option value="open">Open</option>
              <option value="in_progress">In Progress</option>
              <option value="resolved">Resolved</option>
              <option value="closed">Closed</option>
            </select>

            <select
              value={filter.priority}
              onChange={(e) => setFilter({ ...filter, priority: e.target.value })}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Priority</option>
              <option value="urgent">Urgent</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>

            <select
              value={filter.category}
              onChange={(e) => setFilter({ ...filter, category: e.target.value })}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Categories</option>
              <option value="technical">Technical</option>
              <option value="billing">Billing</option>
              <option value="kyc">KYC</option>
              <option value="general">General</option>
            </select>

            <button
              onClick={loadTickets}
              className="px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600"
            >
              Apply Filters
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded-lg">
            {error}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Tickets List */}
          <div className={`lg:col-span-1 p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h2 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Tickets ({tickets.length})
            </h2>
            
            {loading ? (
              <div className="text-center py-4">Loading...</div>
            ) : (
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {tickets.map((ticket) => (
                  <div
                    key={ticket.id}
                    onClick={() => handleSelectTicket(ticket)}
                    className={`p-3 rounded-lg cursor-pointer transition-colors ${
                      selectedTicket?.id === ticket.id
                        ? 'bg-blue-600 text-white'
                        : isDark
                        ? 'bg-gray-700 hover:bg-gray-600'
                        : 'bg-gray-100 hover:bg-gray-200'
                    }`}
                  >
                    <div className="flex justify-between items-start mb-1">
                      <span className="font-medium text-sm truncate">{ticket.subject}</span>
                      <span className={`text-xs px-2 py-0.5 rounded ${getPriorityColor(ticket.priority)}`}>
                        {ticket.priority}
                      </span>
                    </div>
                    <div className="flex justify-between items-center text-xs opacity-75">
                      <span>{ticket.ticket_id}</span>
                      <span className={`px-2 py-0.5 rounded ${getStatusColor(ticket.status)}`}>
                        {ticket.status}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Ticket Detail */}
          <div className={`lg:col-span-2 p-4 rounded-lg ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            {selectedTicket ? (
              <>
                <div className="flex justify-between items-start mb-4">
                  <div>
                    <h2 className={`text-xl font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                      {selectedTicket.subject}
                    </h2>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                      {selectedTicket.ticket_id} • {selectedTicket.category} •{' '}
                      <span className={`px-2 py-0.5 rounded ${getPriorityColor(selectedTicket.priority)}`}>
                        {selectedTicket.priority}
                      </span>
                    </p>
                  </div>
                  <select
                    value={selectedTicket.status}
                    onChange={(e) => handleUpdateStatus(selectedTicket.id, e.target.value)}
                    className={`px-3 py-1 rounded-lg border ${
                      isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                    }`}
                  >
                    <option value="open">Open</option>
                    <option value="in_progress">In Progress</option>
                    <option value="resolved">Resolved</option>
                    <option value="closed">Closed</option>
                  </select>
                </div>

                <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-gray-700' : 'bg-gray-100'}`}>
                  <p className={isDark ? 'text-gray-300' : 'text-gray-700'}>{selectedTicket.description}</p>
                </div>

                {/* Messages */}
                <div className="space-y-3 mb-4 max-h-64 overflow-y-auto">
                  {messages.map((msg) => (
                    <div
                      key={msg.id}
                      className={`p-3 rounded-lg ${
                        msg.sender_type === 'admin'
                          ? isDark
                            ? 'bg-blue-900 ml-8'
                            : 'bg-blue-100 ml-8'
                          : isDark
                          ? 'bg-gray-700 mr-8'
                          : 'bg-gray-100 mr-8'
                      }`}
                    >
                      <p className={isDark ? 'text-gray-300' : 'text-gray-700'}>{msg.message}</p>
                      <p className={`text-xs mt-1 ${isDark ? 'text-gray-500' : 'text-gray-500'}`}>
                        {new Date(msg.created_at).toLocaleString()}
                      </p>
                    </div>
                  ))}
                </div>

                {/* Reply */}
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={newMessage}
                    onChange={(e) => setNewMessage(e.target.value)}
                    placeholder="Type your reply..."
                    className={`flex-1 px-4 py-2 rounded-lg border ${
                      isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                    }`}
                    onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
                  />
                  <button
                    onClick={handleSendMessage}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                  >
                    Send
                  </button>
                </div>
              </>
            ) : (
              <div className="text-center py-12">
                <p className={isDark ? 'text-gray-400' : 'text-gray-600'}>
                  Select a ticket to view details
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Create Ticket Modal */}
        {showCreateModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`p-6 rounded-lg w-full max-w-md ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                Create New Ticket
              </h2>
              
              <div className="space-y-4">
                <div>
                  <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                    Subject
                  </label>
                  <input
                    type="text"
                    value={newTicket.subject}
                    onChange={(e) => setNewTicket({ ...newTicket, subject: e.target.value })}
                    className={`w-full px-4 py-2 rounded-lg border ${
                      isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                    }`}
                  />
                </div>
                
                <div>
                  <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                    Description
                  </label>
                  <textarea
                    value={newTicket.description}
                    onChange={(e) => setNewTicket({ ...newTicket, description: e.target.value })}
                    rows={4}
                    className={`w-full px-4 py-2 rounded-lg border ${
                      isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                    }`}
                  />
                </div>
                
                <div>
                  <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                    Category
                  </label>
                  <select
                    value={newTicket.category}
                    onChange={(e) => setNewTicket({ ...newTicket, category: e.target.value })}
                    className={`w-full px-4 py-2 rounded-lg border ${
                      isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                    }`}
                  >
                    <option value="technical">Technical</option>
                    <option value="billing">Billing</option>
                    <option value="kyc">KYC</option>
                    <option value="general">General</option>
                  </select>
                </div>
                
                <div>
                  <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
                    Priority
                  </label>
                  <select
                    value={newTicket.priority}
                    onChange={(e) => setNewTicket({ ...newTicket, priority: e.target.value })}
                    className={`w-full px-4 py-2 rounded-lg border ${
                      isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
                    }`}
                  >
                    <option value="low">Low</option>
                    <option value="medium">Medium</option>
                    <option value="high">High</option>
                    <option value="urgent">Urgent</option>
                  </select>
                </div>
              </div>
              
              <div className="flex justify-end gap-2 mt-6">
                <button
                  onClick={() => setShowCreateModal(false)}
                  className={`px-4 py-2 rounded-lg ${
                    isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'
                  }`}
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreateTicket}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                >
                  Create
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
