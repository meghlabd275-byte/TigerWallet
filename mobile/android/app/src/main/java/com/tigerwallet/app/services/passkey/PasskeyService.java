package com.tigerwallet.app.services.passkey;

import android.content.Context;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.Signature;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Passkey Service for Android
 * Implements WebAuthn/FIDO2 for passwordless authentication
 */
public class PasskeyService {
    private static final String TAG = "PasskeyService";
    private static final String BASE_URL = "https://api.tigerwallet.com/v1/passkey";
    
    private final Context context;
    private final ExecutorService executor;
    private KeyPair currentKeyPair;
    
    public interface Callback<T> {
        void onSuccess(T result);
        void onError(Exception error);
    }
    
    public PasskeyService(Context context) {
        this.context = context;
        this.executor = Executors.newFixedThreadPool(4);
    }
    
    /**
     * Register a new passkey
     */
    public void register(String username, String domain, Callback<PasskeyCredential> callback) {
        executor.execute(() -> {
            try {
                // Generate key pair in Android Keystore
                KeyPair keyPair = generateKeyPair(domain);
                this.currentKeyPair = keyPair;
                
                // Get public key to send to server
                String publicKeyJwk = publicKeyToJwk(keyPair.getPublic());
                
                JSONObject body = new JSONObject();
                body.put("username", username);
                body.put("domain", domain);
                body.put("publicKey", new JSONObject(publicKeyJwk));
                
                JSONObject response = makePostRequest("/register", body);
                
                // Sign the challenge with private key
                String challenge = response.getString("challenge");
                String signature = signChallenge(challenge);
                
                // Send signature to server
                JSONObject verifyBody = new JSONObject();
                verifyBody.put("challenge", challenge);
                verifyBody.put("signature", signature);
                verifyBody.put("clientDataJSON", response.getString("clientDataJSON"));
                
                JSONObject verifyResponse = makePostRequest("/register/verify", verifyBody);
                
                PasskeyCredential credential = new PasskeyCredential();
                credential.credentialId = verifyResponse.getString("credentialId");
                credential.username = username;
                credential.domain = domain;
                credential.createdAt = verifyResponse.getLong("createdAt");
                
                callback.onSuccess(credential);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Authenticate with passkey
     */
    public void authenticate(String username, Callback<AuthResult> callback) {
        executor.execute(() -> {
            try {
                // Get challenge from server
                JSONObject body = new JSONObject();
                body.put("username", username);
                
                JSONObject response = makePostRequest("/authenticate", body);
                
                String challenge = response.getString("challenge");
                
                // Sign the challenge
                if (currentKeyPair == null) {
                    callback.onError(new Exception("No passkey registered"));
                    return;
                }
                
                String signature = signChallenge(challenge);
                
                // Verify with server
                JSONObject verifyBody = new JSONObject();
                verifyBody.put("challenge", challenge);
                verifyBody.put("signature", signature);
                verifyBody.put("clientDataJSON", response.getString("clientDataJSON"));
                verifyBody.put("credentialId", response.getString("credentialId"));
                
                JSONObject verifyResponse = makePostRequest("/authenticate/verify", verifyBody);
                
                AuthResult result = new AuthResult();
                result.success = verifyResponse.getBoolean("success");
                result.token = verifyResponse.optString("token", "");
                result.expiresAt = verifyResponse.getLong("expiresAt");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * List registered passkeys for user
     */
    public void listPasskeys(String username, Callback<List<PasskeyCredential>> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/credentials/" + username);
                JSONArray credentialsArray = response.getJSONArray("credentials");
                
                List<PasskeyCredential> credentials = new ArrayList<>();
                for (int i = 0; i < credentialsArray.length(); i++) {
                    JSONObject c = credentialsArray.getJSONObject(i);
                    PasskeyCredential credential = new PasskeyCredential();
                    credential.credentialId = c.getString("credentialId");
                    credential.username = c.getString("username");
                    credential.domain = c.getString("domain");
                    credential.createdAt = c.getLong("createdAt");
                    credential.lastUsed = c.optLong("lastUsed", 0);
                    credentials.add(credential);
                }
                
                callback.onSuccess(credentials);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Delete a passkey
     */
    public void deletePasskey(String credentialId, Callback<Boolean> callback) {
        executor.execute(() -> {
            try {
                URL url = new URL(BASE_URL + "/credentials/" + credentialId);
                HttpURLConnection conn = (HttpURLConnection) url.openConnection();
                conn.setRequestMethod("DELETE");
                
                int responseCode = conn.getResponseCode();
                callback.onSuccess(responseCode == 200 || responseCode == 204);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Update passkey usage
     */
    public void updateUsage(String credentialId, Callback<Boolean> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("credentialId", credentialId);
                
                makePostRequest("/credentials/" + credentialId + "/use", body);
                callback.onSuccess(true);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Check if device supports passkeys
     */
    public boolean isSupported() {
        try {
            KeyPairGenerator generator = KeyPairGenerator.getInstance(
                KeyProperties.KEY_ALGORITHM_RSA,
                "AndroidKeyStore"
            );
            return true;
        } catch (Exception e) {
            return false;
        }
    }
    
    private KeyPair generateKeyPair(String domain) throws Exception {
        KeyPairGenerator generator = KeyPairGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_RSA,
            "AndroidKeyStore"
        );
        
        KeyGenParameterSpec spec = new KeyGenParameterSpec.Builder(
            "passkey_" + domain,
            KeyProperties.PURPOSE_SIGN
        )
            .setDigests(KeyProperties.DIGEST_SHA256, KeyProperties.DIGEST_SHA512)
            .setSignaturePaddings(KeyProperties.SIGNATURE_PADDING_RSA_PSS)
            .build();
        
        generator.initialize(spec);
        return generator.generateKeyPair();
    }
    
    private String signChallenge(String challenge) throws Exception {
        if (currentKeyPair == null) {
            throw new Exception("No key pair available");
        }
        
        Signature signature = Signature.getInstance("SHA256withRSA/PSS");
        signature.initSign(currentKeyPair.getPrivate());
        signature.update(challenge.getBytes(StandardCharsets.UTF_8));
        
        byte[] sigBytes = signature.sign();
        return Base64.encodeToString(sigBytes, Base64.NO_WRAP);
    }
    
    private String publicKeyToJwk(PublicKey publicKey) throws Exception {
        // Simplified JWK conversion
        JSONObject jwk = new JSONObject();
        jwk.put("kty", "RSA");
        jwk.put("alg", "RS256");
        jwk.put("ext", true);
        return jwk.toString();
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
    public static class PasskeyCredential {
        public String credentialId;
        public String username;
        public String domain;
        public long createdAt;
        public long lastUsed;
    }
    
    public static class AuthResult {
        public boolean success;
        public String token;
        public long expiresAt;
    }
}
