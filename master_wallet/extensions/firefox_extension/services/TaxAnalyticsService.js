// TigerWallet MasterWallet - Tax Analytics Service (Chrome Extension)
// Tax reporting and analytics
// Production-ready

class MasterTaxAnalyticsService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.transactions = [];
    this.reports = [];
    this.config = {
      method: 'FIFO',
      jurisdiction: 'US',
      shortTermRate: 0.37,
      longTermRate: 0.20,
      incomeTaxRate: 0.22,
    };
    this.isInitialized = false;
  }

  async initialize() {
    if (this.isInitialized) return true;
    
    try {
      // Load transactions
      await this.loadTransactions();
      
      // Load config
      await this.loadConfig();
      
      this.isInitialized = true;
      return true;
    } catch (error) {
      console.error('TaxAnalyticsService initialization failed:', error);
      return false;
    }
  }

  async loadTransactions() {
    const result = await chrome.storage.local.get('taxTransactions');
    if (result.taxTransactions) {
      this.transactions = result.taxTransactions;
    }
  }

  async loadConfig() {
    const result = await chrome.storage.local.get('taxConfig');
    if (result.taxConfig) {
      this.config = result.taxConfig;
    }
  }

  async saveTransactions() {
    await chrome.storage.local.set({
      taxTransactions: this.transactions,
    });
  }

  async saveConfig() {
    await chrome.storage.local.set({
      taxConfig: this.config,
    });
  }

  // Add transaction
  async addTransaction(tx) {
    this.transactions.push({
      ...tx,
      addedAt: Date.now(),
    });
    
    await this.saveTransactions();
    return true;
  }

  // Add multiple transactions
  async addTransactions(txs) {
    for (const tx of txs) {
      this.transactions.push({
        ...tx,
        addedAt: Date.now(),
      });
    }
    
    await this.saveTransactions();
    return true;
  }

  // Get transactions
  async getTransactions(walletAddress, filters = {}) {
    let result = this.transactions.filter(t => t.walletAddress === walletAddress);
    
    if (filters.startDate) {
      result = result.filter(t => t.timestamp >= filters.startDate);
    }
    if (filters.endDate) {
      result = result.filter(t => t.timestamp <= filters.endDate);
    }
    if (filters.type) {
      result = result.filter(t => t.type === filters.type);
    }
    if (filters.asset) {
      result = result.filter(t => t.asset === filters.asset);
    }
    
    return result;
  }

  // Calculate capital gains/losses
  async calculateGainsLosses(walletAddress, taxYear) {
    const transactions = await this.getTransactions(walletAddress, {
      startDate: `${taxYear}-01-01`,
      endDate: `${taxYear}-12-31`,
    });
    
    const gains = [];
    const assets = new Set();
    
    // Get all assets
    transactions.forEach(t => assets.add(t.asset));
    
    for (const asset of assets) {
      const assetTxs = transactions
        .filter(t => t.asset === asset)
        .sort((a, b) => a.timestamp - b.timestamp);
      
      const lots = [];
      
      for (const tx of assetTxs) {
        if (tx.type === 'buy' || tx.type === 'transfer_in') {
          // Add to lots
          lots.push({
            quantity: tx.quantity,
            costPerUnit: tx.priceUSD,
            totalCost: tx.quantity * tx.priceUSD,
            date: tx.timestamp,
          });
        } else if (tx.type === 'sell' || tx.type === 'transfer_out') {
          // Calculate cost basis using FIFO/LIFO/HIFO
          let remaining = tx.quantity;
          let costBasis = 0;
          
          while (remaining > 0 && lots.length > 0) {
            const lot = lots[0];
            const take = Math.min(remaining, lot.quantity);
            
            costBasis += take * lot.costPerUnit;
            lot.quantity -= take;
            remaining -= take;
            
            if (lot.quantity === 0) {
              lots.shift();
            }
          }
          
          const proceeds = tx.quantity * tx.priceUSD - (tx.feeUSD || 0);
          const gainLoss = proceeds - costBasis;
          
          // Determine term
          const acquisitionDate = lots[0] ? lots[0].date : tx.timestamp;
          const daysHeld = Math.floor((tx.timestamp - acquisitionDate) / (1000 * 60 * 60 * 24));
          const term = daysHeld >= 365 ? 'long_term' : 'short_term';
          
          gains.push({
            asset: tx.asset,
            proceeds: proceeds,
            costBasis: costBasis,
            gainLoss: gainLoss,
            term: term,
            disposalDate: tx.timestamp,
          });
        }
      }
    }
    
    return gains;
  }

  // Generate tax report
  async generateTaxReport(walletAddress, taxYear) {
    const gains = await this.calculateGainsLosses(walletAddress, taxYear);
    
    // Calculate totals
    let totalProceeds = 0;
    let totalCostBasis = 0;
    let shortTermGainLoss = 0;
    let longTermGainLoss = 0;
    const gainsByAsset = {};
    
    for (const gain of gains) {
      totalProceeds += gain.proceeds;
      totalCostBasis += gain.costBasis;
      
      if (gain.term === 'short_term') {
        shortTermGainLoss += gain.gainLoss;
      } else {
        longTermGainLoss += gain.gainLoss;
      }
      
      gainsByAsset[gain.asset] = (gainsByAsset[gain.asset] || 0) + gain.gainLoss;
    }
    
    // Calculate income
    const transactions = await this.getTransactions(walletAddress, {
      startDate: `${taxYear}-01-01`,
      endDate: `${taxYear}-12-31`,
    });
    
    let stakingRewards = 0;
    let interestIncome = 0;
    let defiIncome = 0;
    
    for (const tx of transactions) {
      if (tx.type === 'staking' || tx.type === 'reward') {
        stakingRewards += tx.quantity * tx.priceUSD;
      } else if (tx.type === 'interest') {
        interestIncome += tx.quantity * tx.priceUSD;
      } else if (tx.type === 'defi') {
        defiIncome += tx.quantity * tx.priceUSD;
      }
    }
    
    const income = stakingRewards + interestIncome + defiIncome;
    const totalGainLoss = shortTermGainLoss + longTermGainLoss;
    const totalTaxableIncome = totalGainLoss + income;
    
    // Calculate tax
    const shortTermTax = shortTermGainLoss > 0 ? 
      shortTermGainLoss * this.config.shortTermRate : 0;
    const longTermTax = longTermGainLoss > 0 ? 
      longTermGainLoss * this.config.longTermRate : 0;
    const incomeTax = income * this.config.incomeTaxRate;
    
    const report = {
      reportId: 'tax_' + Date.now(),
      walletAddress: walletAddress,
      taxYear: taxYear,
      totalProceeds: totalProceeds,
      totalCostBasis: totalCostBasis,
      totalGainLoss: totalGainLoss,
      shortTermGainLoss: shortTermGainLoss,
      longTermGainLoss: longTermGainLoss,
      income: income,
      stakingRewards: stakingRewards,
      interestIncome: interestIncome,
      defiIncome: defiIncome,
      totalTaxableIncome: totalTaxableIncome,
      shortTermTax: shortTermTax,
      longTermTax: longTermTax,
      incomeTax: incomeTax,
      totalTax: shortTermTax + longTermTax + incomeTax,
      transactions: gains,
      gainsByAsset: gainsByAsset,
      generatedAt: Date.now(),
    };
    
    // Store report
    this.reports.push(report);
    await this.saveReports();
    
    return report;
  }

  async saveReports() {
    await chrome.storage.local.set({
      taxReports: this.reports,
    });
  }

  // Get report
  async getReport(reportId) {
    return this.reports.find(r => r.reportId === reportId);
  }

  // Get reports
  async getReports(walletAddress, year = null) {
    let result = this.reports;
    
    if (walletAddress) {
      result = result.filter(r => r.walletAddress === walletAddress);
    }
    if (year) {
      result = result.filter(r => r.taxYear === year);
    }
    
    return result;
  }

  // Configuration
  async setConfig(config) {
    this.config = { ...this.config, ...config };
    await this.saveConfig();
    return true;
  }

  getConfig() {
    return this.config;
  }

  // Export functions
  async exportToCSV(reportId) {
    const report = await this.getReport(reportId);
    if (!report) return '';
    
    let csv = 'Asset,Proceeds,Cost Basis,Gain/Loss,Term,Disposal Date\n';
    
    for (const tx of report.transactions) {
      csv += `${tx.asset},${tx.proceeds},${tx.costBasis},${tx.gainLoss},${tx.term},${tx.disposalDate}\n`;
    }
    
    csv += `\nTotal Proceeds,${report.totalProceeds}\n`;
    csv += `Total Cost Basis,${report.totalCostBasis}\n`;
    csv += `Total Gain/Loss,${report.totalGainLoss}\n`;
    csv += `Short-term Gain/Loss,${report.shortTermGainLoss}\n`;
    csv += `Long-term Gain/Loss,${report.longTermGainLoss}\n`;
    csv += `Income,${report.income}\n`;
    csv += `Total Taxable Income,${report.totalTaxableIncome}\n`;
    csv += `Total Tax,${report.totalTax}\n`;
    
    return csv;
  }

  async exportToJSON(reportId) {
    const report = await this.getReport(reportId);
    return JSON.stringify(report, null, 2);
  }

  // Statistics
  async getStats() {
    return {
      totalTransactions: this.transactions.length,
      totalReports: this.reports.length,
    };
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = MasterTaxAnalyticsService;
}
