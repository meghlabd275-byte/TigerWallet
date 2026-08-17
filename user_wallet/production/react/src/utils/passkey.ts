/**
 * WebAuthn (passkey) helpers used by the wallet UI.
 *
 * These wrap the real `navigator.credentials` API. No mock data: if the
 * browser does not support WebAuthn, the helpers reject. The returned
 * values (credentialId, publicKey) are base64url strings ready to send to
 * the wallet-api backend.
 */

/** True when the current browser exposes the WebAuthn PublicKeyCredential API. */
export function passkeySupported(): boolean {
  return typeof window !== 'undefined' && typeof window.PublicKeyCredential !== 'undefined';
}

/** Encode an ArrayBuffer/Uint8Array as a base64url string (no padding). */
export function bufferToBase64url(buffer: ArrayBuffer | Uint8Array): string {
  const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  const b64 = btoa(binary);
  // base64 -> base64url: replace +/, strip padding
  return b64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/** Decode a base64url string into a Uint8Array (used for challenge/user id). */
export function base64urlToBuffer(b64url: string): Uint8Array {
  const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const padded = b64 + '==='.slice((b64.length + 3) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

function randomChallenge(): Uint8Array {
  const challenge = new Uint8Array(32);
  crypto.getRandomValues(challenge);
  return challenge;
}

export interface PasskeyCreationResult {
  credentialId: string; // base64url
  publicKey: string; // base64url SPKI
  rawId: ArrayBuffer;
}

/**
 * Create a new passkey via WebAuthn and return base64url credentialId and SPKI.
 * Throws if WebAuthn is unsupported or the user cancels the ceremony.
 */
export async function createPasskey(
  userLabel: string,
  userId?: string
): Promise<PasskeyCreationResult> {
  if (!passkeySupported()) {
    throw new Error('WebAuthn is not supported in this browser');
  }
  if (!window.isSecureContext) {
    throw new Error('WebAuthn requires a secure context (HTTPS or localhost)');
  }

  const userIdBuffer = userId
    ? base64urlToBuffer(userId)
    : (() => {
        const id = new Uint8Array(16);
        crypto.getRandomValues(id);
        return id;
      })();

  const publicKey: PublicKeyCredentialCreationOptions = {
    challenge: randomChallenge() as BufferSource,
    rp: {
      name: 'TigerWallet',
      // id omitted so the credential is scoped to the current effective domain
    },
    user: {
      id: userIdBuffer as BufferSource,
      name: userLabel || 'tiger-wallet-user',
      displayName: userLabel || 'TigerWallet User',
    },
    pubKeyCredParams: [
      { type: 'public-key', alg: -7 }, // ES256
      { type: 'public-key', alg: -257 }, // RS256
    ],
    authenticatorSelection: {
      authenticatorAttachment: 'platform',
      residentKey: 'preferred',
      userVerification: 'preferred',
    },
    timeout: 60000,
    attestation: 'none',
  };

  const credential = (await navigator.credentials.create({ publicKey })) as PublicKeyCredential | null;
  if (!credential) {
    throw new Error('Passkey creation was cancelled');
  }

  const attestation = credential.response as AuthenticatorAttestationResponse;
  const spki = attestation.getPublicKey();
  if (!spki) {
    throw new Error('Passkey creation succeeded but no public key was returned');
  }

  return {
    credentialId: bufferToBase64url(credential.rawId),
    publicKey: bufferToBase64url(spki),
    rawId: credential.rawId,
  };
}

export interface PasskeyAssertionResult {
  credentialId: string; // base64url
  assertion: string; // base64url signature
  authenticatorData: string; // base64url
  clientData: string; // base64url
}

/**
 * Request a passkey assertion (authentication) for the given credential id.
 * Used for passwordless unlock.
 */
export async function assertPasskey(
  credentialId: string
): Promise<PasskeyAssertionResult> {
  if (!passkeySupported()) {
    throw new Error('WebAuthn is not supported in this browser');
  }
  if (!window.isSecureContext) {
    throw new Error('WebAuthn requires a secure context (HTTPS or localhost)');
  }

  const publicKey: PublicKeyCredentialRequestOptions = {
    challenge: randomChallenge() as BufferSource,
    allowCredentials: [
      {
        id: base64urlToBuffer(credentialId) as BufferSource,
        type: 'public-key',
        transports: ['internal', 'hybrid', 'usb', 'ble', 'nfc'],
      },
    ],
    userVerification: 'preferred',
    timeout: 60000,
  };

  const credential = (await navigator.credentials.get({ publicKey })) as PublicKeyCredential | null;
  if (!credential) {
    throw new Error('Passkey authentication was cancelled');
  }

  const assertion = credential.response as AuthenticatorAssertionResponse;
  return {
    credentialId: bufferToBase64url(credential.rawId),
    assertion: bufferToBase64url(assertion.signature),
    authenticatorData: bufferToBase64url(assertion.authenticatorData),
    clientData: bufferToBase64url(assertion.clientDataJSON),
  };
}
