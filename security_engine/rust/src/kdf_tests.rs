use super::security::{derive_key_from_password, PBKDF2_ITERATIONS};

fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{:02x}", b)).collect()
}

#[test]
fn the_iteration_count_matches_the_browser_wallet() {
    // The browser uses 100000 iterations; a mismatch would break interoperability.
    assert_eq!(PBKDF2_ITERATIONS, 100_000);
}

#[test]
fn derivation_matches_the_pbkdf2_reference_vectors() {
    // Vectors computed with Python's hashlib.pbkdf2_hmac("sha256", ...).
    let key = derive_key_from_password("correct horse battery staple", b"tigerwallet-salt");
    assert_eq!(
        hex(&key),
        "c72fdfacf149381f721f69ba40849d3bb30db2d2b9c2fa55ea46cc9f9861698f"
    );
}

#[test]
fn derivation_handles_empty_inputs() {
    let key = derive_key_from_password("", b"");
    assert_eq!(
        hex(&key),
        "64a868d4b23af696d3734d0b814d04cdd1ac280128e97653a05f32b49c13a29a"
    );
}

#[test]
fn derivation_hashes_unicode_passwords_as_utf8() {
    let key = derive_key_from_password("pässwörd", b"x");
    assert_eq!(
        hex(&key),
        "6cbd05b5c2d72e23405d976609cdc8407a7e64aeb30f6f3ca6d190e57e55ff5c"
    );
}

#[test]
fn the_salt_changes_the_key() {
    let a = derive_key_from_password("password", b"salt-a");
    let b = derive_key_from_password("password", b"salt-b");
    assert_ne!(a, b, "two different salts produced the same key");
}

#[test]
fn the_password_changes_the_key() {
    let a = derive_key_from_password("password-a", b"salt");
    let b = derive_key_from_password("password-b", b"salt");
    assert_ne!(a, b, "two different passwords produced the same key");
}

#[test]
fn derivation_is_deterministic() {
    let a = derive_key_from_password("password", b"salt");
    let b = derive_key_from_password("password", b"salt");
    assert_eq!(a, b, "the same inputs must always give the same key");
}

#[test]
fn a_near_miss_password_does_not_collide() {
    // The old XOR-folding implementation folded the password into 32 bytes and
    // ignored everything past the first 32, so long passwords collided. That
    // must not happen with a real KDF.
    let base = "a".repeat(64);
    let other = format!("{}b", "a".repeat(63));

    let a = derive_key_from_password(&base, b"salt");
    let b = derive_key_from_password(&other, b"salt");
    assert_ne!(a, b, "two distinct long passwords derived the same key");
}

#[test]
fn the_key_is_not_the_password_repeated() {
    // A sanity check that the output is not a trivially guessable pattern.
    let key = derive_key_from_password("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", b"");
    let repeated = [b'a'; 32];
    assert_ne!(key, repeated, "the derived key echoed the password");
}
