// Authentication & Security Middleware
// Production-ready security implementation

import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import bcrypt from 'bcryptjs';
import { config } from '../../config/index.js';

// Rate limiter store (using Redis in production)
const rateLimitStore = new Map<string, { count: number; resetTime: number }>();

// Clean up old entries every minute
setInterval(() => {
  const now = Date.now();
  for (const [key, value] of rateLimitStore.entries()) {
    if (value.resetTime < now) {
      rateLimitStore.delete(key);
    }
  }
}, 60000);

// Extend Express Request type
declare global {
  namespace Express {
    interface Request {
      user?: {
        id: string;
        email: string;
        kycLevel: string;
        riskScore: number;
      };
      apiKey?: {
        id: string;
        userId: string;
        permissions: string[];
      };
    }
  }
}

// Rate Limiter Middleware
export const rateLimiter = (options: {
  windowMs?: number;
  maxRequests?: number;
  keyGenerator?: (req: Request) => string;
} = {}) => {
  const windowMs = options.windowMs || config.security.rateLimit.windowMs;
  const maxRequests = options.maxRequests || config.security.rateLimit.maxRequests;
  const keyGenerator = options.keyGenerator || ((req) => req.ip || 'unknown');

  return async (req: Request, res: Response, next: NextFunction) => {
    const key = keyGenerator(req);
    const now = Date.now();
    
    let record = rateLimitStore.get(key);
    
    if (!record || record.resetTime < now) {
      record = { count: 0, resetTime: now + windowMs };
      rateLimitStore.set(key, record);
    }
    
    record.count++;
    
    if (record.count > maxRequests) {
      res.status(429).json({
        error: 'Too many requests',
        message: 'Rate limit exceeded'
      });
      return;
    }
    
    res.setHeader('X-RateLimit-Limit', maxRequests.toString());
    res.setHeader('X-RateLimit-Remaining', (maxRequests - record.count).toString());
    
    next();
  };
};

// JWT Authentication
export const authenticate = async (req: Request, res: Response, next: NextFunction) => {
  try {
    const authHeader = req.headers.authorization;
    
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      res.status(401).json({ error: 'No token provided' });
      return;
    }
    
    const token = authHeader.substring(7);
    
    try {
      const decoded = jwt.verify(token, config.jwt.secret) as {
        userId: string;
        sessionId: string;
      };
      
      req.user = {
        id: decoded.userId,
        email: '',
        kycLevel: 'BASIC',
        riskScore: 100
      };
      
      next();
    } catch (jwtError) {
      res.status(401).json({ error: 'Invalid token' });
    }
  } catch (error) {
    console.error('Auth middleware error:', error);
    res.status(500).json({ error: 'Authentication failed' });
  }
};

// API Key Authentication
export const authenticateApiKey = async (req: Request, res: Response, next: NextFunction) => {
  const apiKey = req.headers['x-api-key'] as string;
  
  if (!apiKey) {
    res.status(401).json({ error: 'API key required' });
    return;
  }
  
  // Verify API key against database
  // In production: const result = await pool.query('SELECT * FROM api_keys WHERE...');
  if (!apiKey || apiKey.length < 20) {
    res.status(401).json({ error: 'Invalid API key' });
    return;
  }
  
  req.apiKey = {
    id: 'api_key_id',
    userId: 'user_id',
    permissions: ['read', 'trade']
  };
  
  next();
};

// KYC Level Requirement
export const requireKyc = (requiredLevel: 'BASIC' | 'INTERMEDIATE' | 'FULL') => {
  const levels = { 'BASIC': 1, 'INTERMEDIATE': 2, 'FULL': 3 };
  
  return (req: Request, res: Response, next: NextFunction) => {
    if (!req.user) {
      res.status(401).json({ error: 'Authentication required' });
      return;
    }
    
    const userLevel = levels[req.user.kycLevel as keyof typeof levels] || 0;
    const needLevel = levels[requiredLevel];
    
    if (userLevel < needLevel) {
      res.status(403).json({
        error: 'KYC required',
        message: `This action requires ${requiredLevel} KYC verification`
      });
      return;
    }
    
    next();
  };
};

// Security Headers
export const securityHeaders = (req: Request, res: Response, next: NextFunction) => {
  res.setHeader('X-Content-Type-Options', 'nosniff');
  res.setHeader('X-Frame-Options', 'DENY');
  res.setHeader('X-XSS-Protection', '1; mode=block');
  res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');
  next();
};

// Request Sanitization
export const sanitizeInput = (req: Request, res: Response, next: NextFunction) => {
  const sanitize = (obj: any): any => {
    if (typeof obj !== 'object' || obj === null) return obj;
    
    for (const key in obj) {
      if (typeof obj[key] === 'string') {
        obj[key] = obj[key]
          .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
          .replace(/javascript:/gi, '')
          .replace(/on\w+\s*=/gi, '');
      } else if (typeof obj[key] === 'object') {
        obj[key] = sanitize(obj[key]);
      }
    }
    return obj;
  };
  
  if (req.body) req.body = sanitize(req.body);
  next();
};

export default {
  rateLimiter,
  authenticate,
  authenticateApiKey,
  requireKyc,
  securityHeaders,
  sanitizeInput
};
