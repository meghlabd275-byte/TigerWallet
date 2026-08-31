use blake2::{Blake2b512, Digest};

#[test]
fn debug_zero_vector() {
    let payload = [0u8; 33]; // prefix 0 + 32 zero bytes
    let mut hasher = Blake2b512::new();
    hasher.update(b"SS58PRE");
    hasher.update(payload);
    let hash = hasher.finalize();
    println!("checksum: {:02x}{:02x}", hash[0], hash[1]);
    assert_eq!(hash[0], 0xd4);
    assert_eq!(hash[1], 0xbe);
}
