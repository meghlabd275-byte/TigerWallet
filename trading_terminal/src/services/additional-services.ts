/**
 * Trading Terminal - Additional Features
 * Token Swap, NFT, Staking, Bridge functionality
 */

import React, { useState, useEffect } from 'react';

// ============================================================================
// Swap Service
// ============================================================================

const SwapService = {
  async getTokens(chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/swap/tokens?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get tokens:', error);
      return [];
    }
  },

  async getQuote(fromToken, toToken, amount, slippage = 0.5) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/swap/quote', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fromToken, toToken, amount, slippage })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get quote:', error);
      return null;
    }
  },

  async executeSwap(walletId, fromToken, toToken, amount, minReceived, route) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/swap/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ walletId, fromToken, toToken, amount, minReceived, route })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to execute swap:', error);
      throw error;
    }
  }
};

// ============================================================================
// NFT Service
// ============================================================================

const NFTSwapService = {
  async getCollections(chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/collections?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get collections:', error);
      return [];
    }
  },

  async getUserNFTs(ownerAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/nft/owners/${ownerAddress}?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get NFTs:', error);
      return [];
    }
  },

  async buyNFT(buyerAddress, listingId, chain = 'ethereum') {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/nft/purchase', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ buyer_address: buyerAddress, listing_id: listingId, chain })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to buy NFT:', error);
      throw error;
    }
  },

  async createListing(walletAddress, collectionAddress, tokenId, price, chain = 'ethereum') {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/nft/listings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_address: walletAddress, collection_address: collectionAddress, token_id: tokenId, price, chain })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to create listing:', error);
      throw error;
    }
  }
};

// ============================================================================
// Staking Service
// ============================================================================

const StakingSwapService = {
  async getValidators(chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/validators?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get validators:', error);
      return [];
    }
  },

  async delegate(walletAddress, validatorAddress, amount, chain = 'ethereum') {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/staking/delegate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_address: walletAddress, validator_address: validatorAddress, amount, chain })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to delegate:', error);
      throw error;
    }
  },

  async undelegate(walletAddress, amount, chain = 'ethereum') {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/staking/undelegate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_address: walletAddress, amount, chain })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to undelegate:', error);
      throw error;
    }
  },

  async getRewards(walletAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/rewards/${walletAddress}?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get rewards:', error);
      return null;
    }
  }
};

// ============================================================================
// Bridge Service
// ============================================================================

const BridgeSwapService = {
  async getQuotes(fromChain, toChain, fromToken, toToken, amount) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/bridge/quotes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ from_chain: fromChain, to_chain: toChain, from_token: fromToken, to_token: toToken, amount })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get bridge quotes:', error);
      return [];
    }
  },

  async executeBridge(walletId, fromChain, toChain, fromToken, toToken, amount, bridgeRoute) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/bridge/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_id: walletId, from_chain: fromChain, to_chain: toChain, from_token: fromToken, to_token: toToken, amount, bridge_route: bridgeRoute })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to execute bridge:', error);
      throw error;
    }
  },

  async getSupportedChains() {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/bridge/chains');
      return await response.json();
    } catch (error) {
      console.error('Failed to get chains:', error);
      return [];
    }
  }
};

// ============================================================================
// Hardware Wallet Service
// ============================================================================

const HardwareWalletService = {
  async connect(deviceType) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/hardware/connect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ device_type: deviceType })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to connect hardware wallet:', error);
      return null;
    }
  },

  async getAddress(derivationPath) {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/hardware/address?path=${derivationPath}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get address:', error);
      return null;
    }
  },

  async signTransaction(derivationPath, transaction) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/hardware/sign', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ derivation_path: derivationPath, transaction })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to sign transaction:', error);
      throw error;
    }
  }
};

// Export
export { SwapService, NFTSwapService, StakingSwapService, BridgeSwapService, HardwareWalletService };
