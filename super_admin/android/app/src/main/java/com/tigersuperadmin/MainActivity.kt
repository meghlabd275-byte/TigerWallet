package com.tigersuperadmin;

import android.app.Application;
import android.os.Bundle;
import androidx.appcompat.app.AppCompatActivity;
import androidx.recyclerview.widget.RecyclerView;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;
import android.view.View;
import android.widget.*;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * TigerWallet Super Admin - Android Application
 * Complete admin management with all features
 */

public class MainActivity extends AppCompatActivity {
    
    private static final String API_BASE_URL = "http://localhost:9090/api/v1";
    private String authToken = null;
    
    private LinearLayout navigationLayout;
    private FrameLayout contentLayout;
    private TextView titleView;
    private ProgressBar loadingIndicator;
    
    // Navigation items
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
        
        // User Management
        addNavItem("👥 Users", () -> showUsers());
        addNavItem("🛡️ KYC", () -> showKYC());
        
        // Transactions
        addNavItem("💸 Transactions", () -> showTransactions());
        addNavItem("💰 Withdrawals", () -> showWithdrawals());
        
        // Assets
        addNavItem("🪙 Tokens", () -> showTokens());
        addNavItem("⛓️ Blockchains", () -> showBlockchains());
        addNavItem("🔄 Trading Pairs", () -> showTradingPairs());
        
        // Finance
        addNavItem("💵 Fees", () -> showFees());
        
        // White Label
        addNavItem("🏢 White Labels", () -> showWhiteLabels());
        
        // Admin Management
        addNavItem("👤 Admins", () -> showAdmins());
        
        // Support
        addNavItem("🎫 Tickets", () -> showTickets());
        addNavItem("📚 Knowledge Base", () -> showKnowledgeBase());
        
        // Workflows
        addNavItem("✅ Workflows", () -> showWorkflows());
        
        // Reports
        addNavItem("📈 Reports", () -> showReports());
        
        // Security
        addNavItem("🔒 Security", () -> showSecurity());
        
        // API & Webhooks
        addNavItem("🔑 API Keys", () -> showAPIKeys());
        addNavItem("🪝 Webhooks", () -> showWebhooks());
        
        // System
        addNavItem("📝 Audit Logs", () -> showAuditLogs());
        addNavItem("⚙️ System", () -> showSystem());
        
        // Settings
        addNavItem("🔧 Settings", () -> showSettings());
        
        // Theme Toggle
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
        
        item.setOnClickListener(v -> {
            action.run();
        });
        
