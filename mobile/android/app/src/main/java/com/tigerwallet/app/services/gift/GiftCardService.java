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

public class GiftCardService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public GiftCardService(String token) { this.authToken = token; }
    
    public List<GiftCardBrand> getBrands() throws Exception {
        List<GiftCardBrand> brands = new ArrayList<>();
        URL url = new URL(API_BASE + "/giftcards/brands");
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
                JSONObject b = data.getJSONObject(i);
                brands.add(new GiftCardBrand(b.getString("id"), b.getString("name"), b.getString("logoUrl"), b.getDouble("discount")));
            }
        }
        return brands;
    }
    
    public String buyGiftCard(String brandId, double amount) throws Exception {
        URL url = new URL(API_BASE + "/giftcards/buy");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("brandId", brandId);
        body.put("amount", amount);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8));
            StringBuilder response = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) response.append(line);
            reader.close();
            return new JSONObject(response.toString()).getJSONObject("data").getString("code");
        }
        return null;
    }
    
    public boolean redeemGiftCard(String code) throws Exception {
        URL url = new URL(API_BASE + "/giftcards/redeem");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("code", code);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 200;
    }
    
    public static class GiftCardBrand { public String id, name, logoUrl; public double discount; public GiftCardBrand(String i, String n, String l, double d) { id=i; name=n; logoUrl=l; discount=d; } }
}
