/**
 * TigerWallet Theme System
 * 
 * Unified theming system that works across:
 * - Mobile (iOS/Android - Swift/React Native/Kotlin)
 * - Web (React/Vue/Angular)
 * - Desktop (Electron/React Native)
 * - Browser Extension (React)
 * 
 * Features:
 * - Light/Dark mode switching
 * - System preference detection
 * - Smooth transitions
 * - Persistent preference storage
 * - High contrast support
 * - Reduced motion support
 */

// ============================================================================
// Theme Types & Constants
// ============================================================================

/** Theme mode options */
export type ThemeMode = 'light' | 'dark' | 'system';

/** Color scheme definitions */
export interface ColorScheme {
  // Primary colors
  primary: string;
  primaryLight: string;
  primaryDark: string;
  onPrimary: string;
  
  // Secondary colors
  secondary: string;
  secondaryLight: string;
  secondaryDark: string;
  onSecondary: string;
  
  // Background colors
  background: string;
  backgroundSecondary: string;
  backgroundTertiary: string;
  surface: string;
  surfaceVariant: string;
  onBackground: string;
  onSurface: string;
  
  // Text colors
  textPrimary: string;
  textSecondary: string;
  textTertiary: string;
  textDisabled: string;
  textInverse: string;
  
  // Border colors
  border: string;
  borderLight: string;
  borderDark: string;
  
  // Status colors
  success: string;
  successLight: string;
  onSuccess: string;
  
  warning: string;
  warningLight: string;
  onWarning: string;
  
  error: string;
  errorLight: string;
  onError: string;
  
  info: string;
  infoLight: string;
  onInfo: string;
  
  // Accent colors
  accent: string;
  accentLight: string;
  
  // Gradient
  gradient: {
    primary: string;
    secondary: string;
    success: string;
  };
  
  // Shadows
  shadow: {
    sm: string;
    md: string;
    lg: string;
    xl: string;
  };
  
  // Overlay
  overlay: string;
  scrim: string;
}

/** Typography definitions */
export interface Typography {
  fontFamily: {
    primary: string;
    secondary: string;
    mono: string;
  };
  
  fontSize: {
    xs: string;
    sm: string;
    base: string;
    lg: string;
    xl: string;
    '2xl': string;
    '3xl': string;
    '4xl': string;
    '5xl': string;
  };
  
  fontWeight: {
    light: number;
    normal: number;
    medium: number;
    semibold: number;
    bold: number;
  };
  
  lineHeight: {
    tight: number;
    normal: number;
    relaxed: number;
  };
  
  letterSpacing: {
    tighter: string;
    tight: string;
    normal: string;
    wide: string;
    wider: string;
  };
}

/** Spacing definitions */
export interface Spacing {
  0: string;
  1: string;
  2: string;
  3: string;
  4: string;
  5: string;
  6: string;
  7: string;
  8: string;
  9: string;
  10: string;
  12: string;
  14: string;
  16: string;
  20: string;
  24: string;
  28: string;
  32: string;
  36: string;
  40: string;
  44: string;
  48: string;
  52: string;
  56: string;
  60: string;
  64: string;
  72: string;
  80: string;
  96: string;
}

/** Border radius definitions */
export interface BorderRadius {
  none: string;
  sm: string;
  DEFAULT: string;
  md: string;
  lg: string;
  xl: string;
  '2xl': string;
  '3xl': string;
  full: string;
}

/** Animation definitions */
export interface Animation {
  duration: {
    instant: string;
    fastest: string;
    faster: string;
    fast: string;
    normal: string;
    slow: string;
    slower: string;
    slowest: string;
  };
  
  easing: {
    linear: string;
    easeIn: string;
    easeOut: string;
    easeInOut: string;
    bounce: string;
  };
  
  transitions: {
    none: string;
    fast: string;
    normal: string;
    slow: string;
  };
}

/** Complete theme configuration */
export interface Theme {
  mode: ThemeMode;
  colors: ColorScheme;
  typography: Typography;
  spacing: Spacing;
  borderRadius: BorderRadius;
  animation: Animation;
  breakpoints: {
    sm: string;
    md: string;
    lg: string;
    xl: string;
    '2xl': string;
  };
  zIndex: {
    dropdown: number;
    sticky: number;
    modal: number;
    popover: number;
    tooltip: number;
    toast: number;
  };
}

// ============================================================================
// Theme Definitions
// ============================================================================

