package com.tigerwallet.app.trading;

/**
 * P2P Trading Service - Android Implementation
 */

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

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
    
    private static final String[] users = {"CryptoTrader1", "BitSeller", "FastTrade", "P2PPro"};
    private static final String[] avatars = {"🧑‍💼", "👨‍💻", "⚡", "🎯"};
    private static final String[] payments = {"Bank Transfer", "PayPal", "AliPay", "UPI"};
    private static final double[] basePrices = {1.0, 43250.0, 2280.0, 312.5};
    
    public List<P2PAdvert> getAdverts(String token, String fiatCurrency, String side) {
        List<P2PAdvert> adverts = new ArrayList<>();
        for (int i = 0; i < users.length; i++) {
            P2PAdvert advert = new P2PAdvert();
            advert.id = "advert_" + i;
            advert.userId = "user_" + i;
            advert.username = users[i];
            advert.avatar = avatars[i];
            advert.side = side != null ? side : (i % 2 == 0 ? "BUY" : "SELL");
            advert.token = token != null ? token : "USDT";
            advert.fiatCurrency = fiatCurrency != null ? fiatCurrency : "USD";
            advert.paymentMethod = payments[i];
            advert.price = basePrices[i % 4] * (1 + (Math.random() * 0.01 - 0.005));
            advert.minAmount = 10;
            advert.maxAmount = 5000;
            advert.availableAmount = basePrices[i % 4] * 10;
            advert.ordersCompleted = 50 + i * 10;
            advert.completionRate = 95 + (i % 5);
            advert.avgReleaseTime = 2 + (i % 10);
            advert.isOnline = i % 2 == 0;
            adverts.add(advert);
        }
        return adverts;
    }
    
    public P2POrder createOrder(String advertId, String takerId, double amount) {
        P2POrder order = new P2POrder();
        order.id = "order_" + System.currentTimeMillis();
        order.advertId = advertId;
        order.side = "BUY";
        order.token = "USDT";
        order.fiatCurrency = "USD";
        order.price = 1.0;
        order.amount = amount;
        order.fiatAmount = amount;
        order.status = "PENDING";
        return order;
    }
}
