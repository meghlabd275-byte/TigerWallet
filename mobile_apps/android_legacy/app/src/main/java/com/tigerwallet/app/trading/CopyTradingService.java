package com.tigerwallet.app.trading;

/**
 * Copy Trading Service - Android Implementation
 *
 * WARNING: This orphan service layer previously fabricated a hardcoded list of
 * fake traders (invented addresses 0x1234.../0xabcd..., invented win rates and
 * follower counts). Those methods now throw UnsupportedOperationException.
 * Real copy-trading data is served by the canonical Go copy_trading_service
 * (:8006, GET /api/v1/copytrading/traders) and consumed by the canonical
 * Android app (mobile_apps/android_app). The pure data classes below are
 * retained for callers that construct Trader/CopyPosition from real backend
 * responses.
 */

import java.util.List;

public class CopyTradingService {

    public static class Trader {
        public String id;
        public String address;
        public String username;
        public String avatar;
        public double winRate;
        public double totalPnl;
        public int followers;
        public int copyCount;
        public String tradingPair;
        public double monthlyPnl;
        public double weeklyPnl;
        public double dailyPnl;
        public String riskLevel;
        public boolean isFollowing;
        public boolean isVerified;
    }

    public static class CopyPosition {
        public String id;
        public String traderId;
        public String traderName;
        public String userId;
        public String symbol;
        public String side;
        public double size;
        public double entryPrice;
        public double currentPrice;
        public double pnl;
        public double pnlPercent;
        public long openTime;
        public String status;
    }

    public CopyTradingService() {}

    /**
     * @throws UnsupportedOperationException no real copy-trading backend is
     *                                       configured in this orphan layer;
     *                                       use the canonical Android app.
     */
    public List<Trader> getTopTraders(int limit) {
        throw new UnsupportedOperationException(
            "No real copy-trading backend configured. Use the canonical " +
            "copy_trading_service (/api/v1/copytrading/traders) via the " +
            "mobile_apps/android_app client.");
    }

    /**
     * @throws UnsupportedOperationException no real copy-trading backend is
     *                                       configured in this orphan layer.
     */
    public List<Trader> getAllTraders() {
        throw new UnsupportedOperationException(
            "No real copy-trading backend configured in this orphan layer.");
    }

    /**
     * @throws UnsupportedOperationException no real copy-trading backend is
     *                                       configured in this orphan layer.
     */
    public Trader getTrader(String traderId) {
        throw new UnsupportedOperationException(
            "No real copy-trading backend configured in this orphan layer.");
    }

    /**
     * @throws UnsupportedOperationException no real copy-trading backend is
     *                                       configured in this orphan layer.
     */
    public void followTrader(String traderId) {
        throw new UnsupportedOperationException(
            "No real copy-trading backend configured in this orphan layer.");
    }

    /**
     * @throws UnsupportedOperationException no real copy-trading backend is
     *                                       configured in this orphan layer.
     */
    public CopyPosition copyTrade(String userId, String traderId, String symbol, String side, double amount) {
        throw new UnsupportedOperationException(
            "No real copy-trading backend configured in this orphan layer.");
    }

    /**
     * @throws UnsupportedOperationException no real copy-trading backend is
     *                                       configured in this orphan layer.
     */
    public List<Trader> searchTraders(String query) {
        throw new UnsupportedOperationException(
            "No real copy-trading backend configured in this orphan layer.");
    }
}
