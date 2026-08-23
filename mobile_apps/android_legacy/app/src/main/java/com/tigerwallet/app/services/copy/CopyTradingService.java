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

public class CopyTradingService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public CopyTradingService(String token) { this.authToken = token; }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn;
    }
    
    public List<Trader> getTopTraders() throws Exception {
        List<Trader> traders = new ArrayList<>();
        HttpURLConnection conn = createConnection("/copy/traders");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject t = data.getJSONObject(i);
                traders.add(new Trader(t.getString("id"), t.getString("name"), 
                    t.getDouble("pnl"), t.getDouble("followers"), t.getDouble("winRate")));
            }
        }
        return traders;
    }
    
    public boolean followTrader(String traderId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/copy/follow");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("traderId", traderId);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public boolean unfollowTrader(String traderId) throws Exception {
        URL url = new URL(API_BASE + "/copy/unfollow");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("traderId", traderId);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 200;
    }
    
    public List<CopyPosition> getMyPositions() throws Exception {
        List<CopyPosition> positions = new ArrayList<>();
        HttpURLConnection conn = createConnection("/copy/positions");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject p = data.getJSONObject(i);
                positions.add(new CopyPosition(p.getString("id"), p.getString("traderId"),
                    p.getDouble("invested"), p.getDouble("pnl"), p.getString("status")));
            }
        }
        return positions;
    }
    
    private String readResponse(HttpURLConnection conn) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) response.append(line);
        reader.close();
        return response.toString();
    }
    
    public static class Trader {
        public String id, name;
        public double pnl, followers, winRate;
        public Trader(String id, String name, double pnl, double followers, double winRate) {
            this.id = id; this.name = name; this.pnl = pnl; this.followers = followers; this.winRate = winRate;
        }
    }
    
    public static class CopyPosition {
        public String id, traderId, status;
        public double invested, pnl;
        public CopyPosition(String id, String traderId, double invested, double pnl, String status) {
            this.id = id; this.traderId = traderId; this.invested = invested; this.pnl = pnl; this.status = status;
        }
    }
}
