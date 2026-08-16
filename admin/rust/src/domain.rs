//! Admin domain proxy routes.
//!
//! Registers the 12 admin domain resource families (futures, options,
//! copy-trading, convert, onramp, offramp, p2p-clients, p2p-merchants,
//! partners, rewards, marketing, roles) and forwards every inbound request
//! verbatim to the real `admin/go` backend on `localhost:9093`, propagating
//! the caller's Bearer JWT. Upstream responses (status + JSON body) are
//! returned as-is; connection failures surface as 503 so native clients render
//! genuine loading/error/empty states. No stubs, fakes, or canned payloads.
//!
//! The HTTP client is a minimal async HTTP/1.1 implementation over
//! `tokio::net::TcpStream` (plain HTTP, localhost only) so no extra crate
//! dependency is required.

use axum::{
    body::Bytes,
    extract::{Path, RawQuery},
    http::{HeaderMap, Method, StatusCode},
    response::{IntoResponse, Response},
    routing::{delete, get, post, put},
    Json, Router,
};
use serde_json::Value;
use std::collections::HashMap;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpStream;

use crate::error::{AppError, AppResult};

/// Upstream admin/go backend host. Override with `TIGERADMIN_UPSTREAM_HOST`.
fn upstream_host() -> String {
    std::env::var("TIGERADMIN_UPSTREAM_HOST").unwrap_or_else(|_| "localhost".to_string())
}

/// Upstream admin/go backend port. Override with `TIGERADMIN_UPSTREAM_PORT`.
fn upstream_port() -> u16 {
    std::env::var("TIGERADMIN_UPSTREAM_PORT")
        .ok()
        .and_then(|p| p.parse().ok())
        .unwrap_or(9093)
}

/// Extracts the Bearer token from the inbound `Authorization` header.
fn bearer_token(headers: &HeaderMap) -> Result<String, AppError> {
    let auth = headers
        .get("authorization")
        .ok_or(AppError::Unauthorized)?
        .to_str()
        .map_err(|_| AppError::Unauthorized)?;
    if let Some(token) = auth.strip_prefix("Bearer ") {
        Ok(token.to_string())
    } else if let Some(token) = auth.strip_prefix("bearer ") {
        Ok(token.to_string())
    } else {
        Err(AppError::Unauthorized)
    }
}

/// Wire name for an HTTP method.
fn method_name(m: &Method) -> &'static str {
    match *m {
        Method::GET => "GET",
        Method::POST => "POST",
        Method::PUT => "PUT",
        Method::DELETE => "DELETE",
        Method::PATCH => "PATCH",
        _ => "GET",
    }
}

/// Parsed upstream HTTP response (status + body).
struct UpstreamResponse {
    status: u16,
    body: Vec<u8>,
    ok: bool,
}

