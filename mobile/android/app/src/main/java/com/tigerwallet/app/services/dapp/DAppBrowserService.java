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

public class DAppBrowserService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public DAppBrowserService(String token) { this.authToken = token; }
    
    public List<DApp> getFeaturedDApps() throws Exception {
        List<DApp> dapps = new ArrayList<>();
        URL url = new URL(API_BASE + "/dapp/featured");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject d = data.getJSONObject(i);
                dapps.add(new DApp(d.getString("id"), d.getString("name"), d.getString("url"), d.getString("category"), d.getString("logoUrl")));
            }
        }
        return dapps;
    }
    
    public boolean connectWallet(String dappId, String address, String chain) throws Exception {
        URL url = new URL(API_BASE + "/dapp/connect");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("dappId", dappId);
        body.put("address", address);
        body.put("chain", chain);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public String requestTransaction(String dappId, String to, String value, String data) throws Exception {
        URL url = new URL(API_BASE + "/dapp/transaction");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("dappId", dappId);
        body.put("to", to);
        body.put("value", value);
        body.put("data", data);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            return new JSONObject(response).getJSONObject("data").getString("requestId");
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
    
    public static class DApp { public String id, name, url, category, logoUrl; public DApp(String i, String n, String u, String c, String l) { id=i; name=n; url=u; category=c; logoUrl=l; } }
}
