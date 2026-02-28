import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

export type AccentTheme = 'red' | 'blue' | 'green' | 'purple'

interface ThemeContextValue {
  theme: AccentTheme
  setTheme: (theme: AccentTheme) => void
}

const storageKey = 'tellonym.theme'

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)

function normalizeTheme(value: string | null): AccentTheme {
  if (value === 'blue' || value === 'green' || value === 'purple' || value === 'red') {
    return value
  }
  return 'red'
}

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [theme, setThemeState] = useState<AccentTheme>(() => normalizeTheme(localStorage.getItem(storageKey)))

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem(storageKey, theme)
  }, [theme])

  const setTheme = useCallback((nextTheme: AccentTheme) => {
    setThemeState(nextTheme)
  }, [])

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      setTheme,
    }),
    [setTheme, theme],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider')
  }
  return context
}
