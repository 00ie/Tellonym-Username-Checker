import React from 'react'
import { useI18n } from '../../i18n'

interface NavbarProps {
  activeTab: string
  onTabChange: (tab: string) => void
}

export const Navbar: React.FC<NavbarProps> = ({ activeTab, onTabChange }) => {
  const { t } = useI18n()
  const tabs = [
    { id: 'dashboard', label: t('nav.dashboard') },
    { id: 'checker', label: t('nav.checker') },
    { id: 'proxies', label: t('nav.proxies') },
    { id: 'statistics', label: t('nav.statistics') },
    { id: 'settings', label: t('nav.settings') },
    { id: 'about', label: t('nav.about') },
  ]

  return (
    <header className="sticky top-0 z-30 border-b border-red-500/40 bg-black/90 shadow-[0_0_22px_rgba(239,68,68,0.2)] backdrop-blur-md">
      <div className="mx-auto flex w-full max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
        <h1 className="text-xl font-semibold tracking-wide text-red-200">{t('app.name')}</h1>
        <nav className="flex flex-wrap gap-2">
          {tabs.map((tab) => {
            const active = activeTab === tab.id
            return (
              <button
                key={tab.id}
                onClick={() => onTabChange(tab.id)}
                className={`rounded-lg border px-3 py-2 text-sm transition ${
                  active
                    ? 'border-red-400/60 bg-red-700/20 text-red-100'
                    : 'border-zinc-800 bg-zinc-950/80 text-zinc-300 hover:border-red-700 hover:bg-zinc-900'
                }`}
              >
                {tab.label}
              </button>
            )
          })}
        </nav>
      </div>
    </header>
  )
}
