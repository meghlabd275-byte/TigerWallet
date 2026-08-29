package com.tigeruserwallet.util

import com.tigeruserwallet.api.UserWalletApiService
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONArray
import org.json.JSONObject

/**
 * UserWallet live price-feed socket (Android).
 *
 * Connects to the canonical wallet_api public live feed:
 *   ws(s)://<host>/api/v1/ws
 *
 * Protocol: client sends { action: "subscribe", symbols: ["BTC", ...] } and
 * the server pushes { type: "ticker", symbol, last_price, volume_24h,
 * market_cap, change_24h_pct } frames sourced from the live price oracle.
 * This helper only transports REAL frames — it never fabricates tickers.
 *
 * Usage:
 *   val feed = LiveFeedSocket()
 *   feed.connect(listOf("BTC", "ETH"), onTicker = { t -> ... })
 *   feed.close()
 */
class LiveFeedSocket(
    private val client: OkHttpClient = OkHttpClient(),
) {
    private var webSocket: WebSocket? = null

    fun connect(
        symbols: List<String>,
        onTicker: (JSONObject) -> Unit,
        onOpen: (() -> Unit)? = null,
        onFailure: ((Throwable) -> Unit)? = null,
        onClosed: (() -> Unit)? = null,
    ) {
        val request = Request.Builder().url(wsUrl()).build()
        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(ws: WebSocket, response: Response) {
                val sub = JSONObject()
                    .put("action", "subscribe")
                    .put("symbols", JSONArray(symbols))
                ws.send(sub.toString())
                onOpen?.invoke()
            }

            override fun onMessage(ws: WebSocket, text: String) {
                try {
                    val frame = JSONObject(text)
                    if (frame.optString("type") == "ticker") onTicker(frame)
                } catch (_: Exception) {
                    // Non-JSON frames are ignored.
                }
            }

            override fun onFailure(ws: WebSocket, t: Throwable, response: Response?) {
                onFailure?.invoke(t)
            }

            override fun onClosed(ws: WebSocket, code: Int, reason: String) {
                onClosed?.invoke()
            }
        })
    }

    fun close() {
        webSocket?.close(1000, "client closing")
        webSocket = null
    }

    companion object {
        /** Derives the WS URL from the configured wallet_api base URL. */
        fun wsUrl(): String {
            val httpBase = UserWalletApiService.apiBaseUrl().trimEnd('/')
            val wsBase = if (httpBase.startsWith("https")) {
                "wss" + httpBase.removePrefix("https")
            } else {
                "ws" + httpBase.removePrefix("http")
            }
            return "$wsBase/ws"
        }
    }
}
