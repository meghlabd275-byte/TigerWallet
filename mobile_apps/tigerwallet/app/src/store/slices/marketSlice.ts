/**
 * Market Data Slice - Redux store for real-time market data
 * Supports: Tickers, Order Books, Trades via WebSocket
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export interface TickerData {
  pair: string;
  price: string;
  change24h: number;
  volume24h: number;
  high24h: string;
  low24h: string;
  timestamp: number;
}

export interface OrderBookEntry {
  price: string;
  amount: string;
  total: string;
}

export interface OrderBookData {
  pair: string;
  bids: OrderBookEntry[];
  asks: OrderBookEntry[];
  timestamp: number;
}

export interface TradeData {
  id: string;
  pair: string;
  price: string;
  amount: string;
  side: 'buy' | 'sell';
  timestamp: number;
}

interface MarketState {
  tickers: Record<string, TickerData>;
  orderBooks: Record<string, OrderBookData>;
  trades: Record<string, TradeData[]>;
  isConnected: boolean;
  lastUpdate: number;
}

const initialState: MarketState = {
  tickers: {},
  orderBooks: {},
  trades: {},
  isConnected: false,
  lastUpdate: 0,
};

const marketSlice = createSlice({
  name: 'market',
  initialState,
  reducers: {
    updateMarketData: (state, action: PayloadAction<TickerData>) => {
      const { pair } = action.payload;
      state.tickers[pair] = action.payload;
      state.lastUpdate = Date.now();
    },
    
    updateOrderBook: (state, action: PayloadAction<OrderBookData>) => {
      const { pair } = action.payload;
      state.orderBooks[pair] = action.payload;
      state.lastUpdate = Date.now();
    },
    
    updateTrade: (state, action: PayloadAction<TradeData>) => {
      const { pair } = action.payload;
      if (!state.trades[pair]) {
        state.trades[pair] = [];
      }
      // Keep only last 100 trades
      state.trades[pair] = [action.payload, ...state.trades[pair]].slice(0, 100);
      state.lastUpdate = Date.now();
    },
    
    setConnected: (state, action: PayloadAction<boolean>) => {
      state.isConnected = action.payload;
    },
    
    clearMarketData: (state) => {
      state.tickers = {};
      state.orderBooks = {};
      state.trades = {};
      state.lastUpdate = 0;
    },
  },
});

export const {
  updateMarketData,
  updateOrderBook,
  updateTrade,
  setConnected,
  clearMarketData,
} = marketSlice.actions;

export default marketSlice.reducer;
