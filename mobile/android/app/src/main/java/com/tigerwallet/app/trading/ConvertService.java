package com.tigerwallet.app.trading;

/**
 * Convert Service - Android Implementation
 * One-click conversion between crypto assets
 */

import java.util.HashMap;
import java.util.Map;

public class ConvertService {
    
    public static class ConvertToken {
        public String symbol;
        public String name;
        public double balance;
        public String icon;
    }
    
    public static class ConvertPair {
        public String from;
        public String to;
        public double rate;
        public double inverseRate;
        public double fee;
    }
    
    public static class ConvertOrder {
        public String id;
        public String fromToken;
        public String toToken;
        public double fromAmount;
        public double toAmount;
        public double rate;
        public double fee;
        public String status;
        public long timestamp;
    }
    
    private Map<String, Double> rates = new HashMap<>();
    
    public ConvertService() {
        initializeRates();
    }
    
    private void initializeRates() {
        rates.put("BTC_USDT", 43250.0);
        rates.put("ETH_USDT", 2280.0);
        rates.put("BNB_USDT", 312.5);
        rates.put("SOL_USDT", 98.75);
        rates.put("XRP_USDT", 0.62);
        rates.put("DOGE_USDT", 0.082);
        rates.put("ADA_USDT", 0.58);
        rates.put("BTC_ETH", 18.97);
    }
    
    public ConvertToken[] getTokens() {
        return new ConvertToken[] {
            createToken("BTC", "Bitcoin", 1.5, "₿"),
            createToken("ETH", "Ethereum", 15.0, "Ξ"),
            createToken("USDT", "Tether", 50000.0, "₮"),
            createToken("USDC", "USD Coin", 25000.0, "$"),
            createToken("BNB", "BNB", 50.0, "B"),
            createToken("SOL", "Solana", 150.0, "S"),
            createToken("XRP", "Ripple", 10000.0, "X"),
            createToken("ADA", "Cardano", 5000.0, "A"),
            createToken("DOGE", "Dogecoin", 100000.0, "D"),
            createToken("AVAX", "Avalanche", 200.0, "A")
        };
    }
    
    private ConvertToken createToken(String symbol, String name, double balance, String icon) {
        ConvertToken token = new ConvertToken();
        token.symbol = symbol;
        token.name = name;
        token.balance = balance;
        token.icon = icon;
        return token;
    }
    
    public double getRate(String from, String to) {
        if (from.equals(to)) return 1.0;
        
        String key = from + "_" + to;
        if (rates.containsKey(key)) {
            return rates.get(key);
        }
        
        // Try through USDT
        String fromToUsdt = from + "_USDT";
        String toFromUsdt = to + "_USDT";
        
        if (rates.containsKey(fromToUsdt) && rates.containsKey(toFromUsdt)) {
            return rates.get(fromToUsdt) / rates.get(toFromUsdt);
        }
        
        return 1.0;
    }
    
    public ConvertOrder convert(String userId, String from, String to, double amount) {
        ConvertOrder order = new ConvertOrder();
        order.id = "convert_" + System.currentTimeMillis();
        order.fromToken = from;
        order.toToken = to;
        order.fromAmount = amount;
        order.rate = getRate(from, to);
        order.toAmount = amount * order.rate;
        order.fee = amount * 0.001;
        order.status = "COMPLETED";
        order.timestamp = System.currentTimeMillis();
        return order;
    }
}
