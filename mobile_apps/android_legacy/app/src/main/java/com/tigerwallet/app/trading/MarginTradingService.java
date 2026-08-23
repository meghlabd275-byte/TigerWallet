package com.tigerwallet.app.trading;

/**
 * Margin Trading Service - Android Implementation
 *
 * WARNING: There is currently NO real backend endpoint for margin trading.
 * The methods that would fabricate market data, account balances, or orders
 * therefore throw UnsupportedOperationException rather than returning invented
 * data. The pure-math helpers (calculateLiquidationPrice, calculatePnL) are
 * retained since they perform deterministic calculation only.
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

    /**
     * @throws UnsupportedOperationException no real margin-trading backend is configured
     */
    public MarginPair[] getPairs() {
        throw new UnsupportedOperationException(
            "margin trading backend is not configured; no market data available");
    }

    /**
     * @throws UnsupportedOperationException no real margin-trading backend is configured
     */
    public MarginAccount getAccount(String userId) {
        throw new UnsupportedOperationException(
            "margin trading backend is not configured; account data unavailable");
    }

    /**
     * @throws UnsupportedOperationException no real margin-trading backend is configured
     */
    public MarginOrder openPosition(String userId, String symbol, String side,
                                    double size, double price, int leverage, String marginMode) {
        throw new UnsupportedOperationException(
            "margin trading backend is not configured; cannot open position");
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
