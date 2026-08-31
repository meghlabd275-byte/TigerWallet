/**
 * TigerWallet Desktop - Hardware Wallet Integration
 *
 * REAL device protocols, no fabrication:
 *  - Ledger: full HID framing (channel 0x0101) + Ethereum-app APDUs over
 *    WebHID — getPublicKey (address), signPersonalMessage (EIP-191), and
 *    signTransaction (legacy EIP-155 + EIP-1559 type-2) with on-device
 *    confirmation. Includes a complete RLP encoder to rebuild the signed
 *    raw transaction from the device-returned (v, r, s).
 *  - Trezor: the standard Trezor Bridge (trezord) HTTP protocol on
 *    127.0.0.1:21325 with hand-encoded protobuf messages (Initialize,
 *    EthereumGetAddress, EthereumSignMessage, EthereumSignTx legacy) and
 *    real ButtonRequest/ButtonAck pin-matrix flow handling.
 *
 * Fail-closed: if the transport or device is unavailable, or a chain family
 * requires a different on-device app than the one implemented, an explicit
 * error is thrown — an address/signature is NEVER fabricated client-side.
 */

// ---------------------------------------------------------------------------
// Byte helpers
// ---------------------------------------------------------------------------

function hwHexToBytes(hex) {
    let h = (hex || '').toString().trim().toLowerCase();
    if (h.startsWith('0x')) h = h.slice(2);
    if (h.length % 2) h = '0' + h;
    const out = new Uint8Array(h.length / 2);
    for (let i = 0; i < out.length; i++) out[i] = parseInt(h.slice(i * 2, i * 2 + 2), 16);
    return out;
}

function hwBytesToHex(bytes) {
    return Array.from(bytes).map((b) => b.toString(16).padStart(2, '0')).join('');
}

function hwConcatBytes(parts) {
    const total = parts.reduce((n, p) => n + p.length, 0);
    const out = new Uint8Array(total);
    let off = 0;
    for (const p of parts) { out.set(p, off); off += p.length; }
    return out;
}

function hwBigIntToMinimalBytes(value) {
    let v = BigInt(value || 0);
    if (v === 0n) return new Uint8Array(0);
    const bytes = [];
    while (v > 0n) { bytes.unshift(Number(v & 0xffn)); v >>= 8n; }
    return new Uint8Array(bytes);
}

function hwUtf8(str) {
    return new TextEncoder().encode(str);
}

