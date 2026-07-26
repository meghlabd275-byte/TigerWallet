/**
 * Browser Adapter for WebAuthn Support Detection
 */

export class BrowserAdapter {
  /**
   * Check if WebAuthn is supported
   */
  isWebAuthnSupported(): boolean {
    return !!(
      typeof navigator !== 'undefined' &&
      navigator.credentials?.create &&
      navigator.credentials?.get
    );
  }

  /**
   * Check if running in secure context (HTTPS or localhost)
   */
  isSecureContext(): boolean {
    if (typeof window === 'undefined') return false;
    return window.isSecureContext === true;
  }

  /**
   * Get browser name
   */
  getBrowserName(): string {
    if (typeof navigator === 'undefined') return 'Unknown';
    
    const ua = navigator.userAgent;
    if (ua.includes('Chrome') && !ua.includes('Edg')) return 'Chrome';
    if (ua.includes('Firefox')) return 'Firefox';
    if (ua.includes('Safari') && !ua.includes('Chrome')) return 'Safari';
    if (ua.includes('Edg')) return 'Edge';
    if (ua.includes('Opera') || ua.includes('OPR')) return 'Opera';
    return 'Unknown';
  }

  /**
   * Check if platform authenticator (biometrics) is available
   */
  async isPlatformAuthenticatorAvailable(): Promise<boolean> {
    if (!this.isWebAuthnSupported()) return false;
    
    try {
      // Check if we can create a discoverable credential
      const available = await navigator.credentials?.create({
        publicKey: {
          challenge: new Uint8Array(32),
          rp: { id: window.location.hostname, name: 'Test' },
          user: { id: new Uint8Array(16), name: 'test', displayName: 'Test' },
          pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
        },
      } as any);
      return !!available;
    } catch {
      return false;
    }
  }

  /**
   * Get supported attestation formats
   */
  getSupportedAttestation(): string[] {
    const formats: string[] = ['direct', 'none'];
    
    // Check for indirect attestation support
    if (typeof navigator !== 'undefined' && navigator.userAgent) {
      const ua = navigator.userAgent;
      // Most modern browsers support indirect
      if (ua.includes('Chrome/') || ua.includes('Firefox/')) {
        formats.push('indirect');
      }
    }
    
    return formats;
  }
}

// Export singleton
export const browserAdapter = new BrowserAdapter();
