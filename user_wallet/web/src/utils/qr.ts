// qr.ts — self-contained QR Code encoder (ISO/IEC 18004).
//
// Byte mode, error-correction levels L/M/Q/H, versions 1-40, Reed-Solomon over
// GF(256) with the 0x11D primitive polynomial, and all eight data masks scored
// by the standard penalty rules so the emitted symbol matches what any other
// conformant encoder produces for the same input. No third-party dependency is
// used: the matrix below is a real, scannable QR symbol, not a decorative
// pattern.

export type Ecc = 'L' | 'M' | 'Q' | 'H';

// Format-information bit values for each error-correction level.
const ECC_FORMAT_BITS: Record<Ecc, number> = { L: 1, M: 0, Q: 3, H: 2 };

// ISO/IEC 18004 Table 9 — error-correction characteristics, indexed by version
// then level. Each entry is [totalCodewords, [[numBlocks, blockTotal, blockData], ...]]
// where blockTotal = blockData + blockEcc. Versions that split into two block
// groups list both groups in order.
const ECC_TABLE: Record<number, Record<Ecc, [number, number[][]]>> = {
  1: {L:[26,[[1,26,19]]], M:[26,[[1,26,16]]], Q:[26,[[1,26,13]]], H:[26,[[1,26,9]]]},
  2: {L:[44,[[1,44,34]]], M:[44,[[1,44,28]]], Q:[44,[[1,44,22]]], H:[44,[[1,44,16]]]},
  3: {L:[70,[[1,70,55]]], M:[70,[[1,70,44]]], Q:[70,[[2,35,17]]], H:[70,[[2,35,13]]]},
  4: {L:[100,[[1,100,80]]], M:[100,[[2,50,32]]], Q:[100,[[2,50,24]]], H:[100,[[4,25,9]]]},
  5: {L:[134,[[1,134,108]]], M:[134,[[2,67,43]]], Q:[134,[[2,33,15],[2,34,16]]], H:[134,[[2,33,11],[2,34,12]]]},
  6: {L:[172,[[2,86,68]]], M:[172,[[4,43,27]]], Q:[172,[[4,43,19]]], H:[172,[[4,43,15]]]},
  7: {L:[196,[[2,98,78]]], M:[196,[[4,49,31]]], Q:[196,[[2,32,14],[4,33,15]]], H:[196,[[4,39,13],[1,40,14]]]},
  8: {L:[242,[[2,121,97]]], M:[242,[[2,60,38],[2,61,39]]], Q:[242,[[4,40,18],[2,41,19]]], H:[242,[[4,40,14],[2,41,15]]]},
  9: {L:[292,[[2,146,116]]], M:[292,[[3,58,36],[2,59,37]]], Q:[292,[[4,36,16],[4,37,17]]], H:[292,[[4,36,12],[4,37,13]]]},
  10: {L:[346,[[2,86,68],[2,87,69]]], M:[346,[[4,69,43],[1,70,44]]], Q:[346,[[6,43,19],[2,44,20]]], H:[346,[[6,43,15],[2,44,16]]]},
  11: {L:[404,[[4,101,81]]], M:[404,[[1,80,50],[4,81,51]]], Q:[404,[[4,50,22],[4,51,23]]], H:[404,[[3,36,12],[8,37,13]]]},
  12: {L:[466,[[2,116,92],[2,117,93]]], M:[466,[[6,58,36],[2,59,37]]], Q:[466,[[4,46,20],[6,47,21]]], H:[466,[[7,42,14],[4,43,15]]]},
  13: {L:[532,[[4,133,107]]], M:[532,[[8,59,37],[1,60,38]]], Q:[532,[[8,44,20],[4,45,21]]], H:[532,[[12,33,11],[4,34,12]]]},
  14: {L:[581,[[3,145,115],[1,146,116]]], M:[581,[[4,64,40],[5,65,41]]], Q:[581,[[11,36,16],[5,37,17]]], H:[581,[[11,36,12],[5,37,13]]]},
  15: {L:[655,[[5,109,87],[1,110,88]]], M:[655,[[5,65,41],[5,66,42]]], Q:[655,[[5,54,24],[7,55,25]]], H:[655,[[11,36,12],[7,37,13]]]},
  16: {L:[733,[[5,122,98],[1,123,99]]], M:[733,[[7,73,45],[3,74,46]]], Q:[733,[[15,43,19],[2,44,20]]], H:[733,[[3,45,15],[13,46,16]]]},
  17: {L:[815,[[1,135,107],[5,136,108]]], M:[815,[[10,74,46],[1,75,47]]], Q:[815,[[1,50,22],[15,51,23]]], H:[815,[[2,42,14],[17,43,15]]]},
  18: {L:[901,[[5,150,120],[1,151,121]]], M:[901,[[9,69,43],[4,70,44]]], Q:[901,[[17,50,22],[1,51,23]]], H:[901,[[2,42,14],[19,43,15]]]},
  19: {L:[991,[[3,141,113],[4,142,114]]], M:[991,[[3,70,44],[11,71,45]]], Q:[991,[[17,47,21],[4,48,22]]], H:[991,[[9,39,13],[16,40,14]]]},
  20: {L:[1085,[[3,135,107],[5,136,108]]], M:[1085,[[3,67,41],[13,68,42]]], Q:[1085,[[15,54,24],[5,55,25]]], H:[1085,[[15,43,15],[10,44,16]]]},
  21: {L:[1156,[[4,144,116],[4,145,117]]], M:[1156,[[17,68,42]]], Q:[1156,[[17,50,22],[6,51,23]]], H:[1156,[[19,46,16],[6,47,17]]]},
  22: {L:[1258,[[2,139,111],[7,140,112]]], M:[1258,[[17,74,46]]], Q:[1258,[[7,54,24],[16,55,25]]], H:[1258,[[34,37,13]]]},
  23: {L:[1364,[[4,151,121],[5,152,122]]], M:[1364,[[4,75,47],[14,76,48]]], Q:[1364,[[11,54,24],[14,55,25]]], H:[1364,[[16,45,15],[14,46,16]]]},
  24: {L:[1474,[[6,147,117],[4,148,118]]], M:[1474,[[6,73,45],[14,74,46]]], Q:[1474,[[11,54,24],[16,55,25]]], H:[1474,[[30,46,16],[2,47,17]]]},
  25: {L:[1588,[[8,132,106],[4,133,107]]], M:[1588,[[8,75,47],[13,76,48]]], Q:[1588,[[7,54,24],[22,55,25]]], H:[1588,[[22,45,15],[13,46,16]]]},
  26: {L:[1706,[[10,142,114],[2,143,115]]], M:[1706,[[19,74,46],[4,75,47]]], Q:[1706,[[28,50,22],[6,51,23]]], H:[1706,[[33,46,16],[4,47,17]]]},
  27: {L:[1828,[[8,152,122],[4,153,123]]], M:[1828,[[22,73,45],[3,74,46]]], Q:[1828,[[8,53,23],[26,54,24]]], H:[1828,[[12,45,15],[28,46,16]]]},
  28: {L:[1921,[[3,147,117],[10,148,118]]], M:[1921,[[3,73,45],[23,74,46]]], Q:[1921,[[4,54,24],[31,55,25]]], H:[1921,[[11,45,15],[31,46,16]]]},
  29: {L:[2051,[[7,146,116],[7,147,117]]], M:[2051,[[21,73,45],[7,74,46]]], Q:[2051,[[1,53,23],[37,54,24]]], H:[2051,[[19,45,15],[26,46,16]]]},
  30: {L:[2185,[[5,145,115],[10,146,116]]], M:[2185,[[19,75,47],[10,76,48]]], Q:[2185,[[15,54,24],[25,55,25]]], H:[2185,[[23,45,15],[25,46,16]]]},
  31: {L:[2323,[[13,145,115],[3,146,116]]], M:[2323,[[2,74,46],[29,75,47]]], Q:[2323,[[42,54,24],[1,55,25]]], H:[2323,[[23,45,15],[28,46,16]]]},
  32: {L:[2465,[[17,145,115]]], M:[2465,[[10,74,46],[23,75,47]]], Q:[2465,[[10,54,24],[35,55,25]]], H:[2465,[[19,45,15],[35,46,16]]]},
  33: {L:[2611,[[17,145,115],[1,146,116]]], M:[2611,[[14,74,46],[21,75,47]]], Q:[2611,[[29,54,24],[19,55,25]]], H:[2611,[[11,45,15],[46,46,16]]]},
  34: {L:[2761,[[13,145,115],[6,146,116]]], M:[2761,[[14,74,46],[23,75,47]]], Q:[2761,[[44,54,24],[7,55,25]]], H:[2761,[[59,46,16],[1,47,17]]]},
  35: {L:[2876,[[12,151,121],[7,152,122]]], M:[2876,[[12,75,47],[26,76,48]]], Q:[2876,[[39,54,24],[14,55,25]]], H:[2876,[[22,45,15],[41,46,16]]]},
  36: {L:[3034,[[6,151,121],[14,152,122]]], M:[3034,[[6,75,47],[34,76,48]]], Q:[3034,[[46,54,24],[10,55,25]]], H:[3034,[[2,45,15],[64,46,16]]]},
  37: {L:[3196,[[17,152,122],[4,153,123]]], M:[3196,[[29,74,46],[14,75,47]]], Q:[3196,[[49,54,24],[10,55,25]]], H:[3196,[[24,45,15],[46,46,16]]]},
  38: {L:[3362,[[4,152,122],[18,153,123]]], M:[3362,[[13,74,46],[32,75,47]]], Q:[3362,[[48,54,24],[14,55,25]]], H:[3362,[[42,45,15],[32,46,16]]]},
  39: {L:[3532,[[20,147,117],[4,148,118]]], M:[3532,[[40,75,47],[7,76,48]]], Q:[3532,[[43,54,24],[22,55,25]]], H:[3532,[[10,45,15],[67,46,16]]]},
  40: {L:[3706,[[19,148,118],[6,149,119]]], M:[3706,[[18,75,47],[31,76,48]]], Q:[3706,[[34,54,24],[34,55,25]]], H:[3706,[[20,45,15],[61,46,16]]]},
};