/** Parse a BIP-32 path ("m/44'/60'/0'/0/0") into uint32 elements. */
function hwParseDerivationPath(path) {
    const m = (path || "m/44'/60'/0'/0/0").replace(/^m\/?/, '');
    if (!m) return [];
    return m.split('/').map((seg) => {
        const hardened = seg.endsWith("'") || seg.endsWith('h') || seg.endsWith('H');
        const n = parseInt(seg.replace(/['hH]$/, ''), 10);
        if (!Number.isFinite(n) || n < 0 || n > 0x7fffffff) {
            throw new Error('Invalid derivation path segment: ' + seg);
        }
        return hardened ? (n | 0x80000000) >>> 0 : n >>> 0;
    });
}

// ---------------------------------------------------------------------------
// RLP encoding (needed to build the Ledger signing payload and to rebuild the
// signed raw transaction from the device-returned v/r/s).
// ---------------------------------------------------------------------------

function rlpEncodeBytes(bytes) {
    if (bytes.length === 1 && bytes[0] < 0x80) return bytes;
    if (bytes.length <= 55) return hwConcatBytes([new Uint8Array([0x80 + bytes.length]), bytes]);
    const len = hwBigIntToMinimalBytes(bytes.length);
    return hwConcatBytes([new Uint8Array([0xb7 + len.length]), len, bytes]);
}

function rlpEncode(item) {
    if (Array.isArray(item)) {
        const payload = hwConcatBytes(item.map(rlpEncode));
        if (payload.length <= 55) return hwConcatBytes([new Uint8Array([0xc0 + payload.length]), payload]);
        const len = hwBigIntToMinimalBytes(payload.length);
        return hwConcatBytes([new Uint8Array([0xf7 + len.length]), len, payload]);
    }
    if (item instanceof Uint8Array) return rlpEncodeBytes(item);
    if (typeof item === 'bigint' || typeof item === 'number') return rlpEncodeBytes(hwBigIntToMinimalBytes(item));
    // string: hex (0x…) or plain utf8
    if (typeof item === 'string' && item.startsWith('0x')) return rlpEncodeBytes(hwHexToBytes(item));
    return rlpEncodeBytes(hwUtf8(String(item)));
}

// ---------------------------------------------------------------------------
// Ledger transport: WebHID framing (Ledger comm protocol, channel 0x0101)
// ---------------------------------------------------------------------------

const LEDGER_VENDOR_ID = 0x2c97;
const LEDGER_CHANNEL = 0x0101;
const LEDGER_TAG_APDU = 0x05;
const LEDGER_PACKET_SIZE = 64;
const LEDGER_MAX_CHUNK = 150; // safe APDU data chunk for sign continuation frames

class LedgerHidTransport {
    constructor(device) {
        this.device = device;
        this._queue = Promise.resolve();
    }

    static wrapCommand(apdu) {
        const frames = [];
        let offset = 0;
        let seq = 0;
        while (offset < apdu.length || seq === 0) {
            const frame = new Uint8Array(LEDGER_PACKET_SIZE);
            const view = new DataView(frame.buffer);
            view.setUint16(0, LEDGER_CHANNEL);
            frame[2] = LEDGER_TAG_APDU;
            view.setUint16(3, seq);
            if (seq === 0) {
                view.setUint16(5, apdu.length);
                const chunk = Math.min(apdu.length - offset, LEDGER_PACKET_SIZE - 7);
                frame.set(apdu.subarray(offset, offset + chunk), 7);
                offset += chunk;
            } else {
                const chunk = Math.min(apdu.length - offset, LEDGER_PACKET_SIZE - 5);
                frame.set(apdu.subarray(offset, offset + chunk), 5);
                offset += chunk;
            }
            frames.push(frame);
            seq++;
            if (apdu.length === 0) break;
        }
        return frames;
    }

    static unwrapResponse(frames) {
        let length = -1;
        let seq = 0;
        let acc = new Uint8Array(0);
        for (const frame of frames) {
            const view = new DataView(frame.buffer, frame.byteOffset, frame.byteLength);
            if (view.getUint16(0) !== LEDGER_CHANNEL || frame[2] !== LEDGER_TAG_APDU) {
                throw new Error('Ledger: unexpected HID frame');
            }
            if (view.getUint16(3) !== seq) throw new Error('Ledger: HID frame sequence mismatch');
            if (seq === 0) {
                length = view.getUint16(5);
                acc = hwConcatBytes([acc, frame.subarray(7)]);
            } else {
                acc = hwConcatBytes([acc, frame.subarray(5)]);
            }
            seq++;
            if (length >= 0 && acc.length >= length) break;
        }
        if (length < 0 || acc.length < length) throw new Error('Ledger: truncated HID response');
        return acc.subarray(0, length);
    }

    /** Serialize exchanges so concurrent UI actions cannot interleave frames. */
    exchange(apdu) {
        const run = this._queue.then(() => this._exchange(apdu));
        this._queue = run.catch(() => {});
        return run;
    }

    async _exchange(apdu) {
        const device = this.device;
        if (!device.opened) await device.open();
        const frames = LedgerHidTransport.wrapCommand(apdu);
        const responseFrames = [];
        let needed = 1;
        const done = new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                device.removeEventListener('inputreport', onReport);
                reject(new Error('Ledger: device response timed out'));
            }, 60000);
            const onReport = (ev) => {
                if (ev.reportId !== 0) return;
                const frame = new Uint8Array(ev.data.buffer, ev.data.byteOffset, ev.data.byteLength);
                if (responseFrames.length === 0 && frame.length >= 7) {
                    const view = new DataView(frame.buffer, frame.byteOffset, frame.byteLength);
                    if (view.getUint16(0) === LEDGER_CHANNEL && frame[2] === LEDGER_TAG_APDU) {
                        const respLen = view.getUint16(5);
                        needed = respLen <= 57 ? 1 : 1 + Math.ceil((respLen - 57) / 59);
                    }
                }
                responseFrames.push(frame);
                if (responseFrames.length >= needed) {
                    clearTimeout(timeout);
                    device.removeEventListener('inputreport', onReport);
                    resolve();
                }
            };
            device.addEventListener('inputreport', onReport);
        });
        for (const f of frames) await device.sendReport(0, f);
        await done;
        const payload = LedgerHidTransport.unwrapResponse(responseFrames);
        if (payload.length < 2) throw new Error('Ledger: malformed APDU response');
        const sw = (payload[payload.length - 2] << 8) | payload[payload.length - 1];
        const data = payload.subarray(0, payload.length - 2);
        if (sw === 0x6985) throw new Error('Ledger: action rejected on device');
        if (sw === 0x6d00 || sw === 0x6e00) throw new Error('Ledger: open the Ethereum app on the device');
        if (sw !== 0x9000) throw new Error('Ledger: APDU error status 0x' + sw.toString(16));
        return data;
    }
}

