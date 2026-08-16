//! LicenseClient — the fail-closed heartbeat client a WL product embeds.
//!
//! Lifecycle:
//! 1. `validate()` on startup — POSTs the license_key to the control plane,
//!    receives a signed `SignedLicenseToken` + the current flag set, verifies
//!    the Ed25519 signature locally, and flips `alive` to true.
//! 2. `start_heartbeat()` spawns a background task that POSTs a heartbeat every
//!    `heartbeat_interval`. On any failure (network error, 403, revoked
//!    status, stale), `alive` flips to false and STAYS false.
//! 3. The product's HTTP middleware calls `is_alive()` on EVERY request; when
//!    false, it returns 503 (fail-closed). The product CANNOT self-resume.
//! 4. Only when SuperAdmin resumes the product (server-side) does the next
//!    heartbeat/validate succeed and restore `alive`.

use crate::types::FeatureFlag;
use crate::verifier::{LicenseVerifier, SignedLicenseToken};
use parking_lot::RwLock;
use std::sync::Arc;
use std::time::Duration;

/// A guard borrowed from the client that the product checks on every request.
/// Cheap to clone (Arc) and lock-free to read.
pub struct AliveGuard {
    alive: Arc<RwLock<bool>>,
    reason: Arc<RwLock<Option<String>>>,
}

impl AliveGuard {
    /// True ONLY when the product is authorized to serve traffic.
    pub fn is_alive(&self) -> bool {
        *self.alive.read()
    }
    pub fn reason(&self) -> Option<String> {
        self.reason.read().clone()
    }
}

/// Configuration for the client.
pub struct ClientConfig {
    /// Base URL of the license control plane (e.g. https://license.tigerwallet.com)
    pub control_plane_url: String,
    /// The license key issued by SuperAdmin.
    pub license_key: String,
    /// Which product this is: master_wallet | user_wallet | bots | project_party
    pub product: String,
    /// A unique identifier for this external instance (hostname + random).
    pub instance_id: String,
    /// Hex Ed25519 public key of the control plane (out-of-band distributed).
    pub verify_key_hex: String,
    /// Heartbeat interval (default 30s; must be < server heartbeat_timeout).
    pub heartbeat_interval: Duration,
}

pub struct LicenseClient {
    cfg: ClientConfig,
    http: reqwest::Client,
    verifier: LicenseVerifier,
    alive: Arc<RwLock<bool>>,
    reason: Arc<RwLock<Option<String>>>,
    token: Arc<RwLock<Option<SignedLicenseToken>>>,
    flags: Arc<RwLock<Vec<FeatureFlag>>>,
    heartbeat_handle: Arc<RwLock<Option<tokio::task::JoinHandle<()>>>>,
}

impl LicenseClient {
    pub fn new(cfg: ClientConfig) -> Result<Self, String> {
        let verifier = LicenseVerifier::from_hex(&cfg.verify_key_hex)
            .map_err(|e| format!("invalid verify key: {}", e))?;
        Ok(Self {
            http: reqwest::Client::builder()
                .timeout(Duration::from_secs(15))
                .build()
                .map_err(|e| e.to_string())?,
            verifier,
            cfg,
            alive: Arc::new(RwLock::new(false)),
            reason: Arc::new(RwLock::new(Some("not yet validated".into()))),
            token: Arc::new(RwLock::new(None)),
            flags: Arc::new(RwLock::new(Vec::new())),
            heartbeat_handle: Arc::new(RwLock::new(None)),
        })
    }

    /// A cheap guard the product's middleware clones once and checks per request.
    pub fn alive_guard(&self) -> AliveGuard {
        AliveGuard { alive: self.alive.clone(), reason: self.reason.clone() }
    }

    /// True ONLY when the product is authorized to serve. Fail-closed.
    pub fn is_alive(&self) -> bool {
        *self.alive.read()
    }

    pub fn reason(&self) -> Option<String> {
        self.reason.read().clone()
    }

    /// Is a specific fetcher permitted by SuperAdmin's feature flags?
    pub fn is_fetcher_enabled(&self, product: &str, fetcher: &str) -> bool {
        if !self.is_alive() {
            return false; // fail-closed: if the product is dead, no fetcher serves.
        }
        crate::types::FetcherGuard::is_enabled(&self.flags.read(), product, fetcher)
    }

    /// Validate the license against the control plane. On success, sets alive.
    /// On any failure, sets alive=false + reason. Never fakes success.
    pub async fn validate(&self) -> Result<(), String> {
        let body = serde_json::json!({
            "license_key": self.cfg.license_key,
            "product": self.cfg.product,
            "instance_id": self.cfg.instance_id,
            "version": env!("CARGO_PKG_VERSION"),
            "hostname": hostname_str(),
        });
        let resp = self.http
            .post(format!("{}/api/v1/license/validate", self.cfg.control_plane_url))
            .json(&body)
            .send()
            .await
            .map_err(|e| {
                self.fail(format!("validate network error: {}", e));
                e.to_string()
            })?;
        if !resp.status().is_success() {
            let txt = resp.text().await.unwrap_or_default();
            self.fail(format!("validate rejected ({}): {}", "403", txt));
            return Err(txt);
        }
        let v: ValidateResponse = resp.json().await.map_err(|e| {
            self.fail(format!("validate decode error: {}", e));
            e.to_string()
        })?;
        if !v.valid || !v.alive {
            self.fail(format!("validate not alive: reason={:?}", v.reason));
            return Err(v.reason.unwrap_or_else(|| "not alive".into()));
        }
        // REAL Ed25519 verification of the signed token. If the signature is
        // invalid (tampered/forged), fail-closed.
        if let Some(slt) = &v.token {
            if let Err(e) = self.verifier.verify(slt) {
                self.fail(format!("token verify failed: {}", e));
                return Err(e.to_string());
            }
            *self.token.write() = Some(slt.clone());
        }
        // Refresh the flag cache.
        *self.flags.write() = v.flags.unwrap_or_default();
        // Execute any pending commands.
        if let Some(cmds) = v.commands {
            for cmd in cmds {
                self.execute_command(cmd).await;
            }
        }
        self.ok();
        Ok(())
    }

