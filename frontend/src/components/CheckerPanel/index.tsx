import React, { useEffect, useState } from 'react'
import { GetCheckerSettings } from '../../services/backend'
import { useStore } from '../../store'
import { useI18n } from '../../i18n'

function clampNumber(value: number, min: number, max: number): number {
  if (value < min) {
    return min
  }
  if (value > max) {
    return max
  }
  return value
}

function parseNumberInput(raw: string, fallback: number): number {
  if (!raw.trim()) {
    return fallback
  }
  const parsed = Number.parseInt(raw, 10)
  if (Number.isNaN(parsed)) {
    return fallback
  }
  return parsed
}

export const CheckerPanel: React.FC = () => {
  const { stats, proxyStats, startChecker, stopChecker, pauseChecker, resumeChecker } = useStore()
  const { t } = useI18n()

  const [minLength, setMinLength] = useState(3)
  const [maxLength, setMaxLength] = useState(30)
  const [usernameLength, setUsernameLength] = useState(6)
  const [usernameLengthInput, setUsernameLengthInput] = useState('6')
  const [threads, setThreads] = useState(50)
  const [threadsInput, setThreadsInput] = useState('50')
  const [loading, setLoading] = useState(false)

  const effectiveMin = clampNumber(minLength, 3, 30)
  const effectiveMax = clampNumber(maxLength, effectiveMin, 30)
  const threadMin = 1
  const threadMax = 500
  const hardwareThreads = typeof navigator !== 'undefined' && navigator.hardwareConcurrency ? navigator.hardwareConcurrency : 8
  const idealThreads = proxyStats.healthy > 0
    ? clampNumber(proxyStats.healthy * 3, 3, 300)
    : clampNumber(hardwareThreads * 2, 8, 64)

  const normalizeUsernameLength = (raw: string): number => {
    const parsed = parseNumberInput(raw, usernameLength)
    return clampNumber(parsed, effectiveMin, effectiveMax)
  }

  const normalizeThreads = (raw: string): number => {
    const parsed = parseNumberInput(raw, threads)
    return clampNumber(parsed, threadMin, threadMax)
  }

  useEffect(() => {
    const load = async () => {
      const settings = await GetCheckerSettings()
      const normalizedMin = clampNumber(settings.minLength, 3, 30)
      const normalizedMax = clampNumber(settings.maxLength, normalizedMin, 30)

      setMinLength(normalizedMin)
      setMaxLength(normalizedMax)
      setUsernameLength((previous) => {
        const normalized = clampNumber(previous, normalizedMin, normalizedMax)
        setUsernameLengthInput(String(normalized))
        return normalized
      })
    }
    void load()
  }, [])

  const onStart = async () => {
    const normalizedLength = normalizeUsernameLength(usernameLengthInput)
    const normalizedThreads = normalizeThreads(threadsInput)
    setUsernameLength(normalizedLength)
    setUsernameLengthInput(String(normalizedLength))
    setThreads(normalizedThreads)
    setThreadsInput(String(normalizedThreads))

    setLoading(true)
    try {
      await startChecker(normalizedLength, normalizedThreads)
    } finally {
      setLoading(false)
    }
  }

  const onStop = async () => {
    setLoading(true)
    try {
      await stopChecker()
    } finally {
      setLoading(false)
    }
  }

  const onPauseResume = async () => {
    setLoading(true)
    try {
      if (stats.isPaused) {
        await resumeChecker()
      } else {
        await pauseChecker()
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="neon-panel rounded-2xl border border-red-500/35 bg-zinc-950/95 p-4 shadow-[0_0_30px_rgba(239,68,68,0.16)]">
        <h2 className="mb-4 text-2xl font-semibold text-red-100">{t('checker.control')}</h2>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('checker.usernameLength')}</span>
            <input
              type="text"
              inputMode="numeric"
              value={usernameLengthInput}
              onChange={(e) => {
                const next = e.target.value
                if (/^\d*$/.test(next)) {
                  setUsernameLengthInput(next)
                }
              }}
              onBlur={() => {
                const normalized = normalizeUsernameLength(usernameLengthInput)
                setUsernameLength(normalized)
                setUsernameLengthInput(String(normalized))
              }}
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
            />
            <span className="text-xs text-zinc-400">{t('checker.allowedRange', { min: effectiveMin, max: effectiveMax })}</span>
          </label>

          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('checker.threads')}</span>
            <input
              type="text"
              inputMode="numeric"
              value={threadsInput}
              onChange={(e) => {
                const next = e.target.value
                if (/^\d*$/.test(next)) {
                  setThreadsInput(next)
                }
              }}
              onBlur={() => {
                const normalized = normalizeThreads(threadsInput)
                setThreads(normalized)
                setThreadsInput(String(normalized))
              }}
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
            />
            <div className="flex items-center justify-between gap-2 text-xs text-zinc-400">
              <span>{t('checker.idealThreads', { value: idealThreads, healthy: proxyStats.healthy })}</span>
              <button
                type="button"
                onClick={() => {
                  setThreads(idealThreads)
                  setThreadsInput(String(idealThreads))
                }}
                className="rounded border border-red-500/50 px-2 py-1 text-red-200 transition hover:bg-red-500/20"
              >
                {t('checker.useIdeal')}
              </button>
            </div>
          </label>
        </div>

        <div className="mt-5 flex flex-wrap gap-3">
          <button
            disabled={loading || stats.isRunning}
            onClick={onStart}
            className="rounded-lg border border-red-500/60 bg-zinc-900 px-4 py-2 font-medium text-red-100 transition hover:bg-zinc-800 disabled:opacity-50"
          >
            {t('checker.start')}
          </button>
          <button
            disabled={loading || !stats.isRunning}
            onClick={onPauseResume}
            className="rounded-lg border border-amber-500/60 bg-amber-700/70 px-4 py-2 font-medium transition hover:bg-amber-600 disabled:opacity-50"
          >
            {stats.isPaused ? t('checker.resume') : t('checker.pause')}
          </button>
          <button
            disabled={loading || !stats.isRunning}
            onClick={onStop}
            className="rounded-lg border border-red-500/80 bg-red-700/90 px-4 py-2 font-medium text-white transition hover:bg-red-600 disabled:opacity-50"
          >
            {t('checker.stop')}
          </button>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <h3 className="mb-3 text-lg font-semibold text-red-100">{t('checker.currentStatus')}</h3>
        <div className="grid grid-cols-1 gap-3 text-sm md:grid-cols-4">
          <div className="rounded-md border border-zinc-800 bg-black/70 p-3">
            {t('checker.running')}: {stats.isRunning ? t('common.yes') : t('common.no')}
          </div>
          <div className="rounded-md border border-zinc-800 bg-black/70 p-3">
            {t('checker.paused')}: {stats.isPaused ? t('common.yes') : t('common.no')}
          </div>
          <div className="rounded-md border border-zinc-800 bg-black/70 p-3">
            {t('checker.rate')}: {stats.rate.toFixed(1)}/s
          </div>
          <div className="rounded-md border border-zinc-800 bg-black/70 p-3">
            {t('checker.errors')}: {stats.errors}
          </div>
        </div>
      </div>
    </div>
  )
}
