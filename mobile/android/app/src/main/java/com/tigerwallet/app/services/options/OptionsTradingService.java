package com.tigerwallet.app.services;

import org.json.JSONArray;
import org.json.JSONObject;
import java.net.HttpURLConnection;
import java.net.URL;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;

public class OptionsTradingService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public OptionsTradingService(String token) { this.authToken = token; }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn;
    }
    
    public OptionsContract[] getOptions() throws Exception {
        HttpURLConnection conn = createConnection("/options/contracts");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            OptionsContract[] contracts = new OptionsContract[data.length()];
            for (int i = 0; i < data.length(); i++) {
                JSONObject c = data.getJSONObject(i);
                contracts[i] = new OptionsContract(c.getString("id"), c.getString("symbol"), 
                    c.getDouble("strike"), c.getDouble("premium"), c.getString("expiry"), c.getString("type"));
            }
            return contracts;
        }
        return new OptionsContract[0];
    }
    
    public OptionsOrder buyOption(String contractId, int quantity) throws Exception {
        URL url = new URL(API_BASE + "/options/buy");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("contractId", contractId);
        body.put("quantity", quantity);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONObject("data");
            return new OptionsOrder(data.getString("id"), data.getString("contractId"), 
                data.getDouble("quantity"), data.getDouble("price"), data.getString("status"));
        }
        return null;
    }
    
    public boolean exerciseOption(String orderId) throws Exception {
        URL url = new URL(API_BASE + "/options/" + orderId + "/exercise");
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
    
    public static class OptionsContract {
        public String id, symbol, expiry, type;
        public double strike, premium;
        public OptionsContract(String id, String symbol, double strike, double premium, String expiry, String type) {
            this.id = id; this.symbol = symbol; this.strike = strike; this.premium = premium;
            this.expiry = expiry; this.type = type;
        }
    }
    
    public static class OptionsOrder {
        public String id, contractId, status;
        public double quantity, price;
        public OptionsOrder(String id, String contractId, double quantity, double price, String status) {
            this.id = id; this.contractId = contractId; this.quantity = quantity; this.price = price; this.status = status;
        }
    }
}
