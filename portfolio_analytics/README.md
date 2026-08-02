# TigerWallet Portfolio Analytics Module

## Overview

The Portfolio Analytics module provides comprehensive portfolio tracking, analysis, and visualization for TigerWallet users. Supports multi-chain portfolio management, performance tracking, and advanced analytics.

## Features

- **Real-Time Portfolio**: Live balance and value tracking
- **Multi-Chain**: Aggregated view across all chains
- **Performance Analytics**: ROI, returns, benchmarks
- **Risk Metrics**: Volatility, Sharpe ratio, drawdown
- **Asset Allocation**: Diversification analysis
- **Price Alerts**: Custom notifications
- **Historical Data**: Full transaction history
- **Export**: CSV, PDF reports

## Installation

```bash
npm install @tigerwallet/portfolio-analytics
```

## Quick Start

```typescript
import { PortfolioAnalytics, PortfolioConfig } from '@tigerwallet/portfolio-analytics';

const portfolio = new PortfolioAnalytics({
  userId: 'user123',
  wallets: ['0xWallet1...', '0xWallet2...'],
  chains: ['ethereum', 'polygon', 'arbitrum'],
});

// Get portfolio summary
const summary = await portfolio.getSummary();
console.log('Total Value:', summary.totalValue);
console.log('24h Change:', summary.change24h);
console.log('Assets:', summary.assets);
```

## Usage Examples

### 1. Portfolio Summary

```typescript
// Get comprehensive portfolio summary
const summary = await portfolio.getSummary({
  currency: 'USD',
  timeframe: '24h',
});

console.log({
  totalValue: summary.totalValue,
  totalChange: summary.change24h,
  changePercent: summary.changePercent24h,
  assets: summary.assets.map(a => ({
    symbol: a.symbol,
    balance: a.balance,
    value: a.value,
    allocation: a.allocationPercent,
  })),
});
```

### 2. Performance Analytics

```typescript
// Get performance metrics
const performance = await portfolio.getPerformance({
  timeframe: '30d',
  benchmark: 'BTC',
});

console.log({
  totalReturn: performance.totalReturn,
  annualizedReturn: performance.annualizedReturn,
  volatility: performance.volatility,
  sharpeRatio: performance.sharpeRatio,
  maxDrawdown: performance.maxDrawdown,
  beta: performance.beta,
  alpha: performance.alpha,
});
```

### 3. Asset Allocation

```typescript
// Get asset allocation breakdown
const allocation = await portfolio.getAllocation({
  groupBy: 'chain', // 'chain', 'asset', 'category'
  includeTokens: true,
  includeNFTs: false,
});

console.log({
  chains: allocation.byChain,
  categories: allocation.byCategory,
  tokens: allocation.byToken,
  diversificationScore: allocation.diversificationScore,
});
```

### 4. Transaction History

```typescript
// Get transaction history
const transactions = await portfolio.getTransactions({
  startDate: '2024-01-01',
  endDate: '2024-12-31',
  type: ['send', 'receive', 'swap', 'stake'],
  chain: 'ethereum',
  limit: 100,
});

console.log('Transactions:', transactions.map(tx => ({
  date: tx.timestamp,
  type: tx.type,
  asset: tx.asset,
  amount: tx.amount,
  value: tx.valueUSD,
  from: tx.from,
  to: tx.to,
})));
```

### 5. Yield Analytics

```typescript
// Get yield/farming positions
const yieldPositions = await portfolio.getYieldPositions();

console.log('Staking:', yieldPositions.staking.map(s => ({
  protocol: s.protocol,
  asset: s.asset,
  staked: s.amount,
  apy: s.apy,
  rewards: s.pendingRewards,
  value: s.totalValue,
})));

console.log('Farming:', yieldPositions.farming.map(f => ({
  protocol: f.protocol,
  pool: f.pool,
  liquidity: f.liquidity,
  apy: f.apy,
})));
```

### 6. NFT Portfolio

```typescript
// Get NFT holdings
const nfts = await portfolio.getNFTs({
  includeFloorPrice: true,
  includeLastSale: true,
});

console.log('NFTs:', nfts.map(nft => ({
  collection: nft.collection,
  name: nft.name,
  quantity: nft.quantity,
  floorPrice: nft.floorPrice,
  lastSale: nft.lastSale,
  totalValue: nft.totalValue,
  holdings: nft.holdings,
})));
```

### 7. Risk Analysis

```typescript
// Get risk metrics
const risk = await portfolio.getRiskMetrics({
  timeframe: '90d',
  confidenceLevel: 0.95,
});

console.log({
  volatility: risk.volatility,
  var: risk.valueAtRisk,
  cvar: risk.conditionalVar,
  sharpeRatio: risk.sharpeRatio,
  sortinoRatio: risk.sortinoRatio,
  beta: risk.beta,
  correlationMatrix: risk.correlationMatrix,
});
```

### 8. Price Alerts

```typescript
// Set price alert
await portfolio.setAlert({
  asset: 'ETH',
  condition: 'above',
  price: 4000,
  notification: ['push', 'email'],
});

// Get active alerts
const alerts = await portfolio.getAlerts();
console.log('Active alerts:', alerts);

// Delete alert
await portfolio.deleteAlert('alert_123');
```

### 9. Historical Data

