/**
 * PrivacyService - Web (React/TypeScript)
 *
 * NOTE: Stealth-address / ZK privacy primitives are NOT part of the canonical
 * MasterWallet backend contract (port 8450). This module exposes the public
 * surface without any client-side fake/stub cryptography: every operation that
 * would require server-side support returns a descriptive error so callers
 * fail loudly instead of receiving fabricated data.
 */

export interface StealthAddressResult {
  success: boolean;
  stealthAddress?: string;
  viewingKey?: string;
  ephemeralPublicKey?: string;
  error?: string;
}

export interface PrivacyError {
  success: false;
  error: string;
}

const unsupported = (op: string): PrivacyError => ({
  success: false,
  error: `${op} is not supported by the canonical MasterWallet backend`,
});

class PrivacyService {
  generateStealthAddress(_ownerAddress: string): StealthAddressResult | PrivacyError {
    return unsupported('generateStealthAddress');
  }
  scanStealthAddresses(_viewingKey: string): StealthAddressResult | PrivacyError {
    return unsupported('scanStealthAddresses');
  }
  spendStealth(_stealthAddress: string, _amount: string): StealthAddressResult | PrivacyError {
    return unsupported('spendStealth');
  }
  createConfidentialTransaction(
    _from: string, _to: string, _amount: string
  ): StealthAddressResult | PrivacyError {
    return unsupported('createConfidentialTransaction');
  }
  generateZKProof(_inputs: unknown): StealthAddressResult | PrivacyError {
    return unsupported('generateZKProof');
  }
  verifyZKProof(_proof: string): StealthAddressResult | PrivacyError {
    return unsupported('verifyZKProof');
  }
}

export const privacyService = new PrivacyService();
export default privacyService;
