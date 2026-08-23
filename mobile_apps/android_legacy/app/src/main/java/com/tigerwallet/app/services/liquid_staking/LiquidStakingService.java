package com.tigerwallet.app.services.liquid_staking;

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
 * Liquid Staking Service for Android
 * Provides liquid staking with derivative tokens
 */
public class LiquidStakingService {
    private static final String TAG = "LiquidStakingService";
    private static final String BASE_URL = "https://api.tigerwallet.com/v1/liquid-staking";
    
    private final Context context;
    private final ExecutorService executor;
    
    public interface Callback<T> {
        void onSuccess(T result);
        void onError(Exception error);
    }
    
    public LiquidStakingService(Context context) {
        this.context = context;
        this.executor = Executors.newFixedThreadPool(4);
    }
    
    /**
     * Get available liquid staking protocols
     */
    public void getProtocols(String chain, Callback<List<LiquidStakingProtocol>> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/" + chain + "/protocols");
                JSONArray protocolsArray = response.getJSONArray("protocols");
                
                List<LiquidStakingProtocol> protocols = new ArrayList<>();
                for (int i = 0; i < protocolsArray.length(); i++) {
                    JSONObject p = protocolsArray.getJSONObject(i);
                    LiquidStakingProtocol protocol = new LiquidStakingProtocol();
                    protocol.id = p.getString("id");
                    protocol.name = p.getString("name");
                    protocol.chain = p.getString("chain");
                    protocol.stakingToken = p.getString("stakingToken");
                    protocol.liquidToken = p.getString("liquidToken");
                    protocol.liquidTokenSymbol = p.getString("liquidTokenSymbol");
                    protocol.totalStaked = p.getDouble("totalStaked");
                    protocol.apy = p.getDouble("apy");
                    protocol.minStake = p.getString("minStake");
                    protocol.unbondingPeriod = p.getInt("unbondingPeriod");
                    protocols.add(protocol);
                }
                
                callback.onSuccess(protocols);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get protocol details
     */
    public void getProtocolDetails(String protocolId, String chain, Callback<ProtocolDetails> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/" + chain + "/protocol/" + protocolId);
                
                ProtocolDetails details = new ProtocolDetails();
                details.id = response.getString("id");
                details.name = response.getString("name");
                details.description = response.getString("description");
                details.website = response.getString("website");
                details.audits = response.getJSONArray("audits");
                details TVL = response.getDouble("TVL");
                details.apy = response.getDouble("apy");
                details.rewardsApy = response.getDouble("rewardsApy");
                
                callback.onSuccess(details);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Stake and receive liquid tokens
     */
    public void stake(String protocolId, String amount, String address, String chain, Callback<StakeResult> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("protocolId", protocolId);
                body.put("amount", amount);
                body.put("address", address);
                body.put("chain", chain);
                
                JSONObject response = makePostRequest("/" + chain + "/stake", body);
                
                StakeResult result = new StakeResult();
                result.transactionHash = response.getString("transactionHash");
                result.stakedAmount = response.getString("stakedAmount");
                result.liquidTokenReceived = response.getString("liquidTokenReceived");
                result.liquidTokenBalance = response.getString("liquidTokenBalance");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Unstake (burn liquid tokens)
     */
    public void unstake(String protocolId, String amount, String address, String chain, Callback<UnstakeResult> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("protocolId", protocolId);
                body.put("amount", amount);
                body.put("address", address);
                body.put("chain", chain);
                
                JSONObject response = makePostRequest("/" + chain + "/unstake", body);
                
                UnstakeResult result = new UnstakeResult();
                result.transactionHash = response.getString("transactionHash");
                result.liquidTokenBurned = response.getString("liquidTokenBurned");
                result.unbondingId = response.optString("unbondingId", "");
                result.availableAt = response.getLong("availableAt");
                
                callback.onSuccess(result);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get user's staked balance
     */
    public void getStakedBalance(String address, String chain, Callback<StakedBalance> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/" + chain + "/balance/" + address);
                
                StakedBalance balance = new StakedBalance();
                balance.stakedNative = response.getString("stakedNative");
                balance.liquidTokenBalance = response.getString("liquidTokenBalance");
                balance.liquidTokenValueUSD = response.getDouble("liquidTokenValueUSD");
                balance.pendingUnstakes = response.getDouble("pendingUnstakes");
                
                callback.onSuccess(balance);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get pending rewards
     */
    public void getRewards(String address, String chain, Callback<List<RewardInfo>> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/" + chain + "/rewards/" + address);
                JSONArray rewardsArray = response.getJSONArray("rewards");
                
                List<RewardInfo> rewards = new ArrayList<>();
                for (int i = 0; i < rewardsArray.length(); i++) {
                    JSONObject r = rewardsArray.getJSONObject(i);
                    RewardInfo reward = new RewardInfo();
                    reward.protocolId = r.getString("protocolId");
                    reward.protocolName = r.getString("protocolName");
                    reward.pendingRewards = r.getString("pendingRewards");
                    reward.pendingRewardsUSD = r.getDouble("pendingRewardsUSD");
                    rewards.add(reward);
                }
                
                callback.onSuccess(rewards);
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Claim rewards
     */
    public void claimRewards(String address, String chain, Callback<String> callback) {
        executor.execute(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("address", address);
                
                JSONObject response = makePostRequest("/" + chain + "/claim", body);
                callback.onSuccess(response.getString("transactionHash"));
            } catch (Exception e) {
                callback.onError(e);
            }
        });
    }
    
    /**
     * Get liquid token price
     */
    public void getLiquidTokenPrice(String protocolId, String chain, Callback<TokenPrice> callback) {
        executor.execute(() -> {
            try {
                JSONObject response = makeGetRequest("/" + chain + "/price/" + protocolId);
                
                TokenPrice price = new TokenPrice();
                price.price = response.getDouble("price");
                price.priceChange24h = response.getDouble("priceChange24h");
                price.totalSupply = response.getString("totalSupply");
                price.circulatingSupply = response.getString("circulatingSupply");
                
                callback.onSuccess(price);
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
    public static class LiquidStakingProtocol {
        public String id;
        public String name;
        public String chain;
        public String stakingToken;
        public String liquidToken;
        public String liquidTokenSymbol;
        public double totalStaked;
        public double apy;
        public String minStake;
        public int unbondingPeriod;
    }
    
    public static class ProtocolDetails {
        public String id;
        public String name;
        public String description;
        public String website;
        public JSONArray audits;
        public double TVL;
        public double apy;
        public double rewardsApy;
    }
    
    public static class StakeResult {
        public String transactionHash;
        public String stakedAmount;
        public String liquidTokenReceived;
        public String liquidTokenBalance;
    }
    
    public static class UnstakeResult {
        public String transactionHash;
        public String liquidTokenBurned;
        public String unbondingId;
        public long availableAt;
    }
    
    public static class StakedBalance {
        public String stakedNative;
        public String liquidTokenBalance;
        public double liquidTokenValueUSD;
        public double pendingUnstakes;
    }
    
    public static class RewardInfo {
        public String protocolId;
        public String protocolName;
        public String pendingRewards;
        public double pendingRewardsUSD;
    }
    
    public static class TokenPrice {
        public double price;
        public double priceChange24h;
        public String totalSupply;
        public String circulatingSupply;
    }
}
