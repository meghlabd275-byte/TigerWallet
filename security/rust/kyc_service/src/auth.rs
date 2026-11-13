//! Request authentication.
//!
//! Two separate tokens are used. The service token authorises submission and
//! status reads. The reviewer token authorises decisions and watchlist changes.
//! Holding the service token therefore does not confer the ability to approve
//! an application.
//!
//! Comparison is constant-time. A byte-by-byte comparison leaks the token
//! prefix through timing, which turns a secret check into an oracle.

use axum::http::HeaderMap;

use crate::AppState;

/// The header carrying the service token.
pub const SERVICE_HEADER: &str = "x-kyc-token";
/// The header carrying the reviewer token.
pub const REVIEWER_HEADER: &str = "x-kyc-reviewer-token";

/// Returns true when the request carries the service token.
pub fn is_service_authenticated(state: &AppState, headers: &HeaderMap) -> bool {
    header_matches(headers, SERVICE_HEADER, &state.api_token)
}

/// Returns true when the request carries the reviewer token.
pub fn is_reviewer_authenticated(state: &AppState, headers: &HeaderMap) -> bool {
    header_matches(headers, REVIEWER_HEADER, &state.reviewer_token)
}

fn header_matches(headers: &HeaderMap, name: &str, expected: &str) -> bool {
    let Some(value) = headers.get(name).and_then(|v| v.to_str().ok()) else {
        return false;
    };
    constant_time_eq(value.as_bytes(), expected.as_bytes())
}

/// Compares two byte slices without early exit.
///
/// The length is compared first, which leaks only the length and not the
/// contents. Every byte is then always examined.
pub fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    let mut diff: u8 = 0;
    for (x, y) in a.iter().zip(b.iter()) {
        diff |= x ^ y;
    }
    diff == 0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn equal_slices_match() {
        assert!(constant_time_eq(b"secret", b"secret"));
    }

    #[test]
    fn different_slices_do_not_match() {
        assert!(!constant_time_eq(b"secret", b"secrez"));
        assert!(!constant_time_eq(b"secret", b"secre"));
        assert!(!constant_time_eq(b"", b"x"));
    }

    #[test]
    fn empty_slices_match() {
        assert!(constant_time_eq(b"", b""));
    }

    #[test]
    fn single_bit_difference_is_detected() {
        assert!(!constant_time_eq(b"\x00", b"\x01"));
        assert!(!constant_time_eq(b"\xff", b"\xfe"));
    }
}