// Total codewords (data + ecc) for a version, and the per-block layout.
function eccInfo(ver: number, ecc: Ecc): { total: number; groups: number[][] } {
  const entry = ECC_TABLE[ver]?.[ecc];
  if (!entry) throw new Error(`qr: no error-correction table for version ${ver} level ${ecc}`);
  return { total: entry[0], groups: entry[1] };
}

// Data codewords carried by a version at a given error-correction level.
function numDataCodewords(ver: number, ecc: Ecc): number {
  return eccInfo(ver, ecc).groups.reduce((sum, g) => sum + g[0] * g[2], 0);
}

const MIN_VERSION = 1;
const MAX_VERSION = 40;
const PENALTY_N1 = 3;
const PENALTY_N2 = 3;
const PENALTY_N3 = 40;
const PENALTY_N4 = 10;

// Module count of a symbol (excluding the quiet zone) for a version.
function sizeForVersion(ver: number): number {
  return ver * 4 + 17;
}


// Character-count indicator width for byte mode.
function byteModeCharCountBits(ver: number): number {
  return ver <= 9 ? 8 : 16;
}

// Alignment-pattern centre coordinates for a version.
function alignmentPositions(ver: number): number[] {
  if (ver === 1) return [];
  const numAlign = Math.floor(ver / 7) + 2;
  const step = ver === 32 ? 26 : Math.ceil((ver * 4 + 4) / (numAlign * 2 - 2)) * 2;
  const result = [6];
  for (let pos = ver * 4 + 10; result.length < numAlign; pos -= step) result.splice(1, 0, pos);
  return result;
}

