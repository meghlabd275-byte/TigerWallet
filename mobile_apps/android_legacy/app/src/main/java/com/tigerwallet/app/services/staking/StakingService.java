package com.tigerwallet.app.services;

import org.json.JSONArray;
import org.json.JSONObject;
import java.net.HttpURLConnection;
import java.net.URL;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

public class StakingService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public StakingService(String token) { this.authToken = token; }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn;
    }
    
    public List<StakingPool> getPools() throws Exception {
        List<StakingPool> pools = new ArrayList<>();
        HttpURLConnection conn = createConnection("/staking/pools");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject p = data.getJSONObject(i);
                pools.add(new StakingPool(p.getString("id"), p.getString("token"), p.getString("name"), p.getDouble("apy"), p.getDouble("totalStaked")));
            }
        }
        return pools;
    }
    
    public boolean stake(String poolId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/staking/stake");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("poolId", poolId);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public boolean unstake(String poolId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/staking/unstake");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("poolId", poolId);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 200;
    }
    
    public double claimRewards(String poolId) throws Exception {
        URL url = new URL(API_BASE + "/staking/claim");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("poolId", poolId);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            return new JSONObject(response).getDouble("data");
        }
        return 0;
    }
    
    private String readResponse(HttpURLConnection conn) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) response.append(line);
        reader.close();
        return response.toString();
    }
    
    public static class StakingPool { public String id, token, name; public double apy, totalStaked; public StakingPool(String i, String t, String n, double a, double s) { id=i; token=t; name=n; apy=a; totalStaked=s; } }
}
