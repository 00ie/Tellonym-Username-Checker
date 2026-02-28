export interface LiveStats {
  attempts: number
  found: number
  errors: number
  rateLimited: number
  rate: number
  avgResponse: number
  uptime: number
  isRunning: boolean
  isPaused: boolean
  recentFinds: string[]
}

export interface ProxyInfo {
  url: string
  type: string
  isAlive: boolean
  failCount: number
  successCount: number
  avgResponseMs: number
  lastCheck: string
}

export interface ProxyStats {
  total: number
  healthy: number
  dead: number
}

export interface ProxyTestResult {
  success: boolean
  latency: number
  error: string
}

export interface ProxyBatchCheckResult {
  checked: number
  healthy: number
  dead: number
  durationMs: number
}

export interface ProxyCleanupResult {
  removed: number
  remaining: number
}

export interface CheckerSettings {
  requestTimeoutMs: number
  maxRetries: number
  batchSize: number
  minLength: number
  maxLength: number
  allowLetters: boolean
  allowNumbers: boolean
  allowUnderscore: boolean
  allowDot: boolean
  disallowLeadingDot: boolean
  disallowTrailingDot: boolean
  maxConsecutiveDots: number
}

export interface WebhookSettings {
  enabled: boolean
  url: string
  username: string
  avatarURL: string
  timeoutMs: number
}

export interface HistoricalStats {
  timestamp: string
  attempts: number
  found: number
  errors: number
  rate: number
}

export interface AppConfig {
  name: string
  version: string
  environment: string
}