// Multiplication in GF(256) modulo the QR primitive polynomial x^8+x^4+x^3+x^2+1.
function gfMultiply(x: number, y: number): number {
  let z = 0;
  for (let i = 7; i >= 0; i--) {
    z = (z << 1) ^ ((z >>> 7) * 0x11d);
    z ^= ((y >>> i) & 1) * x;
  }
  return z & 0xff;
}

// Generator polynomial for `degree` Reed-Solomon error-correction codewords.
function reedSolomonDivisor(degree: number): Uint8Array {
  const result = new Uint8Array(degree);
  result[degree - 1] = 1;
  let root = 1;
  for (let i = 0; i < degree; i++) {
    for (let j = 0; j < degree; j++) {
      result[j] = gfMultiply(result[j], root);
      if (j + 1 < degree) result[j] ^= result[j + 1];
    }
    root = gfMultiply(root, 0x02);
  }
  return result;
}

// Remainder of `data` divided by `divisor` — the ECC codewords.
function reedSolomonRemainder(data: Uint8Array, divisor: Uint8Array): Uint8Array {
  const result = new Uint8Array(divisor.length);
  for (const b of data) {
    const factor = b ^ result[0];
    result.copyWithin(0, 1);
    result[result.length - 1] = 0;
    for (let i = 0; i < divisor.length; i++) result[i] ^= gfMultiply(divisor[i], factor);
  }
  return result;
}

