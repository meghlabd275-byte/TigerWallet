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

public class NFTService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public NFTService(String token) { this.authToken = token; }
    
    private HttpURLConnection createConnection(String endpoint) throws Exception {
        URL url = new URL(API_BASE + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        return conn;
    }
    
    public List<NFTCollection> getCollections() throws Exception {
        List<NFTCollection> collections = new ArrayList<>();
        HttpURLConnection conn = createConnection("/nft/collections");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject c = data.getJSONObject(i);
                collections.add(new NFTCollection(c.getString("id"), c.getString("name"), c.getString("symbol"), c.getString("imageUrl"), c.getDouble("floorPrice")));
            }
        }
        return collections;
    }
    
    public List<NFT> getUserNFTs() throws Exception {
        List<NFT> nfts = new ArrayList<>();
        HttpURLConnection conn = createConnection("/nft/user/nfts");
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject n = data.getJSONObject(i);
                nfts.add(new NFT(n.getString("id"), n.getString("collectionId"), n.getString("name"), n.getString("imageUrl"), n.getDouble("price")));
            }
        }
        return nfts;
    }
    
    public boolean buyNFT(String collectionId, String tokenId, double price) throws Exception {
        URL url = new URL(API_BASE + "/nft/buy");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("collectionId", collectionId);
        body.put("tokenId", tokenId);
        body.put("price", price);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public boolean listNFT(String collectionId, String tokenId, double price) throws Exception {
        URL url = new URL(API_BASE + "/nft/list");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("collectionId", collectionId);
        body.put("tokenId", tokenId);
        body.put("price", price);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 201;
    }
    
    public String mintNFT(String collectionId, String name, String description, String imageUrl) throws Exception {
        URL url = new URL(API_BASE + "/nft/mint");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("collectionId", collectionId);
        body.put("name", name);
        body.put("description", description);
        body.put("imageUrl", imageUrl);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        if (conn.getResponseCode() == 201) {
            String response = readResponse(conn);
            return new JSONObject(response).getJSONObject("data").getString("tokenId");
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
    
    public static class NFTCollection { public String id, name, symbol, imageUrl; public double floorPrice; public NFTCollection(String i, String n, String s, String u, double p) { id=i; name=n; symbol=s; imageUrl=u; floorPrice=p; } }
    public static class NFT { public String id, collectionId, name, imageUrl; public double price; public NFT(String i, String c, String n, String u, double p) { id=i; collectionId=c; name=n; imageUrl=u; price=p; } }
}
