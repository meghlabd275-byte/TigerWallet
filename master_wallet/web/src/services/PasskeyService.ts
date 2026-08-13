/**
 * PasskeyService - Web (React/TypeScript)
 *
 * Real WebAuthn passkey registration/authentication backed by the browser
 * `navigator.credentials` API. No server-side attestation verification is
 * performed (the canonical MasterWallet backend contract has no passkey
 * endpoint); this module performs genuine client-side WebAuthn ceremonies only.
 */

export interface PasskeyCredential {
  id: string;
  publicKey?: string;
  publicKeyAlgorithm?: number;
  transports?: AuthenticatorTransport[];
}

export interface PasskeyRegistrationResult {
  success: boolean;
  credential?: PasskeyCredential;
  error?: string;
}

export interface PasskeyAuthenticationResult {
  success: boolean;
  credentialId?: string;
  authenticatorData?: string;
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

  async register(
    rp: PublicKeyCredentialRpEntity,
    user: PublicKeyCredentialUserEntity,
    challenge: string
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
      };
      const cred = await navigator.credentials.create({ publicKey }) as PublicKeyCredential | null;
      if (!cred) return { success: false, error: 'No credential returned' };
      const credId = bufferToBase64Url(cred.rawId);
      return { success: true, credential: { id: credId } };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : String(err) };
    }
  }

  async authenticate(
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
      return { success: true, credentialId: bufferToBase64Url(assertion.rawId) };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : String(err) };
    }
  }
}

export const PasskeyService = PasskeyServiceClass;
export default PasskeyServiceClass.getInstance();