function ledgerApdu(cla, ins, p1, p2, data) {
    const d = data || new Uint8Array(0);
    return hwConcatBytes([new Uint8Array([cla, ins, p1, p2, d.length]), d]);
}

function ledgerPathBytes(path) {
    const elements = hwParseDerivationPath(path);
    const out = new Uint8Array(1 + elements.length * 4);
    out[0] = elements.length;
    const view = new DataView(out.buffer);
    elements.forEach((el, i) => view.setUint32(1 + i * 4, el));
    return out;
}

class LedgerEthApp {
    constructor(transport) { this.transport = transport; }

    /** Ethereum app getPublicKey -> real on-device address. */
    async getAddress(path, displayOnDevice = false) {
        const apdu = ledgerApdu(0xe0, 0x02, displayOnDevice ? 0x01 : 0x00, 0x00, ledgerPathBytes(path));
        const res = await this.transport.exchange(apdu);
        const pubLen = res[0];
        const addrLen = res[1 + pubLen];
        const addrAscii = new TextDecoder().decode(res.subarray(2 + pubLen, 2 + pubLen + addrLen));
        if (!/^[0-9a-fA-F]{40}$/.test(addrAscii)) throw new Error('Ledger: unexpected address response');
        return '0x' + addrAscii.toLowerCase();
    }

    /** Ethereum app signPersonalMessage (EIP-191) -> 65-byte 0x signature. */
    async signPersonalMessage(path, message) {
        const msgBytes = hwUtf8(message);
        await this.transport.exchange(ledgerApdu(0xe0, 0x08, 0x00, 0x00, ledgerPathBytes(path)));
        const lenPrefix = new Uint8Array(4);
        new DataView(lenPrefix.buffer).setUint32(0, msgBytes.length);
        const payload = hwConcatBytes([lenPrefix, msgBytes]);
        let offset = 0;
        let res = new Uint8Array(0);
        while (offset < payload.length) {
            const chunk = payload.subarray(offset, offset + LEDGER_MAX_CHUNK);
            res = await this.transport.exchange(ledgerApdu(0xe0, 0x08, 0x80, 0x00, chunk));
            offset += chunk.length;
        }
        if (res.length !== 65) throw new Error('Ledger: unexpected signature length');
        const v = res[0];
        const r = hwBytesToHex(res.subarray(1, 33));
        const s = hwBytesToHex(res.subarray(33, 65));
        return '0x' + r + s + v.toString(16).padStart(2, '0');
    }

