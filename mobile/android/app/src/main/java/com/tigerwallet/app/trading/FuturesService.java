package com.tigerwallet.app.trading;

/**
 * Futures Trading Service - Android Implementation
 * Perpetual futures trading
 */

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class FuturesService {
    
    public static class FuturesPair {
        public String id;
        public String base;
        public String quote;
        public String symbol;
        public double price;
        public double change24h;
        public double volume24h;
        public double high24h;
        public double low24h;
        public double makerFee;
        public double takerFee;
    }
    
    public static class FuturesPosition {
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
    
    public static class FuturesOrder {
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
        public long createTime;
    }
    
    private static final String[] bases = {"BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK", "MATIC", "LTC", "UNI", "ATOM", "XLM", "NEAR", "APT", "ARB", "OP", "INJ"};
    private static final double[] prices = {43250.0, 2280.0, 312.5, 98.75, 0.62, 0.082, 0.58, 38.2, 7.85, 14.50, 0.92, 72.30, 6.25, 10.45, 0.125, 3.25, 9.80, 1.12, 2.45, 35.50};
    
    private Map<String, Double> priceMap = new HashMap<>();
    
    public FuturesService() {
        initializePrices();
    }
    
    private void initializePrices() {
        for (int i = 0; i < bases.length; i++) {
            priceMap.put(bases[i], prices[i]);
        }
    }
    
    public List<FuturesPair> getPairs() {
        List<FuturesPair> pairs = new ArrayList<>();
        String[] quotes = {"USDT", "USDC"};
        
        for (int i = 0; i < bases.length; i++) {
            for (String quote : quotes) {
                if (!bases[i].equals(quote)) {
                    FuturesPair pair = new FuturesPair();
                    pair.id = "futures_" + i;
                    pair.base = bases[i];
                    pair.quote = quote;
                    pair.symbol = bases[i] + "/" + quote;
                    pair.price = prices[i];
                    pair.change24h = (Math.random() * 10 - 5);
                    pair.volume24h = prices[i] * 1000000;
                    pair.high24h = prices[i] * 1.05;
                    pair.low24h = prices[i] * 0.95;
                    pair.makerFee = 0.02;
                    pair.takerFee = 0.04;
                    pairs.add(pair);
                }
            }
        }
        return pairs;
    }
    
    public FuturesOrder openPosition(String userId, String symbol, String side, 
                                   double size, double price, int leverage, String marginMode) {
        FuturesOrder order = new FuturesOrder();
        order.id = "futures_order_" + System.currentTimeMillis();
        order.userId = userId;
        order.symbol = symbol;
        order.side = side;
        order.type = "MARKET";
        order.size = size;
        order.price = price;
        order.status = "PENDING";
        order.leverage = leverage;
        order.marginMode = marginMode;
        order.createTime = System.currentTimeMillis();
        return order;
    }
    
    public FuturesOrder placeOrder(String userId, String symbol, String side, String type,
                                  double size, double price, int leverage, String marginMode) {
        FuturesOrder order = new FuturesOrder();
        order.id = "futures_order_" + System.currentTimeMillis();
        order.userId = userId;
        order.symbol = symbol;
        order.side = side;
        order.type = type;
        order.size = size;
        order.price = price;
        order.status = "PENDING";
        order.leverage = leverage;
        order.marginMode = marginMode;
        order.createTime = System.currentTimeMillis();
        return order;
    }
    
    public void cancelOrder(FuturesOrder order) {
        order.status = "CANCELLED";
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
