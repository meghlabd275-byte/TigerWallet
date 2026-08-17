/**
 * PasskeyService - Web (React/TypeScript)
 *
 * Real WebAuthn passkey registration/authentication. The browser performs the
 * navigator.credentials ceremony and the canonical MasterWallet backend acts
 * as the relying party (RP): it stores the credential public key (SPKI) on
 * registration and verifies the assertion signature server-side on
 * authentication. No fake verification — the backend re-derives
 * authenticatorData || SHA-256(clientDataJSON) and runs ECDSA VerifyASN1.
 */

import { masterWalletAPI } from '../api';

export interface PasskeyCredential {
  id: string;
  publicKey?: string;
  publicKeyAlgorithm?: number;
  transports?: AuthenticatorTransport[];
}

export interface PasskeyRegistrationResult {
  success: boolean;
  credential?: PasskeyCredential;
  passkeyId?: string;
  error?: string;
}

export interface PasskeyAuthenticationResult {
  success: boolean;
  credentialId?: string;
  authenticatorData?: string;
  verified?: boolean;
  error?: string;
}

interface PublicKeyCredentialRpEntity { name: string; id?: string }
interface PublicKeyCredentialUserEntity {
  id: string; name: string; displayName: string
}

function base64ToBase64Url(input: string): string {
  return input.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function bufferToBase64Url(buf: ArrayBuffer | ArrayBufferView): string {
  const bytes = buf instanceof ArrayBuffer
    ? new Uint8Array(buf)
    : new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
  let bin = '';
  bytes.forEach((b) => { bin += String.fromCharCode(b); });
  return base64ToBase64Url(btoa(bin));
}

function base64UrlToBuffer(b64url: string): Uint8Array<ArrayBuffer> {
  const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const padded = b64 + '='.repeat((4 - (b64.length % 4)) % 4);
  const bin = atob(padded);
  const buffer = new ArrayBuffer(bin.length);
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

class PasskeyServiceClass {
  private static instance: PasskeyServiceClass | null = null;
  private constructor() {}
  static getInstance(): PasskeyServiceClass {
    if (!PasskeyServiceClass.instance) PasskeyServiceClass.instance = new PasskeyServiceClass();
    return PasskeyServiceClass.instance;
  }

  isSupported(): boolean {
    return typeof window !== 'undefined'
      && typeof window.PublicKeyCredential !== 'undefined';
  }

  /**
   * Register a passkey: performs the WebAuthn create() ceremony, extracts the
   * SPKI public key + credential id, and POSTs them to the backend RP so the
   * server stores the credential for later assertion verification.
   */
  async register(
    masterWalletId: string,
    rp: PublicKeyCredentialRpEntity,
    user: PublicKeyCredentialUserEntity,
    challenge: string,
    label?: string
  ): Promise<PasskeyRegistrationResult> {
    if (!this.isSupported()) {
      return { success: false, error: 'WebAuthn is not supported in this browser' };
    }
    try {
      const publicKey: PublicKeyCredentialCreationOptions = {
        challenge: base64UrlToBuffer(challenge),
        rp,
        user: { id: base64UrlToBuffer(user.id), name: user.name, displayName: user.displayName },
        pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
        authenticatorSelection: { userVerification: 'preferred' },
        attestation: 'none',
      };
      const cred = await navigator.credentials.create({ publicKey }) as PublicKeyCredential | null;
      if (!cred) return { success: false, error: 'No credential returned' };
      const credId = bufferToBase64Url(cred.rawId);
      // Extract the SPKI public key from the attestation response.
      const response = cred.response as AuthenticatorAttestationResponse;
      const spki = response.getPublicKey
        ? bufferToBase64Url(response.getPublicKey() as ArrayBuffer)
        : '';
      const transports = response.getTransports
        ? (response.getTransports() as AuthenticatorTransport[])
        : ([] as AuthenticatorTransport[]);
      const signCount = response.getAuthenticatorData
        ? new DataView(response.getAuthenticatorData() as ArrayBuffer).getUint32(33)
        : 0;
      // Register the credential with the backend relying party.
      const reg = await masterWalletAPI.registerPasskey(masterWalletId, {
        credential_id: credId,
        public_key: spki,
        sign_count: signCount,
        transports,
        label,
      });
      return {
        success: true,
        credential: { id: credId, publicKey: spki, transports },
        passkeyId: reg.passkey_id,
      };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : String(err) };
    }
  }

  /**
   * Authenticate with a passkey: performs the WebAuthn get() ceremony, extracts
   * authenticatorData + clientDataJSON + signature, and POSTs them to the
   * backend RP which verifies the ECDSA signature against the stored P-256 key.
   */
  async authenticate(
    masterWalletId: string,
    credentialIds: string[],
    challenge: string
  ): Promise<PasskeyAuthenticationResult> {
    if (!this.isSupported()) {
      return { success: false, error: 'WebAuthn is not supported in this browser' };
    }
    try {
      const publicKey: PublicKeyCredentialRequestOptions = {
        challenge: base64UrlToBuffer(challenge),
        allowCredentials: credentialIds.map((id) => ({
          type: 'public-key',
          id: base64UrlToBuffer(id),
          transports: ['internal'] as AuthenticatorTransport[],
        })),
        userVerification: 'preferred',
      };
      const assertion = await navigator.credentials.get({ publicKey }) as PublicKeyCredential | null;
      if (!assertion) return { success: false, error: 'No assertion returned' };
      const credId = bufferToBase64Url(assertion.rawId);
      const response = assertion.response as AuthenticatorAssertionResponse;
      const authenticatorData = bufferToBase64Url(response.authenticatorData as ArrayBuffer);
      const clientDataJSON = bufferToBase64Url(response.clientDataJSON as ArrayBuffer);
      const signature = bufferToBase64Url(response.signature as ArrayBuffer);
      // Server-side verification against the stored public key.
      const verify = await masterWalletAPI.verifyPasskeyAssertion(masterWalletId, {
        credential_id: credId,
        authenticator_data: authenticatorData,
        client_data_json: clientDataJSON,
        signature,
      });
      return {
        success: true,
        credentialId: credId,
        authenticatorData,
        verified: verify.verified,
      };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : String(err) };
    }
  }

  /** List registered passkeys from the backend RP. */
  async listRegistered(masterWalletId: string) {
    return masterWalletAPI.listPasskeys(masterWalletId);
  }

  /** Delete a registered passkey from the backend RP. */
  async remove(masterWalletId: string, credentialId: string): Promise<void> {
    await masterWalletAPI.deletePasskey(masterWalletId, credentialId);
  }
}

export const PasskeyService = PasskeyServiceClass;
export default PasskeyServiceClass.getInstance();
