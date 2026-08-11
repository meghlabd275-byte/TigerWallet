/**
 * Chrome Extension - Biometric Authentication Service
 * WebAuthn / WebOTP support for extension
 */

// ============================================================================
// Biometric Authentication Service
// ============================================================================

class BiometricAuthService {
  constructor() {
    this.isSupported = this.checkSupport();
  }

  /**
   * Check if WebAuthn is supported
   */
  checkSupport() {
    return !!(
      window.PublicKeyCredential &&
      navigator.credentials &&
      navigator.credentials.create &&
      navigator.credentials.get
    );
  }

  /**
   * Register biometric credential
   */
  async register(options) {
    if (!this.isSupported) {
      throw new Error('Biometric authentication not supported');
    }

    try {
      const publicKeyCredential = {
        challenge: this.base64ToArrayBuffer(options.challenge),
        rp: {
          name: options.rpName || 'TigerWallet',
          id: options.rpId || window.location.hostname
        },
        user: {
          id: this.base64ToArrayBuffer(options.userId),
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

      return this.credentialToJSON(credential);
    } catch (error) {
      console.error('Biometric registration failed:', error);
      throw error;
    }
  }

  /**
   * Authenticate with biometric
   */
  async authenticate(options) {
    if (!this.isSupported) {
      throw new Error('Biometric authentication not supported');
    }

    try {
      const publicKeyCredential = {
        challenge: this.base64ToArrayBuffer(options.challenge),
        rpId: options.rpId || window.location.hostname,
        timeout: options.timeout || 60000,
        allowCredentials: options.allowCredentials || []
      };

      const credential = await navigator.credentials.get({
        publicKey: publicKeyCredential
      });

      return this.credentialToJSON(credential);
    } catch (error) {
      console.error('Biometric authentication failed:', error);
      throw error;
    }
  }

  /**
   * Check if biometric is enrolled
   */
  async isEnrolled() {
    try {
      const response = await fetch('http://localhost:8443/api/v1/auth/biometric/status', {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      const data = await response.json();
      return data.enrolled || false;
    } catch (error) {
      console.error('Failed to check biometric status:', error);
      return false;
    }
  }

  /**
   * Enable biometric authentication
   */
  async enable() {
    try {
      const response = await fetch('http://localhost:8443/api/v1/auth/biometric/enable', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to enable biometric:', error);
      return false;
    }
  }

  /**
   * Disable biometric authentication
   */
  async disable() {
    try {
      const response = await fetch('http://localhost:8443/api/v1/auth/biometric/disable', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
          'Content-Type': 'application/json'
        }
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to disable biometric:', error);
      return false;
    }
  }

  /**
   * Get stored auth token
   */
  getToken() {
    return localStorage.getItem('auth_token') || '';
  }

  /**
   * Convert base64 to ArrayBuffer
   */
  base64ToArrayBuffer(base64) {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes.buffer;
  }

  /**
   * Convert credential to JSON
   */
  credentialToJSON(credential) {
    return {
      id: credential.id,
      type: credential.type,
      rawId: this.arrayBufferToBase64(credential.rawId),
      response: {
        attestationObject: this.arrayBufferToBase64(credential.response.attestationObject),
        clientDataJSON: this.arrayBufferToBase64(credential.response.clientDataJSON)
      }
    };
  }

  /**
   * Convert ArrayBuffer to base64
   */
  arrayBufferToBase64(buffer) {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
  }
}

// ============================================================================
// WebOTP Service (SMS OTP)
// ============================================================================

class WebOTPService {
  /**
   * Request OTP via SMS
   */
  async requestOTP(phoneNumber) {
    try {
      const response = await fetch('http://localhost:8443/api/v1/auth/otp/request', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          phone: phoneNumber,
          channel: 'sms'
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to request OTP:', error);
      throw error;
    }
  }

  /**
   * Verify OTP
   */
  async verifyOTP(phoneNumber, code) {
    try {
      const response = await fetch('http://localhost:8443/api/v1/auth/otp/verify', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
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
    } catch (error) {
      console.error('Failed to verify OTP:', error);
      throw error;
    }
  }

  /**
   * Listen for WebOTP (if supported)
   */
  async listen() {
    if ('OTPCredential' in window) {
      try {
        const content = await navigator.credentials.get({
          otp: { transport: ['sms'] }
        });
        return content.code;
      } catch (error) {
        console.error('WebOTP failed:', error);
        return null;
      }
    }
    return null;
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { BiometricAuthService, WebOTPService };
}
