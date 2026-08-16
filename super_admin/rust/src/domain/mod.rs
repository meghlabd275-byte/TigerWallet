//! Domain admin governance handlers.
//!
//! These handlers drive the real `super_admin/go` backend (port 8082) by
//! forwarding authenticated requests over HTTP (reqwest) and relaying the
//! upstream JSON response back to the caller. No stubs, fakes, or mocks: a
//! backend failure surfaces as an honest error.
//!
//! Base URL: `http://localhost:8082/api/v1/admin`. JWT bearer auth is
//! forwarded from the incoming `Authorization` header.
//!
//! Governance records only; never moves crypto assets.

use anyhow::Result;
use axum::{
    body::to_bytes,
    extract::{Path, Request, State},
    http::{HeaderMap, HeaderName, HeaderValue, Method, StatusCode},
    response::{IntoResponse, Response},
};
use serde::Serialize;
use serde_json::Value;

/// Upstream Go super-admin backend base.
pub const UPSTREAM_BASE: &str = "http://localhost:8082/api/v1/admin";

/// Shared reqwest client (connection-pooled).
#[derive(Clone)]
pub struct DomainClient {
    http: reqwest::Client,
}

impl DomainClient {
    pub fn new() -> Self {
        Self {
            http: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(30))
                .build()
                .expect("reqwest client"),
        }
    }

    /// Forward a request to `UPSTREAM_BASE/{path}` using the given method,
    /// forwarding the `Authorization` and `Content-Type` headers and the
    /// request body.
    async fn forward(
        &self,
        method: Method,
        path: &str,
        headers: &HeaderMap,
        body: bytes::Bytes,
    ) -> Result<Response> {
        // path already begins with the domain segment, e.g. "futures" or
        // "futures/UUID/status"; it may also carry a query string.
        let url = format!("{}/{}", UPSTREAM_BASE, path);

        // Bridge axum (http 1.x) Method -> reqwest (http 0.2) Method.
        let rw_method = match method.as_str() {
            "GET" => reqwest::Method::GET,
            "POST" => reqwest::Method::POST,
            "PUT" => reqwest::Method::PUT,
            "DELETE" => reqwest::Method::DELETE,
            "PATCH" => reqwest::Method::PATCH,
            other => reqwest::Method::from_bytes(other.as_bytes())
                .map_err(|e| anyhow::anyhow!("invalid method {}: {}", other, e))?,
        };
        let mut req = self.http.request(rw_method, &url);
        req = req.body(body);

        // Forward Authorization (JWT bearer) and Content-Type only.
        if let Some(auth) = headers.get(axum::http::header::AUTHORIZATION) {
            if let Ok(val) = auth.to_str() {
                req = req.header(reqwest::header::AUTHORIZATION, val);
            }
        }
        if let Some(ct) = headers.get(axum::http::header::CONTENT_TYPE) {
            if let Ok(val) = ct.to_str() {
                req = req.header(reqwest::header::CONTENT_TYPE, val);
            }
        }

        let upstream = req.send().await?;
        // reqwest (http 0.2) and axum (http 1.x) use distinct header/status
        // types, so bridge by converting to the axum types explicitly.
        let status = StatusCode::from_u16(upstream.status().as_u16())
            .unwrap_or(StatusCode::BAD_GATEWAY);
        let resp_headers = upstream.headers().clone();
        let resp_bytes = upstream.bytes().await?;

        let mut out = Response::builder().status(status);
        out = out.header(HeaderName::from_static("content-type"), "application/json");
        // Echo a couple of safe upstream headers when present.
        for (k, v) in resp_headers.iter() {
            let name = k.as_str();
            if name == "content-type" || name == "x-request-id" {
                if let Ok(s) = v.to_str() {
                    if let (Ok(hn), Ok(hv)) = (
                        HeaderName::from_bytes(name.as_bytes()),
                        HeaderValue::from_str(s),
                    ) {
                        out = out.header(hn, hv);
                    }
                }
            }
        }
        Ok(out.body(axum::body::Body::from(resp_bytes))?)
    }
}

