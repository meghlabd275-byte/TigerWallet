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

public class RWATradingService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public RWATradingService(String token) { this.authToken = token; }
    
    public List<RWA> getRWAs() throws Exception {
        List<RWA> rwas = new ArrayList<>();
        URL url = new URL(API_BASE + "/rwa/list");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject r = data.getJSONObject(i);
                rwas.add(new RWA(r.getString("id"), r.getString("name"), r.getString("type"), r.getDouble("price"), r.getDouble("marketCap")));
            }
        }
        return rwas;
    }
    
    public boolean buyRWA(String rwaId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/rwa/" + rwaId + "/buy");
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
    
    public boolean sellRWA(String rwaId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/rwa/" + rwaId + "/sell");
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
    
    private String readResponse(HttpURLConnection conn) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) response.append(line);
        reader.close();
        return response.toString();
    }
    
    public static class RWA { public String id, name, type; public double price, marketCap; public RWA(String i, String n, String t, double p, double m) { id=i; name=n; type=t; price=p; marketCap=m; } }
}

class GasTrackerService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    
    public GasPrice getGasPrice(String chain) throws Exception {
        URL url = new URL(API_BASE + "/gas/price?chain=" + chain);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (conn.getResponseCode() == 200) {
            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
            StringBuilder response = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) response.append(line);
            reader.close();
            JSONObject data = new JSONObject(response.toString()).getJSONObject("data");
            return new GasPrice(data.getDouble("slow"), data.getDouble("standard"), data.getDouble("fast"));
        }
        return null;
    }
    
    public static class GasPrice { public double slow, standard, fast; public GasPrice(double s, double st, double f) { slow=s; standard=st; fast=f; } }
}

class OrderbookService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public OrderbookService(String token) { this.authToken = token; }
    
    public Orderbook getOrderbook(String symbol) throws Exception {
        URL url = new URL(API_BASE + "/orderbook/" + symbol);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        if (conn.getResponseCode() == 200) {
            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
            StringBuilder response = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) response.append(line);
            reader.close();
            JSONObject data = new JSONObject(response.toString()).getJSONObject("data");
            return new Orderbook(data.getString("symbol"));
        }
        return null;
    }
    
    public boolean placeLimitOrder(String symbol, String side, double price, double quantity) throws Exception {
        URL url = new URL(API_BASE + "/orderbook/limit");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("symbol", symbol);
        body.put("side", side);
        body.put("price", price);
        body.put("quantity", quantity);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public static class Orderbook { public String symbol; public Orderbook(String s) { symbol=s; } }
}

class TWAPService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public TWAPService(String token) { this.authToken = token; }
    
    public String createTWAP(String symbol, double totalAmount, int intervals, String side) throws Exception {
        URL url = new URL(API_BASE + "/twap/create");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("symbol", symbol);
        body.put("totalAmount", totalAmount);
        body.put("intervals", intervals);
        body.put("side", side);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
            StringBuilder response = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) response.append(line);
            reader.close();
            return new JSONObject(response.toString()).getJSONObject("data").getString("id");
        }
        return null;
    }
    
    public boolean cancelTWAP(String orderId) throws Exception {
        URL url = new URL(API_BASE + "/twap/" + orderId + "/cancel");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn.getResponseCode() == 200;
    }
}
