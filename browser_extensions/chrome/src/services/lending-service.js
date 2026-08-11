// Lending Service - Browser Extension
// Real backend connection to Go service

const API_BASE = 'http://localhost:8443/api/v1';

class LendingService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? 'Bearer ${this.token}' : ''
    };
  }

  async getPools() {
    try {
      const response = await fetch(API_BASE + '/lending/pools', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get pools:', error);
    }
    return [];
  }

  async getUserPositions() {
    try {
      const response = await fetch(API_BASE + '/lending/positions', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get positions:', error);
    }
    return [];
  }

  async supply(token, amount) {
    try {
      const response = await fetch(API_BASE + '/lending/supply', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ token, amount })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to supply:', error);
    }
    return null;
  }

  async borrow(token, amount) {
    try {
      const response = await fetch(API_BASE + '/lending/borrow', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ token, amount })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to borrow:', error);
    }
    return null;
  }

  async repay(token, amount) {
    try {
      const response = await fetch(API_BASE + '/lending/repay', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ token, amount })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to repay:', error);
    }
    return false;
  }

  async withdraw(token, amount) {
    try {
      const response = await fetch(API_BASE + '/lending/withdraw', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ token, amount })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to withdraw:', error);
    }
    return false;
  }

  async getHealthFactor() {
    try {
      const response = await fetch(API_BASE + '/lending/health', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || 999;
      }
    } catch (error) {
      console.error('Failed to get health factor:', error);
    }
    return 999;
  }
}

class GiftCardService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? 'Bearer ${this.token}' : ''
    };
  }

  async getTemplates() {
    try {
      const response = await fetch(API_BASE + '/giftcards/templates', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get templates:', error);
    }
    return [];
  }

  async createGiftCard(token, amount, templateId) {
    try {
      const response = await fetch(API_BASE + '/giftcards', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ token, amount, templateId })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to create gift card:', error);
    }
    return null;
  }

  async redeemGiftCard(code) {
    try {
      const response = await fetch(API_BASE + '/giftcards/redeem', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ code })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to redeem gift card:', error);
    }
    return null;
  }

  async checkBalance(code) {
    try {
      const response = await fetch(API_BASE + '/giftcards/' + code + '/balance', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to check balance:', error);
    }
    return null;
  }
}

class HardwareWalletService {
  constructor(token) {
    this.token = token;
  }

  static SUPPORTED_DEVICES = [
    'LEDGER_NANO_X', 'LEDGER_NANO_S', 'TREZOR_MODEL_T', 
    'TREZOR_ONE', 'KEYSTONE', 'COLDCAED'
  ];

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? 'Bearer ${this.token}' : ''
    };
  }

  async registerDevice(deviceType, serialNumber, firmwareVersion) {
    try {
      const response = await fetch(API_BASE + '/hardware/register', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ deviceType, serialNumber, firmwareVersion })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to register device:', error);
    }
    return null;
  }

  async signTransaction(walletId, txHash) {
    try {
      const response = await fetch(API_BASE + '/hardware/sign', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ walletId, txHash })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data.signature;
      }
    } catch (error) {
      console.error('Failed to sign transaction:', error);
    }
    return null;
  }

  async getWallets() {
    try {
      const response = await fetch(API_BASE + '/hardware/wallets', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get wallets:', error);
    }
    return [];
  }
}

class MPCWalletService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? 'Bearer ${this.token}' : ''
    };
  }

  async createShare(deviceId, publicKey) {
    try {
      const response = await fetch(API_BASE + '/mpc/shares', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ deviceId, publicKey })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to create share:', error);
    }
    return null;
  }

  async signTransaction(txHash) {
    try {
      const response = await fetch(API_BASE + '/mpc/sign', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ txHash })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data.signature;
      }
    } catch (error) {
      console.error('Failed to sign:', error);
    }
    return null;
  }

  async getAddress() {
    try {
      const response = await fetch(API_BASE + '/mpc/address', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data.address;
      }
    } catch (error) {
      console.error('Failed to get address:', error);
    }
    return null;
  }
}

class SocialRecoveryService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? 'Bearer ${this.token}' : ''
    };
  }

  async setupRecovery(guardians) {
    try {
      const response = await fetch(API_BASE + '/recovery/setup', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ guardians })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to setup recovery:', error);
    }
    return false;
  }

  async getGuardians() {
    try {
      const response = await fetch(API_BASE + '/recovery/guardians', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get guardians:', error);
    }
    return [];
  }

  async addGuardian(guardian) {
    try {
      const response = await fetch(API_BASE + '/recovery/guardians', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify(guardian)
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to add guardian:', error);
    }
    return false;
  }
}

class AccountAbstractionService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? 'Bearer ${this.token}' : ''
    };
  }

  async createAccount(ownerAddress, salt) {
    try {
      const response = await fetch(API_BASE + '/account/create', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ ownerAddress, salt })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to create account:', error);
    }
    return null;
  }

  async getAccounts() {
    try {
      const response = await fetch(API_BASE + '/account/list', {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get accounts:', error);
    }
    return [];
  }

  async addSigner(accountAddress, signerAddress, weight) {
    try {
      const response = await fetch(API_BASE + '/account/signers', {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ accountAddress, signerAddress, weight })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to add signer:', error);
    }
    return false;
  }
}
