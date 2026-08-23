package com.tigerwallet.app.services.security;

import android.content.Context;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Security Scanner Service for Android
 * Scans contracts and addresses for security risks
 */
public class SecurityScannerService {
    private static final String TAG = "SecurityScannerService";
    private static final String BASE_URL = "https://api.tigerwallet.com/v1/security";
    
    private final Context context;
    private final ExecutorService executor;
    
    public interface Callback<T> {
        void onSuccess(T result);
        void onError(Exception error);
    }
    
    public SecurityScannerService(Context context) {
        this.context = context;
        this.executor = Executors.newFixedThreadPool(4);
    }
    
    /**
     * Scan a contract address for vulnerabilities
     */
    public void scanContract(String address, String chain, Callback<ScanResult> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("address", address);
                body.put("chain", chain);
                
                JSONObject response = makePostRequest("/scan/contract", body);
                
                ScanResult result = new ScanResult();
                result.address = response.getString("address");
                result.chain = response.getString("chain");
                result.scanId = response.getString("scanId");
                result.score = response.getInt("score");
                result.riskLevel = response.getString("riskLevel");
                
                JSONArray issuesArray = response.getJSONArray("issues");
                result.issues = new ArrayList<>();
                for (int i = 0; i < issuesArray.length(); i++) {
                    JSONObject issue = issuesArray.getJSONObject(i);
                    SecurityIssue si = new SecurityIssue();
                    si.id = issue.getString("id");
                    si.title = issue.getString("title");
                    si.description = issue.getString("description");
                    si.severity = issue.getString("severity");
                    si.category = issue.getString("category");
                    si.status = issue.optString("status", "open");
                    result.issues.add(si);
                }
                
                result.whitelisted = response.getBoolean("whitelisted");
                result.lastScanned = response.getLong("lastScanned");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Scan a token for risks
     */
    public void scanToken(String address, String chain, Callback<TokenScanResult> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("address", address);
                body.put("chain", chain);
                
                JSONObject response = makePostRequest("/scan/token", body);
                
                TokenScanResult result = new TokenScanResult();
                result.address = response.getString("address");
                result.name = response.getString("name");
                result.symbol = response.getString("symbol");
                result.totalSupply = response.getString("totalSupply");
                result.holders = response.getInt("holders");
                result.transfers = response.getInt("transfers");
                result.isMintable = response.getBoolean("isMintable");
                result.isPausable = response.getBoolean("isPausable");
                result.isBlacklisted = response.getBoolean("isBlacklisted");
                result.trustScore = response.getInt("trustScore");
                result.verified = response.getBoolean("verified");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Check if an address is flagged
     */
    public void checkAddress(String address, Callback<AddressCheck> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/check/" + address);
                
                AddressCheck check = new AddressCheck();
                check.address = response.getString("address");
                check.isFlagged = response.getBoolean("isFlagged");
                check.isScam = response.getBoolean("isScam");
                check.isPhishing = response.getBoolean("isPhishing");
                check.reports = response.getInt("reports");
                check.firstSeen = response.getLong("firstSeen");
                check.lastActivity = response.getLong("lastActivity");
                check.labels = new ArrayList<>();
                
                JSONArray labelsArray = response.optJSONArray("labels");
                if (labelsArray != null) {
                    for (int i = 0; i < labelsArray.length(); i++) {
                        check.labels.add(labelsArray.getString(i));
                    }
                }
                
                callback.onSuccess(check);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get transaction simulation
     */
    public void simulateTransaction(String from, String to, String data, String value, String chain, Callback<SimulationResult> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("from", from);
                body.put("to", to);
                body.put("data", data);
                body.put("value", value);
                body.put("chain", chain);
                
                JSONObject response = makePostRequest("/simulate", body);
                
                SimulationResult result = new SimulationResult();
                result.success = response.getBoolean("success");
                result.gasUsed = response.getLong("gasUsed");
                result.stateChanges = new ArrayList<>();
                
                JSONArray changesArray = response.optJSONArray("stateChanges");
                if (changesArray != null) {
                    for (int i = 0; i < changesArray.length(); i++) {
                        result.stateChanges.add(changesArray.getString(i));
                    }
                }
                
                result.calls = new ArrayList<>();
                JSONArray callsArray = response.optJSONArray("calls");
                if (callsArray != null) {
                    for (int i = 0; i < callsArray.length(); i++) {
                        JSONObject c = callsArray.getJSONObject(i);
                        CallInfo call = new CallInfo();
                        call.to = c.getString("to");
                        call.value = c.getString("value");
                        call.success = c.getBoolean("success");
                        result.calls.add(call);
                    }
                }
                
                result.reverts = response.getBoolean("reverts");
                result.revertReason = response.optString("revertReason", "");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get security news/alerts
     */
    public void getSecurityAlerts(Callback<List<SecurityAlert>> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/alerts");
                JSONArray alertsArray = response.getJSONArray("alerts");
                
                List<SecurityAlert> alerts = new ArrayList<>();
                for (int i = 0; i < alertsArray.length(); i++) {
                    JSONObject a = alertsArray.getJSONObject(i);
                    SecurityAlert alert = new SecurityAlert();
                    alert.id = a.getString("id");
                    alert.title = a.getString("title");
                    alert.description = a.getString("description");
                    alert.severity = a.getString("severity");
                    alert.affectedAddresses = a.getJSONArray("affectedAddresses");
                    alert.publishedAt = a.getLong("publishedAt");
                    alerts.add(alert);
                }
                
                callback.onSuccess(alerts);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    private JSONObject makePostRequest(String endpoint, JSONObject body) throws Exception {
        URL url = new URL(BASE_URL + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        conn.setDoOutput(true);
        
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
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
    
    private JSONObject makeGetRequest(String endpoint) throws Exception {
        URL url = new URL(BASE_URL + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        
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
    
    // Data classes
    public static class ScanResult {
        public String address;
        public String chain;
        public String scanId;
        public int score;
        public String riskLevel;
        public List<SecurityIssue> issues;
        public boolean whitelisted;
        public long lastScanned;
    }
    
    public static class SecurityIssue {
        public String id;
        public String title;
        public String description;
        public String severity; // critical, high, medium, low, info
        public String category;
        public String status;
    }
    
    public static class TokenScanResult {
        public String address;
        public String name;
        public String symbol;
        public String totalSupply;
        public int holders;
        public int transfers;
        public boolean isMintable;
        public boolean isPausable;
        public boolean isBlacklisted;
        public int trustScore;
        public boolean verified;
    }
    
    public static class AddressCheck {
        public String address;
        public boolean isFlagged;
        public boolean isScam;
        public boolean isPhishing;
        public int reports;
        public long firstSeen;
        public long lastActivity;
        public List<String> labels;
    }
    
    public static class SimulationResult {
        public boolean success;
        public long gasUsed;
        public List<String> stateChanges;
        public List<CallInfo> calls;
        public boolean reverts;
        public String revertReason;
    }
    
    public static class CallInfo {
        public String to;
        public String value;
        public boolean success;
    }
    
    public static class SecurityAlert {
        public String id;
        public String title;
        public String description;
        public String severity;
        public JSONArray affectedAddresses;
        public long publishedAt;
    }
}
