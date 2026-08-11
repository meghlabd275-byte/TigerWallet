package com.tigerwallet.app

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.math.BigInteger
import java.util.concurrent.TimeUnit

/**
 * NFT Marketplace Service
 * Buy, sell, create listings, and trade NFTs
 *
 * Fail-closed: every write operation (`createListing`, `cancelListing`,
 * `buyNFT`, `makeOffer`, `acceptOffer`) that returns a tx hash or order id
 * either delegates to the REAL backend (POST /api/v1/send for on-chain
 * actions) or throws. No `"0x"+UUID` tx hash and no `"listing_" + millis`
 * / `"offer_" + millis` id is ever fabricated. If the backend is unreachable
 * or rejects the request, the call fails closed.
 */

class NFTMarketplaceService private constructor() {

    companion object {
        val instance: NFTMarketplaceService by lazy { NFTMarketplaceService() }

        const val BACKEND_BASE_URL = "http://localhost:8443"
        private val JSON_MEDIA_TYPE = "application/json".toMediaType()
    }

    /**
     * JWT auth token supplied by the host app. Required for authenticated
     * backend writes (POST /api/v1/send). When empty, authenticated writes
     * throw fail-closed.
     */
    @Volatile
    var authToken: String = ""

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    /**
     * POST a JSON payload to a real backend endpoint and return the real
     * `tx_hash` reported by the backend. Throws fail-closed if the backend is
     * unreachable, rejects the request, or returns no valid tx hash.
     */
    private fun postBackendTx(path: String, payload: JSONObject): String {
        if (authToken.isEmpty()) {
            throw IllegalStateException("No auth token configured for backend write.")
        }
        val builder = Request.Builder()
            .url("$BACKEND_BASE_URL$path")
            .header("Content-Type", "application/json")
            .header("Authorization", "Bearer $authToken")
            .post(payload.toString().toRequestBody(JSON_MEDIA_TYPE))
        val body: String
        val code: Int
        try {
            client.newCall(builder.build()).execute().use { resp ->
                code = resp.code
                body = resp.body?.string() ?: ""
            }
        } catch (e: Exception) {
            throw IllegalStateException("Backend unreachable: ${e.message}", e)
        }
        if (code !in 200..299) {
            throw IllegalStateException("Backend rejected request (HTTP $code): $body")
        }
        val json = JSONObject(body)
        val txHash = json.optString("tx_hash", json.optString("txHash", ""))
        if (txHash.isEmpty() || !txHash.startsWith("0x")) {
            throw IllegalStateException("Backend returned no valid tx_hash: $body")
        }
        return txHash
    }

    // Supported marketplaces
    enum class Marketplace(val displayName: String, val chainId: Int) {
        OPENSEA("OpenSea", 1),
        LOOKSRARE("LooksRare", 1),
        X2Y2("X2Y2", 1),
        BLUR("Blur", 1),
        SOLANART("Solanart", 501),
        MAGIC_EDEN("Magic Eden", 501),
    }

    // NFT Collection data
    data class NFTCollection(
        val address: String,
        val name: String,
        val symbol: String,
        val description: String,
        val imageUrl: String,
        val bannerUrl: String,
        val floorPrice: Double,
        val totalSupply: Int,
        val owners: Int,
        val volume24h: Double,
        val chain: String
    )

    // NFT Listing data
    data class NFTListing(
        val id: String,
        val tokenId: String,
        val collectionAddress: String,
        val seller: String,
        val price: Double,
        val priceToken: String,
        val expirationTime: Long,
        val chain: String
    )

    // NFT Order/Trade data
    data class NFTTrade(
        val id: String,
        val tokenId: String,
        val collectionAddress: String,
        val buyer: String,
        val seller: String,
        val price: Double,
        val timestamp: Long,
        val txHash: String,
        val chain: String
    )

    // ============================================================================
    // Collection Methods
    // ============================================================================