/** Light theme colors */
const lightColors: ColorScheme = {
  // Primary
  primary: '#FF6B35',
  primaryLight: '#FF8F66',
  primaryDark: '#E55100',
  onPrimary: '#FFFFFF',
  
  // Secondary
  secondary: '#1E3A5F',
  secondaryLight: '#2E5A8F',
  secondaryDark: '#0E2A4F',
  onSecondary: '#FFFFFF',
  
  // Background
  background: '#FAFBFC',
  backgroundSecondary: '#F0F2F5',
  backgroundTertiary: '#E8EAED',
  surface: '#FFFFFF',
  surfaceVariant: '#F5F7FA',
  onBackground: '#1A1A2E',
  onSurface: '#1A1A2E',
  
  // Text
  textPrimary: '#1A1A2E',
  textSecondary: '#4A5568',
  textTertiary: '#718096',
  textDisabled: '#A0AEC0',
  textInverse: '#FFFFFF',
  
  // Border
  border: '#E2E8F0',
  borderLight: '#EDF2F7',
  borderDark: '#CBD5E0',
  
  // Status - Success (Green)
  success: '#10B981',
  successLight: '#D1FAE5',
  onSuccess: '#065F46',
  
  // Status - Warning (Amber)
  warning: '#F59E0B',
  warningLight: '#FEF3C7',
  onWarning: '#92400E',
  
  // Status - Error (Red)
  error: '#EF4444',
  errorLight: '#FEE2E2',
  onError: '#991B1B',
  
  // Status - Info (Blue)
  info: '#3B82F6',
  infoLight: '#DBEAFE',
  onInfo: '#1E40AF',
  
  // Accent
  accent: '#8B5CF6',
  accentLight: '#EDE9FE',
  
  // Gradient
  gradient: {
    primary: 'linear-gradient(135deg, #FF6B35 0%, #FF8F66 100%)',
    secondary: 'linear-gradient(135deg, #1E3A5F 0%, #2E5A8F 100%)',
    success: 'linear-gradient(135deg, #10B981 0%, #34D399 100%)',
  },
  
  // Shadows
  shadow: {
    sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
    md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
    lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
    xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
  },
  
  // Overlay
  overlay: 'rgba(0, 0, 0, 0.4)',
  scrim: 'rgba(0, 0, 0, 0.6)',
};

/** Dark theme colors */
const darkColors: ColorScheme = {
  // Primary
  primary: '#FF8F66',
  primaryLight: '#FFB899',
  primaryDark: '#FF6B35',
  onPrimary: '#1A1A2E',
  
  // Secondary
  secondary: '#60A5FA',
  secondaryLight: '#93C5FD',
  secondaryDark: '#3B82F6',
  onSecondary: '#1A1A2E',
  
  // Background
  background: '#0D1117',
  backgroundSecondary: '#161B22',
  backgroundTertiary: '#21262D',
  surface: '#1C2128',
  surfaceVariant: '#252C35',
  onBackground: '#E6EDF3',
  onSurface: '#E6EDF3',
  
  // Text
  textPrimary: '#E6EDF3',
  textSecondary: '#8B949E',
  textTertiary: '#6E7681',
  textDisabled: '#484F58',
  textInverse: '#0D1117',
  
  // Border
  border: '#30363D',
  borderLight: '#21262D',
  borderDark: '#484F58',
  
  // Status - Success (Green)
  success: '#34D399',
  successLight: '#064E3B',
  onSuccess: '#D1FAE5',
  
  // Status - Warning (Amber)
  warning: '#FBBF24',
  warningLight: '#78350F',
  onWarning: '#FEF3C7',
  
  // Status - Error (Red)
  error: '#F87171',
  errorLight: '#7F1D1D',
  onSuccess: '#FEE2E2',
  
  // Status - Info (Blue)
  info: '#60A5FA',
  infoLight: '#1E3A8A',
  onInfo: '#DBEAFE',
  
  // Accent
  accent: '#A78BFA',
  accentLight: '#4C1D95',
  
  // Gradient
  gradient: {
    primary: 'linear-gradient(135deg, #FF8F66 0%, #FFB899 100%)',
    secondary: 'linear-gradient(135deg, #60A5FA 0%, #93C5FD 100%)',
    success: 'linear-gradient(135deg, #34D399 0%, #6EE7B7 100%)',
  },
  
  // Shadows
  shadow: {
    sm: '0 1px 2px 0 rgba(0, 0, 0, 0.3)',
    md: '0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -1px rgba(0, 0, 0, 0.3)',
    lg: '0 10px 15px -3px rgba(0, 0, 0, 0.4), 0 4px 6px -2px rgba(0, 0, 0, 0.3)',
    xl: '0 20px 25px -5px rgba(0, 0, 0, 0.5), 0 10px 10px -5px rgba(0, 0, 0, 0.3)',
  },
  
  // Overlay
  overlay: 'rgba(0, 0, 0, 0.7)',
  scrim: 'rgba(0, 0, 0, 0.8)',
};

