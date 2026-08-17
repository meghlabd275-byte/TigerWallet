/**
 * BiometricService - browser extension biometric authentication.
 *
 * Real biometric verification is performed by the platform (WebAuthn /
 * authenticator hardware); the client never fabricates a biometric match
 * score. The `compareBiometric` method intentionally throws (fail-closed) to
 * prevent any client-side score simulation. Enrollment/session bookkeeping is
 * real; only the cryptographic match is delegated to the platform + backend.
 *
 * NOTE: The previous file used Dart syntax (`Type?`, `??=`, nested `class`
 * declarations inside a class body). That has been removed; this is plain JS.
 */

'use strict';

class BiometricService {
  constructor() {
    // Storage
    this.enrollments = new Map();
    this.userEnrollments = new Map();
    this.sessions = new Map();
    this.behavioralData = new Map();
    this.failedAttempts = new Map();
    this.rateLimiter = new Map();

    // Config
    this.MAX_FAILED_ATTEMPTS = 5;
    this.LOCKOUT_DURATION = 5 * 60 * 1000;
    this.SESSION_DURATION = 30 * 60 * 1000;
  }

  // Singleton accessor (plain JS, no Dart syntax).
  static getInstance() {
    if (!BiometricService._instance) {
      BiometricService._instance = new BiometricService();
    }
    return BiometricService._instance;
  }

  // ==================== Types ====================
  // Plain factory functions (not nested classes, which are invalid JS).

  createEnrollment({ id, userId, type, template, deviceId, createdAt, lastUsedAt, confidence, isActive }) {
    return {
      id, userId, type, template, deviceId, createdAt, lastUsedAt, confidence,
      isActive: isActive !== false,
    };
  }

  createCapability({ type, available, quality, livenessDetection }) {
    return { type, available, quality, livenessDetection };
  }

  createSession({ sessionId, userId, type, startedAt, expiresAt, verified }) {
    return { sessionId, userId, type, startedAt, expiresAt, verified: !!verified };
  }

  // ==================== Capability Detection ====================

  async getCapabilities() {
    const webAuthnAvailable = typeof window !== 'undefined' && !!window.PublicKeyCredential;
    const camera = this.checkCameraAccess();
    return [
      this.createCapability({ type: 'fingerprint', available: webAuthnAvailable, quality: 'high', livenessDetection: true }),
      this.createCapability({ type: 'face', available: webAuthnAvailable && camera, quality: 'high', livenessDetection: true }),
      this.createCapability({ type: 'iris', available: webAuthnAvailable && camera, quality: 'high', livenessDetection: true }),
      this.createCapability({ type: 'behavioral', available: true, quality: 'medium', livenessDetection: false }),
    ];
  }

  checkCameraAccess() {
    return typeof navigator !== 'undefined' && !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
  }

  // ==================== Enrollment ====================

  async startEnrollment(userId, type) {
    if (!this.checkRateLimit(userId)) {
      return { success: false, error: 'Too many attempts. Please try again later.' };
    }
    const existing = this.getUserEnrollments(userId, type);
    if (existing.length >= 5) {
      return { success: false, error: 'Maximum enrollments reached for this biometric type' };
    }
    const sessionId = this.generateId();
    const challenge = this.generateRandom(32);
    const session = this.createSession({
      sessionId, userId, type,
      startedAt: Date.now(),
      expiresAt: Date.now() + this.SESSION_DURATION,
      verified: false,
    });
    this.sessions.set(sessionId, session);
    return { success: true, sessionId, challenge };
  }

  async completeEnrollment(sessionId, biometricData, deviceId) {
    const session = this.sessions.get(sessionId);
    if (!session) return { success: false, error: 'Session not found or expired' };
    if (session.expiresAt < Date.now()) {
      this.sessions.delete(sessionId);
      return { success: false, error: 'Session expired' };
    }
    const enrollment = this.createEnrollment({
      id: this.generateId(),
      userId: session.userId,
      type: session.type,
      template: this.encryptTemplate(biometricData),
      deviceId,
      createdAt: Date.now(),
      lastUsedAt: Date.now(),
      confidence: 100,
      isActive: true,
    });
    this.enrollments.set(enrollment.id, enrollment);
    let userSet = this.userEnrollments.get(session.userId);
    if (!userSet) { userSet = new Set(); this.userEnrollments.set(session.userId, userSet); }
    userSet.add(enrollment.id);
    session.verified = true;
    this.sessions.set(sessionId, session);
    return { success: true, enrollmentId: enrollment.id };
  }

