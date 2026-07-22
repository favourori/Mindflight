import { type FormEvent, useEffect, useMemo, useState } from 'react'
import './App.css'

type Role = 'leadership' | 'airman'

type SessionUser = {
  username: string
  role: Role
}

type MetricCard = {
  label: string
  value: string
  delta: string
  tone: 'good' | 'warn' | 'alert' | 'info'
}

type TrendPoint = {
  day: string
  readiness: number
  completion: number
  stress: number
  sleep: number
  checkedIn: number
}

type BarItem = {
  label: string
  value: number
}

type TrainingModule = {
  id: string
  title: string
  category: string
  duration: number
  description: string
  progress: number
  status: string
}

type CoachInsight = {
  headline: string
  message: string
  recommendations: string[]
}

type WearableStatus = {
  connected: boolean
  source: string
  lastSync: string
  sleepHours: number
  strain: number
  note: string
}

type SupportChannel = {
  key: string
  title: string
  description: string
  availability: string
}

type SupportRequestSummary = {
  channel: string
  urgency: string
  status: string
  createdAt: string
}

type CrisisPayload = {
  options: SupportChannel[]
  notice: string
}

type PrivacySettings = {
  shareTrends: boolean
  allowPeerSupport: boolean
  allowLeadershipOutreach: boolean
  allowWearableSync: boolean
}

type PeerSupportPayload = {
  channels: SupportChannel[]
  requests: SupportRequestSummary[]
}

type ActionItem = {
  title: string
  detail: string
  owner: string
  status: string
}

type LeadershipDashboardResponse = {
  title: string
  subtitle: string
  generatedAt: string
  coverage: {
    totalAirmen: number
    checkedInToday: number
    completionRate: number
  }
  metrics: MetricCard[]
  trends: TrendPoint[]
  stressors: BarItem[]
  resources: BarItem[]
  supportRoutes: BarItem[]
  moduleAdoption: BarItem[]
  alerts: { title: string; detail: string; severity: 'low' | 'medium' | 'high' }[]
  actions: ActionItem[]
}

type AirmanDashboardResponse = {
  title: string
  subtitle: string
  generatedAt: string
  latestMood: number
  latestStress: number
  latestSleep: number
  trend: TrendPoint[]
  resources: BarItem[]
  tips: ActionItem[]
  modules: TrainingModule[]
  coach: CoachInsight
  wearable: WearableStatus
  peerSupport: PeerSupportPayload
  crisis: CrisisPayload
  privacy: PrivacySettings
}

type LoginFormState = {
  username: string
  password: string
}

type CheckinPayload = {
  mood: number
  stress: number
  sleep: number
  note: string
}

type SupportRequestPayload = {
  channel: string
  urgency: string
  note: string
}

type Theme = 'dark' | 'light'

const THEME_STORAGE_KEY = 'mindflight-theme'

function getInitialTheme(): Theme {
  if (typeof window === 'undefined') {
    return 'dark'
  }

  const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY)
  if (storedTheme === 'dark' || storedTheme === 'light') {
    return storedTheme
  }

  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

async function fetchJson<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init)
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(payload?.error ?? `Request failed (${response.status})`)
  }

  return (await response.json()) as T
}

function isAbortError(error: unknown) {
  return error instanceof DOMException
    ? error.name === 'AbortError'
    : error instanceof Error && /abort/i.test(error.message)
}

function formatPercent(value: number) {
  return `${Math.round(value)}%`
}

function formatDecimal(value: number) {
  return value.toFixed(1)
}

