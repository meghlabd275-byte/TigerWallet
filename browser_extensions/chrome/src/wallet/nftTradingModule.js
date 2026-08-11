/**
 * TigerWallet Browser Extension - NFT Trading Module
 * Complete NFT marketplace functionality for extension
 */

class NFTTradingModule {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.supportedChains = ['ethereum', 'polygon', 'arbitrum', 'solana', 'bsc'];
  }

  /**
   * Get NFTs for wallet
   */
  async getNFTs(walletAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`http://localhost:8443/api/v1/nfts?address=${walletAddress}&chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get NFTs:', error);
      return [];
    }
  }

  /**
   * Get NFT collection
   */
  async getCollection(collectionAddress, chain = 'ethereum') {
    try {
      const response = await fetch(`http://localhost:8443/api/v1/nfts/collection/${collectionAddress}?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get collection:', error);
      return null;
    }
  }

  /**
   * Get NFT marketplace listings
   */
  async getListings(collectionAddress, chain = 'ethereum', page = 1) {
    try {
      const response = await fetch(`http://localhost:8443/api/v1/nfts/marketplace?collection=${collectionAddress}&chain=${chain}&page=${page}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get listings:', error);
      return [];
    }
  }

  /**
   * Buy NFT
   */
  async buyNFT(walletAddress, collectionAddress, tokenId, price, chain = 'ethereum') {
    try {
      const response = await fetch('http://localhost:8443/api/v1/nfts/buy', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          walletAddress,
          collectionAddress,
          tokenId,
          price: price.toString(),
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Buy failed:', error);
      throw error;
    }
  }

  /**
   * Sell NFT (list on marketplace)
   */
  async sellNFT(walletAddress, collectionAddress, tokenId, price, chain = 'ethereum') {
    try {
      const response = await fetch('http://localhost:8443/api/v1/nfts/sell', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          walletAddress,
          collectionAddress,
          tokenId,
          price: price.toString(),
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Sell failed:', error);
      throw error;
    }
  }

  /**
   * Transfer NFT
   */
  async transferNFT(walletAddress, toAddress, collectionAddress, tokenId, chain = 'ethereum') {
    try {
      const response = await fetch('http://localhost:8443/api/v1/nfts/transfer', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          walletAddress,
          toAddress,
          collectionAddress,
          tokenId,
          chain
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Transfer failed:', error);
      throw error;
    }
  }

  /**
   * Get popular collections
   */
  async getPopularCollections(chain = 'ethereum') {
    try {
      const response = await fetch(`http://localhost:8443/api/v1/nfts/collections/popular?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get collections:', error);
      return this.getDefaultCollections();
    }
  }

  getDefaultCollections() {
    return [
      { address: '0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d', name: 'Bored Ape Yacht Club', floorPrice: '45 ETH', image: '' },
      { address: '0x23581767a106ae21c074b2276d25e5c3e136a68b', name: 'Moonbird', floorPrice: '3.5 ETH', image: '' },
      { address: '0x49cf6f5d44e70224e2e23fdcdd2f053f3fa6f430', name: 'StepN', floorPrice: '2.8 ETH', image: '' },
      { address: '0x8a90cab2b38dba80c64b7734e58e1cdb38f7f9d3', name: 'Azuki', floorPrice: '12 ETH', image: '' },
    ];
  }

  /**
   * Search NFTs
   */
  async searchNFTs(query, chain = 'ethereum') {
    try {
      const response = await fetch(`http://localhost:8443/api/v1/nfts/search?q=${encodeURIComponent(query)}&chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Search failed:', error);
      return [];
    }
  }

  /**
   * Create NFT trading UI
   */
  createNFTUI() {
    const container = document.createElement('div');
    container.className = 'tigerwallet-nft-popup';
    container.innerHTML = `
      <div class="tw-nft-header">
        <h2>🖼️ NFT Gallery</h2>
        <button class="tw-close-btn">&times;</button>
      </div>
      <div class="tw-nft-tabs">
        <button class="tw-tab active" data-tab="gallery">My NFTs</button>
        <button class="tw-tab" data-tab="marketplace">Marketplace</button>
        <button class="tw-tab" data-tab="transfer">Transfer</button>
      </div>
      <div class="tw-nft-content">
        <div class="tw-nft-gallery">
          <div class="tw-loading">Loading NFTs...</div>
        </div>
        <div class="tw-nft-marketplace" style="display:none;">
          <div class="tw-collections-grid"></div>
        </div>
        <div class="tw-nft-transfer" style="display:none;">
          <div class="tw-form-group">
            <label>Recipient Address</label>
            <input type="text" class="tw-recipient" placeholder="0x..." />
          </div>
          <div class="tw-form-group">
            <label>Collection Address</label>
            <input type="text" class="tw-collection" placeholder="0x..." />
          </div>
          <div class="tw-form-group">
            <label>Token ID</label>
            <input type="text" class="tw-token-id" placeholder="1" />
          </div>
          <button class="tw-transfer-btn btn-primary">Transfer NFT</button>
        </div>
      </div>
    `;
    return container;
  }

  /**
   * Setup NFT event listeners
   */
  setupNFTListeners(container, walletAddress) {
    // Tab switching
    container.querySelectorAll('.tw-tab').forEach(tab => {
      tab.addEventListener('click', (e) => {
        container.querySelectorAll('.tw-tab').forEach(t => t.classList.remove('active'));
        e.target.classList.add('active');
        
        const tabName = e.target.dataset.tab;
        container.querySelector('.tw-nft-gallery').style.display = tabName === 'gallery' ? 'block' : 'none';
        container.querySelector('.tw-nft-marketplace').style.display = tabName === 'marketplace' ? 'block' : 'none';
        container.querySelector('.tw-nft-transfer').style.display = tabName === 'transfer' ? 'block' : 'none';
      });
    });

    // Load gallery
    this.loadGallery(container, walletAddress);

    // Load marketplace
    this.loadMarketplace(container);

    // Transfer
    const transferBtn = container.querySelector('.tw-transfer-btn');
    transferBtn.addEventListener('click', async () => {
      const recipient = container.querySelector('.tw-recipient').value;
      const collection = container.querySelector('.tw-collection').value;
      const tokenId = container.querySelector('.tw-token-id').value;

      if (!recipient || !collection || !tokenId) {
        alert('Please fill all fields');
        return;
      }

      try {
        await this.transferNFT(walletAddress, recipient, collection, tokenId);
        alert('NFT transferred successfully!');
      } catch (error) {
        alert('Transfer failed: ' + error.message);
      }
    });
  }

  async loadGallery(container, walletAddress) {
    const gallery = container.querySelector('.tw-nft-gallery');
    
    try {
      const nfts = await this.getNFTs(walletAddress);
      
      if (nfts.length === 0) {
        gallery.innerHTML = '<div class="tw-empty">No NFTs found</div>';
        return;
      }

      gallery.innerHTML = nfts.map(nft => `
        <div class="tw-nft-card">
          <img src="${nft.imageUrl || 'https://via.placeholder.com/150'}" alt="${nft.name}" />
          <div class="tw-nft-info">
            <div class="tw-nft-name">${nft.name}</div>
            <div class="tw-nft-collection">${nft.collection}</div>
          </div>
        </div>
      `).join('');
    } catch (error) {
      gallery.innerHTML = '<div class="tw-error">Failed to load NFTs</div>';
    }
  }

  async loadMarketplace(container) {
    const grid = container.querySelector('.tw-collections-grid');
    
    try {
      const collections = await this.getPopularCollections();
      
      grid.innerHTML = collections.map(col => `
        <div class="tw-collection-card">
          <div class="tw-collection-image">
            <img src="${col.image || 'https://via.placeholder.com/150'}" alt="${col.name}" />
          </div>
          <div class="tw-collection-info">
            <div class="tw-collection-name">${col.name}</div>
            <div class="tw-collection-floor">Floor: ${col.floorPrice}</div>
          </div>
        </div>
      `).join('');
    } catch (error) {
      grid.innerHTML = '<div class="tw-error">Failed to load marketplace</div>';
    }
  }
}

// Export for extension
if (typeof module !== 'undefined' && module.exports) {
  module.exports = NFTTradingModule;
}