  getUserEnrollments(userId, type) {
    const ids = this.userEnrollments.get(userId);
    if (!ids) return [];
    const result = [];
    for (const id of ids) {
      const e = this.enrollments.get(id);
      if (e && (!type || e.type === type)) {
        result.push({ ...e, template: '[ENCRYPTED]' });
      }
    }
    return result;
  }

  async deleteEnrollment(userId, enrollmentId) {
    const enrollment = this.enrollments.get(enrollmentId);
    if (!enrollment || enrollment.userId !== userId) {
      return { success: false, error: 'Enrollment not found' };
    }
    enrollment.isActive = false;
    this.enrollments.set(enrollmentId, enrollment);
    return { success: true };
  }

  // ==================== Verification ====================

  async verify(userId, type, biometricData) {
    if (!this.checkRateLimit(userId)) {
      return { success: false, error: 'Too many attempts. Account temporarily locked.' };
    }
    const failed = this.failedAttempts.get(userId);
    if (failed && failed.lockedUntil && failed.lockedUntil > Date.now()) {
      const remaining = Math.ceil((failed.lockedUntil - Date.now()) / 1000);
      return { success: false, error: `Account locked. Try again in ${remaining} seconds.`, requiresFallback: true };
    }
    const enrollments = this.getUserEnrollments(userId, type).filter((e) => e.isActive);
    if (enrollments.length === 0) {
      return { success: false, error: 'No enrolled biometrics found' };
    }
    // Biometric match is performed by the platform; client-side fabrication is
    // disabled. compareBiometric throws (see below).
    try {
      this.compareBiometric(biometricData, enrollments[0].template);
    } catch (e) {
      const current = this.failedAttempts.get(userId) || { count: 0 };
      current.count++;
      if (current.count >= this.MAX_FAILED_ATTEMPTS) {
        current.lockedUntil = Date.now() + this.LOCKOUT_DURATION;
      }
      this.failedAttempts.set(userId, current);
      return {
        success: false,
        error: 'Biometric verification requires platform/WebAuthn match; client-side score unavailable',
        requiresFallback: current.count >= this.MAX_FAILED_ATTEMPTS,
      };
    }
  }

  async quickVerify(userId) {
    const caps = await this.getCapabilities();
    const availableTypes = caps.filter((c) => c.available).map((c) => c.type);
    for (const type of availableTypes) {
      const result = await this.verify(userId, type, new ArrayBuffer(0));
      if (result.success) return result;
    }
    return { success: false, error: 'No biometrics enrolled' };
  }

  // ==================== Behavioral Biometrics ====================

  startBehavioralCollection(userId) {
    this.behavioralData.set(userId, {
      keystrokeDynamics: { keyPressTime: [], keyReleaseTime: [], interKeyDelay: [] },
      mouseMovements: { x: [], y: [], timestamps: [] },
      touchGestures: { pressure: [], size: [], angle: [] },
    });
  }

  recordKeystroke(userId, keyPressTime, keyReleaseTime) {
    const data = this.behavioralData.get(userId);
    if (!data) return;
    data.keystrokeDynamics.keyPressTime.push(keyPressTime);
    data.keystrokeDynamics.keyReleaseTime.push(keyReleaseTime);
    if (data.keystrokeDynamics.keyPressTime.length > 1) {
      const last = data.keystrokeDynamics.keyPressTime[data.keystrokeDynamics.keyPressTime.length - 2];
      data.keystrokeDynamics.interKeyDelay.push(keyPressTime - last);
    }
    if (data.keystrokeDynamics.keyPressTime.length > 100) {
      data.keystrokeDynamics.keyPressTime.shift();
      data.keystrokeDynamics.keyReleaseTime.shift();
      data.keystrokeDynamics.interKeyDelay.shift();
    }
  }

  recordMouseMovement(userId, x, y) {
    const data = this.behavioralData.get(userId);
    if (!data) return;
    data.mouseMovements.x.push(x);
    data.mouseMovements.y.push(y);
    data.mouseMovements.timestamps.push(Date.now());
    if (data.mouseMovements.x.length > 200) {
      data.mouseMovements.x.shift();
      data.mouseMovements.y.shift();
      data.mouseMovements.timestamps.shift();
    }
  }

  async verifyBehavioral(userId) {
    const data = this.behavioralData.get(userId);
    if (!data) return { success: false, error: 'No behavioral data collected' };
    const keystrokeScore = this.analyzeKeystrokeDynamics(data.keystrokeDynamics);
    const mouseScore = this.analyzeMouseMovements(data.mouseMovements);
    const confidence = Math.round((keystrokeScore + mouseScore) / 2);
    // Behavioral signals are an auxiliary factor only; they do NOT authorize
    // on their own. Return the confidence but never success without platform
    // verification.
    return { success: false, confidence, error: 'Behavioral score is advisory only; platform verification required' };
  }

