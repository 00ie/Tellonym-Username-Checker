import type {
  AppConfig,
  ProxyBatchCheckResult,
  ProxyCleanupResult,
  CheckerSettings,
  HistoricalStats,
  LiveStats,
  ProxyInfo,
  ProxyStats,
  ProxyTestResult,
  WebhookSettings,
} from '../types/api'

const emptyStats: LiveStats = {
  attempts: 0,
  found: 0,
  errors: 0,
  rateLimited: 0,
  rate: 0,
  avgResponse: 0,
  uptime: 0,
  isRunning: false,
  isPaused: false,
  recentFinds: [],
}

const emptyProxyStats: ProxyStats = {
  total: 0,
  healthy: 0,
  dead: 0,
}

const emptyProxyBatchCheck: ProxyBatchCheckResult = {
  checked: 0,
  healthy: 0,
  dead: 0,
  durationMs: 0,
}

const emptyProxyCleanup: ProxyCleanupResult = {
  removed: 0,
  remaining: 0,
}

const defaultConfig: AppConfig = {
  name: 'Tellonym Username Checker',
  version: '1.0.0',
  environment: 'development',
}

const defaultCheckerSettings: CheckerSettings = {
  requestTimeoutMs: 10000,
  maxRetries: 3,
  batchSize: 100,
  minLength: 3,
  maxLength: 30,
  allowLetters: true,
  allowNumbers: true,
  allowUnderscore: true,
  allowDot: true,
  disallowLeadingDot: true,
  disallowTrailingDot: true,
  maxConsecutiveDots: 1,
}

const defaultWebhookSettings: WebhookSettings = {
  enabled: false,
  url: '',
  username: 'Gon',
  avatarURL: 'https://i.pinimg.com/736x/dd/f4/75/ddf475e4b9767235362fc1cf3a16ed1c.jpg',
  timeoutMs: 10000,
}

function getBoundApp(): any {
  const go = (window as Window).go as any
  if (!go) {
    return null
  }
  return go.backend?.App ?? go.main?.App ?? null
}

async function invoke<T>(method: string, args: unknown[], fallback: T): Promise<T> {
  const app = getBoundApp()
  const fn = app?.[method]
  if (typeof fn !== 'function') {
    return fallback
  }
  return (await fn(...args)) as T
}

export function StartChecker(length: number, threads: number): Promise<void> {
  return invoke<void>('StartChecker', [length, threads], undefined as void)
}

export function StopChecker(): Promise<void> {
  return invoke<void>('StopChecker', [], undefined as void)
}

export function PauseChecker(): Promise<void> {
  return invoke<void>('PauseChecker', [], undefined as void)
}

export function ResumeChecker(): Promise<void> {
  return invoke<void>('ResumeChecker', [], undefined as void)
}

export function GetLiveStats(): Promise<LiveStats> {
  return invoke<LiveStats>('GetLiveStats', [], emptyStats)
}

export function LoadProxies(proxies: string[]): Promise<void> {
  return invoke<void>('LoadProxies', [proxies], undefined as void)
}

export function GetProxies(): Promise<ProxyInfo[]> {
  return invoke<ProxyInfo[]>('GetProxies', [], [])
}

export function RemoveProxy(url: string): Promise<void> {
  return invoke<void>('RemoveProxy', [url], undefined as void)
}

export function TestProxy(url: string): Promise<ProxyTestResult> {
  return invoke<ProxyTestResult>('TestProxy', [url], { success: false, latency: 0, error: 'backend unavailable' })
}

export function CheckAllProxies(): Promise<ProxyBatchCheckResult> {
  return invoke<ProxyBatchCheckResult>('CheckAllProxies', [], emptyProxyBatchCheck)
}

export function RemoveBadProxies(): Promise<ProxyCleanupResult> {
  return invoke<ProxyCleanupResult>('RemoveBadProxies', [], emptyProxyCleanup)
}

export function GetProxyStats(): Promise<ProxyStats> {
  return invoke<ProxyStats>('GetProxyStats', [], emptyProxyStats)
}

export function GetHistoricalStats(from: string, to: string): Promise<HistoricalStats[]> {
  return invoke<HistoricalStats[]>('GetHistoricalStats', [from, to], [])
}

export function UpdateCheckerSettings(settings: CheckerSettings): Promise<void> {
  return invoke<void>('UpdateCheckerSettings', [settings], undefined as void)
}

export function GetCheckerSettings(): Promise<CheckerSettings> {
  return invoke<CheckerSettings>('GetCheckerSettings', [], defaultCheckerSettings)
}

export function GetWebhookSettings(): Promise<WebhookSettings> {
  return invoke<WebhookSettings>('GetWebhookSettings', [], defaultWebhookSettings)
}

export function UpdateWebhookSettings(settings: WebhookSettings): Promise<void> {
  return invoke<void>('UpdateWebhookSettings', [settings], undefined as void)
}

export function SendTestWebhook(username: string): Promise<void> {
  return invoke<void>('SendTestWebhook', [username], undefined as void)
}

export function GetConfig(): Promise<AppConfig> {
  return invoke<AppConfig>('GetConfig', [], defaultConfig)
}
