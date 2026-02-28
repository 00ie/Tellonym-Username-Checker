import React, { useEffect, useRef, useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import { Dashboard } from './components/Dashboard'
import { CheckerPanel } from './components/CheckerPanel'
import { ProxyManager } from './components/ProxyManager'
import { Statistics } from './components/Statistics'
import { Settings } from './components/Settings'
import { Navbar } from './components/Navbar'
import { About } from './components/About'
import { Footer } from './components/Footer'
import { RateLimitWarning } from './components/RateLimitWarning'
import { Ping } from './services/backend'
import { useStore } from './store'
import { useTheme } from './theme'
import { accentPalette } from './theme/palette'

const queryClient = new QueryClient()
const activeTabStorageKey = 'tellonym.activeTab'
const tabIds = ['dashboard', 'checker', 'proxies', 'statistics', 'settings', 'about'] as const
type TabId = (typeof tabIds)[number]

function normalizeTab(value: string | null): TabId {
  if (value && tabIds.includes(value as TabId)) {
    return value as TabId
  }
  return 'dashboard'
}

function App() {
  const [activeTab, setActiveTab] = useState<TabId>(() => normalizeTab(localStorage.getItem(activeTabStorageKey)))
  const [showRateLimitWarning, setShowRateLimitWarning] = useState(false)
  const [backendConnected, setBackendConnected] = useState(false)
  const { connectWebSocket, disconnectWebSocket, loadProxies, stats, proxyStats } = useStore()
  const { theme } = useTheme()
  const lastAlertAt = useRef(0)
  const lastRateLimitedSeen = useRef(0)
  const palette = accentPalette[theme]

  useEffect(() => {
    localStorage.setItem(activeTabStorageKey, activeTab)
  }, [activeTab])

  useEffect(() => {
    connectWebSocket()
    loadProxies()

    return () => {
      disconnectWebSocket()
    }
  }, [connectWebSocket, disconnectWebSocket, loadProxies])

  useEffect(() => {
    let cancelled = false

    const checkBackend = async () => {
      try {
        const healthy = await Promise.race<boolean>([
          Ping(),
          new Promise<boolean>((resolve) => {
            window.setTimeout(() => resolve(false), 1800)
          }),
        ])
        if (!cancelled) {
          setBackendConnected(Boolean(healthy))
        }
      } catch {
        if (!cancelled) {
          setBackendConnected(false)
        }
      }
    }

    void checkBackend()
    const id = window.setInterval(() => {
      void checkBackend()
    }, 5000)

    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [])

  useEffect(() => {
    if (!stats.isRunning) {
      return
    }

    const now = Date.now()
    const errorRate = stats.attempts > 0 ? stats.errors / stats.attempts : 0

    let hasNewRateLimited = false
    if (stats.rateLimited > lastRateLimitedSeen.current) {
      hasNewRateLimited = true
      lastRateLimitedSeen.current = stats.rateLimited
    }

    const severeErrorProfile = stats.attempts >= 120 && errorRate >= 0.7
    const weakProxyPool = proxyStats.total > 0 && proxyStats.healthy <= 1
    const shouldWarn = hasNewRateLimited || (severeErrorProfile && weakProxyPool)

    if (!shouldWarn) {
      return
    }

    if (now - lastAlertAt.current < 45000) {
      return
    }

    lastAlertAt.current = now
    setShowRateLimitWarning(true)
  }, [proxyStats.healthy, proxyStats.total, stats.attempts, stats.errors, stats.isRunning, stats.rateLimited])

  return (
    <QueryClientProvider client={queryClient}>
      <div className="neon-background min-h-screen bg-black text-white">
        <Navbar
          activeTab={activeTab}
          onTabChange={(tab) => setActiveTab(normalizeTab(tab))}
          backendConnected={backendConnected}
        />

        <main className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          <section className={activeTab === 'dashboard' ? 'block' : 'hidden'}>
            <Dashboard />
          </section>
          <section className={activeTab === 'checker' ? 'block' : 'hidden'}>
            <CheckerPanel />
          </section>
          <section className={activeTab === 'proxies' ? 'block' : 'hidden'}>
            <ProxyManager />
          </section>
          <section className={activeTab === 'statistics' ? 'block' : 'hidden'}>
            <Statistics active={activeTab === 'statistics'} />
          </section>
          <section className={activeTab === 'settings' ? 'block' : 'hidden'}>
            <Settings />
          </section>
          <section className={activeTab === 'about' ? 'block' : 'hidden'}>
            <About />
          </section>
        </main>

        <Footer />

        <RateLimitWarning
          open={showRateLimitWarning}
          stats={{
            attempts: stats.attempts,
            errors: stats.errors,
            rateLimited: stats.rateLimited,
            avgResponse: stats.avgResponse,
            rate: stats.rate,
          }}
          proxyStats={proxyStats}
          onClose={() => setShowRateLimitWarning(false)}
          onOpenProxies={() => {
            setActiveTab('proxies')
            setShowRateLimitWarning(false)
          }}
          onOpenChecker={() => {
            setActiveTab('checker')
            setShowRateLimitWarning(false)
          }}
        />

        <Toaster
          position="top-right"
          toastOptions={{
            duration: 3000,
            style: {
              background: '#090909',
              color: '#f8fafc',
              border: `1px solid ${palette.border}`,
            },
          }}
        />
      </div>
    </QueryClientProvider>
  )
}

export default App