    /**
     * Get collection by contract address
     */
    suspend fun getCollection(address: String, chain: String = "ethereum"): NFTCollection? {
        return withContext(Dispatchers.IO) {
            try {
                // Try OpenSea API first
                val request = Request.Builder()
                    .url("https://api.opensea.io/api/v1/collection/$address")
                    .header("Accept", "application/json")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "")
                    val collection = json.getJSONObject("collection")
                    
                    NFTCollection(
                        address = address,
                        name = collection.optString("name", ""),
                        symbol = collection.optString("symbol", ""),
                        description = collection.optString("description", ""),
                        imageUrl = collection.optJSONObject("image_url")?.optString("original") ?: "",
                        bannerUrl = collection.optJSONObject("banner_image_url")?.optString("original") ?: "",
                        floorPrice = collection.optJSONObject("stats")?.optDouble("floor_price", 0.0) ?: 0.0,
                        totalSupply = collection.optJSONObject("stats")?.optInt("total_supply", 0) ?: 0,
                        owners = collection.optJSONObject("stats")?.optInt("num_owners", 0) ?: 0,
                        volume24h = collection.optJSONObject("stats")?.optDouble("one_day_volume", 0.0) ?: 0.0,
                        chain = chain
                    )
                } else null
            } catch (e: Exception) {
                null
            }
        }
    }

    /**
     * Search collections
     */
    suspend fun searchCollections(
        query: String,
        chain: String = "ethereum",
        limit: Int = 20
    ): List<NFTCollection> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.opensea.io/api/v1/collections?search=$query&limit=$limit")
                    .header("Accept", "application/json")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONArray(response.body?.string() ?: "")
                    val collections = mutableListOf<NFTCollection>()
                    
                    for (i in 0 until json.length()) {
                        val collection = json.getJSONObject(i)
                        collections.add(
                            NFTCollection(
                                address = collection.optString("contract_addr", ""),
                                name = collection.optString("name", ""),
                                symbol = collection.optString("symbol", ""),
                                description = collection.optString("description", ""),
                                imageUrl = collection.optJSONObject("image_url")?.optString("original") ?: "",
                                bannerUrl = collection.optJSONObject("banner_image_url")?.optString("original") ?: "",
                                floorPrice = collection.optJSONObject("stats")?.optDouble("floor_price", 0.0) ?: 0.0,
                                totalSupply = collection.optJSONObject("stats")?.optInt("total_supply", 0) ?: 0,
                                owners = collection.optJSONObject("stats")?.optInt("num_owners", 0) ?: 0,
                                volume24h = collection.optJSONObject("stats")?.optDouble("one_day_volume", 0.0) ?: 0.0,
                                chain = chain
                            )
                        )
                    }
                    collections
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Listing Methods
    // ============================================================================

    /**
     * Get listings for a collection
     */
    suspend fun getListings(
        collectionAddress: String,
        chain: String = "ethereum",
        limit: Int = 50
    ): List<NFTListing> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.opensea.io/api/v1/assets?collection=$collectionAddress&limit=$limit&order_direction=desc&include_bundled=false")
                    .header("Accept", "application/json")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "")
                    val assets = json.getJSONArray("assets")
                    val listings = mutableListOf<NFTListing>()
                    
                    for (i in 0 until assets.length()) {
                        val asset = assets.getJSONObject(i)
                        val lastSale = asset.optJSONObject("last_sale")
                        
                        if (lastSale != null) {
                            val paymentToken = lastSale.optJSONObject("payment_token")
                            listings.add(
                                NFTListing(
                                    id = asset.optString("id", ""),
                                    tokenId = asset.optString("token_id", ""),
                                    collectionAddress = collectionAddress,
                                    seller = asset.optString("owner", ""),
                                    price = lastSale.optDouble("total_price", 0.0) / 1e18,
                                    priceToken = paymentToken?.optString("symbol", "ETH") ?: "ETH",
                                    expirationTime = 0,
                                    chain = chain
                                )
                            )
                        }
                    }
                    listings
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    /**
     * Get user's listings
     */
    suspend fun getUserListings(
        ownerAddress: String,
        chain: String = "ethereum"
    ): List<NFTListing> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.opensea.io/api/v1/assets?owner=$ownerAddress&limit=50")
                    .header("Accept", "application/json")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "")
                    val assets = json.getJSONArray("assets")
                    val listings = mutableListOf<NFTListing>()
                    
                    for (i in 0 until assets.length()) {
                        val asset = assets.getJSONObject(i)
                        val lastSale = asset.optJSONObject("last_sale")
                        
                        listings.add(
                            NFTListing(
                                id = asset.optString("id", ""),
                                tokenId = asset.optString("token_id", ""),
                                collectionAddress = asset.optJSONObject("asset_contract")?.optString("address", "") ?: "",
                                seller = ownerAddress,
                                price = lastSale?.optDouble("total_price", 0.0)?.div(1e18) ?: 0.0,
                                priceToken = lastSale?.optJSONObject("payment_token")?.optString("symbol", "ETH") ?: "ETH",
                                expirationTime = 0,
                                chain = chain
                            )
                        )
                    }
                    listings
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Trading Methods
    // ============================================================================

    /**
     * Create a listing (sell NFT). Submits the listing/approval transaction to
     * the REAL backend (POST /api/v1/send) and returns the REAL on-chain
     * tx_hash. No `"listing_" + millis` id is fabricated. Throws fail-closed
     * if the backend is unreachable or rejects the request.
     */
    suspend fun createListing(
        walletAddress: String,
        collectionAddress: String,
        tokenId: String,
        price: Double,
        priceToken: String = "ETH",
        chain: String = "ethereum"
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", walletAddress)
                    put("chain", chain)
                    put("action", "nft_list")
                    put("collection_address", collectionAddress)
                    put("token_id", tokenId)
                    put("price", price)
                    put("price_token", priceToken)
                }
                Result.success(postBackendTx("/api/v1/send", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Cancel a listing. Submits the cancellation transaction to the REAL
     * backend (POST /api/v1/send). No fake success is returned. Throws
     * fail-closed if the backend is unreachable or rejects the request.
     */
    suspend fun cancelListing(
        walletAddress: String,
        listingId: String,
        chain: String = "ethereum"
    ): Result<Boolean> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", walletAddress)
                    put("chain", chain)
                    put("action", "nft_cancel_listing")
                    put("listing_id", listingId)
                }
                postBackendTx("/api/v1/send", payload)
                Result.success(true)
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Buy NFT (fulfill listing). Submits the purchase transaction to the REAL
     * backend (POST /api/v1/send) and returns the REAL on-chain tx_hash. No
     * `"0x"+UUID` tx hash is fabricated. Throws fail-closed if the backend is
     * unreachable or rejects the request.
     */
    suspend fun buyNFT(
        buyerAddress: String,
        listingId: String,
        chain: String = "ethereum"
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", buyerAddress)
                    put("chain", chain)
                    put("action", "nft_buy")
                    put("listing_id", listingId)
                }
                Result.success(postBackendTx("/api/v1/send", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    // ============================================================================
    // Offer Methods
    // ============================================================================

    /**
     * Make an offer on an NFT. Submits the offer transaction to the REAL
     * backend (POST /api/v1/send) and returns the REAL on-chain tx_hash. No
     * `"offer_" + millis` id is fabricated. Throws fail-closed if the backend
     * is unreachable or rejects the request.
     */
    suspend fun makeOffer(
        makerAddress: String,
        collectionAddress: String,
        tokenId: String,
        price: Double,
        priceToken: String = "ETH",
        expirationTime: Long,
        chain: String = "ethereum"
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", makerAddress)
                    put("chain", chain)
                    put("action", "nft_make_offer")
                    put("collection_address", collectionAddress)
                    put("token_id", tokenId)
                    put("price", price)
                    put("price_token", priceToken)
                    put("expiration_time", expirationTime)
                }
                Result.success(postBackendTx("/api/v1/send", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Accept an offer. Submits the acceptance transaction to the REAL backend
     * (POST /api/v1/send) and returns the REAL on-chain tx_hash. No
     * `"0x"+UUID` tx hash is fabricated. Throws fail-closed if the backend is
     * unreachable or rejects the request.
     */
    suspend fun acceptOffer(
        sellerAddress: String,
        offerId: String,
        chain: String = "ethereum"
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("wallet_address", sellerAddress)
                    put("chain", chain)
                    put("action", "nft_accept_offer")
                    put("offer_id", offerId)
                }
                Result.success(postBackendTx("/api/v1/send", payload))
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    // ============================================================================
    // Trade History
    // ============================================================================

    /**
     * Get trade history for a collection
     */
    suspend fun getCollectionTrades(
        collectionAddress: String,
        chain: String = "ethereum",
        limit: Int = 50
    ): List<NFTTrade> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.opensea.io/api/v1/events?collection=$collectionAddress&event_type=successful&limit=$limit")
                    .header("Accept", "application/json")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "")
                    val events = json.getJSONArray("asset_events")
                    val trades = mutableListOf<NFTTrade>()
                    
                    for (i in 0 until events.length()) {
                        val event = events.getJSONObject(i)
                        val asset = event.optJSONObject("asset")
                        val paymentToken = event.optJSONObject("payment_token")
                        val seller = event.optJSONObject("seller_account")
                        val buyer = event.optJSONObject("winner_account")
                        
                        trades.add(
                            NFTTrade(
                                id = event.optString("id", ""),
                                tokenId = asset?.optString("token_id", "") ?: "",
                                collectionAddress = collectionAddress,
                                buyer = buyer?.optString("address", "") ?: "",
                                seller = seller?.optString("address", "") ?: "",
                                price = event.optDouble("total_price", 0.0) / 1e18,
                                timestamp = event.optLong("created_date", 0) * 1000,
                                txHash = event.optString("transaction_hash", ""),
                                chain = chain
                            )
                        )
                    }
                    trades
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Royalty Calculation
    // ============================================================================

    /**
     * Calculate royalties for a sale
     */
    fun calculateRoyalties(
        salePrice: Double,
        royaltyBps: Int // Basis points (e.g., 250 = 2.5%)
    ): Double {
        return salePrice * (royaltyBps / 10000.0)
    }

    /**
     * Calculate marketplace fee
     */
    fun calculateMarketplaceFee(
        salePrice: Double,
        marketplace: Marketplace
    ): Double {
        val feeBps = when (marketplace) {
            Marketplace.OPENSEA -> 250 // 2.5%
            Marketplace.LOOKSRARE -> 200 // 2%
            Marketplace.X2Y2 -> 200 // 2%
            Marketplace.BLUR -> 0 // 0% for BLUR
            Marketplace.SOLANART -> 300 // 3%
            Marketplace.MAGIC_EDEN -> 300 // 3%
        }
        return salePrice * (feeBps / 10000.0)
    }
}

// Extension function for formatting
fun Double.formatNFTPrice(decimals: Int = 4): String {
    return String.format("%.${decimals}f", this)
}
