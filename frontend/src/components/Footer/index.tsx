import React from 'react'
import { useI18n } from '../../i18n'

export const Footer: React.FC = () => {
  const { t } = useI18n()

  return (
    <footer className="border-t border-red-500/40 bg-black/90">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-2 px-4 py-4 text-sm text-zinc-300 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
        <span>{t('footer.project')}</span>
        <div className="flex items-center gap-4">
          <a
            href="https://github.com/00ie"
            target="_blank"
            rel="noreferrer"
            className="neon-orbit-link text-red-300 transition hover:text-red-200"
          >
            github.com/00ie
          </a>
          <a
            href="https://discord.gg/2asv4rEhGh"
            target="_blank"
            rel="noreferrer"
            className="neon-orbit-link text-red-300 transition hover:text-red-200"
          >
            discord.gg/2asv4rEhGh
          </a>
        </div>
      </div>
    </footer>
  )
}
