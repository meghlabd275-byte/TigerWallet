// MasterWallet Tax Analytics Service (iOS)
// Tax reporting and analytics
// Production-ready

import Foundation

class TaxAnalyticsService {
    
    private var transactions: [String: [TaxTransaction]] = [:]
    private var reports: [TaxReport] = []
    private var config: TaxConfig = TaxConfig.default()
    
    // MARK: - Initialize
    
    func initialize() -> Bool {
        loadTransactions()
        loadReports()
        loadConfig()
        return true
    }
    
    // MARK: - Transactions
    
    func addTransaction(_ transaction: TaxTransaction) {
        if transactions[transaction.walletAddress] == nil {
            transactions[transaction.walletAddress] = []
        }
        transactions[transaction.walletAddress]?.append(transaction)
        saveTransactions()
    }
    
    func addTransactions(_ newTransactions: [TaxTransaction]) {
        for transaction in newTransactions {
            addTransaction(transaction)
        }
    }
    
    func getTransactions(walletAddress: String, startDate: Double? = nil, endDate: Double? = nil, type: String? = nil) -> [TaxTransaction] {
        guard let walletTransactions = transactions[walletAddress] else { return [] }
        
        return walletTransactions.filter { tx in
            let dateMatch = (startDate == nil || tx.timestamp >= startDate!) &&
                    (endDate == nil || tx.timestamp <= endDate!)
            let typeMatch = type == nil || tx.type == type
            return dateMatch && typeMatch
        }
    }
    
    // MARK: - Capital Gains
    
    func calculateGainsLosses(walletAddress: String, taxYear: Int) -> [CapitalGainLoss] {
        let yearStart = getYearStartTimestamp(year: taxYear)
        let yearEnd = getYearEndTimestamp(year: taxYear)
        
        let yearTransactions = getTransactions(walletAddress: walletAddress, startDate: yearStart, endDate: yearEnd)
        
        var gains: [CapitalGainLoss] = []
        let assets = Set(yearTransactions.map { $0.asset })
        
        for asset in assets {
            let assetTransactions = yearTransactions.filter { $0.asset == asset }
            let buys = assetTransactions.filter { $0.type == "buy" || $0.type == "transfer_in" }
            let sells = assetTransactions.filter { $0.type == "sell" || $0.type == "transfer_out" }
            
            var lots = buys.map { Lot(quantity: $0.quantity, costPerUnit: $0.priceUSD, timestamp: $0.timestamp) }
            
            for sell in sells {
                var remaining = sell.quantity
                var costBasis = 0.0
                
                while remaining > 0 && !lots.isEmpty {
                    let index = lots.startIndex
                    var lot = lots[index]
                    let take = min(remaining, lot.quantity)
                    
                    costBasis += take * lot.costPerUnit
                    remaining -= take
                    lot.quantity -= take
                    
                    if lot.quantity <= 0 {
                        lots.remove(at: index)
                    }
                }
                
                let proceeds = sell.quantity * sell.priceUSD - sell.feeUSD
                let gainLoss = proceeds - costBasis
                let term = "short_term"
                
                gains.append(CapitalGainLoss(
                    asset: asset,
                    proceeds: proceeds,
                    costBasis: costBasis,
                    gainLoss: gainLoss,
                    term: term,
                    disposalDate: sell.timestamp
                ))
            }
        }
        
        return gains
    }
    
    // MARK: - Tax Report
    
