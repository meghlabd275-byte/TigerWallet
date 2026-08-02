/**
 * React Web - Biometric Authentication Service
 * WebAuthn / WebOTP support
 */

// ============================================================================
// Biometric Authentication
// ============================================================================

export const biometricAuth = {
  /**
   * Check if WebAuthn is supported
   */
  isSupported: (): boolean => {
    return !!(
      window.PublicKeyCredential &&
      navigator.credentials &&
      navigator.credentials.create &&
      navigator.credentials.get
    );
  },

  /**
   * Register biometric credential
   */
  register: async (options: {
    challenge: string;
    userId: string;
    userName?: string;
    displayName?: string;
    rpName?: string;
    rpId?: string;
    timeout?: number;
    existingCredentials?: any[];
  }) => {
    if (!biometricAuth.isSupported()) {
      throw new Error('Biometric authentication not supported');
    }

    const publicKeyCredential = {
      challenge: base64ToArrayBuffer(options.challenge),
      rp: {
        name: options.rpName || 'TigerWallet',
        id: options.rpId || window.location.hostname
      },
      user: {
        id: base64ToArrayBuffer(options.userId),
        name: options.userName || 'TigerWallet User',
        displayName: options.displayName || 'TigerWallet User'
      },
      pubKeyCredParams: [
        { type: 'public-key', alg: -7 },
        { type: 'public-key', alg: -257 }
      ],
      timeout: options.timeout || 60000,
      excludeCredentials: options.existingCredentials || []
    };

    const credential = await navigator.credentials.create({
      publicKey: publicKeyCredential
    });

    return credentialToJSON(credential);
  },

  /**
   * Authenticate with biometric
   */
  authenticate: async (options: {
    challenge: string;
    rpId?: string;
    timeout?: number;
    allowCredentials?: any[];
  }) => {
    if (!biometricAuth.isSupported()) {
      throw new Error('Biometric authentication not supported');
    }

    const publicKeyCredential = {
      challenge: base64ToArrayBuffer(options.challenge),
      rpId: options.rpId || window.location.hostname,
      timeout: options.timeout || 60000,
      allowCredentials: options.allowCredentials || []
    };

    const credential = await navigator.credentials.get({
      publicKey: publicKeyCredential
    });

    return credentialToJSON(credential);
  },

  /**
   * Check if biometric is enrolled
   */
  isEnrolled: async (): Promise<boolean> => {
    try {
      const token = localStorage.getItem('auth_token');
      const response = await fetch('https://api.tigerwallet.com/v1/auth/biometric/status', {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      const data = await response.json();
      return data.enrolled || false;
    } catch {
      return false;
    }
  },

  /**
   * Enable biometric authentication
   */
  enable: async (): Promise<boolean> => {
    try {
      const token = localStorage.getItem('auth_token');
      const response = await fetch('https://api.tigerwallet.com/v1/auth/biometric/enable', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      return response.ok;
    } catch {
      return false;
    }
  },

  /**
   * Disable biometric authentication
   */
  disable: async (): Promise<boolean> => {
    try {
      const token = localStorage.getItem('auth_token');
      const response = await fetch('https://api.tigerwallet.com/v1/auth/biometric/disable', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      return response.ok;
    } catch {
      return false;
    }
  }
};

// ============================================================================
// WebOTP Service (SMS OTP)
// ============================================================================

export const webOTP = {
  /**
   * Request OTP via SMS
   */
  requestOTP: async (phoneNumber: string) => {
    const response = await fetch('https://api.tigerwallet.com/v1/auth/otp/request', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        phone: phoneNumber,
        channel: 'sms'
      })
    });
    return response.json();
  },

  /**
   * Verify OTP
   */
  verifyOTP: async (phoneNumber: string, code: string) => {
    const response = await fetch('https://api.tigerwallet.com/v1/auth/otp/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        phone: phoneNumber,
        code: code
      })
    });
    
    if (response.ok) {
      const data = await response.json();
      localStorage.setItem('auth_token', data.token);
      return data;
    }
    return null;
  },

  /**
   * Listen for WebOTP (if supported)
   */
  listen: async (): Promise<string | null> => {
    if ('OTPCredential' in window) {
      try {
        const content = await navigator.credentials.get({
          otp: { transport: ['sms'] }
        }) as any;
        return content.code;
      } catch {
        return null;
      }
    }
    return null;
  }
};

// ============================================================================
// Helper Functions
// ============================================================================

function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

function credentialToJSON(credential: any) {
  return {
    id: credential.id,
    type: credential.type,
    rawId: arrayBufferToBase64(credential.rawId),
    response: {
      attestationObject: arrayBufferToBase64(credential.response.attestationObject),
      clientDataJSON: arrayBufferToBase64(credential.response.clientDataJSON)
    }
  };
}

export default { biometricAuth, webOTP };
