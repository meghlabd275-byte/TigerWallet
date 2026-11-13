//! ISO 3166-1 alpha-3 country reference data and jurisdiction risk tiers.
//!
//! The code list is the authoritative ISO 3166-1 alpha-3 set. The risk tiers
//! encode which jurisdictions this platform treats as higher risk; that
//! classification is a policy decision the operator owns, so the tiers live in
//! one place and every score is derived from them rather than scattered
//! through the codebase.

/// Every recognised ISO 3166-1 alpha-3 country code.
///
/// Codes are stored upper case. An unknown code is a verification failure
/// rather than a silent low-risk default, because a forged or malformed
/// country would otherwise pass screening.
pub const ISO_3166_ALPHA3: &[&str] = &[
    "ABW", "AFG", "AGO", "AIA", "ALA", "ALB", "AND", "ARE", "ARG", "ARM", "ASM", "ATA", "ATF",
    "ATG", "AUS", "AUT", "AZE", "BDI", "BEL", "BEN", "BES", "BFA", "BGD", "BGR", "BHR", "BHS",
    "BIH", "BLM", "BLR", "BLZ", "BMU", "BOL", "BRA", "BRB", "BRN", "BTN", "BVT", "BWA", "CAF",
    "CAN", "CCK", "CHE", "CHL", "CHN", "CIV", "CMR", "COD", "COG", "COK", "COL", "COM", "CPV",
    "CRI", "CUB", "CUW", "CXR", "CYM", "CYP", "CZE", "DEU", "DJI", "DMA", "DNK", "DOM", "DZA",
    "ECU", "EGY", "ERI", "ESH", "ESP", "EST", "ETH", "FIN", "FJI", "FLK", "FRA", "FRO", "FSM",
    "GAB", "GBR", "GEO", "GGY", "GHA", "GIB", "GIN", "GLP", "GMB", "GNB", "GNQ", "GRC", "GRD",
    "GRL", "GTM", "GUF", "GUM", "GUY", "HKG", "HMD", "HND", "HRV", "HTI", "HUN", "IDN", "IMN",
    "IND", "IOT", "IRL", "IRN", "IRQ", "ISL", "ISR", "ITA", "JAM", "JEY", "JOR", "JPN", "KAZ",
    "KEN", "KGZ", "KHM", "KIR", "KNA", "KOR", "KWT", "LAO", "LBN", "LBR", "LBY", "LCA", "LIE",
    "LKA", "LSO", "LTU", "LUX", "LVA", "MAC", "MAF", "MAR", "MCO", "MDA", "MDG", "MDV", "MEX",
    "MHL", "MKD", "MLI", "MLT", "MMR", "MNE", "MNG", "MNP", "MOZ", "MRT", "MSR", "MTQ", "MUS",
    "MWI", "MYS", "MYT", "NAM", "NCL", "NER", "NFK", "NGA", "NIC", "NIU", "NLD", "NOR", "NPL",
    "NRU", "NZL", "OMN", "PAK", "PAN", "PCN", "PER", "PHL", "PLW", "PNG", "POL", "PRI", "PRK",
    "PRT", "PRY", "PSE", "PYF", "QAT", "REU", "ROU", "RUS", "RWA", "SAU", "SDN", "SEN", "SGP",
    "SGS", "SHN", "SJM", "SLB", "SLE", "SLV", "SMR", "SOM", "SPM", "SRB", "SSD", "STP", "SUR",
    "SVK", "SVN", "SWE", "SWZ", "SXM", "SYC", "SYR", "TCA", "TCD", "TGO", "THA", "TJK", "TKL",
    "TKM", "TLS", "TON", "TTO", "TUN", "TUR", "TUV", "TWN", "TZA", "UGA", "UKR", "UMI", "URY",
    "USA", "UZB", "VAT", "VCT", "VEN", "VGB", "VIR", "VNM", "VUT", "WLF", "WSM", "YEM", "ZAF",
    "ZMB", "ZWE",
];

/// Jurisdiction risk tier, which drives the base risk score.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum RiskTier {
    /// Standard-risk jurisdiction.
    Low,
    /// Jurisdiction carrying additional monitoring obligations.
    Medium,
    /// Jurisdiction subject to comprehensive sanctions or a call for enhanced
    /// due diligence. Applications are not auto-rejected; they are routed to
    /// manual review with the reason recorded.
    High,
}

