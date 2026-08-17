/**
 * TaxAnalyticsService - Web (React/TypeScript)
 *
 * NOTE: Tax/cost-basis analytics are NOT part of the canonical MasterWallet
 * backend contract (port 8450). This module exposes the public typing surface
 * only; operations that previously fabricated tax lots and gains now return a
 * descriptive error instead of fake data.
 */

export type CostBasisMethod = 'FIFO' | 'LIFO' | 'HIFO' | 'AVERAGE';
export type TaxEventType =
  | 'BUY' | 'SELL' | 'TRANSFER_IN' | 'TRANSFER_OUT'
  | 'REWARD' | 'STAKING' | 'MINING' | 'AIRDROP' | 'FEE' | 'SWAP';
export type TaxJurisdiction = 'US' | 'UK' | 'EU' | 'JP' | 'AU' | 'CA' | 'DE' | 'DEFAULT';

export interface JurisdictionConfig {
  code: TaxJurisdiction;
  name: string;
  shortTermThreshold: number;
  capitalGainsRate: number;
  incomeTaxRate: number;
  reportingThreshold: number;
  currency: string;
}

export interface Transaction {
  id: string;
  type: TaxEventType;
  timestamp: number;
  token: string;
  amount: string;
  valueUSD: number;
  feeUSD: number;
  status: string;
}

export interface TaxLot {
  id: string;
  token: string;
  amount: string;
  costBasisUSD: number;
  acquiredAt: number;
}

export interface TaxEvent extends Transaction {
  costBasisUSD?: number;
  gainUSD?: number;
  isShortTerm?: boolean;
}

export interface TaxSummary {
  totalProceeds: number;
  totalCostBasis: number;
  totalGains: number;
  shortTermGains: number;
  longTermGains: number;
}

export interface TaxReport {
  jurisdiction: TaxJurisdiction;
  method: CostBasisMethod;
  events: TaxEvent[];
  summary: TaxSummary;
  generatedAt: number;
}

class TaxAnalyticsServiceClass {
  private static instance: TaxAnalyticsServiceClass | null = null;
  private defaultJurisdiction: TaxJurisdiction = 'US';
  private defaultMethod: CostBasisMethod = 'FIFO';
  private constructor() {}
  static getInstance(): TaxAnalyticsServiceClass {
    if (!TaxAnalyticsServiceClass.instance) TaxAnalyticsServiceClass.instance = new TaxAnalyticsServiceClass();
    return TaxAnalyticsServiceClass.instance;
  }

  setDefaultJurisdiction(jurisdiction: TaxJurisdiction): void { this.defaultJurisdiction = jurisdiction; }
  setDefaultMethod(method: CostBasisMethod): void { this.defaultMethod = method; }
  getDefaultJurisdiction(): TaxJurisdiction { return this.defaultJurisdiction; }
  getDefaultMethod(): CostBasisMethod { return this.defaultMethod; }

  importTransaction(_tx: Transaction): { success: false; error: string } {
    return { success: false, error: 'Tax transaction import is not supported by the canonical MasterWallet backend' };
  }

  generateReport(_year: number, _method?: CostBasisMethod): { success: false; error: string } {
    return { success: false, error: 'Tax report generation is not supported by the canonical MasterWallet backend' };
  }

  getSummary(): TaxSummary {
    // No canonical backend route for tax analytics — fail-closed (do not return
    // fabricated zeros that could be mistaken for real computed gains).
    throw new Error('Tax summary is not supported by the canonical MasterWallet backend');
  }
}

export const TaxAnalyticsService = TaxAnalyticsServiceClass;
export default TaxAnalyticsServiceClass.getInstance();
