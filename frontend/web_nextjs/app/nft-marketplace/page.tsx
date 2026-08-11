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

// Same-origin API base: the Next.js app proxies /api/v1/nft/* to the
// go/nft_service backend (see app/api/v1/nft/ routes).
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

interface NFT {
  id: string;
  listing_id?: string;
  token_id: string;
  contract_address: string;
  name: string;
  symbol: string;
  description: string;
  image_url: string;
  animation_url?: string;
  attributes: { trait_type: string; value: string; rarity?: string }[];
  owner: string;
  price: number;
  price_token: string;
  chain_id: number;
}

interface NFTCollection {
  id: string;
  name: string;
  symbol: string;
  contract_address: string;
  chain_id: number;
  total_supply: number;
  floor_price: number;
  volume_24h: number;
  image_url: string;
}

const CHAIN_NAMES: Record<number, string> = {
  1: 'Ethereum',
  56: 'BNB Chain',
  137: 'Polygon',
  42161: 'Arbitrum',
  43114: 'Avalanche',
};

export default function NFTMarketplace() {
  const [activeTab, setActiveTab] = useState(0);
  const [nfts, setNfts] = useState<NFT[]>([]);
  const [collections, setCollections] = useState<NFTCollection[]>([]);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState('all');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [selectedNFT, setSelectedNFT] = useState<NFT | null>(null);
  const [walletAddress, setWalletAddress] = useState<string>('');
  const [buyDialogOpen, setBuyDialogOpen] = useState(false);
  const { isDark } = useTheme();

  useEffect(() => {
    const savedWallet = localStorage.getItem('tigerwallet_address');
    if (savedWallet) {
      setWalletAddress(savedWallet);
    }
  }, []);

  const fetchNFTs = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE}/api/v1/nft/listings?status=active`);
      if (response.ok) {
        const data = await response.json();
        if (data.listings && data.listings.length > 0) {
          // Map backend NFTListing fields to the frontend NFT interface.
          // Listings only carry price/seller/status; full metadata (name,
          // contract address, attributes) requires a per-NFT detail call, so
          // those fields are left empty rather than fabricated.
          const mapped: NFT[] = data.listings.map((l: any) => ({
            id: l.nft_id || l.id,
            listing_id: l.id,
            token_id: '',
            contract_address: '',
            name: `NFT ${l.nft_id || l.id}`,
            symbol: '',
            description: '',
            image_url: '',
            attributes: [],
            owner: l.seller || '',
            price: parseFloat(l.price) || 0,
            price_token: l.price_token || 'ETH',
            chain_id: 1,
          }));
          setNfts(mapped);
        } else {
          setNfts([]);
        }
      } else {
        setNfts([]);
      }
    } catch (err) {
      // Backend unreachable: show no NFTs rather than fabricated ones.
      setNfts([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchCollections = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/nft/collections?chain_id=1`);
      if (response.ok) {
        const data = await response.json();
        if (data.collections) {
          setCollections(data.collections);
        }
      }
    } catch (err) {
      // Collections not available
    }
  }, []);

  useEffect(() => {
    fetchNFTs();
    fetchCollections();
  }, [fetchNFTs, fetchCollections]);

  const filteredNFTs = nfts.filter(nft => {
    const matchesSearch = nft.name.toLowerCase().includes(search.toLowerCase()) || 
                         nft.symbol.toLowerCase().includes(search.toLowerCase());
    if (filter === 'all') return matchesSearch;
    return matchesSearch && nft.chain_id === parseInt(filter);
  });

  const handleBuy = async (nft: NFT) => {
    if (!walletAddress) {
      setError('Please connect your wallet first');
      return;
    }

    setIsLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const token = localStorage.getItem('tigerwallet_token');
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (token) headers['Authorization'] = `Bearer ${token}`;

      const response = await fetch(`${API_BASE}/api/v1/nft/buy`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          listing_id: nft.listing_id || nft.id,
        }),
      });

      const data = await response.json();
      if (data.success && data.tx_id) {
        setSuccess(`Successfully purchased ${nft.name}! Order: ${data.tx_id}`);
        setBuyDialogOpen(false);
        setSelectedNFT(null);
        fetchNFTs();
      } else {
        setError(data.error || 'Purchase failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Purchase failed');
    } finally {
      setIsLoading(false);
    }
  };

  const getNFTIcon = (symbol: string) => {
    switch (symbol) {
      case 'BAYC': return '🦧';
      case 'PUNK': return '👽';
      case 'AZUKI': return '🥷';
      case 'DEGOD': return '👻';
      case 'MILADY': return '💄';
      default: return '🎨';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'} p-6`}>
      <header className="mb-8">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <span className="text-4xl">🖼️</span>
            <h1 className="text-2xl font-bold">NFT Marketplace</h1>
          </div>
          {walletAddress ? (
            <Chip 
              label={`${walletAddress.slice(0, 6)}...${walletAddress.slice(-4)}`}
              className={isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'}
            />
          ) : (
            <Button 
              variant="contained" 
              onClick={() => setError('Please connect wallet first')}
              className="bg-orange-500 hover:bg-orange-600"
            >
              Connect Wallet
            </Button>
          )}
        </div>
      </header>

      <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} className="mb-6">
        <Tab label="Explore" />
        <Tab label="Collections" />
        <Tab label="My NFTs" />
      </Tabs>

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

      {error && <Alert severity="error" className="mb-4" onClose={() => setError(null)}>{error}</Alert>}
      {success && <Alert severity="success" className="mb-4" onClose={() => setSuccess(null)}>{success}</Alert>}

      {isLoading && (
        <Box className="flex justify-center py-12">
          <CircularProgress />
        </Box>
      )}

      {!isLoading && (
        <Grid container spacing={3}>
          {filteredNFTs.map(nft => (
            <Grid item xs={12} sm={6} md={4} lg={3} key={nft.id}>
              <Card 
                className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} hover:shadow-xl transition-shadow cursor-pointer`}
                onClick={() => { setSelectedNFT(nft); setBuyDialogOpen(true); }}
              >
                <Box className="h-48 bg-gradient-to-br from-orange-400 to-pink-500 flex items-center justify-center text-6xl">
                  {getNFTIcon(nft.symbol)}
                </Box>
                <CardContent>
                  <Box className="flex items-center gap-1 mb-1">
                    <Typography variant="caption" className={isDark ? 'text-gray-400' : 'text-gray-500'}>
                      {nft.symbol}
                    </Typography>
                    <Verified fontSize="small" className="text-blue-500" />
                  </Box>
                  <Typography variant="subtitle1" className="font-semibold mb-1">
                    {nft.name}
                  </Typography>
                  <Typography variant="body2" className={`mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`} noWrap>
                    {nft.description?.substring(0, 50)}...
                  </Typography>
                  <Box className="flex justify-between items-center">
                    <Chip 
                      label={CHAIN_NAMES[nft.chain_id] || `Chain ${nft.chain_id}`} 
                      size="small"
                      className={isDark ? 'bg-gray-700' : 'bg-gray-200'}
                    />
                    <Typography variant="subtitle2" className="text-orange-500">
                      {nft.price} {nft.price_token}
                    </Typography>
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
          <Typography variant="h6" className={isDark ? 'text-gray-400' : 'text-gray-500'}>
            No NFTs found
          </Typography>
        </Box>
      )}

      <Dialog open={buyDialogOpen && !!selectedNFT} onClose={() => { setBuyDialogOpen(false); setSelectedNFT(null); }} maxWidth="md" fullWidth>
        {selectedNFT && (
          <>
            <DialogTitle className="flex justify-between items-center">
              <span>{selectedNFT.name}</span>
              <IconButton onClick={() => { setBuyDialogOpen(false); setSelectedNFT(null); }}><Close /></IconButton>
            </DialogTitle>
            <DialogContent>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Box className="h-64 bg-gradient-to-br from-orange-400 to-pink-500 flex items-center justify-center text-8xl rounded-lg">
                    {getNFTIcon(selectedNFT.symbol)}
                  </Box>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="h6" className="mb-2">{selectedNFT.name}</Typography>
                  <Typography variant="body2" className={`mb-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
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
                      <Typography variant="caption" className={isDark ? 'text-gray-400' : 'text-gray-500'}>Owner</Typography>
                      <Typography variant="body2">{selectedNFT.owner.slice(0, 8)}...{selectedNFT.owner.slice(-6)}</Typography>
                    </Box>
                    <Box className="text-right">
                      <Typography variant="caption" className={isDark ? 'text-gray-400' : 'text-gray-500'}>Price</Typography>
                      <Typography variant="h5" className="text-orange-500">{selectedNFT.price} {selectedNFT.price_token}</Typography>
                    </Box>
                  </Box>
                  
                  <Button 
                    fullWidth 
                    variant="contained" 
                    className="bg-orange-500 hover:bg-orange-600"
                    onClick={() => handleBuy(selectedNFT)}
                    disabled={isLoading || !walletAddress}
                    startIcon={isLoading ? <CircularProgress size={24} /> : <ShoppingCart />}
                  >
                    {!walletAddress ? 'Connect Wallet' : isLoading ? 'Processing...' : 'Buy Now'}
                  </Button>
                </Grid>
              </Grid>
            </DialogContent>
          </>
        )}
      </Dialog>
    </div>
  );
}
