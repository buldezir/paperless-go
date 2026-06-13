import PocketBase from 'pocketbase'

const pbUrl = import.meta.env.VITE_POCKETBASE_URL ?? 'http://127.0.0.1:8090'

export const pb = new PocketBase(pbUrl)

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

export type ProcessingJobRecord = {
  id: string
  document: string
  status: string
  job_type: 'full' | 'extraction'
  retry_count: number
  ocr_provider: string
  ai_provider: string
  ai_model: string
  prompt_version: string
  error_message: string
  started_at: string
  finished_at: string
  created: string
  updated: string
}

export type ReprocessMode = 'full' | 'extraction'

export function fileUrl(record: DocumentRecord, filename?: string) {
  return pb.files.getURL(record, filename ?? record.file)
}

export async function reprocessDocument(documentId: string, mode: ReprocessMode = 'full') {
  await ensureAuth()
  await pb.collection('documents').update(documentId, {
    processing_status: 'pending',
  })
  return pb.collection('processing_jobs').create({
    document: documentId,
    status: 'pending',
    job_type: mode,
  })
}

export async function ensureAuth() {
  if (pb.authStore.isValid) {
    return
  }

  const email = import.meta.env.VITE_DEV_USER_EMAIL ?? 'dev@paperless.local'
  const password = import.meta.env.VITE_DEV_USER_PASSWORD ?? 'devpassword123'

  try {
    await pb.collection('users').authWithPassword(email, password)
  } catch {
    await pb.collection('users').create({
      email,
      password,
      passwordConfirm: password,
      name: 'Developer',
    })
    await pb.collection('users').authWithPassword(email, password)
  }
}
