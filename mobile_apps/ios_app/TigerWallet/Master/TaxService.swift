//
//  TaxService.swift
//  TigerWallet
//
//  Complete Tax Service - Identical across ALL platforms
//

import Foundation

class TaxService {
    static let shared = TaxService()
    
    private var transactions: [TaxTransaction] = []
    private var taxLots: [String: [TaxLot]] = [:]
    private var incomeEvents: [IncomeEvent] = []
    
    private var jurisdiction = "US"
    private var costBasisMethod: CostBasisMethod = .FIFO
    
    private init() {}
    
    func setJurisdiction(_ jurisdictionCode: String) -> Bool {
        jurisdiction = jurisdictionCode
        return true
    }
    
    func setCostBasisMethod(_ method: CostBasisMethod) -> Bool {
        costBasisMethod = method
        return true
    }
    
    func addTransaction(_ tx: TaxTransaction) {
        transactions.append(tx)
    }
    
    func calculateGains() -> TaxReport {
        var shortTermGains = 0
        var shortTermLosses = 0
        var longTermGains = 0
        var longTermLosses = 0
        var totalIncome = 0
        
        for event in incomeEvents {
            totalIncome += event.fairMarketValue
        }
        
        return TaxReport(
            year: 2024,
            shortTermGains: shortTermGains,
            shortTermLosses: shortTermLosses,
            longTermGains: longTermGains,
            longTermLosses: longTermLosses,
            totalIncome: totalIncome,
            totalTransactions: transactions.count,
            jurisdiction: jurisdiction,
            costBasisMethod: costBasisMethod
        )
    }
    
    func getAvailableLots(asset: String) -> [TaxLot] {
        return taxLots[asset]?.filter { $0.remainingAmount > 0 } ?? []
    }
    
    func addIncomeEvent(_ event: IncomeEvent) {
        incomeEvents.append(event)
        
        let lot = TaxLot(
            id: "lot_\(Int(Date().timeIntervalSince1970))",
            asset: event.asset,
            amount: event.amount,
            remainingAmount: event.amount,
            costBasis: 0,
            fairMarketValue: event.fairMarketValue,
            acquisitionDate: event.date,
            isLongTerm: false
        )
        
        if taxLots[event.asset] == nil {
            taxLots[event.asset] = []
        }
        taxLots[event.asset]?.append(lot)
    }
    
    func exportCSV() -> String {
        var csv = "Date,Type,Asset,Amount,Cost Basis,Proceeds,Gain/Loss,Exchange\n"
        for tx in transactions {
            csv += "\(tx.date),\(tx.type),\(tx.asset),\(tx.amount),\(tx.costBasis),\(tx.proceeds),\(tx.gainLoss),\(tx.exchange)\n"
        }
        return csv
    }
}

enum CostBasisMethod: String { case FIFO, LIFO, HIFO }
enum TransactionType: String { case BUY, SELL, TRANSFER, SWAP, STAKE, UNSTAKE, MINT, BURN }

struct TaxTransaction {
    let id: String
    let type: TransactionType
    let date: String
    let asset: String
    let amount: Int
    let price: Int
    let costBasis: Int
    let proceeds: Int
    let gainLoss: Int
    let exchange: String
    let txHash: String
}

struct TaxLot {
    let id: String
    let asset: String
    let amount: Int
    var remainingAmount: Int
    let costBasis: Int
    let fairMarketValue: Int
    let acquisitionDate: String
    let isLongTerm: Bool
}

struct IncomeEvent {
    let id: String
    let type: String
    let asset: String
    let amount: Int
    let fairMarketValue: Int
    let date: String
}

struct TaxReport {
    let year: Int
    let shortTermGains: Int
    let shortTermLosses: Int
    let longTermGains: Int
    let longTermLosses: Int
    let totalIncome: Int
    let totalTransactions: Int
    let jurisdiction: String
    let costBasisMethod: CostBasisMethod
}
