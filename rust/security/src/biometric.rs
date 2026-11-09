//! TigerWallet Biometric Authentication
//! 
//! This module provides biometric authentication for the wallet:
//! - Face ID (iOS)
//! - Touch ID (iOS/macOS)
//! - Fingerprint (Android)
//! - Windows Hello
//! 
//! Security:
//! - Hardware-backed biometric storage
//! - Liveness detection
//! - Anti-replay protection
//! - Session management

use std::sync::Mutex;

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Biometric errors
#[derive(Error, Debug, Clone)]
pub enum BiometricError {
    #[error("Biometric not available: {0}")]
    NotAvailable(String),
    
    #[error("Authentication failed: {0}")]
    AuthFailed(String),
    
    #[error("Enrollment failed: {0}")]
    EnrollmentFailed(String),
    
    #[error("Hardware error: {0}")]
    HardwareError(String),
    
    #[error("Not enrolled")]
    NotEnrolled,
}

/// Biometric type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum BiometricType {
    None = 0,
    Fingerprint = 1,
    FaceID = 2,
    Iris = 4,
    Voice = 8,
    WindowsHello = 16,
}

/// Authentication level
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum AuthLevel {
    /// Any available biometric
    Any,
    /// Strong (Fingerprint, Face ID, Windows Hello)
    Strong,
    /// Weak (fallback)
    Weak,
}

/// Biometric enrollment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Enrollment {
    /// Enrollment ID
    pub id: String,
    
    /// Biometric type used
    pub biometric_type: BiometricType,
    
    /// Created at
    pub created_at: u64,
    
    /// Last used
    pub last_used: u64,
}

/// Biometric authentication request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthRequest {
    /// Reason for authentication
    pub reason: String,
    
    /// Maximum time to wait (seconds)
    pub timeout: u32,
    
    /// Required biometric type
    pub biometric_type: BiometricType,
    
    /// Fallback to device passcode
    pub fallback_to_passcode: bool,
}

/// Biometric authentication result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthResult {
    /// Success
    pub success: bool,
    
    /// Error message (if failed)
    pub error: Option<String>,
    
    /// Authentication timestamp
    pub timestamp: u64,
    
    /// Credential ID (for key retrieval)
    pub credential_id: Option<String>,
}

/// Platform-specific implementations
///
/// The Rust core cannot itself drive a biometric ceremony: the prompt, the
/// sensor, and the result are owned by the operating system and must be
/// requested by the host application (LocalAuthentication on iOS,
/// BiometricPrompt on Android, the Windows Hello API on Windows). What the core
/// can do is refuse to trust anything that is not a verified assertion.
///
/// So the platform layer implements [`BiometricVerifier`], returns a
/// [`BiometricAssertion`] produced by the operating system, and the core
/// checks that assertion against the session policy in
/// [`BiometricService`]. When no verifier is installed, every authentication
/// fails closed instead of reporting a success that never happened.
#[cfg(target_os = "ios")]
pub mod ios {
    use super::*;

    /// Biometric kind reported by LocalAuthentication.
    pub fn available_type() -> BiometricType {
        BiometricType::FaceID
    }
}

#[cfg(target_os = "android")]
pub mod android {
    use super::*;

    /// Biometric kind reported by BiometricManager.
    pub fn available_type() -> BiometricType {
        BiometricType::Fingerprint
    }
}

#[cfg(target_os = "windows")]
pub mod windows {
    use super::*;

    /// Biometric kind reported by the Windows Hello API.
    pub fn available_type() -> BiometricType {
        BiometricType::WindowsHello
    }
}

/// An assertion returned by the platform biometric API.
///
/// `verified` is only ever set by the host application after the operating
/// system has confirmed the user, and `credential_id` identifies the key that
/// the platform released. The core treats any assertion with
/// `verified == false`, an empty credential id, or a stale timestamp as a
/// failure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BiometricAssertion {
    /// Whether the operating system confirmed the user.
    pub verified: bool,

    /// Kind of biometric that was used.
    pub biometric_type: BiometricType,

    /// Identifier of the credential the platform unlocked.
    pub credential_id: String,

    /// Unix timestamp at which the platform completed the ceremony.
    pub timestamp: u64,
}

/// A platform hook that performs the biometric ceremony.
pub trait BiometricVerifier: Send + Sync {
    /// Ask the operating system to confirm the user.
    fn verify(&self, request: &AuthRequest) -> Result<BiometricAssertion, BiometricError>;

    /// Whether a usable biometric is enrolled on this device.
    fn is_enrolled(&self) -> bool;

