/**
 * Network Slice - Complete Implementation
 * 
 * Handles blockchain network selection and management
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Chain } from '../../types/wallet';

interface NetworkState {
  selectedChainId: number;
  chains: Chain[];
  isLoading: boolean;
  rpcStatus: Record<number, 'connected' | 'disconnected' | 'error'>;
}

const initialState: NetworkState = {
  selectedChainId: 1,
  chains: [],
  isLoading: false,
  rpcStatus: {},
};

const networkSlice = createSlice({
  name: 'network',
  initialState,
  reducers: {
    setSelectedChain: (state, action: PayloadAction<number>) => {
      state.selectedChainId = action.payload;
    },
    setChains: (state, action: PayloadAction<Chain[]>) => {
      state.chains = action.payload;
    },
    addChain: (state, action: PayloadAction<Chain>) => {
      state.chains.push(action.payload);
    },
    updateChain: (state, action: PayloadAction<Chain>) => {
      const index = state.chains.findIndex(c => c.id === action.payload.id);
      if (index !== -1) {
        state.chains[index] = action.payload;
      }
    },
    removeChain: (state, action: PayloadAction<number>) => {
      state.chains = state.chains.filter(c => c.id !== action.payload);
    },
    setRpcStatus: (state, action: PayloadAction<{ chainId: number; status: 'connected' | 'disconnected' | 'error' }>) => {
      state.rpcStatus[action.payload.chainId] = action.payload.status;
    },
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
  },
});

export const {
  setSelectedChain,
  setChains,
  addChain,
  updateChain,
  removeChain,
  setRpcStatus,
  setLoading,
} = networkSlice.actions;

export default networkSlice.reducer;
