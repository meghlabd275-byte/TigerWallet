//
//  PriceService.swift
//  TigerWallet
//
//  Production-Ready Price Service with Real-time Updates
//

import Foundation
import Combine

// MARK: - Price Model

struct PriceInfo: Codable {
    let symbol: String
    let name: String
    let price: Double
    let change24h: Double
    let changePercent24h: Double
    let marketCap: Double
    let volume24h: Double
    let high24h: Double
    let low24h: Double
    let lastUpdated: Date
    
    var isPositive: Bool {
        changePercent24h >= 0
    }
    
    var formattedPrice: String {
        if price >= 1 {
            return String(format: "$%.2f", price)
        } else {
            return String(format: "$%.6f", price)
        }
    }
    
    var formattedChange: String {
        let sign = changePercent24h >= 0 ? "+" : ""
        return String(format: "%@%.2f%%", sign, changePercent24h)
    }
    
    var formattedMarketCap: String {
        if marketCap >= 1_000_000_000_000 {
            return String(format: "$%.2fT", marketCap / 1_000_000_000_000)
        } else if marketCap >= 1_000_000_000 {
            return String(format: "$%.2fB", marketCap / 1_000_000_000)
        } else if marketCap >= 1_000_000 {
            return String(format: "$%.2fM", marketCap / 1_000_000)
        }
        return String(format: "$%.0f", marketCap)
    }
}

struct PortfolioItem: Codable {
    let token: Token
    let priceInfo: PriceInfo
    let balance: Double
    let usdValue: Double
    
    var formattedBalance: String {
        if balance >= 1 {
            return String(format: "%.4f", balance)
        } else {
            return String(format: "%.6f", balance)
        }
    }
    
    var formattedValue: String {
        String(format: "$%.2f", usdValue)
    }
}

// MARK: - Price Service

class PriceService {
    static let shared = PriceService()
    
    private var updateTimer: Timer?
    private var cancellables = Set<AnyCancellable>()
    
    // Publishers
    let pricesPublisher = PassthroughSubject<[String: PriceInfo], Never>()
    let portfolioPublisher = PassthroughSubject<[PortfolioItem], Never>()
    let priceUpdatePublisher = PassthroughSubject<PriceInfo, Never>()
    
    // Cached prices
    private var cachedPrices: [String: PriceInfo] = [:]
    private let cacheExpiry: TimeInterval = 60 // 1 minute
    
    private init() {}
    
    // MARK: - Start/Stop Updates
    
