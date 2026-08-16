//! Crypto Cards governance handlers.
//!
//! Dedicated handlers for the `/api/v1/admin/crypto-cards` routes on the
//! upstream Go super-admin backend (port 8082). Each handler forwards the JWT
//! bearer token and request body upstream and relays the JSON response. No
//! stubs: an upstream failure surfaces as an honest error.
//!
//! Routes:
//!   GET    /crypto-cards            -> list
//!   POST   /crypto-cards            -> create
//!   GET    /crypto-cards/:id        -> get one
//!   PUT    /crypto-cards/:id        -> update
//!   DELETE /crypto-cards/:id        -> delete
//!   POST   /crypto-cards/:id/block  -> block card
//!   POST   /crypto-cards/:id/activate -> activate card
//!   PUT    /crypto-cards/:id/limit  -> set card limit
//!   PUT    /crypto-cards/:id/status -> update status

use axum::{
    body::to_bytes,
    extract::{Path, Request, State},
    http::{HeaderName, StatusCode},
    response::{IntoResponse, Response},
};

use super::{AppState, DomainClient};

/// Resource path segment under `/api/v1/admin`.
pub const RESOURCE: &str = "crypto-cards";

/// Dispatch a request to `crypto-cards{suffix}` using the incoming method,
/// forwarding auth + body. `suffix` is appended verbatim (e.g. "", ":id",
/// ":id/block"). Query strings are preserved.
async fn dispatch(
    state: &DomainClient,
    req: Request,
    suffix: &str,
) -> Response {
    let method = req.method().clone();
    let headers = req.headers().clone();

    let mut path = if suffix.is_empty() {
        RESOURCE.to_string()
    } else {
        format!("{}/{}", RESOURCE, suffix)
    };
    if let Some(q) = req.uri().query() {
        if !q.is_empty() {
            path.push('?');
            path.push_str(q);
        }
    }

    let body = match to_bytes(req.into_body(), 1024 * 1024).await {
        Ok(b) => b,
        Err(_) => {
            return (StatusCode::BAD_REQUEST, "invalid body").into_response();
        }
    };

    match state.forward(method, &path, &headers, body).await {
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

/// Extract the card id segment and return it together with the dispatch suffix
/// `"{id}/{action}"` (or just `"{id}"` when action is empty).
fn id_suffix(id: String, action: &str) -> String {
    if action.is_empty() {
        id
    } else {
        format!("{}/{}", id, action)
    }
}

pub async fn list_cards(State(state): State<AppState>, req: Request) -> Response {
    dispatch(&state.client, req, "").await
}

pub async fn create_card(State(state): State<AppState>, req: Request) -> Response {
    dispatch(&state.client, req, "").await
}

pub async fn get_card(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id).await
}

pub async fn update_card(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id).await
}

pub async fn delete_card(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id).await
}

pub async fn block_card(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id_suffix(id, "block")).await
}

pub async fn activate_card(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id_suffix(id, "activate")).await
}

pub async fn set_card_limit(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id_suffix(id, "limit")).await
}

pub async fn set_card_status(State(state): State<AppState>, Path(id): Path<String>, req: Request) -> Response {
    dispatch(&state.client, req, &id_suffix(id, "status")).await
}
