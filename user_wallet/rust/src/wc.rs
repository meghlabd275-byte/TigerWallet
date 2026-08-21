//! Real WalletConnect live-event socket (Rust).
//!
//! Connects to the canonical dapp_browser WalletConnect relay through the
//! wallet_api reverse proxy:  ws(s)://<host>/api/v1/dapp/ws/<topic>
//!
//! The wire protocol is JSON-RPC-style frames: `{ id, method, params }`.
//! Server-pushed events arrive with a `method` field; client requests elicit
//! responses keyed by `id`. This module only transports REAL frames — it never
//! fabricates events.

use futures_util::{SinkExt, StreamExt};
use tokio_tungstenite::tungstenite::Message;

use crate::WalletError;

/// A live WalletConnect socket for one pairing topic.
pub struct WalletConnectSocket {
    stream: tokio_tungstenite::WebSocketStream<
        tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
    >,
    next_id: u64,
}

impl WalletConnectSocket {
    /// Derive the relay WebSocket URL from the HTTP API base URL. Accepts
    /// either ".../api/v1" or the bare host (the /api/v1 prefix is added,
    /// mirroring the HTTP client's path normalization).
    /// e.g. `http://localhost:8443` -> `ws://localhost:8443/api/v1/dapp/ws/<topic>`
    fn topic_url(api_base: &str, topic: &str) -> String {
        let trimmed = api_base.trim_end_matches('/');
        let http_base = if trimmed.ends_with("/api/v1") {
            trimmed.to_string()
        } else {
            format!("{}/api/v1", trimmed)
        };
        let ws_base = if let Some(rest) = http_base.strip_prefix("https") {
            format!("wss{}", rest)
        } else {
            format!("ws{}", http_base.strip_prefix("http").unwrap_or(&http_base))
        };
        format!("{}/dapp/ws/{}", ws_base, crate::url_encode(topic))
    }

    /// Open a live WalletConnect socket for a pairing topic.
    pub async fn connect(api_base: &str, topic: &str) -> Result<Self, WalletError> {
        let url = Self::topic_url(api_base, topic);
        let (stream, _resp) = tokio_tungstenite::connect_async(&url)
            .await
            .map_err(|e| WalletError::Http(format!("walletconnect connect: {e}")))?;
        Ok(Self { stream, next_id: 1 })
    }

    /// Send a JSON-RPC request frame. Returns the frame id.
    pub async fn send_request(
        &mut self,
        method: &str,
        params: Option<serde_json::Value>,
    ) -> Result<u64, WalletError> {
        let id = self.next_id;
        self.next_id += 1;
        let mut frame = serde_json::json!({ "id": id, "method": method });
        if let Some(p) = params {
            frame["params"] = p;
        }
        self.stream
            .send(Message::Text(frame.to_string()))
            .await
            .map_err(|e| WalletError::Http(format!("walletconnect send: {e}")))?;
        Ok(id)
    }

    /// Read the next JSON frame from the relay. Returns `None` on a clean
    /// close; binary frames are parsed as UTF-8 JSON, ping/pong frames are
    /// answered internally by tungstenite.
    pub async fn next_frame(&mut self) -> Result<Option<serde_json::Value>, WalletError> {
        loop {
            match self.stream.next().await {
                Some(Ok(Message::Text(t))) => {
                    let v: serde_json::Value = serde_json::from_str(&t).map_err(|e| {
                        WalletError::Http(format!("walletconnect invalid frame: {e}"))
                    })?;
                    return Ok(Some(v));
                }
                Some(Ok(Message::Binary(b))) => {
                    let v: serde_json::Value = serde_json::from_slice(&b).map_err(|e| {
                        WalletError::Http(format!("walletconnect invalid frame: {e}"))
                    })?;
                    return Ok(Some(v));
                }
                Some(Ok(Message::Ping(_))) | Some(Ok(Message::Pong(_))) => {
                    continue; // handled by tungstenite
                }
                Some(Ok(Message::Close(_))) => return Ok(None),
                Some(Ok(Message::Frame(_))) => continue, // raw frame, skip
                Some(Err(e)) => {
                    return Err(WalletError::Http(format!("walletconnect read: {e}")))
                }
                None => return Ok(None),
            }
        }
    }

    /// Close the socket cleanly.
    pub async fn close(&mut self) -> Result<(), WalletError> {
        self.stream
            .close(None)
            .await
            .map_err(|e| WalletError::Http(format!("walletconnect close: {e}")))
    }
}

#[cfg(test)]
mod tests {
    use super::WalletConnectSocket;

    #[test]
    fn ws_url_from_bare_host() {
        assert_eq!(
            WalletConnectSocket::topic_url("http://localhost:8443", "t1"),
            "ws://localhost:8443/api/v1/dapp/ws/t1"
        );
    }

    #[test]
    fn ws_url_from_https_with_api_prefix() {
        assert_eq!(
            WalletConnectSocket::topic_url("https://wallet.example.com/api/v1/", "t2"),
            "wss://wallet.example.com/api/v1/dapp/ws/t2"
        );
    }
}

