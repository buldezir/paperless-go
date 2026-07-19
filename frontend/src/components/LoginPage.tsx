import { type SubmitEvent, useState } from 'react'
import { accentContrastText, loginWithPassword } from '../lib/pocketbase'
import { AppFooter } from './AppFooter'

type LoginPageProps = {
  appName: string
  accent: string
  onSuccess: () => void
}

const inputClassName =
  'w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2 text-sm outline-none focus:border-gray-900 focus:ring-1 focus:ring-gray-900'

export function LoginPage({ appName, accent, onSuccess }: LoginPageProps) {
  const appInitial = appName.trim().charAt(0).toUpperCase() || 'P'
  const logoStyle = { backgroundColor: accent, color: accentContrastText(accent) }
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()

    try {
      setSubmitting(true)
      setError('')
      await loginWithPassword(email, password)
      onSuccess()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col bg-stone-100">
      <div className="flex flex-1 items-center justify-center px-6">
        <section className="w-full max-w-sm rounded-lg border border-stone-200 bg-stone-50 p-6 shadow-sm">
          <div className="mb-6 flex items-center gap-2">
            <span
              className="flex h-7 w-7 items-center justify-center rounded-md text-sm"
              style={logoStyle}
            >
              {appInitial}
            </span>
            <h1 className="text-lg font-semibold text-stone-950">{appName}</h1>
          </div>

          <form className="flex flex-col gap-4" onSubmit={onSubmit}>
            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-stone-500">Email</span>
              <input
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className={inputClassName}
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-xs font-medium text-stone-500">Password</span>
              <input
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className={inputClassName}
              />
            </label>

            {error && <p className="text-sm text-red-600">{error}</p>}

            <button
              type="submit"
              disabled={submitting}
              className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting ? 'Signing in...' : 'Sign in'}
            </button>
          </form>
        </section>
      </div>
      <AppFooter />
    </div>
  )
}
