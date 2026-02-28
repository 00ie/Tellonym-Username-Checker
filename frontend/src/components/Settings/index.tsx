import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import {
  GetCheckerSettings,
  GetConfig,
  GetWebhookSettings,
  SendTestWebhook,
  UpdateCheckerSettings,
  UpdateWebhookSettings,
} from '../../services/backend'
import { type Language, useI18n } from '../../i18n'
import { type AccentTheme, useTheme } from '../../theme'
import type { CheckerSettings, WebhookSettings } from '../../types/api'

const fixedWebhookUsername = 'Gon'
const fixedWebhookAvatar = 'https://i.pinimg.com/736x/dd/f4/75/ddf475e4b9767235362fc1cf3a16ed1c.jpg'

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
  username: fixedWebhookUsername,
  avatarURL: fixedWebhookAvatar,
  timeoutMs: 10000,
}

type ToggleField =
  | 'allowLetters'
  | 'allowNumbers'
  | 'allowUnderscore'
  | 'allowDot'
  | 'disallowLeadingDot'
  | 'disallowTrailingDot'

type ValidationResult = {
  valid: boolean
  messageKey: string
  messageParams?: Record<string, string | number>
}

function validateUsername(username: string, rules: CheckerSettings): ValidationResult {
  if (!username) {
    return { valid: false, messageKey: 'settings.validation.empty' }
  }

  if (username.length < rules.minLength) {
    return { valid: false, messageKey: 'settings.validation.min', messageParams: { min: rules.minLength } }
  }

  if (username.length > rules.maxLength) {
    return { valid: false, messageKey: 'settings.validation.max', messageParams: { max: rules.maxLength } }
  }

  let charset = ''
  if (rules.allowLetters) {
    charset += 'abcdefghijklmnopqrstuvwxyz'
  }
  if (rules.allowNumbers) {
    charset += '0123456789'
  }
  if (rules.allowUnderscore) {
    charset += '_'
  }
  if (rules.allowDot) {
    charset += '.'
  }

  if (!charset) {
    return { valid: false, messageKey: 'settings.validation.group' }
  }

  for (let i = 0; i < username.length; i += 1) {
    if (!charset.includes(username[i])) {
      return { valid: false, messageKey: 'settings.validation.unsupported' }
    }
  }

  if (rules.disallowLeadingDot && username.startsWith('.')) {
    return { valid: false, messageKey: 'settings.validation.leadingDot' }
  }

  if (rules.disallowTrailingDot && username.endsWith('.')) {
    return { valid: false, messageKey: 'settings.validation.trailingDot' }
  }

  if (rules.allowDot && rules.maxConsecutiveDots > 0) {
    let consecutive = 0
    for (let i = 0; i < username.length; i += 1) {
      if (username[i] === '.') {
        consecutive += 1
        if (consecutive > rules.maxConsecutiveDots) {
          return { valid: false, messageKey: 'settings.validation.dots', messageParams: { max: rules.maxConsecutiveDots } }
        }
      } else {
        consecutive = 0
      }
    }
  }

  return { valid: true, messageKey: 'settings.validation.valid' }
}