// UTF-8 encode without relying on a TextEncoder being present.
function utf8Bytes(text: string): Uint8Array {
  if (typeof TextEncoder !== 'undefined') return new TextEncoder().encode(text);
  const out: number[] = [];
  for (let i = 0; i < text.length; i++) {
    let cp = text.codePointAt(i)!;
    if (cp > 0xffff) i++;
    if (cp < 0x80) out.push(cp);
    else if (cp < 0x800) out.push(0xc0 | (cp >> 6), 0x80 | (cp & 0x3f));
    else if (cp < 0x10000) out.push(0xe0 | (cp >> 12), 0x80 | ((cp >> 6) & 0x3f), 0x80 | (cp & 0x3f));
    else out.push(0xf0 | (cp >> 18), 0x80 | ((cp >> 12) & 0x3f), 0x80 | ((cp >> 6) & 0x3f), 0x80 | (cp & 0x3f));
  }
  return new Uint8Array(out);
}

// Smallest version that can carry `len` bytes of byte-mode data at `ecc`.
function chooseVersion(len: number, ecc: Ecc): number {
  for (let ver = MIN_VERSION; ver <= MAX_VERSION; ver++) {
    const capacityBits = numDataCodewords(ver, ecc) * 8;
    const neededBits = 4 + byteModeCharCountBits(ver) + len * 8;
    if (neededBits <= capacityBits) return ver;
  }
  throw new Error('qr: data too long for a version-40 symbol');
}

// Byte-mode payload for `ver`, already padded to the version's codeword count.
function buildDataCodewords(bytes: Uint8Array, ver: number, ecc: Ecc): Uint8Array {
  const capacity = numDataCodewords(ver, ecc);
  const bits: number[] = [];
  const push = (value: number, len: number) => {
    for (let i = len - 1; i >= 0; i--) bits.push((value >>> i) & 1);
  };

  push(0b0100, 4); // byte mode
  push(bytes.length, byteModeCharCountBits(ver));
  for (const b of bytes) push(b, 8);

  // Terminator, then pad to a byte boundary, then alternating pad codewords.
  const capacityBits = capacity * 8;
  for (let i = 0; i < 4 && bits.length < capacityBits; i++) bits.push(0);
  while (bits.length % 8 !== 0) bits.push(0);

  const codewords = new Uint8Array(capacity);
  for (let i = 0; i < bits.length; i++) {
    if (bits[i]) codewords[i >>> 3] |= 0x80 >>> (i & 7);
  }
  for (let i = Math.ceil(bits.length / 8), pad = 0xec; i < capacity; i++, pad ^= 0xec ^ 0x11) {
    codewords[i] = pad;
  }
  return codewords;
}

