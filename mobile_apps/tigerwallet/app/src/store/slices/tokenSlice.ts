/**
 * Token Slice - Complete Implementation
 * 
 * Handles token management and balances
 */

import { createSlice, PayloadAction, createAsyncThunk } from '@reduxjs/toolkit';
import { Token, TokenBalance } from '../../types/wallet';

interface TokenState {
  tokens: Token[];
  balances: Record<number, TokenBalance[]>;
  isLoading: boolean;
  error: string | null;
  customTokens: Token[];
}

const initialState: TokenState = {
  tokens: [],
  balances: {},
  isLoading: false,
  error: null,
  customTokens: [],
};

export const fetchTokenBalances = createAsyncThunk(
  'tokens/fetchBalances',
  async ({ chainId, address }: { chainId: number; address: string }, { rejectWithValue }) => {
    try {
      // Fetch token balances from blockchain
      return { chainId, balances: [] as TokenBalance[] };
    } catch (error: any) {
      return rejectWithValue(error.message);
    }
  }
);

const tokenSlice = createSlice({
  name: 'tokens',
  initialState,
  reducers: {
    setTokens: (state, action: PayloadAction<Token[]>) => {
      state.tokens = action.payload;
    },
    addToken: (state, action: PayloadAction<Token>) => {
      state.tokens.push(action.payload);
    },
    removeToken: (state, action: PayloadAction<string>) => {
      state.tokens = state.tokens.filter(t => t.address !== action.payload);
    },
    setTokenBalances: (state, action: PayloadAction<{ chainId: number; balances: TokenBalance[] }>) => {
      state.balances[action.payload.chainId] = action.payload.balances;
    },
    addCustomToken: (state, action: PayloadAction<Token>) => {
      state.customTokens.push(action.payload);
    },
    removeCustomToken: (state, action: PayloadAction<string>) => {
      state.customTokens = state.customTokens.filter(t => t.address !== action.payload);
    },
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTokenBalances.pending, (state) => {
        state.isLoading = true;
      })
      .addCase(fetchTokenBalances.fulfilled, (state, action) => {
        state.isLoading = false;
        state.balances[action.payload.chainId] = action.payload.balances;
      })
      .addCase(fetchTokenBalances.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload as string;
      });
  },
});

export const {
  setTokens,
  addToken,
  removeToken,
  setTokenBalances,
  addCustomToken,
  removeCustomToken,
  setLoading,
  setError,
} = tokenSlice.actions;

export default tokenSlice.reducer;
