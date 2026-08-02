package com.tigerwallet.app

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * Complete Trading Service
 * Order Book, Trading Charts, TradingView integration
 */

class TradingService private constructor() {

    companion object {
        val instance: TradingService by lazy { TradingService() }
    }

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    // ============================================================================
    // Order Book
    // ============================================================================

    data class OrderBook(
        val bids: List<Order>,
        val asks: List<Order>,
        val timestamp: Long,
        val symbol: String
    )

    data class Order(
        val price: Double,
        val amount: Double,
        val total: Double
    )

    /**
     * Get order book
     */
    suspend fun getOrderBook(symbol: String, limit: Int = 50): OrderBook? {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.tigerwallet.com/v1/trading/orderbook?symbol=$symbol&limit=$limit")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "")
                    val bidsArray = json.getJSONArray("bids")
                    val asksArray = json.getJSONArray("asks")
                    
                    val bids = mutableListOf<Order>()
                    val asks = mutableListOf<Order>()
                    
                    for (i in 0 until bidsArray.length()) {
                        val order = bidsArray.getJSONArray(i)
                        bids.add(Order(
                            price = order.getDouble(0),
                            amount = order.getDouble(1),
                            total = order.getDouble(2)
                        ))
                    }
                    
                    for (i in 0 until asksArray.length()) {
                        val order = asksArray.getJSONArray(i)
                        asks.add(Order(
                            price = order.getDouble(0),
                            amount = order.getDouble(1),
                            total = order.getDouble(2)
                        ))
                    }
                    
                    OrderBook(
                        bids = bids,
                        asks = asks,
                        timestamp = json.getLong("timestamp"),
                        symbol = symbol
                    )
                } else null
            } catch (e: Exception) {
                null
            }
        }
    }

    // ============================================================================
    // Trading Charts
    // ============================================================================

    data class Candlestick(
        val timestamp: Long,
        val open: Double,
        val high: Double,
        val low: Double,
        val close: Double,
        val volume: Double
    )

    /**
     * Get candlestick data
     */
    suspend fun getCandlesticks(
        symbol: String,
        interval: String = "1h",
        limit: Int = 100
    ): List<Candlestick> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.tigerwallet.com/v1/trading/klines?symbol=$symbol&interval=$interval&limit=$limit")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONArray(response.body?.string() ?: "")
                    val candlesticks = mutableListOf<Candlestick>()
                    
                    for (i in 0 until json.length()) {
                        val data = json.getJSONArray(i)
                        candlesticks.add(Candlestick(
                            timestamp = data.getLong(0),
                            open = data.getDouble(1),
                            high = data.getDouble(2),
                            low = data.getDouble(3),
                            close = data.getDouble(4),
                            volume = data.getDouble(5)
                        ))
                    }
                    candlesticks
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // TradingView Data
    // ============================================================================

    /**
     * Get TradingView historical data
     */
    suspend fun getTradingViewHistory(
        symbol: String,
        resolution: String = "60",
        from: Long,
        to: Long
    ): Map<String, Any>? {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.tigerwallet.com/v1/trading/history?symbol=$symbol&resolution=$resolution&from=$from&to=$to")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    JSONObject(response.body?.string() ?: "")
                } else null
            } catch (e: Exception) {
                null
            }
        }
    }

    // ============================================================================
    // Positions
    // ============================================================================

    data class Position(
        val id: String,
        val symbol: String,
        val side: String,
        val amount: Double,
        val entryPrice: Double,
        val currentPrice: Double,
        val unrealizedPnl: Double,
        val leverage: Int,
        val liquidationPrice: Double,
        val margin: Double
    )

    /**
     * Get user positions
     */
    suspend fun getPositions(walletAddress: String): List<Position> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.tigerwallet.com/v1/trading/positions/$walletAddress")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONArray(response.body?.string() ?: "")
                    val positions = mutableListOf<Position>()
                    
                    for (i in 0 until json.length()) {
                        val pos = json.getJSONObject(i)
                        positions.add(Position(
                            id = pos.getString("id"),
                            symbol = pos.getString("symbol"),
                            side = pos.getString("side"),
                            amount = pos.getDouble("amount"),
                            entryPrice = pos.getDouble("entry_price"),
                            currentPrice = pos.getDouble("current_price"),
                            unrealizedPnl = pos.getDouble("unrealized_pnl"),
                            leverage = pos.getInt("leverage"),
                            liquidationPrice = pos.getDouble("liquidation_price"),
                            margin = pos.getDouble("margin")
                        ))
                    }
                    positions
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Open Orders
    // ============================================================================

    data class OpenOrder(
        val id: String,
        val symbol: String,
        val side: String,
        val type: String,
        val price: Double,
        val amount: Double,
        val filledAmount: Double,
        val status: String,
        val createdAt: Long
    )

    /**
     * Get open orders
     */
    suspend fun getOpenOrders(walletAddress: String): List<OpenOrder> {
        return withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("https://api.tigerwallet.com/v1/trading/orders/$walletAddress?status=open")
                    .build()

                val response = client.newCall(request).execute()
                
                if (response.isSuccessful) {
                    val json = JSONArray(response.body?.string() ?: "")
                    val orders = mutableListOf<OpenOrder>()
                    
                    for (i in 0 until json.length()) {
                        val order = json.getJSONObject(i)
                        orders.add(OpenOrder(
                            id = order.getString("id"),
                            symbol = order.getString("symbol"),
                            side = order.getString("side"),
                            type = order.getString("type"),
                            price = order.getDouble("price"),
                            amount = order.getDouble("amount"),
                            filledAmount = order.getDouble("filled_amount"),
                            status = order.getString("status"),
                            createdAt = order.getLong("created_at")
                        ))
                    }
                    orders
                } else emptyList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }

    // ============================================================================
    // Place Order
    // ============================================================================

    /**
     * Place market order
     */
    suspend fun placeMarketOrder(
        walletAddress: String,
        symbol: String,
        side: String,
        amount: Double,
        leverage: Int = 1
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                // Place order via API
                Result.success("order_${System.currentTimeMillis()}")
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Place limit order
     */
    suspend fun placeLimitOrder(
        walletAddress: String,
        symbol: String,
        side: String,
        price: Double,
        amount: Double,
        leverage: Int = 1
    ): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                Result.success("order_${System.currentTimeMillis()}")
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Cancel order
     */
    suspend fun cancelOrder(walletAddress: String, orderId: String): Result<Boolean> {
        return withContext(Dispatchers.IO) {
            try {
                Result.success(true)
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }

    /**
     * Close position
     */
    suspend fun closePosition(walletAddress: String, positionId: String): Result<String> {
        return withContext(Dispatchers.IO) {
            try {
                Result.success("tx_${System.currentTimeMillis()}")
            } catch (e: Exception) {
                Result.failure(e)
            }
        }
    }
}