    func generateTaxReport(walletAddress: String, taxYear: Int) -> TaxReport {
        let gains = calculateGainsLosses(walletAddress: walletAddress, taxYear: taxYear)
        
        var totalProceeds = 0.0
        var totalCostBasis = 0.0
        var shortTermGainLoss = 0.0
        var longTermGainLoss = 0.0
        var gainsByAsset: [String: Double] = [:]
        
        for gain in gains {
            totalProceeds += gain.proceeds
            totalCostBasis += gain.costBasis
            
            if gain.term == "short_term" {
                shortTermGainLoss += gain.gainLoss
            } else {
                longTermGainLoss += gain.gainLoss
            }
            
            gainsByAsset[gain.asset] = (gainsByAsset[gain.asset] ?? 0) + gain.gainLoss
        }
        
        let yearStart = getYearStartTimestamp(year: taxYear)
        let yearEnd = getYearEndTimestamp(year: taxYear)
        let yearTransactions = getTransactions(walletAddress: walletAddress, startDate: yearStart, endDate: yearEnd)
        
        var stakingRewards = 0.0
        var interestIncome = 0.0
        var defiIncome = 0.0
        
        for tx in yearTransactions {
            switch tx.type {
            case "staking", "reward":
                stakingRewards += tx.quantity * tx.priceUSD
            case "interest":
                interestIncome += tx.quantity * tx.priceUSD
            case "defi":
                defiIncome += tx.quantity * tx.priceUSD
            default:
                break
            }
        }
        
        let income = stakingRewards + interestIncome + defiIncome
        let totalGainLoss = shortTermGainLoss + longTermGainLoss
        let totalTaxableIncome = totalGainLoss + income
        
        let shortTermTax = max(0, shortTermGainLoss * config.shortTermRate)
        let longTermTax = max(0, longTermGainLoss * config.longTermRate)
        let incomeTax = income * config.incomeTaxRate
        
        let report = TaxReport(
            reportId: "tax_\(walletAddress)_\(taxYear)",
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
            generatedAt: Date().timeIntervalSince1970 * 1000
        )
        
        reports.append(report)
        saveReports()
        
        return report
    }
    
    func getReport(reportId: String) -> TaxReport? {
        return reports.first { $0.reportId == reportId }
    }
    
    func getReports(walletAddress: String? = nil, year: Int? = nil) -> [TaxReport] {
        return reports.filter { report in
            let walletMatch = walletAddress == nil || report.walletAddress == walletAddress
            let yearMatch = year == nil || report.taxYear == year
            return walletMatch && yearMatch
        }
    }
    
    // MARK: - Configuration
    
    func setConfig(_ newConfig: TaxConfig) {
        config = newConfig
        saveConfig()
    }
    
    func getConfig() -> TaxConfig {
        return config
    }
    
    // MARK: - Export
    
    func exportToCSV(reportId: String) -> String? {
        guard let report = getReport(reportId: reportId) else { return nil }
        
        var csv = "Asset,Proceeds,Cost Basis,Gain/Loss,Term,Disposal Date\n"
        
        for tx in report.transactions {
            csv += "\(tx.asset),\(tx.proceeds),\(tx.costBasis),\(tx.gainLoss),\(tx.term),\(tx.disposalDate)\n"
        }
        
        csv += "\nTotal Proceeds,\(report.totalProceeds)\n"
        csv += "Total Cost Basis,\(report.totalCostBasis)\n"
        csv += "Total Gain/Loss,\(report.totalGainLoss)\n"
        csv += "Short-term Gain/Loss,\(report.shortTermGainLoss)\n"
        csv += "Long-term Gain/Loss,\(report.longTermGainLoss)\n"
        csv += "Income,\(report.income)\n"
        csv += "Total Taxable Income,\(report.totalTaxableIncome)\n"
        csv += "Total Tax,\(report.totalTax)\n"
        
        return csv
    }
    
    // MARK: - Stats
    
    func getStats() -> TaxStats {
        let totalTransactions = transactions.values.reduce(0) { $0 + $1.count }
        return TaxStats(totalTransactions: totalTransactions, totalReports: reports.count)
    }
    
    // MARK: - Private Methods
    
    private func getYearStartTimestamp(year: Int) -> Double {
        var calendar = Calendar.current
        calendar.timeZone = TimeZone(identifier: "UTC")!
        var components = DateComponents()
        components.year = year
        components.month = 1
        components.day = 1
        components.hour = 0
        components.minute = 0
        components.second = 0
        return calendar.date(from: components)!.timeIntervalSince1970 * 1000
    }
    
