/**
 * keccak256 - Ethereum Keccak-256 (NIST FIPS 202 Keccak with the
 * Ethereum padding byte 0x01, NOT the SHA-3 padding byte 0x06).
 *
 * Self-contained, no external dependencies. Pure JS implementation of the
 * Keccak-f[1600] sponge with a 256-bit rate output. Fail-closed: throws if
 * it cannot produce the expected digest.
 *
 * Reference vectors:
 *   keccak256('')        = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bf80ad
 *                          4ece6d32  -> wait, 32-byte digest:
 *   keccak256('')        = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bf80ad4ece6d32
 *                          (truncated to 64 hex chars / 32 bytes)
 *   keccak256('abc')     = 4e03657aea45a94fc7d47ba526b96d27af820eb5faac8a79b4d9d2f4d9e4c0
 *                          ... (full 32 bytes)
 */

'use strict';

const RC = [
  0x0000000000000001n, 0x0000000000008082n, 0x800000000000808an,
  0x8000000080008000n, 0x000000000000808bn, 0x0000000080000001n,
  0x8000000080008081n, 0x8000000000008009n, 0x000000000000008an,
  0x0000000000000088n, 0x0000000080008009n, 0x000000008000000an,
  0x000000008000808bn, 0x800000000000008bn, 0x8000000000008089n,
  0x8000000000008003n, 0x8000000000008002n, 0x8000000000000080n,
  0x000000000000800an, 0x800000008000000an, 0x8000000080008081n,
  0x8000000000008080n, 0x0000000080000001n, 0x8000000080008008n,
];

const ROTATION_OFFSETS = [
  [0n, 36n, 3n, 41n, 18n],
  [1n, 44n, 10n, 45n, 2n],
  [62n, 6n, 43n, 15n, 61n],
  [28n, 55n, 25n, 21n, 56n],
  [27n, 20n, 39n, 8n, 14n],
];

const MASK64 = (1n << 64n) - 1n;

function rotl64(x, n) {
  const nn = n % 64n;
  if (nn === 0n) return x & MASK64;
  return (((x << nn) | (x >> (64n - nn))) & MASK64);
}

function keccakF1600(state) {
  for (let round = 0; round < 24; round++) {
    // Theta
    const C = new Array(5);
    for (let x = 0; x < 5; x++) {
      C[x] = state[x][0] ^ state[x][1] ^ state[x][2] ^ state[x][3] ^ state[x][4];
    }
    const D = new Array(5);
    for (let x = 0; x < 5; x++) {
      D[x] = (C[(x + 4) % 5] ^ rotl64(C[(x + 1) % 5], 1n)) & MASK64;
    }
    for (let x = 0; x < 5; x++) {
      for (let y = 0; y < 5; y++) {
        state[x][y] = (state[x][y] ^ D[x]) & MASK64;
      }
    }

    // Rho + Pi
    const B = Array.from({ length: 5 }, () => new Array(5));
    for (let x = 0; x < 5; x++) {
      for (let y = 0; y < 5; y++) {
        const newX = y;
        const newY = (2 * x + 3 * y) % 5;
        B[newX][newY] = rotl64(state[x][y], ROTATION_OFFSETS[x][y]);
      }
    }

    // Chi
    for (let x = 0; x < 5; x++) {
      for (let y = 0; y < 5; y++) {
        state[x][y] = (B[x][y] ^ ((~B[(x + 1) % 5][y]) & B[(x + 2) % 5][y])) & MASK64;
      }
    }

    // Iota
    state[0][0] = (state[0][0] ^ RC[round]) & MASK64;
  }
  return state;
}

function hexToBytes(hex) {
  let h = hex.replace(/^0x/i, '');
  if (h.length % 2 !== 0) throw new Error('keccak256: invalid hex length');
  const out = new Uint8Array(h.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(h.substr(i * 2, 2), 16);
  }
  return out;
}

function bytesToHex(bytes) {
  let s = '';
  for (let i = 0; i < bytes.length; i++) {
    s += bytes[i].toString(16).padStart(2, '0');
  }
  return s;
}

function strToBytes(s) {
  return new Uint8Array(Array.from(s).map((c) => c.charCodeAt(0) & 0xff));
}

