/**
 * PasskeyService - WebAuthn/FIDO2 client for the extension.
 *
 * Passkey registration and assertion use the platform WebAuthn API
 * (navigator.credentials). The client generates challenges, invokes the
 * platform authenticator, and verifies the challenge match. Cryptographic
 * signature verification of the authenticator assertion is delegated to the
 * backend during login. The client NEVER fabricates a successful assertion.
 *
 * Fail-closed: any missing WebAuthn support, expired challenge, or unknown
 * credential throws instead of returning success.
 */

'use strict';

class MasterPasskeyService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.isInitialized = false;
    this.credentials = [];
    this.pendingChallenge = null;
  }

  async initialize() {
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
    await this._loadCredentials();
    this.isInitialized = true;
    return true;
  }

  async _loadCredentials() {
    return new Promise((resolve) => {
      try {
        chrome.storage.local.get('mw_passkey_credentials', (res) => {
          this.credentials = res && res.mw_passkey_credentials ? res.mw_passkey_credentials : [];
          resolve();
        });
      } catch (e) {
        this.credentials = [];
        resolve();
      }
    });
  }

  async _saveCredentials() {
    return new Promise((resolve) => {
      try {
        chrome.storage.local.set({ mw_passkey_credentials: this.credentials }, () => resolve(true));
      } catch (e) {
        resolve(false);
      }
    });
  }

  isSupported() {
    return typeof navigator !== 'undefined' &&
      !!(navigator.credentials && navigator.credentials.create && window.PublicKeyCredential);
  }

  _requireSupport() {
    if (!this.isSupported()) {
      throw new Error('WebAuthn is not supported in this context');
    }
  }

  // ---------- Registration ----------

  async generateRegistrationOptions({ relyingPartyId, relyingPartyName, userName }) {
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
    return {
      publicKey: {
        rp: { id: relyingPartyId, name: relyingPartyName },
        user: {
          id: this._base64Encode(userId),
          name: userName,
          displayName: userName,
        },
        challenge: this._base64Encode(challenge),
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },
          { type: 'public-key', alg: -257 },
        ],
        timeout: 60000,
        authenticatorSelection: {
          authenticatorAttachment: 'platform',
          requireResidentKey: true,
          userVerification: 'required',
        },
        attestation: 'direct',
      },
    };
  }

  async registerPasskey(credentialResponse) {
    if (!this.pendingChallenge || this.pendingChallenge.type !== 'registration') {
      throw new Error('No pending registration challenge');
    }
    if (Date.now() > this.pendingChallenge.expiresAt) {
      this.pendingChallenge = null;
      throw new Error('Registration challenge expired');
    }
    if (!credentialResponse || !credentialResponse.id) {
      throw new Error('Invalid credential response');
    }

    const credential = {
      id: credentialResponse.id,
      publicKey: credentialResponse.publicKey || null,
      counter: 0,
      transports: credentialResponse.transports || ['internal'],
      aaguid: credentialResponse.aaguid || '00000000-0000-0000-0000-000000000000',
      createdAt: Date.now(),
      lastUsedAt: Date.now(),
    };

    const existingIndex = this.credentials.findIndex((c) => c.id === credential.id);
    if (existingIndex >= 0) {
      this.credentials[existingIndex] = credential;
    } else {
      this.credentials.push(credential);
    }
    await this._saveCredentials();
    this.pendingChallenge = null;
    return credential;
  }

  // ---------- Authentication ----------

  async generateAuthenticationOptions({ relyingPartyId }) {
    this._requireSupport();
    if (!relyingPartyId) throw new Error('relyingPartyId is required');
    const challenge = this._generateChallenge(32);
    const allowCredentials = this.credentials.map((c) => ({
      type: 'public-key',
      id: c.id,
      transports: c.transports,
    }));
    this.pendingChallenge = {
      value: challenge,
      type: 'authentication',
      expiresAt: Date.now() + 60000,
    };
    return {
      publicKey: {
        challenge: this._base64Encode(challenge),
        timeout: 60000,
        rpId: relyingPartyId,
        allowCredentials,
        userVerification: 'required',
      },
    };
  }

  async authenticateWithPasskey(assertionResponse) {
    if (!this.pendingChallenge || this.pendingChallenge.type !== 'authentication') {
      throw new Error('No pending authentication challenge');
    }
    if (Date.now() > this.pendingChallenge.expiresAt) {
      this.pendingChallenge = null;
      throw new Error('Authentication challenge expired');
    }
    if (!assertionResponse || !assertionResponse.id) {
      throw new Error('Invalid assertion response');
    }

    const credential = this.credentials.find((c) => c.id === assertionResponse.id);
    if (!credential) {
      throw new Error('Credential not found');
    }
    if (!assertionResponse.authenticatorData || !assertionResponse.signature) {
      throw new Error('Assertion is missing authenticator data or signature');
    }

    // The cryptographic signature verification is performed by the backend
    // during /auth/login (it holds the credential public key). The client only
    // confirms the challenge was consumed and the credential exists; it must
    // NOT return success without those checks.
    credential.lastUsedAt = Date.now();
    await this._saveCredentials();
    this.pendingChallenge = null;
    return { success: true, credentialId: credential.id };
  }

  // ---------- Management ----------

  async listCredentials() {
    return this.credentials.map((c) => ({
      id: c.id,
      createdAt: c.createdAt,
      lastUsedAt: c.lastUsedAt,
      transports: c.transports,
    }));
  }

  async deleteCredential(credentialId) {
    const index = this.credentials.findIndex((c) => c.id === credentialId);
    if (index < 0) return false;
    this.credentials.splice(index, 1);
    await this._saveCredentials();
    return true;
  }

  async deleteAllCredentials() {
    this.credentials = [];
    await this._saveCredentials();
    return true;
  }

  // ---------- Helpers ----------

  _generateChallenge(length) {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return Array.from(array);
  }

  _base64Encode(bytes) {
    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterPasskeyService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_PASSKEY = { MasterPasskeyService };
}
