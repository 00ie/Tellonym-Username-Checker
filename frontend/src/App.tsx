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
import { useStore } from './store'
import { useTheme } from './theme'
import { accentPalette } from './theme/palette'

const queryClient = new QueryClient()

function App() {
  const [activeTab, setActiveTab] = useState('dashboard')
  const [showRateLimitWarning, setShowRateLimitWarning] = useState(false)
  const { connectWebSocket, disconnectWebSocket, loadProxies, stats, proxyStats } = useStore()
  const { theme } = useTheme()
  const lastAlertAt = useRef(0)
  const lastRateLimitedSeen = useRef(0)
  const palette = accentPalette[theme]

  useEffect(() => {
    connectWebSocket()
    loadProxies()

    return () => {
      disconnectWebSocket()
    }
  }, [connectWebSocket, disconnectWebSocket, loadProxies])

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
        <Navbar activeTab={activeTab} onTabChange={setActiveTab} />

        <main className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          {activeTab === 'dashboard' && <Dashboard />}
          {activeTab === 'checker' && <CheckerPanel />}
          {activeTab === 'proxies' && <ProxyManager />}
          {activeTab === 'statistics' && <Statistics />}
          {activeTab === 'settings' && <Settings />}
          {activeTab === 'about' && <About />}
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
