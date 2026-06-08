/**
 * TigerWallet - Security Layer
 * 
 * Industrial-grade security with:
 * - AES-256-GCM encryption
 * - CSRF protection
 * - XSS prevention
 * - Rate limiting
 * - Input validation
 * - Secure storage
 */

import { webcrypto } from 'crypto';

// ============================================================================
// Constants
// ============================================================================

const ENCRYPTION_ALGORITHM = 'AES-GCM';
const KEY_LENGTH = 256;
const IV_LENGTH = 12;
const TAG_LENGTH = 128;

// ============================================================================
// Crypto Utilities
// ============================================================================

/**
 * Generate a random encryption key
 */
export const generateKey = async (): Promise<CryptoKey> => {
  return webcrypto.subtle.generateKey(
    { name: ENCRYPTION_ALGORITHM, length: KEY_LENGTH },
    true,
    ['encrypt', 'decrypt']
  );
};

/**
 * Export key to base64
 */
export const exportKey = async (key: CryptoKey): Promise<string> => {
  const exported = await webcrypto.subtle.exportKey('raw', key);
  return btoa(String.fromCharCode(...new Uint8Array(exported)));
};

/**
 * Import key from base64
 */
export const importKey = async (keyData: string): Promise<CryptoKey> => {
  const keyBytes = Uint8Array.from(atob(keyData), c => c.charCodeAt(0));
  return webcrypto.subtle.importKey(
    'raw',
    keyBytes,
    { name: ENCRYPTION_ALGORITHM, length: KEY_LENGTH },
    true,
    ['encrypt', 'decrypt']
  );
};

/**
 * Encrypt data with AES-256-GCM
 */
export const encrypt = async (
  data: string,
  key: CryptoKey
): Promise<string> => {
  const encoder = new TextEncoder();
  const dataBytes = encoder.encode(data);
  
  // Generate random IV
  const iv = webcrypto.getRandomValues(new Uint8Array(IV_LENGTH));
  
  // Encrypt
  const encrypted = await webcrypto.subtle.encrypt(
    { name: ENCRYPTION_ALGORITHM, iv, tagLength: TAG_LENGTH },
    key,
    dataBytes
  );
  
  // Combine IV + encrypted data
  const combined = new Uint8Array(iv.length + encrypted.byteLength);
  combined.set(iv, 0);
  combined.set(new Uint8Array(encrypted), iv.length);
  
  return btoa(String.fromCharCode(...combined));
};

/**
 * Decrypt data with AES-256-GCM
 */
export const decrypt = async (
  encryptedData: string,
  key: CryptoKey
): Promise<string> => {
  const combined = Uint8Array.from(atob(encryptedData), c => c.charCodeAt(0));
  
  // Extract IV and encrypted data
  const iv = combined.slice(0, IV_LENGTH);
  const data = combined.slice(IV_LENGTH);
  
  // Decrypt
  const decrypted = await webcrypto.subtle.decrypt(
    { name: ENCRYPTION_ALGORITHM, iv, tagLength: TAG_LENGTH },
    key,
    data
  );
  
  const decoder = new TextDecoder();
  return decoder.decode(decrypted);
};

/**
 * Hash data with SHA-256
 */
export const hash = async (data: string): Promise<string> => {
  const encoder = new TextEncoder();
  const dataBytes = encoder.encode(data);
  const hashBytes = await webcrypto.subtle.digest('SHA-256', dataBytes);
  return btoa(String.fromCharCode(...new Uint8Array(hashBytes)));
};

/**
 * Derive key from password using PBKDF2
 */
export const deriveKeyFromPassword = async (
  password: string,
  salt: string
): Promise<CryptoKey> => {
  const encoder = new TextEncoder();
  const passwordBytes = encoder.encode(password);
  const saltBytes = encoder.encode(salt);
  
  const key = await webcrypto.subtle.importKey(
    'raw',
    passwordBytes,
    { name: 'PBKDF2' },
    false,
    ['deriveKey']
  );
  
  return webcrypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: saltBytes,
      iterations: 100000,
      hash: 'SHA-256',
    },
    key,
    { name: ENCRYPTION_ALGORITHM, length: KEY_LENGTH },
    true,
    ['encrypt', 'decrypt']
  );
};

// ============================================================================
// Input Validation & Sanitization
// ============================================================================

/**
 * Sanitize HTML to prevent XSS
 */