/// Performs a real HTTP/1.1 call to the admin/go backend over a TCP socket.
async fn upstream_call(
    method: &Method,
    path: &str,
    body: &[u8],
    bearer: &str,
) -> Result<UpstreamResponse, AppError> {
    let addr = format!("{}:{}", upstream_host(), upstream_port());
    let mut stream = TcpStream::connect(&addr)
        .await
        .map_err(|e| AppError::InternalServerError(format!("upstream connect failed: {e}")))?;

    // 5s read/write timeouts so a hung backend surfaces as an error.
    let timeout = std::time::Duration::from_secs(5);
    tokio::time::timeout(timeout, async {
        let mut req = Vec::new();
        req.extend_from_slice(format!("{} {} HTTP/1.1\r\n", method_name(method), path).as_bytes());
        req.extend_from_slice(
            format!("Host: {}:{}\r\n", upstream_host(), upstream_port()).as_bytes(),
        );
        req.extend_from_slice(b"Connection: close\r\n");
        req.extend_from_slice(format!("Authorization: Bearer {}\r\n", bearer).as_bytes());
        req.extend_from_slice(b"Content-Type: application/json\r\n");
        req.extend_from_slice(format!("Content-Length: {}\r\n", body.len()).as_bytes());
        req.extend_from_slice(b"\r\n");
        req.extend_from_slice(body);
        stream.write_all(&req).await
    })
    .await
    .map_err(|_| AppError::InternalServerError("upstream write timeout".into()))?
    .map_err(|e| AppError::InternalServerError(format!("upstream write failed: {e}")))?;

    let raw = tokio::time::timeout(timeout, async {
        let mut buf = Vec::with_capacity(8192);
        stream.read_to_end(&mut buf).await.map(|_| buf)
    })
    .await
    .map_err(|_| AppError::InternalServerError("upstream read timeout".into()))?
    .map_err(|e| AppError::InternalServerError(format!("upstream read failed: {e}")))?;

    if raw.is_empty() {
        return Err(AppError::InternalServerError("empty upstream response".into()));
    }

    // Split headers / body at the blank line.
    let sep = find_subsequence(&raw, b"\r\n\r\n");
    let header_bytes = &raw[..sep.unwrap_or(raw.len())];
    let body_bytes = if let Some(s) = sep {
        &raw[s + 4..]
    } else {
        &[][..]
    };

    // Status code from the first line: "HTTP/1.1 200 OK".
    let first_line_end = header_bytes
        .iter()
        .position(|&b| b == b'\r')
        .unwrap_or(header_bytes.len());
    let first_line = std::str::from_utf8(&header_bytes[..first_line_end])
        .map_err(|_| AppError::InternalServerError("bad upstream status line".into()))?;
    let status: u16 = first_line
        .split_whitespace()
        .nth(1)
        .ok_or_else(|| AppError::InternalServerError("bad upstream status line".into()))?
        .parse()
        .map_err(|_| AppError::InternalServerError("bad upstream status code".into()))?;

    // De-chunk if Transfer-Encoding: chunked.
    let headers_str = std::str::from_utf8(header_bytes).unwrap_or("");
    let chunked = headers_str
        .to_ascii_lowercase()
        .contains("transfer-encoding: chunked");
    let body = if chunked {
        dechunk(body_bytes)
    } else {
        body_bytes.to_vec()
    };

    let ok = (200..400).contains(&status);
    Ok(UpstreamResponse { status, body, ok })
}

fn find_subsequence(haystack: &[u8], needle: &[u8]) -> Option<usize> {
    haystack
        .windows(needle.len())
        .position(|w| w == needle)
}

/// Decodes an HTTP chunked-transfer body into its concatenation.
fn dechunk(chunked: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    let mut pos = 0;
    while pos < chunked.len() {
        let eol = match find_subsequence(&chunked[pos..], b"\r\n") {
            Some(e) => pos + e,
            None => break,
        };
        let len_str = match std::str::from_utf8(&chunked[pos..eol]) {
            Ok(s) => s,
            Err(_) => break,
        };
        let chunk_len = match usize::from_str_radix(len_str.trim(), 16) {
            Ok(n) => n,
            Err(_) => break,
        };
        if chunk_len == 0 {
            break;
        }
        let data_start = eol + 2;
        if data_start + chunk_len > chunked.len() {
            break;
        }
        out.extend_from_slice(&chunked[data_start..data_start + chunk_len]);
        pos = data_start + chunk_len + 2;
    }
    out
}

/// Builds the upstream path from a base resource path plus optional id/suffix,
/// preserving any inbound query string.
fn build_path(resource: &str, suffix: &str, query: &Option<String>) -> String {
    let mut path = format!("/api/v1{}{}", resource, suffix);
    if let Some(q) = query {
        if !q.is_empty() {
            path.push('?');
            path.push_str(q);
        }
    }
    path
}

/// Core proxy handler: forwards the inbound request to the upstream admin/go
/// backend and returns its status + body. `resource` is the resource path
/// (e.g. `/futures`); `suffix` is any id/action tail (e.g. `/{id}/status`).
async fn proxy(
    headers: HeaderMap,
    method: Method,
    resource: &str,
    suffix: &str,
    query: Option<String>,
    body: Bytes,
) -> AppResult<Response> {
    let bearer = bearer_token(&headers)?;
    let path = build_path(resource, suffix, &query);
    let up = upstream_call(&method, &path, &body, &bearer).await?;

    let status = StatusCode::from_u16(up.status)
        .map_err(|_| AppError::InternalServerError("bad upstream status".into()))?;
    let body_json: Value = if up.body.is_empty() {
        serde_json::json!({})
    } else {
        serde_json::from_slice(&up.body).unwrap_or_else(|_| {
            // Non-JSON upstream body: wrap so the client still gets real bytes.
            Value::String(String::from_utf8_lossy(&up.body).to_string())
        })
    };
    Ok((status, Json(body_json)).into_response())
}