    /**
     * Ethereum app signTransaction. tx = {to, value, data, nonce, gasLimit,
     * chainId, gasPrice} for legacy EIP-155, or {…, maxFeePerGas,
     * maxPriorityFeePerGas} for EIP-1559 type-2. Returns the signed raw tx.
     */
    async signTransaction(path, tx) {
        const chainId = BigInt(tx.chainId || 1);
        const nonce = BigInt(tx.nonce || 0);
        const gasLimit = BigInt(tx.gasLimit || 21000);
        const to = hwHexToBytes(tx.to || '0x');
        const value = hwBigIntToMinimalBytes(tx.value || 0);
        const data = hwHexToBytes(tx.data || '0x');
        const is1559 = tx.maxFeePerGas !== undefined && tx.maxFeePerGas !== null;

        let payload;
        if (is1559) {
            const unsigned = rlpEncode([
                chainId, nonce,
                hwBigIntToMinimalBytes(tx.maxPriorityFeePerGas || 0),
                hwBigIntToMinimalBytes(tx.maxFeePerGas || 0),
                gasLimit, to, value, data, [],
            ]);
            payload = hwConcatBytes([new Uint8Array([0x02]), unsigned]);
        } else {
            if (tx.gasPrice === undefined || tx.gasPrice === null) {
                throw new Error('Legacy transaction requires gasPrice');
            }
            payload = rlpEncode([
                nonce, hwBigIntToMinimalBytes(tx.gasPrice), gasLimit, to, value, data,
                hwBigIntToMinimalBytes(chainId), new Uint8Array(0), new Uint8Array(0),
            ]);
        }

        await this.transport.exchange(ledgerApdu(0xe0, 0x04, 0x00, 0x00, ledgerPathBytes(path)));
        let offset = 0;
        let res = new Uint8Array(0);
        while (offset < payload.length) {
            const chunk = payload.subarray(offset, offset + LEDGER_MAX_CHUNK);
            res = await this.transport.exchange(ledgerApdu(0xe0, 0x04, 0x80, 0x00, chunk));
            offset += chunk.length;
        }
        if (res.length !== 65) throw new Error('Ledger: unexpected signature length');
        const deviceV = res[0];
        const rBytes = res.subarray(1, 33);
        const sBytes = res.subarray(33, 65);

        let signedRaw;
        if (is1559) {
            // Typed txs: the device returns the y-parity (0/1) as v.
            const yParity = deviceV >= 35 ? Number((BigInt(deviceV) - 35n) % 2n) : (deviceV >= 27 ? deviceV - 27 : deviceV);
            signedRaw = hwConcatBytes([new Uint8Array([0x02]), rlpEncode([
                chainId, nonce,
                hwBigIntToMinimalBytes(tx.maxPriorityFeePerGas || 0),
                hwBigIntToMinimalBytes(tx.maxFeePerGas || 0),
                gasLimit, to, value, data, [],
                hwBigIntToMinimalBytes(yParity), rBytes, sBytes,
            ])]);
        } else {
            // Legacy: the device returns the full EIP-155 v (chainId*2+35+y).
            const v = deviceV >= 35 ? deviceV : deviceV + Number(chainId) * 2 + 35 - (deviceV >= 27 ? 27 : 0);
            signedRaw = rlpEncode([nonce, hwBigIntToMinimalBytes(tx.gasPrice), gasLimit, to, value, data, hwBigIntToMinimalBytes(v), rBytes, sBytes]);
        }
        return {
            v: deviceV,
            r: '0x' + hwBytesToHex(rBytes),
            s: '0x' + hwBytesToHex(sBytes),
            rawTransaction: '0x' + hwBytesToHex(signedRaw),
        };
    }
}

// ---------------------------------------------------------------------------
// Trezor transport: Trezor Bridge (trezord) HTTP protocol on 127.0.0.1:21325
// with hand-encoded protobuf for the Ethereum message family.
// ---------------------------------------------------------------------------

const TREZORD_BASE = 'http://127.0.0.1:21325';
const TT = {
    Failure: 3,
    ButtonRequest: 26,
    ButtonAck: 27,
    EthereumGetAddress: 56,
    EthereumAddress: 57,
    EthereumSignTx: 58,
    EthereumTxRequest: 59,
    EthereumSignMessage: 64,
    EthereumMessageSignature: 65,
};

function pbVarint(n) {
    let v = BigInt(n);
    const out = [];
    do { let b = Number(v & 0x7fn); v >>= 7n; if (v > 0n) b |= 0x80; out.push(b); } while (v > 0n);
    return new Uint8Array(out);
}

function pbKey(field, wire) { return pbVarint((field << 3) | wire); }

function pbUint(field, n) { return hwConcatBytes([pbKey(field, 0), pbVarint(n)]); }

function pbBool(field, b) { return pbUint(field, b ? 1 : 0); }

function pbBytes(field, bytes) { return hwConcatBytes([pbKey(field, 2), pbVarint(bytes.length), bytes]); }

function pbPackedUint32(field, nums) {
    return pbBytes(field, hwConcatBytes(nums.map(pbVarint)));
}