function formatTimestamp(value: string) {
  return new Date(value).toLocaleString([], { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
}

function buildPath(values: number[], width: number, height: number, padding = 18) {
  if (values.length === 0) {
    return ''
  }

  const min = Math.min(...values)
  const max = Math.max(...values)
  const span = max - min || 1
  const innerWidth = width - padding * 2
  const innerHeight = height - padding * 2

  return values
    .map((value, index) => {
      const x = padding + (index / (values.length - 1 || 1)) * innerWidth
      const y = padding + (1 - (value - min) / span) * innerHeight
      return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
    })
    .join(' ')
}

function maxBarValue(items: BarItem[]) {
  return Math.max(...items.map((item) => item.value), 1)
}

function Sparkline({ values, color }: { values: number[]; color: string }) {
  const path = useMemo(() => buildPath(values, 280, 72, 8), [values])

  return (
    <svg viewBox="0 0 280 72" className="sparkline" role="img" aria-hidden="true">
      <path d={path} fill="none" stroke={color} strokeWidth="3.5" strokeLinecap="round" />
    </svg>
  )
}

function TrendChart({ trends }: { trends: TrendPoint[] }) {
  const readinessPath = buildPath(trends.map((trend) => trend.readiness), 1000, 320)
  const completionPath = buildPath(trends.map((trend) => trend.completion), 1000, 320)
  const stressPath = buildPath(trends.map((trend) => trend.stress), 1000, 320)

  return (
    <section className="panel panel--wide">
      <div className="panel__header">
        <div>
          <p className="eyebrow">14-day operational trend</p>
          <h2>Leadership signal over time</h2>
        </div>
        <div className="legend">
          <span><i className="legend__swatch legend__swatch--readiness" />Readiness</span>
          <span><i className="legend__swatch legend__swatch--completion" />Check-in completion</span>
          <span><i className="legend__swatch legend__swatch--stress" />Stress</span>
        </div>
      </div>
      <div className="trend-chart">
        <svg viewBox="0 0 1000 320" className="trend-chart__svg" role="img" aria-label="Leadership trend chart">
          <defs>
            <linearGradient id="trend-fill" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0%" stopColor="rgba(109, 211, 189, 0.2)" />
              <stop offset="100%" stopColor="rgba(109, 211, 189, 0)" />
            </linearGradient>
          </defs>
          {[64, 128, 192, 256].map((y) => (
            <line key={y} x1="30" x2="970" y1={y} y2={y} className="trend-chart__grid" />
          ))}
          <path d={`${readinessPath} L 970 292 L 30 292 Z`} fill="url(#trend-fill)" opacity="0.75" />
          <path d={readinessPath} className="trend-chart__line trend-chart__line--readiness" />
          <path d={completionPath} className="trend-chart__line trend-chart__line--completion" />
          <path d={stressPath} className="trend-chart__line trend-chart__line--stress" />
        </svg>
      </div>
      <div className="trend-summary">
        {trends.slice(-4).map((trend) => (
          <article key={trend.day} className="trend-summary__item">
            <span>{trend.day}</span>
            <strong>{formatPercent(trend.readiness)}</strong>
            <small>{formatPercent(trend.completion)} checked in</small>
          </article>
        ))}
      </div>
    </section>
  )
}

function BarStack({ title, items, accent }: { title: string; items: BarItem[]; accent: string }) {
  const maxValue = maxBarValue(items)

  return (
    <section className="panel panel--narrow panel--stack">
      <div className="panel__header panel__header--stacked">
        <div>
          <p className="eyebrow">{title}</p>
          <h2>{title}</h2>
        </div>
      </div>
      <div className="bar-stack">
        {items.map((item) => (
          <div key={item.label} className="bar-stack__row">
            <div className="bar-stack__label">
              <span>{item.label}</span>
              <strong>{item.value}</strong>
            </div>
            <div className="bar-stack__track">
              <div className="bar-stack__fill" style={{ width: `${(item.value / maxValue) * 100}%`, background: accent }} />
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

function ThemeToggle({ theme, onToggle }: { theme: Theme; onToggle: () => void }) {
  return (
    <button type="button" className="theme-toggle" onClick={onToggle} aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}>
      <span>{theme === 'dark' ? 'Light mode' : 'Dark mode'}</span>
      <strong>{theme === 'dark' ? 'On' : 'Off'}</strong>
    </button>
  )
}

function LoginScreen({ onLogin, error, loading, theme, onToggleTheme }: { onLogin: (credentials: LoginFormState) => Promise<void>; error: string | null; loading: boolean; theme: Theme; onToggleTheme: () => void }) {
  const [form, setForm] = useState<LoginFormState>({ username: '', password: '' })

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await onLogin(form)
  }

  return (
    <main className="shell shell--gate">
      <section className="auth panel">
        <div className="auth__copy">
          <div className="auth__toolbar">
            <p className="eyebrow">MindFlight</p>
            <ThemeToggle theme={theme} onToggle={onToggleTheme} />
          </div>
          <h1>Sign in to continue</h1>
          <p className="hero__lede">Leadership and Airman views are separated by role and protected by a real session cookie.</p>
          <div className="auth__demo">
            <div>
              <span>Leadership demo</span>
              <strong>leadership / flight2026</strong>
            </div>
            <div>
              <span>Airman demo</span>
              <strong>airman / wingman2026</strong>
            </div>
          </div>
        </div>
        <form className="auth__form" onSubmit={(event) => void handleSubmit(event)}>
          <label>
            <span>Username</span>
            <input value={form.username} onChange={(event) => setForm((current) => ({ ...current, username: event.target.value }))} placeholder="leadership or airman" autoComplete="username" />
          </label>
          <label>
            <span>Password</span>
            <input type="password" value={form.password} onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))} placeholder="••••••••" autoComplete="current-password" />
          </label>
          {error ? <p className="auth__error">{error}</p> : null}
          <button type="submit" className="primary-action" disabled={loading}>{loading ? 'Signing in...' : 'Sign in'}</button>
        </form>
      </section>
    </main>
  )
}

function LeadershipDashboard({ dashboard, user, onLogout, theme, onToggleTheme }: { dashboard: LeadershipDashboardResponse; user: SessionUser; onLogout: () => Promise<void>; theme: Theme; onToggleTheme: () => void }) {
  const [showSupportingDetail, setShowSupportingDetail] = useState(false)
  const readinessSeries = dashboard.trends.map((trend) => trend.readiness)
  const completionSeries = dashboard.trends.map((trend) => trend.completion)
  const stressSeries = dashboard.trends.map((trend) => trend.stress)
  const primaryMetrics = dashboard.metrics.slice(0, 3)
  const secondaryMetrics = dashboard.metrics.slice(3)
  const priorityAlerts = dashboard.alerts.slice(0, 2)
  const priorityActions = dashboard.actions.slice(0, 2)

  return (
    <main className="shell">
      <header className="topbar panel">
        <div>
          <p className="eyebrow">Leadership view</p>
          <h1>{dashboard.title}</h1>
          <p className="topbar__lede">Aggregate-only readiness signals across the wing. Track trends, catch friction early, and direct support where the pattern says it matters.</p>
        </div>
        <div className="topbar__actions">
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
          <span className="user-chip">{user.username}</span>
          <button type="button" className="secondary-action" onClick={() => void onLogout()}>Logout</button>
        </div>
      </header>

      <section className="hero panel panel--hero">
        <div className="hero__copy">
          <p className="eyebrow">Leadership overview</p>
          <h2>Unit-wide status at a glance</h2>
          <p className="hero__lede">{dashboard.subtitle}</p>
          <div className="hero__stats">
            <div><span>Total Airmen</span><strong>{dashboard.coverage.totalAirmen}</strong></div>
            <div><span>Checked in today</span><strong>{dashboard.coverage.checkedInToday}</strong></div>
            <div><span>Completion</span><strong>{formatPercent(dashboard.coverage.completionRate)}</strong></div>
          </div>
        </div>
        <div className="hero__badge">
          <span>Updated</span>
          <strong>{new Date(dashboard.generatedAt).toLocaleString()}</strong>
          <p>Aggregate only. No individual Airman-level data is exposed here.</p>
        </div>
      </section>

      <section className="kpi-grid kpi-grid--leadership">
        {primaryMetrics.map((metric) => (
          <article key={metric.label} className={`kpi kpi--${metric.tone}`}>
            <span className="kpi__label">{metric.label}</span>
            <strong className="kpi__value">{metric.value}</strong>
            <span className="kpi__delta">{metric.delta}</span>
          </article>
        ))}
      </section>

      <section className="grid grid--dashboard grid--leadership-core">
        <TrendChart trends={dashboard.trends} />

        <section className="panel panel--narrow">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Leadership attention</p><h2>Immediate concerns</h2></div></div>
          <div className="alert-list">
            {priorityAlerts.map((alert) => (
              <article key={alert.title} className={`alert alert--${alert.severity}`}>
                <div className="alert__header">
                  <span>{alert.severity}</span>
                  <strong>{alert.title}</strong>
                </div>
                <p>{alert.detail}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="panel panel--narrow">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Recommended actions</p><h2>Next moves</h2></div></div>
          <div className="action-list">
            {priorityActions.map((action) => (
              <article key={action.title} className="action-card">
                <div><strong>{action.title}</strong><p>{action.detail}</p></div>
                <footer><span>{action.owner}</span><em>{action.status}</em></footer>
              </article>
            ))}
          </div>
        </section>

        <section className="panel panel--wide panel--supporting-detail">
          <div className="panel__header">
            <div>
              <p className="eyebrow">Supporting detail</p>
              <h2>Diagnostics and longer-tail signals</h2>
            </div>
            <button
              type="button"
              className="secondary-action"
              onClick={() => setShowSupportingDetail((current) => !current)}
              aria-expanded={showSupportingDetail}
            >
              {showSupportingDetail ? 'Hide detail' : 'Show detail'}
            </button>
          </div>

          {showSupportingDetail ? (
            <div className="supporting-detail">
              {secondaryMetrics.length > 0 ? (
                <section className="kpi-grid kpi-grid--supporting">
                  {secondaryMetrics.map((metric) => (
                    <article key={metric.label} className={`kpi kpi--${metric.tone}`}>
                      <span className="kpi__label">{metric.label}</span>
                      <strong className="kpi__value">{metric.value}</strong>
                      <span className="kpi__delta">{metric.delta}</span>
                    </article>
                  ))}
                </section>
              ) : null}

              <div className="supporting-detail__grid">
                <section className="panel panel--narrow panel--nested">
                  <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Rolling signals</p><h2>Readiness snapshot</h2></div></div>
                  <div className="snapshot-list">
                    <article><span>Readiness</span><strong>{formatPercent(readinessSeries[readinessSeries.length - 1] ?? 0)}</strong><Sparkline values={readinessSeries} color="var(--signal-good)" /></article>
                    <article><span>Completion</span><strong>{formatPercent(completionSeries[completionSeries.length - 1] ?? 0)}</strong><Sparkline values={completionSeries} color="var(--signal-info)" /></article>
                    <article><span>Stress</span><strong>{formatDecimal(stressSeries[stressSeries.length - 1] ?? 0)}</strong><Sparkline values={stressSeries} color="var(--signal-warn)" /></article>
                  </div>
                </section>

                <BarStack title="Top stressors" items={dashboard.stressors} accent="var(--signal-alert)" />
                <BarStack title="Resource engagement" items={dashboard.resources} accent="var(--signal-info)" />
                <BarStack title="Support routing" items={dashboard.supportRoutes} accent="var(--signal-good)" />
                <BarStack title="Training adoption" items={dashboard.moduleAdoption} accent="var(--signal-warn)" />
              </div>
            </div>
          ) : (
            <p className="helptext helptext--compact">Show the deeper breakdown only when you need to inspect secondary metrics, rolling snapshots, stressor mix, and resource engagement.</p>
          )}
        </section>
      </section>
    </main>
  )
}

function AirmanDashboard({ dashboard, user, onLogout, onRefresh, theme, onToggleTheme }: { dashboard: AirmanDashboardResponse; user: SessionUser; onLogout: () => Promise<void>; onRefresh: () => void; theme: Theme; onToggleTheme: () => void }) {
  const [checkin, setCheckin] = useState<CheckinPayload>({ mood: 3, stress: 3, sleep: 7, note: '' })
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [coachPrompt, setCoachPrompt] = useState('')
  const [coachInsight, setCoachInsight] = useState<CoachInsight>(dashboard.coach)
  const [coachLoading, setCoachLoading] = useState(false)
  const [supportRequest, setSupportRequest] = useState<SupportRequestPayload>({ channel: dashboard.peerSupport.channels[0]?.key ?? 'peer', urgency: 'medium', note: '' })
  const [supportMessage, setSupportMessage] = useState<string | null>(null)
  const [supportLoading, setSupportLoading] = useState(false)
  const [privacy, setPrivacy] = useState<PrivacySettings>(dashboard.privacy)
  const [privacyMessage, setPrivacyMessage] = useState<string | null>(null)
  const [privacyLoading, setPrivacyLoading] = useState(false)
  const [moduleUpdating, setModuleUpdating] = useState<string | null>(null)

  const trendReadiness = dashboard.trend.map((trend) => trend.readiness)
  const trendStress = dashboard.trend.map((trend) => trend.stress)

  useEffect(() => {
    setCoachInsight(dashboard.coach)
    setPrivacy(dashboard.privacy)
  }, [dashboard.coach, dashboard.privacy])

  async function submitCheckin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setMessage(null)

    try {
      await fetchJson<{ status: string }>('/api/checkins', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(checkin),
      })
      setCheckin({ mood: 3, stress: 3, sleep: 7, note: '' })
      setMessage('Check-in saved. Your dashboard refreshed.')
      onRefresh()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Unable to save check-in')
    } finally {
      setSaving(false)
    }
  }

  async function requestCoach(prompt = coachPrompt) {
    setCoachLoading(true)
    try {
      const insight = await fetchJson<CoachInsight>('/api/airman/coach', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt }),
      })
      setCoachInsight(insight)
    } finally {
      setCoachLoading(false)
    }
  }

  async function savePrivacy() {
    setPrivacyLoading(true)
    setPrivacyMessage(null)
    try {
      await fetchJson<PrivacySettings>('/api/airman/privacy', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(privacy),
      })
      setPrivacyMessage('Privacy preferences updated.')
    } catch (error) {
      setPrivacyMessage(error instanceof Error ? error.message : 'Unable to save privacy settings')
    } finally {
      setPrivacyLoading(false)
    }
  }

  async function sendSupportRequest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSupportLoading(true)
    setSupportMessage(null)
    try {
      await fetchJson<{ status: string }>('/api/airman/support-request', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(supportRequest),
      })
      setSupportMessage('Support request queued through the discreet routing path.')
      setSupportRequest((current) => ({ ...current, note: '' }))
      onRefresh()
    } catch (error) {
      setSupportMessage(error instanceof Error ? error.message : 'Unable to route support request')
    } finally {
      setSupportLoading(false)
    }
  }

  async function updateModuleProgress(module: TrainingModule) {
    const nextProgress = module.progress >= 100 ? 100 : Math.min(module.progress + 25, 100)
    setModuleUpdating(module.id)
    try {
      await fetchJson<{ status: string; progress: number }>('/api/airman/modules/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: module.id, progress: nextProgress }),
      })
      onRefresh()
    } finally {
      setModuleUpdating(null)
    }
  }

  return (
    <main className="shell">
      <header className="topbar panel">
        <div>
          <p className="eyebrow">Airman view</p>
          <h1>{dashboard.title}</h1>
          <p className="topbar__lede">Your private wellness surface with a quick check-in, personal trend, and resources.</p>
        </div>
        <div className="topbar__actions">
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
          <span className="user-chip">{user.username}</span>
          <button type="button" className="secondary-action" onClick={() => void onLogout()}>Logout</button>
        </div>
      </header>

      <section className="hero panel panel--hero panel--airman">
        <div className="hero__copy">
          <p className="eyebrow">Today's focus</p>
          <h2>Two-minute reset before the next shift</h2>
          <p className="hero__lede">Use the check-in to note stress, sleep, and energy. If the numbers feel off, the app points you to the next step without making it heavy.</p>
          <div className="cta-row">
            <button type="button" className="primary-action" onClick={() => document.getElementById('daily-checkin')?.scrollIntoView({ behavior: 'smooth', block: 'start' })}>Start check-in</button>
            <button type="button" className="secondary-action">Open resilience exercise</button>
          </div>
        </div>
        <div className="hero__badge hero__badge--stacked">
          <span>Latest mood</span>
          <strong>{formatDecimal(dashboard.latestMood)}</strong>
          <span>Stress</span>
          <strong>{formatDecimal(dashboard.latestStress)}</strong>
          <span>Sleep</span>
          <strong>{formatDecimal(dashboard.latestSleep)} h</strong>
        </div>
      </section>

      <section className="grid grid--dashboard grid--airman-extensions">
        <section className="panel panel--wide">
          <div className="panel__header">
            <div><p className="eyebrow">AI coaching</p><h2>Personal resilience coach</h2></div>
            <button type="button" className="secondary-action" onClick={() => void requestCoach('stress reset')} disabled={coachLoading}>{coachLoading ? 'Refreshing...' : 'Refresh guidance'}</button>
          </div>
          <div className="coach-card">
            <div>
              <span className="eyebrow">{coachInsight.headline}</span>
              <p className="coach-card__message">{coachInsight.message}</p>
            </div>
            <div className="coach-card__recommendations">
              {coachInsight.recommendations.map((item) => (
                <article key={item} className="support-tile">
                  <strong>{item}</strong>
                </article>
              ))}
            </div>
            <form className="coach-card__prompt" onSubmit={(event) => {
              event.preventDefault()
              void requestCoach()
            }}>
              <input value={coachPrompt} onChange={(event) => setCoachPrompt(event.target.value)} placeholder="Ask about stress, sleep, recovery, or focus" />
              <button type="submit" className="primary-action" disabled={coachLoading}>{coachLoading ? 'Thinking...' : 'Ask coach'}</button>
            </form>
          </div>
        </section>
      </section>

      <section className="grid grid--dashboard">
        <section className="panel panel--wide" id="daily-checkin">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Daily check-in</p><h2>Send a quick anonymous wellness pulse</h2></div></div>
          <form className="checkin-form" onSubmit={(event) => void submitCheckin(event)}>
            <label>
              <span>Mood</span>
              <input type="range" min="1" max="5" value={checkin.mood} onChange={(event) => setCheckin((current) => ({ ...current, mood: Number(event.target.value) }))} />
              <strong>{checkin.mood} / 5</strong>
            </label>
            <label>
              <span>Stress</span>
              <input type="range" min="1" max="5" value={checkin.stress} onChange={(event) => setCheckin((current) => ({ ...current, stress: Number(event.target.value) }))} />
              <strong>{checkin.stress} / 5</strong>
            </label>
            <label>
              <span>Sleep</span>
              <input type="range" min="0" max="12" step="0.5" value={checkin.sleep} onChange={(event) => setCheckin((current) => ({ ...current, sleep: Number(event.target.value) }))} />
              <strong>{formatDecimal(checkin.sleep)} h</strong>
            </label>
            <label className="checkin-form__note">
              <span>Notes for yourself</span>
              <textarea value={checkin.note} onChange={(event) => setCheckin((current) => ({ ...current, note: event.target.value }))} placeholder="One line about what is driving the score today" rows={4} />
            </label>
            <div className="checkin-form__footer">
              <button type="submit" className="primary-action" disabled={saving}>{saving ? 'Saving...' : 'Save check-in'}</button>
              <span>{message ?? 'Saved check-ins remain private and feed your own history.'}</span>
            </div>
          </form>
        </section>

        <section className="panel panel--narrow">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Personal trend</p><h2>Recent history</h2></div></div>
          <div className="snapshot-list">
            <article><span>Readiness</span><strong>{formatPercent(dashboard.trend[0]?.readiness ?? 0)}</strong><Sparkline values={trendReadiness} color="var(--signal-good)" /></article>
            <article><span>Stress</span><strong>{formatDecimal(dashboard.latestStress)}</strong><Sparkline values={trendStress} color="var(--signal-warn)" /></article>
            <article><span>Resources</span><strong>{dashboard.resources.length}</strong><small>Always available</small></article>
          </div>
        </section>

        <section className="panel panel--narrow">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Biometric pulse</p><h2>Wearable sync</h2></div></div>
          <div className="snapshot-list">
            <article><span>Connection</span><strong>{dashboard.wearable.connected ? dashboard.wearable.source : 'Not connected'}</strong><small>{dashboard.wearable.connected ? `Last sync ${formatTimestamp(dashboard.wearable.lastSync)}` : 'Connect a device to enrich your recovery picture.'}</small></article>
            <article><span>Sleep signal</span><strong>{formatDecimal(dashboard.wearable.sleepHours)} h</strong><small>Recovered from wearable sync</small></article>
            <article><span>Strain</span><strong>{formatDecimal(dashboard.wearable.strain)}</strong><small>{dashboard.wearable.note}</small></article>
          </div>
        </section>

        <section className="panel panel--wide">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Quick support</p><h2>Fast paths for a better day</h2></div></div>
          <div className="action-list action-list--compact">
            {dashboard.tips.map((tip) => (
              <article key={tip.title} className="action-card">
                <div><strong>{tip.title}</strong><p>{tip.detail}</p></div>
                <footer><span>{tip.owner}</span><em>{tip.status}</em></footer>
              </article>
            ))}
          </div>
        </section>

        <section className="panel panel--wide">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Resilience training</p><h2>Personalized learning modules</h2></div></div>
          <div className="module-grid">
            {dashboard.modules.map((module) => (
              <article key={module.id} className="module-card">
                <div className="module-card__top">
                  <span>{module.category}</span>
                  <strong>{module.status}</strong>
                </div>
                <h3>{module.title}</h3>
                <p>{module.description}</p>
                <div className="module-card__meta">
                  <span>{module.duration} min</span>
                  <span>{module.progress}% complete</span>
                </div>
                <div className="bar-stack__track"><div className="bar-stack__fill" style={{ width: `${module.progress}%`, background: 'var(--signal-good)' }} /></div>
                <button type="button" className="secondary-action" onClick={() => void updateModuleProgress(module)} disabled={moduleUpdating === module.id || module.progress >= 100}>
                  {module.progress >= 100 ? 'Completed' : moduleUpdating === module.id ? 'Saving...' : 'Advance module'}
                </button>
              </article>
            ))}
          </div>
        </section>

        <section className="panel panel--wide">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Human support</p><h2>Peer, chaplain, and crisis routing</h2></div></div>
          <div className="support-grid">
            <div className="support-grid__channels">
              {dashboard.peerSupport.channels.map((channel) => (
                <article key={channel.key} className="support-tile">
                  <span>{channel.availability}</span>
                  <strong>{channel.title}</strong>
                  <p>{channel.description}</p>
                </article>
              ))}
              {dashboard.crisis.options.map((option) => (
                <article key={option.key} className="support-tile support-tile--alert">
                  <span>{option.availability}</span>
                  <strong>{option.title}</strong>
                  <p>{option.description}</p>
                </article>
              ))}
            </div>
            <form className="support-form" onSubmit={(event) => void sendSupportRequest(event)}>
              <label>
                <span>Channel</span>
                <select value={supportRequest.channel} onChange={(event) => setSupportRequest((current) => ({ ...current, channel: event.target.value }))}>
                  {dashboard.peerSupport.channels.map((channel) => <option key={channel.key} value={channel.key}>{channel.title}</option>)}
                </select>
              </label>
              <label>
                <span>Urgency</span>
                <select value={supportRequest.urgency} onChange={(event) => setSupportRequest((current) => ({ ...current, urgency: event.target.value }))}>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                </select>
              </label>
              <label>
                <span>Context</span>
                <textarea rows={4} value={supportRequest.note} onChange={(event) => setSupportRequest((current) => ({ ...current, note: event.target.value }))} placeholder="What kind of follow-up would help right now?" />
              </label>
              <div className="checkin-form__footer">
                <button type="submit" className="primary-action" disabled={supportLoading}>{supportLoading ? 'Routing...' : 'Request support'}</button>
                <span>{supportMessage ?? dashboard.crisis.notice}</span>
              </div>
              <div className="support-history">
                {dashboard.peerSupport.requests.map((request) => (
                  <article key={`${request.channel}-${request.createdAt}`} className="support-history__item">
                    <strong>{request.channel}</strong>
                    <span>{request.urgency} priority</span>
                    <small>{request.status} • {formatTimestamp(request.createdAt)}</small>
                  </article>
                ))}
              </div>
            </form>
          </div>
        </section>

        <section className="panel panel--wide">
          <div className="panel__header panel__header--stacked"><div><p className="eyebrow">Privacy controls</p><h2>Choose how your data helps you</h2></div></div>
          <div className="privacy-grid">
            <label className="toggle-card"><input type="checkbox" checked={privacy.shareTrends} onChange={(event) => setPrivacy((current) => ({ ...current, shareTrends: event.target.checked }))} /><div><strong>Share anonymous trends</strong><p>Allow your check-ins to contribute to aggregate unit wellness signals.</p></div></label>
            <label className="toggle-card"><input type="checkbox" checked={privacy.allowPeerSupport} onChange={(event) => setPrivacy((current) => ({ ...current, allowPeerSupport: event.target.checked }))} /><div><strong>Enable peer support routing</strong><p>Allow MindFlight to route voluntary peer support requests through trained wingmen.</p></div></label>
            <label className="toggle-card"><input type="checkbox" checked={privacy.allowLeadershipOutreach} onChange={(event) => setPrivacy((current) => ({ ...current, allowLeadershipOutreach: event.target.checked }))} /><div><strong>Allow leadership outreach offers</strong><p>Receive optional, non-punitive offers for follow-up when strain remains elevated.</p></div></label>
            <label className="toggle-card"><input type="checkbox" checked={privacy.allowWearableSync} onChange={(event) => setPrivacy((current) => ({ ...current, allowWearableSync: event.target.checked }))} /><div><strong>Enable wearable sync</strong><p>Use passive recovery signals to enrich your personal coaching.</p></div></label>
          </div>
          <div className="checkin-form__footer">
            <button type="button" className="secondary-action" onClick={() => void savePrivacy()} disabled={privacyLoading}>{privacyLoading ? 'Saving...' : 'Save privacy settings'}</button>
            <span>{privacyMessage ?? 'You control what stays personal and what contributes to anonymous readiness patterns.'}</span>
          </div>
        </section>
      </section>
    </main>
  )
}