/// Returns true when `code` is a recognised ISO 3166-1 alpha-3 code.
pub fn is_valid_country(code: &str) -> bool {
    let upper = code.trim().to_uppercase();
    ISO_3166_ALPHA3.contains(&upper.as_str())
}

/// Classifies a country code into a risk tier.
///
/// An unrecognised code is treated as high risk: it means the input could not
/// be validated, and defaulting an unvalidated country to low risk would let a
/// malformed document through screening.
pub fn country_risk_tier(code: &str) -> RiskTier {
    let upper = code.trim().to_uppercase();

    // Jurisdictions subject to comprehensive sanctions or designated as
    // jurisdictions of primary money-laundering concern by the platform's
    // policy. These route to manual review rather than auto-approval.
    const HIGH: &[&str] = &[
        "AFG", "BLR", "CUB", "IRN", "IRQ", "KP", "PRK", "MMR", "SYR", "VEN", "YEM", "LBY", "SOM",
        "SSD", "SDN", "MLI", "CAF", "COD", "ZWE", "HTI", "NGA",
    ];
    // Jurisdictions requiring enhanced due diligence under the platform's
    // policy.
    const MEDIUM: &[&str] = &[
        "ALB", "DZA", "AGO", "BHS", "BWA", "KHM", "CYM", "GHA", "GIB", "JAM", "JOR", "KGZ", "LBN",
        "MAR", "MCO", "MOZ", "NAM", "PAK", "PAN", "PHL", "RUS", "SEN", "TUN", "TUR", "UGA", "UKR",
        "VNM", "ARE", "SAU", "QAT", "KWT", "BHR", "OMN", "BRB", "TTO", "VCT", "LCA", "GRD", "KNA",
        "ATG", "DMA", "SXM", "CUW", "ABW", "BES", "MHL", "PLW", "NRU",
    ];

    if HIGH.contains(&upper.as_str()) {
        RiskTier::High
    } else if MEDIUM.contains(&upper.as_str()) {
        RiskTier::Medium
    } else if is_valid_country(&upper) {
        RiskTier::Low
    } else {
        RiskTier::High
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn recognises_common_countries() {
        for c in ["USA", "GBR", "DEU", "ARE", "SGP", "JPN", "NGA"] {
            assert!(is_valid_country(c), "{c} must be recognised");
        }
    }

    #[test]
    fn is_case_insensitive_and_trims() {
        assert!(is_valid_country(" usa "));
        assert!(is_valid_country("deu"));
    }

    #[test]
    fn rejects_unknown_codes() {
        for c in ["XXX", "ZZZ", "", "US", "USAA", "123"] {
            assert!(!is_valid_country(c), "{c} must not be recognised");
        }
    }

    #[test]
    fn unknown_country_is_high_risk() {
        assert_eq!(country_risk_tier("XXX"), RiskTier::High);
    }

    #[test]
    fn sanctions_jurisdictions_are_high_risk() {
        for c in ["PRK", "IRN", "SYR", "CUB"] {
            assert_eq!(
                country_risk_tier(c),
                RiskTier::High,
                "{c} must be high risk"
            );
        }
    }

    #[test]
    fn standard_jurisdictions_are_low_risk() {
        for c in ["USA", "GBR", "DEU", "JPN", "SGP"] {
            assert_eq!(country_risk_tier(c), RiskTier::Low, "{c} must be low risk");
        }
    }

    #[test]
    fn enhanced_due_diligence_jurisdictions_are_medium_risk() {
        for c in ["PAN", "PHL", "TUR"] {
            assert_eq!(
                country_risk_tier(c),
                RiskTier::Medium,
                "{c} must be medium risk"
            );
        }
    }

    #[test]
    fn country_list_has_no_duplicates() {
        let mut seen = std::collections::HashSet::new();
        for c in ISO_3166_ALPHA3 {
            assert!(seen.insert(*c), "duplicate country code {c}");
        }
    }

    #[test]
    fn country_list_is_all_upper_case_alpha3() {
        for c in ISO_3166_ALPHA3 {
            assert_eq!(c.len(), 3, "{c} is not three characters");
            assert!(
                c.bytes().all(|b| b.is_ascii_uppercase()),
                "{c} is not upper case ASCII"
            );
        }
    }
}
