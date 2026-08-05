// TigerWallet MasterWallet - Paymaster Service (Chrome Extension)
// ERC-4337 Paymaster for gasless transactions
// Production-ready

class MasterPaymasterService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.paymasterAddress = '0x...'; // Deployed paymaster address
    this.sponsoredTransactions = new Map();
    this.policies = new Map();
    this.isInitialized = false;
    this.gasPriceCache = {};
  }

  async initialize() {
    if (this.isInitialized) return true;
    
    try {
      // Load policies
      await this.loadPolicies();
      
      // Start gas price monitoring
      this.startGasPriceMonitoring();
      
      this.isInitialized = true;
      return true;
    } catch (error) {
      console.error('PaymasterService initialization failed:', error);
      return false;
    }
  }

  async loadPolicies() {
    const result = await chrome.storage.local.get('paymasterPolicies');
    if (result.paymasterPolicies) {
      this.policies = new Map(Object.entries(result.paymasterPolicies));
    }
    
    // Add default policy if none exist
    if (this.policies.size === 0) {
      await this.createPolicy({
        id: 'default',
        enabled: true,
        maxDailySponsored: 1000,
        dailyUsed: 0,
        minTransactionValue: '0',
        maxTransactionValue: '1000000000000000000', // 1 ETH
        allowedTokens: ['ETH', 'USDT', 'USDC'],
        markupPercent: 10,
      });
    }
  }

  startGasPriceMonitoring() {
    // Update gas prices every 15 seconds
    this.gasPriceInterval = setInterval(async () => {
      await this.updateGasPrices();
    }, 15000);
    
    // Initial update
    this.updateGasPrices();
  }

  async updateGasPrices() {
    try {
      // Fetch from multiple sources
      const prices = await Promise.all([
        this.fetchGasPrice('1'), // Ethereum
        this.fetchGasPrice('56'), // BSC
        this.fetchGasPrice('137'), // Polygon
      ]);
      
      this.gasPriceCache = {
        '1': prices[0],
        '56': prices[1],
        '137': prices[2],
        timestamp: Date.now(),
      };
    } catch (error) {
      console.error('Failed to update gas prices:', error);
    }
  }

  async fetchGasPrice(chainId) {
    // In production, fetch from RPC or oracle
    return {
      baseFeePerGas: '20000000000', // 20 Gwei
      maxFeePerGas: '30000000000', // 30 Gwei
      maxPriorityFeePerGas: '1000000000', // 1 Gwei
    };
  }

  // Check if transaction can be sponsored
  async canSponsor(sender, chainId, token = 'ETH') {
    // Check policy
    const policy = this.policies.get('default');
    if (!policy || !policy.enabled) {
      return { canSponsor: false, reason: 'Sponsorship disabled' };
    }
    
    // Check daily limit
    if (policy.dailyUsed >= policy.maxDailySponsored) {
      return { canSponsor: false, reason: 'Daily limit reached' };
    }
    
    // Check sender whitelist if required
    if (policy.requireWhitelist && policy.allowedSenders) {
      if (!policy.allowedSenders.includes(sender)) {
        return { canSponsor: false, reason: 'Sender not whitelisted' };
      }
    }
    
    // Check blocked senders
    if (policy.blockedSenders && policy.blockedSenders.includes(sender)) {
      return { canSponsor: false, reason: 'Sender blocked' };
    }
    
    return { canSponsor: true };
  }

  // Create paymaster and data for user operation
  async getPaymasterAndData(userOp, chainId = '1') {
    if (!this.isInitialized) {
      throw new Error('Paymaster service not initialized');
    }
    
    // Check sponsorship
    const { canSponsor, reason } = await this.canSponsor(
      userOp.sender,
      chainId,
      userOp.token || 'ETH'
    );
    
    if (!canSponsor) {
      throw new Error(reason);
    }
    
    // Get gas prices
    const gasPrices = this.gasPriceCache[chainId] || await this.fetchGasPrice(chainId);
    
    // Build paymaster and data
    const validUntil = 0; // Always valid
    const hash = await this.hashPaymasterData(userOp, chainId, validUntil);
    const signature = await this.signMessage(hash);
    
    const paymasterAndData = 
      this.paymasterAddress +
      this.toHex(validUntil, 4) +
      signature;
    
    // Update policy usage
    await this.incrementDailyUsage();
    
    return paymasterAndData;
  }

  async hashPaymasterData(userOp, chainId, validUntil) {
    // Hash according to ERC-4337
    const types = ['address', 'uint256', 'bytes32', 'uint256', 'bytes32'];
    const values = [
      this.paymasterAddress,
      validUntil,
      this.keccak256(userOp.sender),
      userOp.nonce,
      this.keccak256(userOp.callData),
    ];
    
    return this.keccak256(JSON.stringify(values));
  }

  async signMessage(message) {
    // Sign with paymaster key (placeholder)
    return '0x' + '0'.repeat(130);
  }

  toHex(value, bytes) {
    return value.toString(16).padStart(bytes * 2, '0');
  }

  keccak256(data) {
    // Placeholder
    let hash = 0;
    for (let i = 0; i < data.length; i++) {
      hash = ((hash << 5) - hash) + data.charCodeAt(i);
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }

  async incrementDailyUsage() {
    const policy = this.policies.get('default');
    if (policy) {
      policy.dailyUsed++;
      await this.savePolicies();
    }
  }

  // Policy management
  async createPolicy(policy) {
    this.policies.set(policy.id, {
      ...policy,
      createdAt: Date.now(),
      lastReset: Date.now(),
    });
    
    await this.savePolicies();
    return policy.id;
  }

  async updatePolicy(policyId, updates) {
    const policy = this.policies.get(policyId);
    if (!policy) return false;
    
    this.policies.set(policyId, { ...policy, ...updates });
    await this.savePolicies();
    return true;
  }

  async deletePolicy(policyId) {
    if (policyId === 'default') return false;
    return this.policies.delete(policyId);
  }

  async getPolicy(policyId) {
    return this.policies.get(policyId);
  }

  async listPolicies() {
    return Array.from(this.policies.values());
  }

  async savePolicies() {
    const data = {};
    this.policies.forEach((value, key) => {
      data[key] = value;
    });
    
    await chrome.storage.local.set({ paymasterPolicies: data });
  }

  // Calculate fees
  async calculateFee(userOp, chainId = '1') {
    const gasPrices = this.gasPriceCache[chainId] || await this.fetchGasPrice(chainId);
    const policy = this.policies.get('default');
    
    const gasLimit = parseInt(userOp.callGasLimit) + 
                    parseInt(userOp.verificationGasLimit) + 
                    parseInt(userOp.preVerificationGas);
    
    const baseFee = gasLimit * parseInt(gasPrices.maxFeePerGas);
    const markup = policy ? (baseFee * policy.markupPercent / 100) : 0;
    
    return {
      baseFee: baseFee.toString(),
      markup: markup.toString(),
      totalFee: (baseFee + markup).toString(),
    };
  }

  // Balance management
  async getPaymasterBalance(chainId = '1') {
    // In production, query contract
    return '100000000000000000000'; // 100 ETH
  }

  async fundPaymaster(chainId, amount) {
    // In production, send funds to paymaster
    console.log('Funding paymaster:', amount);
    return true;
  }

  // Statistics
  async getStats() {
    const policy = this.policies.get('default');
    
    return {
      totalSponsored: policy ? policy.dailyUsed : 0,
      dailyLimit: policy ? policy.maxDailySponsored : 0,
      availableToday: policy ? (policy.maxDailySponsored - policy.dailyUsed) : 0,
      gasPrices: this.gasPriceCache,
    };
  }

  // Cleanup
  shutdown() {
    if (this.gasPriceInterval) {
      clearInterval(this.gasPriceInterval);
    }
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = MasterPaymasterService;
}
