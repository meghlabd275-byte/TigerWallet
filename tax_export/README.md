# TigerWallet Tax Integration Module

## Overview

The Tax Integration module provides comprehensive cryptocurrency tax calculation, reporting, and export functionality for TigerWallet users. Supports multiple tax jurisdictions and formats.

## Features

- **Transaction Tracking**: All DeFi, NFT, and trading transactions
- **Cost Basis Calculation**: FIFO, LIFO, HIFO, Specific ID
- **Tax Lot Management**: Track individual tax lots
- **Capital Gains/Losses**: Short-term and long-term
- **Income Recognition**: Staking, mining, airdrops
- **Export Formats**: CSV, PDF, TurboTax, H&R Block
- **Multi-Jurisdiction**: US, UK, Canada, Australia, Germany

## Installation

```bash
npm install @tigerwallet/tax-integration
```

## Quick Start

```typescript
import { TaxService, TaxConfig } from '@tigerwallet/tax-integration';

const taxService = new TaxService({
  userId: 'user123',
  jurisdiction: 'US',
  taxYear: 2024,
});

// Calculate taxes for all transactions
const report = await taxService.calculateTaxes({
  transactions: allTransactions,
  costBasisMethod: 'FIFO',
});

console.log('Short-term gains:', report.shortTermGains);
console.log('Long-term gains:', report.longTermGains);
console.log('Total income:', report.totalIncome);

// Export for tax software
const exportData = await taxService.export({
  format: 'turbotax',
  year: 2024,
});
```

## Usage Examples

### 1. Import Transactions

```typescript
// Auto-import from wallet
const transactions = await taxService.importFromWallet({
  address: '0xUserAddress...',
  startDate: '2024-01-01',
  endDate: '2024-12-31',
});

// Import from exchange
const exchangeTx = await taxService.importFromExchange({
  exchange: 'binance',
  apiKey: process.env.BINANCE_KEY,
  apiSecret: process.env.BINANCE_SECRET,
});

// Manual transaction entry
await taxService.addTransaction({
  type: 'trade',
  timestamp: '2024-06-15T10:30:00Z',
  assetIn: 'ETH',
  amountIn: '1.0',
  assetOut: 'USDC',
  amountOut: '3500.00',
  fee: '10.00',
  exchange: 'uniswap',
});
```

### 2. Calculate Cost Basis

```typescript
// Using FIFO (First In, First Out)
const fifoGains = await taxService.calculateGains({
  method: 'FIFO',
  transactions: trades,
});

// Using LIFO (Last In, First Out)
const lifoGains = await taxService.calculateGains({
  method: 'LIFO',
  transactions: trades,
});

// Using HIFO (Highest In, First Out)
const hifoGains = await taxService.calculateGains({
  method: 'HIFO',
  transactions: trades,
});

// Specific Lot Identification
const lots = await taxService.getAvailableLots({
  asset: 'ETH',
  address: '0xUser...',
});
```

### 3. Income Events

```typescript
// Staking rewards
await taxService.addIncomeEvent({
  type: 'staking',
  asset: 'ETH',
  amount: '0.5',
  timestamp: '2024-06-01T00:00:00Z',
  costBasis: '0',
  fairMarketValue: '1800.00',
});

// Mining income
await taxService.addIncomeEvent({
  type: 'mining',
  asset: 'BTC',
  amount: '0.01',
  timestamp: '2024-05-15T00:00:00Z',
  costBasis: '0',
  fairMarketValue: '650.00',
});

// Airdrop
await taxService.addIncomeEvent({
  type: 'airdrop',
  asset: 'NEWTOKEN',
  amount: '100',
  timestamp: '2024-04-01T00:00:00Z',
  costBasis: '0',
  fairMarketValue: '50.00',
});

// NFT mint (as income)
await taxService.addIncomeEvent({
  type: 'nft_mint',
  asset: 'NFT',
  tokenId: '1234',
  timestamp: '2024-03-01T00:00:00Z',
  costBasis: '0',
  fairMarketValue: '500.00',
});
```

### 4. Generate Reports

```typescript
// Annual tax report
const annualReport = await taxService.generateReport({
  year: 2024,
  format: 'detailed',
});

// Short summary for quick view
const summary = await taxService.getTaxSummary({
  year: 2024,
});

// Gain/loss by asset
const assetBreakdown = await taxService.getAssetBreakdown({
  year: 2024,
});

// Transaction history
const history = await taxService.getTransactionHistory({
  startDate: '2024-01-01',
  endDate: '2024-12-31',
  types: ['trade', 'transfer'],
});
```

### 5. Export Formats

```typescript
// TurboTax
const turbotax = await taxService.export({
  format: 'turboTax',
  year: 2024,
  output: 'csv',
});

// H&R Block
const hrblock = await taxService.export({
  format: 'hrblock',
  year: 2024,
});

// Generic CSV
const csv = await taxService.export({
  format: 'csv',
  year: 2024,
  include: ['transactions', 'gains', 'income'],
});

// PDF Report
const pdf = await taxService.export({
  format: 'pdf',
  year: 2024,
  template: 'detailed',
  includeSignature: true,
});
```

