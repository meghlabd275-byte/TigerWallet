//! HTTP handlers.
//!
//! Every handler authenticates before doing any work, validates its input, and
//! returns an explicit error rather than a default. A request that cannot be
//! verified is never reported as verified.

use std::sync::Arc;

use axum::extract::{Path, State};
use axum::http::{HeaderMap, StatusCode};
use axum::response::{IntoResponse, Response};
use axum::Json;
use chrono::Utc;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use tiger_kyc_engine::{
    verify, Applicant, Decision, ListKind, Policy, VerificationLevel, WatchlistEntry,
};

use crate::auth::{is_reviewer_authenticated, is_service_authenticated};
use crate::store::status;
use crate::AppState;

/// Health probe. Reports liveness only; readiness is a separate route so that
/// a database outage does not make an orchestrator restart a healthy process.
pub async fn health() -> Response {
    Json(serde_json::json!({
        "status": "healthy",
        "service": "tiger-kyc-service",
        "policy_version": tiger_kyc_engine::POLICY_VERSION,
    }))
    .into_response()
}

/// Readiness probe. Fails when the database cannot answer, because without it
/// no verification can be recorded.
pub async fn readyz(State(state): State<Arc<AppState>>) -> Response {
    match state.store.ping().await {
        Ok(()) => (
            StatusCode::OK,
            Json(serde_json::json!({"status": "ready", "database": "ok"})),
        )
            .into_response(),
        Err(e) => (
            StatusCode::SERVICE_UNAVAILABLE,
            Json(serde_json::json!({"status": "not_ready", "database": e.to_string()})),
        )
            .into_response(),
    }
}

/// A verification submission.
#[derive(Debug, Deserialize)]
pub struct SubmitRequest {
    pub user_id: Uuid,
    /// The raw machine-readable zone, as lines of text.
    pub mrz: String,
    pub first_name: String,
    pub last_name: String,
    pub country: String,
    pub date_of_birth: String,
    #[serde(default)]
    pub address: Option<String>,
    #[serde(default)]
    pub address_country: Option<String>,
    /// Requested level: 1 basic, 2 standard (P2P and cards), 3 enhanced.
    pub level: i32,
}

/// The response describing a decision.
#[derive(Debug, Serialize)]
pub struct StatusResponse {
    pub user_id: Uuid,
    pub status: String,
    pub level: i32,
    pub risk_score: i32,
    pub reasons: Vec<String>,
    pub policy_version: String,
    /// True only when the status is `verified`. This is the flag the wallet
    /// backend reads.
    pub verified: bool,
}

