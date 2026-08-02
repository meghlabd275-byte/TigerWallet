/**
 * Chrome Extension - Swap Service
 * Token swap functionality for browser extension
 */

const SWAP_API_BASE = 'https://api.tigerwallet.com/v1/swap';

class SwapService {
  constructor(apiKey) {
    this.apiKey = apiKey;
  }

  async getTokens(chain = 'ethereum') {
    try {
      const response = await fetch(`${SWAP_API_BASE}/tokens?chain=${chain}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get tokens:', error);
      return [];
    }
  }

  async getQuote(params) {
    try {
      const { fromToken, toToken, amount, slippage = 0.5 } = params;
      const response = await fetch(`${SWAP_API_BASE}/quote`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          fromToken,
          toToken,
          amount,
          slippage
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get quote:', error);
      return null;
    }
  }

  async executeSwap(params) {
    try {
      const { walletId, fromToken, toToken, amount, minReceived, route } = params;
      const response = await fetch(`${SWAP_API_BASE}/execute`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          walletId,
          fromToken,
          toToken,
          amount,
          minReceived,
          route
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to execute swap:', error);
      throw error;
    }
  }
}

// NFT Service for Chrome Extension
class NFTExtensionService {
  constructor(apiKey) {
    this.apiKey = apiKey;
  }

  async getCollections(chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/collections?chain=${chain}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get collections:', error);
      return [];
    }
  }

  async getNFTs(ownerAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/owners/${ownerAddress}?chain=${chain}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get NFTs:', error);
      return [];
    }
  }

  async getListings(collectionAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/listings?collection=${collectionAddress}&chain=${chain}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get listings:', error);
      return [];
    }
  }

  async buyNFT(buyerAddress, listingId, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/purchase`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          buyer_address: buyerAddress,
          listing_id: listingId,
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to buy NFT:', error);
      throw error;
    }
  }

  async createListing(walletAddress, collectionAddress, tokenId, price, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/listings`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          wallet_address: walletAddress,
          collection_address: collectionAddress,
          token_id: tokenId,
          price,
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to create listing:', error);
      throw error;
    }
  }
}

// Staking Service for Chrome Extension
class StakingService {
  constructor(apiKey) {
    this.apiKey = apiKey;
  }

  async getValidators(chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/validators?chain=${chain}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get validators:', error);
      return [];
    }
  }

  async delegate(walletAddress, validatorAddress, amount, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/delegate`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          wallet_address: walletAddress,
          validator_address: validatorAddress,
          amount,
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to delegate:', error);
      throw error;
    }
  }

  async undelegate(walletAddress, amount, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/undelegate`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          wallet_address: walletAddress,
          amount,
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to undelegate:', error);
      throw error;
    }
  }

  async getRewards(walletAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/rewards/${walletAddress}?chain=${chain}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get rewards:', error);
      return null;
    }
  }
}

// Bridge Service for Chrome Extension
class BridgeService {
  constructor(apiKey) {
    this.apiKey = apiKey;
  }

  async getBridgeQuotes(params) {
    try {
      const { fromChain, toChain, fromToken, toToken, amount } = params;
      const response = await fetch(`https://api.tigerwallet.com/v1/bridge/quotes`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          from_chain: fromChain,
          to_chain: toChain,
          from_token: fromToken,
          to_token: toToken,
          amount
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get bridge quotes:', error);
      return [];
    }
  }

  async executeBridge(params) {
    try {
      const { walletId, fromChain, toChain, fromToken, toToken, amount, bridgeRoute } = params;
      const response = await fetch(`https://api.tigerwallet.com/v1/bridge/execute`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          wallet_id: walletId,
          from_chain: fromChain,
          to_chain: toChain,
          from_token: fromToken,
          to_token: toToken,
          amount,
          bridge_route: bridgeRoute
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to execute bridge:', error);
      throw error;
    }
  }

  async getSupportedChains() {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/bridge/chains', {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get chains:', error);
      return [];
    }
  }
}

// Export for use in extension
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { SwapService, NFTExtensionService, StakingService, BridgeService };
}
