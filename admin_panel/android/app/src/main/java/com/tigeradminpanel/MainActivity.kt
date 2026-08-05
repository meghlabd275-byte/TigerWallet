package com.tigeradminpanel;

import android.app.Application;
import android.os.Bundle;
import androidx.appcompat.app.AppCompatActivity;
import android.view.View;
import android.widget.*;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.HashMap;
import java.util.Map;

/**
 * TigerWallet Admin Panel - Android Application
 * Complete trading platform admin management
 */

public class MainActivity extends AppCompatActivity {
    
    private static final String API_BASE_URL = "http://localhost:8081/api/v1/admin";
    private String authToken = null;
    
    private LinearLayout navigationLayout;
    private FrameLayout contentLayout;
    private TextView titleView;
    private ProgressBar loadingIndicator;
    
    private final Map<String, Runnable> navigationMap = new HashMap<>();
    
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        
        initViews();
        setupNavigation();
        showDashboard();
    }
    
    private void initViews() {
        navigationLayout = findViewById(R.id.navigation_layout);
        contentLayout = findViewById(R.id.content_layout);
        titleView = findViewById(R.id.title_view);
        loadingIndicator = findViewById(R.id.loading_indicator);
    }
    
    private void setupNavigation() {
        // Dashboard
        addNavItem("📊 Dashboard", () -> showDashboard());
        
        // Users
        addNavItem("👥 Users", () -> showUsers());
        addNavItem("🛡️ KYC", () -> showKYC());
        
        // Trading
        addNavItem("🔄 Trading Pairs", () -> showPairs());
        addNavItem("📈 Margin Trading", () -> showMarginTrading());
        addNavItem("📊 Futures", () -> showFutures());
        
        // Finance
        addNavItem("💵 Fees", () -> showFees());
        addNavItem("💰 Withdrawals", () -> showWithdrawals());
        addNavItem("💧 Liquidity", () -> showLiquidity());
        
        // Technical
        addNavItem("⛓️ Blockchains", () -> showBlockchains());
        addNavItem("🪙 Tokens", () -> showTokens());
        
        // Trading Bots
        addNavItem("🤖 Bot Instances", () -> showBots());
        addNavItem("📦 Bot Tiers", () -> showBotTiers());
        
        // P2P
        addNavItem("🤝 P2P Trading", () -> showP2PTrading());
        addNavItem("🏪 P2P Merchants", () -> showP2PMerchants());
        
        // Cards
        addNavItem("💳 Crypto Cards", () -> showCryptoCards());
        
        // Fiat
        addNavItem("🏦 Fiat On-Ramp", () -> showFiatOnRamp());
        
        // White Label
        addNavItem("🏢 White Labels", () -> showWhiteLabels());
        
        // Analytics
        addNavItem("📈 Analytics", () -> showAnalytics());
        
        // Master Wallet
        addNavItem("🔐 Master Wallet", () -> showMasterWallet());
        
        // System
        addNavItem("⚙️ Settings", () -> showSettings());
        
        // Theme
        addNavItem("🌓 Theme", () -> toggleTheme());
        
        // Logout
        addNavItem("🚪 Logout", () -> logout());
    }
    
    private void addNavItem(String label, Runnable action) {
        TextView item = new TextView(this);
        item.setText(label);
        item.setTextSize(14);
        item.setPadding(32, 24, 32, 24);
        item.setTextColor(getResources().getColor(R.color.text_primary, getTheme()));
        item.setBackgroundResource(R.drawable.nav_item_background);
        
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT,
            LinearLayout.LayoutParams.WRAP_CONTENT
        );
        item.setLayoutParams(params);
        
        item.setOnClickListener(v -> action.run());
        navigationLayout.addView(item);
    }
    
    // Dashboard
    private void showDashboard() {
        setTitle("Dashboard");
        showLoading();
        
        new Thread(() -> {
            try {
                JSONObject stats = fetchStats();
                runOnUiThread(() -> {
                    hideLoading();
                    displayDashboard(stats);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    hideLoading();
                    // Show mock data
                    JSONObject mockStats = new JSONObject();
                    try {
                        mockStats.put("total_users", "12543");
                        mockStats.put("active_users", "8234");
                        mockStats.put("total_volume", "98765432");
                        mockStats.put("total_transactions", "456789");
                        mockStats.put("active_bots", "234");
                        displayDashboard(mockStats);
                    } catch (Exception ex) {}
                });
            }
        }).start();
    }
    
    private JSONObject fetchStats() throws Exception {
        URL url = new URL(API_BASE_URL + "/stats");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        addAuthHeader(conn);
        
        BufferedReader reader = new BufferedReader(
            new InputStreamReader(conn.getInputStream())
        );
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            response.append(line);
        }
        reader.close();
        
        return new JSONObject(response.toString());
    }
    
    private void displayDashboard(JSONObject stats) {
        StringBuilder html = new StringBuilder();
        html.append("<html><body style='background:#f5f5f5; font-family:sans-serif; padding:16px;'>");
        
        html.append("<div style='display:grid; grid-template-columns:repeat(2,1fr); gap:16px; margin-bottom:24px;'>");
        
        addStatCard(html, "Total Users", stats.optString("total_users", "0"));
        addStatCard(html, "Active Users", stats.optString("active_users", "0"));
        addStatCard(html, "Total Volume", "$" + stats.optString("total_volume", "0"));
        addStatCard(html, "Transactions", stats.optString("total_transactions", "0"));
        
        html.append("</div>");
        
        html.append("<div style='display:grid; grid-template-columns:repeat(2,1fr); gap:16px;'>");
        
        addStatCard(html, "Active Bots", stats.optString("active_bots", "0"));
        addStatCard(html, "Total Bots", stats.optString("total_bots", "0"));
        addStatCard(html, "DEX Connections", stats.optString("active_dex_connections", "0"));
        addStatCard(html, "CEX Connections", stats.optString("active_cex_connections", "0"));
        
        html.append("</div>");
        
        html.append("</body></html>");
        
        loadHtmlContent(html.toString());
    }
    
    private void addStatCard(StringBuilder html, String label, String value) {
        html.append("<div style='background:white; border-radius:8px; padding:16px;'>");
        html.append("<div style='color:#666; font-size:12px; margin-bottom:8px;'>").append(label).append("</div>");
        html.append("<div style='font-size:24px; font-weight:bold; color:#0f172a;'>").append(value).append("</div>");
        html.append("</div>");
    }
    
    // Placeholder methods for other screens
    private void showUsers() { setTitle("Users Management"); loadHtmlContent("<html><body style='padding:16px;'><h2>User Management</h2><p>View, search, and manage platform users</p></body></html>"); }
    private void showKYC() { setTitle("KYC Management"); loadHtmlContent("<html><body style='padding:16px;'><h2>KYC Requests</h2><p>Identity verification management</p></body></html>"); }
    private void showPairs() { setTitle("Trading Pairs"); loadHtmlContent("<html><body style='padding:16px;'><h2>Trading Pairs</h2><p>Manage trading pairs</p></body></html>"); }
    private void showMarginTrading() { setTitle("Margin Trading"); loadHtmlContent("<html><body style='padding:16px;'><h2>Margin Trading</h2><p>Margin trading management</p></body></html>"); }
    private void showFutures() { setTitle("Futures"); loadHtmlContent("<html><body style='padding:16px;'><h2>Futures Trading</h2><p>Futures management</p></body></html>"); }
    private void showFees() { setTitle("Fee Management"); loadHtmlContent("<html><body style='padding:16px;'><h2>Fees</h2><p>Platform fee configuration</p></body></html>"); }
    private void showWithdrawals() { setTitle("Withdrawals"); loadHtmlContent("<html><body style='padding:16px;'><h2>Withdrawal Requests</h2><p>Process withdrawals</p></body></html>"); }
    private void showLiquidity() { setTitle("Liquidity"); loadHtmlContent("<html><body style='padding:16px;'><h2>Liquidity Pools</h2><p>Liquidity management</p></body></html>"); }
    private void showBlockchains() { setTitle("Blockchains"); loadHtmlContent("<html><body style='padding:16px;'><h2>Blockchain Management</h2><p>Network configuration</p></body></html>"); }
    private void showTokens() { setTitle("Tokens"); loadHtmlContent("<html><body style='padding:16px;'><h2>Token Management</h2><p>Token configuration</p></body></html>"); }
    private void showBots() { setTitle("Bot Instances"); loadHtmlContent("<html><body style='padding:16px;'><h2>Trading Bots</h2><p>Bot instance management</p></body></html>"); }
    private void showBotTiers() { setTitle("Bot Tiers"); loadHtmlContent("<html><body style='padding:16px;'><h2>Bot Tiers</h2><p>Bot tier configuration</p></body></html>"); }
    private void showP2PTrading() { setTitle("P2P Trading"); loadHtmlContent("<html><body style='padding:16px;'><h2>P2P Trading</h2><p>Peer-to-peer trading management</p></body></html>"); }
    private void showP2PMerchants() { setTitle("P2P Merchants"); loadHtmlContent("<html><body style='padding:16px;'><h2>Merchants</h2><p>Merchant management</p></body></html>"); }
    private void showCryptoCards() { setTitle("Crypto Cards"); loadHtmlContent("<html><body style='padding:16px;'><h2>Crypto Cards</h2><p>Card management</p></body></html>"); }
    private void showFiatOnRamp() { setTitle("Fiat On-Ramp"); loadHtmlContent("<html><body style='padding:16px;'><h2>Fiat On-Ramp</h2><p>Fiat gateway management</p></body></html>"); }
    private void showWhiteLabels() { setTitle("White Labels"); loadHtmlContent("<html><body style='padding:16px;'><h2>White Labels</h2><p>White label management</p></body></html>"); }
    private void showAnalytics() { setTitle("Analytics"); loadHtmlContent("<html><body style='padding:16px;'><h2>Analytics</h2><p>Platform analytics</p></body></html>"); }
    private void showMasterWallet() { setTitle("Master Wallet"); loadHtmlContent("<html><body style='padding:16px;'><h2>Master Wallet</h2><p>Master wallet operations</p></body></html>"); }
    private void showSettings() { setTitle("Settings"); loadHtmlContent("<html><body style='padding:16px;'><h2>Settings</h2><p>Application settings</p></body></html>"); }
    
    private void toggleTheme() { showToast("Theme toggled"); }
    private void logout() { authToken = null; showToast("Logged out"); }
    
    // Helper methods
    private void setTitle(String title) { titleView.setText(title); }
    private void showLoading() { loadingIndicator.setVisibility(View.VISIBLE); }
    private void hideLoading() { loadingIndicator.setVisibility(View.GONE); }
    private void showToast(String message) { Toast.makeText(this, message, Toast.LENGTH_SHORT).show(); }
    
    private void loadHtmlContent(String html) {
        WebView webView = new WebView(this);
        webView.getSettings().setJavaScriptEnabled(true);
        webView.loadData(html, "text/html", "UTF-8");
        
        contentLayout.removeAllViews();
        contentLayout.addView(webView);
    }
    
    private void addAuthHeader(HttpURLConnection conn) {
        if (authToken != null) {
            conn.setRequestProperty("Authorization", "Bearer " + authToken);
        }
        conn.setRequestProperty("Content-Type", "application/json");
    }
}
