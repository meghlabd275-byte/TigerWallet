/**
 * Chrome Extension - Swap Service
 * Token swap functionality for browser extension.
 *
 * Wired to the REAL on-chain AMM router on the canonical wallet_api backend
 * (http://localhost:8443) — same backend path as the web + desktop clients:
 *   GET  /api/v1/amm/quote   -> Uniswap-V2 getAmountsOut eth_call
 *   POST /api/v1/amm/swap     -> builds swapExactTokensForTokens calldata
 *   POST /api/v1/send         -> signs + broadcasts via eth_sendRawTransaction
 * Honest results only: never fabricates a quote, rate, or tx hash.
 */

const WALLET_API_BASE = 'http://localhost:8443';

class SwapService {
  constructor(apiKey) {
    this.apiKey = apiKey;
  }

  async getTokens(chain = 'ethereum') {
    // The backend exposes token holdings per wallet; there is no global token
    // list endpoint, so return an honest empty list rather than fabricating
    // token metadata. The quote/swap endpoints accept any 0x token address.
    return [];
  }

  async getQuote(params) {
    try {
      const { fromToken, toToken, amount, chainId = 1 } = params;
      const qs = new URLSearchParams({
        chain_id: String(chainId),
        token_in: fromToken,
        token_out: toToken,
        amount_in: String(amount)
      });
      const response = await fetch(`${WALLET_API_BASE}/api/v1/amm/quote?${qs}`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
      });
      if (!response.ok) {
        console.error('Failed to get AMM quote:', response.status);
        return null;
      }
      const data = await response.json();
      return {
        fromToken: data.token_in,
        toToken: data.token_out,
        fromAmount: data.amount_in,
        toAmount: data.amount_out,
        toAmountWei: data.amount_out_wei,
        path: data.path,
        router: data.router,
        chainId: data.chain_id,
        rawReturn: data.raw_return
      };
    } catch (error) {
      console.error('Failed to get quote:', error);
      return null;
    }
  }

  async executeSwap(params) {
    try {
      const { walletId, fromToken, toToken, amount, chainId = 1 } = params;
      // Step 1: build the swap calldata via the AMM router.
      const swapResp = await fetch(`${WALLET_API_BASE}/api/v1/amm/swap`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          from: walletId,
          chain_id: chainId,
          token_in: fromToken,
          token_out: toToken,
          amount_in: String(amount)
        })
      });
      if (!swapResp.ok) {
        console.error('AMM swap calldata failed:', swapResp.status);
        return { txHash: '' };
      }
      const swapData = await swapResp.json();
      const txTo = (swapData.tx && swapData.tx.to) || swapData.to;
      const txData = (swapData.tx && swapData.tx.data) || swapData.data;
      if (!txTo || !txData) {
        return { txHash: '' };
      }
      // Step 2: broadcast the assembled tx via /api/v1/send (real
      // eth_sendRawTransaction). Returns the REAL tx hash, or '' on failure.
      const sendResp = await fetch(`${WALLET_API_BASE}/api/v1/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          walletId,
          to: txTo,
          data: txData,
          value: '0',
          type: 'swap'
        })
      });
      const sendData = await sendResp.json();
      return { txHash: sendData.tx_hash || sendData.txHash || '' };
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
      const response = await fetch(`http://localhost:8443/api/v1/nft/collections?chain=${chain}`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/nft/owners/${ownerAddress}?chain=${chain}`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/nft/listings?collection=${collectionAddress}&chain=${chain}`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/nft/purchase`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/nft/listings`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/staking/validators?chain=${chain}`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/staking/delegate`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/staking/undelegate`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/staking/rewards/${walletAddress}?chain=${chain}`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/bridge/quotes`, {
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
      const response = await fetch(`http://localhost:8443/api/v1/bridge/execute`, {
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
      const response = await fetch('http://localhost:8443/api/v1/bridge/chains', {
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
