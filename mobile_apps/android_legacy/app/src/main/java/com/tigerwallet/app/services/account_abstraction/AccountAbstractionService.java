package com.tigerwallet.app.services.account_abstraction;

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
 * Account Abstraction Service for Android
 * Implements ERC-4337 for smart contract wallets
 */
public class AccountAbstractionService {
    private static final String TAG = "AccountAbstractionService";
    private static final String BASE_URL = "https://api.tigerwallet.com/v1/aa";
    
    // Entry point contract addresses by chain
    public static final String ENTRY_POINT_ETHEREUM = "0x5FF137D4b0FD96D8E563E5b6E3a4D7B7e1d5C8A";
    public static final String ENTRY_POINT_POLYGON = "0x5FF137D4b0FD96D8E563E5b6E3a4D7B7e1d5C8A";
    public static final String ENTRY_POINT_BSC = "0x5FF137D4b0FD96D8E563E5b6E3a4D7B7e1d5C8A";
    
    private final Context context;
    private final ExecutorService executor;
    
    public interface Callback<T> {
        void onSuccess(T result);
        void onError(Exception error);
    }
    
    public AccountAbstractionService(Context context) {
        this.context = context;
        this.executor = Executors.newFixedThreadPool(4);
    }
    
    /**
     * Create a smart account for a user
     */
    public void createAccount(String ownerAddress, String salt, Callback<SmartAccount> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("ownerAddress", ownerAddress);
                if (salt != null) body.put("salt", salt);
                
                JSONObject response = makePostRequest("/account", body);
                
                SmartAccount account = new SmartAccount();
                account.address = response.getString("address");
                account.owner = response.getString("owner");
                account.nonce = response.getLong("nonce");
                account.factory = response.getString("factory");
                account.chainId = response.getInt("chainId");
                
                callback.onSuccess(account);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get the predicted account address before creation
     */
    public void getAccountAddress(String ownerAddress, int index, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/address/" + ownerAddress + "/" + index);
                callback.onSuccess(response.getString("address"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get account nonce for transaction ordering
     */
    public void getNonce(String accountAddress, Callback<Long> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/nonce/" + accountAddress);
                callback.onSuccess(response.getLong("nonce"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get account balance
     */
    public void getBalance(String accountAddress, Callback<AccountBalance> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/balance/" + accountAddress);
                
                AccountBalance balance = new AccountBalance();
                balance.native = response.getString("native");
                balance.nativeUSD = response.getDouble("nativeUSD");
                balance.tokens = new ArrayList<>();
                
                JSONArray tokensArray = response.optJSONArray("tokens");
                if (tokensArray != null) {
                    for (int i = 0; i < tokensArray.length(); i++) {
                        JSONObject t = tokensArray.getJSONObject(i);
                        TokenBalance tb = new TokenBalance();
                        tb.address = t.getString("address");
                        tb.symbol = t.getString("symbol");
                        tb.balance = t.getString("balance");
                        tb.balanceUSD = t.getDouble("balanceUSD");
                        balance.tokens.add(tb);
                    }
                }
                
                callback.onSuccess(balance);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Execute a user operation
     */
    public void executeUserOp(UserOperation userOp, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("sender", userOp.sender);
                body.put("nonce", userOp.nonce);
                body.put("initCode", userOp.initCode);
                body.put("callData", userOp.callData);
                body.put("callGasLimit", userOp.callGasLimit);
                body.put("verificationGasLimit", userOp.verificationGasLimit);
                body.put("preVerificationGas", userOp.preVerificationGas);
                body.put("maxFeePerGas", userOp.maxFeePerGas);
                body.put("maxPriorityFeePerGas", userOp.maxPriorityFeePerGas);
                body.put("paymasterAndData", userOp.paymasterAndData);
                body.put("signature", userOp.signature);
                
                JSONObject response = makePostRequest("/execute", body);
                callback.onSuccess(response.getString("userOpHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Bundle multiple user operations
     */
    public void bundleUserOps(List<UserOperation> userOps, Callback<BundleResult> callback) {
        executor.execute(() -> {
            try {
                JSONArray opsArray = new JSONArray();
                for (UserOperation op : userOps) {
                    JSONObject o = new JSONObject();
                    o.put("sender", op.sender);
                    o.put("nonce", op.nonce);
                    o.put("callData", op.callData);
                    opsArray.put(o);
                }
                
                JSONObject body = new JSONObject();
                body.put("userOps", opsArray);
                
                JSONObject response = makePostRequest("/bundle", body);
                
                BundleResult result = new BundleResult();
                result.bundleId = response.getString("bundleId");
                result.userOpHash = response.getString("userOpHash");
                result.blockNumber = response.getInt("blockNumber");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Estimate gas for a user operation
     */
    public void estimateGas(UserOperation userOp, Callback<GasEstimate> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("sender", userOp.sender);
                body.put("callData", userOp.callData);
                
                JSONObject response = makePostRequest("/estimate", body);
                
                GasEstimate estimate = new GasEstimate();
                estimate.callGasLimit = response.getLong("callGasLimit");
                estimate.verificationGasLimit = response.getLong("verificationGasLimit");
                estimate.preVerificationGas = response.getLong("preVerificationGas");
                estimate.maxFeePerGas = response.getLong("maxFeePerGas");
                estimate.maxPriorityFeePerGas = response.getLong("maxPriorityFeePerGas");
                estimate.totalGas = estimate.callGasLimit + estimate.verificationGasLimit + estimate.preVerificationGas;
                
                callback.onSuccess(estimate);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Set signature aggregator
     */
    public void setAggregator(String aggregatorAddress, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("aggregatorAddress", aggregatorAddress);
                
                JSONObject response = makePostRequest("/aggregator", body);
                callback.onSuccess(response.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get entry point address for a chain
     */
    public String getEntryPoint(String chain) {
        switch (chain.toLowerCase()) {
            case "ethereum":
                return ENTRY_POINT_ETHEREUM;
            case "polygon":
                return ENTRY_POINT_POLYGON;
            case "bsc":
                return ENTRY_POINT_BSC;
            default:
                return ENTRY_POINT_ETHEREUM;
        }
    }
    
    /**
     * Build a user operation
     */
    public UserOperation buildUserOp(String to, String data, String from, long nonce) {
        UserOperation op = new UserOperation();
        op.sender = from;
        op.nonce = Long.toHexString(nonce);
        op.initCode = "0x";
        op.callData = buildCallData(to, data);
        op.callGasLimit = "0x0";
        op.verificationGasLimit = "0x0";
        op.preVerificationGas = "0x0";
        op.maxFeePerGas = "0x0";
        op.maxPriorityFeePerGas = "0x0";
        op.paymasterAndData = "0x";
        op.signature = "0x";
        return op;
    }
    
    private String buildCallData(String to, String data) {
        // ERC-2771 style call data
        try {
            JSONObject callData = new JSONObject();
            callData.put("to", to);
            callData.put("data", data);
            return callData.toString();
        } catch (Exception e) {
            return "0x";
        }
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
    public static class SmartAccount {
        public String address;
        public String owner;
        public long nonce;
        public String factory;
        public int chainId;
    }
    
    public static class AccountBalance {
        public String native;
        public double nativeUSD;
        public List<TokenBalance> tokens;
    }
    
    public static class TokenBalance {
        public String address;
        public String symbol;
        public String balance;
        public double balanceUSD;
    }
    
    public static class UserOperation {
        public String sender;
        public String nonce;
        public String initCode;
        public String callData;
        public String callGasLimit;
        public String verificationGasLimit;
        public String preVerificationGas;
        public String maxFeePerGas;
        public String maxPriorityFeePerGas;
        public String paymasterAndData;
        public String signature;
    }
    
    public static class GasEstimate {
        public long callGasLimit;
        public long verificationGasLimit;
        public long preVerificationGas;
        public long maxFeePerGas;
        public long maxPriorityFeePerGas;
        public long totalGas;
    }
    
    public static class BundleResult {
        public String bundleId;
        public String userOpHash;
        public int blockNumber;
    }
}
