/**
 * Settings Slice - Complete Implementation
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface SettingsState {
  currency: string;
  language: string;
  notifications: {
    transactions: boolean;
    priceAlerts: boolean;
    news: boolean;
    marketing: boolean;
  };
  security: {
    biometricEnabled: boolean;
    autoLockTimeout: number;
    showBalance: boolean;
  };
  theme: 'light' | 'dark' | 'system';
}

const initialState: SettingsState = {
  currency: 'USD',
  language: 'en',
  notifications: {
    transactions: true,
    priceAlerts: true,
    news: true,
    marketing: false,
  },
  security: {
    biometricEnabled: false,
    autoLockTimeout: 300000,
    showBalance: true,
  },
  theme: 'dark',
};

const settingsSlice = createSlice({
  name: 'settings',
  initialState,
  reducers: {
    setCurrency: (state, action: PayloadAction<string>) => {
      state.currency = action.payload;
    },
    setLanguage: (state, action: PayloadAction<string>) => {
      state.language = action.payload;
    },
    setNotifications: (state, action: PayloadAction<Partial<SettingsState['notifications']>>) => {
      state.notifications = { ...state.notifications, ...action.payload };
    },
    setSecurity: (state, action: PayloadAction<Partial<SettingsState['security']>>) => {
      state.security = { ...state.security, ...action.payload };
    },
    setTheme: (state, action: PayloadAction<SettingsState['theme']>) => {
      state.theme = action.payload;
    },
    resetSettings: () => initialState,
  },
});

export const { setCurrency, setLanguage, setNotifications, setSecurity, setTheme, resetSettings } = settingsSlice.actions;
export default settingsSlice.reducer;
