/**
 * TigerWallet Redux Store - Complete Implementation
 * 
 * No stubs, no simulations - Production-ready
 */

import { configureStore, combineReducers } from '@reduxjs/toolkit';
import { 
  persistStore, 
  persistReducer,
  FLUSH,
  REHYDRATE,
  PAUSE,
  PERSIST,
  PURGE,
  REGISTER,
} from 'redux-persist';
import AsyncStorage from '@react-native-async-storage/async-storage';

// Slices
import themeReducer from './slices/themeSlice';
import walletReducer from './slices/walletSlice';
import userReducer from './slices/userSlice';
import networkReducer from './slices/networkSlice';
import tokenReducer from './slices/tokenSlice';
import transactionReducer from './slices/transactionSlice';
import swapReducer from './slices/swapSlice';
import stakingReducer from './slices/stakingSlice';
import nftReducer from './slices/nftSlice';
import dappReducer from './slices/dappSlice';
import settingsReducer from './slices/settingsSlice';

// Combine reducers
const rootReducer = combineReducers({
  theme: themeReducer,
  wallet: walletReducer,
  user: userReducer,
  network: networkReducer,
  tokens: tokenReducer,
  transactions: transactionReducer,
  swap: swapReducer,
  staking: stakingReducer,
  nfts: nftReducer,
  dapps: dappReducer,
  settings: settingsReducer,
});

// Persist config
const persistConfig = {
  key: 'tigerwallet',
  version: 1,
  storage: AsyncStorage,
  whitelist: ['theme', 'wallet', 'network', 'tokens', 'settings'],
};

// Persisted reducer
const persistedReducer = persistReducer(persistConfig, rootReducer);

// Create store
export const store = configureStore({
  reducer: persistedReducer,
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        ignoredActions: [FLUSH, REHYDRATE, PAUSE, PERSIST, PURGE, REGISTER],
      },
    }),
});

// Create persistor
export const persistor = persistStore(store);

// Export types
export type RootState = ReturnType<typeof rootReducer>;
export type AppDispatch = typeof store.dispatch;