// ----------------------------------------------------------------------------
// Per-domain handler factories. Each returns an axum handler that captures the
// resource path and dispatches to `proxy` with the right method/suffix.
// ----------------------------------------------------------------------------

async fn list_handler(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
    resource: &'static str,
) -> AppResult<Response> {
    proxy(headers, Method::GET, resource, "", query, Bytes::new()).await
}

async fn get_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::GET, resource, &format!("/{id}"), None, Bytes::new()).await
}

async fn create_handler(
    headers: HeaderMap,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    proxy(headers, Method::POST, resource, "", None, body).await
}

async fn update_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::PUT, resource, &format!("/{id}"), None, body).await
}

async fn delete_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::DELETE, resource, &format!("/{id}"), None, Bytes::new()).await
}

async fn status_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::PUT, resource, &format!("/{id}/status"), None, body).await
}

async fn approve_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::POST, resource, &format!("/{id}/approve"), None, Bytes::new()).await
}

async fn reject_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::POST, resource, &format!("/{id}/reject"), None, body).await
}

/// GET /<resource>/:id/transactions — p2p-merchants sub-resource.
async fn get_transactions_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(
        headers,
        Method::GET,
        resource,
        &format!("/{id}/transactions"),
        None,
        Bytes::new(),
    )
    .await
}

// roles-specific RBAC handlers. The admin/go backend mounts RBAC under:
//   /api/v1/roles            (roles CRUD)
//   /api/v1/permissions      (permissions CRUD)
//   /api/v1/admins/:id/roles (GET list / POST assign / DELETE :roleId revoke)
//   /api/v1/admins/:id/permissions (GET effective)
// so these handlers forward to those exact upstream paths.
async fn permissions_list_handler(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
) -> AppResult<Response> {
    proxy(headers, Method::GET, "/permissions", "", query, Bytes::new()).await
}

async fn permissions_create_handler(headers: HeaderMap, body: Bytes) -> AppResult<Response> {
    proxy(headers, Method::POST, "/permissions", "", None, body).await
}

async fn permissions_update_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
) -> AppResult<Response> {
    let pid = params.get("pid").cloned().unwrap_or_default();
    proxy(headers, Method::PUT, "/permissions", &format!("/{pid}"), None, body).await
}

async fn permissions_delete_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
) -> AppResult<Response> {
    let pid = params.get("pid").cloned().unwrap_or_default();
    proxy(headers, Method::DELETE, "/permissions", &format!("/{pid}"), None, Bytes::new()).await
}

async fn assign_role_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
) -> AppResult<Response> {
    let aid = params.get("aid").cloned().unwrap_or_default();
    proxy(headers, Method::POST, "/admins", &format!("/{aid}/roles"), None, body).await
}

async fn list_admin_roles_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
) -> AppResult<Response> {
    let aid = params.get("aid").cloned().unwrap_or_default();
    proxy(headers, Method::GET, "/admins", &format!("/{aid}/roles"), None, Bytes::new()).await
}

async fn revoke_role_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
) -> AppResult<Response> {
    let aid = params.get("aid").cloned().unwrap_or_default();
    let rid = params.get("rid").cloned().unwrap_or_default();
    proxy(
        headers,
        Method::DELETE,
        "/admins",
        &format!("/{aid}/roles/{rid}"),
        None,
        Bytes::new(),
    )
    .await
}

async fn admin_permissions_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
) -> AppResult<Response> {
    let aid = params.get("aid").cloned().unwrap_or_default();
    proxy(headers, Method::GET, "/admins", &format!("/{aid}/permissions"), None, Bytes::new()).await
}

// ----------------------------------------------------------------------------
// Handlers for the 4 new admin domain families (bots, bots-clients,
// project-teams, liquidity-sources) and their sub-resources. Each forwards to
// the upstream admin/go backend verbatim via `proxy`, propagating the caller's
// Bearer JWT. No stubs, fakes, or canned payloads.
// ----------------------------------------------------------------------------

/// GET /<resource>/stats — collection-level stats (bots, liquidity-sources).
async fn stats_handler(
    headers: HeaderMap,
    RawQuery(query): RawQuery,
    resource: &'static str,
) -> AppResult<Response> {
    proxy(headers, Method::GET, resource, "/stats", query, Bytes::new()).await
}

