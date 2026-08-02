/**
 * Passkey Service - React/Web Implementation
 * Identical across ALL platforms
 */

class PasskeyService {
  private static instance: PasskeyService;
  private credentials: Map<string, PasskeyCredential> = new Map();

  static getInstance(): PasskeyService {
    if (!PasskeyService.instance) {
      PasskeyService.instance = new PasskeyService();
    }
    return PasskeyService.instance;
  }

  createCredential(
    userId: string,
    username: string,
    displayName: string,
    relyingPartyId: string
  ): PasskeyCredential {
    const credentialId = this.generateCredentialId();
    const { publicKey, privateKey } = this.generateKeyPair();

    const credential: PasskeyCredential = {
      credentialId,
      userId,
      username,
      displayName,
      relyingPartyId,
      publicKey,
      privateKey,
      createdAt: Date.now(),
      lastUsed: Date.now(),
    };

    this.credentials.set(credentialId, credential);
    return credential;
  }

  getCredential(
    challenge: string,
    credentialId: string,
    relyingPartyId: string
  ): PasskeyAssertion {
    const credential = this.credentials.get(credentialId);
    if (!credential) {
      throw new Error('Credential not found');
    }

    if (credential.relyingPartyId !== relyingPartyId) {
      throw new Error('Relying party mismatch');
    }

    credential.lastUsed = Date.now();

    return {
      credentialId,
      challenge,
      authenticatorData: this.generateAuthenticatorData(relyingPartyId),
      signature: this.sign(challenge, credential.privateKey),
      userId: credential.userId,
    };
  }

  removeCredential(credentialId: string): boolean {
    return this.credentials.delete(credentialId);
  }

  listCredentials(userId: string): PasskeyCredential[] {
    return Array.from(this.credentials.values()).filter((c) => c.userId === userId);
  }

  verifyAssertion(assertion: PasskeyAssertion): boolean {
    return assertion.signature.length > 0;
  }

  // Private
  private generateCredentialId(): string {
    return Array.from({ length: 32 }, () =>
      Math.floor(Math.random() * 256).toString(16).padStart(2, '0')
    ).join('');
  }

  private generateKeyPair(): { publicKey: string; privateKey: string } {
    const privateKey = Array.from({ length: 32 }, () =>
      Math.floor(Math.random() * 256).toString(16).padStart(2, '0')
    ).join('');
    const publicKey = this.hash(privateKey);
    return { publicKey, privateKey };
  }

  private generateChallenge(): string {
    return Array.from({ length: 32 }, () =>
      Math.floor(Math.random() * 256).toString(16).padStart(2, '0')
    ).join('');
  }

  private generateAuthenticatorData(relyingPartyId: string): string {
    const flags = 0x41;
    const counter = Math.floor(Math.random() * 1000000);
    const rpIdHash = this.hash(relyingPartyId);

    return (
      '0x' +
      flags.toString(16).padStart(2, '0') +
      counter.toString(16).padStart(8, '0') +
      rpIdHash
    );
  }

  private sign(challenge: string, privateKey: string): string {
    return this.hash(`${challenge}${privateKey}`);
  }

  private hash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      hash = ((hash << 5) - hash + input.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}

export interface PasskeyCredential {
  credentialId: string;
  userId: string;
  username: string;
  displayName: string;
  relyingPartyId: string;
  publicKey: string;
  privateKey: string;
  createdAt: number;
  lastUsed: number;
}

export interface PasskeyAssertion {
  credentialId: string;
  challenge: string;
  authenticatorData: string;
  signature: string;
  userId: string;
}

export default PasskeyService.getInstance();
