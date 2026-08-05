package com.tigeradmin.data.api

import okhttp3.*
import okio.ByteString
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * Admin WebSocket Service
 * Handles real-time updates for admin dashboard
 */
class AdminWebSocketService(
    private val wsUrl: String,
    private val okHttpClient: OkHttpClient
) {
    
    private var webSocket: WebSocket? = null
    private var isConnected = false
    private val listeners = mutableListOf<WebSocketListener>()
    
    companion object {
        private const val RECONNECT_DELAY_SECONDS = 5
        private const val PING_INTERVAL_SECONDS = 30
    }

    /**
     * Connect to WebSocket server
     */
    fun connect(authToken: String) {
        if (isConnected) return
        
        val request = Request.Builder()
            .url("$wsUrl?token=$authToken")
            .build()
        
        webSocket = okHttpClient.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                isConnected = true
                notifyListeners { onConnect() }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                handleMessage(bytes.utf8())
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                webSocket.close(1000, null)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                isConnected = false
                notifyListeners { onDisconnect(reason) }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                isConnected = false
                notifyListeners { onError(t.message ?: "Unknown error") }
                // Attempt to reconnect
                scheduleReconnect(authToken)
            }
        })
    }

    /**
     * Disconnect from WebSocket server
     */
    fun disconnect() {
        webSocket?.close(1000, "User disconnected")
        webSocket = null
        isConnected = false
    }

    /**
     * Send message to server
     */
    fun send(message: String): Boolean {
        return webSocket?.send(message) ?: false
    }

    /**
     * Send JSON message to server
     */
    fun sendJson(data: JSONObject): Boolean {
        return send(data.toString())
    }

    /**
     * Subscribe to event
     */
    fun subscribe(event: String, data: JSONObject? = null): Boolean {
        val message = JSONObject().apply {
            put("action", "subscribe")
            put("event", event)
            data?.let { put("data", it) }
        }
        return sendJson(message)
    }

    /**
     * Unsubscribe from event
     */
    fun unsubscribe(event: String): Boolean {
        val message = JSONObject().apply {
            put("action", "unsubscribe")
            put("event", event)
        }
        return sendJson(message)
    }

    /**
     * Handle incoming message
     */
    private fun handleMessage(text: String) {
        try {
            val json = JSONObject(text)
            val event = json.optString("event", "")
            val data = json.optJSONObject("data")
            
            when (event) {
                "user.created" -> notifyListeners { onUserCreated(data) }
                "user.updated" -> notifyListeners { onUserUpdated(data) }
                "user.suspended" -> notifyListeners { onUserSuspended(data) }
                
                "transaction.new" -> notifyListeners { onNewTransaction(data) }
                "transaction.flagged" -> notifyListeners { onTransactionFlagged(data) }
                "transaction.confirmed" -> notifyListeners { onTransactionConfirmed(data) }
                
                "kyc.submitted" -> notifyListeners { onKYCSubmitted(data) }
                "kyc.approved" -> notifyListeners { onKYCApproved(data) }
                "kyc.rejected" -> notifyListeners { onKYCRejected(data) }
                
                "withdrawal.request" -> notifyListeners { onWithdrawalRequest(data) }
                "withdrawal.processed" -> notifyListeners { onWithdrawalProcessed(data) }
                
                "token.listing_request" -> notifyListeners { onTokenListingRequest(data) }
                
                "bot.started" -> notifyListeners { onBotStarted(data) }
                "bot.stopped" -> notifyListeners { onBotStopped(data) }
                "bot.error" -> notifyListeners { onBotError(data) }
                
                "system.alert" -> notifyListeners { onSystemAlert(data) }
                "system.maintenance" -> notifyListeners { onSystemMaintenance(data) }
                
                "ping" -> sendPong()
                
                else -> notifyListeners { onMessageReceived(event, data) }
            }
        } catch (e: Exception) {
            notifyListeners { onError("Failed to parse message: ${e.message}") }
        }
    }

    /**
     * Send pong response
     */
    private fun sendPong() {
        val message = JSONObject().apply {
            put("action", "pong")
        }
        sendJson(message)
    }

    /**
     * Schedule reconnection
     */
    private fun scheduleReconnect(authToken: String) {
        okHttpClient.dispatcher.executorService.schedule({
            connect(authToken)
        }, RECONNECT_DELAY_SECONDS.toLong(), TimeUnit.SECONDS)
    }

    /**
     * Add WebSocket listener
     */
    fun addListener(listener: WebSocketListener) {
        listeners.add(listener)
    }

    /**
     * Remove WebSocket listener
     */
    fun removeListener(listener: WebSocketListener) {
        listeners.remove(listener)
    }

    /**
     * Notify all listeners
     */
    private fun notifyListeners(action: WebSocketListener.() -> Unit) {
        listeners.forEach { it.action() }
    }

    /**
     * Check if connected
     */
    fun isConnected(): Boolean = isConnected

    /**
     * WebSocket Listener interface
     */
    interface WebSocketListener {
        fun onConnect() {}
        fun onDisconnect(reason: String) {}
        fun onError(error: String) {}
        fun onMessageReceived(event: String, data: JSONObject?) {}
        
        // User events
        fun onUserCreated(data: JSONObject?) {}
        fun onUserUpdated(data: JSONObject?) {}
        fun onUserSuspended(data: JSONObject?) {}
        
        // Transaction events
        fun onNewTransaction(data: JSONObject?) {}
        fun onTransactionFlagged(data: JSONObject?) {}
        fun onTransactionConfirmed(data: JSONObject?) {}
        
        // KYC events
        fun onKYCSubmitted(data: JSONObject?) {}
        fun onKYCApproved(data: JSONObject?) {}
        fun onKYCRejected(data: JSONObject?) {}
        
        // Withdrawal events
        fun onWithdrawalRequest(data: JSONObject?) {}
        fun onWithdrawalProcessed(data: JSONObject?) {}
        
        // Token events
        fun onTokenListingRequest(data: JSONObject?) {}
        
        // Bot events
        fun onBotStarted(data: JSONObject?) {}
        fun onBotStopped(data: JSONObject?) {}
        fun onBotError(data: JSONObject?) {}
        
        // System events
        fun onSystemAlert(data: JSONObject?) {}
        fun onSystemMaintenance(data: JSONObject?) {}
    }
}

/**
 * Simple WebSocket listener implementation
 */
abstract class SimpleWebSocketListener : AdminWebSocketService.WebSocketListener {
    override fun onConnect() {}
    override fun onDisconnect(reason: String) {}
    override fun onError(error: String) {}
    override fun onMessageReceived(event: String, data: JSONObject?) {}
}