export const sanitizeHTML = (input: string): string => {
  const div = document.createElement('div');
  div.textContent = input;
  return div.innerHTML;
};

/**
 * Validate and sanitize user input
 */
export const sanitizeInput = (input: string, type: 'text' | 'email' | 'address' | 'amount'): string => {
  let sanitized = input.trim();
  
  // Remove null bytes and control characters
  sanitized = sanitized.replace(/[\x00-\x1F\x7F]/g, '');
  
  if (type === 'email') {
    // Basic email sanitization
    sanitized = sanitized.toLowerCase().replace(/[^a-z0-9@._-]/g, '');
  } else if (type === 'address') {
    // Address validation (EVM)
    if (!/^0x[a-fA-F0-9]{40}$/.test(sanitized)) {
      throw new Error('Invalid address format');
    }
  } else if (type === 'amount') {
    // Amount validation
    if (!/^\d+(\.\d+)?$/.test(sanitized)) {
      throw new Error('Invalid amount format');
    }
  }
  
  return sanitized;
};

/**
 * Validate EVM address
 */
export const isValidEVMAddress = (address: string): boolean => {
  return /^0x[a-fA-F0-9]{40}$/.test(address);
};

/**
 * Validate transaction amount
 */
export const isValidAmount = (amount: string): boolean => {
  const num = parseFloat(amount);
  return !isNaN(num) && num > 0 && /^\d+(\.\d+)?$/.test(amount);
};

/**
 * Validate seed phrase (24 words)
 */
export const isValidSeedPhrase = (phrase: string): boolean => {
  const words = phrase.trim().split(/\s+/);
  return words.length === 24 && words.every(w => /^[a-z]+$/.test(w));
};

// ============================================================================
// Rate Limiting
// ============================================================================

interface RateLimitConfig {
  maxAttempts: number;
  windowMs: number;
  lockoutMs: number;
}

const RATE_LIMIT_CONFIGS: { [key: string]: RateLimitConfig } = {
  login: { maxAttempts: 5, windowMs: 15 * 60 * 1000, lockoutMs: 15 * 60 * 1000 },
  transaction: { maxAttempts: 10, windowMs: 60 * 1000, lockoutMs: 5 * 60 * 1000 },
  api: { maxAttempts: 100, windowMs: 60 * 1000, lockoutMs: 60 * 1000 },
};

const rateLimitStore: { [key: string]: { attempts: number; firstAttempt: number; lockedUntil?: number } } = {};

/**
 * Check rate limit
 */
export const checkRateLimit = (key: string, type: string): { allowed: boolean; remainingAttempts: number; lockoutRemaining?: number } => {
  const config = RATE_LIMIT_CONFIGS[type] || { maxAttempts: 60, windowMs: 60 * 1000, lockoutMs: 30 * 1000 };
  
  const now = Date.now();
  let record = rateLimitStore[key];
  
  // Initialize or reset if window passed
  if (!record || now - record.firstAttempt > config.windowMs) {
    record = { attempts: 0, firstAttempt: now };
    rateLimitStore[key] = record;
  }
  
  // Check if locked out
  if (record.lockedUntil && now < record.lockedUntil) {
    return {
      allowed: false,
      remainingAttempts: 0,
      lockoutRemaining: record.lockedUntil - now,
    };
  }
  
  // Check attempts
  if (record.attempts >= config.maxAttempts) {
    record.lockedUntil = now + config.lockoutMs;
    return {
      allowed: false,
      remainingAttempts: 0,
      lockoutRemaining: config.lockoutMs,
    };
  }
  
  return {
    allowed: true,
    remainingAttempts: config.maxAttempts - record.attempts,
  };
};

/**
 * Record an attempt
 */
export const recordAttempt = (key: string, type: string, success: boolean): void => {
  const config = RATE_LIMIT_CONFIGS[type] || { maxAttempts: 60, windowMs: 60 * 1000 };
  
  const now = Date.now();
  let record = rateLimitStore[key];
  
  if (!record) {
    record = { attempts: 0, firstAttempt: now };
  }
  
  if (success) {
    // Reset on success
    record.attempts = 0;
    record.lockedUntil = undefined;
  } else {
    // Increment on failure
    record.attempts++;
    
    // Lock out if max attempts reached
    if (record.attempts >= config.maxAttempts) {
      record.lockedUntil = now + config.lockoutMs;
    }
  }
  
  rateLimitStore[key] = record;
};