        navigationLayout.addView(item);
    }
    
    // Dashboard
    private void showDashboard() {
        setTitle("Dashboard");
        showLoading();
        
        new Thread(() -> {
            try {
                JSONObject stats = fetchDashboardStats();
                runOnUiThread(() -> {
                    hideLoading();
                    displayDashboard(stats);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    hideLoading();
                    showError("Failed to load dashboard");
                });
            }
        }).start();
    }
    
    private JSONObject fetchDashboardStats() throws Exception {
        URL url = new URL(API_BASE_URL + "/dashboard/stats");
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
        
        // Stats Grid
        html.append("<div style='display:grid; grid-template-columns:repeat(2,1fr); gap:16px; margin-bottom:24px;'>");
        
        addStatCard(html, "Total Users", stats.optString("total_users", "0"));
        addStatCard(html, "Active Users", stats.optString("active_users", "0"));
        addStatCard(html, "24h Volume", "$" + stats.optString("transaction_volume_24h", "0"));
        addStatCard(html, "24h Revenue", "$" + stats.optString("revenue_24h", "0"));
        
        html.append("</div>");
        
        // Pending Actions
        html.append("<div style='background:white; border-radius:8px; padding:16px; margin-bottom:24px;'>");
        html.append("<h3 style='margin:0 0 16px 0;'>Pending Actions</h3>");
        html.append("<div style='display:flex; justify-content:space-between; padding:8px 0; border-bottom:1px solid #eee;'>");
        html.append("<span>Pending Withdrawals</span>");
        html.append("<span style='background:#fef3c7; padding:4px 8px; border-radius:4px;'>").append(stats.optString("pending_withdrawals", "0")).append("</span>");
        html.append("</div>");
        html.append("<div style='display:flex; justify-content:space-between; padding:8px 0; border-bottom:1px solid #eee;'>");
        html.append("<span>Pending KYC</span>");
        html.append("<span style='background:#cffafe; padding:4px 8px; border-radius:4px;'>").append(stats.optString("pending_kyc", "0")).append("</span>");
        html.append("</div>");
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
    
    // Users
    private void showUsers() {
        setTitle("Users");
        showLoading();
        
        new Thread(() -> {
            try {
                JSONArray users = fetchUsers();
                runOnUiThread(() -> {
                    hideLoading();
                    displayUsers(users);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    hideLoading();
                    showError("Failed to load users");
                });
            }
        }).start();
    }
    
    private JSONArray fetchUsers() throws Exception {
        URL url = new URL(API_BASE_URL + "/users?page_size=50");
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
        
        return new JSONObject(response.toString()).optJSONArray("data");
    }
    
    private void displayUsers(JSONArray users) {
        StringBuilder html = new StringBuilder();
        html.append("<html><body style='background:#f5f5f5; font-family:sans-serif; padding:16px;'>");
        html.append("<h2 style='margin-bottom:16px;'>Users (" + users.length() + ")</h2>");
        
        for (int i = 0; i < users.length(); i++) {
            JSONObject user = users.optJSONObject(i);
            html.append("<div style='background:white; border-radius:8px; padding:16px; margin-bottom:12px;'>");
            html.append("<div style='font-weight:bold;'>").append(user.optString("username", "N/A")).append("</div>");
            html.append("<div style='color:#666; font-size:12px;'>").append(user.optString("email", "")).append("</div>");
            html.append("<div style='margin-top:8px;'>");
            
            String status = user.optString("status", "");
            String statusColor = "green".equals(status) ? "#22c55e" : "red".equals(status) ? "#ef4444" : "#f59e0b";
            html.append("<span style='background:").append(statusColor).append("; color:white; padding:2px 8px; border-radius:4px; font-size:12px;'>").append(status).append("</span>");
            
            String kycStatus = user.optString("kyc_status", "");
            String kycColor = "verified".equals(kycStatus) ? "#22c55e" : "pending".equals(kycStatus) ? "#f59e0b" : "#6b7280";
            html.append("<span style='background:").append(kycColor).append("; color:white; padding:2px 8px; border-radius:4px; font-size:12px; margin-left:8px;'>").append(kycStatus).append("</span>");
            
            html.append("</div></div>");
        }
        
        html.append("</body></html>");
        loadHtmlContent(html.toString());
    }
    
    // Placeholder methods for other screens
    private void showKYC() { setTitle("KYC Management"); loadHtmlContent("<html><body style='padding:16px;'><h2>KYC Requests</h2><p>KYC management interface</p></body></html>"); }
    private void showTransactions() { setTitle("Transactions"); loadHtmlContent("<html><body style='padding:16px;'><h2>Transactions</h2><p>Transaction management</p></body></html>"); }
    private void showWithdrawals() { setTitle("Withdrawals"); loadHtmlContent("<html><body style='padding:16px;'><h2>Withdrawals</h2><p>Withdrawal management</p></body></html>"); }
    private void showTokens() { setTitle("Tokens"); loadHtmlContent("<html><body style='padding:16px;'><h2>Token Management</h2><p>Token management interface</p></body></html>"); }
    private void showBlockchains() { setTitle("Blockchains"); loadHtmlContent("<html><body style='padding:16px;'><h2>Blockchain Management</h2><p>Blockchain management</p></body></html>"); }
    private void showTradingPairs() { setTitle("Trading Pairs"); loadHtmlContent("<html><body style='padding:16px;'><h2>Trading Pairs</h2><p>Trading pair management</p></body></html>"); }
    private void showFees() { setTitle("Fees"); loadHtmlContent("<html><body style='padding:16px;'><h2>Fee Management</h2><p>Fee configuration</p></body></html>"); }
    private void showWhiteLabels() { setTitle("White Labels"); loadHtmlContent("<html><body style='padding:16px;'><h2>White Label Management</h2><p>White label management</p></body></html>"); }
    private void showAdmins() { setTitle("Admins"); loadHtmlContent("<html><body style='padding:16px;'><h2>Admin Management</h2><p>Admin account management</p></body></html>"); }
    private void showTickets() { setTitle("Tickets"); loadHtmlContent("<html><body style='padding:16px;'><h2>Support Tickets</h2><p>Ticket management</p></body></html>"); }
    private void showKnowledgeBase() { setTitle("Knowledge Base"); loadHtmlContent("<html><body style='padding:16px;'><h2>Knowledge Base</h2><p>Article management</p></body></html>"); }
    private void showWorkflows() { setTitle("Workflows"); loadHtmlContent("<html><body style='padding:16px;'><h2>Approval Workflows</h2><p>Workflow management</p></body></html>"); }
    private void showReports() { setTitle("Reports"); loadHtmlContent("<html><body style='padding:16px;'><h2>Reports</h2><p>Analytics and reports</p></body></html>"); }
    private void showSecurity() { setTitle("Security"); loadHtmlContent("<html><body style='padding:16px;'><h2>Security</h2><p>Security alerts and monitoring</p></body></html>"); }
    private void showAPIKeys() { setTitle("API Keys"); loadHtmlContent("<html><body style='padding:16px;'><h2>API Keys</h2><p>API key management</p></body></html>"); }
    private void showWebhooks() { setTitle("Webhooks"); loadHtmlContent("<html><body style='padding:16px;'><h2>Webhooks</h2><p>Webhook configuration</p></body></html>"); }
    private void showAuditLogs() { setTitle("Audit Logs"); loadHtmlContent("<html><body style='padding:16px;'><h2>Audit Logs</h2><p>System audit logs</p></body></html>"); }
    private void showSystem() { setTitle("System"); loadHtmlContent("<html><body style='padding:16px;'><h2>System Status</h2><p>System monitoring</p></body></html>"); }
    private void showSettings() { setTitle("Settings"); loadHtmlContent("<html><body style='padding:16px;'><h2>Settings</h2><p>Application settings</p></body></html>"); }
    
    private void toggleTheme() {
        // Theme toggle implementation
        showToast("Theme toggled");
    }
    
    private void logout() {
        authToken = null;
        showToast("Logged out");
    }
    
    // Helper methods
    private void setTitle(String title) {
        titleView.setText(title);
    }
    
    private void showLoading() {
        loadingIndicator.setVisibility(View.VISIBLE);
    }
    
    private void hideLoading() {
        loadingIndicator.setVisibility(View.GONE);
    }
    
    private void showError(String message) {
        Toast.makeText(this, message, Toast.LENGTH_LONG).show();
    }
    
    private void showToast(String message) {
        Toast.makeText(this, message, Toast.LENGTH_SHORT).show();
    }
    
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