// Splits data into Reed-Solomon blocks, appends ECC, and interleaves the result
// per ISO/IEC 18004 §8.6: data codewords are interleaved block by block, then
// the ECC codewords, with shorter blocks simply contributing fewer codewords.
function addEccAndInterleave(data: Uint8Array, ver: number, ecc: Ecc): Uint8Array {
  const { groups } = eccInfo(ver, ecc);

  const dataBlocks: number[][] = [];
  const eccBlocks: number[][] = [];
  let k = 0;
  for (const [numBlocks, blockTotal, blockData] of groups) {
    const eccLen = blockTotal - blockData;
    const divisor = reedSolomonDivisor(eccLen);
    for (let i = 0; i < numBlocks; i++) {
      const dat = Array.from(data.slice(k, k + blockData));
      k += blockData;
      dataBlocks.push(dat);
      eccBlocks.push(Array.from(reedSolomonRemainder(new Uint8Array(dat), divisor)));
    }
  }

  const result: number[] = [];
  const maxData = Math.max(...dataBlocks.map((b) => b.length));
  for (let i = 0; i < maxData; i++) {
    for (const block of dataBlocks) if (i < block.length) result.push(block[i]);
  }
  const maxEcc = Math.max(...eccBlocks.map((b) => b.length));
  for (let i = 0; i < maxEcc; i++) {
    for (const block of eccBlocks) if (i < block.length) result.push(block[i]);
  }
  return new Uint8Array(result);
}

class QrSymbol {
  readonly size: number;
  readonly modules: boolean[][];
  readonly isFunction: boolean[][];

  constructor(ver: number) {
    this.size = sizeForVersion(ver);
    this.modules = Array.from({ length: this.size }, () => new Array<boolean>(this.size).fill(false));
    this.isFunction = Array.from({ length: this.size }, () => new Array<boolean>(this.size).fill(false));
  }

  private setFunction(x: number, y: number, dark: boolean): void {
    this.modules[y][x] = dark;
    this.isFunction[y][x] = true;
  }

  drawFunctionPatterns(ver: number): void {
    for (let i = 0; i < this.size; i++) {
      this.setFunction(6, i, i % 2 === 0);
      this.setFunction(i, 6, i % 2 === 0);
    }
    this.drawFinder(3, 3);
    this.drawFinder(this.size - 4, 3);
    this.drawFinder(3, this.size - 4);

    const align = alignmentPositions(ver);
    for (let i = 0; i < align.length; i++) {
      for (let j = 0; j < align.length; j++) {
        const corner =
          (i === 0 && j === 0) ||
          (i === 0 && j === align.length - 1) ||
          (i === align.length - 1 && j === 0);
        if (!corner) this.drawAlignment(align[i], align[j]);
      }
    }

    // Reserve the format-info modules; real values are written once a mask is chosen.
    this.drawFormatBits('L', 0);
    this.drawVersionBits(ver);
  }

  private drawFinder(cx: number, cy: number): void {
    for (let dy = -4; dy <= 4; dy++) {
      for (let dx = -4; dx <= 4; dx++) {
        const dist = Math.max(Math.abs(dx), Math.abs(dy));
        const x = cx + dx;
        const y = cy + dy;
        if (x >= 0 && x < this.size && y >= 0 && y < this.size) {
          this.setFunction(x, y, dist !== 2 && dist !== 4);
        }
      }
    }
  }