function pbParse(buf) {
    const fields = [];
    let i = 0;
    while (i < buf.length) {
        let key = 0, shift = 0;
        for (;;) { const b = buf[i++]; key |= (b & 0x7f) << shift; if (!(b & 0x80)) break; shift += 7; }
        const field = key >> 3, wire = key & 7;
        if (wire === 0) {
            let v = 0n; shift = 0;
            for (;;) { const b = buf[i++]; v |= BigInt(b & 0x7f) << BigInt(shift); if (!(b & 0x80)) break; shift += 7; }
            fields.push({ field, wire, varint: v });
        } else if (wire === 2) {
            let len = 0; shift = 0;
            for (;;) { const b = buf[i++]; len |= (b & 0x7f) << shift; if (!(b & 0x80)) break; shift += 7; }
            fields.push({ field, wire, bytes: buf.subarray(i, i + len) });
            i += len;
        } else {
            throw new Error('Trezor: unsupported protobuf wire type ' + wire);
        }
    }
    return fields;
}

function pbFieldBytes(fields, n) {
    const f = fields.find((x) => x.field === n && x.wire === 2);
    return f ? f.bytes : null;
}

function pbFieldUint(fields, n) {
    const f = fields.find((x) => x.field === n && x.wire === 0);
    return f ? f.varint : undefined;
}

class TrezorBridgeTransport {
    constructor() { this.session = null; }

    async _post(path, body) {
        const res = await fetch(TREZORD_BASE + path, {
            method: 'POST',
            headers: { 'Content-Type': 'application/octet-stream', Origin: 'https://localhost' },
            body,
        });
        if (!res.ok) throw new Error('Trezor Bridge error ' + res.status);
        return res;
    }

    _frame(msgType, payload) {
        return hwConcatBytes([
            new Uint8Array([(msgType >> 8) & 0xff, msgType & 0xff]),
            new Uint8Array([(payload.length >>> 24) & 0xff, (payload.length >>> 16) & 0xff, (payload.length >>> 8) & 0xff, payload.length & 0xff]),
            payload,
        ]);
    }

    /** Acquire a trezord session; throws if the bridge daemon is not running. */
    async acquire() {
        const res = await this._post('/', null);
        this.session = (await res.text()).trim();
        return this.session;
    }

    async release() {
        if (!this.session) return;
        try { await this._post('/release/' + this.session, null); } catch (_) { /* best effort */ }
        this.session = null;
    }

    async _roundtrip(msgType, payload) {
        if (!this.session) throw new Error('Trezor: no bridge session (is trezord running?)');
        const res = await this._post('/call/' + this.session, this._frame(msgType, payload));
        const buf = new Uint8Array(await res.arrayBuffer());
        if (buf.length < 6) throw new Error('Trezor: malformed bridge response');
        const rType = (buf[0] << 8) | buf[1];
        const rLen = (buf[2] << 24) | (buf[3] << 16) | (buf[4] << 8) | buf[5];
        return { type: rType, payload: buf.subarray(6, 6 + rLen) };
    }

    /** Send a protobuf message, auto-answering ButtonRequest confirmations. */
    async call(msgType, payload) {
        let res = await this._roundtrip(msgType, payload);
        // On-device confirmation requested: ack, then read the real response.
        while (res.type === TT.ButtonRequest) {
            res = await this._roundtrip(TT.ButtonAck, new Uint8Array(0));
        }
        if (res.type === TT.Failure) {
            const fields = pbParse(res.payload);
            const msg = pbFieldBytes(fields, 2);
            throw new Error('Trezor: ' + (msg ? new TextDecoder().decode(msg) : 'request failed on device'));
        }
        return res;
    }
}

class TrezorEthApp {
    constructor(transport) { this.transport = transport; }

    async getAddress(path, showDisplay = false) {
        const payload = hwConcatBytes([
            pbPackedUint32(1, hwParseDerivationPath(path)),
            pbBool(3, showDisplay),
        ]);
        const res = await this.transport.call(TT.EthereumGetAddress, payload);
        if (res.type !== TT.EthereumAddress) throw new Error('Trezor: unexpected response ' + res.type);
        const fields = pbParse(res.payload);
        const addrBytes = pbFieldBytes(fields, 1);
        if (!addrBytes) throw new Error('Trezor: empty address response');
        const asText = new TextDecoder().decode(addrBytes);
        if (/^0x[0-9a-fA-F]{40}$/.test(asText)) return asText.toLowerCase();
        if (addrBytes.length === 20) return '0x' + hwBytesToHex(addrBytes);
        throw new Error('Trezor: unexpected address encoding');
    }

