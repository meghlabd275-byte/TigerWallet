/**
 * TaxAnalyticsService - Web/React Implementation
 * Complete tax tracking and reporting for Master Wallet
 * Features: Cost basis, Capital gains/losses, Multi-jurisdiction, Tax reports
 * Ultra-low latency with optimized queries
 */

import { randomBytes, createHash } from 'crypto';

// Tax types
type CostBasisMethod = 'FIFO' | 'LIFO' | 'HIFO' | 'AVERAGE';
type TaxEventType = 'BUY' | 'SELL' | 'TRANSFER_IN' | 'TRANSFER_OUT' | 'REWARD' | 'STAKING' | 'MINING' | 'AIRDROP' | 'FEE' | 'SWAP';
type TaxJurisdiction = 'US' | 'UK' | 'EU' | 'JP' | 'AU' | 'CA' | 'DE' | 'DEFAULT';

interface Transaction {
  id: string;
  hash: string;
  timestamp: number;
  blockNumber: number;
  from: string;
  to: string;
  token: string;
  amount: string;
  valueUSD: number;
  fee: string;
  feeUSD: number;
  status: 'pending' | 'confirmed' | 'failed';
}

interface TaxLot {
  id: string;
  transactionId: string;
  token: string;
  amount: string;
  costBasis: string;
  costBasisUSD: number;
  acquisitionDate: number;
  holdingPeriod: 'short' | 'long';
}

interface TaxEvent {
  id: string;
  type: TaxEventType;
  token: string;
  amount: string;
  proceeds: string;
  proceedsUSD: number;
  costBasis: string;
  costBasisUSD: number;
  gainLoss: string;
  gainLossUSD: number;
  transactionId: string;
  timestamp: number;
  jurisdiction: TaxJurisdiction;
}

interface TaxReport {
  id: string;
  generatedAt: number;
  startDate: number;
  endDate: number;
  jurisdiction: TaxJurisdiction;
  summary: {
    totalProceeds: string;
    totalCostBasis: string;
    totalGainLoss: string;
    shortTermGainLoss: string;
    longTermGainLoss: string;
    totalIncome: string;
    totalFees: string;
  };
  events: TaxEvent[];
  transactions: Transaction[];
}

interface TaxSummary {
  totalProceeds: number;
  totalCostBasis: number;
  totalGainLoss: number;
  shortTermGain: number;
  shortTermLoss: number;
  longTermGain: number;
  longTermLoss: number;
  totalIncome: number;
  totalFees: number;
  washSaleAdjustments: number;
  tokenBreakdown: Map<string, {
    proceeds: number;
    costBasis: number;
    gainLoss: number;
    count: number;
  }>;
}

interface JurisdictionConfig {
  code: TaxJurisdiction;
  name: string;
  shortTermThreshold: number;  // Days
  capitalGainsRate: number;     // Percentage
  incomeTaxRate: number;
  reportingThreshold: number;
  currency: string;
}

class TaxAnalyticsService {
  private static instance: TaxAnalyticsService | null = null;
  
  // Data storage
  private transactions: Map<string, Transaction> = new Map();
  private taxLots: Map<string, TaxLot> = new Map();
  private tokenPrices: Map<string, Map<number, number>> = new Map(); // token -> timestamp -> price
  
  // Configuration
  private defaultJurisdiction: TaxJurisdiction = 'US';
  private defaultMethod: CostBasisMethod = 'FIFO';
  