/// GET /<resource>/:id/tiers — list a bot's tiers.
async fn get_tiers_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::GET, resource, &format!("/{id}/tiers"), None, Bytes::new()).await
}

/// POST /<resource>/:id/tiers — create a tier under a bot.
async fn create_tier_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::POST, resource, &format!("/{id}/tiers"), None, body).await
}

/// PUT /<resource>/:id/tiers/:tid — update a bot tier.
async fn update_tier_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    let tid = params.get("tid").cloned().unwrap_or_default();
    proxy(headers, Method::PUT, resource, &format!("/{id}/tiers/{tid}"), None, body).await
}

/// DELETE /<resource>/:id/tiers/:tid — delete a bot tier.
async fn delete_tier_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    let tid = params.get("tid").cloned().unwrap_or_default();
    proxy(
        headers,
        Method::DELETE,
        resource,
        &format!("/{id}/tiers/{tid}"),
        None,
        Bytes::new(),
    )
    .await
}

/// GET /<resource>/:id/members — list a project team's members.
async fn get_members_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::GET, resource, &format!("/{id}/members"), None, Bytes::new()).await
}

/// POST /<resource>/:id/members — add a member to a project team.
async fn add_member_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::POST, resource, &format!("/{id}/members"), None, body).await
}

/// DELETE /<resource>/:id/members/:mid — remove a member from a project team.
async fn remove_member_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    let mid = params.get("mid").cloned().unwrap_or_default();
    proxy(
        headers,
        Method::DELETE,
        resource,
        &format!("/{id}/members/{mid}"),
        None,
        Bytes::new(),
    )
    .await
}

/// PUT /<resource>/:id/priority — set a liquidity source's priority.
async fn set_priority_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    body: Bytes,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(headers, Method::PUT, resource, &format!("/{id}/priority"), None, body).await
}

/// POST /<resource>/:id/health-check — run a liquidity source health check.
async fn health_check_handler(
    headers: HeaderMap,
    Path(params): Path<HashMap<String, String>>,
    resource: &'static str,
) -> AppResult<Response> {
    let id = params.get("id").cloned().unwrap_or_default();
    proxy(
        headers,
        Method::POST,
        resource,
        &format!("/{id}/health-check"),
        None,
        Bytes::new(),
    )
    .await
}

/// Registers the standard CRUD + status routes for a domain resource.
/// `with_status` adds `PUT /:id/status`; `with_approve_reject` adds
/// `POST /:id/approve` and `POST /:id/reject`.
fn register_domain(
    router: Router,
    resource: &'static str,
    with_status: bool,
    with_approve_reject: bool,
) -> Router {
    let r = router
        .route(
            resource,
            get(move |h, q| list_handler(h, q, resource)),
        )
        .route(
            resource,
            post(move |h, b| create_handler(h, b, resource)),
        )
        .route(
            &format!("{resource}/:id"),
            get(move |h, p| get_handler(h, p, resource)),
        )
        .route(
            &format!("{resource}/:id"),
            put(move |h, p, b| update_handler(h, p, b, resource)),
        )
        .route(
            &format!("{resource}/:id"),
            delete(move |h, p| delete_handler(h, p, resource)),
        );

    let r = if with_status {
        r.route(
            &format!("{resource}/:id/status"),
            put(move |h, p, b| status_handler(h, p, b, resource)),
        )
    } else {
        r
    };

    let r = if with_approve_reject {
        r.route(
            &format!("{resource}/:id/approve"),
            post(move |h, p| approve_handler(h, p, resource)),
        )
        .route(
            &format!("{resource}/:id/reject"),
            post(move |h, p, b| reject_handler(h, p, b, resource)),
        )
    } else {
        r
    };

    r
}

