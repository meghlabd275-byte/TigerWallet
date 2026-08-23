package com.tigerwallet.app.services;

import org.json.JSONObject;
import java.net.HttpURLConnection;
import java.net.URL;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;

public class FiatRampService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public FiatRampService(String token) { this.authToken = token; }
    
    public String buyCrypto(String amount, String currency, String crypto) throws Exception {
        URL url = new URL(API_BASE + "/fiat/buy");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("amount", amount);
        body.put("currency", currency);
        body.put("crypto", crypto);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            return new JSONObject(response).getJSONObject("data").getString("orderId");
        }
        return null;
    }
    
    public String sellCrypto(double amount, String crypto, String currency) throws Exception {
        URL url = new URL(API_BASE + "/fiat/sell");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("amount", amount);
        body.put("crypto", crypto);
        body.put("currency", currency);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            return new JSONObject(response).getJSONObject("data").getString("orderId");
        }
        return null;
    }
    
    public String getStatus(String orderId) throws Exception {
        URL url = new URL(API_BASE + "/fiat/order/" + orderId);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            return new JSONObject(response).getJSONObject("data").getString("status");
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
}
