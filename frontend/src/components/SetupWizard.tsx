import { type SubmitEvent, useEffect, useState } from 'react'
import {
  accentContrastText,
  createSetupAdmin,
  getAppSettings,
  getSetupStatus,
  loginWithPassword,
  updateAppSettings,
  type SetupStatus,
} from '../lib/pocketbase'
import { AppFooter } from './AppFooter'

type SetupWizardProps = {
  appName: string
  accent: string
  initialStatus: SetupStatus
  onComplete: () => void
}

type Step = 'admin' | 'ocr' | 'openai' | 'done'

const inputClassName =
  'w-full rounded-md border border-stone-300 bg-stone-50 px-3 py-2 text-sm outline-none focus:border-gray-900 focus:ring-1 focus:ring-gray-900'
const labelClassName = 'flex flex-col gap-1'
const labelTextClassName = 'text-xs font-medium text-stone-500'

function initialStep(status: SetupStatus): Step {
  if (status.needs_admin) return 'admin'
  if (status.needs_config) {
    const ocrReady =
      status.ocr_provider === 'mistral'
        ? status.mistral_api_key_set
        : status.google_vision_api_key_set
    if (!ocrReady) return 'ocr'
    return 'openai'
  }
  return 'done'
}

export function SetupWizard({ appName, accent, initialStatus, onComplete }: SetupWizardProps) {
  const appInitial = appName.trim().charAt(0).toUpperCase() || 'P'
  const logoStyle = { backgroundColor: accent, color: accentContrastText(accent) }

  const [step, setStep] = useState<Step>(() => initialStep(initialStatus))
  const [status, setStatus] = useState(initialStatus)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [showAdvanced, setShowAdvanced] = useState(false)

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirm, setPasswordConfirm] = useState('')

  const [ocrProvider, setOcrProvider] = useState(initialStatus.ocr_provider || 'google_vision')
  const [googleKey, setGoogleKey] = useState('')
  const [mistralKey, setMistralKey] = useState('')
  const [mistralModel, setMistralModel] = useState('mistral-ocr-latest')
  const [mistralBaseURL, setMistralBaseURL] = useState('https://api.mistral.ai/v1')
  const [googleKeySet, setGoogleKeySet] = useState(initialStatus.google_vision_api_key_set)
  const [mistralKeySet, setMistralKeySet] = useState(initialStatus.mistral_api_key_set)

  const [openaiKey, setOpenaiKey] = useState('')
  const [openaiKeySet, setOpenaiKeySet] = useState(initialStatus.openai_api_key_set)
  const [openaiModel, setOpenaiModel] = useState('gpt-4o-mini')
  const [openaiBaseURL, setOpenaiBaseURL] = useState('https://api.openai.com/v1')

  useEffect(() => {
    if (step === 'admin') return

    let active = true
    async function loadSettings() {
      try {
        const settings = await getAppSettings()
        if (!active) return
        setOcrProvider(settings.ocr_provider || 'google_vision')
        setMistralModel(settings.mistral_ocr_model || 'mistral-ocr-latest')
        setMistralBaseURL(settings.mistral_api_base_url || 'https://api.mistral.ai/v1')
        setGoogleKeySet(settings.google_vision_api_key_set)
        setMistralKeySet(settings.mistral_api_key_set)
        setOpenaiKeySet(settings.openai_api_key_set)
        setOpenaiModel(settings.openai_model || 'gpt-4o-mini')
        setOpenaiBaseURL(settings.openai_base_url || 'https://api.openai.com/v1')
      } catch {
        // Prefill is best-effort; form still works with defaults.
      }
    }
    void loadSettings()
    return () => {
      active = false
    }
  }, [step])

  async function refreshStatus() {
    const next = await getSetupStatus()
    setStatus(next)
    return next
  }

  async function onCreateAdmin(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    try {
      setSubmitting(true)
      setError('')
      if (password !== passwordConfirm) {
        throw new Error('Passwords do not match.')
      }
      await createSetupAdmin(email.trim(), password, passwordConfirm)
      await loginWithPassword(email.trim(), password)
      const next = await refreshStatus()
      if (!next.needs_config) {
        setStep('done')
        return
      }
      const ocrReady =
        next.ocr_provider === 'mistral' ? next.mistral_api_key_set : next.google_vision_api_key_set
      setStep(ocrReady ? 'openai' : 'ocr')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create admin')
    } finally {
      setSubmitting(false)
    }
  }

  async function onSaveOCR(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    try {
      setSubmitting(true)
      setError('')

      const providerReady =
        ocrProvider === 'mistral' ? mistralKeySet || mistralKey.trim() !== '' : googleKeySet || googleKey.trim() !== ''
      if (!providerReady) {
        throw new Error('Enter an API key for the selected OCR provider.')
      }

      const patch: Parameters<typeof updateAppSettings>[0] = {
        ocr_provider: ocrProvider,
      }
      if (ocrProvider === 'mistral') {
        if (mistralKey.trim()) patch.mistral_api_key = mistralKey.trim()
        patch.mistral_ocr_model = mistralModel.trim()
        patch.mistral_api_base_url = mistralBaseURL.trim()
      } else if (googleKey.trim()) {
        patch.google_vision_api_key = googleKey.trim()
      }

      const settings = await updateAppSettings(patch)
      setGoogleKeySet(settings.google_vision_api_key_set)
      setMistralKeySet(settings.mistral_api_key_set)
      setGoogleKey('')
      setMistralKey('')

      const next = await refreshStatus()
      if (next.openai_api_key_set) {
        setStep(next.needs_config ? 'openai' : 'done')
      } else {
        setStep('openai')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save OCR settings')
    } finally {
      setSubmitting(false)
    }
  }

  async function onSaveOpenAI(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault()
    try {
      setSubmitting(true)
      setError('')

      if (!openaiKeySet && openaiKey.trim() === '') {
        throw new Error('Enter an OpenAI API key.')
      }

      const patch: Parameters<typeof updateAppSettings>[0] = {}
      if (openaiKey.trim()) patch.openai_api_key = openaiKey.trim()
      if (showAdvanced) {
        patch.openai_model = openaiModel.trim()
        patch.openai_base_url = openaiBaseURL.trim()
      }

      const settings = await updateAppSettings(patch)
      setOpenaiKeySet(settings.openai_api_key_set)
      setOpenaiKey('')

      const next = await refreshStatus()
      if (next.needs_config) {
        const ocrReady =
          next.ocr_provider === 'mistral' ? next.mistral_api_key_set : next.google_vision_api_key_set
        setStep(ocrReady ? 'openai' : 'ocr')
        setError('Setup is still incomplete. Check OCR and OpenAI keys.')
        return
      }
      setStep('done')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save OpenAI settings')
    } finally {
      setSubmitting(false)
    }
  }

  const stepLabel =
    step === 'admin'
      ? '1 · Admin account'
      : step === 'ocr'
        ? '2 · OCR'
        : step === 'openai'
          ? '3 · AI'
          : 'Ready'

  return (
    <div className="flex min-h-screen flex-col bg-stone-100">
      <div className="flex flex-1 items-center justify-center px-6 py-10">
        <section className="w-full max-w-md rounded-lg border border-stone-200 bg-stone-50 p-6 shadow-sm">
          <div className="mb-2 flex items-center gap-2">
            <span
              className="flex h-7 w-7 items-center justify-center rounded-md text-sm"
              style={logoStyle}
            >
              {appInitial}
            </span>
            <h1 className="text-lg font-semibold text-stone-950">{appName}</h1>
          </div>
          <p className="mb-1 text-xs font-medium uppercase tracking-wide text-stone-400">{stepLabel}</p>
          <h2 className="mb-4 text-base font-semibold text-stone-900">
            {step === 'admin' && 'Create your admin account'}
            {step === 'ocr' && 'Configure OCR'}
            {step === 'openai' && 'Configure AI'}
            {step === 'done' && 'Setup complete'}
          </h2>

          {step === 'admin' && (
            <form className="flex flex-col gap-4" onSubmit={onCreateAdmin}>
              <p className="text-sm text-stone-600">
                This account manages settings and can access PocketBase Admin.
              </p>
              <label className={labelClassName}>
                <span className={labelTextClassName}>Email</span>
                <input
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className={inputClassName}
                />
              </label>
              <label className={labelClassName}>
                <span className={labelTextClassName}>Password</span>
                <input
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={inputClassName}
                />
              </label>
              <label className={labelClassName}>
                <span className={labelTextClassName}>Confirm password</span>
                <input
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={8}
                  value={passwordConfirm}
                  onChange={(e) => setPasswordConfirm(e.target.value)}
                  className={inputClassName}
                />
              </label>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <button
                type="submit"
                disabled={submitting}
                className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {submitting ? 'Creating...' : 'Create admin'}
              </button>
            </form>
          )}

          {step === 'ocr' && (
            <form className="flex flex-col gap-4" onSubmit={onSaveOCR}>
              <p className="text-sm text-stone-600">
                Choose a provider and add its API key. Required for document processing.
              </p>
              <label className={labelClassName}>
                <span className={labelTextClassName}>OCR provider</span>
                <select
                  value={ocrProvider}
                  onChange={(e) => setOcrProvider(e.target.value)}
                  className={inputClassName}
                >
                  <option value="google_vision">Google Cloud Vision</option>
                  <option value="mistral">Mistral OCR</option>
                </select>
              </label>
              {ocrProvider === 'google_vision' ? (
                <label className={labelClassName}>
                  <span className={labelTextClassName}>
                    Google Vision API key{googleKeySet ? ' (set)' : ''}
                  </span>
                  <input
                    type="password"
                    autoComplete="off"
                    placeholder={googleKeySet ? 'Leave blank to keep' : 'Required'}
                    required={!googleKeySet}
                    value={googleKey}
                    onChange={(e) => setGoogleKey(e.target.value)}
                    className={inputClassName}
                  />
                </label>
              ) : (
                <>
                  <label className={labelClassName}>
                    <span className={labelTextClassName}>
                      Mistral API key{mistralKeySet ? ' (set)' : ''}
                    </span>
                    <input
                      type="password"
                      autoComplete="off"
                      placeholder={mistralKeySet ? 'Leave blank to keep' : 'Required'}
                      required={!mistralKeySet}
                      value={mistralKey}
                      onChange={(e) => setMistralKey(e.target.value)}
                      className={inputClassName}
                    />
                  </label>
                  <label className={labelClassName}>
                    <span className={labelTextClassName}>Mistral OCR model</span>
                    <input
                      type="text"
                      value={mistralModel}
                      onChange={(e) => setMistralModel(e.target.value)}
                      className={inputClassName}
                    />
                  </label>
                  <label className={labelClassName}>
                    <span className={labelTextClassName}>Mistral API base URL</span>
                    <input
                      type="url"
                      value={mistralBaseURL}
                      onChange={(e) => setMistralBaseURL(e.target.value)}
                      className={inputClassName}
                    />
                  </label>
                </>
              )}
              {error && <p className="text-sm text-red-600">{error}</p>}
              <button
                type="submit"
                disabled={submitting}
                className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {submitting ? 'Saving...' : 'Continue'}
              </button>
            </form>
          )}

          {step === 'openai' && (
            <form className="flex flex-col gap-4" onSubmit={onSaveOpenAI}>
              <p className="text-sm text-stone-600">
                An OpenAI-compatible API key powers metadata extraction, chat, and Deep Search.
              </p>
              <label className={labelClassName}>
                <span className={labelTextClassName}>
                  OpenAI API key{openaiKeySet ? ' (set)' : ''}
                </span>
                <input
                  type="password"
                  autoComplete="off"
                  placeholder={openaiKeySet ? 'Leave blank to keep' : 'Required'}
                  required={!openaiKeySet}
                  value={openaiKey}
                  onChange={(e) => setOpenaiKey(e.target.value)}
                  className={inputClassName}
                />
              </label>
              <button
                type="button"
                className="text-left text-xs font-medium text-stone-500 hover:text-stone-800"
                onClick={() => setShowAdvanced((v) => !v)}
              >
                {showAdvanced ? 'Hide advanced' : 'Show advanced'}
              </button>
              {showAdvanced && (
                <>
                  <label className={labelClassName}>
                    <span className={labelTextClassName}>Extraction model</span>
                    <input
                      type="text"
                      value={openaiModel}
                      onChange={(e) => setOpenaiModel(e.target.value)}
                      className={inputClassName}
                    />
                  </label>
                  <label className={labelClassName}>
                    <span className={labelTextClassName}>API base URL</span>
                    <input
                      type="url"
                      value={openaiBaseURL}
                      onChange={(e) => setOpenaiBaseURL(e.target.value)}
                      className={inputClassName}
                    />
                  </label>
                </>
              )}
              {error && <p className="text-sm text-red-600">{error}</p>}
              <button
                type="submit"
                disabled={submitting}
                className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {submitting ? 'Saving...' : 'Finish setup'}
              </button>
            </form>
          )}

          {step === 'done' && (
            <div className="flex flex-col gap-4">
              <p className="text-sm text-stone-600">
                Your admin account and processing keys are ready. You can change them anytime in
                Settings.
              </p>
              {status.needs_config && (
                <p className="text-sm text-red-600">Setup still reports missing configuration.</p>
              )}
              <button
                type="button"
                onClick={onComplete}
                className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700"
              >
                Open {appName}
              </button>
            </div>
          )}
        </section>
      </div>
      <AppFooter />
    </div>
  )
}

type SetupBlockedProps = {
  appName: string
  accent: string
  onLogout: () => void
}

export function SetupBlocked({ appName, accent, onLogout }: SetupBlockedProps) {
  const appInitial = appName.trim().charAt(0).toUpperCase() || 'P'
  const logoStyle = { backgroundColor: accent, color: accentContrastText(accent) }

  return (
    <div className="flex min-h-screen flex-col bg-stone-100">
      <div className="flex flex-1 items-center justify-center px-6">
        <section className="w-full max-w-sm rounded-lg border border-stone-200 bg-stone-50 p-6 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <span
              className="flex h-7 w-7 items-center justify-center rounded-md text-sm"
              style={logoStyle}
            >
              {appInitial}
            </span>
            <h1 className="text-lg font-semibold text-stone-950">{appName}</h1>
          </div>
          <h2 className="mb-2 text-base font-semibold text-stone-900">Setup incomplete</h2>
          <p className="mb-4 text-sm text-stone-600">
            An administrator must finish first-launch configuration before the app can be used.
          </p>
          <button
            type="button"
            onClick={onLogout}
            className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700"
          >
            Log out
          </button>
        </section>
      </div>
      <AppFooter />
    </div>
  )
}
