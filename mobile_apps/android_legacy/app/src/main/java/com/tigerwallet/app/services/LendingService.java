package com.tigerwallet.app.services;

import android.util.Base64;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

/**
 * Lending Service - Android Native Implementation
 * Real backend connection to Go lending service
 */
public class LendingService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public LendingService(String token) {
        this.authToken = token;
    }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) {
            conn.setRequestProperty("Authorization", "Bearer " + authToken);
        }
        return conn;
    }
    
    /**
     * Get all lending pools
     */
    public LendingPool[] getPools() throws Exception {
        HttpURLConnection conn = createConnection("/lending/pools");
        int responseCode = conn.getResponseCode();
        
        if (responseCode == 200) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONArray("data");
            LendingPool[] pools = new LendingPool[data.length()];
            for (int i = 0; i < data.length(); i++) {
                JSONObject p = data.getJSONObject(i);
                pools[i] = new LendingPool(
                    p.getString("token"),
                    p.getString("name"),
                    p.getDouble("totalSupplied"),
                    p.getDouble("totalBorrowed"),
                    p.getDouble("supplyAPY"),
                    p.getDouble("borrowAPY"),
                    p.getDouble("liquidity")
                );
            }
            return pools;
        }
        throw new Exception("Failed to get pools: " + responseCode);
    }
    
    /**
     * Supply assets to lending pool
     */
    public LendingPosition supply(String token, double amount) throws Exception {
        URL url = new URL(API_BASE + "/lending/supply");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) {
            conn.setRequestProperty("Authorization", "Bearer " + authToken);
        }
        conn.setDoOutput(true);
        
        JSONObject body = new JSONObject();
        body.put("token", token);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        int responseCode = conn.getResponseCode();
        if (responseCode == 201) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONObject("data");
            return new LendingPosition(
                data.getString("id"),
                data.getString("token"),
                data.getDouble("supplied"),
                data.getDouble("borrowed"),
                data.getDouble("apy"),
                data.getString("status")
            );
        }
        throw new Exception("Failed to supply: " + responseCode);
    }
    
    /**
     * Borrow from lending pool
     */
    public LendingPosition borrow(String token, double amount) throws Exception {
        URL url = new URL(API_BASE + "/lending/borrow");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) {
            conn.setRequestProperty("Authorization", "Bearer " + authToken);
        }
        conn.setDoOutput(true);
        
        JSONObject body = new JSONObject();
        body.put("token", token);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        int responseCode = conn.getResponseCode();
        if (responseCode == 201) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONObject("data");
            return new LendingPosition(
                data.getString("id"),
                data.getString("token"),
                data.getDouble("supplied"),
                data.getDouble("borrowed"),
                data.getDouble("apy"),
                data.getString("status")
            );
        }
        throw new Exception("Failed to borrow: " + responseCode);
    }
    
    /**
     * Repay borrowed amount
     */
    public boolean repay(String token, double amount) throws Exception {
        URL url = new URL(API_BASE + "/lending/repay");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) {
            conn.setRequestProperty("Authorization", "Bearer " + authToken);
        }
        conn.setDoOutput(true);
        
        JSONObject body = new JSONObject();
        body.put("token", token);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        return conn.getResponseCode() == 200;
    }
    
    /**
     * Withdraw supplied assets
     */
    public boolean withdraw(String token, double amount) throws Exception {
        URL url = new URL(API_BASE + "/lending/withdraw");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) {
            conn.setRequestProperty("Authorization", "Bearer " + authToken);
        }
        conn.setDoOutput(true);
        
        JSONObject body = new JSONObject();
        body.put("token", token);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        return conn.getResponseCode() == 200;
    }
    
    /**
     * Get user's lending positions
     */
    public LendingPosition[] getUserPositions() throws Exception {
        HttpURLConnection conn = createConnection("/lending/positions");
        int responseCode = conn.getResponseCode();
        
        if (responseCode == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            LendingPosition[] positions = new LendingPosition[data.length()];
            for (int i = 0; i < data.length(); i++) {
                JSONObject p = data.getJSONObject(i);
                positions[i] = new LendingPosition(
                    p.getString("id"),
                    p.getString("token"),
                    p.getDouble("supplied"),
                    p.getDouble("borrowed"),
                    p.getDouble("apy"),
                    p.getString("status")
                );
            }
            return positions;
        }
        return new LendingPosition[0];
    }
    
    /**
     * Get health factor
     */
    public double getHealthFactor() throws Exception {
        HttpURLConnection conn = createConnection("/lending/health");
        int responseCode = conn.getResponseCode();
        
        if (responseCode == 200) {
            String response = readResponse(conn);
            return new JSONObject(response).getDouble("data");
        }
        return 999.0;
    }
    
    private String readResponse(HttpURLConnection conn) throws Exception {
        BufferedReader reader = new BufferedReader(
            new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            response.append(line);
        }
        reader.close();
        return response.toString();
    }
    
    // Data classes
    public static class LendingPool {
        public String token;
        public String name;
        public double totalSupplied;
        public double totalBorrowed;
        public double supplyAPY;
        public double borrowAPY;
        public double liquidity;
        
        public LendingPool(String token, String name, double totalSupplied, 
                         double totalBorrowed, double supplyAPY, double borrowAPY, double liquidity) {
            this.token = token;
            this.name = name;
            this.totalSupplied = totalSupplied;
            this.totalBorrowed = totalBorrowed;
            this.supplyAPY = supplyAPY;
            this.borrowAPY = borrowAPY;
            this.liquidity = liquidity;
        }
    }
    
    public static class LendingPosition {
        public String id;
        public String token;
        public double supplied;
        public double borrowed;
        public double apy;
        public String status;
        
        public LendingPosition(String id, String token, double supplied, 
                             double borrowed, double apy, String status) {
            this.id = id;
            this.token = token;
            this.supplied = supplied;
            this.borrowed = borrowed;
            this.apy = apy;
            this.status = status;
        }
    }
}
