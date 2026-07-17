import PocketBase from 'pocketbase'

const pbUrl = import.meta.env.VITE_POCKETBASE_URL ?? document.location.origin

export const pb = new PocketBase(pbUrl)
export const pbAdminUrl = `${pbUrl}/_/`

export type DocumentTypeRecord = {
  id: string
  name: string
  name_original: string
}

export type CorrespondentRecord = {
  id: string
  name: string
  name_original: string
}

export type DocumentRecord = {
  id: string
  collectionId: string
  collectionName: string
  created: string
  updated: string
  file: string
  user: string
  title: string
  title_original: string
  purpose: string
  purpose_original: string
  document_date: string
  document_type: string
  correspondent: string
  ocr_text: string
  summary: string
  summary_original: string
  processing_status: 'pending' | 'processing' | 'completed' | 'failed' | 'needs_review'
  metadata_source: string
  confidence: number
  people_or_organizations: string[]
  tags: string[]
  expand?: {
    tags?: TagRecord[]
    document_type?: DocumentTypeRecord
    correspondent?: CorrespondentRecord
  }
}

export type TagRecord = {
  id: string
  name: string
}

export type ProcessingStep = 'preview' | 'ocr' | 'extract_metadata' | 'apply_metadata'

export type StepRunRecord = {
  name: ProcessingStep
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
  attempts: number
  provider?: string
  model?: string
  prompt_version?: string
  started_at?: string
  finished_at?: string
  error?: string
}

export const FULL_PIPELINE_STEPS: ProcessingStep[] = [
  'preview',
  'ocr',
  'extract_metadata',
  'apply_metadata',
]

export const EXTRACTION_PIPELINE_STEPS: ProcessingStep[] = ['extract_metadata', 'apply_metadata']

export const PROCESSING_STEP_LABELS: Record<ProcessingStep, string> = {
  preview: 'Preview',
  ocr: 'OCR',
  extract_metadata: 'Extract metadata',
  apply_metadata: 'Apply metadata',
}

export const PROCESSING_STEP_DESCRIPTIONS: Record<ProcessingStep, string> = {
  preview: 'Regenerate the first-page preview image (PDF only)',
  ocr: 'Re-run text extraction on the original file',
  extract_metadata: 'Re-run AI metadata extraction from OCR text',
  apply_metadata: 'Write extracted metadata onto the document',
}

export function orderedProcessingSteps(selected: Iterable<ProcessingStep>): ProcessingStep[] {
  const chosen = new Set(selected)
  return FULL_PIPELINE_STEPS.filter((step) => chosen.has(step))
}

export function forceStepsForReprocess(steps: ProcessingStep[]): ProcessingStep[] {
  return steps.filter((step) => step !== 'apply_metadata')
}

export function defaultReprocessSteps(hasOcrText: boolean): ProcessingStep[] {
  return hasOcrText ? [...EXTRACTION_PIPELINE_STEPS] : [...FULL_PIPELINE_STEPS]
}

export type ProcessingJobRecord = {
  id: string
  document: string
  status: string
  steps: ProcessingStep[]
  step_runs?: StepRunRecord[]
  current_step?: string
  started_at: string
  finished_at: string
  created: string
  updated: string
}

export type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
}

export type OCRProviderInfo = {
  id: string
  name: string
}

export type OCRTestResult = {
  provider: string
  text: string
  char_count: number
  duration: string
}

export function fileUrl(record: DocumentRecord, filename?: string) {
  return pb.files.getURL(record, filename ?? record.file)
}

export async function reprocessDocument(
  documentId: string,
  steps: ProcessingStep[],
  forceSteps?: ProcessingStep[],
) {
  await ensureAuth()
  await pb.collection('documents').update(documentId, {
    processing_status: 'pending',
  })
  return pb.collection('processing_jobs').create({
    document: documentId,
    status: 'pending',
    steps,
    ...(forceSteps?.length ? { force_steps: forceSteps } : {}),
  })
}

