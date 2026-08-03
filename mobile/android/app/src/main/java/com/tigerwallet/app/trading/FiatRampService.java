package com.tigerwallet.app.trading;

/**
 * Fiat On-Ramp Service - Android Implementation
 */

import java.util.ArrayList;
import java.util.List;

public class FiatRampService {
    
    public static class FiatProvider {
        public String id;
        public String name;
        public String logo;
        public String[] supportedFiat;
        public String[] supportedCrypto;
        public double minAmount;
        public double maxAmount;
        public double feePercent;
        public String processingTime;
        public boolean isAvailable;
    }
    
    public static class FiatOrder {
        public String id;
        public String providerId;
        public String providerName;
        public String side;
        public String fiatCurrency;
        public String cryptoCurrency;
        public double fiatAmount;
        public double cryptoAmount;
        public double fee;
        public String status;
    }
    
    public List<FiatProvider> getProviders() {
        List<FiatProvider> providers = new ArrayList<>();
        
        FiatProvider p1 = new FiatProvider();
        p1.id = "provider_1";
        p1.name = "MoonPay";
        p1.logo = "🌙";
        p1.supportedFiat = new String[]{"USD", "EUR", "GBP", "AUD"};
        p1.supportedCrypto = new String[]{"BTC", "ETH", "USDT", "BNB"};
        p1.minAmount = 30;
        p1.maxAmount = 50000;
        p1.feePercent = 2.5;
        p1.processingTime = "5-30 min";
        p1.isAvailable = true;
        providers.add(p1);
        
        FiatProvider p2 = new FiatProvider();
        p2.id = "provider_2";
        p2.name = "Simplex";
        p2.logo = "💳";
        p2.supportedFiat = new String[]{"USD", "EUR", "GBP"};
        p2.supportedCrypto = new String[]{"BTC", "ETH", "USDT"};
        p2.minAmount = 50;
        p2.maxAmount = 25000;
        p2.feePercent = 3.5;
        p2.processingTime = "10-60 min";
        p2.isAvailable = true;
        providers.add(p2);
        
        FiatProvider p3 = new FiatProvider();
        p3.id = "provider_3";
        p3.name = "Transak";
        p3.logo = "🔄";
        p3.supportedFiat = new String[]{"USD", "EUR", "GBP", "INR"};
        p3.supportedCrypto = new String[]{"BTC", "ETH", "USDT", "MATIC"};
        p3.minAmount = 20;
        p3.maxAmount = 100000;
        p3.feePercent = 2.0;
        p3.processingTime = "15-45 min";
        p3.isAvailable = true;
        providers.add(p3);
        
        return providers;
    }
    
    public double calculateRate(String providerId, String cryptoCurrency, double fiatAmount) {
        double baseRate = 43250.0;
        switch(cryptoCurrency) {
            case "BTC": baseRate = 43250; break;
            case "ETH": baseRate = 2280; break;
            case "USDT": 
            case "USDC": baseRate = 1; break;
            case "BNB": baseRate = 312.5; break;
            case "SOL": baseRate = 98.75; break;
        }
        
        double fee = 0.025;
        if ("provider_2".equals(providerId)) fee = 0.035;
        if ("provider_3".equals(providerId)) fee = 0.020;
        
        return (fiatAmount * (1 - fee)) / baseRate;
    }
    
    public FiatOrder createOrder(String providerId, String side, String fiatCurrency, 
                                String cryptoCurrency, double fiatAmount) {
        FiatOrder order = new FiatOrder();
        order.id = "fiat_order_" + System.currentTimeMillis();
        order.providerId = providerId;
        order.side = side;
        order.fiatCurrency = fiatCurrency;
        order.cryptoCurrency = cryptoCurrency;
        order.fiatAmount = fiatAmount;
        order.cryptoAmount = calculateRate(providerId, cryptoCurrency, fiatAmount);
        order.fee = fiatAmount * 0.025;
        order.status = "PENDING";
        return order;
    }
}