/// Submits an application, runs verification, and records the decision.
pub async fn submit(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(req): Json<SubmitRequest>,
) -> Response {
    if !is_service_authenticated(&state, &headers) {
        return err(StatusCode::UNAUTHORIZED, "invalid or missing service token");
    }

    let level = match req.level {
        1 => VerificationLevel::Basic,
        2 => VerificationLevel::Standard,
        3 => VerificationLevel::Enhanced,
        other => {
            return err(
                StatusCode::BAD_REQUEST,
                &format!("level {other} is not a valid verification level (1, 2 or 3)"),
            )
        }
    };

    // Parse and fully check-digit-validate the machine-readable zone. A
    // document that fails any ICAO 9303 check digit is refused here, before
    // any screening or scoring happens.
    let mrz = match tiger_kyc_engine::parse_mrz(&req.mrz) {
        Ok(m) => m,
        Err(e) => {
            // Record the attempt. A malformed or forged document is exactly the
            // event an operator needs to see, so it is audited even though the
            // request is refused.
            let _ = state
                .store
                .audit(
                    Some(req.user_id),
                    None,
                    None,
                    "submission_rejected_invalid_mrz",
                    &serde_json::json!({"reason": e.to_string()}),
                )
                .await;
            return err(
                StatusCode::BAD_REQUEST,
                &format!("document check failed: {e}"),
            );
        }
    };

    // The MRZ expiry is a YYMMDD field; expand it to ISO form using the
    // current century, because an expiry is always in the present or future.
    let expiry_iso = match tiger_kyc_engine::mrz_date_to_iso(&mrz.date_of_expiry, 2000) {
        Ok(d) => d,
        Err(e) => {
            return err(
                StatusCode::BAD_REQUEST,
                &format!("invalid expiry date: {e}"),
            )
        }
    };

    let watchlist = match state.store.load_watchlist().await {
        Ok(w) => w,
        Err(e) => {
            // Screening against an unreadable list would produce a false clear,
            // so the request fails instead.
            return err(
                StatusCode::SERVICE_UNAVAILABLE,
                &format!("screening lists are unavailable: {e}"),
            );
        }
    };

    let today = Utc::now().format("%Y-%m-%d").to_string();
    let applicant = Applicant {
        first_name: req.first_name.clone(),
        last_name: req.last_name.clone(),
        country: req.country.clone(),
        date_of_birth: req.date_of_birth.clone(),
        address: req.address.clone(),
        address_country: req.address_country.clone(),
    };

    let report = match verify(
        &mrz,
        &applicant,
        level,
        &Policy::default(),
        &watchlist,
        &expiry_iso,
        &today,
    ) {
        Ok(r) => r,
        Err(e) => {
            // A hard refusal (underage, expired document, unknown country).
            let _ = state
                .store
                .audit(
                    Some(req.user_id),
                    None,
                    None,
                    "submission_refused",
                    &serde_json::json!({"reason": e.to_string()}),
                )
                .await;
            return err(
                StatusCode::BAD_REQUEST,
                &format!("verification refused: {e}"),
            );
        }
    };

    let (status_str, reasons) = match &report.decision {
        Decision::Approved => (status::VERIFIED.to_string(), Vec::new()),
        Decision::Referred { reasons } => (status::PENDING_REVIEW.to_string(), reasons.clone()),
        Decision::Rejected { reasons } => (status::REJECTED.to_string(), reasons.clone()),
    };

    // The engine is deliberately dependency-free (no serde), so risk factors
    // are converted to JSON here rather than in the engine.
    let factors: serde_json::Value = serde_json::Value::Array(
        report
            .factors
            .iter()
            .map(|f| serde_json::json!({"reason": f.reason, "score": f.score}))
            .collect(),
    );

    let app = match state
        .store
        .upsert_application(
            req.user_id,
            req.level,
            &status_str,
            report.score as i32,
            &req.country.to_uppercase(),
            &reasons,
            &factors,
            tiger_kyc_engine::POLICY_VERSION,
        )
        .await
    {
        Ok(a) => a,
        Err(e) => {
            return err(
                StatusCode::INTERNAL_SERVER_ERROR,
                &format!("cannot record decision: {e}"),
            )
        }
    };

    let _ = state
        .store
        .audit(
            Some(req.user_id),
            Some(app.id),
            None,
            "verification_decided",
            &serde_json::json!({
                "status": status_str,
                "score": report.score,
                "country_tier": format!("{:?}", report.country_tier),
                "policy_version": tiger_kyc_engine::POLICY_VERSION,
            }),
        )
        .await;

    (
        StatusCode::OK,
        Json(StatusResponse {
            user_id: req.user_id,
            verified: app.status == status::VERIFIED,
            status: app.status,
            level: app.level,
            risk_score: app.risk_score,
            reasons: app.decision_reasons,
            policy_version: app.policy_version,
        }),
    )
        .into_response()
}

/// Returns a user's current verification status.
pub async fn status(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(user_id): Path<Uuid>,
) -> Response {
    if !is_service_authenticated(&state, &headers) {
        return err(StatusCode::UNAUTHORIZED, "invalid or missing service token");
    }

    match state.store.get_application(user_id).await {
        Ok(Some(app)) => (
            StatusCode::OK,
            Json(StatusResponse {
                user_id,
                verified: app.status == status::VERIFIED,
                status: app.status,
                level: app.level,
                risk_score: app.risk_score,
                reasons: app.decision_reasons,
                policy_version: app.policy_version,
            }),
        )
            .into_response(),
        Ok(None) => (
            StatusCode::OK,
            Json(StatusResponse {
                user_id,
                verified: false,
                status: status::NOT_STARTED.to_string(),
                level: 0,
                risk_score: 0,
                reasons: Vec::new(),
                policy_version: tiger_kyc_engine::POLICY_VERSION.to_string(),
            }),
        )
            .into_response(),
        Err(e) => err(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("cannot read status: {e}"),
        ),
    }
}

/// A reviewer's decision.
#[derive(Debug, Deserialize)]
pub struct ReviewRequest {
    pub reviewer_id: Uuid,
    /// `verified` or `rejected`.
    pub decision: String,
    #[serde(default)]
    pub reason: Option<String>,
}

/// Applies a human reviewer's decision.
pub async fn review(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(user_id): Path<Uuid>,
    Json(req): Json<ReviewRequest>,
) -> Response {
    if !is_reviewer_authenticated(&state, &headers) {
        return err(StatusCode::FORBIDDEN, "invalid or missing reviewer token");
    }

    let new_status = match req.decision.as_str() {
        "verified" => status::VERIFIED,
        "rejected" => status::REJECTED,
        other => {
            return err(
                StatusCode::BAD_REQUEST,
                &format!("decision {other:?} is not valid; use \"verified\" or \"rejected\""),
            )
        }
    };

    match state
        .store
        .set_reviewer_decision(user_id, new_status, req.reviewer_id, req.reason.as_deref())
        .await
    {
        Ok(Some(app)) => {
            let _ = state
                .store
                .audit(
                    Some(user_id),
                    Some(app.id),
                    Some(&req.reviewer_id.to_string()),
                    "reviewer_decision",
                    &serde_json::json!({"status": new_status, "reason": req.reason}),
                )
                .await;
            (
                StatusCode::OK,
                Json(StatusResponse {
                    user_id,
                    verified: app.status == status::VERIFIED,
                    status: app.status,
                    level: app.level,
                    risk_score: app.risk_score,
                    reasons: app.decision_reasons,
                    policy_version: app.policy_version,
                }),
            )
                .into_response()
        }
        Ok(None) => err(StatusCode::NOT_FOUND, "no application exists for that user"),
        Err(e) => err(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("cannot record decision: {e}"),
        ),
    }
}

