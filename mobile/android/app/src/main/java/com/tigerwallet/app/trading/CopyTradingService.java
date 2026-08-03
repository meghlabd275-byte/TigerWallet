package com.tigerwallet.app.trading;

/**
 * Copy Trading Service - Android Implementation
 * Follow expert traders
 */

import java.util.ArrayList;
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
    
    private List<Trader> traders = new ArrayList<>();
    
    public CopyTradingService() {
        initializeTraders();
    }
    
    private void initializeTraders() {
        traders.add(createTrader("TraderAlex", "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E", 78.5, 125000, 5420, "BTC/USDT", "MEDIUM", true));
        traders.add(createTrader("CryptoKing", "0x1234567890abcdef1234567890abcdef12345678", 72.3, 98500, 3210, "ETH/USDT", "HIGH", true));
        traders.add(createTrader("DeFiMaster", "0xabcdef1234567890abcdef1234567890abcdef12", 85.0, 150000, 8930, "SOL/USDT", "LOW", true));
        traders.add(createTrader("AltSeason", "0x9876543210fedcba9876543210fedcba98765432", 65.0, 87000, 1890, "XRP/USDT", "HIGH", false));
        traders.add(createTrader("BitcoinWhale", "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", 82.0, 200000, 6540, "BTC/USDT", "MEDIUM", true));
    }
    
    private Trader createTrader(String username, String address, double winRate, double pnl, int followers, String pair, String risk, boolean verified) {
        Trader trader = new Trader();
        trader.id = "trader_" + traders.size();
        trader.address = address;
        trader.username = username;
        trader.avatar = "🐋";
        trader.winRate = winRate;
        trader.totalPnl = pnl;
        trader.followers = followers;
        trader.copyCount = followers / 2;
        trader.tradingPair = pair;
        trader.monthlyPnl = pnl * 0.25;
        trader.weeklyPnl = pnl * 0.06;
        trader.dailyPnl = pnl * 0.01;
        trader.riskLevel = risk;
        trader.isFollowing = false;
        trader.isVerified = verified;
        return trader;
    }
    
    public List<Trader> getTopTraders(int limit) {
        return traders.subList(0, Math.min(limit, traders.size()));
    }
    
    public List<Trader> getAllTraders() {
        return traders;
    }
    
    public Trader getTrader(String traderId) {
        for (Trader trader : traders) {
            if (trader.id.equals(traderId)) {
                return trader;
            }
        }
        return null;
    }
    
    public void followTrader(String traderId) {
        Trader trader = getTrader(traderId);
        if (trader != null) {
            trader.isFollowing = !trader.isFollowing;
            if (trader.isFollowing) {
                trader.followers++;
            } else {
                trader.followers--;
            }
        }
    }
    
    public CopyPosition copyTrade(String userId, String traderId, String symbol, String side, double amount) {
        Trader trader = getTrader(traderId);
        
        CopyPosition position = new CopyPosition();
        position.id = "copy_" + System.currentTimeMillis();
        position.traderId = traderId;
        position.traderName = trader != null ? trader.username : "Unknown";
        position.userId = userId;
        position.symbol = symbol;
        position.side = side;
        position.size = amount;
        position.entryPrice = 43250.0;
        position.currentPrice = position.entryPrice;
        position.pnl = 0;
        position.pnlPercent = 0;
        position.openTime = System.currentTimeMillis();
        position.status = "OPEN";
        
        return position;
    }
    
    public List<Trader> searchTraders(String query) {
        List<Trader> results = new ArrayList<>();
        for (Trader trader : traders) {
            if (trader.username.toLowerCase().contains(query.toLowerCase()) ||
                trader.address.toLowerCase().contains(query.toLowerCase())) {
                results.add(trader);
            }
        }
        return results;
    }
}
