//
//  CryptoCardService.swift
//  TigerWallet
//
//  Crypto Card Service - iOS Implementation
//

import Foundation

struct CryptoCard {
    var id: String
    var userId: String
    var cardNumber: String
    var cardHolder: String
    var expiryDate: String
    var cvv: String
    var type: String
    var network: String
    var status: String
    var dailyLimit: Double
    var monthlyLimit: Double
    var dailySpent: Double
    var monthlySpent: Double
    var applePayEnabled: Bool
    var googlePayEnabled: Bool
    var maskedNumber: String
}

struct CardTransaction {
    var id: String
    var cardId: String
    var userId: String
    var amount: Double
    var currency: String
    var merchantName: String
    var status: String
    var timestamp: Date
}

class CryptoCardService {
    
    func generateCardNumber() -> String {
        let random = String(format: "%012d", Int.random(in: 0...999999999999))
        return "4532" + random
    }
    
    func generateCVV() -> String {
        return String(format: "%03d", Int.random(in: 100...999))
    }
    
    func generateExpiry() -> String {
        let month = Int.random(in: 1...12)
        let year = (Calendar.current.component(.year, from: Date()) + 3) % 100
        return String(format: "%02d/%02d", month, year)
    }
    
    func createVirtualCard(userId: String, cardHolder: String, type: String? = "VIRTUAL", network: String? = "VISA") -> CryptoCard {
        let cardNumber = generateCardNumber()
        return CryptoCard(
            id: "card_\(Int(Date().timeIntervalSince1970 * 1000))",
            userId: userId,
            cardNumber: cardNumber,
            cardHolder: cardHolder,
            expiryDate: generateExpiry(),
            cvv: generateCVV(),
            type: type ?? "VIRTUAL",
            network: network ?? "VISA",
            status: "ACTIVE",
            dailyLimit: 10000,
            monthlyLimit: 100000,
            dailySpent: 0,
            monthlySpent: 0,
            applePayEnabled: true,
            googlePayEnabled: true,
            maskedNumber: "•••• •••• •••• " + String(cardNumber.suffix(4))
        )
    }
    
    func createPhysicalCard(userId: String, cardHolder: String, shippingAddress: String) -> CryptoCard {
        var card = createVirtualCard(userId: userId, cardHolder: cardHolder, type: "PHYSICAL")
        card.status = "PENDING_ACTIVATION"
        return card
    }
    
    func processPayment(card: inout CryptoCard, amount: Double, currency: String, merchantName: String) throws -> CardTransaction {
        guard card.status == "ACTIVE" else {
            throw NSError(domain: "CryptoCard", code: 1, userInfo: [NSLocalizedDescriptionKey: "Card is not active"])
        }
        
        guard card.dailySpent + amount <= card.dailyLimit else {
            throw NSError(domain: "CryptoCard", code: 2, userInfo: [NSLocalizedDescriptionKey: "Daily limit exceeded"])
        }
        
        card.dailySpent += amount
        card.monthlySpent += amount
        
        return CardTransaction(
            id: "txn_\(Int(Date().timeIntervalSince1970 * 1000))",
            cardId: card.id,
            userId: card.userId,
            amount: amount,
            currency: currency,
            merchantName: merchantName,
            status: "COMPLETED",
            timestamp: Date()
        )
    }
    
    func freezeCard(card: inout CryptoCard) {
        card.status = "FROZEN"
    }
    
    func unfreezeCard(card: inout CryptoCard) {
        card.status = "ACTIVE"
    }
    
    func terminateCard(card: inout CryptoCard) {
        card.status = "TERMINATED"
        card.applePayEnabled = false
        card.googlePayEnabled = false
    }
    
    func enableApplePay(card: inout CryptoCard) {
        card.applePayEnabled = true
    }
    
    func enableGooglePay(card: inout CryptoCard) {
        card.googlePayEnabled = true
    }
}
