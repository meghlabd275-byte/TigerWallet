package com.tigeruserwallet.util

import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.ByteString
import org.json.JSONObject
import java.net.URLEncoder
import java.util.concurrent.atomic.AtomicLong

/**
 * UserWallet WalletConnect live-event socket (Android).
 *
 * Connects to the canonical dapp_browser WalletConnect relay through the
 * wallet_api reverse proxy:  ws(s)://<host>/api/v1/dapp/ws/<topic>
 *
 * The wire protocol is JSON-RPC-style frames: { id, method, params }.
 * Server-pushed events arrive with a `method` field; client requests elicit
 * responses keyed by `id`. This helper only transports REAL frames — it never
 * fabricates events.
 *
 * Usage:
 *   val socket = WalletConnectSocket()
 *   socket.connect(topic,
 *       onMessage = { frame -> /* JSONObject frame */ },
 *       onFailure = { err -> /* Throwable */ })
 *   socket.sendRequest("wc_sessionRequest", params)
 *   socket.close()
 */
class WalletConnectSocket(
    private val client: OkHttpClient = OkHttpClient(),
) {
    private var webSocket: WebSocket? = null
    private val nextId = AtomicLong(1L)

    fun connect(
        topic: String,
        onMessage: (JSONObject) -> Unit,
        onOpen: (() -> Unit)? = null,
        onFailure: ((Throwable) -> Unit)? = null,
        onClosed: (() -> Unit)? = null,
    ) {
        val encoded = URLEncoder.encode(topic, "UTF-8")
        val url = "${wsBase()}/$encoded"
        val request = Request.Builder().url(url).build()
        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(ws: WebSocket, response: Response) {
                onOpen?.invoke()
            }

            override fun onMessage(ws: WebSocket, text: String) {
                try {
                    onMessage(JSONObject(text))
                } catch (_: Exception) {
                    // Non-JSON frames are ignored (server never sends them today).
                }
            }

            override fun onMessage(ws: WebSocket, bytes: ByteString) {
                onMessage(ws, bytes.utf8())
            }

            override fun onFailure(ws: WebSocket, t: Throwable, response: Response?) {
                onFailure?.invoke(t)
            }

            override fun onClosed(ws: WebSocket, code: Int, reason: String) {
                onClosed?.invoke()
            }
        })
    }

    /** Send a JSON-RPC request frame. Returns the frame id. */
    fun sendRequest(method: String, params: JSONObject? = null): Long {
        val id = nextId.getAndIncrement()
        val frame = JSONObject()
            .put("id", id)
            .put("method", method)
        if (params != null) frame.put("params", params)
        webSocket?.send(frame.toString())
            ?: throw IllegalStateException("WalletConnectSocket is not connected")
        return id
    }

    fun close(code: Int = 1000, reason: String = "closing") {
        webSocket?.close(code, reason)
        webSocket = null
    }

    companion object {
        /** HTTP API base; mirrors UserWalletApiService.DEFAULT_BASE_URL. */
        private const val API_BASE_URL = "http://localhost:8443/api/v1"

        fun wsBase(): String {
            val httpBase = API_BASE_URL.trimEnd('/')
            val wsBase = if (httpBase.startsWith("https")) {
                "wss" + httpBase.removePrefix("https")
            } else {
                "ws" + httpBase.removePrefix("http")
            }
            return "$wsBase/dapp/ws"
        }
    }
}
