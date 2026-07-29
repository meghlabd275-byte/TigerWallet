/**
 * User Slice - Complete Implementation
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface User {
  id: string;
  email?: string;
  phone?: string;
  kycStatus: 'none' | 'pending' | 'verified' | 'rejected';
  createdAt: number;
}

interface UserState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

const initialState: UserState = {
  user: null,
  isAuthenticated: false,
  isLoading: false,
  error: null,
};

const userSlice = createSlice({
  name: 'user',
  initialState,
  reducers: {
    setUser: (state, action: PayloadAction<User | null>) => {
      state.user = action.payload;
      state.isAuthenticated = !!action.payload;
    },
    setAuthLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
    updateKycStatus: (state, action: PayloadAction<User['kycStatus']>) => {
      if (state.user) {
        state.user.kycStatus = action.payload;
      }
    },
    clearUser: (state) => {
      state.user = null;
      state.isAuthenticated = false;
    },
  },
});

export const { setUser, setAuthLoading, setError, updateKycStatus, clearUser } = userSlice.actions;
export default userSlice.reducer;
