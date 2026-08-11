// NFT Service - Browser Extension
// NFT gallery, trading, and minting

const API_BASE = 'http://localhost:8443/api/v1';

class NFTService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getCollections() {
    try {
      const response = await fetch(`${API_BASE}/nft/collections`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get collections:', error);
    }
    return [];
  }

  async getNFTs(collection, owner) {
    try {
      const response = await fetch(`${API_BASE}/nft/${collection}/nfts?owner=${owner}`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get NFTs:', error);
    }
    return [];
  }

  async getUserNFTs() {
    try {
      const response = await fetch(`${API_BASE}/nft/user/nfts`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get user NFTs:', error);
    }
    return [];
  }

  async buyNFT(collection, tokenId, price) {
    try {
      const response = await fetch(`${API_BASE}/nft/buy`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ collection, tokenId, price })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to buy NFT:', error);
    }
    return null;
  }

  async listNFT(collection, tokenId, price) {
    try {
      const response = await fetch(`${API_BASE}/nft/list`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ collection, tokenId, price })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to list NFT:', error);
    }
    return false;
  }

  async transferNFT(collection, tokenId, to) {
    try {
      const response = await fetch(`${API_BASE}/nft/transfer`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ collection, tokenId, to })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to transfer NFT:', error);
    }
    return false;
  }

  async mintNFT(collection, metadata) {
    try {
      const response = await fetch(`${API_BASE}/nft/mint`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ collection, ...metadata })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to mint NFT:', error);
    }
    return null;
  }
}

class DAOExtensionService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getDAOs() {
    try {
      const response = await fetch(`${API_BASE}/dao/list`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get DAOs:', error);
    }
    return [];
  }

  async getProposals(daoId) {
    try {
      const response = await fetch(`${API_BASE}/dao/${daoId}/proposals`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get proposals:', error);
    }
    return [];
  }

  async vote(proposalId, choice, weight) {
    try {
      const response = await fetch(`${API_BASE}/dao/proposals/${proposalId}/vote`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ choice, weight })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to vote:', error);
    }
    return false;
  }
}

class LaunchpadExtensionService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getActiveLaunches() {
    try {
      const response = await fetch(`${API_BASE}/launchpad/active`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get launches:', error);
    }
    return [];
  }

  async participate(launchId, amount) {
    try {
      const response = await fetch(`${API_BASE}/launchpad/${launchId}/participate`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ amount })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to participate:', error);
    }
    return null;
  }
}

class PredictionExtensionService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getMarkets() {
    try {
      const response = await fetch(`${API_BASE}/prediction/markets`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get markets:', error);
    }
    return [];
  }

  async placeBet(marketId, outcome, amount) {
    try {
      const response = await fetch(`${API_BASE}/prediction/${marketId}/bet`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ outcome, amount })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to place bet:', error);
    }
    return null;
  }
}

class RWAExtensionService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getRWAs() {
    try {
      const response = await fetch(`${API_BASE}/rwa/list`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get RWAs:', error);
    }
    return [];
  }

  async buyRWA(rwaId, amount) {
    try {
      const response = await fetch(`${API_BASE}/rwa/${rwaId}/buy`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ amount })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to buy RWA:', error);
    }
    return null;
  }
}

class OrderbookExtensionService {
  constructor(token) {
    this.token = token;
  }

  async getOrderbook(symbol) {
    try {
      const response = await fetch(`${API_BASE}/orderbook/${symbol}`);
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to get orderbook:', error);
    }
    return null;
  }

  async placeLimitOrder(symbol, side, price, quantity) {
    try {
      const response = await fetch(`${API_BASE}/orderbook/limit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ symbol, side, price, quantity })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to place order:', error);
    }
    return false;
  }
}

class SecurityScannerExtensionService {
  async scanContract(address, chain) {
    try {
      const response = await fetch(`${API_BASE}/security/scan?address=${address}&chain=${chain}`);
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to scan:', error);
    }
    return null;
  }
}

module.exports = {
  NFTService,
  DAOExtensionService,
  LaunchpadExtensionService,
  PredictionExtensionService,
  RWAExtensionService,
  OrderbookExtensionService,
  SecurityScannerExtensionService
};
