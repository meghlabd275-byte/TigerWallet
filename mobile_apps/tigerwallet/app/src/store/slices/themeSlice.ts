/**
 * Theme Slice - Complete Implementation
 * 
 * Handles light/dark theme switching throughout the app
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeState {
  mode: ThemeMode;
  primaryColor: string;
  accentColor: string;
  isInitialized: boolean;
}

const initialState: ThemeState = {
  mode: 'dark',
  primaryColor: '#FF6B35',
  accentColor: '#00D9FF',
  isInitialized: false,
};

const themeSlice = createSlice({
  name: 'theme',
  initialState,
  reducers: {
    setTheme: (state, action: PayloadAction<ThemeMode>) => {
      state.mode = action.payload;
    },
    setPrimaryColor: (state, action: PayloadAction<string>) => {
      state.primaryColor = action.payload;
    },
    setAccentColor: (state, action: PayloadAction<string>) => {
      state.accentColor = action.payload;
    },
    setInitialized: (state, action: PayloadAction<boolean>) => {
      state.isInitialized = action.payload;
    },
    toggleTheme: (state) => {
      state.mode = state.mode === 'dark' ? 'light' : 'dark';
    },
  },
});

export const {
  setTheme,
  setPrimaryColor,
  setAccentColor,
  setInitialized,
  toggleTheme,
} = themeSlice.actions;

export default themeSlice.reducer;