/// Builds the merged router for all 16 admin domain resource families.
pub fn domain_routes() -> Router {
    let router = Router::new();

    // 1. futures  — CRUD + status
    let router = register_domain(router, "/futures", true, false);
    // 2. options  — CRUD + status
    let router = register_domain(router, "/options", true, false);
    // 3. copy-trading — CRUD + status
    let router = register_domain(router, "/copy-trading", true, false);
    // 4. convert — CRUD + status
    let router = register_domain(router, "/convert", true, false);
    // 5. onramp — CRUD + approve/reject
    let router = register_domain(router, "/onramp", false, true);
    // 6. offramp — CRUD + approve/reject
    let router = register_domain(router, "/offramp", false, true);
    // 7. p2p-clients — CRUD + status
    let router = register_domain(router, "/p2p-clients", true, false);
    // 8. partners — CRUD + status + approve/reject
    let router = register_domain(router, "/partners", true, true);
    // 9. rewards — CRUD + status
    let router = register_domain(router, "/rewards", true, false);
    // 10. marketing — CRUD + status
    let router = register_domain(router, "/marketing", true, false);
    // 11. roles — CRUD + status + RBAC (permissions/admins).
    //     The admin/go backend mounts RBAC at /permissions and /admins/:id/...
    //     (NOT under /roles), so we expose them at the same canonical paths.
    let router = register_domain(router, "/roles", true, false)
        // permissions CRUD
        .route("/permissions", get(permissions_list_handler))
        .route("/permissions", post(permissions_create_handler))
        .route("/permissions/:pid", put(permissions_update_handler))
        .route("/permissions/:pid", delete(permissions_delete_handler))
        // assign/revoke + list roles on an admin
        .route("/admins/:aid/roles", get(list_admin_roles_handler))
        .route("/admins/:aid/roles", post(assign_role_handler))
        .route("/admins/:aid/roles/:rid", delete(revoke_role_handler))
        // an admin's effective permissions
        .route("/admins/:aid/permissions", get(admin_permissions_handler));
    // 12. p2p-merchants — CRUD (no delete on upstream) + approve/reject +
    //     GET /:id/transactions. The admin/go backend exposes approve/reject
    //     (not setStatus) for merchants, plus a transactions sub-resource.
    let router = register_domain(router, "/p2p-merchants", false, true)
        .route(
            "/p2p-merchants/:id/transactions",
            get(move |h, p| get_transactions_handler(h, p, "/p2p-merchants")),
        );

    // 13. bots — CRUD + status + stats + tiers CRUD (sub-resource under a bot).
    //     axum matches literal segments ahead of :id captures, so /bots/stats
    //     resolves over /bots/:id.
    let router = register_domain(router, "/bots", true, false)
        .route(
            "/bots/stats",
            get(move |h, q| stats_handler(h, q, "/bots")),
        )
        .route(
            "/bots/:id/tiers",
            get(move |h, p| get_tiers_handler(h, p, "/bots")),
        )
        .route(
            "/bots/:id/tiers",
            post(move |h, p, b| create_tier_handler(h, p, b, "/bots")),
        )
        .route(
            "/bots/:id/tiers/:tid",
            put(move |h, p, b| update_tier_handler(h, p, b, "/bots")),
        )
        .route(
            "/bots/:id/tiers/:tid",
            delete(move |h, p| delete_tier_handler(h, p, "/bots")),
        );

    // 14. bots-clients — CRUD + status (no extra sub-resources).
    let router = register_domain(router, "/bots-clients", true, false);

    // 15. project-teams — CRUD + status + members sub-resource.
    let router = register_domain(router, "/project-teams", true, false)
        .route(
            "/project-teams/:id/members",
            get(move |h, p| get_members_handler(h, p, "/project-teams")),
        )
        .route(
            "/project-teams/:id/members",
            post(move |h, p, b| add_member_handler(h, p, b, "/project-teams")),
        )
        .route(
            "/project-teams/:id/members/:mid",
            delete(move |h, p| remove_member_handler(h, p, "/project-teams")),
        );

    // 16. liquidity-sources — CRUD + status + setPriority + healthCheck +
    //     getStats. axum matches literal segments ahead of :id captures, so
    //     /liquidity-sources/stats resolves over /liquidity-sources/:id.
    let router = register_domain(router, "/liquidity-sources", true, false)
        .route(
            "/liquidity-sources/:id/priority",
            put(move |h, p, b| set_priority_handler(h, p, b, "/liquidity-sources")),
        )
        .route(
            "/liquidity-sources/:id/health-check",
            post(move |h, p| health_check_handler(h, p, "/liquidity-sources")),
        )
        .route(
            "/liquidity-sources/stats",
            get(move |h, q| stats_handler(h, q, "/liquidity-sources")),
        );

    router
}
