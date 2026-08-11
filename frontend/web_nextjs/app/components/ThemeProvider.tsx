'use client'

import { createContext, useContext, useEffect, useState, useCallback, ReactNode } from 'react'

type ThemeMode = 'light' | 'dark' | 'system'

interface ThemeColors {
  bgPrimary: string;
  bgSecondary: string;
  bgTertiary: string;
  bgCard: string;
  textPrimary: string;
  textSecondary: string;
  border: string;
  accent: string;
  success: string;
  error: string;
  warning: string;
  overlay: string;
}

interface ThemeContextType {
  theme: 'light' | 'dark'
  themeMode: ThemeMode
  toggleTheme: () => void
  setTheme: (theme: 'light' | 'dark') => void
  setThemeMode: (mode: ThemeMode) => void
  isDark: boolean
  colors: ThemeColors
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

const THEME_MODE_KEY = 'tigerwallet_theme_mode'

const LIGHT_COLORS: ThemeColors = {
  bgPrimary: '#ffffff',
  bgSecondary: '#f7f8fa',
  bgTertiary: '#eef0f4',
  bgCard: '#ffffff',
  textPrimary: '#0f172a',
  textSecondary: '#64748b',
  border: '#e2e8f0',
  accent: '#f97316',
  success: '#16a34a',
  error: '#dc2626',
  warning: '#d97706',
  overlay: 'rgba(15, 23, 42, 0.4)',
}

const DARK_COLORS: ThemeColors = {
  bgPrimary: '#0b0e14',
  bgSecondary: '#131722',
  bgTertiary: '#1b2030',
  bgCard: '#131722',
  textPrimary: '#e5e7eb',
  textSecondary: '#9ca3af',
  border: '#272d3a',
  accent: '#f97316',
  success: '#22c55e',
  error: '#ef4444',
  warning: '#f59e0b',
  overlay: 'rgba(0, 0, 0, 0.6)',
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeMode, setThemeModeState] = useState<ThemeMode>('system')
  const [systemPreference, setSystemPreference] = useState<'light' | 'dark'>('dark')
  const [theme, setThemeState] = useState<'light' | 'dark'>('dark')
  const [mounted, setMounted] = useState(false)

  // Get system preference
  useEffect(() => {
    if (typeof window === 'undefined') return
    
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    setSystemPreference(mediaQuery.matches ? 'dark' : 'light')

    const handler = (e: MediaQueryListEvent) => {
      setSystemPreference(e.matches ? 'dark' : 'light')
    }

    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }, [])

  // Initialize theme on mount
  useEffect(() => {
    const stored = localStorage.getItem(THEME_MODE_KEY) as ThemeMode
    if (stored && (stored === 'light' || stored === 'dark' || stored === 'system')) {
      setThemeModeState(stored)
    }
    setMounted(true)
  }, [])

  // Calculate effective theme
  useEffect(() => {
    if (!mounted) return
    
    let effectiveTheme: 'light' | 'dark'
    if (themeMode === 'system') {
      effectiveTheme = systemPreference
    } else {
      effectiveTheme = themeMode
    }
    
    setThemeState(effectiveTheme)

    // Apply to document
    localStorage.setItem(THEME_MODE_KEY, themeMode)
    document.documentElement.classList.remove('light', 'dark')
    document.documentElement.classList.add(effectiveTheme)
    document.documentElement.setAttribute('data-theme', effectiveTheme)

    // Inject the full theme palette as CSS custom properties on :root so that
    // every page — including ones that use plain CSS `var(--bg-primary)` — gets
    // the correct tokens for the active theme. This makes light/dark switching
    // work globally, not just in components that read the React context.
    const palette = effectiveTheme === 'dark' ? DARK_COLORS : LIGHT_COLORS
    const root = document.documentElement.style
    root.setProperty('--bg-primary', palette.bgPrimary)
    root.setProperty('--bg-secondary', palette.bgSecondary)
    root.setProperty('--bg-tertiary', palette.bgTertiary)
    root.setProperty('--bg-card', palette.bgCard)
    root.setProperty('--text-primary', palette.textPrimary)
    root.setProperty('--text-secondary', palette.textSecondary)
    root.setProperty('--border-color', palette.border)
    root.setProperty('--accent', palette.accent)
    root.setProperty('--success', palette.success)
    root.setProperty('--error', palette.error)
    root.setProperty('--warning', palette.warning)
    root.setProperty('--overlay', palette.overlay)
  }, [themeMode, systemPreference, mounted])

  const setThemeMode = useCallback((mode: ThemeMode) => {
    setThemeModeState(mode)
    localStorage.setItem(THEME_MODE_KEY, mode)
  }, [])

  const toggleTheme = useCallback(() => {
    const newTheme = theme === 'dark' ? 'light' : 'dark'
    setThemeModeState(newTheme)
    localStorage.setItem(THEME_MODE_KEY, newTheme)
  }, [theme])

  const setTheme = useCallback((newTheme: 'light' | 'dark') => {
    setThemeModeState(newTheme)
    localStorage.setItem(THEME_MODE_KEY, newTheme)
  }, [])

  return (
    <ThemeContext.Provider value={{ 
      theme, 
      themeMode,
      toggleTheme, 
      setTheme, 
      setThemeMode,
      isDark: theme === 'dark',
      colors: theme === 'dark' ? DARK_COLORS : LIGHT_COLORS
    }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider')
  }
  return context
}

// Hook for components that need theme info but can't throw
export function useThemeSafe() {
  return useContext(ThemeContext)
}