    /// Biometric kinds enrolled on this device.
    fn enrolled_types(&self) -> Vec<BiometricType>;
}

/// Cross-platform biometric service.
///
/// Owns the session policy: how long a successful authentication is valid for,
/// how many failed attempts are tolerated before the wallet locks, and which
/// biometric kinds are acceptable. Verification itself is delegated to the
/// installed [`BiometricVerifier`].
pub struct BiometricService {
    verifier: Option<Box<dyn BiometricVerifier>>,

    /// Accepted biometric kinds. Empty means any kind the platform reports.
    accepted_types: Vec<BiometricType>,

    /// How long a successful authentication stays valid, in seconds.
    session_timeout: u64,

    /// Failed attempts tolerated before the service locks.
    max_attempts: u8,

    /// Consecutive failed attempts, and the lockout deadline once the limit is
    /// reached. Held behind a lock because `authenticate` takes `&self`: the
    /// host shares one service across threads.
    failures: Mutex<FailureState>,

    /// When the current session expires, if one is open.
    session_expires_at: Mutex<Option<u64>>,
}

/// Failure bookkeeping for the lockout policy.
struct FailureState {
    attempts: u8,
    locked_until: Option<u64>,
}

/// How long a lockout lasts, in seconds.
const LOCKOUT_SECONDS: u64 = 60;

/// Clock skew tolerated on an assertion timestamp, in seconds.
const ASSERTION_SKEW_SECONDS: u64 = 5;

impl BiometricService {
    /// Create a service with no verifier installed.
    ///
    /// Until [`BiometricService::set_verifier`] is called, authentication fails
    /// closed.
    pub fn new() -> Self {
        Self {
            verifier: None,
            accepted_types: Vec::new(),
            session_timeout: 300,
            max_attempts: 3,
            failures: Mutex::new(FailureState {
                attempts: 0,
                locked_until: None,
            }),
            session_expires_at: Mutex::new(None),
        }
    }

    /// Install the platform verifier.
    pub fn set_verifier(&mut self, verifier: Box<dyn BiometricVerifier>) {
        self.verifier = Some(verifier);
    }

    /// Restrict the accepted biometric kinds.
    pub fn set_accepted_types(&mut self, types: Vec<BiometricType>) {
        self.accepted_types = types;
    }

    /// Set the session lifetime in seconds. Zero is rejected: a session that
    /// expires the moment it opens is never useful.
    pub fn set_session_timeout(&mut self, seconds: u64) -> Result<(), BiometricError> {
        if seconds == 0 {
            return Err(BiometricError::AuthFailed(
                "session timeout must be greater than zero".to_string(),
            ));
        }
        self.session_timeout = seconds;
        Ok(())
    }

    /// Set how many consecutive failures are tolerated.
    pub fn set_max_attempts(&mut self, attempts: u8) -> Result<(), BiometricError> {
        if attempts == 0 {
            return Err(BiometricError::AuthFailed(
                "max attempts must be greater than zero".to_string(),
            ));
        }
        self.max_attempts = attempts;
        Ok(())
    }

    /// Whether a usable biometric is enrolled.
    pub fn is_available(&self) -> bool {
        self.verifier
            .as_ref()
            .map(|verifier| verifier.is_enrolled())
            .unwrap_or(false)
    }

    /// Biometric kinds available on this device.
    pub fn available_types(&self) -> Vec<BiometricType> {
        self.verifier
            .as_ref()
            .map(|verifier| verifier.enrolled_types())
            .unwrap_or_default()
    }

    /// Whether the service is currently locked out.
    pub fn is_locked_out(&self, now: u64) -> bool {
        let failures = self.failures.lock().unwrap();
        matches!(failures.locked_until, Some(until) if now < until)
    }

    /// Whether an authenticated session is open.
    pub fn has_valid_session(&self, now: u64) -> bool {
        let expires_at = self.session_expires_at.lock().unwrap();
        matches!(*expires_at, Some(until) if now < until)
    }

    /// Record a failed attempt and lock out once the limit is reached.
    fn record_failure(&self, now: u64) {
        let mut failures = self.failures.lock().unwrap();
        failures.attempts = failures.attempts.saturating_add(1);

        if failures.attempts >= self.max_attempts {
            failures.locked_until = Some(now.saturating_add(LOCKOUT_SECONDS));
            failures.attempts = 0;
        }
    }

    /// Clear the failure count and open a session.
    fn record_success(&self, now: u64) {
        let mut failures = self.failures.lock().unwrap();
        failures.attempts = 0;
        failures.locked_until = None;

        let mut expires_at = self.session_expires_at.lock().unwrap();
        *expires_at = Some(now.saturating_add(self.session_timeout));
    }