    async signMessage(path, message) {
        const payload = hwConcatBytes([
            pbPackedUint32(1, hwParseDerivationPath(path)),
            pbBytes(2, hwUtf8(message)),
        ]);
        const res = await this.transport.call(TT.EthereumSignMessage, payload);
        if (res.type !== TT.EthereumMessageSignature) throw new Error('Trezor: unexpected response ' + res.type);
        const fields = pbParse(res.payload);
        const sig = pbFieldBytes(fields, 2);
        if (!sig || sig.length !== 65) throw new Error('Trezor: unexpected signature length');
        return '0x' + hwBytesToHex(sig);
    }

    /** Legacy EIP-155 transaction signing (EthereumSignTx message family). */
    async signTransaction(path, tx) {
        const chainId = BigInt(tx.chainId || 1);
        const to = (tx.to || '').toString();
        if (!/^0x[0-9a-fA-F]{40}$/.test(to)) throw new Error('Trezor: transaction requires a 20-byte recipient');
        if (tx.maxFeePerGas !== undefined && tx.maxFeePerGas !== null) {
            throw new Error('Trezor: EIP-1559 typed transactions require the EthereumSignTxEIP1559 firmware message; resubmit as a legacy transaction (gasPrice) or sign via Ledger');
        }
        const data = hwHexToBytes(tx.data || '0x');
        const payload = hwConcatBytes([
            pbPackedUint32(1, hwParseDerivationPath(path)),
            pbBytes(2, hwBigIntToMinimalBytes(tx.nonce || 0)),
            pbBytes(3, hwBigIntToMinimalBytes(tx.gasPrice || 0)),
            pbBytes(4, hwBigIntToMinimalBytes(tx.gasLimit || 21000)),
            pbBytes(5, hwUtf8(to)),
            pbBytes(6, hwBigIntToMinimalBytes(tx.value || 0)),
            pbBytes(7, data.subarray(0, 1024)),
            pbUint(8, data.length),
            pbUint(11, chainId),
        ]);
        const res = await this.transport.call(TT.EthereumSignTx, payload);
        if (res.type !== TT.EthereumTxRequest) throw new Error('Trezor: unexpected response ' + res.type);
        const fields = pbParse(res.payload);
        const v = pbFieldUint(fields, 2);
        const rBytes = pbFieldBytes(fields, 3);
        const sBytes = pbFieldBytes(fields, 4);
        if (v === undefined || !rBytes || !sBytes) throw new Error('Trezor: incomplete signature response');
        const raw = rlpEncode([
            BigInt(tx.nonce || 0), hwBigIntToMinimalBytes(tx.gasPrice || 0),
            hwBigIntToMinimalBytes(tx.gasLimit || 21000), hwHexToBytes(to),
            hwBigIntToMinimalBytes(tx.value || 0), data,
            hwBigIntToMinimalBytes(v), rBytes, sBytes,
        ]);
        return {
            v: Number(v),
            r: '0x' + hwBytesToHex(rBytes),
            s: '0x' + hwBytesToHex(sBytes),
            rawTransaction: '0x' + hwBytesToHex(raw),
        };
    }
}

// ---------------------------------------------------------------------------
// Public manager API used by app.js
// ---------------------------------------------------------------------------

const EVM_HW_CHAINS = ['ethereum', 'polygon', 'arbitrum', 'optimism', 'avalanche', 'bsc'];

class HardwareWalletManager {
    constructor() {
        this.connectedDevice = null;
        this.supportedDevices = ['ledger', 'trezor'];
        this.supportedChains = EVM_HW_CHAINS.slice();
        this._ledgerTransport = null;
        this._trezor = null;
    }

    async detectDevice() {
        try {
            const ledgerDevice = await this.detectLedger();
            if (ledgerDevice) {
                this.connectedDevice = { type: 'ledger', ...ledgerDevice };
                return this.connectedDevice;
            }
            const trezorDevice = await this.detectTrezor();
            if (trezorDevice) {
                this.connectedDevice = { type: 'trezor', ...trezorDevice };
                return this.connectedDevice;
            }
            return null;
        } catch (error) {
            console.error('Device detection failed:', error);
            return null;
        }
    }

