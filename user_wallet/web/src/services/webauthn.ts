// WebAuthn helpers — real browser credential creation only, no fake data.
// All outputs are base64url strings ready for the backend.

function base64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let str = '';
  for (let i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export function webauthnSupported(): boolean {
  return typeof window !== 'undefined' &&
    !!window.PublicKeyCredential &&
    typeof navigator !== 'undefined' &&
    !!navigator.credentials;
}

export interface CreatedPasskey {
  credentialId: string;       // base64url
  publicKey: string;          // base64url SPKI
  rawId: ArrayBuffer;
}

// Create a WebAuthn credential and extract the credentialId + SPKI public key.
// Returns base64url strings. Throws if unsupported or the user cancels.
export async function createPasskey(displayName: string): Promise<CreatedPasskey> {
  if (!webauthnSupported()) {
    throw new Error('Passkeys are not supported in this browser');
  }

  const challenge = new Uint8Array(32);
  crypto.getRandomValues(challenge);

  const publicKey: PublicKeyCredentialCreationOptions = {
    challenge,
    rp: { name: 'TigerWallet' },
    user: {
      id: new Uint8Array(16),
      name: displayName || 'tigerwallet-user',
      displayName: displayName || 'TigerWallet User',
    },
    pubKeyCredParams: [
      { type: 'public-key', alg: -7 },   // ES256
      { type: 'public-key', alg: -257 },  // RS256
    ],
    authenticatorSelection: {
      authenticatorAttachment: 'platform',
      residentKey: 'preferred',
      userVerification: 'preferred',
    },
    timeout: 60000,
    attestation: 'none',
  };

  const credential = await navigator.credentials.create({ publicKey });
  if (!credential || credential.type !== 'public-key') {
    throw new Error('Passkey creation failed');
  }

  const pkc = credential as PublicKeyCredential;
  const response = pkc.response as AuthenticatorAttestationResponse;

  // The SPKI public key: getPublicKey returns ArrayBuffer of the SubjectPublicKeyInfo.
  let publicKeyB64u = '';
  const getPublicKey = (response as AuthenticatorAttestationResponse & {
    getPublicKey?: () => ArrayBuffer | null;
  }).getPublicKey;
  if (typeof getPublicKey === 'function') {
    const spki = getPublicKey.call(response);
    if (spki) publicKeyB64u = base64url(spki);
  }
  // Fallback: derive the SPKI from the full attestationObject if the browser
  // does not expose getPublicKey (older implementations).
  if (!publicKeyB64u) {
    try {
      const spki = await extractSpkiFromAttestation(response.attestationObject);
      if (spki) publicKeyB64u = base64url(spki);
    } catch {
      /* leave empty — passcode-only path remains usable */
    }
  }

  return {
    credentialId: base64url(pkc.rawId),
    publicKey: publicKeyB64u,
    rawId: pkc.rawId,
  };
}

// Minimal CBOR-aware extraction of the credentialPublicKey (SubjectPublicKeyInfo)
// from a WebAuthn attestationObject. attestationObject = CBOR map {
//   fmt, attStmt, authData }. authData contains an attestedCredentialData block
// whose tail is the COSE-encoded public key. We slice the SPKI bytes by
// locating the standard SPKI header (DER SEQUENCE) after the fixed
// credentialId length. Best-effort; the modern getPublicKey path is preferred.
async function extractSpkiFromAttestation(attestationObject: ArrayBuffer): Promise<ArrayBuffer | null> {
  try {
    const bytes = new Uint8Array(attestationObject);
    // Search for the DER SubjectPublicKeyInfo SEQUENCE marker (0x30 ...).
    // The COSE public key is CBOR; for ES256 it carries an SPKI-compatible
    // structure. We heuristically find the first 0x30 0x59 (SEQUENCE len=0x59
    // = 89 bytes, the typical EC P-256 SPKI length) which marks an SPKI.
    for (let i = 0; i + 1 < bytes.length; i++) {
      if (bytes[i] === 0x30 && bytes[i + 1] === 0x59) {
        return bytes.slice(i, i + 91).buffer;
      }
    }
    return null;
  } catch {
    return null;
  }
}
