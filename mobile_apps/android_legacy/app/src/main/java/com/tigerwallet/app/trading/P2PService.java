package com.tigerwallet.app.trading;

/**
 * P2P Trading Service - Android Implementation
 *
 * WARNING: There is currently NO real backend endpoint for P2P trading.
 * getAdverts and createOrder previously returned fabricated adverts/orders
 * (random prices, invented ids). They now throw UnsupportedOperationException
 * rather than producing invented data.
 */
public class P2PService {

    public static class P2PAdvert {
        public String id;
        public String userId;
        public String username;
        public String avatar;
        public String side;
        public String token;
        public String fiatCurrency;
        public String paymentMethod;
        public double price;
        public double minAmount;
        public double maxAmount;
        public double availableAmount;
        public int ordersCompleted;
        public double completionRate;
        public double avgReleaseTime;
        public boolean isOnline;
    }

    public static class P2POrder {
        public String id;
        public String advertId;
        public String side;
        public String token;
        public String fiatCurrency;
        public String paymentMethod;
        public double price;
        public double amount;
        public double fiatAmount;
        public String status;
    }

    /**
     * @throws UnsupportedOperationException no real P2P backend is configured
     */
    public java.util.List<P2PAdvert> getAdverts(String token, String fiatCurrency, String side) {
        throw new UnsupportedOperationException(
            "P2P trading backend is not configured; no adverts available");
    }

    /**
     * @throws UnsupportedOperationException no real P2P backend is configured
     */
    public P2POrder createOrder(String advertId, String takerId, double amount) {
        throw new UnsupportedOperationException(
            "P2P trading backend is not configured; cannot create order");
    }
}
