import React, { useMemo } from 'react'
import type { ProxyStats } from '../../types/api'
import { useI18n } from '../../i18n'

interface WarningStats {
  attempts: number
  errors: number
  rateLimited: number
  avgResponse: number
  rate: number
}

interface RateLimitWarningProps {
  open: boolean
  stats: WarningStats
  proxyStats: ProxyStats
  onClose: () => void
  onOpenProxies: () => void
  onOpenChecker: () => void
}

export const RateLimitWarning: React.FC<RateLimitWarningProps> = ({
  open,
  stats,
  proxyStats,
  onClose,
  onOpenProxies,
  onOpenChecker,
}) => {
  const { t } = useI18n()

  const errorRate = useMemo(() => {
    if (stats.attempts <= 0) {
      return 0
    }
    return (stats.errors / stats.attempts) * 100
  }, [stats.attempts, stats.errors])

  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 backdrop-blur-sm">
      <div className="neon-panel w-full max-w-2xl rounded-2xl border border-red-500/45 bg-zinc-950 p-6">
        <h3 className="text-2xl font-semibold text-red-200">{t('alert.title')}</h3>
        <p className="mt-2 text-sm text-zinc-300">{t('alert.subtitle')}</p>

        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div className="rounded-lg border border-zinc-800 bg-black/70 p-3">
            <p className="text-xs uppercase tracking-wider text-zinc-400">{t('alert.rateLimited')}</p>
            <p className="mt-1 text-xl font-semibold text-red-300">{stats.rateLimited}</p>
          </div>
          <div className="rounded-lg border border-zinc-800 bg-black/70 p-3">
            <p className="text-xs uppercase tracking-wider text-zinc-400">{t('alert.errorRate')}</p>
            <p className="mt-1 text-xl font-semibold text-red-300">{errorRate.toFixed(2)}%</p>
          </div>
          <div className="rounded-lg border border-zinc-800 bg-black/70 p-3">
            <p className="text-xs uppercase tracking-wider text-zinc-400">{t('alert.avgResponse')}</p>
            <p className="mt-1 text-xl font-semibold text-red-300">{stats.avgResponse}ms</p>
          </div>
        </div>

        <div className="mt-4 rounded-lg border border-zinc-800 bg-black/70 p-4 text-sm text-zinc-300">
          <p>{t('alert.tip1')}</p>
          <p className="mt-2">{t('alert.tip2', { healthy: proxyStats.healthy, total: proxyStats.total })}</p>
          <p className="mt-2">{t('alert.tip3')}</p>
          <p className="mt-2">{t('alert.tip4')}</p>
        </div>

        <div className="mt-5 flex flex-wrap gap-3">
          <button
            onClick={onOpenProxies}
            className="rounded-lg border border-red-500/60 bg-red-700/90 px-4 py-2 font-medium text-white transition hover:bg-red-600"
          >
            {t('alert.goProxies')}
          </button>
          <button
            onClick={onOpenChecker}
            className="rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-2 font-medium text-zinc-200 transition hover:border-red-500 hover:text-red-200"
          >
            {t('alert.goChecker')}
          </button>
          <button
            onClick={onClose}
            className="rounded-lg border border-zinc-700 bg-black px-4 py-2 font-medium text-zinc-300 transition hover:bg-zinc-900"
          >
            {t('alert.dismiss')}
          </button>
        </div>
      </div>
    </div>
  )
}
