/**
 * Transaction Slice - Complete Implementation
 * 
 * Handles transaction history and management
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Transaction } from '../../types/wallet';

interface TransactionState {
  transactions: Transaction[];
  pendingTransactions: Transaction[];
  isLoading: boolean;
  error: string | null;
}

const initialState: TransactionState = {
  transactions: [],
  pendingTransactions: [],
  isLoading: false,
  error: null,
};

const transactionSlice = createSlice({
  name: 'transactions',
  initialState,
  reducers: {
    setTransactions: (state, action: PayloadAction<Transaction[]>) => {
      state.transactions = action.payload;
    },
    addTransaction: (state, action: PayloadAction<Transaction>) => {
      state.transactions.unshift(action.payload);
    },
    updateTransaction: (state, action: PayloadAction<Transaction>) => {
      const index = state.transactions.findIndex(t => t.hash === action.payload.hash);
      if (index !== -1) {
        state.transactions[index] = action.payload;
      }
    },
    removeTransaction: (state, action: PayloadAction<string>) => {
      state.transactions = state.transactions.filter(t => t.hash !== action.payload);
    },
    addPendingTransaction: (state, action: PayloadAction<Transaction>) => {
      state.pendingTransactions.push(action.payload);
    },
    removePendingTransaction: (state, action: PayloadAction<string>) => {
      state.pendingTransactions = state.pendingTransactions.filter(t => t.hash !== action.payload);
    },
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
    clearTransactions: (state) => {
      state.transactions = [];
      state.pendingTransactions = [];
    },
  },
});

export const {
  setTransactions,
  addTransaction,
  updateTransaction,
  removeTransaction,
  addPendingTransaction,
  removePendingTransaction,
  setLoading,
  setError,
  clearTransactions,
} = transactionSlice.actions;

export default transactionSlice.reducer;