// ============================================================================
// CSRF Protection
// ============================================================================

let csrfToken: string = '';

/**
 * Generate CSRF token
 */
export const generateCSRFToken = async (): Promise<string> => {
  const random = Math.random().toString(36).substring(2);
  const timestamp = Date.now().toString(36);
  csrfToken = await hash(random + timestamp);
  return csrfToken;
};

/**
 * Get CSRF token
 */
export const getCSRFToken = (): string => csrfToken;

/**
 * Verify CSRF token
 */
export const verifyCSRFToken = (token: string): boolean => {
  return token === csrfToken && csrfToken.length > 0;
};

// ============================================================================
// Secure Storage
// ============================================================================

const STORAGE_PREFIX = 'tigerwallet_';

/**
 * Securely store data (encrypted)
 */
export const secureStore = {
  async set(key: string, value: any, encryptionKey?: CryptoKey): Promise<void> {
    const serialized = JSON.stringify(value);
    
    if (encryptionKey) {
      const encrypted = await encrypt(serialized, encryptionKey);
      localStorage.setItem(STORAGE_PREFIX + key, 'encrypted:' + encrypted);
    } else {
      // Use sessionStorage for sensitive data
      sessionStorage.setItem(STORAGE_PREFIX + key, serialized);
    }
  },
  
  async get(key: string, encryptionKey?: CryptoKey): Promise<any> {
    const stored = sessionStorage.getItem(STORAGE_PREFIX + key) || localStorage.getItem(STORAGE_PREFIX + key);
    
    if (!stored) return null;
    
    if (stored.startsWith('encrypted:') && encryptionKey) {
      const encrypted = stored.substring(10);
      const decrypted = await decrypt(encrypted, encryptionKey);
      return JSON.parse(decrypted);
    }
    
    return JSON.parse(stored);
  },
  
  remove(key: string): void {
    localStorage.removeItem(STORAGE_PREFIX + key);
    sessionStorage.removeItem(STORAGE_PREFIX + key);
  },
  
  clear(): void {
    const keys = [...Object.keys(localStorage), ...Object.keys(sessionStorage)];
    keys.forEach(key => {
      if (key.startsWith(STORAGE_PREFIX)) {
        localStorage.removeItem(key);
        sessionStorage.removeItem(key);
      }
    });
  },
};

// ============================================================================
// Transaction Signing Security
// ============================================================================

/**
 * Secure transaction request with validation
 */
export interface SecureTransactionRequest {
  chainId: number | string;
  to: string;
  token: string;
  amount: string;
  data?: string;
  gasLimit?: number;
  nonce?: number;
}

/**
 * Validate transaction request
 */
export const validateTransactionRequest = (req: SecureTransactionRequest): { valid: boolean; errors: string[] } => {
  const errors: string[] = [];
  
  // Validate chain
  if (!req.chainId) {
    errors.push('Chain ID required');
  }
  
  // Validate recipient
  if (!isValidEVMAddress(req.to)) {
    errors.push('Invalid recipient address');
  }
  
  // Validate amount
  if (!isValidAmount(req.amount)) {
    errors.push('Invalid amount');
  }
  
  // Check for suspicious patterns
  if (req.data && req.data.length > 10000) {
    errors.push('Data too long');
  }
  
  return { valid: errors.length === 0, errors };
};

// ============================================================================
// Security Headers & CSP
// ============================================================================

/**
 * Get security headers
 */
export const getSecurityHeaders = (): Headers => {
  const headers = new Headers();
  
  headers.set('X-Content-Type-Options', 'nosniff');
  headers.set('X-Frame-Options', 'DENY');
  headers.set('X-XSS-Protection', '1; mode=block');
  headers.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
  headers.set('Content-Security-Policy', "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';");
  
  return headers;
};

// ============================================================================
// Export
// ============================================================================

export default {
  generateKey,
  exportKey,
  importKey,
  encrypt,
  decrypt,
  hash,
  deriveKeyFromPassword,
  sanitizeHTML,
  sanitizeInput,
  isValidEVMAddress,
  isValidAmount,
  isValidSeedPhrase,
  checkRateLimit,
  recordAttempt,
  generateCSRFToken,
  getCSRFToken,
  verifyCSRFToken,
  secureStore,
  validateTransactionRequest,
  getSecurityHeaders,
};