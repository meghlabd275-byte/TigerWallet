import React from 'react'
import { useOutletContext } from 'react-router-dom'
import { Box, Grid, Card, CardContent, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Chip } from '@mui/material'
import { LineChart, Line, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'

interface ContextType {
  darkMode?: boolean;
  toggleTheme?: () => void;
}

const volumeData = [
  { name: 'Mon', volume: 2400000 },
  { name: 'Tue', volume: 3200000 },
  { name: 'Wed', volume: 2800000 },
  { name: 'Thu', volume: 4100000 },
  { name: 'Fri', volume: 3800000 },
  { name: 'Sat', volume: 2900000 },
  { name: 'Sun', volume: 2600000 },
]

const chainDistribution = [
  { name: 'Ethereum', value: 35 },
  { name: 'BSC', value: 28 },
  { name: 'Polygon', value: 15 },
  { name: 'Arbitrum', value: 12 },
  { name: 'Other', value: 10 },
]

const recentSwaps = [
  { id: '0x1234...5678', user: '0xAbc...789', tokenIn: 'ETH', amountIn: '2.5', tokenOut: 'USDT', amountOut: '6125', chain: 'Ethereum', time: '2m ago' },
  { id: '0x2345...6789', user: '0xDef...012', tokenIn: 'USDC', amountIn: '10000', tokenOut: 'ETH', amountOut: '4.1', chain: 'BSC', time: '5m ago' },
  { id: '0x3456...7890', user: '0xGhi...345', tokenIn: 'BNB', amountIn: '15', tokenOut: 'CAKE', amountOut: '225', chain: 'BSC', time: '8m ago' },
  { id: '0x4567...8901', user: '0xJkl...678', tokenIn: 'MATIC', amountIn: '5000', tokenOut: 'ETH', amountOut: '2.8', chain: 'Polygon', time: '12m ago' },
  { id: '0x5678...9012', user: '0xMno...901', tokenIn: 'ETH', amountIn: '1.2', tokenOut: 'ARB', amountOut: '1200', chain: 'Arbitrum', time: '15m ago' },
]

const Dashboard: React.FC = () => {
  const { darkMode } = useOutletContext<ContextType>()
  
  const cardBg = darkMode ? '#1a1a1a' : '#fff'
  const textPrimary = darkMode ? '#fff' : '#000'
  const textSecondary = darkMode ? '#aaa' : '#666'

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ color: textPrimary, fontWeight: 700 }}>Dashboard</Typography>
      <Grid container spacing={3}>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: cardBg }}>
            <CardContent>
              <Typography sx={{ color: textSecondary }} gutterBottom>Total Volume (24h)</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>$21.8M</Typography>
              <Typography variant="body2" color="success.main">+12.5%</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: cardBg }}>
            <CardContent>
              <Typography sx={{ color: textSecondary }} gutterBottom>Total Users</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>154,289</Typography>
              <Typography variant="body2" color="success.main">+8.2%</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: cardBg }}>
            <CardContent>
              <Typography sx={{ color: textSecondary }} gutterBottom>Total Transactions</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>1.2M</Typography>
              <Typography variant="body2" color="success.main">+15.3%</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: cardBg }}>
            <CardContent>
              <Typography sx={{ color: textSecondary }} gutterBottom>Active Pools</Typography>
              <Typography variant="h4" sx={{ color: textPrimary }}>3,456</Typography>
              <Typography variant="body2" color="success.main">+5.1%</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={8}>
          <Card sx={{ height: 400, bgcolor: cardBg }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ color: textPrimary }}>Volume Trend</Typography>
              <ResponsiveContainer width="100%" height="90%">
                <LineChart data={volumeData}>
                  <CartesianGrid strokeDasharray="3 3" stroke={darkMode ? '#333' : '#eee'} />
                  <XAxis dataKey="name" stroke={textSecondary} />
                  <YAxis stroke={textSecondary} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: cardBg, border: `1px solid ${darkMode ? '#333' : '#eee'}`, color: textPrimary }} 
                  />
                  <Legend />
                  <Line type="monotone" dataKey="volume" stroke="#f97316" strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={4}>
          <Card sx={{ height: 400, bgcolor: cardBg }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ color: textPrimary }}>Chain Distribution</Typography>
              <ResponsiveContainer width="100%" height="90%">
                <PieChart>
                  <Pie 
                    data={chainDistribution} 
                    dataKey="value" 
                    nameKey="name" 
                    cx="50%" 
                    cy="50%" 
                    outerRadius={80} 
                    label
                  >
                    {chainDistribution.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={['#f97316', '#3b82f6', '#10b981', '#8b5cf6', '#ec4899'][index % 5]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    contentStyle={{ backgroundColor: cardBg, border: `1px solid ${darkMode ? '#333' : '#eee'}`, color: textPrimary }} 
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12}>
          <Card sx={{ bgcolor: cardBg }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ color: textPrimary }}>Recent Swaps</Typography>
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: textSecondary }}>Tx Hash</TableCell>
                      <TableCell sx={{ color: textSecondary }}>User</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Token In</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Amount In</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Token Out</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Amount Out</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Chain</TableCell>
                      <TableCell sx={{ color: textSecondary }}>Time</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {recentSwaps.map((swap) => (
                      <TableRow key={swap.id}>
                        <TableCell><Chip label={swap.id} size="small" /></TableCell>
                        <TableCell sx={{ color: textPrimary }}>{swap.user}</TableCell>
                        <TableCell sx={{ color: textPrimary }}>{swap.tokenIn}</TableCell>
                        <TableCell sx={{ color: textPrimary }}>{swap.amountIn}</TableCell>
                        <TableCell sx={{ color: textPrimary }}>{swap.tokenOut}</TableCell>
                        <TableCell sx={{ color: textPrimary }}>{swap.amountOut}</TableCell>
                        <TableCell><Chip label={swap.chain} size="small" color="primary" /></TableCell>
                        <TableCell sx={{ color: textPrimary }}>{swap.time}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  )
}
export default Dashboard