    /// Whether the platform reports an enrollment.
    pub fn is_enrolled(&self) -> bool {
        self.is_available()
    }

    /// Authenticate through the installed verifier.
    ///
    /// Every rejection is counted; the service locks out once the failure count
    /// reaches `max_attempts`. A verified assertion opens a session for
    /// `session_timeout` seconds and clears the failure count.
    pub fn authenticate(&self, request: &AuthRequest) -> Result<AuthResult, BiometricError> {
        self.authenticate_at(request, current_timestamp())
    }

    /// Authenticate with an explicit clock, so the policy is testable.
    pub fn authenticate_at(
        &self,
        request: &AuthRequest,
        now: u64,
    ) -> Result<AuthResult, BiometricError> {
        if self.is_locked_out(now) {
            return Err(BiometricError::AuthFailed(
                "too many failed attempts; the wallet is temporarily locked".to_string(),
            ));
        }

        let outcome = self.attempt(request, now);
        match outcome {
            Ok(result) => {
                self.record_success(now);
                Ok(result)
            }
            Err(error) => {
                self.record_failure(now);
                Err(error)
            }
        }
    }

    /// Run one authentication attempt, without touching the failure counters.
    fn attempt(&self, request: &AuthRequest, now: u64) -> Result<AuthResult, BiometricError> {
        let verifier = self
            .verifier
            .as_ref()
            .ok_or_else(|| BiometricError::NotAvailable(
                "no platform biometric verifier is installed".to_string(),
            ))?;

        if !verifier.is_enrolled() {
            return Err(BiometricError::NotEnrolled);
        }

        if !self.accepted_types.is_empty() && !self.accepted_types.contains(&request.biometric_type)
        {
            return Err(BiometricError::AuthFailed(format!(
                "biometric kind {:?} is not accepted by this wallet",
                request.biometric_type
            )));
        }

        let assertion = verifier.verify(request)?;
        self.validate(&assertion, now)
    }

    /// Apply the session policy to an assertion from the platform.
    fn validate(
        &self,
        assertion: &BiometricAssertion,
        now: u64,
    ) -> Result<AuthResult, BiometricError> {
        if !assertion.verified {
            return Err(BiometricError::AuthFailed(
                "the platform did not confirm the user".to_string(),
            ));
        }

        if assertion.credential_id.trim().is_empty() {
            return Err(BiometricError::AuthFailed(
                "the platform returned no credential".to_string(),
            ));
        }

        if assertion.timestamp > now + ASSERTION_SKEW_SECONDS {
            return Err(BiometricError::AuthFailed(
                "the assertion is dated in the future".to_string(),
            ));
        }

        if now.saturating_sub(assertion.timestamp) > ASSERTION_SKEW_SECONDS {
            return Err(BiometricError::AuthFailed(
                "the assertion is stale and was rejected as a replay".to_string(),
            ));
        }

        Ok(AuthResult {
            success: true,
            error: None,
            timestamp: now,
            credential_id: Some(assertion.credential_id.clone()),
        })
    }
}

/// A verifier that always refuses.
///
/// Used in tests and as the default for hosts that have not yet wired their
/// platform API, so an unwired host cannot silently accept.
pub struct DenyAllVerifier;

impl BiometricVerifier for DenyAllVerifier {
    fn verify(&self, _request: &AuthRequest) -> Result<BiometricAssertion, BiometricError> {
        Err(BiometricError::NotAvailable(
            "this host has no biometric verifier installed".to_string(),
        ))
    }

    fn is_enrolled(&self) -> bool {
        false
    }

    fn enrolled_types(&self) -> Vec<BiometricType> {
        Vec::new()
    }
}

impl Default for BiometricService {
    fn default() -> Self {
        Self::new()
    }
}

