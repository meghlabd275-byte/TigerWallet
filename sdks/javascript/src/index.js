/**
 * TigerWallet JavaScript SDK
 * A client library for interacting with TigerWallet API
 */

const crypto = require('crypto');

class TigerWalletClient {
  /**
   * Create a new TigerWallet API client
   * @param {string} apiKey - API key
   * @param {string} apiSecret - API secret
   * @param {Object} options - Configuration options
   */
  constructor(apiKey, apiSecret, options = {}) {
    this.apiKey = apiKey;
    this.apiSecret = apiSecret;
    this.baseUrl = options.baseUrl || 'https://api.tigerwallet.com';
    this.tenantId = options.tenantId || null;
    this.timeout = options.timeout || 30000;
    this.rateLimit = options.rateLimit || 100;
  }

  /**
   * Generate HMAC-SHA256 signature
   */
  _generateSignature(method, path, timestamp, body = '') {
    const message = `${method}\n${path}\n${timestamp}\n${body}`;
    return crypto
      .createHmac('sha256', this.apiSecret)
      .update(message)
      .digest('hex');
  }

  /**
   * Make authenticated API request
   */
  async _request(method, path, data = null) {
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const body = data ? JSON.stringify(data) : '';
    
    const signature = this._generateSignature(method, path, timestamp, body);
    
    const headers = {
      'Content-Type': 'application/json',
      'X-API-Key': this.apiKey,
      'X-Timestamp': timestamp,
      'X-Signature': signature,
    };
    
    if (this.tenantId) {
      headers['X-Tenant-ID'] = this.tenantId;
    }

    const url = `${this.baseUrl}${path}`;
    
    const response = await fetch(url, {
      method,
      headers,
      body: body || undefined,
      signal: AbortSignal.timeout(this.timeout),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(`API Error: ${response.status} - ${error.message || response.statusText}`);
    }

    return response.json();
  }

  // ==================== Fetcher Service ====================

  /**
   * Get token prices
   * @param {string[]} symbols - List of token symbols
   */
  async getPrices(symbols) {
    const path = `/api/v1/fetcher/prices?symbols=${symbols.join(',')}`;
    return this._request('GET', path);
  }

  /**
   * Get wallet balance
   * @param {string} chain - Blockchain chain (eth, bsc, etc.)
   * @param {string} address - Wallet address
   */
  async getWalletBalance(chain, address) {
    const path = `/api/v1/fetcher/wallet/${chain}/${address}`;
    return this._request('GET', path);
  }

  /**
   * Get transactions
   * @param {string} chain - Blockchain chain
   * @param {string} address - Wallet address
   * @param {number} limit - Number of transactions
   */
  async getTransactions(chain, address, limit = 50) {
    const path = `/api/v1/fetcher/transactions/${chain}/${address}?limit=${limit}`;
    return this._request('GET', path);
  }

  /**
   * Get token information
   * @param {string} chain - Blockchain chain
   * @param {string} tokenAddress - Token contract address
   */
  async getTokenInfo(chain, tokenAddress) {
    const path = `/api/v1/fetcher/token/${chain}/${tokenAddress}`;
    return this._request('GET', path);
  }

  /**
   * Get market data
   * @param {string[]} symbols - List of token symbols
   */
  async getMarketData(symbols) {
    const path = `/api/v1/fetcher/market?symbols=${symbols.join(',')}`;
    return this._request('GET', path);
  }

  // ==================== Permission Service ====================

  /**
   * Get all permissions
   */
  async getPermissions() {
    return this._request('GET', '/api/v1/permissions');
  }

  /**
   * Check if a feature is enabled
   * @param {string} feature - Feature name
   */
  async checkPermission(feature) {
    const path = `/api/v1/permissions/${feature}`;
    const result = await this._request('GET', path);
    return result.enabled || false;
  }

  /**
   * Sync permissions from server
   */
  async syncPermissions() {
    return this._request('POST', '/api/v1/permissions/sync');
  }

  // ==================== License Service ====================

  /**
   * Validate license key
   * @param {string} licenseKey - License key
   * @param {string} hardwareId - Hardware ID
   */
  async validateLicense(licenseKey, hardwareId) {
    return this._request('POST', '/api/v1/licenses/validate', {
      license_key: licenseKey,
      hardware_id: hardwareId,
    });
  }

  /**
   * Get license information
   */
  async getLicenseInfo() {
    return this._request('GET', '/api/v1/licenses/info');
  }

  // ==================== Webhook Service ====================

  /**
   * Register a webhook
   * @param {string} eventType - Event type
   * @param {string} url - Webhook URL
   * @param {string} secret - Webhook secret
   */
  async registerWebhook(eventType, url, secret) {
    return this._request('POST', '/api/v1/webhooks', {
      event_type: eventType,
      url,
      secret,
    });
  }

  /**
   * Verify webhook signature
   * @param {string} payload - Webhook payload
   * @param {string} signature - Webhook signature
   * @param {string} secret - Webhook secret
   */
  verifyWebhook(payload, signature, secret) {
    const expected = crypto
      .createHmac('sha256', secret)
      .update(payload)
      .digest('hex');
    return crypto.timingSafeEqual(
      Buffer.from(signature),
      Buffer.from(expected)
    );
  }
}

// ==================== Service Classes ====================

class FetcherService {
  constructor(client) {
    this.client = client;
  }

  getPrices(symbols) {
    return this.client.getPrices(symbols);
  }

  getWalletBalance(chain, address) {
    return this.client.getWalletBalance(chain, address);
  }

  getTransactions(chain, address, limit) {
    return this.client.getTransactions(chain, address, limit);
  }

  getTokenInfo(chain, tokenAddress) {
    return this.client.getTokenInfo(chain, tokenAddress);
  }

  getMarketData(symbols) {
    return this.client.getMarketData(symbols);
  }
}

class PermissionService {
  constructor(client) {
    this.client = client;
  }

  getPermissions() {
    return this.client.getPermissions();
  }

  checkPermission(feature) {
    return this.client.checkPermission(feature);
  }

  syncPermissions() {
    return this.client.syncPermissions();
  }
}

class LicenseService {
  constructor(client) {
    this.client = client;
  }

  validate(licenseKey, hardwareId) {
    return this.client.validateLicense(licenseKey, hardwareId);
  }

  getInfo() {
    return this.client.getLicenseInfo();
  }
}

class WebhookService {
  constructor(client) {
    this.client = client;
  }

  register(eventType, url, secret) {
    return this.client.registerWebhook(eventType, url, secret);
  }

  verify(payload, signature, secret) {
    return this.client.verifyWebhook(payload, signature, secret);
  }
}

// Export for different module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    TigerWalletClient,
    FetcherService,
    PermissionService,
    LicenseService,
    WebhookService,
  };
} else if (typeof window !== 'undefined') {
  window.TigerWallet = {
    TigerWalletClient,
    FetcherService,
    PermissionService,
    LicenseService,
    WebhookService,
  };
}

export {
  TigerWalletClient,
  FetcherService,
  PermissionService,
  LicenseService,
  WebhookService,
};