/** Typography */
const typography: Typography = {
  fontFamily: {
    primary: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    secondary: "'SF Pro Display', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
    mono: "'SF Mono', 'Fira Code', 'JetBrains Mono', Consolas, monospace",
  },
  
  fontSize: {
    xs: '0.75rem',      // 12px
    sm: '0.875rem',     // 14px
    base: '1rem',       // 16px
    lg: '1.125rem',     // 18px
    xl: '1.25rem',      // 20px
    '2xl': '1.5rem',    // 24px
    '3xl': '1.875rem',  // 30px
    '4xl': '2.25rem',   // 36px
    '5xl': '3rem',      // 48px
  },
  
  fontWeight: {
    light: 300,
    normal: 400,
    medium: 500,
    semibold: 600,
    bold: 700,
  },
  
  lineHeight: {
    tight: 1.25,
    normal: 1.5,
    relaxed: 1.75,
  },
  
  letterSpacing: {
    tighter: '-0.05em',
    tight: '-0.025em',
    normal: '0',
    wide: '0.025em',
    wider: '0.05em',
  },
};

/** Spacing */
const spacing: Spacing = {
  0: '0',
  1: '0.25rem',   // 4px
  2: '0.5rem',    // 8px
  3: '0.75rem',   // 12px
  4: '1rem',      // 16px
  5: '1.25rem',   // 20px
  6: '1.5rem',    // 24px
  7: '1.75rem',   // 28px
  8: '2rem',      // 32px
  9: '2.25rem',   // 36px
  10: '2.5rem',   // 40px
  12: '3rem',     // 48px
  14: '3.5rem',   // 56px
  16: '4rem',     // 64px
  20: '5rem',     // 80px
  24: '6rem',     // 96px
  28: '7rem',     // 112px
  32: '8rem',     // 128px
  36: '9rem',     // 144px
  40: '10rem',    // 160px
  44: '11rem',    // 176px
  48: '12rem',    // 192px
  52: '13rem',    // 208px
  56: '14rem',    // 224px
  60: '15rem',    // 240px
  64: '16rem',    // 256px
  72: '18rem',    // 288px
  80: '20rem',    // 320px
  96: '24rem',    // 384px
};

/** Border radius */
const borderRadius: BorderRadius = {
  none: '0',
  sm: '0.125rem',  // 2px
  DEFAULT: '0.25rem', // 4px
  md: '0.375rem',  // 6px
  lg: '0.5rem',    // 8px
  xl: '0.75rem',   // 12px
  '2xl': '1rem',   // 16px
  '3xl': '1.5rem', // 24px
  full: '9999px',
};

/** Animation */
const animation: Animation = {
  duration: {
    instant: '0ms',
    fastest: '50ms',
    faster: '100ms',
    fast: '150ms',
    normal: '200ms',
    slow: '300ms',
    slower: '500ms',
    slowest: '1000ms',
  },
  
  easing: {
    linear: 'linear',
    easeIn: 'cubic-bezier(0.4, 0, 1, 1)',
    easeOut: 'cubic-bezier(0, 0, 0.2, 1)',
    easeInOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
    bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
  },
  
  transitions: {
    none: 'none',
    fast: 'all 100ms ease-out',
    normal: 'all 200ms ease-out',
    slow: 'all 300ms ease-out',
  },
};

// ============================================================================
// Theme Factory
// ============================================================================

/** Create a theme based on mode */
export function createTheme(mode: ThemeMode): Theme {
  const colors = mode === 'dark' ? darkColors : lightColors;
  
  return {
    mode,
    colors,
    typography,
    spacing,
    borderRadius,
    animation,
    breakpoints: {
      sm: '640px',
      md: '768px',
      lg: '1024px',
      xl: '1280px',
      '2xl': '1536px',
    },
    zIndex: {
      dropdown: 1000,
      sticky: 1020,
      modal: 1040,
      popover: 1060,
      tooltip: 1080,
      toast: 1100,
    },
  };
}

