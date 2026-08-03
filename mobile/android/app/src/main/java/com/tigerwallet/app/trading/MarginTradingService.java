package com.tigerwallet.app.trading;

/**
 * Margin Trading Service - Android Implementation
 * Supports Cross/Isolated Margin, Long/Short, Leverage 1-125x
 */

public class MarginTradingService {
    
    public static class MarginPair {
        public String id;
        public String base;
        public String quote;
        public String symbol;
        public double price;
        public double change24h;
        public double volume24h;
        public double borrowable;
        public double interestRate;
        public boolean isActive;
        
        public MarginPair() {}
    }
    
    public static class MarginPosition {
        public String id;
        public String userId;
        public String symbol;
        public String side;
        public double size;
        public double entryPrice;
        public double markPrice;
        public int leverage;
        public double margin;
        public double pnl;
        public double pnlPercent;
        public double liquidationPrice;
        public String marginMode;
        public long openTime;
    }
    
    public static class MarginOrder {
        public String id;
        public String userId;
        public String symbol;
        public String side;
        public String type;
        public double size;
        public double price;
        public double filled;
        public String status;
        public int leverage;
        public String marginMode;
    }
    
    public static class MarginAccount {
        public String userId;
        public double totalAssets;
        public double totalLiabilities;
        public double netAssets;
        public double availableBalance;
        public double totalBorrowed;
        public double marginRatio;
        public String riskLevel;
    }
    
    private static MarginPair[] generatePairs() {
        String[] bases = {"BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK"};
        double[] prices = {43250.0, 2280.0, 312.5, 98.75, 0.62, 0.082, 0.58, 38.2, 7.85, 14.50};
        
        MarginPair[] pairs = new MarginPair[bases.length];
        for (int i = 0; i < bases.length; i++) {
            MarginPair pair = new MarginPair();
            pair.id = "margin_" + i;
            pair.base = bases[i];
            pair.quote = "USDT";
            pair.symbol = bases[i] + "/USDT";
            pair.price = prices[i];
            pair.change24h = (Math.random() * 10 - 5);
            pair.volume24h = prices[i] * 1000000;
            pair.borrowable = prices[i] * 50000000;
            pair.interestRate = 0.0001;
            pair.isActive = true;
            pairs[i] = pair;
        }
        return pairs;
    }
    
    public MarginPair[] getPairs() {
        return generatePairs();
    }
    
    public MarginAccount getAccount(String userId) {
        MarginAccount account = new MarginAccount();
        account.userId = userId;
        account.totalAssets = 50000.0;
        account.totalLiabilities = 5000.0;
        account.netAssets = 45000.0;
        account.availableBalance = 40000.0;
        account.totalBorrowed = 5000.0;
        account.marginRatio = 9.0;
        account.riskLevel = "SAFE";
        return account;
    }
    
    public MarginOrder openPosition(String userId, String symbol, String side, 
                                   double size, double price, int leverage, String marginMode) {
        MarginOrder order = new MarginOrder();
        order.id = "margin_order_" + System.currentTimeMillis();
        order.userId = userId;
        order.symbol = symbol;
        order.side = side;
        order.type = "MARKET";
        order.size = size;
        order.price = price;
        order.status = "PENDING";
        order.leverage = leverage;
        order.marginMode = marginMode;
        return order;
    }
    
    public double calculateLiquidationPrice(double entryPrice, int leverage, String side) {
        double liquidationPercent = 1.0 / leverage;
        if ("LONG".equals(side)) {
            return entryPrice * (1 - liquidationPercent);
        } else {
            return entryPrice * (1 + liquidationPercent);
        }
    }
    
    public double calculatePnL(double entryPrice, double closePrice, double size, String side) {
        if ("LONG".equals(side)) {
            return (closePrice - entryPrice) * size;
        } else {
            return (entryPrice - closePrice) * size;
        }
    }
}