    private func getYearEndTimestamp(year: Int) -> Double {
        var calendar = Calendar.current
        calendar.timeZone = TimeZone(identifier: "UTC")!
        var components = DateComponents()
        components.year = year
        components.month = 12
        components.day = 31
        components.hour = 23
        components.minute = 59
        components.second = 59
        return calendar.date(from: components)!.timeIntervalSince1970 * 1000
    }
    
    private func loadTransactions() {
        if let data = UserDefaults.standard.data(forKey: "taxTransactions"),
           let decoded = try? JSONDecoder().decode([String: [TaxTransaction]].self, from: data) {
            transactions = decoded
        }
    }
    
    private func saveTransactions() {
        if let encoded = try? JSONEncoder().encode(transactions) {
            UserDefaults.standard.set(encoded, forKey: "taxTransactions")
        }
    }
    
    private func loadReports() {
        if let data = UserDefaults.standard.data(forKey: "taxReports"),
           let decoded = try? JSONDecoder().decode([TaxReport].self, from: data) {
            reports = decoded
        }
    }
    
    private func saveReports() {
        if let encoded = try? JSONEncoder().encode(reports) {
            UserDefaults.standard.set(encoded, forKey: "taxReports")
        }
    }
    
    private func loadConfig() {
        if let data = UserDefaults.standard.data(forKey: "taxConfig"),
           let decoded = try? JSONDecoder().decode(TaxConfig.self, from: data) {
            config = decoded
        }
    }
    
    private func saveConfig() {
        if let encoded = try? JSONEncoder().encode(config) {
            UserDefaults.standard.set(encoded, forKey: "taxConfig")
        }
    }
}

// MARK: - Models

struct TaxTransaction: Codable {
    let id: String
    let walletAddress: String
    let hash: String
    let type: String
    let asset: String
    let quantity: Double
    let priceUSD: Double
    let feeUSD: Double
    let chainId: String
    let timestamp: Double
    let counterpart: String
    let notes: String
}

struct CapitalGainLoss: Codable {
    let asset: String
    let proceeds: Double
    let costBasis: Double
    let gainLoss: Double
    let term: String
    let disposalDate: Double
}

struct TaxReport: Codable {
    let reportId: String
    let walletAddress: String
    let taxYear: Int
    let totalProceeds: Double
    let totalCostBasis: Double
    let totalGainLoss: Double
    let shortTermGainLoss: Double
    let longTermGainLoss: Double
    let income: Double
    let stakingRewards: Double
    let interestIncome: Double
    let defiIncome: Double
    let totalTaxableIncome: Double
    let shortTermTax: Double
    let longTermTax: Double
    let incomeTax: Double
    let totalTax: Double
    let transactions: [CapitalGainLoss]
    let gainsByAsset: [String: Double]
    let generatedAt: Double
}

struct TaxConfig: Codable {
    let method: String
    let jurisdiction: String
    let shortTermRate: Double
    let longTermRate: Double
    let incomeTaxRate: Double
    let includeStakingRewards: Bool
    let includeDeFiIncome: Bool
    let includeNFTs: Bool
    let applyWashSaleRules: Bool
    let ignoredAssets: [String]
    
    static func `default`() -> TaxConfig {
        return TaxConfig(
            method: "FIFO",
            jurisdiction: "US",
            shortTermRate: 0.37,
            longTermRate: 0.20,
            incomeTaxRate: 0.22,
            includeStakingRewards: true,
            includeDeFiIncome: true,
            includeNFTs: true,
            applyWashSaleRules: true,
            ignoredAssets: []
        )
    }
}

private struct Lot {
    var quantity: Double
    let costPerUnit: Double
    let timestamp: Double
}

struct TaxStats {
    let totalTransactions: Int
    let totalReports: Int
}
