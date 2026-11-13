//! The verification decision engine.
//!
//! This module turns validated identity data plus screening results into one of
//! three outcomes: approve, refer to a human reviewer, or reject. Every branch
//! is deterministic and every reason is recorded, so a decision can be
//! explained to a regulator after the fact.

use crate::country::{country_risk_tier, is_valid_country, RiskTier};
use crate::errors::KycError;
use crate::mrz::Mrz;
use crate::screening::{ScreeningOutcome, ScreeningSubject, WatchlistEntry};

/// The verification level an applicant is applying for. Higher levels require
/// more evidence and are held to a stricter risk ceiling.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum VerificationLevel {
    /// Email and phone only. No identity document.
    Basic,
    /// Identity document validated, for P2P trading and card issuance.
    Standard,
    /// Document plus proof of address and enhanced screening.
    Enhanced,
}

/// The platform's risk ceiling for a given level. A score at or above the
/// ceiling is referred rather than approved.
pub fn risk_ceiling(level: VerificationLevel) -> u32 {
    match level {
        VerificationLevel::Basic => 70,
        VerificationLevel::Standard => 60,
        VerificationLevel::Enhanced => 40,
    }
}

/// The outcome of a verification attempt.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Decision {
    /// All checks passed and the score is below the level's ceiling.
    Approved,
    /// A human must decide. The reasons explain why.
    Referred { reasons: Vec<String> },
    /// The application cannot proceed. The reasons are recorded.
    Rejected { reasons: Vec<String> },
}

impl Decision {
    /// The stable status string written to storage and read by callers.
    ///
    /// This vocabulary is the contract with the wallet backend, which gates P2P
    /// and card access on `verified`.
    pub fn status(&self) -> &'static str {
        match self {
            Self::Approved => "verified",
            Self::Referred { .. } => "pending_review",
            Self::Rejected { .. } => "rejected",
        }
    }
}

/// Applicant-supplied identity data.
#[derive(Debug, Clone)]
pub struct Applicant {
    pub first_name: String,
    pub last_name: String,
    pub country: String,
    pub date_of_birth: String,
    pub address: Option<String>,
    pub address_country: Option<String>,
}

/// The engine's configuration. The operator owns these policy values.
#[derive(Debug, Clone)]
pub struct Policy {
    pub minimum_age: u32,
    /// Days of remaining validity an identity document must have.
    pub minimum_document_validity_days: i64,
    /// Whether a PEP hit blocks approval at this level.
    pub pep_blocks_approval: bool,
    /// Whether any adverse-media hit blocks approval at this level.
    pub adverse_media_blocks_approval: bool,
}

impl Default for Policy {
    fn default() -> Self {
        Self {
            minimum_age: 18,
            minimum_document_validity_days: 0,
            pep_blocks_approval: false,
            adverse_media_blocks_approval: false,
        }
    }
}

/// One risk signal contributing to the final score.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RiskFactor {
    pub reason: String,
    pub score: u32,
}

/// The complete result of a verification run.
#[derive(Debug, Clone)]
pub struct VerificationReport {
    pub decision: Decision,
    pub score: u32,
    pub factors: Vec<RiskFactor>,
    pub screening: ScreeningOutcome,
    pub country_tier: RiskTier,
}

/// Computes age in whole years at `today` (YYYY-MM-DD) from a birth date
/// (YYYY-MM-DD).
pub fn age_at(date_of_birth: &str, today: &str) -> Result<u32, KycError> {
    let dob = parse_iso_date(date_of_birth)?;
    let now = parse_iso_date(today)?;
    if now < dob {
        return Err(KycError::MrzDateMalformed {
            field: date_of_birth.to_string(),
        });
    }
    let mut age = now.0 - dob.0;
    // Subtract a year if the birthday has not occurred yet this year.
    if (now.1, now.2) < (dob.1, dob.2) {
        age -= 1;
    }
    Ok(age)
}

/// Parses YYYY-MM-DD into (year, month, day).
fn parse_iso_date(s: &str) -> Result<(u32, u32, u32), KycError> {
    let malformed = || KycError::MrzDateMalformed {
        field: s.to_string(),
    };
    let parts: Vec<&str> = s.trim().split('-').collect();
    if parts.len() != 3 {
        return Err(malformed());
    }
    let y: u32 = parts[0].parse().map_err(|_| malformed())?;
    let m: u32 = parts[1].parse().map_err(|_| malformed())?;
    let d: u32 = parts[2].parse().map_err(|_| malformed())?;
    if !(1..=12).contains(&m) || !(1..=31).contains(&d) || !(1900..=2200).contains(&y) {
        return Err(malformed());
    }
    Ok((y, m, d))
}

