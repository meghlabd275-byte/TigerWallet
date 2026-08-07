'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { 
  Box, Typography, Card, CardContent, Button, TextField, 
  Chip, CircularProgress, Alert, IconButton, InputAdornment,
  Grid, Divider, Dialog, DialogTitle, DialogContent, Tabs, Tab
} from '@mui/material';
import {
  Search, FilterList, ShoppingCart, Visibility, Favorite,
  Refresh, Close, Verified, Collections, Hexagon
} from '@mui/icons-material';
import { useTheme } from '../components/ThemeProvider';

// ============================================================================
// Types
// ============================================================================

interface NFT {
  id: number;
  token_id: string;
  contract_address: string;
  owner_address: string;
  name: string;
  symbol: string;
  description: string;
  image_url: string;
  animation_url?: string;
  attributes: NFTAttribute[];
  uri: string;
  chain_id: number;
}

interface NFTAttribute {
  trait_type: string;
  value: string;
}

interface Collection {
  id: number;
  contract_address: string;
  name: string;
  symbol: string;
  description: string;
  image_url: string;
  creator: string;
  total_supply: number;
  is_verified: boolean;
}

interface NFTOffer {
  offer_id: string;
  token_id: string;
  contract_address: string;
  seller_address: string;
  price: string;
  price_token: string;
  expires_at: string;
}

interface BuyRequest {
  user_address: string;
  offer_id: string;
}

// ============================================================================
// API Configuration
// ============================================================================

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8086';

const CHAIN_NAMES: Record<number, string> = {
  1: 'Ethereum',
  56: 'BNB Chain',
  137: 'Polygon',
  42161: 'Arbitrum',
  43114: 'Avalanche',
};

// ============================================================================
// Utility Functions
// ============================================================================

