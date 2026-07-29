/**
 * DApp Slice - Complete Implementation
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface DApp {
  id: string;
  name: string;
  url: string;
  iconUrl?: string;
  description?: string;
  category: string;
  chainIds: number[];
}

interface DAppConnection {
  dappId: string;
  address: string;
  connectedAt: number;
}

interface DAppState {
  dapps: DApp[];
  connections: DAppConnection[];
  isLoading: boolean;
  error: string | null;
}

const initialState: DAppState = {
  dapps: [],
  connections: [],
  isLoading: false,
  error: null,
};

const dappSlice = createSlice({
  name: 'dapps',
  initialState,
  reducers: {
    setDapps: (state, action: PayloadAction<DApp[]>) => {
      state.dapps = action.payload;
    },
    addDapp: (state, action: PayloadAction<DApp>) => {
      state.dapps.push(action.payload);
    },
    removeDapp: (state, action: PayloadAction<string>) => {
      state.dapps = state.dapps.filter(d => d.id !== action.payload);
    },
    addConnection: (state, action: PayloadAction<DAppConnection>) => {
      state.connections.push(action.payload);
    },
    removeConnection: (state, action: PayloadAction<string>) => {
      state.connections = state.connections.filter(c => c.dappId !== action.payload);
    },
    setDappLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setDappError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
});

export const { setDapps, addDapp, removeDapp, addConnection, removeConnection, setDappLoading, setDappError } = dappSlice.actions;
export default dappSlice.reducer;
