import { create } from 'zustand'
import {
  CheckAllProxies,
  GetLiveStats,
  GetProxies,
  GetProxyStats,
  LoadProxies,
  PauseChecker,
  RemoveProxy,
  RemoveBadProxies,
  ResumeChecker,
  StartChecker,
  StopChecker,
  TestProxy,
} from '../services/backend'
import type { LiveStats, ProxyBatchCheckResult, ProxyCleanupResult, ProxyInfo, ProxyStats, ProxyTestResult } from '../types/api'

interface Stats {
  attempts: number
  found: number
  errors: number
  rateLimited: number
  rate: number
  avgResponse: number
  uptime: number
  isRunning: boolean
  isPaused: boolean
}

interface LiveData {
  timestamps: string[]
  rates: number[]
  recentFinds: string[]
}

interface AppState {
  stats: Stats
  liveData: LiveData
  proxies: ProxyInfo[]
  proxyStats: ProxyStats
  statsInterval?: number
  proxiesInterval?: number
  connectWebSocket: () => void
  disconnectWebSocket: () => void
  startChecker: (length: number, threads: number) => Promise<void>
  stopChecker: () => Promise<void>
  pauseChecker: () => Promise<void>
  resumeChecker: () => Promise<void>
  loadProxies: () => Promise<void>
  addProxies: (proxies: string[]) => Promise<void>
  removeProxy: (url: string) => Promise<void>
  removeBadProxies: () => Promise<ProxyCleanupResult>
  testProxy: (url: string) => Promise<ProxyTestResult>
  checkAllProxies: () => Promise<ProxyBatchCheckResult>
  updateLiveStats: () => Promise<void>
}

const statsPollMs = 1000
const proxiesPollMs = 5000

let statsPollingInFlight = false
let proxiesPollingInFlight = false

const defaultStats: Stats = {
  attempts: 0,
  found: 0,
  errors: 0,
  rateLimited: 0,
  rate: 0,
  avgResponse: 0,
  uptime: 0,
  isRunning: false,
  isPaused: false,
}

export const useStore = create<AppState>((set, get) => ({
  stats: defaultStats,
  liveData: {
    timestamps: [],
    rates: [],
    recentFinds: [],
  },
  proxies: [],
  proxyStats: {
    total: 0,
    healthy: 0,
    dead: 0,
  },

  connectWebSocket: () => {
    if (get().statsInterval || get().proxiesInterval) {
      return
    }

    void get().updateLiveStats()
    void get().loadProxies()

    const statsInterval = window.setInterval(() => {
      if (statsPollingInFlight) {
        return
      }
      statsPollingInFlight = true
      void get()
        .updateLiveStats()
        .finally(() => {
          statsPollingInFlight = false
        })
    }, statsPollMs)

    const proxiesInterval = window.setInterval(() => {
      if (proxiesPollingInFlight) {
        return
      }
      proxiesPollingInFlight = true
      void get()
        .loadProxies()
        .finally(() => {
          proxiesPollingInFlight = false
        })
    }, proxiesPollMs)

    set({ statsInterval, proxiesInterval })
  },

  disconnectWebSocket: () => {
    const { statsInterval, proxiesInterval } = get()
    if (statsInterval) {
      clearInterval(statsInterval)
    }
    if (proxiesInterval) {
      clearInterval(proxiesInterval)
    }
    statsPollingInFlight = false
    proxiesPollingInFlight = false
    set({ statsInterval: undefined, proxiesInterval: undefined })
  },

  updateLiveStats: async () => {
    try {
      const stats: LiveStats = await GetLiveStats()

      set((state) => {
        const timestamp = new Date().toLocaleTimeString()
        const timestamps = [...state.liveData.timestamps, timestamp].slice(-30)
        const rates = [...state.liveData.rates, stats.rate].slice(-30)

        return {
          stats: {
            attempts: stats.attempts,
            found: stats.found,
            errors: stats.errors,
            rateLimited: stats.rateLimited,
            rate: stats.rate,
            avgResponse: stats.avgResponse,
            uptime: stats.uptime,
            isRunning: stats.isRunning,
            isPaused: stats.isPaused,
          },
          liveData: {
            timestamps,
            rates,
            recentFinds: stats.recentFinds || [],
          },
        }
      })
    } catch (error) {
      console.error('failed to fetch live stats', error)
    }
  },

  startChecker: async (length: number, threads: number) => {
    await StartChecker(length, threads)
    await get().updateLiveStats()
    await get().loadProxies()
  },

  stopChecker: async () => {
    await StopChecker()
    await get().updateLiveStats()
  },

  pauseChecker: async () => {
    await PauseChecker()
    await get().updateLiveStats()
  },

  resumeChecker: async () => {
    await ResumeChecker()
    await get().updateLiveStats()
  },

  loadProxies: async () => {
    try {
      const [proxies, proxyStats] = await Promise.all([GetProxies(), GetProxyStats()])
      set({ proxies, proxyStats })
    } catch (error) {
      console.error('failed to fetch proxies', error)
    }
  },

  addProxies: async (proxies: string[]) => {
    await LoadProxies(proxies)
    await get().loadProxies()
  },

  removeProxy: async (url: string) => {
    await RemoveProxy(url)
    await get().loadProxies()
  },

  removeBadProxies: async () => {
    const result = await RemoveBadProxies()
    await get().loadProxies()
    return result
  },

  testProxy: async (url: string) => {
    return await TestProxy(url)
  },

  checkAllProxies: async () => {
    const result = await CheckAllProxies()
    await get().loadProxies()
    return result
  },
}))
