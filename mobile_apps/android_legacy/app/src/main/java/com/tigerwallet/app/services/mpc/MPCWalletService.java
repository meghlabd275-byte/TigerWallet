package com.tigerwallet.app.services.mpc;

import android.content.Context;
import org.json.JSONObject;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * MPC Wallet Service for Android.
 *
 * Talks to the REAL Go MPC backend (go/mpc), which exposes ONLY:
 *   POST /api/v1/mpc/create           { threshold, totalShards } -> { keyId, address, publicKey, threshold, totalShards, createdAt }
 *   POST /api/v1/mpc/sign             { keyId, messageHash }      -> { signature, keyId, address, signedAt }
 *   GET  /api/v1/mpc/wallet/{keyId}                               -> { keyId, address, publicKey, threshold, totalShards, createdAt }
 *
 * There is NO import / publickey / session / rotate / sign-message / txData
 * endpoint on the backend, so those operations are not implemented here. No
 * signatures, public keys, or wallet metadata are ever fabricated locally;
 * everything comes from the backend.
 */
public class MPCWalletService {
    private static final String TAG = "MPCWalletService";

    /** Default MPC backend URL. Override via the "tigerwallet.mpc.baseurl" system property. */
    private static final String DEFAULT_BASE_URL = "http://localhost:8085";

    private final Context context;
    private final ExecutorService executor;
    private final String baseUrl;
    // Map from wallet address -> backend keyId, populated when wallets are created.
    private final Map<String, String> addressToKeyId;

    public interface Callback<T> {
        void onSuccess(T result);
        void onError(Exception error);
    }

    public MPCWalletService(Context context) {
        this(context, System.getProperty("tigerwallet.mpc.baseurl", DEFAULT_BASE_URL));
    }

    public MPCWalletService(Context context, String baseUrl) {
        this.context = context;
        this.baseUrl = baseUrl;
        this.executor = Executors.newFixedThreadPool(4);
        this.addressToKeyId = new HashMap<>();
    }

    /**
     * Create a new MPC wallet.
     *
     * The backend's createRequest only accepts {threshold, totalShards}; the
     * userId argument is intentionally NOT sent in the body. The signature is
     * kept for caller compatibility.
     */
    public void createWallet(String userId, Callback<MPCWallet> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("threshold", 2);
                body.put("totalShards", 3);

                JSONObject response = makePostRequest("/api/v1/mpc/create", body);

                MPCWallet wallet = new MPCWallet();
                wallet.keyId = response.getString("keyId");
                wallet.address = response.getString("address");
                wallet.publicKey = response.getString("publicKey");
                wallet.threshold = response.optInt("threshold");
                wallet.totalShards = response.optInt("totalShards");
                wallet.createdAt = response.getLong("createdAt");

                addressToKeyId.put(wallet.address, wallet.keyId);

                callback.onSuccess(wallet);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }

    /**
     * Sign a transaction payload using MPC.
     *
     * @param txData   transaction data to sign (raw bytes represented as a UTF-8 string)
     * @param address  wallet address whose MPC key should sign
     */
    public void signTransaction(String txData, String address, Callback<String> callback) {
        signPayload(txData, address, callback);
    }

    /**
     * Sign an arbitrary message using MPC.
     *
     * @param message  message to sign
     * @param address  wallet address whose MPC key should sign
     */
    public void signMessage(String message, String address, Callback<String> callback) {
        signPayload(message, address, callback);
    }

