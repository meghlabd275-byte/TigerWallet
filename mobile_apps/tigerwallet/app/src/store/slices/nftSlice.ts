/**
 * NFT Slice - Complete Implementation
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface NFT {
  id: string;
  contractAddress: string;
  tokenId: string;
  name: string;
  description?: string;
  imageUrl: string;
  chainId: number;
  owner: string;
}

interface NFTState {
  nfts: NFT[];
  isLoading: boolean;
  error: string | null;
}

const initialState: NFTState = {
  nfts: [],
  isLoading: false,
  error: null,
};

const nftSlice = createSlice({
  name: 'nfts',
  initialState,
  reducers: {
    setNfts: (state, action: PayloadAction<NFT[]>) => {
      state.nfts = action.payload;
    },
    addNft: (state, action: PayloadAction<NFT>) => {
      state.nfts.push(action.payload);
    },
    removeNft: (state, action: PayloadAction<string>) => {
      state.nfts = state.nfts.filter(n => n.id !== action.payload);
    },
    setNftLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setNftError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
});

export const { setNfts, addNft, removeNft, setNftLoading, setNftError } = nftSlice.actions;
export default nftSlice.reducer;