function App() {
  const [user, setUser] = useState<SessionUser | null>(null)
  const [theme, setTheme] = useState<Theme>(getInitialTheme)
  const [loadingSession, setLoadingSession] = useState(true)
  const [loginLoading, setLoginLoading] = useState(false)
  const [authError, setAuthError] = useState<string | null>(null)
  const [dashboardError, setDashboardError] = useState<string | null>(null)
  const [loginSeed, setLoginSeed] = useState(0)
  const [dashboardSeed, setDashboardSeed] = useState(0)
  const [leadershipDashboard, setLeadershipDashboard] = useState<LeadershipDashboardResponse | null>(null)
  const [airmanDashboard, setAirmanDashboard] = useState<AirmanDashboardResponse | null>(null)

  useEffect(() => {
    const root = document.documentElement
    root.dataset.theme = theme
    root.style.colorScheme = theme
    window.localStorage.setItem(THEME_STORAGE_KEY, theme)
  }, [theme])

  useEffect(() => {
    const controller = new AbortController()

    async function loadSession() {
      try {
        const session = await fetchJson<SessionUser>('/api/auth/me', { signal: controller.signal })
        setUser(session)
      } catch (error) {
        if (isAbortError(error)) {
          return
        }

        setUser(null)
      } finally {
        if (!controller.signal.aborted) {
          setLoadingSession(false)
        }
      }
    }

    void loadSession()

    return () => controller.abort()
  }, [loginSeed])

  useEffect(() => {
    const currentUser = user

    if (!currentUser) {
      return
    }

    const currentRole = currentUser.role

    const controller = new AbortController()

    async function loadDashboard() {
      try {
        setDashboardError(null)
        if (currentRole === 'leadership') {
          const payload = await fetchJson<LeadershipDashboardResponse>('/api/leadership', { signal: controller.signal })
          setLeadershipDashboard(payload)
        } else {
          const payload = await fetchJson<AirmanDashboardResponse>('/api/airman', { signal: controller.signal })
          setAirmanDashboard(payload)
        }
      } catch (error) {
        if (isAbortError(error) || controller.signal.aborted) {
          return
        }

        setDashboardError(error instanceof Error ? error.message : 'Unable to load dashboard')
      }
    }

    void loadDashboard()

    return () => controller.abort()
  }, [user, dashboardSeed])

  async function handleLogin(credentials: LoginFormState) {
    setLoginLoading(true)
    setAuthError(null)

    try {
      const session = await fetchJson<SessionUser>('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentials),
      })
      setUser(session)
    } catch (error) {
      setAuthError(error instanceof Error ? error.message : 'Unable to sign in')
    } finally {
      setLoginLoading(false)
    }
  }

  async function handleLogout() {
    try {
      await fetchJson<{ status: string }>('/api/auth/logout', { method: 'POST' })
    } finally {
      setUser(null)
      setLeadershipDashboard(null)
      setAirmanDashboard(null)
      setDashboardError(null)
      setLoginSeed((value) => value + 1)
    }
  }

  if (loadingSession) {
    return (
      <main className="shell shell--loading">
        <section className="panel panel--error">
          <p className="eyebrow">MindFlight</p>
          <h1>Loading session...</h1>
        </section>
      </main>
    )
  }

  if (!user) {
    return <LoginScreen onLogin={handleLogin} error={authError} loading={loginLoading} theme={theme} onToggleTheme={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))} />
  }

  if (dashboardError) {
    return (
      <main className="shell shell--error">
        <section className="panel panel--error">
          <p className="eyebrow">MindFlight</p>
          <h1>Dashboard data could not load.</h1>
          <p>{dashboardError}</p>
          <button type="button" className="secondary-action" onClick={() => setDashboardSeed((value) => value + 1)}>Retry</button>
        </section>
      </main>
    )
  }

  if (user.role === 'leadership' && leadershipDashboard) {
    return <LeadershipDashboard dashboard={leadershipDashboard} user={user} onLogout={handleLogout} theme={theme} onToggleTheme={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))} />
  }

  if (user.role === 'airman' && airmanDashboard) {
    return <AirmanDashboard dashboard={airmanDashboard} user={user} onLogout={handleLogout} onRefresh={() => setDashboardSeed((value) => value + 1)} theme={theme} onToggleTheme={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))} />
  }

  return (
    <main className="shell shell--loading">
      <section className="panel panel--error">
        <p className="eyebrow">MindFlight</p>
        <h1>Loading dashboard...</h1>
      </section>
    </main>
  )
}

export default App