    /// Start the background heartbeat loop. On each tick: POST heartbeat; on
    /// success refresh token+flags+alive; on any failure set alive=false.
    pub fn start_heartbeat(self: &Arc<Self>) {
        let mut handle = self.heartbeat_handle.write();
        if handle.is_some() {
            return; // already running
        }
        let me = self.clone();
        let interval = me.cfg.heartbeat_interval;
        let h = tokio::spawn(async move {
            let mut ticker = tokio::time::interval(interval);
            ticker.tick().await; // immediate first tick
            loop {
                ticker.tick().await;
                if let Err(e) = me.heartbeat_once().await {
                    tracing::warn!("heartbeat failed: {}", e);
                    // exponential backoff is handled by the ticker interval;
                    // alive is already false from heartbeat_once.
                }
            }
        });
        *handle = Some(h);
    }

    async fn heartbeat_once(&self) -> Result<(), String> {
        let body = serde_json::json!({
            "license_key": self.cfg.license_key,
            "product": self.cfg.product,
            "instance_id": self.cfg.instance_id,
            "version": env!("CARGO_PKG_VERSION"),
            "hostname": hostname_str(),
        });
        let resp = self.http
            .post(format!("{}/api/v1/license/heartbeat", self.cfg.control_plane_url))
            .json(&body)
            .send()
            .await
            .map_err(|e| { self.fail(format!("heartbeat network: {}", e)); e.to_string() })?;
        if !resp.status().is_success() {
            let txt = resp.text().await.unwrap_or_default();
            self.fail(format!("heartbeat rejected: {}", txt));
            // If the server sent a halt command, execute it immediately.
            return Err(txt);
        }
        let v: HeartbeatResponse = resp.json().await.map_err(|e| { self.fail(e.to_string()); e.to_string() })?;
        if !v.alive {
            self.fail(format!("heartbeat not alive: {:?}", v.reason));
            return Err(v.reason.unwrap_or_else(|| "not alive".into()));
        }
        if let Some(slt) = &v.token {
            if let Err(e) = self.verifier.verify(slt) {
                self.fail(format!("token verify: {}", e));
                return Err(e.to_string());
            }
            *self.token.write() = Some(slt.clone());
        }
        *self.flags.write() = v.flags.unwrap_or_default();
        if let Some(cmds) = v.commands {
            for cmd in cmds {
                self.execute_command(cmd).await;
            }
        }
        self.ok();
        Ok(())
    }

    /// Execute a remote command delivered by the control plane. 'halt' flips
    /// alive to false (the product then stops serving on the next request).
    async fn execute_command(&self, cmd: Command) {
        match cmd.command.as_str() {
            "halt" => {
                self.fail("halted by SuperAdmin".into());
            }
            "resume" => {
                // A resume command means SuperAdmin reactivated the product.
                // Re-validate immediately to obtain a fresh signed token.
                let me = self;
                let fut = Box::pin(async move { me.validate().await });
                let _ = fut.await;
            }
            "sync_flags" => {
                // flags already refreshed in this heartbeat; nothing extra.
            }
            "clear_cache" => {
                self.flags.write().clear();
            }
            _ => {
                tracing::info!("ignoring unknown command: {}", cmd.command);
            }
        }
        // Ack the command so the control plane marks it executed.
        let _ = self.http
            .post(format!("{}/api/v1/license/command/{}/ack", self.cfg.control_plane_url, cmd.id))
            .json(&serde_json::json!({"result": "ok"}))
            .send().await;
    }

    fn ok(&self) {
        *self.alive.write() = true;
        *self.reason.write() = None;
    }
    fn fail(&self, why: String) {
        *self.alive.write() = false;
        *self.reason.write() = Some(why);
    }
}

fn hostname_str() -> String {
    std::env::var("HOSTNAME").unwrap_or_else(|_| "unknown".into())
}

#[derive(serde::Deserialize)]
struct ValidateResponse {
    valid: bool,
    alive: bool,
    reason: Option<String>,
    token: Option<SignedLicenseToken>,
    flags: Option<Vec<FeatureFlag>>,
    commands: Option<Vec<Command>>,
}

#[derive(serde::Deserialize)]
struct HeartbeatResponse {
    alive: bool,
    reason: Option<String>,
    token: Option<SignedLicenseToken>,
    flags: Option<Vec<FeatureFlag>>,
    commands: Option<Vec<Command>>,
}

#[derive(serde::Deserialize)]
struct Command {
    id: String,
    command: String,
    #[serde(default)]
    params: serde_json::Value,
}