  stopBehavioralCollection(userId) {
    this.behavioralData.delete(userId);
  }

  // ==================== Session Management ====================

  createSession(userId, enrollmentId) {
    const sessionId = this.generateId();
    const enrollment = this.enrollments.get(enrollmentId);
    const session = this.createSession({
      sessionId,
      userId,
      type: enrollment ? enrollment.type : 'behavioral',
      startedAt: Date.now(),
      expiresAt: Date.now() + this.SESSION_DURATION,
      verified: true,
    });
    this.sessions.set(sessionId, session);
    return sessionId;
  }

  validateSession(sessionId) {
    const session = this.sessions.get(sessionId);
    if (!session || session.expiresAt < Date.now()) {
      if (session) this.sessions.delete(sessionId);
      return { valid: false };
    }
    return { valid: true, userId: session.userId };
  }

  extendSession(sessionId) {
    const session = this.sessions.get(sessionId);
    if (!session || !session.verified) return false;
    session.expiresAt = Date.now() + this.SESSION_DURATION;
    this.sessions.set(sessionId, session);
    return true;
  }

  // ==================== Helpers ====================

  checkRateLimit(userId) {
    const now = Date.now();
    const attempts = this.rateLimiter.get(userId) || [];
    const recent = attempts.filter((t) => now - t < 60000);
    if (recent.length >= 10) return false;
    recent.push(now);
    this.rateLimiter.set(userId, recent);
    return true;
  }

  encryptTemplate(data) {
    // Encode the template for storage. This is transport encoding, not a
    // substitute for backend-side key-protected storage.
    return btoa(String.fromCharCode(...new Uint8Array(data)));
  }

  decryptTemplate(encrypted) {
    return Uint8Array.from(atob(encrypted), (c) => c.charCodeAt(0));
  }

  compareBiometric(data1, template2) {
    // Biometric verification is performed by the platform WebAuthn/hardware;
    // client-side score fabrication is disabled (fail-closed).
    throw new Error('Biometric verification is performed by the platform WebAuthn/hardware; client-side score fabrication is disabled');
  }

  analyzeKeystrokeDynamics(dynamics) {
    if (!dynamics || dynamics.interKeyDelay.length < 5) return 50;
    const delays = dynamics.interKeyDelay;
    const avg = delays.reduce((a, b) => a + b, 0) / delays.length;
    const variance = delays.reduce((sum, d) => sum + Math.pow(d - avg, 2), 0) / delays.length;
    const stdDev = Math.sqrt(variance);
    const consistency = Math.max(0, 100 - (stdDev / avg) * 100);
    return Math.round(consistency);
  }

  analyzeMouseMovements(movements) {
    if (!movements || movements.x.length < 10) return 50;
    const velocities = [];
    for (let i = 1; i < movements.x.length; i++) {
      const dx = movements.x[i] - movements.x[i - 1];
      const dy = movements.y[i] - movements.y[i - 1];
      const dt = movements.timestamps[i] - movements.timestamps[i - 1];
      if (dt > 0) velocities.push(Math.sqrt(dx * dx + dy * dy) / dt);
    }
    if (velocities.length === 0) return 50;
    const avg = velocities.reduce((a, b) => a + b, 0) / velocities.length;
    const variance = velocities.reduce((sum, v) => sum + Math.pow(v - avg, 2), 0) / velocities.length;
    const stdDev = Math.sqrt(variance);
    const humanity = Math.min(100, (stdDev / avg) * 50);
    return Math.round(humanity);
  }

  generateId() {
    return `id_${Date.now()}_${this.generateRandom(8)}`;
  }

  generateRandom(length) {
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    const bytes = new Uint8Array(length);
    crypto.getRandomValues(bytes);
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars[bytes[i] % chars.length];
    }
    return result;
  }

  generateBackupCodes(count = 8) {
    const codes = [];
    for (let i = 0; i < count; i++) {
      codes.push(this.generateRandom(4).toUpperCase());
    }
    return codes;
  }

  resetFailedAttempts(userId) {
    this.failedAttempts.delete(userId);
  }
}

BiometricService._instance = null;

if (typeof module !== 'undefined' && module.exports) {
  module.exports = BiometricService;
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_BIOMETRIC = { BiometricService };
}
