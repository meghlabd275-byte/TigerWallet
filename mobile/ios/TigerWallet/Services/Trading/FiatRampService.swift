//
//  FiatRampService.swift
//  TigerWallet
//
//  Fiat On-Ramp Service - iOS Implementation
//

import Foundation

struct FiatProvider {
    var id: String
    var name: String
    var logo: String
    var supportedFiat: [String]
    var supportedCrypto: [String]
    var minAmount: Double
    var maxAmount: Double
    var feePercent: Double
    var processingTime: String
    var isAvailable: Bool
}

struct FiatOrder {
    var id: String
    var providerId: String
    var providerName: String
    var side: String
    var fiatCurrency: String
    var cryptoCurrency: String
    var fiatAmount: Double
    var cryptoAmount: Double
    var fee: Double
    var status: String
}

class FiatRampService {
    
    func getProviders() -> [FiatProvider] {
        return [
            FiatProvider(
                id: "provider_1",
                name: "MoonPay",
                logo: "🌙",
                supportedFiat: ["USD", "EUR", "GBP", "AUD"],
                supportedCrypto: ["BTC", "ETH", "USDT", "BNB", "SOL"],
                minAmount: 30,
                maxAmount: 50000,
                feePercent: 2.5,
                processingTime: "5-30 min",
                isAvailable: true
            ),
            FiatProvider(
                id: "provider_2",
                name: "Simplex",
                logo: "💳",
                supportedFiat: ["USD", "EUR", "GBP"],
                supportedCrypto: ["BTC", "ETH", "USDT"],
                minAmount: 50,
                maxAmount: 25000,
                feePercent: 3.5,
                processingTime: "10-60 min",
                isAvailable: true
            ),
            FiatProvider(
                id: "provider_3",
                name: "Transak",
                logo: "🔄",
                supportedFiat: ["USD", "EUR", "GBP", "INR"],
                supportedCrypto: ["BTC", "ETH", "USDT", "MATIC", "AVAX"],
                minAmount: 20,
                maxAmount: 100000,
                feePercent: 2.0,
                processingTime: "15-45 min",
                isAvailable: true
            ),
            FiatProvider(
                id: "provider_4",
                name: "OnRamper",
                logo: "📱",
                supportedFiat: ["USD", "EUR", "GBP", "AUD"],
                supportedCrypto: ["BTC", "ETH", "USDT", "ADA", "DOT"],
                minAmount: 25,
                maxAmount: 75000,
                feePercent: 1.8,
                processingTime: "5-20 min",
                isAvailable: true
            )
        ]
    }
    
    func calculateRate(providerId: String, cryptoCurrency: String, fiatAmount: Double) -> Double {
        let baseRates: [String: Double] = [
            "BTC": 43250, "ETH": 2280, "USDT": 1, "USDC": 1, "BNB": 312.5, "SOL": 98.75, "ADA": 0.58
        ]
        
        let baseRate = baseRates[cryptoCurrency] ?? 1.0
        var fee = 0.025
        
        switch providerId {
        case "provider_2": fee = 0.035
        case "provider_3": fee = 0.020
        case "provider_4": fee = 0.018
        default: break
        }
        
        return (fiatAmount * (1 - fee)) / baseRate
    }
    
    func createOrder(providerId: String, side: String, fiatCurrency: String, 
                    cryptoCurrency: String, fiatAmount: Double) -> FiatOrder {
        let providers = getProviders()
        let provider = providers.first { $0.id == providerId } ?? providers[0]
        
        return FiatOrder(
            id: "fiat_order_\(Int(Date().timeIntervalSince1970 * 1000))",
            providerId: providerId,
            providerName: provider.name,
            side: side,
            fiatCurrency: fiatCurrency,
            cryptoCurrency: cryptoCurrency,
            fiatAmount: fiatAmount,
            cryptoAmount: calculateRate(providerId: providerId, cryptoCurrency: cryptoCurrency, fiatAmount: fiatAmount),
            fee: fiatAmount * provider.feePercent / 100,
            status: "PENDING"
        )
    }
}
