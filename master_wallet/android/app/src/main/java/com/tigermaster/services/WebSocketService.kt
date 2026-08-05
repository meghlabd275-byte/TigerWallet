/**
 * WebSocketService - Android Implementation
 * Real-time connection for Master Wallet
 */

package com.tigermaster.services

import android.util.Log
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import okhttp3.*
import org.json.JSONObject
import java.util.concurrent.TimeUnit
import javax.websocket.*
import kotlin.random.Random

class WebSocketService {
    private var webSocket: WebSocket? = null
    private val client = OkHttpClient.Builder()
        .readTimeout(30, TimeUnit.SECONDS)
        .pingInterval(15, TimeUnit.SECONDS)
        .build()
    
    private val _connectionState = MutableStateFlow(ConnectionState.DISCONNECTED)
    val connectionState: StateFlow<ConnectionState> = _connectionState
    
    private val _messages = MutableSharedFlow<WSMessage>(replay = 0)
    val messages: SharedFlow<WSMessage> = _messages
    
    private val _balanceUpdates = MutableSharedFlow<BalanceUpdate>(replay = 1)
    val balanceUpdates: SharedFlow<BalanceUpdate> = _balanceUpdates
    
    private val _transactionUpdates = MutableSharedFlow<TransactionUpdate>(replay = 1)
    val transactionUpdates: SharedFlow<TransactionUpdate> = _transactionUpdates
    
    private var reconnectJob: Job? = null
    private var heartbeatJob: Job? = null
    
    private var walletId: String? = null
    private var authToken: String? = null
    
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    
    companion object {
        private const val TAG = "WebSocketService"
        private const val WS_URL = "wss://master-ws.tigerwallet.com/ws"
        private const val RECONNECT_DELAY = 5000L
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }
    
    enum class ConnectionState {
        DISCONNECTED,
        CONNECTING,
        CONNECTED,
        RECONNECTING,
        ERROR
    }
    
    /**
     * Connect to WebSocket server
     */
    fun connect(walletId: String, token: String?) {
        this.walletId = walletId
        this.authToken = token
        _connectionState.value = ConnectionState.CONNECTING
        
        val request = Request.Builder()
            .url(WS_URL)
            .build()
        
        webSocket = client.newWebSocket(request, createListener())
    }
    
    /**
     * Disconnect from server
     */
    fun disconnect() {
        reconnectJob?.cancel()
        heartbeatJob?.cancel()
        webSocket?.close(1000, "Client disconnected")
        webSocket = null
        _connectionState.value = ConnectionState.DISCONNECTED
    }
    
    /**
     * Subscribe to balance updates for specific chain
     */
    fun subscribeToBalance(chainId: Int) {
        sendMessage(
            type = "subscribe",
            channel = "balance",
            data = mapOf("chainId" to chainId)
        )
    }
    
    /**
     * Unsubscribe from balance updates
     */
    fun unsubscribeFromBalance(chainId: Int) {
        sendMessage(
            type = "unsubscribe",
            channel = "balance",
            data = mapOf("chainId" to chainId)
        )
    }
    
    /**
     * Subscribe to transaction updates
     */
    fun subscribeToTransactions(address: String) {
        sendMessage(
            type = "subscribe",
            channel = "transactions",
            data = mapOf("address" to address)
        )
    }
    
    /**
     * Subscribe to market ticker updates
     */
    fun subscribeToTicker(pair: String) {
        sendMessage(
            type = "subscribe",
            channel = "ticker",
            data = mapOf("pair" to pair)
        )
    }
    
    /**
     * Subscribe to order book updates
     */
    fun subscribeToOrderBook(pair: String) {
        sendMessage(
            type = "subscribe",
            channel = "orderbook",
            data = mapOf("pair" to pair)
        )
    }
    
    /**
     * Send authenticated message
     */
    fun authenticate() {
        sendMessage(
            type = "auth",
            channel = "auth",
            data = mapOf(
                "walletId" to (walletId ?: ""),
                "token" to (authToken ?: "")
            )
        )
    }
    
    private fun sendMessage(type: String, channel: String, data: Map<String, Any>) {
        val message = JSONObject().apply {
            put("type", type)
            put("channel", channel)
            put("data", JSONObject(data))
            put("timestamp", System.currentTimeMillis())
        }
        
        webSocket?.send(message.toString())
    }
    
