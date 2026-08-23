//! rlp.rs — minimal RLP (Recursive Length Prefix) encoder.
//!
//! Sufficient for Ethereum transaction signing (EIP-1559 / legacy EIP-155).

/// RLP-encode a byte string.
pub fn encode_bytes(data: &[u8], out: &mut Vec<u8>) {
    if data.len() == 1 && data[0] < 0x80 {
        out.push(data[0]);
        return;
    }
    encode_header(0x80, data.len(), out);
    out.extend_from_slice(data);
}

/// RLP-encode a u64 as a minimal big-endian byte string (0 → empty string).
pub fn encode_u64(v: u64, out: &mut Vec<u8>) {
    encode_bytes(&minimal_be_u64(v), out);
}

/// RLP-encode an unsigned integer given as a minimal big-endian byte slice.
pub fn encode_uint_be(be: &[u8], out: &mut Vec<u8>) {
    let trimmed = trim_be(be);
    encode_bytes(trimmed, out);
}

/// Encode a list header + concatenated payload.
pub fn encode_list(payload: &[u8], out: &mut Vec<u8>) {
    encode_header(0xc0, payload.len(), out);
    out.extend_from_slice(payload);
}

fn encode_header(offset: u8, len: usize, out: &mut Vec<u8>) {
    if len <= 55 {
        out.push(offset + len as u8);
    } else {
        let len_bytes = minimal_be_usize(len);
        out.push(offset + 55 + len_bytes.len() as u8);
        out.extend_from_slice(&len_bytes);
    }
}

fn minimal_be_u64(v: u64) -> Vec<u8> {
    if v == 0 {
        return Vec::new();
    }
    let b = v.to_be_bytes();
    let start = b.iter().position(|&x| x != 0).unwrap_or(8);
    b[start..].to_vec()
}

fn minimal_be_usize(v: usize) -> Vec<u8> {
    minimal_be_u64(v as u64)
}

/// Trim leading zero bytes (big-endian canonical form).
pub fn trim_be(be: &[u8]) -> &[u8] {
    let start = be.iter().position(|&x| x != 0).unwrap_or(be.len());
    &be[start..]
}

#[cfg(test)]
mod tests {
    use super::*;

    fn enc_bytes(d: &[u8]) -> Vec<u8> {
        let mut o = Vec::new();
        encode_bytes(d, &mut o);
        o
    }

    #[test]
    fn rlp_known_vectors() {
        // dog
        assert_eq!(enc_bytes(b"dog"), vec![0x83, b'd', b'o', b'g']);
        // empty string
        assert_eq!(enc_bytes(b""), vec![0x80]);
        // single byte < 0x80 encodes as itself
        assert_eq!(enc_bytes(&[0x7f]), vec![0x7f]);
        // single byte >= 0x80 gets a header
        assert_eq!(enc_bytes(&[0x80]), vec![0x81, 0x80]);
        // 1024-byte string (long form)
        let big = vec![0u8; 1024];
        let e = enc_bytes(&big);
        assert_eq!(&e[..3], &[0xb9, 0x04, 0x00]);
        assert_eq!(e.len(), 3 + 1024);
    }

    #[test]
    fn rlp_integers() {
        let mut o = Vec::new();
        encode_u64(0, &mut o);
        assert_eq!(o, vec![0x80]);
        o.clear();
        encode_u64(15, &mut o);
        assert_eq!(o, vec![0x0f]);
        o.clear();
        encode_u64(1024, &mut o);
        assert_eq!(o, vec![0x82, 0x04, 0x00]);
        // 256-bit value
        o.clear();
        let mut v = [0u8; 32];
        v[31] = 1;
        encode_uint_be(&v, &mut o);
        assert_eq!(o, vec![0x01]);
    }

    #[test]
    fn rlp_list_vector() {
        // [] and [cat, dog]
        let mut o = Vec::new();
        encode_list(&[], &mut o);
        assert_eq!(o, vec![0xc0]);
        let mut payload = Vec::new();
        encode_bytes(b"cat", &mut payload);
        encode_bytes(b"dog", &mut payload);
        o.clear();
        encode_list(&payload, &mut o);
        assert_eq!(
            o,
            vec![0xc8, 0x83, b'c', b'a', b't', 0x83, b'd', b'o', b'g']
        );
    }
}