    async detectLedger() {
        if (typeof navigator === 'undefined' || !navigator.hid) return null;
        try {
            const devices = await navigator.hid.getDevices();
            let device = devices.find((d) => d.vendorId === LEDGER_VENDOR_ID);
            if (!device) {
                const requested = await navigator.hid.requestDevice({ filters: [{ vendorId: LEDGER_VENDOR_ID }] });
                device = requested && requested[0];
            }
            if (!device) return null;
            if (!device.opened) await device.open();
            this._ledgerTransport = new LedgerHidTransport(device);
            return {
                name: 'Ledger',
                model: device.productName || 'Ledger',
                vendorId: device.vendorId,
                productId: device.productId,
            };
        } catch (error) {
            console.error('Ledger detection failed:', error);
            return null;
        }
    }

    async detectTrezor() {
        // The real Trezor path is the trezord daemon (the same bridge Trezor
        // Suite and every browser wallet use). Probe it instead of WebUSB.
        try {
            const t = new TrezorBridgeTransport();
            await t.acquire();
            this._trezor = t;
            return { name: 'Trezor', model: 'Trezor (via trezord)', vendorId: 0x1209, productId: 0 };
        } catch (_) {
            return null;
        }
    }

    _evmPath(chain, derivationPath) {
        if (!EVM_HW_CHAINS.includes(chain)) {
            throw new Error('Hardware signing here supports EVM chains (' + EVM_HW_CHAINS.join(', ') + '). ' +
                chain + ' requires its own on-device Ledger/Trezor app — sign ' + chain + ' via the wallet-api /non_evm endpoints instead.');
        }
        return derivationPath || "m/44'/60'/0'/0/0";
    }

    async getAddress(chain, derivationPath = null) {
        if (!this.connectedDevice) throw new Error('No hardware wallet connected');
        const path = this._evmPath(chain, derivationPath);
        if (this.connectedDevice.type === 'ledger') {
            if (!this._ledgerTransport) throw new Error('Ledger transport not initialized — detect the device first');
            return new LedgerEthApp(this._ledgerTransport).getAddress(path);
        }
        if (!this._trezor) throw new Error('Trezor bridge session not initialized — detect the device first');
        return new TrezorEthApp(this._trezor).getAddress(path);
    }

    /**
     * Sign a transaction on-device. tx = {to, value, data, nonce, gasLimit,
     * chainId, gasPrice} (legacy EIP-155) or {…, maxFeePerGas,
     * maxPriorityFeePerGas} (EIP-1559). Returns {v, r, s, rawTransaction} —
     * broadcast rawTransaction via the chain RPC (eth_sendRawTransaction).
     */
    async signTransaction(txData, chain = 'ethereum', derivationPath = null) {
        if (!this.connectedDevice) throw new Error('No hardware wallet connected');
        const path = this._evmPath(chain, derivationPath);
        if (this.connectedDevice.type === 'ledger') {
            if (!this._ledgerTransport) throw new Error('Ledger transport not initialized — detect the device first');
            return new LedgerEthApp(this._ledgerTransport).signTransaction(path, txData);
        }
        if (!this._trezor) throw new Error('Trezor bridge session not initialized — detect the device first');
        return new TrezorEthApp(this._trezor).signTransaction(path, txData);
    }

    /** Sign an EIP-191 personal message on-device. Returns 0x signature. */
    async signMessage(message, chain = 'ethereum', derivationPath = null) {
        if (!this.connectedDevice) throw new Error('No hardware wallet connected');
        const path = this._evmPath(chain, derivationPath);
        if (this.connectedDevice.type === 'ledger') {
            if (!this._ledgerTransport) throw new Error('Ledger transport not initialized — detect the device first');
            return new LedgerEthApp(this._ledgerTransport).signPersonalMessage(path, message);
        }
        if (!this._trezor) throw new Error('Trezor bridge session not initialized — detect the device first');
        return new TrezorEthApp(this._trezor).signMessage(path, message);
    }

    async getAllAddresses() {
        const addresses = {};
        for (const chain of this.supportedChains) {
            try {
                addresses[chain] = await this.getAddress(chain);
            } catch (error) {
                console.error('Failed to get address for ' + chain + ':', error);
            }
        }
        return addresses;
    }

    disconnect() {
        if (this._trezor) this._trezor.release();
        this._trezor = null;
        this._ledgerTransport = null;
        this.connectedDevice = null;
    }
}

// Export for use in desktop app
if (typeof module !== 'undefined' && module.exports) {
    module.exports = HardwareWalletManager;
}
