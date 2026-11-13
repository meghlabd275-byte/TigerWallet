//! TigerWallet in-house KYC verification engine.
//!
//! This crate performs identity verification without any external vendor. It
//! validates machine-readable zones against the ICAO 9303 check digits, screens
//! the applicant against watchlists held in the operator's own database, and
//! produces a deterministic approve / refer / reject decision with a recorded
//! reason for every branch.
//!
//! The crate has no dependencies, which keeps the supply-chain surface at zero
//! for the component that decides who may trade.
//!
//! # What this crate does not do
//!
//! It does not read identity documents. Producing an MRZ string from a passport
//! photograph is an OCR problem and belongs in a separate component; this crate
//! consumes the MRZ text and is responsible for the part that can be proven
//! correct: the arithmetic and the policy. Feeding it text that did not come
//! from a real document is an OCR-layer failure, which is why the OCR layer must
//! return an explicit error rather than a guessed MRZ.
//!
//! It does not perform liveness detection for the same reason.

pub mod country;
pub mod decision;
pub mod errors;
pub mod mrz;
pub mod screening;

pub use country::{country_risk_tier, is_valid_country, RiskTier};
pub use decision::{
    age_at, risk_ceiling, verify, Applicant, Decision, Policy, RiskFactor, VerificationLevel,
    VerificationReport,
};
pub use errors::KycError;
pub use mrz::{check_digit, parse as parse_mrz, verify_check_digit, Mrz, MrzFormat, Sex};
pub use screening::{
    normalise_name, screen, screen_with_variants, ListKind, MatchStrength, ScreeningMatch,
    ScreeningOutcome, ScreeningSubject, WatchlistEntry,
};

/// The version of the verification policy this build implements.
///
/// It is recorded on every decision so that a decision can be reproduced
/// against the policy that produced it.
pub const POLICY_VERSION: &str = "2026.11.1";
