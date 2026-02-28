import type { AccentTheme } from './index'

export type AccentPalette = {
  border: string
  strong: string
  fill: string
}

export const accentPalette: Record<AccentTheme, AccentPalette> = {
  red: {
    border: '#ef4444',
    strong: '#dc2626',
    fill: 'rgba(239, 68, 68, 0.16)',
  },
  blue: {
    border: '#3b82f6',
    strong: '#2563eb',
    fill: 'rgba(59, 130, 246, 0.16)',
  },
  green: {
    border: '#22c55e',
    strong: '#16a34a',
    fill: 'rgba(34, 197, 94, 0.16)',
  },
  purple: {
    border: '#a855f7',
    strong: '#9333ea',
    fill: 'rgba(168, 85, 247, 0.16)',
  },
}
