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

public class LaunchpadService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public LaunchpadService(String token) { this.authToken = token; }
    
    public List<Launch> getActiveLaunches() throws Exception {
        List<Launch> launches = new ArrayList<>();
        URL url = new URL(API_BASE + "/launchpad/active");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject l = data.getJSONObject(i);
                launches.add(new Launch(l.getString("id"), l.getString("name"), l.getString("symbol"), l.getDouble("price"), l.getDouble("hardCap"), l.getDouble("raised"), l.getString("status")));
            }
        }
        return launches;
    }
    
    public boolean participate(String launchId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/launchpad/" + launchId + "/participate");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public boolean claimTokens(String launchId) throws Exception {
        URL url = new URL(API_BASE + "/launchpad/" + launchId + "/claim");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn.getResponseCode() == 200;
    }
    
    private String readResponse(HttpURLConnection conn) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) response.append(line);
        reader.close();
        return response.toString();
    }
    
    public static class Launch { public String id, name, symbol, status; public double price, hardCap, raised; public Launch(String i, String n, String s, double p, double h, double r, String st) { id=i; name=n; symbol=s; price=p; hardCap=h; raised=r; status=st; } }
}

class PredictionMarketsService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public PredictionMarketsService(String token) { this.authToken = token; }
    
    public List<PredictionMarket> getMarkets() throws Exception {
        List<PredictionMarket> markets = new ArrayList<>();
        URL url = new URL(API_BASE + "/prediction/markets");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
            StringBuilder response = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) response.append(line);
            reader.close();
            JSONArray data = new JSONObject(response.toString()).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject m = data.getJSONObject(i);
                markets.add(new PredictionMarket(m.getString("id"), m.getString("question"), m.getDouble("volume"), m.getString("status")));
            }
        }
        return markets;
    }
    
    public boolean placeBet(String marketId, String outcome, double amount) throws Exception {
        URL url = new URL(API_BASE + "/prediction/" + marketId + "/bet");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("outcome", outcome);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public static class PredictionMarket { public String id, question, status; public double volume; public PredictionMarket(String i, String q, double v, String s) { id=i; question=q; volume=v; status=s; } }
}
