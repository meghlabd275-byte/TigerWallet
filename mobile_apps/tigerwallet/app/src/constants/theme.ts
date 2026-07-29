/**
 * TigerWallet Theme Constants - Complete Implementation
 */

export const COLORS = {
  // Light theme
  backgroundLight: '#FFFFFF',
  cardLight: '#F8F9FA',
  textLight: '#1A1A1A',
  borderLight: '#E5E7EB',
  
  // Dark theme
  backgroundDark: '#0D0D0D',
  cardDark: '#1A1A1A',
  textDark: '#FFFFFF',
  borderDark: '#2D2D2D',
  
  // Primary colors
  primary: '#FF6B35',
  primaryLight: '#FF8A5C',
  primaryDark: '#E55A2B',
  
  // Accent colors
  accent: '#00D9FF',
  accentLight: '#33E1FF',
  accentDark: '#00B8D9',
  
  // Status colors
  success: '#10B981',
  warning: '#F59E0B',
  error: '#EF4444',
  info: '#3B82F6',
  
  // Common colors
  white: '#FFFFFF',
  black: '#000000',
  gray: '#6B7280',
  lightGray: '#9CA3AF',
  darkGray: '#374151',
  
  // Gradient
  gradientStart: '#FF6B35',
  gradientEnd: '#00D9FF',
};

export const SPACING = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
};

export const FONT_SIZES = {
  xs: 10,
  sm: 12,
  md: 14,
  lg: 16,
  xl: 18,
  xxl: 24,
  xxxl: 32,
};

export const BORDER_RADIUS = {
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
  xxl: 24,
  full: 9999,
};

export const SHADOWS = {
  sm: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.1,
    shadowRadius: 2,
    elevation: 2,
  },
  md: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.15,
    shadowRadius: 4,
    elevation: 4,
  },
  lg: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 8,
  },
};

export const getThemeColors = (isDark: boolean) => ({
  background: isDark ? COLORS.backgroundDark : COLORS.backgroundLight,
  card: isDark ? COLORS.cardDark : COLORS.cardLight,
  text: isDark ? COLORS.textDark : COLORS.textLight,
  border: isDark ? COLORS.borderDark : COLORS.borderLight,
  primary: COLORS.primary,
  accent: COLORS.accent,
  success: COLORS.success,
  warning: COLORS.warning,
  error: COLORS.error,
  info: COLORS.info,
});
