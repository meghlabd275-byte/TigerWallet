/**
 * BiometricService - Web (React/TypeScript)
 *
 * NOTE: Client-side biometric templates are NOT part of the canonical
 * MasterWallet backend contract (port 8450). Real biometric unlock on the web
 * is performed via the platform WebAuthn/PasskeyService. This module exposes
 * the public typing surface only; methods that previously fabricated biometric
 * templates now return a descriptive error instead of fake data.
 */

export type BiometricType = 'fingerprint' | 'face' | 'voice' | 'iris' | 'behavioral';

export interface BiometricEnrollment {
  id: string;
  userId: string;
  type: BiometricType;
  template: string;
  deviceId: string;
  createdAt: number;
  lastUsedAt: number;
}

export interface BiometricVerificationResult {
  success: boolean;
  confidence?: number;
  error?: string;
}

export interface BiometricCapability {
  available: boolean;
  type: BiometricType;
  supported: boolean;
}

export interface BiometricSession {
  userId: string;
  type: BiometricType;
  startedAt: number;
}

class BiometricServiceClass {
  private static instance: BiometricServiceClass | null = null;
  private constructor() {}
  static getInstance(): BiometricServiceClass {
    if (!BiometricServiceClass.instance) BiometricServiceClass.instance = new BiometricServiceClass();
    return BiometricServiceClass.instance;
  }

  getCapabilities(): BiometricCapability[] {
    return [];
  }

  enroll(
    _userId: string, _type: BiometricType
  ): { success: false; error: string } {
    return { success: false, error: 'Biometric enrollment is not supported by the canonical MasterWallet backend; use WebAuthn via PasskeyService' };
  }

  verify(
    _userId: string, _type: BiometricType
  ): BiometricVerificationResult {
    return { success: false, error: 'Biometric verification is not supported by the canonical MasterWallet backend' };
  }

  startSession(_userId: string, _type: BiometricType): { success: false; error: string } {
    return { success: false, error: 'Biometric sessions are not supported by the canonical MasterWallet backend' };
  }
}

export const BiometricService = BiometricServiceClass;
export default BiometricServiceClass.getInstance();