// ============================================================================
// Theme Hook (for React)
// ============================================================================

/**
 * React hook for theme management
 * 
 * @example
 * ```tsx
 * const { theme, colors, setTheme, toggleTheme } = useTheme();
 * 
 * return (
 *   <div style={{ backgroundColor: colors.background }}>
 *     <button onClick={toggleTheme}>Toggle Theme</button>
 *   </div>
 * );
 * ```
 */
export function useTheme() {
  // In actual implementation, this would use React Context
  // and sync with localStorage/AsyncStorage
  
  const [mode, setMode] = React.useState<ThemeMode>('system');
  const [theme, setTheme] = React.useState<Theme>(() => createTheme(mode));
  
  React.useEffect(() => {
    // Detect system preference
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = (e: MediaQueryListEvent | MediaQueryList) => {
      const newMode = e.matches ? 'dark' : 'light';
      if (mode === 'system') {
        setTheme(createTheme(newMode));
      }
    };
    
    mediaQuery.addEventListener('change', handleChange);
    handleChange(mediaQuery);
    
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [mode]);
  
  const setTheme = (newMode: ThemeMode) => {
    setMode(newMode);
    const effectiveMode = newMode === 'system' 
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : newMode;
    setTheme(createTheme(effectiveMode));
    
    // Persist preference
    localStorage.setItem('theme-mode', newMode);
  };
  
  const toggleTheme = () => {
    const newMode = theme.mode === 'light' ? 'dark' : 'light';
    setTheme(newMode);
  };
  
  return {
    theme,
    colors: theme.colors,
    typography: theme.typography,
    spacing: theme.spacing,
    borderRadius: theme.borderRadius,
    animation: theme.animation,
    mode,
    setTheme,
    toggleTheme,
  };
}

// ============================================================================
// CSS Variables Generation
// ============================================================================

/** Generate CSS variables from theme */
export function generateCSSVariables(theme: Theme): string {
  const { colors, typography, spacing, borderRadius, animation } = theme;
  
  return `
    :root {
      /* Primary Colors */
      --color-primary: ${colors.primary};
      --color-primary-light: ${colors.primaryLight};
      --color-primary-dark: ${colors.primaryDark};
      --color-on-primary: ${colors.onPrimary};
      
      /* Secondary Colors */
      --color-secondary: ${colors.secondary};
      --color-secondary-light: ${colors.secondaryLight};
      --color-secondary-dark: ${colors.secondaryDark};
      --color-on-secondary: ${colors.onSecondary};
      
      /* Background Colors */
      --color-background: ${colors.background};
      --color-background-secondary: ${colors.backgroundSecondary};
      --color-background-tertiary: ${colors.backgroundTertiary};
      --color-surface: ${colors.surface};
      --color-surface-variant: ${colors.surfaceVariant};
      --color-on-background: ${colors.onBackground};
      --color-on-surface: ${colors.onSurface};
      
      /* Text Colors */
      --color-text-primary: ${colors.textPrimary};
      --color-text-secondary: ${colors.textSecondary};
      --color-text-tertiary: ${colors.textTertiary};
      --color-text-disabled: ${colors.textDisabled};
      --color-text-inverse: ${colors.textInverse};
      
      /* Border Colors */
      --color-border: ${colors.border};
      --color-border-light: ${colors.borderLight};
      --color-border-dark: ${colors.borderDark};
      
      /* Status Colors */
      --color-success: ${colors.success};
      --color-success-light: ${colors.successLight};
      --color-on-success: ${colors.onSuccess};
      --color-warning: ${colors.warning};
      --color-warning-light: ${colors.warningLight};
      --color-on-warning: ${colors.onWarning};
      --color-error: ${colors.error};
      --color-error-light: ${colors.errorLight};
      --color-on-error: ${colors.onError};
      --color-info: ${colors.info};
      --color-info-light: ${colors.infoLight};
      --color-on-info: ${colors.onInfo};
      
      /* Accent Colors */
      --color-accent: ${colors.accent};
      --color-accent-light: ${colors.accentLight};
      
      /* Gradients */
      --gradient-primary: ${colors.gradient.primary};
      --gradient-secondary: ${colors.gradient.secondary};
      --gradient-success: ${colors.gradient.success};
      
      /* Shadows */
      --shadow-sm: ${colors.shadow.sm};
      --shadow-md: ${colors.shadow.md};
      --shadow-lg: ${colors.shadow.lg};
      --shadow-xl: ${colors.shadow.xl};
      
      /* Overlay */
      --color-overlay: ${colors.overlay};
      --color-scrim: ${colors.scrim};
      
      /* Typography */
      --font-family-primary: ${typography.fontFamily.primary};
      --font-family-secondary: ${typography.fontFamily.secondary};
      --font-family-mono: ${typography.fontFamily.mono};
      
      --font-size-xs: ${typography.fontSize.xs};
      --font-size-sm: ${typography.fontSize.sm};
      --font-size-base: ${typography.fontSize.base};
      --font-size-lg: ${typography.fontSize.lg};
      --font-size-xl: ${typography.fontSize.xl};
      --font-size-2xl: ${typography.fontSize['2xl']};
      --font-size-3xl: ${typography.fontSize['3xl']};
      --font-size-4xl: ${typography.fontSize['4xl']};
      --font-size-5xl: ${typography.fontSize['5xl']};
      
      --font-weight-light: ${typography.fontWeight.light};
      --font-weight-normal: ${typography.fontWeight.normal};
      --font-weight-medium: ${typography.fontWeight.medium};
      --font-weight-semibold: ${typography.fontWeight.semibold};
      --font-weight-bold: ${typography.fontWeight.bold};
      
      --line-height-tight: ${typography.lineHeight.tight};
      --line-height-normal: ${typography.lineHeight.normal};
      --line-height-relaxed: ${typography.lineHeight.relaxed};
      
      --letter-spacing-tighter: ${typography.letterSpacing.tighter};
      --letter-spacing-tight: ${typography.letterSpacing.tight};
      --letter-spacing-normal: ${typography.letterSpacing.normal};
      --letter-spacing-wide: ${typography.letterSpacing.wide};
      --letter-spacing-wider: ${typography.letterSpacing.wider};
      
      /* Spacing */
      --spacing-0: ${spacing[0]};
      --spacing-1: ${spacing[1]};
      --spacing-2: ${spacing[2]};
      --spacing-3: ${spacing[3]};
      --spacing-4: ${spacing[4]};
      --spacing-5: ${spacing[5]};
      --spacing-6: ${spacing[6]};
      --spacing-7: ${spacing[7]};
      --spacing-8: ${spacing[8]};
      --spacing-9: ${spacing[9]};
      --spacing-10: ${spacing[10]};
      --spacing-12: ${spacing[12]};
      --spacing-14: ${spacing[14]};
      --spacing-16: ${spacing[16]};
      --spacing-20: ${spacing[20]};
      --spacing-24: ${spacing[24]};
      
      /* Border Radius */
      --radius-none: ${borderRadius.none};
      --radius-sm: ${borderRadius.sm};
      --radius: ${borderRadius.DEFAULT};
      --radius-md: ${borderRadius.md};
      --radius-lg: ${borderRadius.lg};
      --radius-xl: ${borderRadius.xl};
      --radius-2xl: ${borderRadius['2xl']};
      --radius-3xl: ${borderRadius['3xl']};
      --radius-full: ${borderRadius.full};
      
      /* Animation */
      --duration-instant: ${animation.duration.instant};
      --duration-fastest: ${animation.duration.fastest};
      --duration-faster: ${animation.duration.faster};
      --duration-fast: ${animation.duration.fast};
      --duration-normal: ${animation.duration.normal};
      --duration-slow: ${animation.duration.slow};
      --duration-slower: ${animation.duration.slower};
      --duration-slowest: ${animation.duration.slowest};
      
      --ease-linear: ${animation.easing.linear};
      --ease-in: ${animation.easing.easeIn};
      --ease-out: ${animation.easing.easeOut};
      --ease-in-out: ${animation.easing.easeInOut};
      --ease-bounce: ${animation.easing.bounce};
      
      --transition-none: ${animation.transitions.none};
      --transition-fast: ${animation.transitions.fast};
      --transition-normal: ${animation.transitions.normal};
      --transition-slow: ${animation.transitions.slow};
    }
  `;
}

// Export default theme
export const lightTheme = createTheme('light');
export const darkTheme = createTheme('dark');
