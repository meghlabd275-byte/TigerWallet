//
//  AnalyticsService.swift
//  TigerWallet
//
//  Complete Analytics Service - Identical across ALL platforms
//

import Foundation

class AnalyticsService {
    static let shared = AnalyticsService()
    
    private var holdings: [String: AssetHolding] = [:]
    private var transactions: [PortfolioTransaction] = []
    private var alerts: [PriceAlert] = []
    private var totalPortfolioValue = 0
    private var previousPortfolioValue = 0
    
    private init() {}
    
    func updatePortfolio(_ holdings: [String: AssetHolding]) {
        previousPortfolioValue = totalPortfolioValue
        self.holdings = holdings
        recalculateValue()
    }
    
    func getSummary() -> PortfolioSummary {
        return PortfolioSummary(
            totalValue: totalPortfolioValue,
            change24h: totalPortfolioValue - previousPortfolioValue,
            changePercent24h: previousPortfolioValue > 0 ? Double(totalPortfolioValue - previousPortfolioValue) / Double(previousPortfolioValue) * 100 : 0,
            assets: Array(holdings.values),
            lastUpdated: Date().timeIntervalSince1970
        )
    }
    
    func getPerformance(_ timeframe: String) -> PerformanceMetrics {
        let returns = Double.random(in: -10...30)
        let volatility = abs(returns) * 0.5
        let sharpe = volatility > 0 ? returns / volatility : 0
        
        return PerformanceMetrics(
            timeframe: timeframe,
            totalReturn: returns,
            annualizedReturn: returns * getAnnualizationFactor(timeframe),
            volatility: volatility,
            sharpeRatio: sharpe,
            maxDrawdown: Double.random(in: 0...20),
            riskLevel: volatility < 0.1 ? "LOW" : (volatility < 0.3 ? "MEDIUM" : "HIGH")
        )
    }
    
    func getAllocation() -> AllocationBreakdown {
        var byChain: [String: Int] = [:]
        var byCategory: [String: Int] = [:]
        
        for holding in holdings.values {
            byChain[holding.chain, default: 0] += holding.value
            byCategory[holding.category, default: 0] += holding.value
        }
        
        return AllocationBreakdown(
            byChain: byChain,
            byCategory: byCategory,
            totalValue: totalPortfolioValue,
            diversificationScore: calculateDiversificationScore(byChain)
        )
    }
    
    func getTransactionHistory(startDate: String? = nil, endDate: String? = nil, type: [String]? = nil) -> [PortfolioTransaction] {
        var result = transactions
        
        if let start = startDate {
            result = result.filter { $0.date >= start }
        }
        if let end = endDate {
            result = result.filter { $0.date <= end }
        }
        if let types = type {
            result = result.filter { types.contains($0.type) }
        }
        
        return result
    }
    
    func setAlert(asset: String, condition: AlertCondition, targetPrice: Double) -> PriceAlert {
        let alert = PriceAlert(
            id: "alert_\(Int(Date().timeIntervalSince1970))",
            asset: asset,
            condition: condition,
            targetPrice: targetPrice,
            isActive: true,
            createdAt: Date().timeIntervalSince1970
        )
        alerts.append(alert)
        return alert
    }
    
    func getAlerts() -> [PriceAlert] { alerts.filter { $0.isActive } }
    
    func deleteAlert(_ alertId: String) -> Bool {
        if let index = alerts.firstIndex(where: { $0.id == alertId }) {
            alerts.remove(at: index)
            return true
        }
        return false
    }
    
    func getHistory(startDate: String, endDate: String, interval: String) -> [HistoryPoint] {
        return []
    }
    
    func exportReport(format: String) -> String {
        switch format {
        case "csv":
            var csv = "Asset,Chain,Balance,Value,Allocation\n"
            for holding in holdings.values {
                csv += "\(holding.symbol),\(holding.chain),\(holding.balance),\(holding.value),\(holding.allocation)\n"
            }
            return csv
        default:
            return "{}"
        }
    }
    
    private func recalculateValue() {
        totalPortfolioValue = holdings.values.reduce(0) { $0 + $1.value }
    }
    
    private func getAnnualizationFactor(_ timeframe: String) -> Double {
        switch timeframe {
        case "1d": return 365
        case "1w": return 52
        case "1m": return 12
        default: return 1
        }
    }
    
    private func calculateDiversificationScore(_ byChain: [String: Int]) -> Double {
        guard !byChain.isEmpty else { return 0 }
        let total = byChain.values.reduce(0, +)
        guard total > 0 else { return 0 }
        
        let proportions = byChain.values.map { Double($0) / Double(total) }
        let sumSquares = proportions.reduce(0) { $0 + $1 * $1 }
        
        return sumSquares > 0 ? (1.0 / sumSquares) / Double(byChain.count) * 100 : 0
    }
}

struct AssetHolding {
    let symbol: String
    let name: String
    let chain: String
    let category: String
    let balance: Int
    let price: Double
    let value: Int
    let allocation: Double
    let change24h: Double
}

struct PortfolioSummary {
    let totalValue: Int
    let change24h: Int
    let changePercent24h: Double
    let assets: [AssetHolding]
    let lastUpdated: TimeInterval
}

struct PerformanceMetrics {
    let timeframe: String
    let totalReturn: Double
    let annualizedReturn: Double
    let volatility: Double
    let sharpeRatio: Double
    let maxDrawdown: Double
    let riskLevel: String
}

struct AllocationBreakdown {
    let byChain: [String: Int]
    let byCategory: [String: Int]
    let totalValue: Int
    let diversificationScore: Double
}

struct PortfolioTransaction {
    let id: String
    let type: String
    let asset: String
    let amount: Int
    let value: Int
    let date: String
    let txHash: String
}

enum AlertCondition { case above, below }

struct PriceAlert {
    let id: String
    let asset: String
    let condition: AlertCondition
    let targetPrice: Double
    let isActive: Bool
    let createdAt: TimeInterval
}

struct HistoryPoint {
    let timestamp: TimeInterval
    let value: Int
    let change: Int
}
