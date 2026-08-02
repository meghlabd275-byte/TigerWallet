/**
 * Trading Terminal - Biometric Authentication Service
 * WebAuthn / WebOTP support
 */

// ============================================================================
// Biometric Authentication
// ============================================================================

export const biometricAuth = {
  isSupported: () => {
    return !!(
      window.PublicKeyCredential &&
      navigator.credentials &&
      navigator.credentials.create &&
      navigator.credentials.get
    );
  },

  register: async (options) => {
    if (!biometricAuth.isSupported()) {
      throw new Error('Biometric authentication not supported');
    }

    const publicKeyCredential = {
      challenge: base64ToArrayBuffer(options.challenge),
      rp: {
        name: options.rpName || 'TigerWallet Trading',
        id: options.rpId || window.location.hostname
      },
      user: {
        id: base64ToArrayBuffer(options.userId),
        name: options.userName || 'Trader',
        displayName: options.displayName || 'TigerWallet Trader'
      },
      pubKeyCredParams: [
        { type: 'public-key', alg: -7 },
        { type: 'public-key', alg: -257 }
      ],
      timeout: options.timeout || 60000
    };

    const credential = await navigator.credentials.create({
      publicKey: publicKeyCredential
    });

    return credentialToJSON(credential);
  },

  authenticate: async (options) => {
    if (!biometricAuth.isSupported()) {
      throw new Error('Biometric authentication not supported');
    }

    const publicKeyCredential = {
      challenge: base64ToArrayBuffer(options.challenge),
      rpId: options.rpId || window.location.hostname,
      timeout: options.timeout || 60000
    };

    const credential = await navigator.credentials.get({
      publicKey: publicKeyCredential
    });

    return credentialToJSON(credential);
  },

  isEnrolled: async () => {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/auth/biometric/status', {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
      });
      const data = await response.json();
      return data.enrolled || false;
    } catch {
      return false;
    }
  },

  enable: async () => {
    const response = await fetch('https://api.tigerwallet.com/v1/auth/biometric/enable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    return response.ok;
  },

  disable: async () => {
    const response = await fetch('https://api.tigerwallet.com/v1/auth/biometric/disable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    return response.ok;
  }
};

// WebOTP
export const webOTP = {
  requestOTP: async (phoneNumber) => {
    const response = await fetch('https://api.tigerwallet.com/v1/auth/otp/request', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone: phoneNumber, channel: 'sms' })
    });
    return response.json();
  },

  verifyOTP: async (phoneNumber, code) => {
    const response = await fetch('https://api.tigerwallet.com/v1/auth/otp/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone: phoneNumber, code })
    });
    
    if (response.ok) {
      const data = await response.json();
      localStorage.setItem('auth_token', data.token);
      return data;
    }
    return null;
  },

  listen: async () => {
    if ('OTPCredential' in window) {
      try {
        const content = await navigator.credentials.get({ otp: { transport: ['sms'] } });
        return content.code;
      } catch {
        return null;
      }
    }
    return null;
  }
};

// Helpers
function base64ToArrayBuffer(base64) {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
}

function arrayBufferToBase64(buffer) {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

function credentialToJSON(credential) {
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
