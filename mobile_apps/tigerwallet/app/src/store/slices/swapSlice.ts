/**
 * Swap Slice - Complete Implementation
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface SwapState {
  fromToken: string | null;
  toToken: string | null;
  fromAmount: string;
  toAmount: string;
  slippage: number;
  isLoading: boolean;
  error: string | null;
}

const initialState: SwapState = {
  fromToken: null,
  toToken: null,
  fromAmount: '',
  toAmount: '',
  slippage: 0.5,
  isLoading: false,
  error: null,
};

const swapSlice = createSlice({
  name: 'swap',
  initialState,
  reducers: {
    setFromToken: (state, action: PayloadAction<string | null>) => {
      state.fromToken = action.payload;
    },
    setToToken: (state, action: PayloadAction<string | null>) => {
      state.toToken = action.payload;
    },
    setFromAmount: (state, action: PayloadAction<string>) => {
      state.fromAmount = action.payload;
    },
    setToAmount: (state, action: PayloadAction<string>) => {
      state.toAmount = action.payload;
    },
    setSlippage: (state, action: PayloadAction<number>) => {
      state.slippage = action.payload;
    },
    setSwapLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setSwapError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
    clearSwap: (state) => {
      state.fromAmount = '';
      state.toAmount = '';
      state.error = null;
    },
  },
});

export const { setFromToken, setToToken, setFromAmount, setToAmount, setSlippage, setSwapLoading, setSwapError, clearSwap } = swapSlice.actions;
export default swapSlice.reducer;
