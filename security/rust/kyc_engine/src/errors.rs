//! Error types for the KYC engine.
//!
//! Every failure is explicit. A verification step that cannot be completed
//! returns an error; it never degrades to a pass.

use std::fmt;

/// Errors produced while parsing or verifying identity data.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum KycError {
    /// The machine-readable zone was empty.
    MrzEmpty,
    /// The MRZ did not have 2 (TD3) or 3 (TD1) lines.
    MrzUnexpectedLineCount { found: usize },
    /// An MRZ line exceeded its format's fixed width.
    MrzLineTooLong { found: usize, max: usize },
    /// A padded MRZ line did not reach its format's fixed width.
    MrzWrongLength {
        format: crate::mrz::MrzFormat,
        line1: usize,
        line2: usize,
    },
    /// An MRZ character outside the ICAO 9303 alphabet appeared.
    MrzInvalidCharacter { character: char },
    /// A check digit position held something other than a digit.
    MrzCheckDigitNotNumeric { field: String, found: char },
    /// A check digit did not match the value computed over its field.
    MrzCheckDigitMismatch {
        field: String,
        expected: u8,
        found: u8,
    },
    /// A date field was not six digits, or was not a real calendar date.
    MrzDateMalformed { field: String },
    /// The document number was empty after removing fillers.
    DocumentNumberEmpty,
    /// A name field was empty on both sides of the separator.
    NameEmpty,
    /// The supplied country code was not a recognised ISO 3166-1 alpha-3 code.
    UnknownCountryCode { code: String },
    /// The applicant is younger than the platform's minimum age.
    ApplicantUnderMinimumAge { age: u32, minimum: u32 },
    /// The identity document has expired.
    DocumentExpired { expiry: String },
    /// The identity document expires sooner than the platform allows.
    DocumentExpiringSoon { expiry: String, days: i64 },
    /// A field required for the requested verification level was absent.
    MissingField { field: &'static str },
}

impl fmt::Display for KycError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::MrzEmpty => write!(f, "the machine-readable zone is empty"),
            Self::MrzUnexpectedLineCount { found } => write!(
                f,
                "the machine-readable zone has {found} lines; expected 2 (passport) or 3 (identity card)"
            ),
            Self::MrzLineTooLong { found, max } => {
                write!(f, "machine-readable zone line is {found} characters; the maximum is {max}")
            }
            Self::MrzWrongLength { format, line1, line2 } => write!(
                f,
                "machine-readable zone is malformed for {format:?}: line lengths {line1} and {line2}"
            ),
            Self::MrzInvalidCharacter { character } => write!(
                f,
                "character {character:?} is not permitted in a machine-readable zone"
            ),
            Self::MrzCheckDigitNotNumeric { field, found } => write!(
                f,
                "the check digit for {field:?} is {found:?}; a digit was expected"
            ),
            Self::MrzCheckDigitMismatch { field, expected, found } => write!(
                f,
                "the check digit for {field:?} is {found} but the field computes to {expected}"
            ),
            Self::MrzDateMalformed { field } => {
                write!(f, "date field {field:?} is not a valid YYMMDD date")
            }
            Self::DocumentNumberEmpty => write!(f, "the document number is empty"),
            Self::NameEmpty => write!(f, "the name is empty"),
            Self::UnknownCountryCode { code } => {
                write!(f, "country code {code:?} is not a recognised ISO 3166-1 alpha-3 code")
            }
            Self::ApplicantUnderMinimumAge { age, minimum } => write!(
                f,
                "the applicant is {age} years old; the minimum age is {minimum}"
            ),
            Self::DocumentExpired { expiry } => {
                write!(f, "the identity document expired on {expiry}")
            }
            Self::DocumentExpiringSoon { expiry, days } => write!(
                f,
                "the identity document expires on {expiry}, within the required {days}-day window"
            ),
            Self::MissingField { field } => write!(f, "required field {field:?} is missing"),
        }
    }
}

impl std::error::Error for KycError {}