  private drawAlignment(cx: number, cy: number): void {
    for (let dy = -2; dy <= 2; dy++) {
      for (let dx = -2; dx <= 2; dx++) {
        this.setFunction(cx + dx, cy + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
      }
    }
  }

  drawFormatBits(ecc: Ecc, mask: number): void {
    const data = (ECC_FORMAT_BITS[ecc] << 3) | mask;
    let rem = data;
    for (let i = 0; i < 10; i++) rem = (rem << 1) ^ ((rem >>> 9) * 0x537);
    const bits = ((data << 10) | rem) ^ 0x5412;

    const bit = (i: number) => ((bits >>> i) & 1) !== 0;
    for (let i = 0; i <= 5; i++) this.setFunction(8, i, bit(i));
    this.setFunction(8, 7, bit(6));
    this.setFunction(8, 8, bit(7));
    this.setFunction(7, 8, bit(8));
    for (let i = 9; i < 15; i++) this.setFunction(14 - i, 8, bit(i));

    for (let i = 0; i < 8; i++) this.setFunction(this.size - 1 - i, 8, bit(i));
    for (let i = 8; i < 15; i++) this.setFunction(8, this.size - 15 + i, bit(i));
    this.setFunction(8, this.size - 8, true); // always-dark module
  }

  private drawVersionBits(ver: number): void {
    if (ver < 7) return;
    let rem = ver;
    for (let i = 0; i < 12; i++) rem = (rem << 1) ^ ((rem >>> 11) * 0x1f25);
    const bits = (ver << 12) | rem;
    for (let i = 0; i < 18; i++) {
      const bit = ((bits >>> i) & 1) !== 0;
      const a = this.size - 11 + (i % 3);
      const b = Math.floor(i / 3);
      this.setFunction(a, b, bit);
      this.setFunction(b, a, bit);
    }
  }

  // Places the interleaved codewords in the standard zig-zag order.
  drawCodewords(data: Uint8Array): void {
    let i = 0;
    for (let right = this.size - 1; right >= 1; right -= 2) {
      if (right === 6) right = 5;
      for (let vert = 0; vert < this.size; vert++) {
        for (let j = 0; j < 2; j++) {
          const x = right - j;
          const upward = ((right + 1) & 2) === 0;
          const y = upward ? this.size - 1 - vert : vert;
          if (!this.isFunction[y][x] && i < data.length * 8) {
            this.modules[y][x] = ((data[i >>> 3] >>> (7 - (i & 7))) & 1) !== 0;
            i++;
          }
        }
      }
    }
  }

  applyMask(mask: number): void {
    for (let y = 0; y < this.size; y++) {
      for (let x = 0; x < this.size; x++) {
        if (this.isFunction[y][x]) continue;
        let invert: boolean;
        switch (mask) {
          case 0: invert = (x + y) % 2 === 0; break;
          case 1: invert = y % 2 === 0; break;
          case 2: invert = x % 3 === 0; break;
          case 3: invert = (x + y) % 3 === 0; break;
          case 4: invert = (Math.floor(x / 3) + Math.floor(y / 2)) % 2 === 0; break;
          case 5: invert = ((x * y) % 2) + ((x * y) % 3) === 0; break;
          case 6: invert = (((x * y) % 2) + ((x * y) % 3)) % 2 === 0; break;
          case 7: invert = (((x + y) % 2) + ((x * y) % 3)) % 2 === 0; break;
          default: throw new Error('qr: invalid mask');
        }
        if (invert) this.modules[y][x] = !this.modules[y][x];
      }
    }
  }

  // Total penalty across the four ISO/IEC 18004 rules; lower is better.
  penaltyScore(): number {
    const n = this.size;
    let result = 0;

    // Rule 1: runs of five or more same-coloured modules in a row or column.
    for (let y = 0; y < n; y++) {
      let runColor = false;
      let runLen = 0;
      for (let x = 0; x < n; x++) {
        if (this.modules[y][x] === runColor) {
          runLen++;
          if (runLen === 5) result += PENALTY_N1;
          else if (runLen > 5) result++;
        } else {
          runColor = this.modules[y][x];
          runLen = 1;
        }
      }
    }
    for (let x = 0; x < n; x++) {
      let runColor = false;
      let runLen = 0;
      for (let y = 0; y < n; y++) {
        if (this.modules[y][x] === runColor) {
          runLen++;
          if (runLen === 5) result += PENALTY_N1;
          else if (runLen > 5) result++;
        } else {
          runColor = this.modules[y][x];
          runLen = 1;
        }
      }
    }

    // Rule 2: 2x2 blocks of one colour.
    for (let y = 0; y < n - 1; y++) {
      for (let x = 0; x < n - 1; x++) {
        const c = this.modules[y][x];
        if (c === this.modules[y][x + 1] && c === this.modules[y + 1][x] && c === this.modules[y + 1][x + 1]) {
          result += PENALTY_N2;
        }
      }
    }

    // Rule 3: finder-like 1:1:3:1:1 patterns preceded or followed by four light modules.
    const finderLike = (run: boolean[]): boolean => {
      const sum = run.reduce((a, b) => a + (b ? 1 : 0), 0);
      return (
        run.length === 7 &&
        sum === 3 &&
        run[0] && !run[1] && run[2] && run[3] && run[4] && !run[5] && run[6]
      );
    };
    const hasFinderPattern = (get: (i: number) => boolean, start: number, n2: number): boolean => {
      // Checks [start, start+7) for the pattern with four light modules on either side.
      const run: boolean[] = [];
      for (let i = 0; i < 7; i++) run.push(get(start + i));
      if (!finderLike(run)) return false;
      let before = true;
      for (let i = 1; i <= 4; i++) if (start - i >= 0 && get(start - i)) before = false;
      let after = true;
      for (let i = 7; i < 11; i++) if (start + i < n2 && get(start + i)) after = false;
      return before || after;
    };
    for (let y = 0; y < n; y++) {
      const get = (x: number) => this.modules[y][x];
      for (let x = 0; x + 7 <= n; x++) if (hasFinderPattern(get, x, n)) result += PENALTY_N3;
    }
    for (let x = 0; x < n; x++) {
      const get = (y: number) => this.modules[y][x];
      for (let y = 0; y + 7 <= n; y++) if (hasFinderPattern(get, y, n)) result += PENALTY_N3;
    }

    // Rule 4: deviation of the dark-module proportion from 50%.
    let dark = 0;
    for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) if (this.modules[y][x]) dark++;
    const total = n * n;
    const k = Math.ceil(Math.abs(dark * 20 - total * 10) / total) - 1;
    result += k * PENALTY_N4;

    return result;
  }
}

