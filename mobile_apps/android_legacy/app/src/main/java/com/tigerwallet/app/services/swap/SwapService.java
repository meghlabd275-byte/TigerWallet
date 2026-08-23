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

public class SwapService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public SwapService(String token) { this.authToken = token; }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn;
    }
    
    public List<Token> getTokens() throws Exception {
        List<Token> tokens = new ArrayList<>();
        HttpURLConnection conn = createConnection("/swap/tokens");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject t = data.getJSONObject(i);
                tokens.add(new Token(t.getString("symbol"), t.getString("name"), t.getDouble("price")));
            }
        }
        return tokens;
    }
    
    public SwapQuote getQuote(String fromToken, String toToken, double amount) throws Exception {
        URL url = new URL(API_BASE + "/swap/quote?from=" + fromToken + "&to=" + toToken + "&amount=" + amount);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONObject data = new JSONObject(response).getJSONObject("data");
            return new SwapQuote(data.getDouble("fromAmount"), data.getDouble("toAmount"), data.getDouble("priceImpact"), data.getDouble("gasFee"));
        }
        return null;
    }
    
    public String executeSwap(String fromToken, String toToken, double amount, double slippage) throws Exception {
        URL url = new URL(API_BASE + "/swap/execute");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("fromToken", fromToken);
        body.put("toToken", toToken);
        body.put("amount", amount);
        body.put("slippage", slippage);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            return new JSONObject(response).getJSONObject("data").getString("txHash");
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
    
    public static class Token { public String symbol, name; public double price; public Token(String s, String n, double p) { symbol=s; name=n; price=p; } }
    public static class SwapQuote { public double fromAmount, toAmount, priceImpact, gasFee; public SwapQuote(double f, double t, double p, double g) { fromAmount=f; toAmount=t; priceImpact=p; gasFee=g; } }
}
