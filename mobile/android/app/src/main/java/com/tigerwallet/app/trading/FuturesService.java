package com.tigerwallet.app.trading;

/**
 * Futures Trading Service - Android Implementation
 *
 * WARNING: There is currently NO real backend endpoint for futures trading.
 * Market data (pairs/prices) and order placement were previously fabricated
 * (hardcoded price arrays + Math.random for change24h). Those methods now
 * throw UnsupportedOperationException. The pure-math helpers
 * (calculateLiquidationPrice, calculatePnL) and the in-memory order status
 * mutator are retained since they perform deterministic operations only.
 */
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

    public FuturesService() {
    }

    /**
     * @throws UnsupportedOperationException no real futures backend is configured
     */
    public java.util.List<FuturesPair> getPairs() {
        throw new UnsupportedOperationException(
            "futures trading backend is not configured; no market data available");
    }

    /**
     * @throws UnsupportedOperationException no real futures backend is configured
     */
    public FuturesOrder openPosition(String userId, String symbol, String side,
                                     double size, double price, int leverage, String marginMode) {
        throw new UnsupportedOperationException(
            "futures trading backend is not configured; cannot open position");
    }

    /**
     * @throws UnsupportedOperationException no real futures backend is configured
     */
    public FuturesOrder placeOrder(String userId, String symbol, String side, String type,
                                   double size, double price, int leverage, String marginMode) {
        throw new UnsupportedOperationException(
            "futures trading backend is not configured; cannot place order");
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
