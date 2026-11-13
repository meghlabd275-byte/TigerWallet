//! Local screening against sanctions, PEP and adverse-media watchlists.
//!
//! This module is deliberately self-contained: it screens against watchlists
//! loaded into the operator's own database and performs no network calls. That
//! satisfies the requirement that identity verification not depend on an
//! external vendor, and it means a vendor outage cannot silently clear a
//! customer.
//!
//! The matching is exact and normalised rather than fuzzy. Fuzzy name matching
//! produces false negatives at a rate that makes it unusable as a control, so
//! candidates are surfaced for human adjudication instead of being auto-cleared
//! or auto-rejected.

use crate::country::RiskTier;

/// The kind of list a watchlist entry came from.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ListKind {
    /// Sanctions list (asset freezes, travel bans).
    Sanctions,
    /// Politically exposed person.
    Pep,
    /// Adverse media.
    AdverseMedia,
    /// Internal blocklist maintained by the operator.
    InternalBlocklist,
}

/// One entry in a screening watchlist.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WatchlistEntry {
    pub id: String,
    pub kind: ListKind,
    /// Normalised full name, upper case, single-spaced.
    pub name: String,
    /// Optional ISO 3166-1 alpha-3 country of the listed party.
    pub country: Option<String>,
    /// Optional date of birth in YYYY-MM-DD form.
    pub date_of_birth: Option<String>,
    /// Optional wallet address, for on-chain screening.
    pub address: Option<String>,
    /// Which list this entry came from, for the audit record.
    pub source: String,
}

/// The subject being screened.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ScreeningSubject {
    /// Normalised full name, upper case, single-spaced.
    pub name: String,
    pub country: Option<String>,
    pub date_of_birth: Option<String>,
    pub address: Option<String>,
}

/// How strongly a watchlist entry matched the subject.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MatchStrength {
    /// Name and at least one other attribute matched.
    Confirmed,
    /// Name and country matched.
    Strong,
    /// Name matched but nothing else did, so a human must adjudicate.
    NameOnly,
}

/// A single match found during screening.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ScreeningMatch {
    pub entry_id: String,
    pub kind: ListKind,
    pub strength: MatchStrength,
    pub source: String,
}

/// The outcome of a screening run.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ScreeningOutcome {
    pub matches: Vec<ScreeningMatch>,
    /// True when at least one sanctions entry matched.
    pub sanctions_hit: bool,
    /// True when at least one PEP entry matched.
    pub pep_hit: bool,
    /// True when at least one adverse-media entry matched.
    pub adverse_media_hit: bool,
    /// True when the operator's own blocklist matched.
    pub blocklist_hit: bool,
}

impl ScreeningOutcome {
    /// True when any match was found at all.
    pub fn has_matches(&self) -> bool {
        !self.matches.is_empty()
    }

    /// True when the outcome must be routed to a human reviewer.
    ///
    /// Any match at all requires review. An exact sanctions hit also requires
    /// review rather than automatic rejection, because the operator must be
    /// able to see the reason and record its decision.
    pub fn requires_review(&self) -> bool {
        self.has_matches()
    }
}

/// Normalises a name for comparison: upper case, filler characters removed,
/// punctuation dropped, whitespace collapsed.
pub fn normalise_name(name: &str) -> String {
    name.replace('<', " ")
        .chars()
        .map(|c| {
            if c.is_alphanumeric() || c == ' ' {
                c
            } else {
                ' '
            }
        })
        .collect::<String>()
        .split_whitespace()
        .collect::<Vec<_>>()
        .join(" ")
        .to_uppercase()
}

/// Screens a subject against the supplied watchlists.
///
/// `entries` is the full watchlist set to screen against. The caller is
/// responsible for loading it from the database; this function performs no I/O
/// so it stays deterministic and testable.
pub fn screen(subject: &ScreeningSubject, entries: &[WatchlistEntry]) -> ScreeningOutcome {
    screen_with_variants(subject, &[], entries)
}