/// Lists applications in a status, for the review queue.
pub async fn list_applications(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    axum::extract::Query(q): axum::extract::Query<std::collections::HashMap<String, String>>,
) -> Response {
    if !is_reviewer_authenticated(&state, &headers) {
        return err(StatusCode::FORBIDDEN, "invalid or missing reviewer token");
    }
    let wanted = q
        .get("status")
        .cloned()
        .unwrap_or_else(|| status::PENDING_REVIEW.to_string());
    let limit: i64 = q
        .get("limit")
        .and_then(|v| v.parse().ok())
        .unwrap_or(100)
        .min(500);

    match state.store.list_by_status(&wanted, limit).await {
        Ok(apps) => {
            let out: Vec<StatusResponse> = apps
                .iter()
                .map(|a| StatusResponse {
                    user_id: a.user_id,
                    verified: a.status == status::VERIFIED,
                    status: a.status.clone(),
                    level: a.level,
                    risk_score: a.risk_score,
                    reasons: a.decision_reasons.clone(),
                    policy_version: a.policy_version.clone(),
                })
                .collect();
            (StatusCode::OK, Json(out)).into_response()
        }
        Err(e) => err(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("cannot list applications: {e}"),
        ),
    }
}

/// A watchlist entry as supplied by an operator.
#[derive(Debug, Deserialize)]
pub struct WatchlistRequest {
    /// `sanctions`, `pep`, `adverse_media` or `internal_blocklist`.
    pub kind: String,
    pub name: String,
    #[serde(default)]
    pub country: Option<String>,
    #[serde(default)]
    pub date_of_birth: Option<String>,
    #[serde(default)]
    pub address: Option<String>,
    pub source: String,
}

/// Adds a watchlist entry.
pub async fn add_watchlist_entry(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(req): Json<WatchlistRequest>,
) -> Response {
    if !is_reviewer_authenticated(&state, &headers) {
        return err(StatusCode::FORBIDDEN, "invalid or missing reviewer token");
    }
    let kind = match req.kind.as_str() {
        "sanctions" => ListKind::Sanctions,
        "pep" => ListKind::Pep,
        "adverse_media" => ListKind::AdverseMedia,
        "internal_blocklist" => ListKind::InternalBlocklist,
        other => {
            return err(
                StatusCode::BAD_REQUEST,
                &format!("unknown list kind {other:?}"),
            )
        }
    };
    if req.name.trim().is_empty() {
        return err(StatusCode::BAD_REQUEST, "a watchlist entry needs a name");
    }

    let entry = WatchlistEntry {
        id: String::new(),
        kind,
        name: tiger_kyc_engine::normalise_name(&req.name),
        country: req.country,
        date_of_birth: req.date_of_birth,
        address: req.address,
        source: req.source,
    };

    match state.store.add_watchlist_entry(&entry).await {
        Ok(id) => (StatusCode::CREATED, Json(serde_json::json!({"id": id}))).into_response(),
        Err(e) => err(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("cannot add entry: {e}"),
        ),
    }
}

/// Removes a watchlist entry.
pub async fn remove_watchlist_entry(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    axum::extract::Query(q): axum::extract::Query<std::collections::HashMap<String, String>>,
) -> Response {
    if !is_reviewer_authenticated(&state, &headers) {
        return err(StatusCode::FORBIDDEN, "invalid or missing reviewer token");
    }
    let Some(id) = q.get("id").and_then(|v| Uuid::parse_str(v).ok()) else {
        return err(StatusCode::BAD_REQUEST, "a valid entry id is required");
    };
    match state.store.remove_watchlist_entry(id).await {
        Ok(true) => (StatusCode::OK, Json(serde_json::json!({"removed": true}))).into_response(),
        Ok(false) => err(StatusCode::NOT_FOUND, "no such watchlist entry"),
        Err(e) => err(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("cannot remove entry: {e}"),
        ),
    }
}

/// Builds an error response. Returning `Response` lets every handler branch
/// share one return type regardless of success or failure.
fn err(code: StatusCode, message: &str) -> Response {
    (code, Json(serde_json::json!({"error": message}))).into_response()
}
