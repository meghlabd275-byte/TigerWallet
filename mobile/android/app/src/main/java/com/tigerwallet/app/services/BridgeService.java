package com.tigerwallet.app.services;

import org.json.JSONArray;
import org.json.JSONObject;
import java.net.HttpURLConnection;
import java.net.URL;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;

/**
 * Bridge Service - Android Native Implementation
 */
public class BridgeService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public BridgeService(String token) { this.authToken = token; }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn;
    }
    
    public Chain[] getSupportedChains() throws Exception {
        HttpURLConnection conn = createConnection("/bridge/chains");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            Chain[] chains = new Chain[data.length()];
            for (int i = 0; i < data.length(); i++) {
                JSONObject c = data.getJSONObject(i);
                chains[i] = new Chain(c.getString("id"), c.getString("name"), c.getString("symbol"), c.getBoolean("isActive"));
            }
            return chains;
        }
        return new Chain[0];
    }
    
    public BridgeToken[] getTokens(String chain) throws Exception {
        HttpURLConnection conn = createConnection("/bridge/tokens?chain=" + chain);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            BridgeToken[] tokens = new BridgeToken[data.length()];
            for (int i = 0; i < data.length(); i++) {
                JSONObject t = data.getJSONObject(i);
                tokens[i] = new BridgeToken(t.getString("token"), t.getString("name"), t.getDouble("minAmount"), t.getDouble("maxAmount"), t.getBoolean("isActive"));
            }
            return tokens;
        }
        return new BridgeToken[0];
    }
    
    public BridgeEstimate getEstimate(String fromChain, String toChain, String token, double amount) throws Exception {
        HttpURLConnection conn = createConnection("/bridge/estimate?fromChain=" + fromChain + "&toChain=" + toChain + "&token=" + token + "&amount=" + amount);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONObject("data");
            return new BridgeEstimate(data.getDouble("receivedAmount"), data.getDouble("fee"), data.getDouble("feePercentage"), data.getString("estimatedTime"));
        }
        return null;
    }
    
    public BridgeTransaction initiateBridge(String fromChain, String toChain, String token, double amount) throws Exception {
        URL url = new URL(API_BASE + "/bridge/transactions");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        
        JSONObject body = new JSONObject();
        body.put("fromChain", fromChain);
        body.put("toChain", toChain);
        body.put("token", token);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONObject("data");
            return new BridgeTransaction(data.getString("id"), data.getString("fromChain"), data.getString("toChain"), 
                data.getString("token"), data.getDouble("amount"), data.getDouble("fee"), data.getDouble("receivedAmount"), data.getString("status"));
        }
        return null;
    }
    
    private String readResponse(HttpURLConnection conn) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) response.append(line);
        reader.close();
        return response.toString();
    }
    
    public static class Chain {
        public String id, name, symbol;
        public boolean isActive;
        public Chain(String id, String name, String symbol, boolean isActive) {
            this.id = id; this.name = name; this.symbol = symbol; this.isActive = isActive;
        }
    }
    
    public static class BridgeToken {
        public String token, name;
        public double minAmount, maxAmount;
        public boolean isActive;
        public BridgeToken(String token, String name, double minAmount, double maxAmount, boolean isActive) {
            this.token = token; this.name = name; this.minAmount = minAmount; this.maxAmount = maxAmount; this.isActive = isActive;
        }
    }
    
    public static class BridgeEstimate {
        public double receivedAmount, fee, feePercentage;
        public String estimatedTime;
        public BridgeEstimate(double receivedAmount, double fee, double feePercentage, String estimatedTime) {
            this.receivedAmount = receivedAmount; this.fee = fee; this.feePercentage = feePercentage; this.estimatedTime = estimatedTime;
        }
    }
    
    public static class BridgeTransaction {
        public String id, fromChain, toChain, token, status;
        public double amount, fee, receivedAmount;
        public BridgeTransaction(String id, String fromChain, String toChain, String token, double amount, double fee, double receivedAmount, String status) {
            this.id = id; this.fromChain = fromChain; this.toChain = toChain; this.token = token;
            this.amount = amount; this.fee = fee; this.receivedAmount = receivedAmount; this.status = status;
        }
    }
}
