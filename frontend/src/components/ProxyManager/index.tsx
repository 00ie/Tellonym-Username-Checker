import React, { useState } from 'react'
import {
  ArrowPathIcon,
  CheckBadgeIcon,
  CheckCircleIcon,
  PlusIcon,
  TrashIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { useStore } from '../../store'
import { useI18n } from '../../i18n'

export const ProxyManager: React.FC = () => {
  const { proxies, proxyStats, addProxies, removeProxy, removeBadProxies, testProxy, checkAllProxies } = useStore()
  const { t } = useI18n()

  const [newProxies, setNewProxies] = useState('')
  const [testing, setTesting] = useState<string | null>(null)
  const [checkingAll, setCheckingAll] = useState(false)
  const [cleaning, setCleaning] = useState(false)

  const handleAddProxies = async () => {
    const list = newProxies
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line.length > 0)

    if (list.length === 0) {
      toast.error(t('proxy.toast.enterOne'))
      return
    }

    try {
      await addProxies(list)
      setNewProxies('')
      toast.success(t('proxy.toast.added', { count: list.length }))
    } catch {
      toast.error(t('proxy.toast.addFailed'))
    }
  }

  const handleTestProxy = async (url: string) => {
    setTesting(url)
    try {
      const result = await testProxy(url)
      if (result.success) {
        toast.success(t('proxy.toast.ok', { latency: result.latency }))
      } else {
        toast.error(result.error || t('proxy.toast.failed'))
      }
    } catch {
      toast.error(t('proxy.toast.testFailed'))
    } finally {
      setTesting(null)
    }
  }

  const handleCheckAll = async () => {
    setCheckingAll(true)
    try {
      const result = await checkAllProxies()
      toast.success(
        t('proxy.toast.batchResult', {
          checked: result.checked,
          healthy: result.healthy,
          dead: result.dead,
          duration: result.durationMs,
        }),
      )
    } catch {
      toast.error(t('proxy.toast.batchFailed'))
    } finally {
      setCheckingAll(false)
    }
  }

  const handleRemoveProxy = async (url: string) => {
    try {
      await removeProxy(url)
      toast.success(t('proxy.toast.removed'))
    } catch {
      toast.error(t('proxy.toast.removeFailed'))
    }
  }

  const handleRemoveBadProxies = async () => {
    setCleaning(true)
    try {
      const result = await removeBadProxies()
      toast.success(t('proxy.toast.badRemoved', { removed: result.removed }))
    } catch {
      toast.error(t('proxy.toast.badRemoveFailed'))
    } finally {
      setCleaning(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="neon-panel flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-red-500/35 bg-gradient-to-r from-black via-zinc-950 to-red-950/60 p-4 shadow-[0_0_40px_rgba(239,68,68,0.2)]">
        <h2 className="text-2xl font-semibold text-red-100">{t('proxy.title')}</h2>
        <div className="text-sm text-zinc-300">
          {t('proxy.total')} <span className="font-semibold text-zinc-100">{proxyStats.total}</span>
          <span className="mx-2 text-zinc-600">|</span>
          {t('proxy.healthy')} <span className="font-semibold text-emerald-400">{proxyStats.healthy}</span>
          <span className="mx-2 text-zinc-600">|</span>
          {t('proxy.dead')} <span className="font-semibold text-red-400">{proxyStats.dead}</span>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <h3 className="mb-3 text-lg font-semibold text-red-100">{t('proxy.addSection')}</h3>
        <textarea
          value={newProxies}
          onChange={(e) => setNewProxies(e.target.value)}
          placeholder={t('proxy.placeholder')}
          className="h-32 w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 font-mono text-sm text-zinc-200 outline-none transition focus:border-red-500"
        />
        <div className="mt-3 flex flex-wrap gap-2">
          <button
            onClick={handleAddProxies}
            className="inline-flex items-center gap-2 rounded-lg border border-red-500/60 bg-red-700/90 px-4 py-2 font-medium text-white transition hover:bg-red-600"
          >
            <PlusIcon className="h-5 w-5" />
            {t('proxy.addButton')}
          </button>
          <button
            onClick={handleCheckAll}
            disabled={checkingAll || proxies.length === 0}
            className="inline-flex items-center gap-2 rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-2 font-medium text-zinc-200 transition hover:border-red-500 hover:text-red-200 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <CheckBadgeIcon className="h-5 w-5" />
            {checkingAll ? t('proxy.checking') : t('proxy.checkAll')}
          </button>
          <button
            onClick={handleRemoveBadProxies}
            disabled={cleaning || proxies.length === 0}
            className="inline-flex items-center gap-2 rounded-lg border border-red-500/40 bg-black px-4 py-2 font-medium text-red-200 transition hover:bg-zinc-900 disabled:cursor-not-allowed disabled:opacity-60"
          >
            <TrashIcon className="h-5 w-5" />
            {cleaning ? t('proxy.removeBadLoading') : t('proxy.removeBad')}
          </button>
        </div>
      </div>

      <div className="neon-panel overflow-hidden rounded-2xl border border-zinc-800 bg-zinc-950/95">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[880px]">
            <thead className="bg-black/80">
              <tr>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.status')}</th>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.url')}</th>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.type')}</th>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.successFail')}</th>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.avg')}</th>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.lastCheck')}</th>
                <th className="px-4 py-3 text-left text-xs uppercase tracking-wider text-zinc-400">{t('proxy.table.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {proxies.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-sm text-zinc-500">
                    {t('proxy.none')}
                  </td>
                </tr>
              )}
              {proxies.map((proxy) => (
                <tr key={proxy.url} className="border-t border-zinc-900 hover:bg-zinc-900/50">
                  <td className="px-4 py-3">
                    {proxy.isAlive ? (
                      <CheckCircleIcon className="h-5 w-5 text-emerald-400" />
                    ) : (
                      <XCircleIcon className="h-5 w-5 text-red-400" />
                    )}
                  </td>
                  <td className="px-4 py-3 font-mono text-sm text-zinc-200">{proxy.url}</td>
                  <td className="px-4 py-3 text-sm text-zinc-300">{proxy.type}</td>
                  <td className="px-4 py-3 text-sm text-zinc-300">
                    <span className="text-emerald-400">{proxy.successCount}</span>
                    <span className="mx-1 text-zinc-600">/</span>
                    <span className="text-red-400">{proxy.failCount}</span>
                  </td>
                  <td className="px-4 py-3 text-sm text-zinc-300">{proxy.avgResponseMs ? `${proxy.avgResponseMs}ms` : '-'}</td>
                  <td className="px-4 py-3 text-sm text-zinc-400">
                    {proxy.lastCheck ? new Date(proxy.lastCheck).toLocaleString() : t('common.never')}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleTestProxy(proxy.url)}
                        disabled={testing === proxy.url}
                        className="rounded p-1 text-zinc-300 transition hover:bg-zinc-800 hover:text-red-300 disabled:opacity-60"
                      >
                        <ArrowPathIcon className={`h-4 w-4 ${testing === proxy.url ? 'animate-spin' : ''}`} />
                      </button>
                      <button
                        onClick={() => handleRemoveProxy(proxy.url)}
                        className="rounded p-1 text-red-400 transition hover:bg-zinc-800"
                      >
                        <TrashIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
