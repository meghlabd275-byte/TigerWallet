import React, { useState, useEffect, useCallback } from 'react'
import { useOutletContext } from 'react-router-dom'
import { Box, Grid, Card, CardContent, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Chip, Alert, CircularProgress } from '@mui/material'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { adminFetch } from '../api'

interface ContextType {
  darkMode?: boolean;
  toggleTheme?: () => void;
}

interface AdminStats {
  totalUsers: number;
  activeUsers: number;
  totalTransactions: number;
  totalVolume: number;
  dailyRevenue: number;
  monthlyRevenue: number;
}

interface TxLogRecord {
  id: string;
  user_id: string;
  wallet_id: string;
  tx_hash: string;
  chain_id: number;
  from_addr: string;
  to_addr: string;
  value: string;
  status: string;
}

const fmtNum = (n: number) => {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return `${n}`
}

const shortHash = (h: string) => (h ? `${h.slice(0, 8)}...${h.slice(-4)}` : '')

const Dashboard: React.FC = () => {
  const { darkMode } = useOutletContext<ContextType>()
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [txs, setTxs] = useState<TxLogRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [s, t] = await Promise.all([
        adminFetch<AdminStats>('/api/v1/admin/stats'),
        adminFetch<TxLogRecord[]>('/api/v1/admin/transactions'),
      ])
      setStats(s)
      setTxs(t || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard data')
      setStats(null)
      setTxs([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const cardBg = darkMode ? '#1a1a1a' : '#fff'
  const textPrimary = darkMode ? '#fff' : '#000'
  const textSecondary = darkMode ? '#aaa' : '#666'

  // Derive chain distribution from REAL recent transactions (count per chain_id).
  const chainCounts = new Map<number, number>()
  txs.forEach((t) => chainCounts.set(t.chain_id, (chainCounts.get(t.chain_id) || 0) + 1))
  const chainDistribution = Array.from(chainCounts.entries())
    .map(([chainId, value]) => ({ name: `Chain ${chainId}`, value }))
    .sort((a, b) => b.value - a.value)

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ color: textPrimary, fontWeight: 700 }}>Dashboard</Typography>
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}><CircularProgress /></Box>
      ) : (
        <Grid container spacing={3}>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography sx={{ color: textSecondary }} gutterBottom>Total Volume</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats ? fmtNum(stats.totalVolume) : '0'}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography sx={{ color: textSecondary }} gutterBottom>Total Users</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats ? fmtNum(stats.totalUsers) : '0'}</Typography>
                <Typography variant="body2" sx={{ color: textSecondary }}>{stats ? `${stats.activeUsers} active (24h)` : ''}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography sx={{ color: textSecondary }} gutterBottom>Total Transactions</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats ? fmtNum(stats.totalTransactions) : '0'}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card sx={{ bgcolor: cardBg }}>
              <CardContent>
                <Typography sx={{ color: textSecondary }} gutterBottom>Monthly Revenue</Typography>
                <Typography variant="h4" sx={{ color: textPrimary }}>{stats ? fmtNum(stats.monthlyRevenue) : '0'}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={8}>
            <Card sx={{ height: 400, bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="h6" gutterBottom sx={{ color: textPrimary }}>Recent Transactions</Typography>
                <TableContainer sx={{ maxHeight: 320 }}>
                  <Table size="small" stickyHeader>
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ color: textSecondary }}>Tx Hash</TableCell>
                        <TableCell sx={{ color: textSecondary }}>From</TableCell>
                        <TableCell sx={{ color: textSecondary }}>To</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Value</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Chain</TableCell>
                        <TableCell sx={{ color: textSecondary }}>Status</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {txs.length === 0 ? (
                        <TableRow><TableCell colSpan={6} align="center"><Typography sx={{ py: 3 }} color="text.secondary">No transactions yet.</Typography></TableCell></TableRow>
                      ) : txs.slice(0, 8).map((t) => (
                        <TableRow key={t.id}>
                          <TableCell><Chip label={shortHash(t.tx_hash)} size="small" /></TableCell>
                          <TableCell sx={{ color: textPrimary }}>{shortHash(t.from_addr)}</TableCell>
                          <TableCell sx={{ color: textPrimary }}>{shortHash(t.to_addr)}</TableCell>
                          <TableCell sx={{ color: textPrimary }}>{t.value}</TableCell>
                          <TableCell><Chip label={t.chain_id} size="small" color="primary" /></TableCell>
                          <TableCell><Chip label={t.status} size="small" color={t.status === 'completed' ? 'success' : 'default'} /></TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card sx={{ height: 400, bgcolor: cardBg }}>
              <CardContent>
                <Typography variant="h6" gutterBottom sx={{ color: textPrimary }}>Chain Distribution (recent)</Typography>
                {chainDistribution.length === 0 ? (
                  <Typography sx={{ py: 6, textAlign: 'center' }} color="text.secondary">No data yet.</Typography>
                ) : (
                  <ResponsiveContainer width="100%" height="90%">
                    <PieChart>
                      <Pie data={chainDistribution} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={80} label>
                        {chainDistribution.map((_, index) => (
                          <Cell key={`cell-${index}`} fill={['#f97316', '#3b82f6', '#10b981', '#8b5cf6', '#ec4899', '#06b6d4'][index % 6]} />
                        ))}
                      </Pie>
                      <Tooltip contentStyle={{ backgroundColor: cardBg, border: `1px solid ${darkMode ? '#333' : '#eee'}`, color: textPrimary }} />
                      <Legend />
                    </PieChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}
    </Box>
  )
}
export default Dashboard
