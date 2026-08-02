/**
 * Tax and Analytics Services - Browser Extension
 */

const CostBasisMethod = { FIFO: 'fifo', LIFO: 'lifo', HIFO: 'hifo' };
const TransactionType = { BUY: 'buy', SELL: 'sell', TRANSFER: 'transfer', SWAP: 'swap', STAKE: 'stake', UNSTAKE: 'unstake', MINT: 'mint', BURN: 'burn' };
const AlertCondition = { ABOVE: 'above', BELOW: 'below' };

class TaxService {
  static instance = null;

  static getInstance() {
    if (!TaxService.instance) TaxService.instance = new TaxService();
    return TaxService.instance;
  }

  constructor() {
    this.transactions = [];
    this.taxLots = new Map();
    this.incomeEvents = [];
    this.jurisdiction = 'US';
    this.costBasisMethod = CostBasisMethod.FIFO;
  }

  setJurisdiction(code) { this.jurisdiction = code; return true; }
  setCostBasisMethod(method) { this.costBasisMethod = method; return true; }
  addTransaction(tx) { this.transactions.push(tx); }
  
  calculateGains() {
    let totalIncome = this.incomeEvents.reduce((sum, e) => sum + e.fairMarketValue, 0);
    return {
      year: 2024, shortTermGains: 0, shortTermLosses: 0,
      longTermGains: 0, longTermLosses: 0, totalIncome,
      totalTransactions: this.transactions.length,
      jurisdiction: this.jurisdiction, costBasisMethod: this.costBasisMethod
    };
  }

  getAvailableLots(asset) {
    const lots = this.taxLots.get(asset);
    return lots ? lots.filter(lot => lot.remainingAmount > 0) : [];
  }

  addIncomeEvent(event) {
    this.incomeEvents.push(event);
    const lot = { id: `lot_${Date.now()}`, asset: event.asset, amount: event.amount,
      remainingAmount: event.amount, costBasis: 0, fairMarketValue: event.fairMarketValue,
      acquisitionDate: event.date, isLongTerm: false };
    const existing = this.taxLots.get(event.asset) || [];
    this.taxLots.set(event.asset, [...existing, lot]);
  }

  exportCSV() {
    let csv = 'Date,Type,Asset,Amount,Cost Basis,Proceeds,Gain/Loss,Exchange\n';
    for (const tx of this.transactions) {
      csv += `${tx.date},${tx.type},${tx.asset},${tx.amount},${tx.costBasis},${tx.proceeds},${tx.gainLoss},${tx.exchange}\n`;
    }
    return csv;
  }
}

class AnalyticsService {
  static instance = null;

  static getInstance() {
    if (!AnalyticsService.instance) AnalyticsService.instance = new AnalyticsService();
    return AnalyticsService.instance;
  }

  constructor() {
    this.holdings = new Map();
    this.transactions = [];
    this.alerts = [];
    this.totalPortfolioValue = 0;
    this.previousPortfolioValue = 0;
  }

  updatePortfolio(holdings) {
    this.previousPortfolioValue = this.totalPortfolioValue;
    this.holdings = new Map(holdings);
    this.recalculateValue();
  }

  getSummary() {
    return {
      totalValue: this.totalPortfolioValue,
      change24h: this.totalPortfolioValue - this.previousPortfolioValue,
      changePercent24h: this.previousPortfolioValue > 0 
        ? ((this.totalPortfolioValue - this.previousPortfolioValue) / this.previousPortfolioValue) * 100 : 0,
      assets: Array.from(this.holdings.values()),
      lastUpdated: Date.now()
    };
  }

  getPerformance(timeframe) {
    const returns = Math.random() * 40 - 10;
    const volatility = Math.abs(returns) * 0.5;
    const factor = timeframe === '1d' ? 365 : timeframe === '1w' ? 52 : timeframe === '1m' ? 12 : 1;
    return {
      timeframe, totalReturn: returns, annualizedReturn: returns * factor,
      volatility, sharpeRatio: volatility > 0 ? returns / volatility : 0,
      maxDrawdown: Math.random() * 20,
      riskLevel: volatility < 0.1 ? 'LOW' : volatility < 0.3 ? 'MEDIUM' : 'HIGH'
    };
  }

  getAllocation() {
    const byChain = {}, byCategory = {};
    for (const h of this.holdings.values()) {
      byChain[h.chain] = (byChain[h.chain] || 0) + h.value;
      byCategory[h.category] = (byCategory[h.category] || 0) + h.value;
    }
    return { byChain, byCategory, totalValue: this.totalPortfolioValue,
      diversificationScore: this.calculateDiversificationScore(byChain) };
  }

  getTransactionHistory(options = {}) {
    let result = [...this.transactions];
    if (options.startDate) result = result.filter(t => t.date >= options.startDate);
    if (options.endDate) result = result.filter(t => t.date <= options.endDate);
    if (options.type?.length) result = result.filter(t => options.type.includes(t.type));
    return result;
  }

  setAlert(asset, condition, targetPrice) {
    const alert = { id: `alert_${Date.now()}`, asset, condition, targetPrice, isActive: true, createdAt: Date.now() };
    this.alerts.push(alert);
    return alert;
  }

  getAlerts() { return this.alerts.filter(a => a.isActive); }
  deleteAlert(alertId) { const i = this.alerts.findIndex(a => a.id === alertId); return i !== -1 ? this.alerts.splice(i, 1) : false; }
  exportReport(format) {
    if (format === 'csv') {
      let csv = 'Asset,Chain,Balance,Value,Allocation\n';
      for (const h of this.holdings.values()) csv += `${h.symbol},${h.chain},${h.balance},${h.value},${h.allocation}\n`;
      return csv;
    }
    return '{}';
  }

  recalculateValue() {
    this.totalPortfolioValue = Array.from(this.holdings.values()).reduce((s, h) => s + h.value, 0);
  }

  calculateDiversificationScore(byChain) {
    if (!Object.keys(byChain).length) return 0;
    const total = Object.values(byChain).reduce((s, v) => s + v, 0);
    if (!total) return 0;
    const sumSquares = Object.values(byChain).reduce((s, v) => s + Math.pow(v / total, 2), 0);
    return sumSquares > 0 ? (1 / sumSquares) / Object.keys(byChain).length * 100 : 0;
  }
}

export default { TaxService, AnalyticsService, CostBasisMethod, TransactionType, AlertCondition };
export { TaxService, AnalyticsService, CostBasisMethod, TransactionType, AlertCondition };
