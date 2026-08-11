/**
 * Analytics Service - React/Web Implementation
 * Identical across ALL platforms
 */

class AnalyticsService {
  private static instance: AnalyticsService;
  private holdings: Map<string, AssetHolding> = new Map();
  private transactions: PortfolioTransaction[] = [];
  private alerts: PriceAlert[] = [];
  private totalPortfolioValue = 0;
  private previousPortfolioValue = 0;

  static getInstance(): AnalyticsService {
    if (!AnalyticsService.instance) {
      AnalyticsService.instance = new AnalyticsService();
    }
    return AnalyticsService.instance;
  }

  updatePortfolio(holdings: Map<string, AssetHolding>): void {
    this.previousPortfolioValue = this.totalPortfolioValue;
    this.holdings = new Map(holdings);
    this.recalculateValue();
  }

  getSummary(): PortfolioSummary {
    return {
      totalValue: this.totalPortfolioValue,
      change24h: this.totalPortfolioValue - this.previousPortfolioValue,
      changePercent24h:
        this.previousPortfolioValue > 0
          ? ((this.totalPortfolioValue - this.previousPortfolioValue) /
              this.previousPortfolioValue) *
            100
          : 0,
      assets: Array.from(this.holdings.values()),
      lastUpdated: Date.now(),
    };
  }

  getPerformance(timeframe: string): PerformanceMetrics {
    // Performance metrics must be computed from real on-chain history
    // (wallet-api /transactions + price feed). Without a real analytics
    // backend, return honest zeros rather than fabricated random returns.
    return {
      timeframe,
      totalReturn: 0,
      annualizedReturn: 0,
      volatility: 0,
      sharpeRatio: 0,
      maxDrawdown: 0,
      riskLevel: 'LOW',
    };
  }

  getAllocation(): AllocationBreakdown {
    const byChain: Record<string, number> = {};
    const byCategory: Record<string, number> = {};

    for (const holding of this.holdings.values()) {
      byChain[holding.chain] = (byChain[holding.chain] ?? 0) + holding.value;
      byCategory[holding.category] =
        (byCategory[holding.category] ?? 0) + holding.value;
    }

    return {
      byChain,
      byCategory,
      totalValue: this.totalPortfolioValue,
      diversificationScore: this.calculateDiversificationScore(byChain),
    };
  }

  getTransactionHistory(options?: {
    startDate?: string;
    endDate?: string;
    type?: string[];
  }): PortfolioTransaction[] {
    let result = [...this.transactions];

    if (options?.startDate) {
      result = result.filter((tx) => tx.date >= options.startDate!);
    }
    if (options?.endDate) {
      result = result.filter((tx) => tx.date <= options.endDate!);
    }
    if (options?.type) {
      result = result.filter((tx) => options.type!.includes(tx.type));
    }

    return result;
  }

  setAlert(config: {
    asset: string;
    condition: AlertCondition;
    targetPrice: number;
  }): PriceAlert {
    const alert: PriceAlert = {
      id: `alert_${Date.now()}`,
      asset: config.asset,
      condition: config.condition,
      targetPrice: config.targetPrice,
      isActive: true,
      createdAt: Date.now(),
    };
    this.alerts.push(alert);
    return alert;
  }

  getAlerts(): PriceAlert[] {
    return this.alerts.filter((a) => a.isActive);
  }

  deleteAlert(alertId: string): boolean {
    const index = this.alerts.findIndex((a) => a.id === alertId);
    if (index !== -1) {
      this.alerts.splice(index, 1);
      return true;
    }
    return false;
  }

  getHistory(
    startDate: string,
    endDate: string,
    interval: string
  ): HistoryPoint[] {
    return [];
  }

  exportReport(format: string): string {
    if (format === 'csv') {
      let csv = 'Asset,Chain,Balance,Value,Allocation\n';
      for (const holding of this.holdings.values()) {
        csv += `${holding.symbol},${holding.chain},${holding.balance},${holding.value},${holding.allocation}\n`;
      }
      return csv;
    }
    return '{}';
  }

  // Private
  private recalculateValue(): void {
    this.totalPortfolioValue = Array.from(this.holdings.values()).reduce(
      (sum, h) => sum + h.value,
      0
    );
  }

  private getAnnualizationFactor(timeframe: string): number {
    switch (timeframe) {
      case '1d':
        return 365;
      case '1w':
        return 52;
      case '1m':
        return 12;
      default:
        return 1;
    }
  }

  private calculateDiversificationScore(byChain: Record<string, number>): number {
    if (Object.keys(byChain).length === 0) return 0;
    const total = Object.values(byChain).reduce((sum, v) => sum + v, 0);
    if (total === 0) return 0;

    const proportions = Object.values(byChain).map((v) => v / total);
    const sumSquares = proportions.reduce((sum, p) => sum + p * p, 0);

    return sumSquares > 0 ? (1 / sumSquares) / Object.keys(byChain).length * 100 : 0;
  }
}

export interface AssetHolding {
  symbol: string;
  name: string;
  chain: string;
  category: string;
  balance: number;
  price: number;
  value: number;
  allocation: number;
  change24h: number;
}

export interface PortfolioSummary {
  totalValue: number;
  change24h: number;
  changePercent24h: number;
  assets: AssetHolding[];
  lastUpdated: number;
}

export interface PerformanceMetrics {
  timeframe: string;
  totalReturn: number;
  annualizedReturn: number;
  volatility: number;
  sharpeRatio: number;
  maxDrawdown: number;
  riskLevel: string;
}

export interface AllocationBreakdown {
  byChain: Record<string, number>;
  byCategory: Record<string, number>;
  totalValue: number;
  diversificationScore: number;
}

export interface PortfolioTransaction {
  id: string;
  type: string;
  asset: string;
  amount: number;
  value: number;
  date: string;
  txHash: string;
}

export enum AlertCondition {
  ABOVE = 'above',
  BELOW = 'below',
}

export interface PriceAlert {
  id: string;
  asset: string;
  condition: AlertCondition;
  targetPrice: number;
  isActive: boolean;
  createdAt: number;
}

export interface HistoryPoint {
  timestamp: number;
  value: number;
  change: number;
}

export default AnalyticsService.getInstance();
