package com.tigerwallet.app.trading;

/**
 * Crypto Card Service - Android Implementation
 * Virtual and Physical Crypto Cards
 */

import java.util.UUID;
import java.util.Random;

public class CryptoCardService {
    private Random random = new Random();
    
    public static class CryptoCard {
        public String id;
        public String userId;
        public String cardNumber;
        public String cardHolder;
        public String expiryDate;
        public String cvv;
        public String type;
        public String network;
        public String status;
        public double dailyLimit;
        public double monthlyLimit;
        public double dailySpent;
        public double monthlySpent;
        public boolean applePayEnabled;
        public boolean googlePayEnabled;
        public String maskedNumber;
    }
    
    public static class CardTransaction {
        public String id;
        public String cardId;
        public String userId;
        public double amount;
        public String currency;
        public String merchantName;
        public String status;
        public long timestamp;
    }
    
    private String generateCardNumber() {
        return "4532" + String.format("%012d", Math.abs(random.nextLong()) % 1000000000000L);
    }
    
    private String generateCVV() {
        return String.format("%03d", random.nextInt(900) + 100);
    }
    
    private String generateExpiry() {
        int month = random.nextInt(12) + 1;
        int year = (2024 + 3) % 100;
        return String.format("%02d/%02d", month, year);
    }
    
    public CryptoCard createVirtualCard(String userId, String cardHolder, String type, String network) {
        CryptoCard card = new CryptoCard();
        card.id = "card_" + UUID.randomUUID().toString();
        card.userId = userId;
        card.cardNumber = generateCardNumber();
        card.cardHolder = cardHolder;
        card.expiryDate = generateExpiry();
        card.cvv = generateCVV();
        card.type = type != null ? type : "VIRTUAL";
        card.network = network != null ? network : "VISA";
        card.status = "ACTIVE";
        card.dailyLimit = 10000;
        card.monthlyLimit = 100000;
        card.dailySpent = 0;
        card.monthlySpent = 0;
        card.applePayEnabled = true;
        card.googlePayEnabled = true;
        card.maskedNumber = "•••• •••• •••• " + card.cardNumber.substring(card.cardNumber.length() - 4);
        return card;
    }
    
    public CryptoCard createPhysicalCard(String userId, String cardHolder, String shippingAddress) {
        CryptoCard card = createVirtualCard(userId, cardHolder, "PHYSICAL", "VISA");
        card.status = "PENDING_ACTIVATION";
        return card;
    }
    
    public CardTransaction processPayment(CryptoCard card, double amount, String currency, String merchantName) {
        if (!"ACTIVE".equals(card.status)) {
            throw new RuntimeException("Card is not active");
        }
        if (card.dailySpent + amount > card.dailyLimit) {
            throw new RuntimeException("Daily limit exceeded");
        }
        
        card.dailySpent += amount;
        card.monthlySpent += amount;
        
        CardTransaction txn = new CardTransaction();
        txn.id = "txn_" + System.currentTimeMillis();
        txn.cardId = card.id;
        txn.userId = card.userId;
        txn.amount = amount;
        txn.currency = currency;
        txn.merchantName = merchantName;
        txn.status = "COMPLETED";
        txn.timestamp = System.currentTimeMillis();
        
        return txn;
    }
    
    public void freezeCard(CryptoCard card) {
        card.status = "FROZEN";
    }
    
    public void unfreezeCard(CryptoCard card) {
        card.status = "ACTIVE";
    }
    
    public void terminateCard(CryptoCard card) {
        card.status = "TERMINATED";
        card.applePayEnabled = false;
        card.googlePayEnabled = false;
    }
    
    public void enableApplePay(CryptoCard card) {
        card.applePayEnabled = true;
    }
    
    public void enableGooglePay(CryptoCard card) {
        card.googlePayEnabled = true;
    }
}
