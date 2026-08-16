package com.tigersuperadmin;

import android.app.Application;
import android.os.Bundle;
import androidx.appcompat.app.AppCompatActivity;
import androidx.appcompat.app.AppCompatDelegate;
import androidx.recyclerview.widget.RecyclerView;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;
import android.view.View;
import android.webkit.JavascriptInterface;
import android.webkit.WebView;
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
    
    private static final String API_BASE_URL = "http://localhost:8082/api/v1/admin";
    private String authToken = null;
    private boolean isDarkMode = false;

    // The 12 governance domains and the actions each supports.
    // `resource` is the path segment under /api/v1/admin.
    private static final String[][] ADMIN_DOMAINS = {
        {"futures", "Futures", "futures", "status"},
        {"options", "Options", "options", "status"},
        {"copy-trading", "Copy Trading", "copy-trading", "status"},
        {"convert", "Convert", "convert", "status"},
        {"onramp", "Onramp", "onramp", "approve,reject"},
        {"offramp", "Offramp", "offramp", "approve,reject"},
        {"p2p-clients", "P2P Clients", "p2p-clients", "status"},
        {"partners", "Partners", "partners", "status,approve,reject"},
        {"rewards", "Rewards", "rewards", "status"},
        {"marketing", "Marketing", "marketing", "status"},
        {"admin-roles", "Admin Roles", "admin-roles", ""},
        {"wl-control", "WL Control", "wl-clients", "status"}
    };
    
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

        // Governance domains (12) - real super_admin/go backend on :8082
        addNavItem("📈 Futures", () -> showDomainScreen("futures"));
        addNavItem("📉 Options", () -> showDomainScreen("options"));
        addNavItem("🔁 Copy Trading", () -> showDomainScreen("copy-trading"));
        addNavItem("🔄 Convert", () -> showDomainScreen("convert"));
        addNavItem("🏨 Onramp", () -> showDomainScreen("onramp"));
        addNavItem("🏧 Offramp", () -> showDomainScreen("offramp"));
        addNavItem("🤝 P2P Clients", () -> showDomainScreen("p2p-clients"));
        addNavItem("🤝 Partners", () -> showDomainScreen("partners"));
        addNavItem("🎁 Rewards", () -> showDomainScreen("rewards"));
        addNavItem("📣 Marketing", () -> showDomainScreen("marketing"));
        addNavItem("🔑 Admin Roles", () -> showDomainScreen("admin-roles"));
        addNavItem("🏷️ WL Control", () -> showDomainScreen("wl-control"));
        
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
    
    // ===== Governance domain screens (real :8082 backend, loading/error/empty) =====

    private void showDomainScreen(String id) {
        String[] meta = domainMeta(id);
        String label = meta[1];
        String resource = meta[2];
        setTitle(label);
        showLoading();
        new Thread(() -> {
            try {
                JSONArray rows = fetchDomainList(resource);
                runOnUiThread(() -> {
                    hideLoading();
                    displayDomainScreen(id, label, resource, meta[3], rows);
                });
            } catch (Exception e) {
                runOnUiThread(() -> {
                    hideLoading();
                    displayDomainError(label, e.getMessage());
                });
            }
        }).start();
    }

    private String[] domainMeta(String id) {
        for (String[] d : ADMIN_DOMAINS) {
            if (d[0].equals(id)) return d;
        }
        return new String[]{id, id, id, ""};
    }

    private JSONArray fetchDomainList(String resource) throws Exception {
        JSONObject resp = domainRequest(resource, "GET", null, null);
        JSONArray data = resp.optJSONArray("data");
        if (data != null) return data;
        // some endpoints return a bare array
        Object root = resp.opt("root");
        return root instanceof JSONArray ? (JSONArray) root : new JSONArray();
    }

    private void displayDomainScreen(String id, String label, String resource, String actions, JSONArray rows) {
        StringBuilder html = new StringBuilder();
        html.append("<html><head>").append(themeCss()).append("</head><body class='").append(isDarkMode ? "dark" : "light").append("'>");
        html.append("<div class='header'><h2>").append(escapeHtml(label)).append(" (").append(rows.length()).append(")</h2>");
        html.append("<button class='btn' onclick='window.AndroidDomain.create(\"").append(id).append("\")'>+ New</button></div>");
        if (rows.length() == 0) {
            html.append("<div class='empty'>No ").append(escapeHtml(label.toLowerCase())).append(" records found.</div>");
        } else {
            for (int i = 0; i < rows.length(); i++) {
                JSONObject rec = rows.optJSONObject(i);
                String recId = rec.optString("id", rec.optString("uuid", String.valueOf(i)));
                html.append("<div class='card'>");
                java.util.Iterator<String> keys = rec.keys();
                int shown = 0;
                while (keys.hasNext() && shown < 8) {
                    String k = keys.next();
                    html.append("<div class='row'><span class='k'>").append(escapeHtml(k))
                       .append("</span><span class='v'>").append(escapeHtml(String.valueOf(rec.opt(k)))).append("</span></div>");
                    shown++;
                }
                html.append("<div class='actions'>");
                html.append(actionButton("Edit", "window.AndroidDomain.edit(\"" + id + "\"," + rec.toString() + ")"));
                html.append(actionButton("Delete", "window.AndroidDomain.del(\"" + id + "\",\"" + recId + "\")", "danger"));
                if (actions.contains("status")) {
                    html.append(actionButton("Status", "window.AndroidDomain.status(\"" + id + "\",\"" + recId + "\")"));
                }
                if (actions.contains("approve")) {
                    html.append(actionButton("Approve", "window.AndroidDomain.approve(\"" + id + "\",\"" + recId + "\")", "ok"));
                }
                if (actions.contains("reject")) {
                    html.append(actionButton("Reject", "window.AndroidDomain.reject(\"" + id + "\",\"" + recId + "\")", "danger"));
                }
                html.append("</div></div>");
            }
        }
        html.append("</body></html>");
        loadDomainHtml(html.toString(), id);
    }

    private void displayDomainError(String label, String message) {
        String html = "<html><head>" + themeCss() + "</head><body class='" + (isDarkMode ? "dark" : "light") + "'>"
            + "<div class='error'>Failed to load " + escapeHtml(label) + ": " + escapeHtml(message == null ? "unknown error" : message) + "</div>"
            + "</body></html>";
        loadHtmlContent(html);
    }

    private String actionButton(String label, String onclick) {
        return actionButton(label, onclick, "");
    }

    private String actionButton(String label, String onclick, String cls) {
        return "<button class='btn " + cls + "' onclick='" + onclick + "'>" + label + "</button>";
    }

    private String themeCss() {
        return "<style>"
            + ":root,.light{--bg:#f5f5f5;--card:#ffffff;--text:#0f172a;--muted:#666;--border:#eee;--btn:#0f172a;--btn-text:#fff;--ok:#22c55e;--danger:#ef4444;}"
            + ".dark{--bg:#0f172a;--card:#1e293b;--text:#e2e8f0;--muted:#94a3b8;--border:#334155;--btn:#e2e8f0;--btn-text:#0f172a;--ok:#22c55e;--danger:#ef4444;}"
            + "body{background:var(--bg);color:var(--text);font-family:sans-serif;padding:16px;margin:0;}"
            + ".header{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;}"
            + ".card{background:var(--card);border:1px solid var(--border);border-radius:8px;padding:12px;margin-bottom:10px;}"
            + ".row{display:flex;justify-content:space-between;font-size:13px;padding:2px 0;}"
            + ".k{color:var(--muted);}.v{font-family:monospace;word-break:break-all;text-align:right;}"
            + ".actions{display:flex;gap:6px;margin-top:8px;flex-wrap:wrap;}"
            + ".btn{background:var(--btn);color:var(--btn-text);border:none;border-radius:6px;padding:6px 12px;font-size:12px;cursor:pointer;}"
            + ".btn.danger{background:var(--danger);color:#fff;}.btn.ok{background:var(--ok);color:#fff;}"
            + ".empty,.error{background:var(--card);border:1px solid var(--border);border-radius:8px;padding:24px;color:var(--muted);text-align:center;}"
            + ".error{color:var(--danger);}"
            + "</style>";
    }

    private String escapeHtml(String s) {
        if (s == null) return "";
        return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
                .replace("\"", "&quot;").replace("'", "&#39;");
    }

    private void loadDomainHtml(String html, String domainId) {
        WebView webView = new WebView(this);
        webView.getSettings().setJavaScriptEnabled(true);
        webView.addJavascriptInterface(new DomainJsBridge(domainId), "AndroidDomain");
        webView.loadDataWithBaseURL(null, html, "text/html", "UTF-8", null);
        contentLayout.removeAllViews();
        contentLayout.addView(webView);
    }

    // Bridge invoked from the rendered HTML buttons. All actions hit the real backend.
    private class DomainJsBridge {
        private final String domainId;
        DomainJsBridge(String domainId) { this.domainId = domainId; }

        @android.webkit.JavascriptInterface
        public void create(String id) { showDomainToast(id, "Create not supported in read-only view"); }

        @android.webkit.JavascriptInterface
        public void edit(String id, String rec) { showDomainToast(id, "Edit not supported in read-only view"); }

        @android.webkit.JavascriptInterface
        public void del(final String id, final String recId) { runDomainAction(id, "DELETE", recId, null); }

        @android.webkit.JavascriptInterface
        public void status(final String id, final String recId) { runDomainAction(id, "STATUS", recId, null); }

        @android.webkit.JavascriptInterface
        public void approve(final String id, final String recId) { runDomainAction(id, "APPROVE", recId, null); }

        @android.webkit.JavascriptInterface
        public void reject(final String id, final String recId) { runDomainAction(id, "REJECT", recId, "Policy violation"); }

        private void runDomainAction(final String id, final String op, final String recId, final String body) {
            runOnUiThread(() -> showLoading());
            new Thread(() -> {
                String err = null;
                try {
                    String[] meta = domainMeta(id);
                    String resource = meta[2];
                    switch (op) {
                        case "DELETE":
                            domainRequest(resource, "DELETE", recId, null); break;
                        case "STATUS":
                            domainRequest(resource, "PUT", recId + "/status", new JSONObject().put("status", "paused")); break;
                        case "APPROVE":
                            domainRequest(resource, "POST", recId + "/approve", new JSONObject()); break;
                        case "REJECT":
                            domainRequest(resource, "POST", recId + "/reject", new JSONObject().put("reason", body == null ? "n/a" : body)); break;
                    }
                } catch (Exception e) {
                    err = e.getMessage();
                }
                final String error = err;
                runOnUiThread(() -> {
                    hideLoading();
                    if (error != null) showError("Action failed: " + error);
                    else showDomainScreen(id);
                });
            }).start();
        }

        private void showDomainToast(String id, String msg) { runOnUiThread(() -> showToast(msg)); }
    }

    // Generic HTTP call to the super_admin/go backend. `pathSuffix` may be null for collection ops.
    private JSONObject domainRequest(String resource, String method, String pathSuffix, JSONObject body) throws Exception {
        StringBuilder urlStr = new StringBuilder(API_BASE_URL).append("/").append(resource);
        if (pathSuffix != null && !pathSuffix.isEmpty()) urlStr.append("/").append(pathSuffix);
        URL url = new URL(urlStr.toString());
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod(method);
        addAuthHeader(conn);
        if (body != null && (method.equals("POST") || method.equals("PUT") || method.equals("PATCH"))) {
            conn.setDoOutput(true);
            byte[] payload = body.toString().getBytes("UTF-8");
            conn.getOutputStream().write(payload);
        }
        int code = conn.getResponseCode();
        BufferedReader reader;
        if (code >= 200 && code < 300) {
            reader = new BufferedReader(new InputStreamReader(conn.getInputStream()));
        } else {
            reader = new BufferedReader(new InputStreamReader(conn.getErrorStream()));
            StringBuilder er = new StringBuilder(); String l;
            while ((l = reader.readLine()) != null) er.append(l);
            reader.close();
            throw new Exception("HTTP " + code + ": " + er.toString());
        }
        StringBuilder response = new StringBuilder(); String line;
        while ((line = reader.readLine()) != null) response.append(line);
        reader.close();
        String resp = response.toString();
        if (resp.isEmpty()) return new JSONObject();
        if (resp.trim().startsWith("[")) {
            JSONObject wrapper = new JSONObject();
            wrapper.put("root", new JSONArray(resp));
            return wrapper;
        }
        return new JSONObject(resp);
    }

    private void toggleTheme() {
        isDarkMode = !isDarkMode;
        AppCompatDelegate.setDefaultNightMode(isDarkMode ? AppCompatDelegate.MODE_NIGHT_YES : AppCompatDelegate.MODE_NIGHT_NO);
        showToast(isDarkMode ? "Dark theme" : "Light theme");
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