function formatAddress(address: string, chars: number = 4): string {
  if (!address) return '';
  if (address.length <= chars * 2 + 2) return address;
  return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`;
}

// ============================================================================
// Main Component
// ============================================================================

export default function NFTMarketplace() {
  const { isDarkMode } = useTheme();
  const [activeTab, setActiveTab] = useState(0);
  const [nfts, setNfts] = useState<NFT[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState('all');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [selectedNFT, setSelectedNFT] = useState<NFT | null>(null);
  const [walletAddress, setWalletAddress] = useState<string>('');
  const [buyDialogOpen, setBuyDialogOpen] = useState(false);

  // Fetch NFTs from API
  const fetchNFTs = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await fetch(`${API_BASE}/api/v1/nft/collections`);
      if (!response.ok) {
        throw new Error(`NFT catalog request failed with HTTP ${response.status}`)
      }
      const data = await response.json();
      if (!Array.isArray(data.nfts)) {
        throw new Error('NFT catalog response did not contain a valid nfts array')
      }
      setNfts(data.nfts)
    } catch (err) {
      setError('NFT catalog is unavailable because the marketplace API could not be reached.')
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch collections
  const fetchCollections = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/nft/collections`);
      if (response.ok) {
        const data = await response.json();
        if (data.collections) {
          setCollections(data.collections);
        }
      }
    } catch (err) {
      console.log('No collections data');
    }
  }, []);

  useEffect(() => {
    fetchNFTs();
    fetchCollections();
  }, [fetchNFTs, fetchCollections]);

  // Simulated wallet address
  useEffect(() => {
    const savedWallet = localStorage.getItem('tigerwallet_address');
    if (savedWallet) {
      setWalletAddress(savedWallet);
    }
  }, []);

  const filteredNFTs = nfts.filter(nft => {
    const matchesSearch = nft.name.toLowerCase().includes(search.toLowerCase()) || 
                         nft.symbol.toLowerCase().includes(search.toLowerCase());
    if (filter === 'all') return matchesSearch;
    return matchesSearch && nft.chain_id === parseInt(filter);
  });

  // Handle buy
  const handleBuy = async (nft: NFT) => {
    if (!walletAddress) {
      setError('Please connect your wallet first');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      throw new Error(`NFT purchase is unavailable until a connected wallet, signed transaction provider, and marketplace execution endpoint are configured for ${nft.name}.`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Purchase failed because the marketplace execution service is unavailable.')
    } finally {
      setIsLoading(false);
    }
  };

  // Connect wallet through the real wallet-core/provider bridge.
  const handleConnectWallet = () => {
    setError('Wallet connection is unavailable until the canonical wallet-core provider bridge is configured. No wallet address was created.')
  };

  return (
    <div className={`min-h-screen ${isDarkMode ? 'bg-slate-900' : 'bg-slate-50'} text-${isDarkMode ? 'white' : 'gray-900'}`}>
      {/* Header */}
      <header className={`${isDarkMode ? 'bg-slate-800' : 'bg-white'} border-b ${isDarkMode ? 'border-slate-700' : 'border-slate-200'} p-4 sticky top-0 z-50`}>
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <a href="/" className="text-2xl">🐯</a>
            <h1 className="text-xl font-bold">NFT Marketplace</h1>
          </div>
          <div className="flex items-center gap-4">
            {walletAddress ? (
              <Chip 
                label={formatAddress(walletAddress)} 
                onDelete={() => { localStorage.removeItem('tigerwallet_address'); setWalletAddress(''); }}
                className={isDarkMode ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'}
              />
            ) : (
              <Button 
                variant="contained" 
                onClick={handleConnectWallet}
                className="bg-orange-500 hover:bg-orange-600"
              >
                Connect Wallet
              </Button>
            )}
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto p-6">
        {/* Tabs */}
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} className="mb-6">
          <Tab label="Explore" />
          <Tab label="Collections" />
          <Tab label="My NFTs" />
        </Tabs>

        {/* Search and Filters */}
        <div className="flex flex-wrap gap-4 mb-6">
          <TextField
            placeholder="Search NFTs..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            size="small"
            className="flex-1 min-w-[200px]"
            InputProps={{
              startAdornment: <InputAdornment position="start"><Search /></InputAdornment>,
            }}
          />
          <Box className="flex gap-2">
            {['all', '1', '56', '137'].map(chain => (
              <Button 
                key={chain}
                variant={filter === chain ? 'contained' : 'outlined'}
                onClick={() => setFilter(chain)}
                size="small"
                className={filter === chain ? 'bg-orange-500' : ''}
              >
                {chain === 'all' ? 'All' : CHAIN_NAMES[parseInt(chain)] || chain}
              </Button>
            ))}
          </Box>
        </div>

        {/* Error/Success Messages */}
        {error && (
          <Alert severity="error" className="mb-4" onClose={() => setError(null)}>
            {error}
          </Alert>
        )}
        {success && (
          <Alert severity="success" className="mb-4" onClose={() => setSuccess(null)}>
            {success}
          </Alert>
        )}

        {/* Loading */}
        {isLoading && (
          <Box className="flex justify-center py-12">
            <CircularProgress />
          </Box>
        )}

        {/* NFT Grid */}
        {!isLoading && (
          <Grid container spacing={3}>
            {filteredNFTs.map(nft => (
              <Grid item xs={12} sm={6} md={4} lg={3} key={nft.id}>
                <Card className={`${isDarkMode ? 'bg-slate-800' : 'bg-white'} hover:shadow-xl transition-shadow cursor-pointer`} onClick={() => setSelectedNFT(nft)}>
                  {/* NFT Image Placeholder */}
                  <Box className="h-48 bg-gradient-to-br from-orange-400 to-pink-500 flex items-center justify-center text-6xl">
                    {nft.symbol === 'BAYC' ? '🦧' : 
                     nft.symbol === 'PUNK' ? '👽' : 
                     nft.symbol === 'AZUKI' ? '🥷' : 
                     nft.symbol === 'DEGOD' ? '👻' : 
                     nft.symbol === 'MILADY' ? '💄' : '🎨'}
                  </Box>
                  <CardContent>
                    <Box className="flex items-center gap-1 mb-1">
                      <Typography variant="caption" className={isDarkMode ? 'text-slate-400' : 'text-gray-500'}>
                        {nft.symbol}
                      </Typography>
                      <Verified fontSize="small" className="text-blue-500" />
                    </Box>
                    <Typography variant="subtitle1" className="font-semibold mb-1">
                      {nft.name}
                    </Typography>
                    <Typography variant="body2" className={`mb-2 ${isDarkMode ? 'text-slate-400' : 'text-gray-500'}`} noWrap>
                      {nft.description?.substring(0, 50)}...
                    </Typography>
                    <Box className="flex justify-between items-center">
                      <Chip 
                        label={CHAIN_NAMES[nft.chain_id] || `Chain ${nft.chain_id}`} 
                        size="small"
                        className={isDarkMode ? 'bg-slate-700' : 'bg-slate-200'}
                      />
                      <Button 
                        size="small" 
                        variant="contained"
                        className="bg-orange-500 hover:bg-orange-600"
                        onClick={(e) => { e.stopPropagation(); setSelectedNFT(nft); setBuyDialogOpen(true); }}
                      >
                        View
                      </Button>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}

        {filteredNFTs.length === 0 && !isLoading && (
          <Box className="text-center py-12">
            <Collections className="text-6xl text-slate-400 mb-4" />
            <Typography variant="h6" className={isDarkMode ? 'text-slate-400' : 'text-gray-500'}>
              No NFTs found
            </Typography>
          </Box>
        )}
      </div>

      {/* NFT Detail Dialog */}
      <Dialog open={!!selectedNFT && !buyDialogOpen} onClose={() => setSelectedNFT(null)} maxWidth="md" fullWidth>
        {selectedNFT && (
          <>
            <DialogTitle className="flex justify-between items-center">
              <span>{selectedNFT.name}</span>
              <IconButton onClick={() => setSelectedNFT(null)}><Close /></IconButton>
            </DialogTitle>
            <DialogContent>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Box className="h-64 bg-gradient-to-br from-orange-400 to-pink-500 flex items-center justify-center text-8xl rounded-lg">
                    {selectedNFT.symbol === 'BAYC' ? '🦧' : 
                     selectedNFT.symbol === 'PUNK' ? '👽' : 
                     selectedNFT.symbol === 'AZUKI' ? '🥷' : '🎨'}
                  </Box>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="h6" className="mb-2">{selectedNFT.name}</Typography>
                  <Typography variant="body2" className={`mb-4 ${isDarkMode ? 'text-slate-400' : 'text-gray-500'}`}>
                    {selectedNFT.description}
                  </Typography>
                  
                  <Divider className="my-3" />
                  
                  <Typography variant="subtitle2" className="mb-2">Attributes</Typography>
                  <Box className="flex flex-wrap gap-2 mb-4">
                    {selectedNFT.attributes?.map((attr, idx) => (
                      <Chip 
                        key={idx} 
                        label={`${attr.trait_type}: ${attr.value}`} 
                        size="small"
                        variant="outlined"
                      />
                    ))}
                  </Box>
                  
                  <Divider className="my-3" />
                  
                  <Box className="flex justify-between items-center mb-4">
                    <Box>
                      <Typography variant="caption" className={isDarkMode ? 'text-slate-400' : 'text-gray-500'}>Owner</Typography>
                      <Typography variant="body2">{formatAddress(selectedNFT.owner_address)}</Typography>
                    </Box>
                    <Box className="text-right">
                      <Typography variant="caption" className={isDarkMode ? 'text-slate-400' : 'text-gray-500'}>Contract</Typography>
                      <Typography variant="body2">{formatAddress(selectedNFT.contract_address)}</Typography>
                    </Box>
                  </Box>
                  
                  <Button 
                    fullWidth 
                    variant="contained" 
                    className="bg-orange-500 hover:bg-orange-600"
                    onClick={() => setBuyDialogOpen(true)}
                  >
                    Buy Now
                  </Button>
                </Grid>
              </Grid>
            </DialogContent>
          </>
        )}
      </Dialog>

      {/* Buy Confirmation Dialog */}
      <Dialog open={buyDialogOpen} onClose={() => setBuyDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Confirm Purchase</DialogTitle>
        <DialogContent>
          {selectedNFT && (
            <Box>
              <Typography variant="body1" className="mb-4">
                Are you sure you want to purchase {selectedNFT.name}?
              </Typography>
              <Alert severity="warning" className="mb-4">
                Purchase execution is unavailable until a real wallet connection, signed transaction provider, and marketplace endpoint are configured.
              </Alert>
              <Box className="flex gap-2">
                <Button 
                  fullWidth 
                  variant="contained" 
                  className="bg-orange-500 hover:bg-orange-600"
                  onClick={() => handleBuy(selectedNFT)}
                  disabled={isLoading}
                >
                  {isLoading ? <CircularProgress size={24} /> : 'Confirm Purchase'}
                </Button>
                <Button fullWidth variant="outlined" onClick={() => setBuyDialogOpen(false)}>
                  Cancel
                </Button>
              </Box>
            </Box>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