impl Default for DomainClient {
    fn default() -> Self {
        Self::new()
    }
}

/// Shared application state.
#[derive(Clone)]
pub struct AppState {
    pub client: DomainClient,
}

impl AppState {
    pub fn new() -> Self {
        Self { client: DomainClient::new() }
    }
}

impl Default for AppState {
    fn default() -> Self {
        Self::new()
    }
}

/// Generic catch-all proxy for a domain subtree.
///
/// `segments` is the full remaining path under `/api/v1/admin/` (e.g.
/// `["futures"]`, `["futures", ":id"]`, `["futures", ":id", "status"]`).
/// Query strings are preserved by reading the raw request URI.
pub async fn proxy_domain(
    State(state): State<AppState>,
    Path(segments): Path<Vec<String>>,
    req: Request,
) -> Response {
    let method = req.method().clone();
    let headers = req.headers().clone();

    // Rebuild the remaining path under /api/v1/admin/.
    let mut path = segments.join("/");
    // Preserve the query string if any.
    if let Some(q) = req.uri().query() {
        if !q.is_empty() {
            path.push('?');
            path.push_str(q);
        }
    }

    // Collect the body (empty for GET/DELETE).
    let body = match to_bytes(req.into_body(), 1024 * 1024).await {
        Ok(b) => b,
        Err(_) => {
            return (StatusCode::BAD_REQUEST, "invalid body").into_response();
        }
    };

    match state.client.forward(method, &path, &headers, body).await {
        Ok(resp) => resp,
        Err(e) => {
            let msg = format!(
                "{{\"error\":\"upstream unreachable: {}\"}}",
                e.to_string().replace('"', "\\\"")
            );
            (
                StatusCode::BAD_GATEWAY,
                [(HeaderName::from_static("content-type"), "application/json")],
                msg,
            )
                .into_response()
        }
    }
}

/// Description of one admin domain.
#[derive(Debug, Clone, Serialize)]
pub struct DomainInfo {
    pub name: String,
    pub resource: String,
    pub actions: Vec<String>,
}

/// The 12 supported domains and their governance actions.
pub fn domain_manifest() -> Vec<DomainInfo> {
    vec![
        DomainInfo { name: "futures".into(), resource: "/futures".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "options".into(), resource: "/options".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "copy-trading".into(), resource: "/copy-trading".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "convert".into(), resource: "/convert".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "onramp".into(), resource: "/onramp".into(), actions: vec!["crud".into(), "approve".into(), "reject".into()] },
        DomainInfo { name: "offramp".into(), resource: "/offramp".into(), actions: vec!["crud".into(), "approve".into(), "reject".into()] },
        DomainInfo { name: "p2p-clients".into(), resource: "/p2p-clients".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "partners".into(), resource: "/partners".into(), actions: vec!["crud".into(), "status".into(), "approve".into(), "reject".into()] },
        DomainInfo { name: "rewards".into(), resource: "/rewards".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "marketing".into(), resource: "/marketing".into(), actions: vec!["crud".into(), "status".into()] },
        DomainInfo { name: "admin-roles".into(), resource: "/admin-roles".into(), actions: vec!["roles-crud".into(), "permissions-crud".into(), "assign-role".into(), "effective-permissions".into()] },
        DomainInfo { name: "wl-control".into(), resource: "/wl-clients,/wl-master-wallets,/wl-user-wallets,/wl-bots,/wl-bots-clients".into(), actions: vec!["crud".into(), "status".into()] },
    ]
}

/// Handler returning the domain manifest as JSON.
pub async fn list_domains(State(_): State<AppState>) -> axum::Json<Value> {
    axum::Json(serde_json::json!({ "domains": domain_manifest() }))
}
