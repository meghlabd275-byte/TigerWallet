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

const MOCK_NFTS: NFT[] = [
  { id: 1, token_id: '7854', contract_address: '0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D', owner_address: '0x742d35Cc6634C0532925a3b844Bc9e7595f5eA1E', name: 'Bored Ape #7854', symbol: 'BAYC', description: 'The Bored Ape Yacht Club is a collection of 10,000 unique Bored Ape NFTs.', image_url: '', animation_url: '', attributes: [{ trait_type: 'Background', value: 'Orange' }, { trait_type: 'Fur', value: 'Dark Brown' }], uri: '', chain_id: 1 },
  { id: 2, token_id: '3456', contract_address: '0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB', owner_address: '0xabcd...1234', name: 'CryptoPunk #3456', symbol: 'PUNK', description: 'One of 10,000 unique collectible characters with proof of ownership stored on the Ethereum blockchain.', image_url: '', animation_url: '', attributes: [{ trait_type: 'Type', value: 'Alien' }, { trait_type: 'Accessory', value: 'Cap' }], uri: '', chain_id: 1 },
  { id: 3, token_id: '7821', contract_address: '0xED5AF388653567Af2F388E6224dC7C4b3241C544', owner_address: '0xdef1...5678', name: 'Azuki #7821', symbol: 'AZUKI', description: 'Azuki starts with a collection of 10,000 avatars that give you membership access to The Garden.', image_url: '', animation_url: '', attributes: [{ trait_type: 'Type', value: 'Human' }, { trait_type: 'Hair', value: 'Pink' }], uri: '', chain_id: 1 },
  { id: 4, token_id: '999', contract_address: '0x8821aDD4d618C616d97eFBB33E8fA60f9fA1E73f', owner_address: 'G3d...xyz', name: 'DeGod #999', symbol: 'DEGOD', description: 'DeGods is a digital character collection and community.', image_url: '', animation_url: '', attributes: [{ trait_type: 'Type', value: 'Alien' }, { trait_type: 'Background', value: 'Red' }], uri: '', chain_id: 1 },
  { id: 5, token_id: '123', contract_address: '0x763bE8c3E1A4D0D0A6d4e1E9f7A2C8e3F9D2E1B', owner_address: 'A7x...123', name: 'MadLads #123', symbol: 'MAD', description: 'Mad Lads is a collection of 10,000 NFTs on Solana.', image_url: '', animation_url: '', attributes: [{ trait_type: 'Type', value: 'Skeleton' }], uri: '', chain_id: 1 },
  { id: 6, token_id: '4567', contract_address: '0xA7945d92d6b7AE6534fDFA76a96fE50dA8fEBb8d', owner_address: '0x9876...abcd', name: 'Milady #4567', symbol: 'MILADY', description: 'Milady is a collection of 10,000 NFTs.', image_url: '', animation_url: '', attributes: [{ trait_type: 'Type', value: 'Human' }], uri: '', chain_id: 1 },
];

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
  const [nfts, setNfts] = useState<NFT[]>(MOCK_NFTS);
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
      if (response.ok) {
        const data = await response.json();
        if (data.nfts && data.nfts.length > 0) {
          setNfts(data.nfts);
        }
      }
    } catch (err) {
      console.log('Using mock NFT data');
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
      // Simulate API call
      setSuccess(`Successfully purchased ${nft.name}! Transaction hash will be available shortly.`);
      setBuyDialogOpen(false);
    } catch (err) {
      setError('Purchase failed. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  // Connect wallet
  const handleConnectWallet = () => {
    const address = '0x742d35Cc6634C0532925a3b844Bc9e7595f5eA1E';
    localStorage.setItem('tigerwallet_address', address);
    setWalletAddress(address);
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
              <Alert severity="info" className="mb-4">
                This is a simulated transaction for demonstration purposes.
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
