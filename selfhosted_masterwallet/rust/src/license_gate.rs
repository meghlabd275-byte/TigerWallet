//! license_gate.rs — fail-closed SuperAdmin license gate for the self-hosted
//! MasterWallet (Rust).
//!
//! Mirrors the pure-Go `wl_shared/go/wlgate` semantics and the C++
//! `wl_control_plane/cpp` WlGate hot-path checker. On startup the gate is
//! DEAD (fail-closed); it only becomes alive after a successful
//! `/api/v1/license/validate` against the TigerWallet control plane. It keeps
//! phoning home at a configurable interval and reverts to DEAD whenever the
//! control plane is unreachable, the license is invalid/expired/suspended, or
//! a kill-switch halt is returned.
//!
//! License: fail-closed. Not a single protected request is served without a
//! valid license validated against the control plane. This closes the P0 gap
//! that previously allowed this reference implementation to run unlicensed.

use serde::Deserialize;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;
use tracing::{error, info};

/// Control-plane validation response (subset we act on).
#[derive(Deserialize)]
struct ValidateResponse {
    #[serde(default)]
    valid: bool,
    #[serde(default)]
    alive: bool,
    #[serde(default)]
    reason: String,
    #[serde(default)]
    command: Option<String>,
}

/// The in-process license gate. `alive` starts false; it is flipped true only
/// by a successful control-plane validation and back to false on any failure.
pub struct LicenseGate {
    alive: AtomicBool,
    reason: std::sync::Mutex<String>,
}

impl LicenseGate {
    pub fn new() -> Self {
        Self {
            alive: AtomicBool::new(false),
            reason: std::sync::Mutex::new("license not yet validated (heartbeat pending)".into()),
        }
    }

    pub fn is_alive(&self) -> bool {
        self.alive.load(Ordering::Acquire)
    }

    pub fn reason(&self) -> String {
        self.reason.lock().unwrap().clone()
    }

    fn set_alive(&self, alive: bool, reason: &str) {
        self.alive.store(alive, Ordering::Release);
        *self.reason.lock().unwrap() = reason.to_string();
    }

    /// Perform a single validation call against the control plane and update
    /// the gate accordingly. Fail-closed: any error leaves/puts the gate dead.
    async fn validate_once(
        &self,
        client: &reqwest::Client,
        control_plane_url: &str,
        token: &str,
        license_key: &str,
        product: &str,
        instance_id: &str,
    ) {
        let url = format!("{control_plane_url}/api/v1/license/validate");
        let body = serde_json::json!({
            "license_key": license_key,
            "product": product,
            "instance_id": instance_id,
            "version": env!("CARGO_PKG_VERSION"),
        });

        let mut req = client.post(&url).json(&body);
        if !token.is_empty() {
            req = req.bearer_auth(token);
        }

        match req.send().await {
            Err(e) => {
                self.set_alive(false, &format!("control plane unreachable: {e}"));
            }
            Ok(resp) => {
                if resp.status().as_u16() != 200 {
                    self.set_alive(
                        false,
                        &format!("control plane rejected license (HTTP {})", resp.status().as_u16()),
                    );
                    return;
                }
                match resp.json::<ValidateResponse>().await {
                    Err(e) => {
                        self.set_alive(false, &format!("control plane response parse error: {e}"));
                    }
                    Ok(vr) => {
                        if !vr.valid || !vr.alive {
                            self.set_alive(false, &vr.reason);
                        } else if vr.command.as_deref() == Some("halt") {
                            self.set_alive(false, "control plane issued halt command");
                        } else {
                            self.set_alive(true, "");
                        }
                    }
                }
            }
        }
    }

    /// Run the heartbeat loop until the process exits. If the control plane URL
    /// is not configured the gate stays dead forever (fail-closed).
    pub async fn heartbeat_loop(
        gate: std::sync::Arc<Self>,
        control_plane_url: String,
        token: String,
        license_key: String,
        product: String,
        instance_id: String,
        interval: Duration,
    ) {
        if control_plane_url.is_empty() {
            gate.set_alive(false, "license control plane not configured (TWO_PARTY_GATE_URL unset)");
            error!("license control plane not configured; serving will remain disabled (fail-closed)");
            return;
        }
        if license_key.is_empty() {
            gate.set_alive(false, "license key not configured (WL_LICENSE_KEY unset)");
            error!("WL_LICENSE_KEY not configured; serving will remain disabled (fail-closed)");
            return;
        }

        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .expect("build http client");

        let mut ticker = tokio::time::interval(interval);
        // First tick fires immediately.
        loop {
            ticker.tick().await;
            LicenseGate::validate_once(
                &gate,
                &client,
                &control_plane_url,
                &token,
                &license_key,
                &product,
                &instance_id,
            )
            .await;
            if gate.is_alive() {
                info!("license heartbeat ok (product={product}, instance={instance_id})");
            }
        }
    }
}

impl Default for LicenseGate {
    fn default() -> Self {
        Self::new()
    }
}
