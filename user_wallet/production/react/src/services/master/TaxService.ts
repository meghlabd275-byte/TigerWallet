/**
 * Tax Service - React/Web Implementation
 * Identical across ALL platforms
 */

export enum CostBasisMethod {
  FIFO = 'fifo',
  LIFO = 'lifo',
  HIFO = 'hifo',
}

export enum TransactionType {
  BUY = 'buy',
  SELL = 'sell',
  TRANSFER = 'transfer',
  SWAP = 'swap',
  STAKE = 'stake',
  UNSTAKE = 'unstake',
  MINT = 'mint',
  BURN = 'burn',
}

class TaxService {
  private static instance: TaxService;
  private transactions: TaxTransaction[] = [];
  private taxLots: Map<string, TaxLot[]> = new Map();
  private incomeEvents: IncomeEvent[] = [];
  private jurisdiction = 'US';
  private costBasisMethod: CostBasisMethod = CostBasisMethod.FIFO;

  static getInstance(): TaxService {
    if (!TaxService.instance) {
      TaxService.instance = new TaxService();
    }
    return TaxService.instance;
  }

  setJurisdiction(jurisdictionCode: string): boolean {
    this.jurisdiction = jurisdictionCode;
    return true;
  }

  setCostBasisMethod(method: CostBasisMethod): boolean {
    this.costBasisMethod = method;
    return true;
  }

  addTransaction(tx: TaxTransaction): void {
    this.transactions.push(tx);
  }

  calculateGains(): TaxReport {
    let shortTermGains = 0;
    let shortTermLosses = 0;
    let longTermGains = 0;
    let longTermLosses = 0;
    let totalIncome = 0;

    for (const event of this.incomeEvents) {
      totalIncome += event.fairMarketValue;
    }

    return {
      year: 2024,
      shortTermGains,
      shortTermLosses,
      longTermGains,
      longTermLosses,
      totalIncome,
      totalTransactions: this.transactions.length,
      jurisdiction: this.jurisdiction,
      costBasisMethod: this.costBasisMethod,
    };
  }

  getAvailableLots(asset: string): TaxLot[] {
    return this.taxLots.get(asset)?.filter((lot) => lot.remainingAmount > 0) ?? [];
  }

  addIncomeEvent(event: IncomeEvent): void {
    this.incomeEvents.push(event);

    const lot: TaxLot = {
      id: `lot_${Date.now()}`,
      asset: event.asset,
      amount: event.amount,
      remainingAmount: event.amount,
      costBasis: 0,
      fairMarketValue: event.fairMarketValue,
      acquisitionDate: event.date,
      isLongTerm: false,
    };

    const existing = this.taxLots.get(event.asset) ?? [];
    this.taxLots.set(event.asset, [...existing, lot]);
  }

  exportCSV(): string {
    let csv = 'Date,Type,Asset,Amount,Cost Basis,Proceeds,Gain/Loss,Exchange\n';
    for (const tx of this.transactions) {
      csv += `${tx.date},${tx.type},${tx.asset},${tx.amount},${tx.costBasis},${tx.proceeds},${tx.gainLoss},${tx.exchange}\n`;
    }
    return csv;
  }
}

export interface TaxTransaction {
  id: string;
  type: TransactionType;
  date: string;
  asset: string;
  amount: number;
  price: number;
  costBasis: number;
  proceeds: number;
  gainLoss: number;
  exchange: string;
  txHash: string;
}

export interface TaxLot {
  id: string;
  asset: string;
  amount: number;
  remainingAmount: number;
  costBasis: number;
  fairMarketValue: number;
  acquisitionDate: string;
  isLongTerm: boolean;
}

export interface IncomeEvent {
  id: string;
  type: string;
  asset: string;
  amount: number;
  fairMarketValue: number;
  date: string;
}

export interface TaxReport {
  year: number;
  shortTermGains: number;
  shortTermLosses: number;
  longTermGains: number;
  longTermLosses: number;
  totalIncome: number;
  totalTransactions: number;
  jurisdiction: string;
  costBasisMethod: CostBasisMethod;
}

export default TaxService.getInstance();