export const Settings: React.FC = () => {
  const { language, setLanguage, t } = useI18n()
  const { theme, setTheme } = useTheme()
  const [appName, setAppName] = useState('Tellonym Username Checker')
  const [environment, setEnvironment] = useState('production')
  const [checkerForm, setCheckerForm] = useState<CheckerSettings>(defaultCheckerSettings)
  const [webhookForm, setWebhookForm] = useState<WebhookSettings>(defaultWebhookSettings)
  const [sampleUsername, setSampleUsername] = useState('_o1')
  const [testWebhookUsername, setTestWebhookUsername] = useState('available_name')
  const [saving, setSaving] = useState(false)
  const [sendingTest, setSendingTest] = useState(false)

  useEffect(() => {
    const load = async () => {
      const [cfg, checker, webhook] = await Promise.all([GetConfig(), GetCheckerSettings(), GetWebhookSettings()])
      setAppName(cfg.name)
      setEnvironment(cfg.environment)
      setCheckerForm(checker)
      setWebhookForm({
        enabled: webhook.enabled,
        url: webhook.url,
        timeoutMs: webhook.timeoutMs,
        username: fixedWebhookUsername,
        avatarURL: fixedWebhookAvatar,
      })
    }

    void load()
  }, [])

  const hasCharset =
    checkerForm.allowLetters || checkerForm.allowNumbers || checkerForm.allowUnderscore || checkerForm.allowDot
  const hasValidLength =
    checkerForm.minLength >= 3 && checkerForm.maxLength <= 30 && checkerForm.minLength <= checkerForm.maxLength
  const hasValidWebhook = !webhookForm.enabled || webhookForm.url.trim().length > 0
  const canSave = hasCharset && hasValidLength && hasValidWebhook

  const validation = useMemo(() => validateUsername(sampleUsername, checkerForm), [sampleUsername, checkerForm])
  const profileLink = sampleUsername ? `https://tellonym.me/${sampleUsername}` : 'https://tellonym.me/<username>'

  const onSave = async () => {
    if (!canSave) {
      toast.error(t('settings.toast.fixInvalid'))
      return
    }

    setSaving(true)
    try {
      await UpdateCheckerSettings(checkerForm)
      await UpdateWebhookSettings({
        ...webhookForm,
        username: fixedWebhookUsername,
        avatarURL: fixedWebhookAvatar,
      })
      toast.success(t('settings.toast.saved'))
    } catch {
      toast.error(t('settings.toast.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const onSendTest = async () => {
    if (!webhookForm.url.trim()) {
      toast.error(t('settings.toast.addUrlFirst'))
      return
    }

    setSendingTest(true)
    try {
      await SendTestWebhook(testWebhookUsername.trim())
      toast.success(t('settings.toast.testSent'))
    } catch {
      toast.error(t('settings.toast.testFailed'))
    } finally {
      setSendingTest(false)
    }
  }

  const toggleChecker = (key: ToggleField) => {
    setCheckerForm((previous) => ({ ...previous, [key]: !previous[key] }))
  }

  const onLanguageChange = (value: string) => {
    if (value === 'pt' || value === 'en') {
      setLanguage(value as Language)
    }
  }

  const onThemeChange = (value: string) => {
    if (value === 'red' || value === 'blue' || value === 'green' || value === 'purple') {
      setTheme(value as AccentTheme)
    }
  }

  return (
    <div className="space-y-6">
      <div className="neon-panel rounded-2xl border border-red-500/35 bg-gradient-to-r from-black via-zinc-950 to-red-950/60 p-5 shadow-[0_0_40px_rgba(239,68,68,0.2)]">
        <h2 className="text-2xl font-semibold text-red-100">{t('settings.title')}</h2>
        <p className="mt-1 text-sm text-zinc-300">{t('settings.header', { app: appName, env: environment })}</p>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <h3 className="mb-3 text-lg font-semibold text-red-100">{t('settings.language')}</h3>
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={language}
            onChange={(e) => onLanguageChange(e.target.value)}
            className="rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
          >
            <option value="pt">{t('settings.lang.pt')}</option>
            <option value="en">{t('settings.lang.en')}</option>
          </select>
          <p className="text-sm text-zinc-400">{t('settings.languageHelp')}</p>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <h3 className="mb-3 text-lg font-semibold text-red-100">{t('settings.theme')}</h3>
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={theme}
            onChange={(e) => onThemeChange(e.target.value)}
            className="rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
          >
            <option value="red">{t('settings.theme.red')}</option>
            <option value="blue">{t('settings.theme.blue')}</option>
            <option value="green">{t('settings.theme.green')}</option>
            <option value="purple">{t('settings.theme.purple')}</option>
          </select>
          <p className="text-sm text-zinc-400">{t('settings.themeHelp')}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
          <h3 className="mb-4 text-lg font-semibold text-red-100">{t('settings.runtime')}</h3>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.requestTimeout')}</span>
              <input
                type="number"
                min={1000}
                value={checkerForm.requestTimeoutMs}
                onChange={(e) => setCheckerForm((previous) => ({ ...previous, requestTimeoutMs: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.maxRetries')}</span>
              <input
                type="number"
                min={0}
                max={10}
                value={checkerForm.maxRetries}
                onChange={(e) => setCheckerForm((previous) => ({ ...previous, maxRetries: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.batchSize')}</span>
              <input
                type="number"
                min={1}
                value={checkerForm.batchSize}
                onChange={(e) => setCheckerForm((previous) => ({ ...previous, batchSize: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
          </div>
        </div>

        <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
          <h3 className="mb-4 text-lg font-semibold text-red-100">{t('settings.allowedChars')}</h3>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.letters')}</span>
              <input type="checkbox" checked={checkerForm.allowLetters} onChange={() => toggleChecker('allowLetters')} />
            </label>
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.numbers')}</span>
              <input type="checkbox" checked={checkerForm.allowNumbers} onChange={() => toggleChecker('allowNumbers')} />
            </label>
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.underscore')}</span>
              <input
                type="checkbox"
                checked={checkerForm.allowUnderscore}
                onChange={() => toggleChecker('allowUnderscore')}
              />
            </label>
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.dot')}</span>
              <input type="checkbox" checked={checkerForm.allowDot} onChange={() => toggleChecker('allowDot')} />
            </label>
          </div>
          {!hasCharset && <p className="mt-3 text-sm text-red-400">{t('settings.enableOneGroup')}</p>}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
          <h3 className="mb-4 text-lg font-semibold text-red-100">{t('settings.lengthRules')}</h3>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.minLength')}</span>
              <input
                type="number"
                min={3}
                max={30}
                value={checkerForm.minLength}
                onChange={(e) => setCheckerForm((previous) => ({ ...previous, minLength: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.maxLength')}</span>
              <input
                type="number"
                min={3}
                max={30}
                value={checkerForm.maxLength}
                onChange={(e) => setCheckerForm((previous) => ({ ...previous, maxLength: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.maxDots')}</span>
              <input
                type="number"
                min={1}
                max={5}
                value={checkerForm.maxConsecutiveDots}
                onChange={(e) => setCheckerForm((previous) => ({ ...previous, maxConsecutiveDots: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500 disabled:opacity-60"
                disabled={!checkerForm.allowDot}
              />
            </label>
          </div>
          <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.blockLeadingDot')}</span>
              <input
                type="checkbox"
                checked={checkerForm.disallowLeadingDot}
                onChange={() => toggleChecker('disallowLeadingDot')}
                disabled={!checkerForm.allowDot}
              />
            </label>
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.blockTrailingDot')}</span>
              <input
                type="checkbox"
                checked={checkerForm.disallowTrailingDot}
                onChange={() => toggleChecker('disallowTrailingDot')}
                disabled={!checkerForm.allowDot}
              />
            </label>
          </div>
          {!hasValidLength && <p className="mt-3 text-sm text-red-400">{t('settings.lengthError')}</p>}
        </div>

        <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
          <h3 className="mb-4 text-lg font-semibold text-red-100">{t('settings.liveValidation')}</h3>
          <label className="space-y-2">
            <span className="text-sm text-zinc-300">{t('settings.sampleUsername')}</span>
            <input
              value={sampleUsername}
              onChange={(e) => setSampleUsername(e.target.value)}
              placeholder="_o1"
              className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 font-mono text-zinc-100 outline-none focus:border-red-500"
            />
          </label>
          <div className="mt-4 rounded-lg border border-zinc-800 bg-black/60 p-3">
            <p className="text-sm text-zinc-300">{t('settings.profileLink')}</p>
            <p className="mt-1 break-all font-mono text-red-300">{profileLink}</p>
          </div>
          <div className="mt-3 rounded-lg border border-zinc-800 bg-black/60 p-3">
            <p className={`text-sm font-medium ${validation.valid ? 'text-emerald-400' : 'text-amber-400'}`}>
              {t(validation.messageKey, validation.messageParams)}
            </p>
          </div>
          <div className="mt-4 text-sm text-zinc-300">
            <p>{t('settings.examples')}</p>
            <p className="font-mono text-emerald-400">_o1  |  o.1  |  o__1</p>
            <p className="font-mono text-red-400">.o1  |  o1.</p>
          </div>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <h3 className="mb-4 text-lg font-semibold text-red-100">{t('settings.webhook')}</h3>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="space-y-4">
            <label className="flex items-center justify-between rounded-lg border border-zinc-800 bg-black/60 px-3 py-2">
              <span>{t('settings.enableWebhook')}</span>
              <input
                type="checkbox"
                checked={webhookForm.enabled}
                onChange={(e) => setWebhookForm((previous) => ({ ...previous, enabled: e.target.checked }))}
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.webhookUrl')}</span>
              <input
                value={webhookForm.url}
                onChange={(e) => setWebhookForm((previous) => ({ ...previous, url: e.target.value }))}
                placeholder="https://discord.com/api/webhooks/..."
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.timeout')}</span>
              <input
                type="number"
                min={1000}
                value={webhookForm.timeoutMs}
                onChange={(e) => setWebhookForm((previous) => ({ ...previous, timeoutMs: Number(e.target.value) }))}
                className="w-full rounded-lg border border-zinc-700 bg-black px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            {!hasValidWebhook && <p className="text-sm text-red-400">{t('settings.webhookUrlRequired')}</p>}
          </div>

          <div className="space-y-4 rounded-lg border border-zinc-800 bg-black/60 p-4">
            <div className="flex items-center gap-3">
              <img src={fixedWebhookAvatar} alt="Webhook avatar" className="h-14 w-14 rounded object-cover" />
              <div>
                <p className="text-sm text-zinc-400">{t('settings.lockedIdentity')}</p>
                <p className="font-medium text-red-200">{fixedWebhookUsername}</p>
              </div>
            </div>
            <div className="rounded-lg border border-zinc-800 bg-zinc-950/80 p-3 text-sm text-zinc-300">
              {t('settings.lockedIdentityInfo')}
            </div>
            <label className="space-y-2">
              <span className="text-sm text-zinc-300">{t('settings.testUsername')}</span>
              <input
                value={testWebhookUsername}
                onChange={(e) => setTestWebhookUsername(e.target.value)}
                className="w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 outline-none focus:border-red-500"
              />
            </label>
            <button
              onClick={onSendTest}
              disabled={sendingTest}
              className="w-full rounded-lg border border-red-500/60 bg-red-700/90 px-4 py-2 font-medium text-white transition hover:bg-red-600 disabled:opacity-60"
            >
              {sendingTest ? t('settings.sending') : t('settings.sendTest')}
            </button>
            <p className="text-xs text-zinc-400">{t('settings.webhookHelp')}</p>
          </div>
        </div>
      </div>

      <div className="neon-panel rounded-2xl border border-zinc-800 bg-zinc-950/95 p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="text-sm text-zinc-300">{t('settings.summary')}</div>
          <button
            onClick={onSave}
            disabled={saving || !canSave}
            className="rounded-lg border border-red-500/60 bg-red-700/90 px-5 py-2 font-medium text-white transition hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? t('settings.saving') : t('settings.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
