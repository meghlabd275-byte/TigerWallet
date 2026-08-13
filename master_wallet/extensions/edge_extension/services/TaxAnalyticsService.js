/**
 * TaxAnalyticsService - tax reporting client for the extension.
 *
 * Tax lots, cost basis and the tax report are computed by the backend from
 * the authoritative on-chain transaction history. The client requests a
 * report for a given master wallet + tax year and may export the returned
 * report to CSV/JSON. It never fabricates gains, lots, or tax liabilities.
 *
 * Fail-closed: if the backend does not return a report, exports return null.
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const { authedFetch } = (typeof require === 'function')
  ? require('./apiClient.js')
  : ((globalThis.MW_API) || {});

class MasterTaxAnalyticsService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.isInitialized = false;
  }

  async initialize() {
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
    this.isInitialized = true;
    return true;
  }

  _assert() {
    if (!this.isInitialized) throw new Error('TaxAnalytics service not initialized');
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
  }

  // Authoritative transaction history comes from the canonical transactions
  // route (real RPC + DB), not from locally cached client data.
  async getTransactions() {
    this._assert();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/transactions', { method: 'GET' });
    return res.transactions || [];
  }

  // Request the backend compute the tax report for a year. Returns the report
  // object as produced by the backend (lots, gains, totals, tax).
  async generateTaxReport(taxYear) {
    this._assert();
    if (!taxYear) throw new Error('taxYear is required');
    return authedFetch('/master-wallet/' + this.masterWalletId + '/analytics/transactions', {
      method: 'GET',
      query: { tax_year: taxYear },
    });
  }

  // CSV export of a backend-produced report. Returns null if no report.
  async exportToCSV(report) {
    if (!report) return null;
    const rows = report.transactions || report.gains || [];
    let csv = 'Asset,Proceeds,Cost Basis,Gain/Loss,Term,Disposal Date\n';
    for (const tx of rows) {
      csv += [
        tx.asset || '',
        tx.proceeds != null ? tx.proceeds : '',
        tx.cost_basis != null ? tx.cost_basis : (tx.costBasis != null ? tx.costBasis : ''),
        tx.gain_loss != null ? tx.gain_loss : (tx.gainLoss != null ? tx.gainLoss : ''),
        tx.term || '',
        tx.disposal_date != null ? tx.disposal_date : (tx.disposalDate != null ? tx.disposalDate : ''),
      ].join(',') + '\n';
    }
    csv += '\nTotal Proceeds,' + (report.total_proceeds || report.totalProceeds || '') + '\n';
    csv += 'Total Cost Basis,' + (report.total_cost_basis || report.totalCostBasis || '') + '\n';
    csv += 'Total Gain/Loss,' + (report.total_gain_loss || report.totalGainLoss || '') + '\n';
    csv += 'Total Tax,' + (report.total_tax || report.totalTax || '') + '\n';
    return csv;
  }

  async exportToJSON(report) {
    if (!report) return null;
    return JSON.stringify(report, null, 2);
  }

  // Local UI config cache (method, jurisdiction, rates). These are user
  // preferences for display only; the backend computes the authoritative tax.
  async getConfig() {
    return new Promise((resolve) => {
      try {
        chrome.storage.local.get('mw_tax_config', (res) => {
          resolve(res && res.mw_tax_config ? res.mw_tax_config : {
            method: 'FIFO',
            jurisdiction: 'US',
          });
        });
      } catch (e) {
        resolve({ method: 'FIFO', jurisdiction: 'US' });
      }
    });
  }

  async setConfig(config) {
    const current = await this.getConfig();
    const merged = { ...current, ...config };
    return new Promise((resolve) => {
      try {
        chrome.storage.local.set({ mw_tax_config: merged }, () => resolve(true));
      } catch (e) {
        resolve(false);
      }
    });
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterTaxAnalyticsService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_TAX = { MasterTaxAnalyticsService };
}