/// Screens a subject against watchlists, also trying the supplied alternative
/// name forms.
///
/// Name order varies by source: a machine-readable zone writes the surname
/// first (`ERIKSSON ANNA MARIA`) while sanctions and PEP lists usually write
/// the given name first (`ANNA MARIA ERIKSSON`). Comparing only one order
/// produces false negatives on every list that uses the other, so the caller
/// passes the alternative forms it considers equivalent and each is screened.
///
/// This is deliberately not fuzzy matching. Only whole-name forms the caller
/// explicitly declares as equivalent are compared, so the match set stays
/// explainable and a reviewer can see exactly which form matched.
pub fn screen_with_variants(
    subject: &ScreeningSubject,
    alternative_names: &[String],
    entries: &[WatchlistEntry],
) -> ScreeningOutcome {
    let mut name_forms: Vec<String> = Vec::with_capacity(alternative_names.len() + 1);
    let primary = normalise_name(&subject.name);
    if !primary.is_empty() {
        name_forms.push(primary);
    }
    for alt in alternative_names {
        let n = normalise_name(alt);
        if !n.is_empty() && !name_forms.contains(&n) {
            name_forms.push(n);
        }
    }

    let subject_country = subject.country.as_ref().map(|c| c.trim().to_uppercase());
    let subject_dob = subject.date_of_birth.clone();
    let subject_address = subject.address.as_ref().map(|a| a.trim().to_lowercase());

    let mut matches: Vec<ScreeningMatch> = Vec::new();

    for entry in entries {
        let entry_name = normalise_name(&entry.name);
        let entry_country = entry.country.as_ref().map(|c| c.trim().to_uppercase());

        // Address screening is exact and independent of the name, because an
        // on-chain blocklist is keyed by address.
        let address_match = match (&subject_address, &entry.address) {
            (Some(sa), Some(ea)) => *sa == ea.trim().to_lowercase(),
            _ => false,
        };

        let name_match = !entry_name.is_empty() && name_forms.iter().any(|f| *f == entry_name);
        let country_match = match (&subject_country, &entry_country) {
            (Some(sc), Some(ec)) => sc == ec,
            _ => false,
        };
        let dob_match = match (&subject_dob, &entry.date_of_birth) {
            (Some(sd), Some(ed)) => sd == ed,
            _ => false,
        };

        let strength = if address_match && !name_match {
            // An address match is exact by nature and does not depend on a
            // name, so it is treated as confirmed.
            MatchStrength::Confirmed
        } else if name_match && dob_match {
            MatchStrength::Confirmed
        } else if name_match && country_match {
            MatchStrength::Strong
        } else if name_match {
            MatchStrength::NameOnly
        } else {
            continue;
        };

        matches.push(ScreeningMatch {
            entry_id: entry.id.clone(),
            kind: entry.kind,
            strength,
            source: entry.source.clone(),
        });
    }

    ScreeningOutcome {
        sanctions_hit: matches.iter().any(|m| m.kind == ListKind::Sanctions),
        pep_hit: matches.iter().any(|m| m.kind == ListKind::Pep),
        adverse_media_hit: matches.iter().any(|m| m.kind == ListKind::AdverseMedia),
        blocklist_hit: matches
            .iter()
            .any(|m| m.kind == ListKind::InternalBlocklist),
        matches,
    }
}

