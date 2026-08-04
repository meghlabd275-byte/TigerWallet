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

public class DAOService {
    private static final String API_BASE = "https://api.tigerwallet.com/api/v1";
    private String authToken;
    
    public DAOService(String token) { this.authToken = token; }
    
    public List<DAO> getDAOs() throws Exception {
        List<DAO> daos = new ArrayList<>();
        URL url = new URL(API_BASE + "/dao/list");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject d = data.getJSONObject(i);
                daos.add(new DAO(d.getString("id"), d.getString("name"), d.getString("description"), d.getInt("memberCount")));
            }
        }
        return daos;
    }
    
    public List<Proposal> getProposals(String daoId) throws Exception {
        List<Proposal> proposals = new ArrayList<>();
        URL url = new URL(API_BASE + "/dao/" + daoId + "/proposals");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        if (conn.getResponseCode() == 200) {
            String response = readResponse(conn);
            JSONArray data = new JSONObject(response).getJSONArray("data");
            for (int i = 0; i < data.length(); i++) {
                JSONObject p = data.getJSONObject(i);
                proposals.add(new Proposal(p.getString("id"), p.getString("title"), p.getString("description"), p.getString("status"), p.getDouble("forVotes"), p.getDouble("againstVotes")));
            }
        }
        return proposals;
    }
    
    public boolean vote(String proposalId, String choice, double weight) throws Exception {
        URL url = new URL(API_BASE + "/dao/proposals/" + proposalId + "/vote");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("choice", choice);
        body.put("weight", weight);
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        return conn.getResponseCode() == 200;
    }
    
    public boolean createProposal(String daoId, String title, String description, String type) throws Exception {
        URL url = new URL(API_BASE + "/dao/" + daoId + "/proposals");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        if (authToken != null) conn.setRequestProperty("Authorization", "Bearer " + authToken);
        conn.setDoOutput(true);
        JSONObject body = new JSONObject();
        body.put("title", title);
        body.put("description", description);
        body.put("type", type);
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
    
    public static class DAO { public String id, name, description; public int memberCount; public DAO(String i, String n, String d, int m) { id=i; name=n; description=d; memberCount=m; } }
    public static class Proposal { public String id, title, description, status; public double forVotes, againstVotes; public Proposal(String i, String t, String d, String s, double f, double a) { id=i; title=t; description=d; status=s; forVotes=f; againstVotes=a; } }
}
