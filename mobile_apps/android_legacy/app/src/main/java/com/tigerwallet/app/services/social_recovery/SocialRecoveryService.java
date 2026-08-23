package com.tigerwallet.app.services.social_recovery;

import android.content.Context;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Social Recovery Service for Android
 * Enables wallet recovery through trusted guardians
 */
public class SocialRecoveryService {
    private static final String TAG = "SocialRecoveryService";
    private static final String BASE_URL = "https://api.tigerwallet.com/v1/social-recovery";
    
    private final Context context;
    private final ExecutorService executor;
    
    public interface Callback<T> {
        void onSuccess(T result);
        void onError(Exception error);
    }
    
    public SocialRecoveryService(Context context) {
        this.context = context;
        this.executor = Executors.newFixedThreadPool(4);
    }
    
    /**
     * Setup social recovery for a wallet
     */
    public void setupRecovery(String walletAddress, List<Guardian> guardians, Callback<SetupResult> callback) {
        executor.execute(() -> {
            try {
                JSONArray guardianArray = new JSONArray();
                for (Guardian guardian : guardians) {
                    JSONObject g = new JSONObject();
                    g.put("address", guardian.address);
                    g.put("type", guardian.type);
                    g.put("weight", guardian.weight);
                    guardianArray.put(g);
                }
                
                JSONObject body = new JSONObject();
                body.put("walletAddress", walletAddress);
                body.put("guardians", guardianArray);
                
                JSONObject response = makePostRequest("/setup", body);
                
                SetupResult result = new SetupResult();
                result.success = response.getBoolean("success");
                result.threshold = response.getInt("threshold");
                result.guardianCount = response.getInt("guardianCount");
                result.transactionHash = response.optString("transactionHash", "");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Add a guardian to the wallet
     */
    public void addGuardian(String walletAddress, String guardianAddress, String guardianType, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("walletAddress", walletAddress);
                body.put("guardianAddress", guardianAddress);
                body.put("guardianType", guardianType);
                
                JSONObject response = makePostRequest("/guardian", body);
                callback.onSuccess(response.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Remove a guardian from the wallet
     */
    public void removeGuardian(String walletAddress, String guardianAddress, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("walletAddress", walletAddress);
                body.put("guardianAddress", guardianAddress);
                
                // DELETE request
                URL url = new URL(BASE_URL + "/guardian");
                HttpURLConnection conn = (HttpURLConnection) url.openConnection();
                conn.setRequestMethod("DELETE");
                conn.setRequestProperty("Content-Type", "application/json");
                conn.setDoOutput(true);
                conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
                
                BufferedReader reader = new BufferedReader(
                    new InputStreamReader(conn.getInputStream())
                );
                StringBuilder response = new StringBuilder();
                String line;
                while ((line = reader.readLine()) != null) {
                    response.append(line);
                }
                reader.close();
                
                JSONObject result = new JSONObject(response.toString());
                callback.onSuccess(result.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get list of guardians for a wallet
     */
    public void getGuardians(String walletAddress, Callback<List<Guardian>> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/guardians/" + walletAddress);
                JSONArray guardiansArray = response.getJSONArray("guardians");
                
                List<Guardian> guardians = new ArrayList<>();
                for (int i = 0; i < guardiansArray.length(); i++) {
                    JSONObject g = guardiansArray.getJSONObject(i);
                    Guardian guardian = new Guardian();
                    guardian.address = g.getString("address");
                    guardian.type = g.getString("type");
                    guardian.weight = g.getInt("weight");
                    guardian.confirmed = g.getBoolean("confirmed");
                    guardians.add(guardian);
                }
                
                callback.onSuccess(guardians);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Initiate wallet recovery
     */
    public void initiateRecovery(String walletAddress, String newOwnerAddress, Callback<RecoveryRequest> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("walletAddress", walletAddress);
                body.put("newOwnerAddress", newOwnerAddress);
                
                JSONObject response = makePostRequest("/initiate", body);
                
                RecoveryRequest request = new RecoveryRequest();
                request.recoveryId = response.getString("recoveryId");
                request.walletAddress = response.getString("walletAddress");
                request.newOwnerAddress = response.getString("newOwnerAddress");
                request.initiatedAt = response.getLong("initiatedAt");
                request.expiresAt = response.getLong("expiresAt");
                request.confirmations = response.getInt("confirmations");
                request.requiredConfirmations = response.getInt("requiredConfirmations");
                request.status = response.getString("status");
                
                callback.onSuccess(request);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Confirm recovery as a guardian
     */
    public void confirmRecovery(String recoveryId, String guardianAddress, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("recoveryId", recoveryId);
                body.put("guardianAddress", guardianAddress);
                
                JSONObject response = makePostRequest("/confirm", body);
                callback.onSuccess(response.getString("confirmationHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Execute recovery after sufficient confirmations
     */
    public void executeRecovery(String recoveryId, String newOwnerAddress, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("recoveryId", recoveryId);
                body.put("newOwnerAddress", newOwnerAddress);
                
                JSONObject response = makePostRequest("/execute", body);
                callback.onSuccess(response.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Cancel recovery attempt
     */
    public void cancelRecovery(String walletAddress, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("walletAddress", walletAddress);
                
                JSONObject response = makePostRequest("/cancel", body);
                callback.onSuccess(response.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get pending recovery requests
     */
    public void getPendingRecoveries(String address, Callback<List<RecoveryRequest>> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/pending/" + address);
                JSONArray requestsArray = response.getJSONArray("recoveries");
                
                List<RecoveryRequest> requests = new ArrayList<>();
                for (int i = 0; i < requestsArray.length(); i++) {
                    JSONObject r = requestsArray.getJSONObject(i);
                    RecoveryRequest request = new RecoveryRequest();
                    request.recoveryId = r.getString("recoveryId");
                    request.walletAddress = r.getString("walletAddress");
                    request.newOwnerAddress = r.getString("newOwnerAddress");
                    request.initiatedAt = r.getLong("initiatedAt");
                    request.expiresAt = r.getLong("expiresAt");
                    request.confirmations = r.getInt("confirmations");
                    request.requiredConfirmations = r.getInt("requiredConfirmations");
                    request.status = r.getString("status");
                    requests.add(request);
                }
                
                callback.onSuccess(requests);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Change recovery threshold
     */
    public void changeThreshold(String walletAddress, int newThreshold, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("walletAddress", walletAddress);
                body.put("threshold", newThreshold);
                
                JSONObject response = makePostRequest("/threshold", body);
                callback.onSuccess(response.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    private JSONObject makePostRequest(String endpoint, JSONObject body) throws Exception {
        URL url = new URL(BASE_URL + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setRequestProperty("Content-Type", "application/json");
        conn.setDoOutput(true);
        
        conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));
        
        BufferedReader reader = new BufferedReader(
            new InputStreamReader(conn.getInputStream())
        );
        
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            response.append(line);
        }
        reader.close();
        
        return new JSONObject(response.toString());
    }
    
    private JSONObject makeGetRequest(String endpoint) throws Exception {
        URL url = new URL(BASE_URL + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("GET");
        
        BufferedReader reader = new BufferedReader(
            new InputStreamReader(conn.getInputStream())
        );
        
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            response.append(line);
        }
        reader.close();
        
        return new JSONObject(response.toString());
    }
    
    // Data classes
    public static class Guardian {
        public String address;
        public String type; // "wallet", "email", "phone", "social"
        public int weight;
        public boolean confirmed;
    }
    
    public static class SetupResult {
        public boolean success;
        public int threshold;
        public int guardianCount;
        public String transactionHash;
    }
    
    public static class RecoveryRequest {
        public String recoveryId;
        public String walletAddress;
        public String newOwnerAddress;
        public long initiatedAt;
        public long expiresAt;
        public int confirmations;
        public int requiredConfirmations;
        public String status; // "pending", "confirming", "ready", "executed", "cancelled"
    }
}
