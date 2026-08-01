'use client'

import { createContext, useContext, useEffect, useState, useCallback, ReactNode } from 'react'

type ThemeMode = 'light' | 'dark' | 'system'

interface ThemeContextType {
  theme: 'light' | 'dark'
  themeMode: ThemeMode
  toggleTheme: () => void
  setTheme: (theme: 'light' | 'dark') => void
  setThemeMode: (mode: ThemeMode) => void
  isDark: boolean
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

const THEME_MODE_KEY = 'tigerwallet_theme_mode'

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
      isDark: theme === 'dark' 
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