// qrMatrix encodes `text` and returns the module grid (true = dark), quiet zone
// excluded. The smallest version that fits is selected automatically.
export function qrMatrix(text: string, ecc: Ecc = 'M'): boolean[][] {
  const bytes = utf8Bytes(text);
  const ver = chooseVersion(bytes.length, ecc);
  const data = addEccAndInterleave(buildDataCodewords(bytes, ver, ecc), ver, ecc);

  const symbol = new QrSymbol(ver);
  symbol.drawFunctionPatterns(ver);
  symbol.drawCodewords(data);

  // Try every mask, keep the one with the lowest penalty (ties keep the lowest mask).
  let bestMask = 0;
  let bestPenalty = Infinity;
  for (let mask = 0; mask < 8; mask++) {
    symbol.applyMask(mask);
    symbol.drawFormatBits(ecc, mask);
    const penalty = symbol.penaltyScore();
    if (penalty < bestPenalty) {
      bestPenalty = penalty;
      bestMask = mask;
    }
    symbol.applyMask(mask); // XOR is its own inverse — undo before the next try
  }
  symbol.applyMask(bestMask);
  symbol.drawFormatBits(ecc, bestMask);
  return symbol.modules;
}

// drawQr renders `text` into a canvas with a standards-compliant quiet zone.
// `scale` is the pixel size of one module; `quietZone` defaults to 4 modules.
export function drawQr(
  canvas: HTMLCanvasElement | null,
  text: string,
  opts: { scale?: number; quietZone?: number; ecc?: Ecc; dark?: string; light?: string } = {}
): void {
  if (!canvas) return;
  const quietZone = opts.quietZone ?? 4;
  const scale = opts.scale ?? 4;
  const dark = opts.dark ?? '#000000';
  const light = opts.light ?? '#ffffff';

  const matrix = qrMatrix(text, opts.ecc ?? 'M');
  const count = matrix.length + quietZone * 2;
  const side = count * scale;
  canvas.width = side;
  canvas.height = side;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  ctx.fillStyle = light;
  ctx.fillRect(0, 0, side, side);
  ctx.fillStyle = dark;
  for (let y = 0; y < matrix.length; y++) {
    for (let x = 0; x < matrix.length; x++) {
      if (matrix[y][x]) {
        ctx.fillRect((x + quietZone) * scale, (y + quietZone) * scale, scale, scale);
      }
    }
  }
}