```typescript
// Get portfolio value history
const history = await portfolio.getHistory({
  startDate: '2024-01-01',
  endDate: '2024-12-31',
  interval: '1d', // 1h, 1d, 1w
});

console.log('History:', history.map(h => ({
  date: h.timestamp,
  value: h.value,
  change: h.change,
  changePercent: h.changePercent,
})));
```

### 10. Export Reports

```typescript
// Export to CSV
const csv = await portfolio.export({
  format: 'csv',
  include: ['summary', 'transactions', 'performance'],
  dateRange: { start: '2024-01-01', end: '2024-12-31' },
});

// Export to PDF
const pdf = await portfolio.export({
  format: 'pdf',
  template: 'detailed',
  includeCharts: true,
  include: ['summary', 'allocation', 'performance'],
});

// Export for taxes
const taxExport = await portfolio.export({
  format: 'tax',
  year: 2024,
});
```

## API Reference

### PortfolioAnalytics Class

```typescript
class PortfolioAnalytics {
  constructor(config: PortfolioConfig);
  
  // Summary
  async getSummary(options?: SummaryOptions): Promise<PortfolioSummary>;
  
  // Performance
  async getPerformance(options: PerformanceOptions): Promise<PerformanceMetrics>;
  
  // Allocation
  async getAllocation(options?: AllocationOptions): Promise<AllocationBreakdown>;
  
  // Transactions
  async getTransactions(options: TransactionQuery): Promise<Transaction[]>;
  
  // Yield
  async getYieldPositions(): Promise<YieldPositions>;
  
  // NFTs
  async getNFTs(options?: NFTQuery): Promise<NFTPortfolio>;
  
  // Risk
  async getRiskMetrics(options?: RiskOptions): Promise<RiskMetrics>;
  
  // Alerts
  async setAlert(alert: AlertConfig): Promise<void>;
  async getAlerts(): Promise<Alert[]>;
  async deleteAlert(id: string): Promise<void>;
  
  // History
  async getHistory(options: HistoryOptions): Promise<HistoryPoint[]>;
  
  // Export
  async export(options: ExportOptions): Promise<Blob>;
}
```

### Types

```typescript
interface PortfolioConfig {
  userId: string;
  wallets: string[];
  chains?: string[];
  currency?: string;
}

interface PortfolioSummary {
  totalValue: number;
  totalChange24h: number;
  changePercent24h: number;
  totalChange7d: number;
  changePercent7d: number;
  totalChange30d: number;
  changePercent30d: number;
  assets: Asset[];
  chains: ChainBreakdown[];
}

interface PerformanceMetrics {
  totalReturn: number;
  annualizedReturn: number;
  volatility: number;
  sharpeRatio: number;
  sortinoRatio: number;
  maxDrawdown: number;
  beta: number;
  alpha: number;
  benchmarkComparison: BenchmarkResult[];
}

interface Asset {
  symbol: string;
  name: string;
  chain: string;
  balance: string;
  price: number;
  value: number;
  change24h: number;
  allocationPercent: number;
  logoUrl?: string;
}

interface Transaction {
  id: string;
  timestamp: string;
  type: 'send' | 'receive' | 'swap' | 'stake' | 'unstake' | 'mint' | 'burn';
  chain: string;
  asset: string;
  amount: string;
  valueUSD: number;
  from?: string;
  to?: string;
  txHash: string;
  status: 'pending' | 'confirmed' | 'failed';
}
```

## Charts & Visualization Data

### Portfolio Pie Chart

```typescript
const pieData = await portfolio.getPieChartData({
  groupBy: 'asset',
  showNFTs: true,
});
// Returns: { labels, values, colors }
```

### Line Chart

```typescript
const lineData = await portfolio.getLineChartData({
  startDate: '2024-01-01',
  endDate: '2024-12-31',
  compare: ['BTC', 'SPY'],
});
// Returns: { portfolio: [], benchmark1: [], benchmark2: [] }
```

### Bar Chart

```typescript
const barData = await portfolio.getBarChartData({
  metric: 'pnl',
  period: 'monthly',
});
// Returns: { labels: [], values: [], colors: [] }
```

## Real-Time Updates

```typescript
// Subscribe to real-time updates
portfolio.subscribe((update) => {
  console.log('Portfolio updated:', update);
  console.log('New value:', update.totalValue);
  console.log('Changed assets:', update.changedAssets);
});

// Unsubscribe
portfolio.unsubscribe();
```

## Dashboard Widgets

```typescript
// Get data for dashboard widgets
const widgets = await portfolio.getDashboardWidgets();

// Widget 1: Portfolio Value
widgets.portfolioValue:
// Returns: { value, change, sparklineData }

// Widget 2: Top Movers
widgets.topMovers:
// Returns: { gainers: [], losers: [] }

// Widget 3: Recent Activity
widgets.recentActivity:
// Returns: { transactions: [] }

// Widget 4: Allocation
widgets.allocation:
// Returns: { pieChartData, categoryBreakdown }

// Widget 5: Yield
widgets.yieldOverview:
// Returns: { totalYield, positions: [] }
```

## Best Practices

1. **Multi-Chain Tracking**: Add all wallets across chains
2. **Regular Updates**: Sync transactions frequently
3. **Price Alerts**: Set meaningful alerts
4. **Backup Data**: Export regularly
5. **Review Allocation**: Check diversification
6. **Track Yield**: Monitor staking/farming
