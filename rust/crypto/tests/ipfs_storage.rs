// IPFS client tests.
//
// These run against a real HTTP server bound to an ephemeral loopback port, so
// the request the client builds is actually parsed and answered. A stubbed
// transport would not catch a malformed path, a missing query parameter, or a
// response field read from the wrong place.

use std::io::{BufRead, BufReader, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::thread;

use tigerwallet_crypto::security_module::IPFSStorage;

/// Serves one canned HTTP response and reports the request line it received.
fn serve_once(status: &'static str, body: &'static str) -> (String, thread::JoinHandle<String>) {
    let listener = TcpListener::bind("127.0.0.1:0").expect("bind");
    let addr = listener.local_addr().expect("addr");

    let handle = thread::spawn(move || {
        let (mut stream, _) = listener.accept().expect("accept");
        let request_line = read_request(&mut stream);
        let response = format!(
            "HTTP/1.1 {status}\r\nContent-Length: {}\r\nContent-Type: application/json\r\nConnection: close\r\n\r\n{body}",
            body.len()
        );
        stream.write_all(response.as_bytes()).expect("write");
        stream.flush().ok();
        request_line
    });

    (format!("http://{addr}"), handle)
}

fn read_request(stream: &mut TcpStream) -> String {
    let mut reader = BufReader::new(stream.try_clone().expect("clone"));
    let mut request_line = String::new();
    reader.read_line(&mut request_line).expect("read line");

    let mut content_length = 0usize;
    loop {
        let mut header = String::new();
        reader.read_line(&mut header).expect("read header");
        if header == "\r\n" || header.is_empty() {
            break;
        }
        let lower = header.to_ascii_lowercase();
        if let Some(rest) = lower.strip_prefix("content-length:") {
            content_length = rest.trim().parse().unwrap_or(0);
        }
    }

    if content_length > 0 {
        let mut body = vec![0u8; content_length];
        reader.read_exact(&mut body).expect("read body");
    }
    request_line
}

#[test]
fn upload_returns_the_cid_the_server_reported() {
    let (base, handle) = serve_once(
        "200 OK",
        r#"{"Name":"blob","Hash":"QmRealCidFromNode","Size":"4"}"#,
    );
    let storage = IPFSStorage::new(base, None);

    let cid = storage.upload(b"test").expect("upload should succeed");
    assert_eq!(cid, "QmRealCidFromNode");

    let request_line = handle.join().expect("join");
    assert!(
        request_line.contains("/api/v0/add"),
        "must call the add endpoint, got: {request_line}"
    );
    assert!(request_line.contains("pin=true"), "must request pinning");
}

#[test]
fn upload_reports_a_missing_hash_field() {
    let (base, handle) = serve_once("200 OK", r#"{"Name":"blob"}"#);
    let storage = IPFSStorage::new(base, None);

    let err = storage.upload(b"test").expect_err("no Hash field means no CID");
    assert!(err.contains("Hash"), "error should name the field: {err}");
    handle.join().ok();
}

#[test]
fn upload_reports_an_http_error() {
    let (base, handle) = serve_once("500 Internal Server Error", r#"{"Message":"boom"}"#);
    let storage = IPFSStorage::new(base, None);

    let err = storage.upload(b"test").expect_err("500 must not yield a CID");
    assert!(err.contains("500"), "error should carry the status: {err}");
    handle.join().ok();
}

#[test]
fn download_returns_the_body() {
    let (base, handle) = serve_once("200 OK", "payload-bytes");
    let storage = IPFSStorage::new(base, None);

    let data = storage.download("QmSomething").expect("download");
    assert_eq!(data, b"payload-bytes");

    let request_line = handle.join().expect("join");
    assert!(request_line.contains("/api/v0/cat"), "must call cat: {request_line}");
    assert!(
        request_line.contains("arg=QmSomething"),
        "must pass the CID: {request_line}"
    );
}

#[test]
fn download_rejects_an_empty_cid() {
    let storage = IPFSStorage::new("http://127.0.0.1:1".to_string(), None);
    assert!(storage.download("").is_err());
}

#[test]
fn download_reports_an_http_error() {
    let (base, handle) = serve_once("404 Not Found", "not found");
    let storage = IPFSStorage::new(base, None);
    assert!(storage.download("QmMissing").is_err());
    handle.join().ok();
}

#[test]
fn pin_calls_the_pin_add_endpoint() {
    let (base, handle) = serve_once("200 OK", r#"{"Pins":["QmPinMe"]}"#);
    let storage = IPFSStorage::new(base, None);
    storage.pin("QmPinMe").expect("pin");
    let request_line = handle.join().expect("join");
    assert!(request_line.contains("/api/v0/pin/add"), "got: {request_line}");
}

#[test]
fn unpin_calls_the_pin_rm_endpoint() {
    let (base, handle) = serve_once("200 OK", r#"{"Pins":["QmPinMe"]}"#);
    let storage = IPFSStorage::new(base, None);
    storage.unpin("QmPinMe").expect("unpin");
    let request_line = handle.join().expect("join");
    assert!(request_line.contains("/api/v0/pin/rm"), "got: {request_line}");
}

#[test]
fn pin_rejects_an_empty_cid() {
    let storage = IPFSStorage::new("http://127.0.0.1:1".to_string(), None);
    assert!(storage.pin("").is_err());
    assert!(storage.unpin("").is_err());
}

#[test]
fn unreachable_endpoint_is_an_error_not_a_placeholder() {
    // Nothing listens on port 1, so every call must fail rather than return a
    // fabricated CID or an empty success.
    let storage = IPFSStorage::new("http://127.0.0.1:1".to_string(), None);
    assert!(storage.upload(b"data").is_err());
    assert!(storage.download("QmX").is_err());
    assert!(storage.pin("QmX").is_err());
}
