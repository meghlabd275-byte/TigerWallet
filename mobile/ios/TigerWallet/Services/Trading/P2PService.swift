//
//  P2PService.swift
//  TigerWallet
//
//  P2P Trading Service - iOS Implementation
//

import Foundation

struct P2PAdvert {
    var id: String
    var userId: String
    var username: String
    var avatar: String
    var side: String
    var token: String
    var fiatCurrency: String
    var paymentMethod: String
    var price: Double
    var minAmount: Double
    var maxAmount: Double
    var availableAmount: Double
    var ordersCompleted: Int
    var completionRate: Double
    var avgReleaseTime: Double
    var isOnline: Bool
}

struct P2POrder {
    var id: String
    var advertId: String
    var side: String
    var token: String
    var fiatCurrency: String
    var paymentMethod: String
    var price: Double
    var amount: Double
    var fiatAmount: Double
    var status: String
}

class P2PService {
    
    func getAdverts(token: String?, fiatCurrency: String?, side: String?) -> [P2PAdvert] {
        let users = ["CryptoTrader1", "BitSeller", "FastTrade", "P2PPro"]
        let avatars = ["🧑‍💼", "👨‍💻", "⚡", "🎯"]
        let payments = ["Bank Transfer", "PayPal", "AliPay", "UPI"]
        let basePrices = [1.0, 43250.0, 2280.0, 312.5]
        
        var adverts: [P2PAdvert] = []
        for i in 0..<users.count {
            let advert = P2PAdvert(
                id: "advert_\(i)",
                userId: "user_\(i)",
                username: users[i],
                avatar: avatars[i],
                side: side ?? (i % 2 == 0 ? "BUY" : "SELL"),
                token: token ?? "USDT",
                fiatCurrency: fiatCurrency ?? "USD",
                paymentMethod: payments[i],
                price: basePrices[i % 4] * (1 + Double.random(in: -0.005...0.005)),
                minAmount: 10,
                maxAmount: 5000,
                availableAmount: basePrices[i % 4] * 10,
                ordersCompleted: 50 + i * 10,
                completionRate: 95 + Double(i % 5),
                avgReleaseTime: 2 + Double(i % 10),
                isOnline: i % 2 == 0
            )
            adverts.append(advert)
        }
        return adverts
    }
    
    func createOrder(advertId: String, takerId: String, amount: Double) -> P2POrder {
        return P2POrder(
            id: "order_\(Int(Date().timeIntervalSince1970 * 1000))",
            advertId: advertId,
            side: "BUY",
            token: "USDT",
            fiatCurrency: "USD",
            paymentMethod: "Bank Transfer",
            price: 1.0,
            amount: amount,
            fiatAmount: amount,
            status: "PENDING"
        )
    }
}