export async function chatWithDocument(documentId: string, messages: ChatMessage[]) {
  await ensureAuth()

  const response = await fetch(`${pbUrl}/api/app/documents/${documentId}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: pb.authStore.token,
    },
    body: JSON.stringify({ messages }),
  })

  const data = (await response.json()) as { message?: ChatMessage; detail?: string }
  if (!response.ok) {
    throw new Error(data.detail ?? 'Failed to get AI response')
  }
  if (!data.message) {
    throw new Error('AI response was empty')
  }
  return data.message
}

export async function listOCRProviders() {
  await ensureAuth()

  const response = await fetch(`${pbUrl}/api/app/ocr/providers`, {
    headers: {
      Authorization: pb.authStore.token,
    },
  })

  const data = (await response.json()) as { providers?: OCRProviderInfo[]; detail?: string }
  if (!response.ok) {
    throw new Error(data.detail ?? 'Failed to load OCR providers')
  }
  return data.providers ?? []
}

export async function testOCR(file: File, provider: string) {
  await ensureAuth()

  const formData = new FormData()
  formData.append('file', file)
  formData.append('provider', provider)

  const response = await fetch(`${pbUrl}/api/app/ocr/test`, {
    method: 'POST',
    headers: {
      Authorization: pb.authStore.token,
    },
    body: formData,
  })

  const data = (await response.json()) as OCRTestResult & { detail?: string }
  if (!response.ok) {
    throw new Error(data.detail ?? 'OCR test failed')
  }
  return data
}

export class AuthRequiredError extends Error {
  constructor() {
    super('Authentication required')
    this.name = 'AuthRequiredError'
  }
}

export function hasDevCredentials() {
  const email = import.meta.env.VITE_DEV_USER_EMAIL
  const password = import.meta.env.VITE_DEV_USER_PASSWORD
  return Boolean(email && password)
}

export async function loginWithPassword(email: string, password: string) {
  try {
    await pb.collection('users').authWithPassword(email, password)
  } catch {
    await pb.collection('_superusers').authWithPassword(email, password)
  }
}

export function isSuperuser() {
  return pb.authStore.record?.collectionName === '_superusers'
}

export function logout() {
  pb.authStore.clear()
}

export function getUserDisplayName() {
  const record = pb.authStore.record
  if (!record) {
    return ''
  }

  const name = typeof record.name === 'string' ? record.name.trim() : ''
  const email = typeof record.email === 'string' ? record.email.trim() : ''
  return name || email
}

export type AppSettings = {
  ocr_provider: string
  google_vision_api_key_set: boolean
  mistral_api_key_set: boolean
  mistral_ocr_model: string
  mistral_api_base_url: string
  ocr_timeout_sec: number
  processing_result_language: string
  openai_api_key_set: boolean
  openai_model: string
  openai_chat_model: string
  openai_base_url: string
  openai_timeout_sec: number
  worker_timeout_sec: number
  worker_max_retries: number
  extraction_prompt_version: string
}

export type AppSettingsPatch = {
  ocr_provider?: string
  google_vision_api_key?: string
  mistral_api_key?: string
  mistral_ocr_model?: string
  mistral_api_base_url?: string
  ocr_timeout_sec?: number
  processing_result_language?: string
  openai_api_key?: string
  openai_model?: string
  openai_chat_model?: string
  openai_base_url?: string
  openai_timeout_sec?: number
  worker_timeout_sec?: number
  worker_max_retries?: number
  extraction_prompt_version?: string
}

export async function getAppSettings() {
  await ensureAuth()

  const response = await fetch(`${pbUrl}/api/app/settings`, {
    headers: {
      Authorization: pb.authStore.token,
    },
  })

  const data = (await response.json()) as AppSettings & { detail?: string }
  if (!response.ok) {
    throw new Error(data.detail ?? 'Failed to load settings')
  }
  return data as AppSettings
}

export async function updateAppSettings(patch: AppSettingsPatch) {
  await ensureAuth()

  const response = await fetch(`${pbUrl}/api/app/settings`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      Authorization: pb.authStore.token,
    },
    body: JSON.stringify(patch),
  })

  const data = (await response.json()) as AppSettings & { detail?: string }
  if (!response.ok) {
    throw new Error(data.detail ?? 'Failed to save settings')
  }
  return data as AppSettings
}

export async function ensureAuth() {
  if (pb.authStore.isValid) {
    return
  }

  const email = import.meta.env.VITE_DEV_USER_EMAIL
  const password = import.meta.env.VITE_DEV_USER_PASSWORD

  if (!email || !password) {
    throw new AuthRequiredError()
  }

  try {
    await pb.collection('users').authWithPassword(email, password)
  } catch {
    throw new AuthRequiredError()
  }
}
