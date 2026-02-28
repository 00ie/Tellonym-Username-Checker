import React from 'react'
import { useI18n } from '../../i18n'

export const About: React.FC = () => {
  const { t } = useI18n()

  const links = [
    { label: t('about.github'), value: '00ie', href: 'https://github.com/00ie' },
    { label: t('about.telegram'), value: 'feicoes', href: 'https://t.me/feicoes' },
    { label: t('about.discord'), value: 'tlwm', href: 'https://discord.gg/2asv4rEhGh' },
    { label: t('about.server'), value: 'discord.gg/2asv4rEhGh', href: 'https://discord.gg/2asv4rEhGh' },
    { label: t('about.site'), value: '00ie.github.io', href: 'https://00ie.github.io/' },
    { label: t('about.site'), value: 'cutz.lol/gon', href: 'https://cutz.lol/gon' },
  ]

  return (
    <div className="space-y-6">
      <div className="neon-panel rounded-2xl border border-red-500/35 bg-zinc-950/95 p-6 shadow-[0_0_35px_rgba(239,68,68,0.14)]">
        <h2 className="text-2xl font-semibold text-red-200">{t('about.title')}</h2>
        <p className="mt-2 max-w-3xl text-sm text-zinc-300">{t('about.description')}</p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {links.map((item) => (
          <a
            key={`${item.label}-${item.value}`}
            href={item.href}
            target="_blank"
            rel="noreferrer"
            className="neon-orbit group rounded-xl border border-zinc-800 bg-zinc-950/95 p-4 transition hover:border-red-500/80 hover:bg-zinc-900"
          >
            <p className="text-xs uppercase tracking-[0.2em] text-zinc-400">{item.label}</p>
            <p className="mt-2 font-medium text-red-300 group-hover:text-red-200">{item.value}</p>
          </a>
        ))}
      </div>
    </div>
  )
}
