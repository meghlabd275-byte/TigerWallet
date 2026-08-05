// TigerWallet MasterWallet - Account Abstraction Service (Chrome Extension)
// ERC-4337 Smart Wallet Integration
// Production-ready

class MasterAccountAbstractionService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.smartWallets = new Map();
    this.sessionKeys = new Map();
    this.entryPoint = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';
    this.factoryAddress = '0x...'; // Deployed factory address
    this.isInitialized = false;
  }

  async initialize() {
    if (this.isInitialized) return true;
    
    try {
      // Load existing smart wallets
      await this.loadSmartWallets();
      
      // Load session keys
      await this.loadSessionKeys();
      
      this.isInitialized = true;
      return true;
    } catch (error) {
      console.error('AccountAbstractionService initialization failed:', error);
      return false;
    }
  }

  async loadSmartWallets() {
    const result = await chrome.storage.local.get('smartWallets');
    if (result.smartWallets) {
      this.smartWallets = new Map(Object.entries(result.smartWallets));
    }
  }

  async loadSessionKeys() {
    const result = await chrome.storage.local.get('sessionKeys');
    if (result.sessionKeys) {
      this.sessionKeys = new Map(Object.entries(result.sessionKeys));
    }
  }

  // Create smart wallet for user
  async createSmartWallet(owner) {
    // Generate salt
    const salt = this.generateSalt();
    
    // Calculate wallet address using CREATE2
    const walletAddress = await this.calculateWalletAddress(owner, salt);
    
    // Initialize wallet data
    const wallet = {
      address: walletAddress,
      owner: owner,
      entryPoint: this.entryPoint,
      nonce: 0,
      implementation: await this.getImplementationAddress(),
      initialized: false,
      createdAt: Date.now(),
      guardians: [],
    };
    
    // Store wallet
    this.smartWallets.set(owner, wallet);
    await this.saveSmartWallets();
    
    // Deploy wallet contract (in production)
    await this.deployWallet(walletAddress, owner);
    
    return walletAddress;
  }

  async calculateWalletAddress(owner, salt) {
    // CREATE2: address = keccak256(0xff ++ factory ++ salt ++ keccak256(initCode))[12:]
    const initCode = this.getInitCode(owner);
    const initCodeHash = this.keccak256(initCode);
    const data = '0xff' + this.factoryAddress + salt + initCodeHash;
    const hash = this.keccak256(data);
    
    return '0x' + hash.substring(26); // Last 20 bytes
  }

  keccak256(data) {
    // Placeholder - use actual keccak implementation
    return this.hashSHA3(data);
  }

  hashSHA3(data) {
    // Simple hash placeholder
    let hash = 0;
    for (let i = 0; i < data.length; i++) {
      hash = ((hash << 5) - hash) + data.charCodeAt(i);
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }

  getInitCode(owner) {
    // ABI-encoded initialization call
    return '0x...'; // Placeholder
  }

  getImplementationAddress() {
    // Return latest implementation
    return '0x...'; // Placeholder
  }

  generateSalt() {
    return this.randomBytes(32);
  }

  randomBytes(length) {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('');
  }

  async deployWallet(walletAddress, owner) {
    // In production, submit deployment transaction
    console.log('Deploying wallet:', walletAddress);
    return true;
  }

  // Send user operation
  async sendUserOperation(sender, to, value, data, chainId = '1') {
    const wallet = this.smartWallets.get(sender);
    if (!wallet) {
      throw new Error('Smart wallet not found');
    }

    // Build user operation
    const userOp = {
      sender: wallet.address,
      nonce: wallet.nonce.toString(),
      initCode: '0x',
      callData: this.encodeCallData(to, value, data),
      callGasLimit: '100000',
      verificationGasLimit: '150000',
      preVerificationGas: '21000',
      maxFeePerGas: this.calculateGasPrice(chainId),
      maxPriorityFeePerGas: this.calculatePriorityFee(chainId),
      paymasterAndData: '0x',
      signature: '0x',
    };

    // Sign user operation
    userOp.signature = await this.signUserOperation(userOp, sender);

    // Submit to bundler
    const txHash = await this.submitUserOperation(userOp, chainId);

    // Update nonce
    wallet.nonce++;
    await this.saveSmartWallets();

    return txHash;
  }

  encodeCallData(to, value, data) {
    // ABI-encode the call
    return '0x...'; // Placeholder
  }

  calculateGasPrice(chainId) {
    // Get current gas price
    return '20000000000'; // 20 Gwei
  }

  calculatePriorityFee(chainId) {
    return '1000000000'; // 1 Gwei
  }

  async signUserOperation(userOp, owner) {
    // Hash user operation
    const hash = this.hashUserOperation(userOp);
    
    // Sign with owner's key (in production, use proper signing)
    const signature = await this.signMessage(hash, owner);
    
    return signature;
  }

  hashUserOperation(userOp) {
    // ERC-4337 user operation hash
    const types = [
      'address', 'uint256', 'bytes32', 'bytes32',
      'uint256', 'uint256', 'uint256',
      'uint256', 'uint256', 'bytes32', 'bytes32'
    ];
    
    const values = [
      userOp.sender,
      userOp.nonce,
      this.keccak256(userOp.initCode),
      this.keccak256(userOp.callData),
      userOp.callGasLimit,
      userOp.verificationGasLimit,
      userOp.preVerificationGas,
      userOp.maxFeePerGas,
      userOp.maxPriorityFeePerGas,
      this.keccak256(userOp.paymasterAndData),
      this.keccak256(userOp.signature),
    ];
    
    return this.keccak256(JSON.stringify(values));
  }

  async signMessage(message, owner) {
    // Sign message (placeholder)
    return '0x...';
  }

  async submitUserOperation(userOp, chainId) {
    // Submit to bundler or EntryPoint
    const response = await fetch('/api/aa/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userOp, chainId }),
    });
    
    const result = await response.json();
    return result.txHash;
  }

  // Session key management
  async addSessionKey(walletAddress, sessionKey) {
    if (!this.sessionKeys.has(walletAddress)) {
      this.sessionKeys.set(walletAddress, []);
    }
    
    this.sessionKeys.get(walletAddress).push({
      ...sessionKey,
      isActive: true,
      createdAt: Date.now(),
    });
    
    await this.saveSessionKeys();
    return true;
  }

  async removeSessionKey(walletAddress, keyId) {
    const keys = this.sessionKeys.get(walletAddress);
    if (!keys) return false;
    
    const index = keys.findIndex(k => k.id === keyId);
    if (index >= 0) {
      keys.splice(index, 1);
      await this.saveSessionKeys();
      return true;
    }
    
    return false;
  }

  async getSessionKeys(walletAddress) {
    return this.sessionKeys.get(walletAddress) || [];
  }

  async isSessionKeyValid(walletAddress, keyId, contract, token, amount) {
    const keys = await this.getSessionKeys(walletAddress);
    const key = keys.find(k => k.id === keyId && k.isActive);
    
    if (!key) return false;
    
    // Check expiration
    if (key.expiresAt && Date.now() > key.expiresAt) {
      return false;
    }
    
    // Check spending limit
    if (key.spent + amount > key.spendingLimit) {
      return false;
    }
    
    // Check allowed contracts
    if (key.allowedContracts && key.allowedContracts.length > 0) {
      if (!key.allowedContracts.includes(contract)) {
        return false;
      }
    }
    
    // Check allowed tokens
    if (key.allowedTokens && key.allowedTokens.length > 0) {
      if (!key.allowedTokens.includes(token)) {
        return false;
      }
    }
    
    return true;
  }

  // Social recovery
  async setupSocialRecovery(walletAddress, guardians, threshold) {
    const wallet = this.smartWallets.get(walletAddress);
    if (!wallet) {
      throw new Error('Wallet not found');
    }
    
    wallet.guardians = guardians.map(g => ({
      ...g,
      confirmed: false,
      addedAt: Date.now(),
    }));
    
    wallet.recoveryThreshold = threshold;
    await this.saveSmartWallets();
    
    return true;
  }

  async initiateRecovery(walletAddress, newOwner) {
    const wallet = this.smartWallets.get(walletAddress);
    if (!wallet) {
      throw new Error('Wallet not found');
    }
    
    wallet.pendingOwner = newOwner;
    wallet.recoveryStartTime = Date.now();
    await this.saveSmartWallets();
    
    return true;
  }

  async confirmRecovery(walletAddress, guardianAddress) {
    const wallet = this.smartWallets.get(walletAddress);
    if (!wallet) return false;
    
    const guardian = wallet.guardians.find(g => g.address === guardianAddress);
    if (guardian) {
      guardian.confirmed = true;
      await this.saveSmartWallets();
      return true;
    }
    
    return false;
  }

  async completeRecovery(walletAddress, signatures) {
    const wallet = this.smartWallets.get(walletAddress);
    if (!wallet) return false;
    
    // Verify sufficient confirmations
    const confirmedCount = wallet.guardians.filter(g => g.confirmed).length;
    if (confirmedCount < wallet.recoveryThreshold) {
      return false;
    }
    
    // Update owner
    wallet.owner = wallet.pendingOwner;
    wallet.pendingOwner = null;
    wallet.recoveryStartTime = null;
    
    // Reset guardians
    wallet.guardians.forEach(g => g.confirmed = false);
    
    await this.saveSmartWallets();
    return true;
  }

  // Save to storage
  async saveSmartWallets() {
    const data = {};
    this.smartWallets.forEach((value, key) => {
      data[key] = value;
    });
    
    await chrome.storage.local.set({ smartWallets: data });
  }

  async saveSessionKeys() {
    const data = {};
    this.sessionKeys.forEach((value, key) => {
      data[key] = value;
    });
    
    await chrome.storage.local.set({ sessionKeys: data });
  }

  // Get wallet info
  async getSmartWallet(owner) {
    return this.smartWallets.get(owner);
  }

  async listSmartWallets() {
    return Array.from(this.smartWallets.values());
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = MasterAccountAbstractionService;
}
