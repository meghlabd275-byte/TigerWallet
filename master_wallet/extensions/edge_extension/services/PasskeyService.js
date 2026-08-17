/**
 * PasskeyService - WebAuthn/FIDO2 client for the extension.
 *
 * The backend is the WebAuthn relying party (RP): it stores credential public
 * keys and verifies assertions server-side. The client performs the
 * navigator.credentials ceremonies (create/get) and POSTs the resulting
 * credential/assertion to the backend's relying-party routes:
 *   POST   /api/v1/master-wallet/:id/passkey/register
 *   GET    /api/v1/master-wallet/:id/passkey/credentials
 *   DELETE /api/v1/master-wallet/:id/passkey/credentials/:credId
 *   POST   /api/v1/master-wallet/:id/passkey/verify-assertion
 *
 * Challenges and RP options are generated client-side (the backend exposes no
 * option-generation route); only the resulting ceremony output is shipped to
 * the backend. The client NEVER fabricates a successful assertion and returns
 * success ONLY when the backend reports `verified: true`.
 *
 * Fail-closed: any missing WebAuthn support, expired challenge, or backend
 * verification failure throws instead of returning success.
 */

'use strict';

// UMD: resolve the MasterWalletService that owns authedFetch + the passkey
// routes. In MV3 service workers all modules share globalThis via
// importScripts; under node/tests it is require()'d.
function _resolveMasterWalletService() {
  if (typeof require === 'function') {
    try {
      const mod = require('./masterWalletService.js');
      if (mod && mod.masterWalletService) return mod.masterWalletService;
    } catch (_) { /* fall through to globalThis */ }
  }
  if (typeof globalThis !== 'undefined' && globalThis.MW_SERVICE && globalThis.MW_SERVICE.masterWalletService) {
    return globalThis.MW_SERVICE.masterWalletService;
  }
  throw new Error('MasterWalletService not available; cannot reach passkey relying-party routes');
}

function _storageArea() {
  if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) return chrome.storage.local;
  if (typeof browser !== 'undefined' && browser.storage && browser.storage.local) return browser.storage.local;
  return null;
}

