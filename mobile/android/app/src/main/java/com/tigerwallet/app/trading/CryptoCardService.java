package com.tigerwallet.app.trading;

/**
 * Crypto Card Service - Android Implementation
 *
 * WARNING: There is currently NO real backend endpoint for crypto cards.
 * Card numbers, CVVs, expiry dates, and payment transaction records were
 * previously FABRICATED locally (java.util.Random). That code has been
 * removed: card creation and payment processing now throw
 * UnsupportedOperationException. The simple in-memory status mutators are
 * retained for state management of a card object, but no card can be issued
 * without a real backend.
 */
public class CryptoCardService {

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

    /**
     * @throws UnsupportedOperationException no real crypto-card backend is configured
     */
    public CryptoCard createVirtualCard(String userId, String cardHolder, String type, String network) {
        throw new UnsupportedOperationException(
            "crypto card backend is not configured; cannot issue virtual card");
    }

    /**
     * @throws UnsupportedOperationException no real crypto-card backend is configured
     */
    public CryptoCard createPhysicalCard(String userId, String cardHolder, String shippingAddress) {
        throw new UnsupportedOperationException(
            "crypto card backend is not configured; cannot issue physical card");
    }

    /**
     * @throws UnsupportedOperationException no real crypto-card backend is configured
     */
    public CardTransaction processPayment(CryptoCard card, double amount, String currency, String merchantName) {
        throw new UnsupportedOperationException(
            "crypto card backend is not configured; cannot process payment");
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