/// Returns the number of days from `from` to `to`, both YYYY-MM-DD.
fn days_between(from: &str, to: &str) -> Result<i64, KycError> {
    let f = parse_iso_date(from)?;
    let t = parse_iso_date(to)?;
    Ok(days_from_civil(t) - days_from_civil(f))
}

/// Days since the Unix epoch for a civil date, using Howard Hinnant's
/// algorithm. Correct for the full proleptic Gregorian range.
fn days_from_civil(date: (u32, u32, u32)) -> i64 {
    let (y, m, d) = (date.0 as i64, date.1 as i64, date.2 as i64);
    let y = if m <= 2 { y - 1 } else { y };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = (m + 9) % 12;
    let doy = (153 * mp + 2) / 5 + d - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

/// Runs the full verification pipeline.
///
/// The MRZ must already have passed every ICAO 9303 check digit, which is
/// enforced by `Mrz`'s construction. `today` is passed in rather than read from
/// the clock so that decisions are reproducible and testable.
#[allow(clippy::too_many_arguments)]
pub fn verify(
    mrz: &Mrz,
    applicant: &Applicant,
    level: VerificationLevel,
    policy: &Policy,
    watchlists: &[WatchlistEntry],
    document_expiry_iso: &str,
    today: &str,
) -> Result<VerificationReport, KycError> {
    let mut factors: Vec<RiskFactor> = Vec::new();
    let mut rejections: Vec<String> = Vec::new();

    if mrz.document_number.is_empty() {
        return Err(KycError::DocumentNumberEmpty);
    }
    if mrz.surname.is_empty() && mrz.given_names.is_empty() {
        return Err(KycError::NameEmpty);
    }

    // --- jurisdiction ---
    let country = applicant.country.trim().to_uppercase();
    if !is_valid_country(&country) {
        return Err(KycError::UnknownCountryCode { code: country });
    }
    let tier = country_risk_tier(&country);
    let tier_pts = crate::screening::tier_score(tier);
    if tier_pts > 0 {
        factors.push(RiskFactor {
            reason: format!("jurisdiction {country} is {tier:?} risk"),
            score: tier_pts,
        });
    }
    if tier == RiskTier::High {
        rejections.push(format!(
            "jurisdiction {country} requires manual review before approval"
        ));
    }

    // --- age ---
    let dob_iso = crate::mrz::parse_mrz_date(&mrz.date_of_birth, 1900)?;
    let age = age_at(&dob_iso, today)?;
    if age < policy.minimum_age {
        return Err(KycError::ApplicantUnderMinimumAge {
            age,
            minimum: policy.minimum_age,
        });
    }
    if age < 21 {
        factors.push(RiskFactor {
            reason: format!("applicant is {age} years old"),
            score: 5,
        });
    }

    // --- document validity ---
    let expiry_days = days_between(today, document_expiry_iso)?;
    if expiry_days < 0 {
        return Err(KycError::DocumentExpired {
            expiry: document_expiry_iso.to_string(),
        });
    }
    if expiry_days < policy.minimum_document_validity_days {
        return Err(KycError::DocumentExpiringSoon {
            expiry: document_expiry_iso.to_string(),
            days: policy.minimum_document_validity_days,
        });
    }
    if expiry_days < 180 {
        factors.push(RiskFactor {
            reason: format!("document expires in {expiry_days} days"),
            score: 10,
        });
    }

    // --- self-declaration must agree with the document ---
    let declared = format!("{} {}", applicant.first_name, applicant.last_name);
    let declared_norm = crate::screening::normalise_name(&declared);
    let document_norm = mrz.normalised_name();
    if declared_norm != document_norm {
        // A mismatch is a hard stop: either the applicant mistyped their own
        // name or the document is not theirs.
        rejections.push(format!(
            "the declared name {declared_norm:?} does not match the document name {document_norm:?}"
        ));
    }

    // --- proof of address, required above the basic level ---
    if level >= VerificationLevel::Standard {
        match (&applicant.address, &applicant.address_country) {
            (Some(a), Some(c)) if !a.trim().is_empty() && is_valid_country(c) => {}
            _ => {
                rejections
                    .push("proof of address is required for this verification level".to_string());
            }
        }
    }

    // --- screening ---
    // The MRZ writes the surname first; sanctions and PEP lists usually write
    // the given name first. Both orders are screened, because comparing only
    // one produces false negatives against every list using the other.
    let given_first = format!("{} {}", mrz.given_names, mrz.surname);
    let surname_first = format!("{} {}", mrz.surname, mrz.given_names);
    let screening = crate::screening::screen_with_variants(
        &ScreeningSubject {
            name: surname_first.clone(),
            country: Some(country.clone()),
            date_of_birth: Some(dob_iso.clone()),
            address: applicant.address.clone(),
        },
        &[given_first, mrz.surname.clone(), mrz.given_names.clone()],
        watchlists,
    );

    if screening.sanctions_hit {
        // A sanctions hit is never auto-approved, and never auto-rejected
        // either: the operator must confirm the match and record the decision.
        rejections.push("a sanctions list match was found".to_string());
    }
    if screening.blocklist_hit {
        rejections.push("an internal blocklist match was found".to_string());
    }
    if screening.pep_hit {
        factors.push(RiskFactor {
            reason: "a politically exposed person match was found".to_string(),
            score: 35,
        });
        if policy.pep_blocks_approval {
            rejections.push(
                "a politically exposed person match blocks approval at this level".to_string(),
            );
        }
    }
    if screening.adverse_media_hit {
        factors.push(RiskFactor {
            reason: "an adverse media match was found".to_string(),
            score: 25,
        });
        if policy.adverse_media_blocks_approval {
            rejections.push("an adverse media match blocks approval at this level".to_string());
        }
    }

    let score: u32 = factors.iter().map(|f| f.score).sum();
    let ceiling = risk_ceiling(level);

    let decision = if !rejections.is_empty() {
        Decision::Rejected {
            reasons: rejections,
        }
    } else if score >= ceiling {
        Decision::Referred {
            reasons: vec![format!(
                "risk score {score} reaches the {level:?} level ceiling of {ceiling}"
            )],
        }
    } else {
        Decision::Approved
    };

    Ok(VerificationReport {
        decision,
        score,
        factors,
        screening,
        country_tier: tier,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::mrz;
    use crate::screening::ListKind;

    const SPECIMEN_MRZ: &str = "P<UTOERIKSSON<<ANNA<MARIA<<<<<<<<<<<<<<<<<<<\n\
                                L898902C36UTO7408122F1204159ZE184226B<<<<<10";

    fn applicant() -> Applicant {
        Applicant {
            first_name: "Anna Maria".to_string(),
            last_name: "Eriksson".to_string(),
            country: "SWE".to_string(),
            date_of_birth: "1974-08-12".to_string(),
            address: Some("1 Example Street".to_string()),
            address_country: Some("SWE".to_string()),
        }
    }

    fn run(
        mrz_doc: &Mrz,
        app: &Applicant,
        level: VerificationLevel,
        lists: &[WatchlistEntry],
    ) -> Result<VerificationReport, KycError> {
        verify(
            mrz_doc,
            app,
            level,
            &Policy::default(),
            lists,
            "2030-01-01",
            "2026-11-12",
        )
    }

    #[test]
    fn age_at_computes_whole_years() {
        assert_eq!(age_at("1990-06-15", "2026-11-12").unwrap(), 36);
        // Birthday not yet reached this year.
        assert_eq!(age_at("1990-12-15", "2026-11-12").unwrap(), 35);
        // Birthday is today.
        assert_eq!(age_at("1990-11-12", "2026-11-12").unwrap(), 36);
    }

    #[test]
    fn age_at_rejects_future_birth_dates() {
        assert!(age_at("2030-01-01", "2026-11-12").is_err());
    }

    #[test]
    fn days_between_handles_month_and_year_boundaries() {
        assert_eq!(days_between("2026-01-01", "2026-01-01").unwrap(), 0);
        assert_eq!(days_between("2026-01-01", "2026-01-02").unwrap(), 1);
        assert_eq!(days_between("2026-01-31", "2026-02-01").unwrap(), 1);
        assert_eq!(days_between("2025-12-31", "2026-01-01").unwrap(), 1);
        assert_eq!(days_between("2026-01-01", "2025-01-01").unwrap(), -365);
    }

    #[test]
    fn clean_applicant_is_approved() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let report = run(&doc, &applicant(), VerificationLevel::Standard, &[]).unwrap();
        assert_eq!(report.decision, Decision::Approved);
        assert_eq!(report.decision.status(), "verified");
    }

    /// This is the defect being fixed: a real, valid application must be able
    /// to reach `verified`, which nothing in the previous implementation did.
    #[test]
    fn approved_decision_yields_the_verified_status_p2p_requires() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let report = run(&doc, &applicant(), VerificationLevel::Standard, &[]).unwrap();
        assert_eq!(report.decision.status(), "verified");
    }

    #[test]
    fn underage_applicant_is_rejected_with_an_error() {
        // The specimen was born in 1974, so it is not underage. Build an
        // applicant whose declared birth date would be underage by verifying
        // against a policy with a minimum age of 60.
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let strict = Policy {
            minimum_age: 60,
            ..Policy::default()
        };
        let err = verify(
            &doc,
            &applicant(),
            VerificationLevel::Standard,
            &strict,
            &[],
            "2030-01-01",
            "2026-11-12",
        )
        .unwrap_err();
        assert!(matches!(err, KycError::ApplicantUnderMinimumAge { .. }));
    }

    #[test]
    fn expired_document_is_rejected() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let err = verify(
            &doc,
            &applicant(),
            VerificationLevel::Standard,
            &Policy::default(),
            &[],
            "2020-01-01",
            "2026-11-12",
        )
        .unwrap_err();
        assert!(matches!(err, KycError::DocumentExpired { .. }));
    }

    #[test]
    fn name_mismatch_is_rejected() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let mut app = applicant();
        app.last_name = "SOMEONEELSE".to_string();
        let report = run(&doc, &app, VerificationLevel::Standard, &[]).unwrap();
        assert!(matches!(report.decision, Decision::Rejected { .. }));
        assert_eq!(report.decision.status(), "rejected");
    }

    #[test]
    fn invalid_country_is_rejected() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let mut app = applicant();
        app.country = "ZZZ".to_string();
        let err = run(&doc, &app, VerificationLevel::Standard, &[]).unwrap_err();
        assert!(matches!(err, KycError::UnknownCountryCode { .. }));
    }

    #[test]
    fn sanctions_hit_is_rejected_not_approved() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let lists = vec![WatchlistEntry {
            id: "s1".to_string(),
            kind: ListKind::Sanctions,
            name: "ERIKSSON ANNA MARIA".to_string(),
            country: Some("SWE".to_string()),
            date_of_birth: Some("1974-08-12".to_string()),
            address: None,
            source: "TEST".to_string(),
        }];
        let report = run(&doc, &applicant(), VerificationLevel::Standard, &lists).unwrap();
        assert!(report.screening.sanctions_hit);
        assert!(matches!(report.decision, Decision::Rejected { .. }));
    }

    #[test]
    fn high_risk_jurisdiction_is_referred_not_approved() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let mut app = applicant();
        app.country = "PRK".to_string();
        app.address_country = Some("PRK".to_string());
        let report = run(&doc, &app, VerificationLevel::Standard, &[]).unwrap();
        assert_eq!(report.country_tier, RiskTier::High);
        assert!(matches!(report.decision, Decision::Rejected { .. }));
    }

    #[test]
    fn pep_hit_raises_the_score() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let lists = vec![WatchlistEntry {
            id: "p1".to_string(),
            kind: ListKind::Pep,
            name: "ERIKSSON ANNA MARIA".to_string(),
            country: None,
            date_of_birth: None,
            address: None,
            source: "TEST".to_string(),
        }];
        let report = run(&doc, &applicant(), VerificationLevel::Standard, &lists).unwrap();
        assert!(report.screening.pep_hit);
        assert!(
            report.score >= 35,
            "a PEP hit must add at least 35 to the score"
        );
    }

    #[test]
    fn missing_proof_of_address_rejects_standard_level() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let mut app = applicant();
        app.address = None;
        app.address_country = None;
        let report = run(&doc, &app, VerificationLevel::Standard, &[]).unwrap();
        assert!(matches!(report.decision, Decision::Rejected { .. }));
    }

    #[test]
    fn basic_level_does_not_require_proof_of_address() {
        let doc = mrz::parse(SPECIMEN_MRZ).unwrap();
        let mut app = applicant();
        app.address = None;
        app.address_country = None;
        let report = run(&doc, &app, VerificationLevel::Basic, &[]).unwrap();
        assert_eq!(report.decision, Decision::Approved);
    }

    #[test]
    fn risk_ceilings_tighten_with_level() {
        assert!(risk_ceiling(VerificationLevel::Basic) > risk_ceiling(VerificationLevel::Standard));
        assert!(
            risk_ceiling(VerificationLevel::Standard) > risk_ceiling(VerificationLevel::Enhanced)
        );
    }

    #[test]
    fn decision_status_vocabulary_matches_the_backend_contract() {
        assert_eq!(Decision::Approved.status(), "verified");
        assert_eq!(
            Decision::Referred { reasons: vec![] }.status(),
            "pending_review"
        );
        assert_eq!(Decision::Rejected { reasons: vec![] }.status(), "rejected");
    }
}
