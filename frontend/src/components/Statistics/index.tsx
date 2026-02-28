import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'react-chartjs-2'
import { GetHistoricalStats } from '../../services/backend'
import type { HistoricalStats } from '../../types/api'
import { useStore } from '../../store'
import { useI18n } from '../../i18n'
import { useTheme } from '../../theme'
import { accentPalette } from '../../theme/palette'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)

type RangePreset = '1h' | '24h' | '7d' | '30d' | 'today' | 'custom'
type Granularity = 'auto' | 'minute' | 'hour' | 'day'

type NormalizedPoint = {
  timestamp: number
  attempts: number
  found: number
  errors: number
  rate: number
}

function toDateTimeLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

function resolveRange(preset: RangePreset, customFrom: string, customTo: string): { from: Date; to: Date } {
  const now = new Date()
  const from = new Date(now)

  if (preset === 'custom') {
    const parsedFrom = customFrom ? new Date(customFrom) : new Date(now.getTime() - 60*60*1000)
    const parsedTo = customTo ? new Date(customTo) : now
    const validFrom = Number.isNaN(parsedFrom.getTime()) ? new Date(now.getTime() - 60*60*1000) : parsedFrom
    const validTo = Number.isNaN(parsedTo.getTime()) ? now : parsedTo
    if (validFrom <= validTo) {
      return { from: validFrom, to: validTo }
    }
    return { from: validTo, to: validFrom }
  }

  switch (preset) {
    case '1h':
      from.setHours(from.getHours() - 1)
      break
    case '24h':
      from.setDate(from.getDate() - 1)
      break
    case '7d':
      from.setDate(from.getDate() - 7)
      break
    case '30d':
      from.setDate(from.getDate() - 30)
      break
    case 'today':
      from.setHours(0, 0, 0, 0)
      break
    default:
      from.setHours(from.getHours() - 1)
      break
  }

  return { from, to: now }
}

function resolveBucketMs(granularity: Granularity, rangeMs: number): number {
  if (granularity === 'minute') {
    return 60 * 1000
  }
  if (granularity === 'hour') {
    return 60 * 60 * 1000
  }
  if (granularity === 'day') {
    return 24 * 60 * 60 * 1000
  }
  if (rangeMs <= 3*60*60*1000) {
    return 60 * 1000
  }
  if (rangeMs <= 3*24*60*60*1000) {
    return 60 * 60 * 1000
  }
  return 24 * 60 * 60 * 1000
}

function aggregatePoints(rows: HistoricalStats[], bucketMs: number): NormalizedPoint[] {
  const buckets = new Map<number, { point: NormalizedPoint; rateSum: number; count: number }>()

  for (const row of rows) {
    const timestamp = new Date(row.timestamp).getTime()
    if (Number.isNaN(timestamp)) {
      continue
    }

    const bucket = Math.floor(timestamp / bucketMs) * bucketMs
    const existing = buckets.get(bucket)
    if (!existing) {
      buckets.set(bucket, {
        point: {
          timestamp,
          attempts: row.attempts,
          found: row.found,
          errors: row.errors,
          rate: row.rate,
        },
        rateSum: row.rate,
        count: 1,
      })
      continue
    }

    if (timestamp >= existing.point.timestamp) {
      existing.point.timestamp = timestamp
      existing.point.attempts = row.attempts
      existing.point.found = row.found
      existing.point.errors = row.errors
    }

    existing.rateSum += row.rate
    existing.count += 1
  }

  const points = Array.from(buckets.values())
    .map((entry) => ({
      ...entry.point,
      rate: entry.count > 0 ? entry.rateSum / entry.count : entry.point.rate,
    }))
    .sort((a, b) => a.timestamp - b.timestamp)

  return points
}

interface StatisticsProps {
  active?: boolean
}