/// Maps a jurisdiction tier to the numeric risk contribution it adds.
pub fn tier_score(tier: RiskTier) -> u32 {
    match tier {
        RiskTier::Low => 0,
        RiskTier::Medium => 20,
        RiskTier::High => 40,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn entry(id: &str, kind: ListKind, name: &str) -> WatchlistEntry {
        WatchlistEntry {
            id: id.to_string(),
            kind,
            name: name.to_string(),
            country: None,
            date_of_birth: None,
            address: None,
            source: "TEST".to_string(),
        }
    }

    fn subject(name: &str) -> ScreeningSubject {
        ScreeningSubject {
            name: name.to_string(),
            country: None,
            date_of_birth: None,
            address: None,
        }
    }

    #[test]
    fn normalise_name_collapses_and_upper_cases() {
        assert_eq!(
            normalise_name("  eriksson<<anna   maria "),
            "ERIKSSON ANNA MARIA"
        );
        assert_eq!(normalise_name("O'Brien-Smith"), "O BRIEN SMITH");
        assert_eq!(normalise_name(""), "");
    }

    #[test]
    fn clean_subject_produces_no_matches() {
        let entries = vec![entry("1", ListKind::Sanctions, "IVAN SOKOLOV")];
        let out = screen(&subject("ANNA ERIKSSON"), &entries);
        assert!(!out.has_matches());
        assert!(!out.requires_review());
        assert!(!out.sanctions_hit);
    }

    #[test]
    fn exact_sanctions_name_match_is_reported() {
        let entries = vec![entry("1", ListKind::Sanctions, "IVAN SOKOLOV")];
        let out = screen(&subject("Ivan Sokolov"), &entries);
        assert!(out.sanctions_hit);
        assert!(out.requires_review());
        assert_eq!(out.matches.len(), 1);
        assert_eq!(out.matches[0].kind, ListKind::Sanctions);
    }

    #[test]
    fn name_and_dob_match_is_confirmed() {
        let mut e = entry("1", ListKind::Sanctions, "IVAN SOKOLOV");
        e.date_of_birth = Some("1970-01-01".to_string());
        let mut s = subject("IVAN SOKOLOV");
        s.date_of_birth = Some("1970-01-01".to_string());
        let out = screen(&s, &[e]);
        assert_eq!(out.matches[0].strength, MatchStrength::Confirmed);
    }

    #[test]
    fn name_and_country_match_is_strong() {
        let mut e = entry("1", ListKind::Pep, "IVAN SOKOLOV");
        e.country = Some("RUS".to_string());
        let mut s = subject("IVAN SOKOLOV");
        s.country = Some("rus".to_string());
        let out = screen(&s, &[e]);
        assert_eq!(out.matches[0].strength, MatchStrength::Strong);
        assert!(out.pep_hit);
    }

    /// A name-only hit must never be treated as a confirmed identity.
    #[test]
    fn name_only_match_is_flagged_for_review_not_confirmed() {
        let entries = vec![entry("1", ListKind::Sanctions, "IVAN SOKOLOV")];
        let out = screen(&subject("IVAN SOKOLOV"), &entries);
        assert_eq!(out.matches[0].strength, MatchStrength::NameOnly);
        assert!(
            out.requires_review(),
            "a name-only hit still needs a human decision"
        );
    }

    /// A country mismatch must not suppress a name hit, because sanctions
    /// evaders change jurisdiction.
    #[test]
    fn country_mismatch_does_not_suppress_name_match() {
        let mut e = entry("1", ListKind::Sanctions, "IVAN SOKOLOV");
        e.country = Some("RUS".to_string());
        let mut s = subject("IVAN SOKOLOV");
        s.country = Some("USA".to_string());
        let out = screen(&s, &[e]);
        assert!(out.has_matches());
        assert_eq!(out.matches[0].strength, MatchStrength::NameOnly);
    }

    #[test]
    fn address_match_is_confirmed() {
        let mut e = entry("1", ListKind::InternalBlocklist, "SOMETHING ELSE");
        e.address = Some("0xAbCdEf1234567890".to_string());
        let mut s = subject("ANY NAME");
        s.address = Some("0xabcdef1234567890".to_string());
        let out = screen(&s, &[e]);
        assert!(out.blocklist_hit);
        assert_eq!(out.matches[0].strength, MatchStrength::Confirmed);
    }

    #[test]
    fn empty_subject_name_never_matches_by_name() {
        let entries = vec![entry("1", ListKind::Sanctions, "IVAN SOKOLOV")];
        let out = screen(&subject(""), &entries);
        assert!(
            !out.has_matches(),
            "an empty subject name must not match a name entry"
        );
    }

    #[test]
    fn empty_watchlist_clears_nothing_falsely() {
        let out = screen(&subject("ANYONE"), &[]);
        assert!(!out.has_matches());
        assert!(!out.requires_review());
    }

    #[test]
    fn multiple_list_kinds_are_reported_independently() {
        let entries = vec![
            entry("1", ListKind::Sanctions, "IVAN SOKOLOV"),
            entry("2", ListKind::Pep, "IVAN SOKOLOV"),
            entry("3", ListKind::AdverseMedia, "IVAN SOKOLOV"),
        ];
        let out = screen(&subject("IVAN SOKOLOV"), &entries);
        assert!(out.sanctions_hit);
        assert!(out.pep_hit);
        assert!(out.adverse_media_hit);
        assert_eq!(out.matches.len(), 3);
    }

    #[test]
    fn tier_scores_are_ordered() {
        assert!(tier_score(RiskTier::Low) < tier_score(RiskTier::Medium));
        assert!(tier_score(RiskTier::Medium) < tier_score(RiskTier::High));
    }
}
