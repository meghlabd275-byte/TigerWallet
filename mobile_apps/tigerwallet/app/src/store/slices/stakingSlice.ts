/**
 * Staking Slice - Complete Implementation
 */

import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface StakingPosition {
  id: string;
  token: string;
  amount: string;
  rewards: string;
  startTime: number;
  endTime?: number;
}

interface StakingState {
  positions: StakingPosition[];
  isLoading: boolean;
  error: string | null;
}

const initialState: StakingState = {
  positions: [],
  isLoading: false,
  error: null,
};

const stakingSlice = createSlice({
  name: 'staking',
  initialState,
  reducers: {
    setPositions: (state, action: PayloadAction<StakingPosition[]>) => {
      state.positions = action.payload;
    },
    addPosition: (state, action: PayloadAction<StakingPosition>) => {
      state.positions.push(action.payload);
    },
    updatePosition: (state, action: PayloadAction<StakingPosition>) => {
      const index = state.positions.findIndex(p => p.id === action.payload.id);
      if (index !== -1) {
        state.positions[index] = action.payload;
      }
    },
    removePosition: (state, action: PayloadAction<string>) => {
      state.positions = state.positions.filter(p => p.id !== action.payload);
    },
    setStakingLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    setStakingError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
});

export const { setPositions, addPosition, updatePosition, removePosition, setStakingLoading, setStakingError } = stakingSlice.actions;
export default stakingSlice.reducer;