    private void signPayload(String payload, String address, Callback<String> callback) {
        executor.execute(() -> {
            try {
                String keyId = addressToKeyId.get(address);
                if (keyId == null) {
                    callback.onError(new IllegalStateException(
                        "no MPC wallet for address " + address
                            + " (call createWallet first)"));
                    return;
                }

                String messageHash = sha256Hex(payload);

                JSONObject body = new JSONObject();
                body.put("keyId", keyId);
                body.put("messageHash", messageHash);

                JSONObject response = makePostRequest("/api/v1/mpc/sign", body);
                callback.onSuccess(response.getString("signature"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }

    /**
     * Get wallet information for a previously created wallet.
     *
     * @param address wallet address; the backend is keyed by keyId, which is
     *                looked up from the local cache populated at create time.
     */
    public void getWalletInfo(String address, Callback<WalletInfo> callback) {
        executor.execute(() -> {
            try {
                String keyId = addressToKeyId.get(address);
                if (keyId == null) {
                    callback.onError(new IllegalStateException(
                        "no MPC wallet for address " + address
                            + " (call createWallet first)"));
                    return;
                }

                JSONObject response = makeGetRequest("/api/v1/mpc/wallet/" + keyId);

                WalletInfo info = new WalletInfo();
                info.keyId = response.getString("keyId");
                info.address = response.getString("address");
                info.publicKey = response.getString("publicKey");
                info.threshold = response.optInt("threshold");
                info.totalShards = response.optInt("totalShards");
                info.createdAt = response.getLong("createdAt");

                callback.onSuccess(info);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }

    /**
     * Compute a 32-byte message hash as hex ("0x" + hex).
     *
     * TODO: EVM-compatible signing requires keccak-256, which Android does not
     * provide in the JDK. Replace this with a real keccak-256 implementation
     * (e.g. BouncyCastle) before using these signatures against an EVM chain.
     * SHA-256 is used here only so the backend's "32-byte hex" requirement is
     * satisfied; it is NOT a valid EVM message digest. The signature itself is
     * always produced by the real backend -- nothing is fabricated.
     */
    private static String sha256Hex(String input) throws Exception {
        MessageDigest digest = MessageDigest.getInstance("SHA-256");
        byte[] hash = digest.digest(input.getBytes(StandardCharsets.UTF_8));
        StringBuilder hex = new StringBuilder(2 + hash.length * 2);
        hex.append("0x");
        for (byte b : hash) {
            hex.append(String.format("%02x", b));
        }
        return hex.toString();
    }

    private JSONObject makePostRequest(String endpoint, JSONObject body) throws Exception {
        URL url = new URL(baseUrl + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        try {
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setConnectTimeout(15000);
            conn.setReadTimeout(30000);
            conn.setDoOutput(true);

            conn.getOutputStream().write(body.toString().getBytes(StandardCharsets.UTF_8));

            return readResponse(conn);
        } finally {
            conn.disconnect();
        }
    }

    private JSONObject makeGetRequest(String endpoint) throws Exception {
        URL url = new URL(baseUrl + endpoint);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        try {
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(15000);
            conn.setReadTimeout(30000);

            return readResponse(conn);
        } finally {
            conn.disconnect();
        }
    }

    private static JSONObject readResponse(HttpURLConnection conn) throws Exception {
        int code = conn.getResponseCode();
        BufferedReader reader = new BufferedReader(
            new InputStreamReader(
                (code >= 200 && code < 400) ? conn.getInputStream() : conn.getErrorStream()
            )
        );
        StringBuilder response = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            response.append(line);
        }
        reader.close();

        String body = response.toString();
        if (code < 200 || code >= 400) {
            throw new RuntimeException("MPC backend HTTP " + code + ": " + body);
        }
        return new JSONObject(body);
    }

    // Data classes
    public static class MPCWallet {
        public String keyId;
        public String address;
        public String publicKey; // base64-encoded 65-byte uncompressed secp256k1 key (from backend)
        public int threshold;
        public int totalShards;
        public long createdAt;

        public String toJson() {
            try {
                JSONObject obj = new JSONObject();
                obj.put("keyId", keyId);
                obj.put("address", address);
                obj.put("publicKey", publicKey);
                obj.put("threshold", threshold);
                obj.put("totalShards", totalShards);
                obj.put("createdAt", createdAt);
                return obj.toString();
            } catch (Exception e) {
                return "{}";
            }
        }
    }

    // Retained for compilation compatibility. The real MPC backend has no
    // session-key endpoint, so no method populates it.
    public static class SessionKey {
        public String key;
        public long expiresAt;
    }

    public static class WalletInfo {
        public String keyId;
        public String address;
        public String publicKey;
        public int threshold;
        public int totalShards;
        public long createdAt;
    }
}