## Supported Jurisdictions

| Country | Tax Year | Features |
|---------|----------|----------|
| United States | Jan-Dec | Form 8949, Schedule D |
| United Kingdom | Apr-Apr | CGT calculation |
| Canada | Jan-Dec | T5008, Schedule 3 |
| Australia | Jul-Jun | CGT events |
| Germany | Jan-Dec | ESt-R |
| Japan | Jan-Dec | Bitcoin taxation |

## API Reference

### TaxService

```typescript
class TaxService {
  constructor(config: TaxConfig);
  
  // Import transactions
  async importFromWallet(config: ImportConfig): Promise<Transaction[]>;
  async importFromExchange(config: ExchangeConfig): Promise<Transaction[]>;
  async addTransaction(tx: ManualTransaction): Promise<void>;
  
  // Cost basis
  async calculateGains(config: GainsConfig): Promise<GainsResult>;
  async getAvailableLots(asset: string): Promise<TaxLot[]>;
  
  // Income
  async addIncomeEvent(event: IncomeEvent): Promise<void>;
  
  // Reports
  async generateReport(config: ReportConfig): Promise<TaxReport>;
  async getTaxSummary(year: number): Promise<TaxSummary>;
  async getAssetBreakdown(year: number): Promise<AssetBreakdown[]>;
  
  // Export
  async export(config: ExportConfig): Promise<ExportResult>;
}
```

### Types

```typescript
interface Transaction {
  id: string;
  type: 'trade' | 'transfer' | 'mint' | 'burn' | 'swap';
  timestamp: string;
  assetIn: string;
  amountIn: string;
  assetOut: string;
  amountOut: string;
  fee: string;
  exchange?: string;
  txHash: string;
}

interface TaxLot {
  id: string;
  asset: string;
  amount: string;
  costBasis: string;
  acquisitionDate: string;
  isLongTerm: boolean;
}

interface TaxReport {
  year: number;
  shortTermGains: string;
  shortTermLosses: string;
  longTermGains: string;
  longTermLosses: string;
  totalIncome: string;
  totalGains: string;
  transactions: Transaction[];
  lots: TaxLot[];
}
```

## Features

### DeFi Tax Tracking

```typescript
// Track DeFi transactions
await taxService.addDefiTransaction({
  protocol: 'uniswap',
  type: 'swap',
  path: ['ETH', 'USDC'],
  amounts: ['1.0', '3500'],
  fee: '10',
  timestamp: '2024-06-15',
});

// Track liquidity provision
await taxService.addDefiTransaction({
  protocol: 'aave',
  type: 'supply',
  asset: 'USDC',
  amount: '10000',
  timestamp: '2024-05-01',
});

// Track yield
await taxService.addDefiTransaction({
  protocol: 'aave',
  type: 'yield',
  asset: 'USDC',
  amount: '50',
  timestamp: '2024-06-01',
});
```

### NFT Tax Handling

```typescript
// NFT purchase
await taxService.addNftTransaction({
  type: 'purchase',
  collection: 'Bored Ape',
  tokenId: '1234',
  amount: '100000',
  currency: 'USDC',
  timestamp: '2024-01-01',
});

// NFT sale
await taxService.addNftTransaction({
  type: 'sale',
  collection: 'Bored Ape',
  tokenId: '1234',
  amount: '150000',
  currency: 'USDC',
  costBasis: '100000',
  timestamp: '2024-06-01',
});

// NFT mint (as income)
await taxService.addNftTransaction({
  type: 'mint',
  collection: 'New NFT',
  tokenId: '5678',
  amount: '500',
  currency: 'ETH',
  fairMarketValue: '1000',
  timestamp: '2024-03-01',
});
```

## Supported Formats

### CSV Export
```csv
Date,Type,Asset,Amount,Cost Basis,Proceeds,Gain/Loss,Exchange
2024-01-15,buy,ETH,1.0,2500.00,0.00,0.00,coinbase
2024-06-15,sell,ETH,1.0,2500.00,3500.00,1000.00,uniswap
```

### TurboTax Format
```json
{
  "form8949": [
    {
      "description": "Ethereum",
      "dateAcquired": "2024-01-15",
      "dateSold": "2024-06-15",
      "proceeds": 3500.00,
      "costBasis": 2500.00,
      "gainLoss": 1000.00
    }
  ]
}
```

## Best Practices

1. **Record All Transactions**: Include small transfers
2. **Track Costs**: Keep all purchase records
3. **Categorize Income**: Separate staking/mining/airdrops
4. **Use Accurate Prices**: Use fair market values
5. **Consult Tax Pro**: For complex situations