    private fun createListener(): WebSocketListener = object : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            Log.d(TAG, "WebSocket connected")
            _connectionState.value = ConnectionState.CONNECTED
            authenticate()
            startHeartbeat()
        }
        
        override fun onMessage(webSocket: WebSocket, text: String) {
            try {
                val json = JSONObject(text)
                val type = json.optString("type")
                val channel = json.optString("channel")
                val data = json.optJSONObject("data")
                
                when (channel) {
                    "balance" -> handleBalanceUpdate(data)
                    "transactions" -> handleTransactionUpdate(data)
                    "ticker" -> handleTickerUpdate(data)
                    "orderbook" -> handleOrderBookUpdate(data)
                    "auth" -> handleAuthResponse(data)
                }
                
                scope.launch {
                    _messages.emit(WSMessage(type, channel, data?.toString() ?: ""))
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error parsing message: ${e.message}")
            }
        }
        
        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
            Log.d(TAG, "WebSocket closing: $code $reason")
            webSocket.close(1000, null)
        }
        
        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            Log.d(TAG, "WebSocket closed: $code $reason")
            _connectionState.value = ConnectionState.DISCONNECTED
            stopHeartbeat()
            handleReconnect()
        }
        
        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            Log.e(TAG, "WebSocket error: ${t.message}")
            _connectionState.value = ConnectionState.ERROR
            stopHeartbeat()
            handleReconnect()
        }
    }
    
    private fun handleBalanceUpdate(data: JSONObject?) {
        data?.let {
            scope.launch {
                _balanceUpdates.emit(BalanceUpdate(
                    chainId = it.optInt("chainId"),
                    address = it.optString("address"),
                    balance = it.optString("balance"),
                    token = it.optString("token", "ETH"),
                    timestamp = it.optLong("timestamp")
                ))
            }
        }
    }
    
    private fun handleTransactionUpdate(data: JSONObject?) {
        data?.let {
            scope.launch {
                _transactionUpdates.emit(TransactionUpdate(
                    txHash = it.optString("txHash"),
                    from = it.optString("from"),
                    to = it.optString("to"),
                    amount = it.optString("amount"),
                    status = it.optString("status"),
                    timestamp = it.optLong("timestamp")
                ))
            }
        }
    }
    
    private fun handleTickerUpdate(data: JSONObject?) {
        // Handle ticker updates
    }
    
    private fun handleOrderBookUpdate(data: JSONObject?) {
        // Handle order book updates
    }
    
    private fun handleAuthResponse(data: JSONObject?) {
        val success = data?.optBoolean("success") ?: false
        if (success) {
            Log.d(TAG, "Authentication successful")
        } else {
            Log.e(TAG, "Authentication failed: ${data?.optString("error")}")
        }
    }
    
    private fun startHeartbeat() {
        heartbeatJob = scope.launch {
            while (isActive) {
                delay(15000)
                sendMessage(
                    type = "ping",
                    channel = "heartbeat",
                    data = emptyMap()
                )
            }
        }
    }
    
    private fun stopHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = null
    }
    
    private fun handleReconnect() {
        reconnectJob?.cancel()
        reconnectJob = scope.launch {
            var attempts = 0
            while (attempts < MAX_RECONNECT_ATTEMPTS && isActive) {
                attempts++
                _connectionState.value = ConnectionState.RECONNECTING
                delay(RECONNECT_DELAY * attempts)
                
                try {
                    walletId?.let { id ->
                        connect(id, authToken)
                    }
                    if (_connectionState.value == ConnectionState.CONNECTED) {
                        break
                    }
                } catch (e: Exception) {
                    Log.e(TAG, "Reconnect attempt $attempts failed: ${e.message}")
                }
            }
            
            if (_connectionState.value != ConnectionState.CONNECTED) {
                _connectionState.value = ConnectionState.ERROR
            }
        }
    }
}

// Data classes

data class WSMessage(
    val type: String,
    val channel: String,
    val data: String
)

data class BalanceUpdate(
    val chainId: Int,
    val address: String,
    val balance: String,
    val token: String,
    val timestamp: Long
)

data class TransactionUpdate(
    val txHash: String,
    val from: String,
    val to: String,
    val amount: String,
    val status: String,
    val timestamp: Long
)
