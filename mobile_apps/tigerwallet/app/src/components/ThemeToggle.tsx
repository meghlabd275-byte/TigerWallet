/**
 * ThemeToggle Component - Complete Implementation
 * 
 * Light/Dark theme switch that works everywhere
 */

import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../store';
import { setTheme, toggleTheme } from '../../store/slices/themeSlice';
import { COLORS, SPACING } from '../../constants/theme';

export const ThemeToggle: React.FC = () => {
  const dispatch = useDispatch();
  const theme = useSelector((state: RootState) => state.theme.mode);
  const isDark = theme === 'dark';

  const handleToggle = () => {
    dispatch(toggleTheme());
  };

  return (
    <TouchableOpacity 
      style={[styles.container, isDark ? styles.darkContainer : styles.lightContainer]} 
      onPress={handleToggle}
      activeOpacity={0.7}
    >
      <View style={[styles.track, isDark ? styles.darkTrack : styles.lightTrack]}>
        <View style={[styles.thumb, isDark ? styles.darkThumb : styles.lightThumb]}>
          <Text style={styles.icon}>{isDark ? '🌙' : '☀️'}</Text>
        </View>
      </View>
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  container: {
    padding: SPACING.xs,
  },
  lightContainer: {
    backgroundColor: COLORS.cardLight,
  },
  darkContainer: {
    backgroundColor: COLORS.cardDark,
  },
  track: {
    width: 50,
    height: 28,
    borderRadius: 14,
    justifyContent: 'center',
    padding: 2,
  },
  lightTrack: {
    backgroundColor: '#E5E7EB',
  },
  darkTrack: {
    backgroundColor: '#374151',
  },
  thumb: {
    width: 24,
    height: 24,
    borderRadius: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  lightThumb: {
    backgroundColor: COLORS.white,
    alignSelf: 'flex-start',
  },
  darkThumb: {
    backgroundColor: COLORS.primary,
    alignSelf: 'flex-end',
  },
  icon: {
    fontSize: 14,
  },
});

export default ThemeToggle;
