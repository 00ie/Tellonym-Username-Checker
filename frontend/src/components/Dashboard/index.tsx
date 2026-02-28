import React, { useMemo, useState } from 'react'
import {
  Chart as ChartJS,
  CategoryScale,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip,
  type ChartData,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'react-chartjs-2'
import {
  ArrowPathIcon,
  BoltIcon,
  CheckBadgeIcon,
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  PlayCircleIcon,
  SparklesIcon,
  StopCircleIcon,
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { useStore } from '../../store'
import { useI18n } from '../../i18n'
import { formatDuration, formatNumber } from '../../utils/format'
import { useTheme } from '../../theme'
import { accentPalette } from '../../theme/palette'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const iconColor: Record<string, string> = {
  red: 'text-red-200',
  emerald: 'text-emerald-200',
  amber: 'text-amber-200',
  zinc: 'text-zinc-300',
}

const cardSurface: Record<string, string> = {
  red: 'border-red-500/30 bg-zinc-950/95',
  emerald: 'border-emerald-500/30 bg-zinc-950/95',
  amber: 'border-amber-500/30 bg-zinc-950/95',
  zinc: 'border-zinc-800 bg-zinc-950/95',
}

export const Dashboard: React.FC = () => {
  const { stats, liveData, startChecker, stopChecker, clearDashboardData } = useStore()
  const { t } = useI18n()
  const { theme } = useTheme()
  const [loading, setLoading] = useState(false)
  const palette = accentPalette[theme]

  const handleToggle = async () => {
    setLoading(true)
    try {
      if (stats.isRunning) {
        await stopChecker()
      } else {
        await startChecker(0, 50)
      }
    } finally {
      setLoading(false)
    }
  }

  const handleClearData = async () => {
    setLoading(true)
    try {
      await clearDashboardData()
      toast.success(t('dashboard.toast.clearSuccess'))
    } catch {
      toast.error(t('dashboard.toast.clearFailed'))
    } finally {
      setLoading(false)
    }
  }

  const chartData = useMemo<ChartData<'line'>>(
    () => ({
      labels: liveData.timestamps,
      datasets: [
        {
          label: t('dashboard.chart.rate'),
          data: liveData.rates,
          borderColor: palette.border,
          backgroundColor: palette.fill,
          pointRadius: 0,
          borderWidth: 2.5,
          fill: true,
          tension: 0.3,
        },
      ],
    }),
    [liveData.rates, liveData.timestamps, palette.border, palette.fill, t],
  )

  const availabilityPct = stats.attempts > 0 ? (stats.found / stats.attempts) * 100 : 0
  const errorPct = stats.attempts > 0 ? (stats.errors / stats.attempts) * 100 : 0

  const runtimeStatus = stats.isRunning
    ? stats.isPaused
      ? t('dashboard.statusPaused')
      : t('dashboard.statusRunning')
    : t('dashboard.statusStopped')

  const runtimeStatusStyle = stats.isRunning
    ? stats.isPaused
      ? 'border-amber-400/40 bg-amber-500/10 text-amber-200'
      : 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200'
    : 'border-red-500/40 bg-red-500/10 text-red-200'

  const chartOptions: ChartOptions<'line'> = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
    },
    scales: {
      x: {
        ticks: { color: 'rgba(212, 212, 216, 0.85)' },
        grid: { color: 'rgba(82, 82, 91, 0.25)' },
      },
      y: {
        ticks: { color: 'rgba(212, 212, 216, 0.85)' },
        grid: { color: 'rgba(82, 82, 91, 0.25)' },
      },
    },
  }

  const cards = [
    { title: t('dashboard.cards.attempts'), value: formatNumber(stats.attempts), icon: BoltIcon, color: 'zinc' },
    { title: t('dashboard.cards.found'), value: formatNumber(stats.found), icon: CheckCircleIcon, color: 'emerald' },
    { title: t('dashboard.cards.errors'), value: formatNumber(stats.errors), icon: ExclamationTriangleIcon, color: 'red' },
    { title: t('dashboard.cards.rate'), value: `${stats.rate.toFixed(1)}/s`, icon: ArrowPathIcon, color: 'amber' },
    { title: t('dashboard.cards.uptime'), value: formatDuration(stats.uptime), icon: ClockIcon, color: 'zinc' },
    { title: t('dashboard.cards.avgResponse'), value: `${stats.avgResponse}ms`, icon: ClockIcon, color: 'red' },
  ]

  const highlights = [
    {
      title: t('dashboard.highlights.available'),
      value: `${availabilityPct.toFixed(2)}%`,
      icon: CheckBadgeIcon,
      tone: 'emerald',
    },
    {
      title: t('dashboard.highlights.errorRate'),
      value: `${errorPct.toFixed(2)}%`,
      icon: ExclamationTriangleIcon,
      tone: 'red',
    },
    {
      title: t('dashboard.highlights.totalFinds'),
      value: formatNumber(stats.found),
      icon: SparklesIcon,
      tone: 'amber',
    },
  ]

  return (
    <div className="space-y-6">
      <section className="neon-panel relative overflow-hidden rounded-2xl border border-red-500/45 bg-gradient-to-r from-black via-zinc-950 to-zinc-950 p-6 shadow-[0_0_55px_rgba(239,68,68,0.22)]">
        <div className="relative grid gap-6 lg:grid-cols-[1.35fr,1fr] lg:items-center">
          <div>
            <div className="flex items-center gap-3">
              <p className="text-xs uppercase tracking-[0.26em] text-red-300/90">{t('dashboard.live')}</p>
              <span className={`rounded-full border px-3 py-1 text-xs font-medium ${runtimeStatusStyle}`}>{runtimeStatus}</span>
            </div>
            <h2 className="mt-3 text-3xl font-semibold text-red-100 sm:text-4xl">{t('dashboard.title')}</h2>
            <p className="mt-3 max-w-2xl text-sm text-zinc-300">{t('dashboard.subtitle')}</p>
            <div className="mt-5 flex flex-wrap items-center gap-3">
              <button
                onClick={handleToggle}
                disabled={loading}
                className={`inline-flex items-center gap-2 rounded-xl border px-5 py-3 font-medium transition ${
                  stats.isRunning
                    ? 'border-red-500/80 bg-red-700/90 text-white hover:bg-red-600'
                    : 'border-red-500/60 bg-zinc-900 text-red-100 hover:bg-zinc-800'
                } disabled:cursor-not-allowed disabled:opacity-60`}
              >
                {stats.isRunning ? <StopCircleIcon className="h-5 w-5" /> : <PlayCircleIcon className="h-5 w-5" />}
                {stats.isRunning ? t('dashboard.stop') : t('dashboard.start')}
              </button>
              <button
                onClick={handleClearData}
                disabled={loading || stats.isRunning}
                className="inline-flex items-center gap-2 rounded-xl border border-zinc-700/90 bg-black/55 px-5 py-3 font-medium text-zinc-200 transition hover:border-red-500/70 hover:text-red-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {t('dashboard.clearData')}
              </button>
              <div className="rounded-xl border border-zinc-700/70 bg-black/40 px-4 py-3">
                <p className="text-xs uppercase tracking-[0.2em] text-zinc-400">{t('dashboard.instantRate')}</p>
                <p className="text-2xl font-semibold text-red-100">{stats.rate.toFixed(1)}/s</p>
              </div>
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
            {highlights.map((item) => (
              <article key={item.title} className={`neon-panel rounded-xl border p-4 ${cardSurface[item.tone]}`}>
                <div className="flex items-center justify-between">
                  <p className="text-xs uppercase tracking-wider text-zinc-400">{item.title}</p>
                  <item.icon className={`h-5 w-5 ${iconColor[item.tone]}`} />
                </div>
                <p className="mt-2 text-xl font-semibold text-zinc-100">{item.value}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        {cards.map((card) => (
          <article key={card.title} className={`neon-panel rounded-xl border p-4 ${cardSurface[card.color]}`}>
            <div className="mb-3 flex items-center justify-between">
              <p className="text-xs uppercase tracking-wider text-zinc-400">{card.title}</p>
              <card.icon className={`h-5 w-5 ${iconColor[card.color]}`} />
            </div>
            <p className="text-2xl font-semibold text-zinc-100">{card.value}</p>
          </article>
        ))}
      </section>

      <section className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4 sm:p-5">
        <h3 className="mb-3 text-lg font-semibold text-red-100">{t('dashboard.timeline')}</h3>
        <div className="h-72">
          <Line data={chartData} options={chartOptions} />
        </div>
      </section>

      <section className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <h3 className="mb-3 text-lg font-semibold text-red-100">{t('dashboard.recentFinds')}</h3>
        <div className="grid grid-cols-2 gap-2 md:grid-cols-4 lg:grid-cols-6">
          {liveData.recentFinds.length === 0 && <span className="text-sm text-zinc-500">{t('dashboard.noneFound')}</span>}
          {liveData.recentFinds.map((username, index) => (
            <div
              key={`${username}-${index}`}
              className="rounded-md border border-emerald-300/20 bg-emerald-400/10 px-3 py-2 font-mono text-sm text-emerald-300"
            >
              @{username}
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