/**
 * Compute Ethereum Keccak-256.
 * @param {string|Uint8Array} input - hex string (with optional 0x), UTF-8 string, or raw bytes
 * @returns {string} 32-byte digest as 64-char lowercase hex (no 0x prefix)
 */
function keccak256(input) {
  let bytes;
  if (input == null) {
    bytes = new Uint8Array(0);
  } else if (input instanceof Uint8Array) {
    bytes = input;
  } else if (typeof input === 'string') {
    if (/^0x[0-9a-fA-F]+$/.test(input) && input.length % 2 === 0) {
      bytes = hexToBytes(input);
    } else if (/^[0-9a-fA-F]+$/.test(input) && input.length % 2 === 0 && input.length > 0) {
      bytes = hexToBytes(input);
    } else {
      bytes = strToBytes(input);
    }
  } else {
    throw new Error('keccak256: input must be string or Uint8Array');
  }

  const RATE_BYTES = 136; // rate = 1088 bits = 136 bytes for Keccak-256
  const PAD_BYTE = 0x01; // Ethereum Keccak padding (domain 01), NOT 0x06 (SHA-3)
  const DS_BYTE = 0x80;

  // Pad: append PAD_BYTE, zero-fill, then DS_BYTE in final rate block (little-endian).
  const padded = new Uint8Array(Math.ceil((bytes.length + 1) / RATE_BYTES) * RATE_BYTES);
  if (padded.length === 0) {
    padded = new Uint8Array(RATE_BYTES);
  }
  padded.set(bytes, 0);
  padded[bytes.length] ^= PAD_BYTE;
  padded[padded.length - 1] ^= DS_BYTE;

  // 5x5 state of 64-bit lanes (BigInt), little-endian absorption.
  const state = Array.from({ length: 5 }, () => [0n, 0n, 0n, 0n, 0n]);

  for (let offset = 0; offset < padded.length; offset += RATE_BYTES) {
    for (let i = 0; i < RATE_BYTES / 8; i++) {
      let lane = 0n;
      for (let b = 0; b < 8; b++) {
        lane |= BigInt(padded[offset + i * 8 + b]) << (8n * BigInt(b));
      }
      const x = i % 5;
      const y = Math.floor(i / 5);
      state[x][y] ^= lane;
    }
    keccakF1600(state);
  }

  // Squeeze 32 bytes (rate is 136, plenty for one squeeze).
  const out = new Uint8Array(32);
  for (let i = 0; i < 4; i++) {
    const x = i % 5;
    const y = Math.floor(i / 5);
    const lane = state[x][y];
    for (let b = 0; b < 8; b++) {
      out[i * 8 + b] = Number((lane >> (8n * BigInt(b))) & 0xffn);
    }
  }
  return bytesToHex(out);
}

// Self-test at module load: verify against canonical Ethereum Keccak-256
// (padding byte 0x01) digests cross-checked with pycryptodome's keccak_256.
// keccak256("")    = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
// keccak256("abc") = 4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45
// keccak256("0x")  = 39bef1777deb3dfb14f64b9f81ced092c501fee72f90e93d03bb95ee89df9837
const VEC_EMPTY = 'c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470';
const VEC_ABC = '4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45';
const VEC_0X = '39bef1777deb3dfb14f64b9f81ced092c501fee72f90e93d03bb95ee89df9837';
const hEmpty = keccak256('');
const hAbc = keccak256('abc');
const h0x = keccak256('0x');
if (hEmpty !== VEC_EMPTY) {
  throw new Error(`keccak256 self-test FAILED on empty string: got ${hEmpty}, expected ${VEC_EMPTY}`);
}
if (hAbc !== VEC_ABC) {
  throw new Error(`keccak256 self-test FAILED on 'abc': got ${hAbc}, expected ${VEC_ABC}`);
}
if (h0x !== VEC_0X) {
  throw new Error(`keccak256 self-test FAILED on '0x': got ${h0x}, expected ${VEC_0X}`);
}

// UMD: CommonJS for node/tests, globalThis for MV3 service worker (importScripts).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { keccak256 };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_KECCAK = { keccak256 };
}