class MasterPasskeyService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.isInitialized = false;
    this.credentials = [];
    this.pendingChallenge = null;
  }

  _svc() {
    return _resolveMasterWalletService();
  }

  async initialize() {
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
    // The authoritative credential list lives in the backend; mirror it locally
    // so allowCredentials can be built without an extra round-trip per get().
    try {
      this.credentials = await this._svc().listPasskeys(this.masterWalletId);
    } catch (_) {
      // Backend may be unreachable during init; do not fabricate credentials.
      this.credentials = [];
    }
    this.isInitialized = true;
    return true;
  }

  isSupported() {
    return typeof navigator !== 'undefined' &&
      !!(navigator.credentials && navigator.credentials.create && navigator.credentials.get &&
        (typeof window !== 'undefined' ? window.PublicKeyCredential : globalThis.PublicKeyCredential));
  }

  _requireSupport() {
    if (!this.isSupported()) {
      throw new Error('WebAuthn is not supported in this context');
    }
  }

  // ---------- Registration ----------

  // Build PublicKeyCredentialCreationOptions. The challenge is generated and
  // remembered client-side; the backend only consumes the ceremony output.
  async _buildRegistrationOptions({ relyingPartyId, relyingPartyName, userName, displayName, label }) {
    this._requireSupport();
    if (!relyingPartyId || !relyingPartyName || !userName) {
      throw new Error('relyingPartyId, relyingPartyName and userName are required');
    }
    const challenge = this._generateChallenge(32);
    const userId = this._generateChallenge(32);
    this.pendingChallenge = {
      value: challenge,
      type: 'registration',
      expiresAt: Date.now() + 60000,
    };
    const authenticatorSelection = {
      authenticatorAttachment: 'platform',
      residentKey: 'required',
      userVerification: 'required',
    };
    // Safari < 16.4 rejects `residentKey`; fall back to the legacy flag if the
    // platform throws on the newer key during option construction.
    try {
      const probe = { ...authenticatorSelection };
      PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable &&
        await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
      authenticatorSelection.requireResidentKey = true;
    } catch (_) { /* keep authenticatorSelection as-is */ }
    return {
      publicKey: {
        rp: { id: relyingPartyId, name: relyingPartyName },
        user: {
          id: new Uint8Array(userId),
          name: userName,
          displayName: displayName || userName,
        },
        challenge: new Uint8Array(challenge),
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },
          { type: 'public-key', alg: -257 },
        ],
        timeout: 60000,
        authenticatorSelection,
        attestation: 'none',
        // surface an optional label to the caller; the backend stores it.
        extensions: label ? { appLabel: label } : undefined,
      },
    };
  }

  /**
   * Full WebAuthn registration ceremony.
   * Runs navigator.credentials.create, extracts the SPKI public key via
   * credential.response.getPublicKey(), and POSTs credential_id + public_key
   * (+ transports, sign_count, label) to /passkey/register.
   *
   * @param {object} opts - { relyingPartyId, relyingPartyName, userName,
   *   displayName?, label? }
   * @returns {Promise<{passkey_id, credential_id, registered}>} backend result
   */
  async register(opts) {
    const options = await this._buildRegistrationOptions(opts || {});
    let credential;
    try {
      credential = await navigator.credentials.create(options);
    } finally {
      // Consume the challenge regardless of ceremony outcome.
      this.pendingChallenge = null;
    }
    if (!credential || !credential.id) {
      throw new Error('WebAuthn ceremony returned no credential');
    }

    const resp = credential.response || {};
    // SPKI public key (SubjectPublicKeyInfo). getPublicKey() is the modern
    // accessor; fall back to the raw `publicKey` field for older runtimes.
    let publicKeyBytes = null;
    if (typeof resp.getPublicKey === 'function') {
      const spki = resp.getPublicKey();
      if (spki) publicKeyBytes = spki;
    }
    if (!publicKeyBytes && resp.publicKey) publicKeyBytes = resp.publicKey;
    if (!publicKeyBytes) {
      throw new Error('Credential response is missing a SPKI public key');
    }
    const publicKeyB64 = this._bufToB64url(publicKeyBytes);

    const transports = (typeof credential.getTransports === 'function')
      ? (credential.getTransports() || [])
      : (credential.transports || []);

    let signCount = 0;
    if (typeof resp.getAuthenticatorData === 'function') {
      try {
        const ad = resp.getAuthenticatorData();
        if (ad && ad.byteLength >= 37) {
          const view = new DataView(ad instanceof ArrayBuffer ? ad : ad.buffer);
          signCount = view.getUint32(ad.byteLength - 4 || 0);
        }
      } catch (_) { /* sign_count is advisory; default 0 */ }
    }

    const result = await this._svc().registerPasskey(this.masterWalletId, {
      credential_id: credential.id,
      public_key: publicKeyB64,
      sign_count: signCount,
      transports,
      label: opts && opts.label ? opts.label : undefined,
    });
    // Refresh the local mirror from the authoritative backend list.
    try { this.credentials = await this._svc().listPasskeys(this.masterWalletId); } catch (_) { /* ignore */ }
    return result;
  }

  // Backward-compatible alias: callers that already ran navigator.credentials
  // create() themselves can hand the raw ceremony credential here.
  async registerPasskey(credential, opts) {
    if (!this.pendingChallenge || this.pendingChallenge.type !== 'registration') {
      throw new Error('No pending registration challenge');
    }
    if (Date.now() > this.pendingChallenge.expiresAt) {
      this.pendingChallenge = null;
      throw new Error('Registration challenge expired');
    }
    const resp = credential && credential.response ? credential.response : {};
    let publicKeyBytes = null;
    if (typeof resp.getPublicKey === 'function') {
      const spki = resp.getPublicKey();
      if (spki) publicKeyBytes = spki;
    }
    if (!publicKeyBytes && resp.publicKey) publicKeyBytes = resp.publicKey;
    if (!publicKeyBytes) {
      this.pendingChallenge = null;
      throw new Error('Credential response is missing a SPKI public key');
    }
    const transports = (typeof credential.getTransports === 'function')
      ? (credential.getTransports() || [])
      : (credential.transports || []);
    this.pendingChallenge = null;
    const result = await this._svc().registerPasskey(this.masterWalletId, {
      credential_id: credential.id,
      public_key: this._bufToB64url(publicKeyBytes),
      sign_count: 0,
      transports,
      label: opts && opts.label ? opts.label : undefined,
    });
    try { this.credentials = await this._svc().listPasskeys(this.masterWalletId); } catch (_) { /* ignore */ }
    return result;
  }

  // ---------- Authentication ----------

  async _buildAuthenticationOptions({ relyingPartyId, allowCredentialIds }) {
    this._requireSupport();
    if (!relyingPartyId) throw new Error('relyingPartyId is required');
    const challenge = this._generateChallenge(32);
    const allowCredentials = (Array.isArray(allowCredentialIds) ? allowCredentialIds : this.credentials)
      .map((c) => {
        const id = typeof c === 'string' ? c : (c && c.credential_id) || (c && c.id);
        const transports = (c && c.transports) || ['internal'];
        if (!id) return null;
        return { type: 'public-key', id: this._b64urlToBuf(id), transports };
      })
      .filter(Boolean);
    this.pendingChallenge = {
      value: challenge,
      type: 'authentication',
      expiresAt: Date.now() + 60000,
    };
    return {
      publicKey: {
        challenge: new Uint8Array(challenge),
        timeout: 60000,
        rpId: relyingPartyId,
        // Empty allowCredentials => discoverable credentials (resident keys).
        allowCredentials: allowCredentials.length ? allowCredentials : undefined,
        userVerification: 'required',
      },
    };
  }

  /**
   * Full WebAuthn authentication ceremony.
   * Runs navigator.credentials.get, then POSTs authenticator_data +
   * client_data_json + signature (+ credential_id) to
   * /passkey/verify-assertion for server-side verification. Returns success
   * ONLY when the backend reports `verified: true`.
   *
   * @param {object} opts - { relyingPartyId, allowCredentialIds? }
   * @returns {Promise<{verified:true, credential_id}>}
   */
  async authenticate(opts) {
    const options = await this._buildAuthenticationOptions(opts || {});
    let assertion;
    try {
      assertion = await navigator.credentials.get(options);
    } finally {
      this.pendingChallenge = null;
    }
    if (!assertion || !assertion.id) {
      throw new Error('WebAuthn ceremony returned no assertion');
    }
    const resp = assertion.response || {};
    if (!resp.authenticatorData || !resp.clientDataJSON || !resp.signature) {
      throw new Error('Assertion is missing authenticator data, client data, or signature');
    }

    const result = await this._svc().verifyPasskeyAssertion(this.masterWalletId, {
      credential_id: assertion.id,
      authenticator_data: this._bufToB64url(resp.authenticatorData),
      client_data_json: this._bufToB64url(resp.clientDataJSON),
      signature: this._bufToB64url(resp.signature),
    });

    if (!result || !result.verified) {
      // Fail-closed: never report success on a failed/absent backend verdict.
      throw new Error('Backend did not verify the passkey assertion');
    }
    return { verified: true, credential_id: result.credential_id || assertion.id };
  }

  // Backward-compatible alias for callers that already ran navigator.credentials
  // get() themselves. Still requires the backend to verify the assertion.
  async authenticateWithPasskey(assertion) {
    if (!assertion || !assertion.id) throw new Error('Invalid assertion response');
    const resp = assertion.response || {};
    if (!resp.authenticatorData || !resp.clientDataJSON || !resp.signature) {
      throw new Error('Assertion is missing authenticator data, client data, or signature');
    }
    const result = await this._svc().verifyPasskeyAssertion(this.masterWalletId, {
      credential_id: assertion.id,
      authenticator_data: this._bufToB64url(resp.authenticatorData),
      client_data_json: this._bufToB64url(resp.clientDataJSON),
      signature: this._bufToB64url(resp.signature),
    });
    if (!result || !result.verified) {
      throw new Error('Backend did not verify the passkey assertion');
    }
    return { success: true, verified: true, credentialId: result.credential_id || assertion.id };
  }

  // ---------- Management (backend is authoritative) ----------

  async listCredentials() {
    return this._svc().listPasskeys(this.masterWalletId);
  }

  async deleteCredential(credentialId) {
    if (!credentialId) throw new Error('credentialId is required');
    return this._svc().deletePasskey(this.masterWalletId, credentialId);
  }

  async deleteAllCredentials() {
    const creds = await this._svc().listPasskeys(this.masterWalletId);
    for (const c of (Array.isArray(creds) ? creds : [])) {
      const id = c && (c.credential_id || c.id);
      if (id) await this._svc().deletePasskey(this.masterWalletId, id);
    }
    return true;
  }

  // ---------- Helpers ----------

  _generateChallenge(length) {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return array;
  }

  // ArrayBuffer / TypedArray -> base64url (no padding).
  _bufToB64url(buf) {
    const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
    let binary = '';
    for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
    const b64 = (typeof btoa === 'function')
      ? btoa(binary)
      : Buffer.from(bytes).toString('base64');
    return b64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  // base64url string -> Uint8Array (for allowCredentials ids).
  _b64urlToBuf(b64url) {
    const b64 = String(b64url).replace(/-/g, '+').replace(/_/g, '/');
    const pad = b64.length % 4 ? '='.repeat(4 - (b64.length % 4)) : '';
    const binary = (typeof atob === 'function')
      ? atob(b64 + pad)
      : Buffer.from(b64 + pad, 'base64').toString('binary');
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    return bytes;
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterPasskeyService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_PASSKEY = { MasterPasskeyService };
}