export const Statistics: React.FC<StatisticsProps> = ({ active = true }) => {
  const { stats } = useStore()
  const { t } = useI18n()
  const { theme } = useTheme()
  const palette = accentPalette[theme]

  const now = new Date()
  const [preset, setPreset] = useState<RangePreset>('1h')
  const [granularity, setGranularity] = useState<Granularity>('auto')
  const [customFrom, setCustomFrom] = useState(toDateTimeLocal(new Date(now.getTime() - 60*60*1000)))
  const [customTo, setCustomTo] = useState(toDateTimeLocal(now))
  const [loading, setLoading] = useState(false)
  const [historicalRows, setHistoricalRows] = useState<HistoricalStats[]>([])

  const activeRange = useMemo(() => resolveRange(preset, customFrom, customTo), [customFrom, customTo, preset])
  const rangeMs = activeRange.to.getTime() - activeRange.from.getTime()
  const bucketMs = useMemo(() => resolveBucketMs(granularity, rangeMs), [granularity, rangeMs])

  const refreshHistory = useCallback(async () => {
    if (!active) {
      return
    }
    setLoading(true)
    try {
      const rows = await GetHistoricalStats(activeRange.from.toISOString(), activeRange.to.toISOString())
      setHistoricalRows(rows)
    } finally {
      setLoading(false)
    }
  }, [active, activeRange.from, activeRange.to])

  useEffect(() => {
    if (!active) {
      return
    }
    void refreshHistory()
  }, [active, refreshHistory])

  useEffect(() => {
    if (!active) {
      return
    }
    const interval = window.setInterval(() => {
      void refreshHistory()
    }, 10000)

    return () => {
      clearInterval(interval)
    }
  }, [active, refreshHistory])

  const aggregated = useMemo(() => aggregatePoints(historicalRows, bucketMs), [bucketMs, historicalRows])

  const chartData = useMemo(
    () => ({
      labels: aggregated.map((point) => {
        const date = new Date(point.timestamp)
        if (bucketMs >= 24*60*60*1000) {
          return date.toLocaleDateString()
        }
        if (bucketMs >= 60*60*1000) {
          return date.toLocaleString([], { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
        }
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      }),
      datasets: [
        {
          label: t('stats.rate'),
          data: aggregated.map((point) => point.rate),
          borderColor: palette.border,
          backgroundColor: palette.fill,
          fill: true,
          tension: 0.35,
        },
      ],
    }),
    [aggregated, bucketMs, palette.border, palette.fill, t],
  )

  const chartOptions: ChartOptions<'line'> = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        labels: { color: '#d4d4d8' },
      },
    },
    scales: {
      x: {
        ticks: { color: '#a1a1aa' },
        grid: { color: 'rgba(63, 63, 70, 0.3)' },
      },
      y: {
        ticks: { color: '#a1a1aa' },
        grid: { color: 'rgba(63, 63, 70, 0.3)' },
      },
    },
  }

  const rangeTotals = useMemo(() => {
    if (aggregated.length === 0) {
      return { attempts: 0, found: 0, errors: 0, avgRate: 0 }
    }

    const first = aggregated[0]
    const last = aggregated[aggregated.length - 1]
    const totalRate = aggregated.reduce((sum, point) => sum + point.rate, 0)
    return {
      attempts: Math.max(0, last.attempts - first.attempts),
      found: Math.max(0, last.found - first.found),
      errors: Math.max(0, last.errors - first.errors),
      avgRate: aggregated.length > 0 ? totalRate / aggregated.length : 0,
    }
  }, [aggregated])

  return (
    <div className="space-y-6">
      <div className="neon-panel rounded-2xl border border-red-500/35 bg-zinc-950/95 p-4 shadow-[0_0_25px_rgba(239,68,68,0.16)]">
        <h2 className="text-2xl font-semibold text-red-100">{t('stats.title')}</h2>
        <p className="mt-1 text-sm text-zinc-400">{t('stats.subtitle')}</p>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-4">
          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('stats.rangePreset')}</span>
            <select
              value={preset}
              onChange={(e) => setPreset(e.target.value as RangePreset)}
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
            >
              <option value="1h">{t('stats.range.1h')}</option>
              <option value="24h">{t('stats.range.24h')}</option>
              <option value="7d">{t('stats.range.7d')}</option>
              <option value="30d">{t('stats.range.30d')}</option>
              <option value="today">{t('stats.range.today')}</option>
              <option value="custom">{t('stats.range.custom')}</option>
            </select>
          </label>

          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('stats.granularity')}</span>
            <select
              value={granularity}
              onChange={(e) => setGranularity(e.target.value as Granularity)}
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
            >
              <option value="auto">{t('stats.granularity.auto')}</option>
              <option value="minute">{t('stats.granularity.minute')}</option>
              <option value="hour">{t('stats.granularity.hour')}</option>
              <option value="day">{t('stats.granularity.day')}</option>
            </select>
          </label>

          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('stats.from')}</span>
            <input
              type="datetime-local"
              value={customFrom}
              onChange={(e) => {
                setCustomFrom(e.target.value)
                setPreset('custom')
              }}
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
            />
          </label>

          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('stats.to')}</span>
            <input
              type="datetime-local"
              value={customTo}
              onChange={(e) => {
                setCustomTo(e.target.value)
                setPreset('custom')
              }}
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
            />
          </label>
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button
            onClick={() => void refreshHistory()}
            className="rounded-lg border border-red-500/60 bg-red-700/90 px-4 py-2 font-medium text-white transition hover:bg-red-600"
          >
            {t('stats.refresh')}
          </button>
          <span className="text-sm text-zinc-400">
            {t('stats.currentWindow', {
              from: activeRange.from.toLocaleString(),
              to: activeRange.to.toLocaleString(),
            })}
          </span>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <div className="h-72">
          <Line data={chartData} options={chartOptions} />
        </div>
        {loading && <p className="mt-3 text-sm text-zinc-500">{t('stats.loading')}</p>}
        {!loading && aggregated.length === 0 && <p className="mt-3 text-sm text-zinc-500">{t('stats.noData')}</p>}
        <p className="mt-2 text-xs text-zinc-500">{t('stats.retentionNote')}</p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <div className="neon-panel rounded-lg border border-zinc-800 bg-zinc-950/95 p-4">
          <p className="text-xs uppercase tracking-wide text-zinc-400">{t('stats.attempts')}</p>
          <p className="mt-1 text-2xl font-semibold">{rangeTotals.attempts}</p>
        </div>
        <div className="neon-panel rounded-lg border border-zinc-800 bg-zinc-950/95 p-4">
          <p className="text-xs uppercase tracking-wide text-zinc-400">{t('stats.found')}</p>
          <p className="mt-1 text-2xl font-semibold text-emerald-400">{rangeTotals.found}</p>
        </div>
        <div className="neon-panel rounded-lg border border-zinc-800 bg-zinc-950/95 p-4">
          <p className="text-xs uppercase tracking-wide text-zinc-400">{t('stats.errors')}</p>
          <p className="mt-1 text-2xl font-semibold text-red-400">{rangeTotals.errors}</p>
        </div>
        <div className="neon-panel rounded-lg border border-zinc-800 bg-zinc-950/95 p-4">
          <p className="text-xs uppercase tracking-wide text-zinc-400">{t('stats.rate')}</p>
          <p className="mt-1 text-2xl font-semibold text-red-300">{rangeTotals.avgRate.toFixed(1)}/s</p>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4 text-sm text-zinc-300">
        <p>{t('stats.liveNow', { attempts: stats.attempts, errors: stats.errors, rate: stats.rate.toFixed(1) })}</p>
      </div>
    </div>
  )
}