    func startPriceUpdates(interval: TimeInterval = 30) {
        // Initial fetch
        Task {
            await fetchPrices()
        }
        
        // Setup timer for periodic updates
        updateTimer?.invalidate()
        updateTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] _ in
            Task {
                await self?.fetchPrices()
            }
        }
    }
    
    func stopPriceUpdates() {
        updateTimer?.invalidate()
        updateTimer = nil
    }
    
    // MARK: - Fetch Prices
    
    func fetchPrices() async {
        do {
            let symbols = getTrackedSymbols()
            let prices = try await fetchPricesFromAPI(symbols: symbols)
            
            // Update cache
            cachedPrices = prices
            
            // Publish updates
            pricesPublisher.send(prices)
            
            // Update individual prices
            for price in prices.values {
                priceUpdatePublisher.send(price)
            }
            
        } catch {
            print("Failed to fetch prices: \(error)")
        }
    }
    
    func fetchPrice(for symbol: String) async throws -> PriceInfo {
        // Check cache first
        if let cached = cachedPrices[symbol],
           Date().timeIntervalSince(cached.lastUpdated) < cacheExpiry {
            return cached
        }
        
        // Fetch from API
        let prices = try await fetchPricesFromAPI(symbols: [symbol])
        
        guard let price = prices[symbol] else {
            throw PriceError.notFound
        }
        
        cachedPrices[symbol] = price
        return price
    }
    
    // MARK: - API Call
    
    private func fetchPricesFromAPI(symbols: [String]) async throws -> [String: PriceInfo] {
        let symbolsParam = symbols.joined(separator: ",")
        
        // Using CoinGecko API (free tier)
        let urlString = "https://api.coingecko.com/api/v3/simple/price?ids=\(symbolsParam)&vs_currencies=usd&include_24hr_change=true&include_market_cap=true&include_24hr_vol=true&include_high_low=true"
        
        guard let url = URL(string: urlString) else {
            throw PriceError.invalidURL
        }
        
        let (data, response) = try await URLSession.shared.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw PriceError.networkError
        }
        
        // Parse response
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        
        var prices: [String: PriceInfo] = [:]
        
        for (coinId, value) in json ?? [:] {
            guard let coinData = value as? [String: Any] else { continue }
            
            let price = coinData["usd"] as? Double ?? 0
            let change24h = coinData["usd_24h_change"] as? Double ?? 0
            let marketCap = coinData["usd_market_cap"] as? Double ?? 0
            let volume = coinData["usd_24h_vol"] as? Double ?? 0
            let high = coinData["usd_high_24h"] as? Double ?? price
            let low = coinData["usd_low_24h"] as? Double ?? price
            
            let priceInfo = PriceInfo(
                symbol: coinId.uppercased(),
                name: coinId.capitalized,
                price: price,
                change24h: change24h,
                changePercent24h: change24h,
                marketCap: marketCap,
                volume24h: volume,
                high24h: high,
                low24h: low,
                lastUpdated: Date()
            )
            
            prices[coinId] = priceInfo
        }
        
        return prices
    }
    
    // MARK: - Portfolio
    
    func fetchPortfolio() async throws -> [PortfolioItem] {
        let wallets = BlockchainService.shared.getWallets()
        
        var portfolio: [PortfolioItem] = []
        
        for wallet in wallets {
            // Get balance
            let balance = try await BlockchainService.shared.getBalance(for: wallet)
            
            // Get price
            let symbol = wallet.blockchain.symbol.lowercased()
            var priceInfo: PriceInfo
            
            do {
                priceInfo = try await fetchPrice(for: symbol)
            } catch {
                // Use default price if not found
                priceInfo = PriceInfo(
                    symbol: symbol.uppercased(),
                    name: symbol.capitalized,
                    price: 0,
                    change24h: 0,
                    changePercent24h: 0,
                    marketCap: 0,
                    volume24h: 0,
                    high24h: 0,
                    low24h: 0,
                    lastUpdated: Date()
                )
            }
            
            let usdValue = balance * priceInfo.price
            
            let token = Token(
                id: wallet.id,
                symbol: wallet.blockchain.symbol,
                name: wallet.blockchain.rawValue.capitalized,
                decimals: wallet.blockchain.decimals,
                contractAddress: nil,
                blockchain: wallet.blockchain,
                logoURL: nil,
                price: priceInfo.price,
                balance: balance,
                usdValue: usdValue
            )
            
            let item = PortfolioItem(
                token: token,
                priceInfo: priceInfo,
                balance: balance,
                usdValue: usdValue
            )
            
            portfolio.append(item)
        }
        
        portfolioPublisher.send(portfolio)
        
        return portfolio
    }
    
    // MARK: - Tracked Symbols
    
    private func getTrackedSymbols() -> [String] {
        // Return list of commonly tracked cryptocurrencies
        return [
            "bitcoin", "ethereum", "tether", "usd-coin", "binancecoin",
            "ripple", "cardano", "solana", "dogecoin", "polkadot",
            "polygon", "avalanche-2", "chainlink", "uniswap", "litecoin",
            "near", "aptos", "sui", "toncoin", "cosmos"
        ]
    }
    
    // MARK: - Price Conversion
    
    func convert(amount: Double, from: String, to: String) async throws -> Double {
        let fromPrice = try await fetchPrice(for: from.lowercased())
        let toPrice = try await fetchPrice(for: to.lowercased())
        
        // Convert to USD first, then to target currency
        let usdValue = amount * fromPrice.price
        return usdValue / toPrice.price
    }
    
    // MARK: - Historical Data
    
    func getPriceHistory(symbol: String, days: Int) async throws -> [PricePoint] {
        let urlString = "https://api.coingecko.com/api/v3/coins/\(symbol)/market_chart?vs_currency=usd&days=\(days)"
        
        guard let url = URL(string: urlString) else {
            throw PriceError.invalidURL
        }
        
        let (data, response) = try await URLSession.shared.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw PriceError.networkError
        }
        
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        
        guard let prices = json?["prices"] as? [[Double]] else {
            throw PriceError.invalidResponse
        }
        
        return prices.compactMap { point in
            guard point.count >= 2 else { return nil }
            return PricePoint(timestamp: Date(timeIntervalSince1970: point[0] / 1000), price: point[1])
        }
    }
    
    // MARK: - Search
    
    func searchCoins(query: String) async throws -> [CoinSearchResult] {
        let urlString = "https://api.coingecko.com/api/v3/search?query=\(query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        
        guard let url = URL(string: urlString) else {
            throw PriceError.invalidURL
        }
        
        let (data, response) = try await URLSession.shared.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw PriceError.networkError
        }
        
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        
        guard let coins = json?["coins"] as? [[String: Any]] else {
            return []
        }
        
        return coins.prefix(10).compactMap { coin in
            guard let id = coin["id"] as? String,
                  let name = coin["name"] as? String,
                  let symbol = coin["symbol"] as? String,
                  let thumb = coin["thumb"] as? String else {
                return nil
            }
            
            return CoinSearchResult(
                id: id,
                name: name,
                symbol: symbol,
                thumb: thumb
            )
        }
    }
}

// MARK: - Supporting Types

struct PricePoint: Codable {
    let timestamp: Date
    let price: Double
}

struct CoinSearchResult: Codable {
    let id: String
    let name: String
    let symbol: String
    let thumb: String
}

enum PriceError: Error {
    case invalidURL
    case networkError
    case invalidResponse
    case notFound
    case rateLimited
}
