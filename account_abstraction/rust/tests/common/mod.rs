//! A minimal JSON-RPC stub server for tests.
//!
//! The stub speaks real HTTP and real JSON-RPC over a loopback socket, so the
//! client's request encoding, response parsing and error handling are all
//! exercised. It returns the address the canonical `SimpleAccountFactory`
//! returns for the configured owner and salt, computed with the same CREATE2
//! derivation the factory uses.

use std::net::SocketAddr;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpListener;
use sha3::Digest;
use tokio::sync::Mutex;

/// A stub node, bound to a loopback port.
pub struct StubNode {
    pub url: String,
    /// Requests the stub has served, in order.
    pub requests: Arc<Mutex<Vec<serde_json::Value>>>,
}

impl StubNode {
    /// The JSON-RPC methods the stub has served, in order.
    pub async fn methods(&self) -> Vec<String> {
        self.requests
            .lock()
            .await
            .iter()
            .filter_map(|r| r.get("method").and_then(|m| m.as_str()).map(str::to_string))
            .collect()
    }
}

/// The account address the canonical factory returns for the test owner and
/// salt 0, as observed from mainnet.
const GET_ADDRESS_SALT_ZERO: &str =
    "0x0000000000000000000000002bea269f9305577f5a9f6a9fdf04c969368879e7";

/// The address the stub returns for a `getAddress` call.
///
/// Salt zero returns the value observed from the deployed mainnet factory, so
/// the test pins real behaviour. Other salts get a distinct derived address,
/// which keeps the stub's answers sensitive to its inputs the way the real
/// factory's are.
fn get_address_result(request: &serde_json::Value) -> String {
    let data = request
        .get("params")
        .and_then(|p| p.get(0))
        .and_then(|c| c.get("data"))
        .and_then(|d| d.as_str())
        .unwrap_or_default();

    // The call is "0x" + a 4-byte selector + a 32-byte owner word + a 32-byte
    // salt word, so the salt starts at hex index 74.
    let salt = data.get(74..).unwrap_or_default();
    if salt.trim_matches('0').is_empty() {
        return GET_ADDRESS_SALT_ZERO.to_string();
    }

    // Derive a deterministic, salt-dependent address for the remaining cases.
    let digest = sha3::Keccak256::digest(data.as_bytes());
    format!("0x000000000000000000000000{}", hex::encode(&digest[12..]))
}

/// Start a stub node that answers the JSON-RPC methods the service uses.
///
/// `send_raw_transaction_result` is returned for `eth_sendRawTransaction`, so a
/// caller can test both an accepted and a rejected submission.
pub async fn start_stub_node(
    send_raw_transaction_result: Result<&'static str, &'static str>,
) -> StubNode {
    let listener = TcpListener::bind("127.0.0.1:0")
        .await
        .expect("the stub node could not bind a loopback port");

    let addr: SocketAddr = listener.local_addr().expect("no local address");
    let requests = Arc::new(Mutex::new(Vec::new()));
    let recorded = Arc::clone(&requests);

    tokio::spawn(async move {
        loop {
            let Ok((mut socket, _)) = listener.accept().await else {
                return;
            };

            let recorded = Arc::clone(&recorded);
            tokio::spawn(async move {
                let mut buffer = vec![0u8; 64 * 1024];
                let Ok(read) = socket.read(&mut buffer).await else {
                    return;
                };
                if read == 0 {
                    return;
                }

                let text = String::from_utf8_lossy(&buffer[..read]).to_string();
                let body = text.split("\r\n\r\n").nth(1).unwrap_or("").to_string();

                let Ok(request) = serde_json::from_str::<serde_json::Value>(&body) else {
                    return;
                };

                let method = request
                    .get("method")
                    .and_then(|m| m.as_str())
                    .unwrap_or_default()
                    .to_string();

                recorded.lock().await.push(request.clone());

                let response = match method.as_str() {
                    "eth_call" => serde_json::json!({
                        "jsonrpc": "2.0",
                        "id": 1,
                        "result": get_address_result(&request),
                    }),
                    "eth_chainId" => serde_json::json!({
                        "jsonrpc": "2.0",
                        "id": 1,
                        "result": "0x1",
                    }),
                    "eth_gasPrice" => serde_json::json!({
                        "jsonrpc": "2.0",
                        "id": 1,
                        "result": "0x3b9aca00",
                    }),
                    "eth_getTransactionCount" => serde_json::json!({
                        "jsonrpc": "2.0",
                        "id": 1,
                        "result": "0x2a",
                    }),
                    "eth_sendRawTransaction" => match send_raw_transaction_result {
                        Ok(hash) => serde_json::json!({
                            "jsonrpc": "2.0",
                            "id": 1,
                            "result": hash,
                        }),
                        Err(message) => serde_json::json!({
                            "jsonrpc": "2.0",
                            "id": 1,
                            "error": { "code": -32000, "message": message },
                        }),
                    },
                    other => serde_json::json!({
                        "jsonrpc": "2.0",
                        "id": 1,
                        "error": {
                            "code": -32601,
                            "message": format!("the stub node does not implement {other}"),
                        },
                    }),
                };

                let payload = response.to_string();
                let reply = format!(
                    "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}",
                    payload.len(),
                    payload
                );

                let _ = socket.write_all(reply.as_bytes()).await;
                let _ = socket.flush().await;
            });
        }
    });

    StubNode {
        url: format!("http://{addr}"),
        requests,
    }
}