fn current_timestamp() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|duration| duration.as_secs())
        .unwrap_or(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A verifier that replays a canned assertion, so the service policy can be
    /// exercised without a device.
    struct StubVerifier {
        enrolled: bool,
        assertion: Result<BiometricAssertion, BiometricError>,
    }

    impl BiometricVerifier for StubVerifier {
        fn verify(&self, _request: &AuthRequest) -> Result<BiometricAssertion, BiometricError> {
            self.assertion.clone()
        }

        fn is_enrolled(&self) -> bool {
            self.enrolled
        }

        fn enrolled_types(&self) -> Vec<BiometricType> {
            vec![BiometricType::Fingerprint]
        }
    }

    fn request() -> AuthRequest {
        AuthRequest {
            reason: "unlock".to_string(),
            timeout: 30,
            biometric_type: BiometricType::Fingerprint,
            fallback_to_passcode: false,
        }
    }

    fn verified_at(timestamp: u64) -> BiometricAssertion {
        BiometricAssertion {
            verified: true,
            biometric_type: BiometricType::Fingerprint,
            credential_id: "cred-1".to_string(),
            timestamp,
        }
    }

    fn service_with(assertion: Result<BiometricAssertion, BiometricError>) -> BiometricService {
        let mut service = BiometricService::new();
        service.set_verifier(Box::new(StubVerifier {
            enrolled: true,
            assertion,
        }));
        service
    }

    #[test]
    fn test_auth_request() {
        let request = request();
        assert_eq!(request.timeout, 30);
        assert_eq!(request.biometric_type, BiometricType::Fingerprint);
    }

    #[test]
    fn without_a_verifier_authentication_fails_closed() {
        let service = BiometricService::new();

        assert!(!service.is_available());
        assert!(service.available_types().is_empty());
        assert!(matches!(
            service.authenticate(&request()),
            Err(BiometricError::NotAvailable(_))
        ));
    }

    #[test]
    fn a_verified_assertion_is_accepted() {
        let service = service_with(Ok(verified_at(1_000)));

        let result = service.authenticate_at(&request(), 1_000).unwrap();
        assert!(result.success);
        assert!(result.error.is_none());
        assert_eq!(result.credential_id.as_deref(), Some("cred-1"));
    }

    #[test]
    fn an_unverified_assertion_is_rejected() {
        let service = service_with(Ok(BiometricAssertion {
            verified: false,
            ..verified_at(1_000)
        }));

        assert!(matches!(
            service.authenticate_at(&request(), 1_000),
            Err(BiometricError::AuthFailed(_))
        ));
    }

    #[test]
    fn an_assertion_without_a_credential_is_rejected() {
        let service = service_with(Ok(BiometricAssertion {
            credential_id: String::new(),
            ..verified_at(1_000)
        }));

        assert!(service.authenticate_at(&request(), 1_000).is_err());
    }

    #[test]
    fn a_replayed_or_future_assertion_is_rejected() {
        let service = service_with(Ok(verified_at(1_000)));

        // Long after the ceremony: a replay.
        assert!(service.authenticate_at(&request(), 5_000).is_err());

        // Before the ceremony: a forged timestamp.
        assert!(service.authenticate_at(&request(), 10).is_err());

        // Within the tolerated skew: accepted.
        assert!(service.authenticate_at(&request(), 1_003).is_ok());
    }

    #[test]
    fn a_denied_ceremony_is_reported_not_swallowed() {
        let service = service_with(Err(BiometricError::AuthFailed("user cancelled".to_string())));

        assert!(matches!(
            service.authenticate_at(&request(), 1_000),
            Err(BiometricError::AuthFailed(_))
        ));
    }

    #[test]
    fn an_unenrolled_device_reports_not_enrolled() {
        let mut service = BiometricService::new();
        service.set_verifier(Box::new(StubVerifier {
            enrolled: false,
            assertion: Ok(verified_at(1_000)),
        }));

        assert!(!service.is_available());
        assert!(matches!(
            service.authenticate_at(&request(), 1_000),
            Err(BiometricError::NotEnrolled)
        ));
    }

    #[test]
    fn a_kind_outside_the_accepted_list_is_rejected() {
        let mut service = service_with(Ok(verified_at(1_000)));
        service.set_accepted_types(vec![BiometricType::FaceID]);

        assert!(service.authenticate_at(&request(), 1_000).is_err());
    }

    #[test]
    fn session_and_lockout_settings_reject_zero() {
        let mut service = BiometricService::new();

        assert!(service.set_session_timeout(0).is_err());
        assert!(service.set_max_attempts(0).is_err());

        service.set_session_timeout(60).unwrap();
        assert!(!service.has_valid_session(0));
        assert!(!service.is_locked_out(0));
    }

    #[test]
    fn repeated_failures_lock_the_service_out() {
        let mut service = service_with(Ok(BiometricAssertion {
            verified: false,
            ..verified_at(1_000)
        }));
        service.set_max_attempts(2).unwrap();

        assert!(service.authenticate_at(&request(), 1_000).is_err());
        assert!(service.authenticate_at(&request(), 1_000).is_err());
        assert!(service.is_locked_out(1_000));
    }

    #[test]
    fn the_deny_all_verifier_refuses_everything() {
        let verifier = DenyAllVerifier;

        assert!(!verifier.is_enrolled());
        assert!(verifier.enrolled_types().is_empty());
        assert!(verifier.verify(&request()).is_err());
    }
}