  // Jurisdiction configurations
  private jurisdictions: Map<TaxJurisdiction, JurisdictionConfig> = new Map([
    ['US', { code: 'US', name: 'United States', shortTermThreshold: 365, capitalGainsRate: 20, incomeTaxRate: 37, reportingThreshold: 600, currency: 'USD' }],
    ['UK', code: 'UK', name: 'United Kingdom', shortTermThreshold: 0, capitalGainsRate: 20, incomeTaxRate: 45, reportingThreshold: 0, currency: 'GBP' }],
    ['EU', { code: 'EU', name: 'European Union', shortTermThreshold: 365, capitalGainsRate: 25, incomeTaxRate: 45, reportingThreshold: 0, currency: 'EUR' }],
    ['JP', { code: 'JP', name: 'Japan', shortTermThreshold: 365, capitalGainsRate: 20, incomeTaxRate: 45, reportingThreshold: 50000, currency: 'JPY' }],
    ['AU', { code: 'AU', name: 'Australia', shortTermThreshold: 365, capitalGainsRate: 22.5, incomeTaxRate: 45, reportingThreshold: 0, currency: 'AUD' }],
    ['CA', { code: 'CA', name: 'Canada', shortTermThreshold: 365, capitalGainsRate: 50, incomeTaxRate: 54, reportingThreshold: 1000, currency: 'CAD' }],
    ['DE', { code: 'DE', name: 'Germany', shortTermThreshold: 365, capitalGainsRate: 25, incomeTaxRate: 45, reportingThreshold: 0, currency: 'EUR' }],
    ['DEFAULT', { code: 'DEFAULT', name: 'Default', shortTermThreshold: 365, capitalGainsRate: 20, incomeTaxRate: 30, reportingThreshold: 0, currency: 'USD' }],
  ]);

  private constructor() {}

  static getInstance(): TaxAnalyticsService {
    if (!TaxAnalyticsService.instance) {
      TaxAnalyticsService.instance = new TaxAnalyticsService();
    }
    return TaxAnalyticsService.instance;
  }

  // ==================== Configuration ====================

  /**
   * Set default jurisdiction
   */
  setDefaultJurisdiction(jurisdiction: TaxJurisdiction): void {
    if (this.jurisdictions.has(jurisdiction)) {
      this.defaultJurisdiction = jurisdiction;
    }
  }

  /**
   * Set default cost basis method
   */
  setDefaultMethod(method: CostBasisMethod): void {
    this.defaultMethod = method;
  }

  /**
   * Get jurisdiction config
   */
  getJurisdictionConfig(jurisdiction?: TaxJurisdiction): JurisdictionConfig {
    return this.jurisdictions.get(jurisdiction || this.defaultJurisdiction) || this.jurisdictions.get('DEFAULT')!;
  }

  /**
   * Add/update jurisdiction config
   */
  setJurisdictionConfig(config: JurisdictionConfig): void {
    this.jurisdictions.set(config.code, config);
  }

  // ==================== Transaction Import ====================

  /**
   * Import transaction
   */
  importTransaction(tx: Transaction): void {
    this.transactions.set(tx.id, tx);
    
    // Create tax lot for incoming transactions
    if (tx.status === 'confirmed' && tx.valueUSD > 0) {
      this.createTaxLot(tx);
    }
  }

  /**
   * Batch import transactions
   */
  batchImport(transactions: Transaction[]): { imported: number; failed: number } {
    let imported = 0;
    let failed = 0;

    for (const tx of transactions) {
      try {
        this.importTransaction(tx);
        imported++;
      } catch {
        failed++;
      }
    }

    return { imported, failed };
  }

  /**
   * Create tax lot from transaction
   */
  private createTaxLot(tx: Transaction): void {
    if (tx.to.toLowerCase() === tx.from.toLowerCase()) return; // Skip self-transfers

    const lot: TaxLot = {
      id: `lot_${tx.id}`,
      transactionId: tx.id,
      token: tx.token,
      amount: tx.amount,
      costBasis: tx.valueUSD.toString(),
      costBasisUSD: tx.valueUSD,
      acquisitionDate: tx.timestamp,
      holdingPeriod: 'short',
    };

    this.taxLots.set(lot.id, lot);
  }

  // ==================== Price Data ====================

  /**
   * Update token price
   */
  updateTokenPrice(token: string, timestamp: number, priceUSD: number): void {
    if (!this.tokenPrices.has(token)) {
      this.tokenPrices.set(token, new Map());
    }
    this.tokenPrices.get(token)!.set(timestamp, priceUSD);
  }

  /**
   * Batch update token prices
   */
  batchUpdatePrices(prices: { token: string; timestamp: number; priceUSD: number }[]): void {
    for (const { token, timestamp, priceUSD } of prices) {
      this.updateTokenPrice(token, timestamp, priceUSD);
    }
  }

  /**
   * Get token price at timestamp
   */
  getTokenPrice(token: string, timestamp: number): number {
    const prices = this.tokenPrices.get(token);
    if (!prices || prices.size === 0) return 0;

    // Find closest price
    let closestPrice = 0;
    let minDiff = Infinity;

    for (const [time, price] of prices) {
      const diff = Math.abs(time - timestamp);
      if (diff < minDiff) {
        minDiff = diff;
        closestPrice = price;
      }
    }

    return closestPrice;
  }

  // ==================== Tax Calculation ====================

  /**
   * Calculate cost basis using specified method
   */
  calculateCostBasis(
    token: string,
    sellAmount: string,
    method?: CostBasisMethod,
    timestamp?: number
  ): { costBasis: string; lotsUsed: { lotId: string; amount: string; costBasis: string }[] } {
    const lots = this.getAvailableLots(token, timestamp);
    const methodToUse = method || this.defaultMethod;
    
    // Sort lots based on method
    let sortedLots = [...lots];
    switch (methodToUse) {
      case 'FIFO':
        sortedLots.sort((a, b) => a.acquisitionDate - b.acquisitionDate);
        break;
      case 'LIFO':
        sortedLots.sort((a, b) => b.acquisitionDate - a.acquisitionDate);
        break;
      case 'HIFO':
        sortedLots.sort((a, b) => parseFloat(b.costBasis) - parseFloat(a.costBasis));
        break;
      case 'AVERAGE':
        // Average cost basis
        const totalAmount = lots.reduce((sum, lot) => sum + parseFloat(lot.amount), 0);
        const totalCost = lots.reduce((sum, lot) => sum + lot.costBasisUSD, 0);
        const avgCost = totalAmount > 0 ? totalCost / totalAmount : 0;
        return {
          costBasis: (parseFloat(sellAmount) * avgCost).toString(),
          lotsUsed: [],
        };
    }

    // Calculate cost basis using sorted lots
    let remaining = parseFloat(sellAmount);
    let totalCostBasis = 0;
    const lotsUsed: { lotId: string; amount: string; costBasis: string }[] = [];

    for (const lot of sortedLots) {
      if (remaining <= 0) break;
      
      const lotAmount = parseFloat(lot.amount);
      const amountToUse = Math.min(remaining, lotAmount);
      const costBasisForAmount = (amountToUse / lotAmount) * lot.costBasisUSD;
      
      totalCostBasis += costBasisForAmount;
      lotsUsed.push({
        lotId: lot.id,
        amount: amountToUse.toString(),
        costBasis: costBasisForAmount.toString(),
      });
      
      remaining -= amountToUse;
    }

    return {
      costBasis: totalCostBasis.toString(),
      lotsUsed,
    };
  }

  /**
   * Get available tax lots for token
   */
  private getAvailableLots(token: string, beforeTimestamp?: number): TaxLot[] {
    const lots: TaxLot[] = [];
    const cutoff = beforeTimestamp || Date.now();

    for (const lot of this.taxLots.values()) {
      if (lot.token === token && lot.acquisitionDate < cutoff) {
        lots.push(lot);
      }
    }

    return lots;
  }

  /**
   * Process sale and calculate gain/loss
   */
  processSale(
    saleTx: Transaction,
    method?: CostBasisMethod
  ): TaxEvent {
    const costBasisResult = calculateCostBasis(
      saleTx.token,
      saleTx.amount,
      method,
      saleTx.timestamp
    );

    const proceeds = saleTx.valueUSD;
    const costBasis = parseFloat(costBasisResult.costBasis);
    const gainLoss = proceeds - costBasis;
    
    // Determine holding period
    const lotsUsed = costBasisResult.lotsUsed;
    let holdingPeriod: 'short' | 'long' = 'short';
    
    if (lotsUsed.length > 0) {
      const earliestAcquisition = Math.min(...lotsUsed.map(l => {
        const lot = this.taxLots.get(l.lotId);
        return lot?.acquisitionDate || 0;
      }));
      
      const jurisdiction = this.getJurisdictionConfig();
      const daysSinceAcquisition = (saleTx.timestamp - earliestAcquisition) / (1000 * 60 * 60 * 24);
      holdingPeriod = daysSinceAcquisition >= jurisdiction.shortTermThreshold ? 'long' : 'short';
    }

    // Create tax event
    const event: TaxEvent = {
      id: `event_${saleTx.id}`,
      type: 'SELL',
      token: saleTx.token,
      amount: saleTx.amount,
      proceeds: proceeds.toString(),
      proceedsUSD: proceeds,
      costBasis: costBasis.toString(),
      costBasisUSD: costBasis,
      gainLoss: gainLoss.toString(),
      gainLossUSD: gainLoss,
      transactionId: saleTx.id,
      timestamp: saleTx.timestamp,
      jurisdiction: this.defaultJurisdiction,
    };

    // Update tax lots (remove sold amounts)
    this.updateLotsAfterSale(costBasisResult.lotsUsed);

    return event;
  }

  private calculateCostBasis(
    token: string,
    sellAmount: string,
    method?: CostBasisMethod,
    timestamp?: number
  ): { costBasis: string; lotsUsed: { lotId: string; amount: string; costBasis: string }[] } {
    const lots = this.getAvailableLots(token, timestamp);
    const methodToUse = method || this.defaultMethod;
    
    let sortedLots = [...lots];
    switch (methodToUse) {
      case 'FIFO':
        sortedLots.sort((a, b) => a.acquisitionDate - b.acquisitionDate);
        break;
      case 'LIFO':
        sortedLots.sort((a, b) => b.acquisitionDate - a.acquisitionDate);
        break;
      case 'HIFO':
        sortedLots.sort((a, b) => parseFloat(b.costBasis) - parseFloat(a.costBasis));
        break;
      case 'AVERAGE':
        const totalAmount = lots.reduce((sum, lot) => sum + parseFloat(lot.amount), 0);
        const totalCost = lots.reduce((sum, lot) => sum + lot.costBasisUSD, 0);
        const avgCost = totalAmount > 0 ? totalCost / totalAmount : 0;
        return {
          costBasis: (parseFloat(sellAmount) * avgCost).toString(),
          lotsUsed: [],
        };
    }

    let remaining = parseFloat(sellAmount);
    let totalCostBasis = 0;
    const lotsUsed: { lotId: string; amount: string; costBasis: string }[] = [];

    for (const lot of sortedLots) {
      if (remaining <= 0) break;
      const lotAmount = parseFloat(lot.amount);
      const amountToUse = Math.min(remaining, lotAmount);
      const costBasisForAmount = (amountToUse / lotAmount) * lot.costBasisUSD;
      totalCostBasis += costBasisForAmount;
      lotsUsed.push({ lotId: lot.id, amount: amountToUse.toString(), costBasis: costBasisForAmount.toString() });
      remaining -= amountToUse;
    }

    return { costBasis: totalCostBasis.toString(), lotsUsed };
  }

  private updateLotsAfterSale(lotsUsed: { lotId: string; amount: string; costBasis: string }[]): void {
    for (const used of lotsUsed) {
      const lot = this.taxLots.get(used.lotId);
      if (lot) {
        const remaining = parseFloat(lot.amount) - parseFloat(used.amount);
        if (remaining <= 0) {
          this.taxLots.delete(lot.id);
        } else {
          lot.amount = remaining.toString();
          lot.costBasisUSD = lot.costBasisUSD * (remaining / parseFloat(used.amount));
          lot.costBasis = lot.costBasisUSD.toString();
          this.taxLots.set(lot.id, lot);
        }
      }
    }
  }

  // ==================== Tax Summary ====================

  /**
   * Calculate tax summary for period
   */
  calculateTaxSummary(
    startDate: number,
    endDate: number,
    jurisdiction?: TaxJurisdiction
  ): TaxSummary {
    const events = this.getTaxEvents(startDate, endDate, jurisdiction);
    const summary: TaxSummary = {
      totalProceeds: 0,
      totalCostBasis: 0,
      totalGainLoss: 0,
      shortTermGain: 0,
      shortTermLoss: 0,
      longTermGain: 0,
      longTermLoss: 0,
      totalIncome: 0,
      totalFees: 0,
      washSaleAdjustments: 0,
      tokenBreakdown: new Map(),
    };

    for (const event of events) {
      if (event.type === 'SELL') {
        summary.totalProceeds += event.proceedsUSD;
        summary.totalCostBasis += event.costBasisUSD;
        summary.totalGainLoss += event.gainLossUSD;

        if (event.gainLossUSD > 0) {
          // Check holding period
          const lot = this.taxLots.get(event.transactionId);
          if (lot?.holdingPeriod === 'short') {
            summary.shortTermGain += event.gainLossUSD;
          } else {
            summary.longTermGain += event.gainLossUSD;
          }
        } else if (event.gainLossUSD < 0) {
          if (lot?.holdingPeriod === 'short') {
            summary.shortTermLoss += Math.abs(event.gainLossUSD);
          } else {
            summary.longTermLoss += Math.abs(event.gainLossUSD);
          }
        }

        // Update token breakdown
        const tokenData = summary.tokenBreakdown.get(event.token) || {
          proceeds: 0,
          costBasis: 0,
          gainLoss: 0,
          count: 0,
        };
        tokenData.proceeds += event.proceedsUSD;
        tokenData.costBasis += event.costBasisUSD;
        tokenData.gainLoss += event.gainLossUSD;
        tokenData.count++;
        summary.tokenBreakdown.set(event.token, tokenData);
      } else if (['REWARD', 'STAKING', 'MINING', 'AIRDROP'].includes(event.type)) {
        summary.totalIncome += event.proceedsUSD;
      } else if (event.type === 'FEE') {
        summary.totalFees += event.proceedsUSD;
      }
    }

    return summary;
  }

  /**
   * Get tax events for period
   */
  getTaxEvents(
    startDate: number,
    endDate: number,
    jurisdiction?: TaxJurisdiction
  ): TaxEvent[] {
    const events: TaxEvent[] = [];

    for (const tx of this.transactions.values()) {
      if (tx.timestamp < startDate || tx.timestamp > endDate) continue;
      if (jurisdiction && this.defaultJurisdiction !== jurisdiction) continue;

      // Process each transaction as a tax event
      if (tx.status === 'confirmed' && tx.valueUSD > 0) {
        const event = this.processSale(tx);
        events.push(event);
      }
    }

    return events.sort((a, b) => a.timestamp - b.timestamp);
  }

  // ==================== Report Generation ====================

  /**
   * Generate tax report
   */
  generateTaxReport(
    startDate: number,
    endDate: number,
    jurisdiction?: TaxJurisdiction
  ): TaxReport {
    const events = this.getTaxEvents(startDate, endDate, jurisdiction);
    const summary = this.calculateTaxSummary(startDate, endDate, jurisdiction);
    
    const transactions = Array.from(this.transactions.values())
      .filter(tx => tx.timestamp >= startDate && tx.timestamp <= endDate)
      .sort((a, b) => a.timestamp - b.timestamp);

    const report: TaxReport = {
      id: `report_${Date.now()}`,
      generatedAt: Date.now(),
      startDate,
      endDate,
      jurisdiction: jurisdiction || this.defaultJurisdiction,
      summary: {
        totalProceeds: summary.totalProceeds.toString(),
        totalCostBasis: summary.totalCostBasis.toString(),
        totalGainLoss: summary.totalGainLoss.toString(),
        shortTermGainLoss: (summary.shortTermGain - summary.shortTermLoss).toString(),
        longTermGainLoss: (summary.longTermGain - summary.longTermLoss).toString(),
        totalIncome: summary.totalIncome.toString(),
        totalFees: summary.totalFees.toString(),
      },
      events,
      transactions,
    };

    return report;
  }

  /**
   * Export report as CSV
   */
  exportAsCSV(report: TaxReport): string {
    const lines: string[] = [];
    
    // Header
    lines.push('Type,Token,Amount,Proceeds,Cost Basis,Gain/Loss,Date,Jurisdiction');
    
    // Events
    for (const event of report.events) {
      lines.push([
        event.type,
        event.token,
        event.amount,
        event.proceedsUSD.toString(),
        event.costBasisUSD.toString(),
        event.gainLossUSD.toString(),
        new Date(event.timestamp).toISOString(),
        event.jurisdiction,
      ].join(','));
    }

    return lines.join('\n');
  }

  /**
   * Export report as JSON
   */
  exportAsJSON(report: TaxReport): string {
    return JSON.stringify(report, null, 2);
  }

  // ==================== Wash Sale Detection ====================

  /**
   * Detect wash sales
   */
  detectWashSales(
    token: string,
    startDate: number,
    endDate: number,
    thresholdDays: number = 30
  ): { events: TaxEvent[]; adjustment: number } {
    const events = this.getTaxEvents(startDate, endDate)
      .filter(e => e.token === token && e.gainLossUSD < 0);
    
    const washSaleEvents: TaxEvent[] = [];
    let totalAdjustment = 0;

    for (const event of events) {
      // Check if there was a purchase within threshold days before or after
      const purchaseWindow = {
        start: event.timestamp - thresholdDays * 24 * 60 * 60 * 1000,
        end: event.timestamp + thresholdDays * 24 * 60 * 60 * 1000,
      };

      const repurchases = this.transactions.values().filter(tx =>
        tx.token === token &&
        tx.timestamp >= purchaseWindow.start &&
        tx.timestamp <= purchaseWindow.end &&
        tx.valueUSD > 0
      );

      if (repurchases.length > 0) {
        washSaleEvents.push(event);
        totalAdjustment += Math.abs(event.gainLossUSD);
      }
    }

    return { events: washSaleEvents, adjustment: totalAdjustment };
  }

  // ==================== Multi-Jurisdiction ====================

  /**
   * Calculate tax for multiple jurisdictions
   */
  calculateMultiJurisdictionTax(
    startDate: number,
    endDate: number
  ): Map<TaxJurisdiction, TaxSummary> {
    const results = new Map<TaxJurisdiction, TaxSummary>();

    for (const [code] of this.jurisdictions) {
      if (code === 'DEFAULT') continue;
      const summary = this.calculateTaxSummary(startDate, endDate, code);
      results.set(code, summary);
    }

    return results;
  }

  /**
   * Get optimal jurisdiction for tax purposes
   */
  getOptimalJurisdiction(startDate: number, endDate: number): {
    jurisdiction: TaxJurisdiction;
    estimatedTax: number;
    savings: number;
  } {
    const usSummary = this.calculateTaxSummary(startDate, endDate, 'US');
    const usTax = this.calculateTax(usSummary, 'US');
    
    let bestJurisdiction: TaxJurisdiction = 'US';
    let bestTax = usTax;
    let maxSavings = 0;

    for (const [code, config] of this.jurisdictions) {
      if (code === 'US' || code === 'DEFAULT') continue;
      
      const summary = this.calculateTaxSummary(startDate, endDate, code);
      const tax = this.calculateTax(summary, code);
      
      const savings = usTax - tax;
      if (savings > maxSavings) {
        maxSavings = savings;
        bestTax = tax;
        bestJurisdiction = code;
      }
    }

    return {
      jurisdiction: bestJurisdiction,
      estimatedTax: bestTax,
      savings: maxSavings,
    };
  }

  /**
   * Calculate estimated tax
   */
  private calculateTax(summary: TaxSummary, jurisdictionCode: TaxJurisdiction): number {
    const config = this.getJurisdictionConfig(jurisdictionCode);
    
    // Short-term gains taxed as income
    const shortTermTax = Math.max(0, summary.shortTermGain - summary.shortTermLoss) * (config.incomeTaxRate / 100);
    
    // Long-term gains at capital gains rate
    const longTermTax = Math.max(0, summary.longTermGain - summary.longTermLoss) * (config.capitalGainsRate / 100);
    
    // Income taxed at income rate
    const incomeTax = summary.totalIncome * (config.incomeTaxRate / 100);
    
    return shortTermTax + longTermTax + incomeTax;
  }
}

export default TaxAnalyticsService.getInstance();
export { TaxAnalyticsService, TaxReport, TaxEvent, TaxSummary, Transaction, TaxLot, CostBasisMethod, TaxJurisdiction, JurisdictionConfig };
