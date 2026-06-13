import { type FormEvent, useEffect, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { ensureAuth, fileUrl, pb, reprocessDocument, type ReprocessMode } from '../lib/pocketbase'
import type { DocumentRecord, ProcessingJobRecord } from '../lib/pocketbase'

export function DocumentDetailPage() {
  const { documentId } = useParams({ from: '/documents/$documentId' })
  const [document, setDocument] = useState<DocumentRecord | null>(null)
  const [job, setJob] = useState<ProcessingJobRecord | null>(null)
  const [tagInput, setTagInput] = useState('')
  const [documentTypeInput, setDocumentTypeInput] = useState('')
  const [correspondentInput, setCorrespondentInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [reprocessing, setReprocessing] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')

  useEffect(() => {
    let active = true

    async function load() {
      try {
        setLoading(true)
        await ensureAuth()

        const doc = await pb.collection('documents').getOne<DocumentRecord>(documentId, {
          expand: 'tags,document_type,correspondent',
        })

        const jobs = await pb.collection('processing_jobs').getList<ProcessingJobRecord>(1, 1, {
          filter: `document = "${documentId}"`,
          sort: '-created',
        })

        if (active) {
          setDocument(doc)
          setJob(jobs.items[0] ?? null)
          setTagInput((doc.expand?.tags ?? []).map((tag) => tag.name).join(', '))
          setDocumentTypeInput(doc.expand?.document_type?.name ?? '')
          setCorrespondentInput(doc.expand?.correspondent?.name ?? '')
          setError('')
        }
      } catch (err) {
        if (active) {
          setError(err instanceof Error ? err.message : 'Failed to load document')
        }
      } finally {
        if (active) {
          setLoading(false)
        }
      }
    }

    load()

    let unsubscribe: (() => void) | undefined

    void pb.collection('documents').subscribe(documentId, () => {
      load()
    }).then((fn) => {
      unsubscribe = fn
    })

    return () => {
      active = false
      unsubscribe?.()
    }
  }, [documentId])

  const canReprocess =
    document?.processing_status !== 'processing' && document?.processing_status !== 'pending'

  const canReprocessExtraction = canReprocess && Boolean(document?.ocr_text?.trim())

  async function onReprocess(mode: ReprocessMode) {
    if (!document || !canReprocess) {
      return
    }
    if (mode === 'extraction' && !canReprocessExtraction) {
      return
    }

    try {
      setReprocessing(true)
      setMessage('')
      setError('')

      await reprocessDocument(document.id, mode)

      const doc = await pb.collection('documents').getOne<DocumentRecord>(document.id, {
        expand: 'tags,document_type,correspondent',
      })
      const jobs = await pb.collection('processing_jobs').getList<ProcessingJobRecord>(1, 1, {
        filter: `document = "${document.id}"`,
        sort: '-created',
      })

      setDocument(doc)
      setJob(jobs.items[0] ?? null)
      setMessage(
        mode === 'full'
          ? 'Document queued for full reprocessing (with OCR).'
          : 'Document queued for extraction reprocessing.',
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reprocess document')
    } finally {
      setReprocessing(false)
    }
  }

  async function onSave(event: FormEvent) {
    event.preventDefault()
    if (!document) {
      return
    }

    try {
      setSaving(true)
      setMessage('')
      setError('')

      const tagNames = tagInput
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean)

      const tagIds: string[] = []
      for (const name of tagNames) {
        const existing = await pb.collection('tags').getList(1, 1, {
          filter: `name = "${name.replace(/"/g, '\\"')}"`,
        })
        if (existing.items.length > 0) {
          tagIds.push(existing.items[0].id)
        } else {
          const created = await pb.collection('tags').create({ name })
          tagIds.push(created.id)
        }
      }

      let documentTypeId = ''
      const documentTypeName = documentTypeInput.trim()
      if (documentTypeName) {
        const existing = await pb.collection('document_types').getList(1, 1, {
          filter: `name = "${documentTypeName.replace(/"/g, '\\"')}"`,
        })
        if (existing.items.length > 0) {
          documentTypeId = existing.items[0].id
        } else {
          const created = await pb.collection('document_types').create({ name: documentTypeName })
          documentTypeId = created.id
        }
      }

      let correspondentId = ''
      const correspondentName = correspondentInput.trim()
      if (correspondentName) {
        const existing = await pb.collection('correspondents').getList(1, 1, {
          filter: `name = "${correspondentName.replace(/"/g, '\\"')}"`,
        })
        if (existing.items.length > 0) {
          correspondentId = existing.items[0].id
        } else {
          const created = await pb.collection('correspondents').create({
            name: correspondentName,
            name_original: correspondentName,
          })
          correspondentId = created.id
        }
      }

      const updated = await pb.collection('documents').update<DocumentRecord>(document.id, {
        title: document.title,
        purpose: document.purpose,
        document_date: document.document_date || null,
        document_type: documentTypeId || null,
        correspondent: correspondentId || null,
        summary: document.summary,
        tags: tagIds,
        metadata_source: 'user',
        processing_status:
          document.processing_status === 'needs_review' ? 'completed' : document.processing_status,
      })

      setDocument(updated)
      setMessage('Metadata saved.')
      const refreshed = await pb.collection('documents').getOne<DocumentRecord>(document.id, {
        expand: 'tags,document_type,correspondent',
      })
      setDocument(refreshed)
      setDocumentTypeInput(refreshed.expand?.document_type?.name ?? '')
      setCorrespondentInput(refreshed.expand?.correspondent?.name ?? '')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save metadata')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-gray-500">Loading document...</p>
  }

  if (!document) {
    return (
      <section className="flex flex-col gap-3">
        <p className="text-sm text-red-600">{error || 'Document not found.'}</p>
        <Link to="/" className="text-sm font-medium text-gray-900 underline">
          Back to documents
        </Link>
      </section>
    )
  }

  return (
    <section className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link to="/" className="text-sm text-gray-500 hover:text-gray-900">
            &larr; Back to documents
          </Link>
          <h2 className="mt-1 text-xl font-semibold text-gray-900">
            {document.title || 'Untitled document'}
          </h2>
          <p className="text-sm text-gray-500">Status: {document.processing_status}</p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <select
            value=""
            onChange={(event) => {
              const mode = event.target.value as ReprocessMode
              if (mode) {
                void onReprocess(mode)
              }
            }}
            disabled={!canReprocess || reprocessing}
            className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 outline-none transition-colors hover:bg-gray-50 focus:border-gray-900 focus:ring-1 focus:ring-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <option value="" disabled>
              {reprocessing ? 'Reprocessing...' : 'Reprocess'}
            </option>
            <option value="full">Reprocess full (with OCR)</option>
            <option value="extraction" disabled={!canReprocessExtraction}>
              Reprocess extraction
            </option>
          </select>
          {document.file && (
            <a
              className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
              href={fileUrl(document)}
              target="_blank"
              rel="noreferrer"
            >
              Open file
            </a>
          )}
        </div>
      </div>

      {job && (
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h3 className="mb-3 text-sm font-semibold text-gray-900">Processing job</h3>
          <dl className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div>
              <dt className="text-xs text-gray-400">Status</dt>
              <dd className="text-sm text-gray-700">{job.status}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-400">Job type</dt>
              <dd className="text-sm text-gray-700">{job.job_type || 'full'}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-400">OCR provider</dt>
              <dd className="text-sm text-gray-700">{job.ocr_provider || 'n/a'}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-400">AI provider</dt>
              <dd className="text-sm text-gray-700">{job.ai_provider || 'n/a'}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-400">Prompt version</dt>
              <dd className="text-sm text-gray-700">{job.prompt_version || 'n/a'}</dd>
            </div>
            {job.error_message && (
              <div className="col-span-2 sm:col-span-4">
                <dt className="text-xs text-gray-400">Error</dt>
                <dd className="text-sm text-red-600">{job.error_message}</dd>
              </div>
            )}
          </dl>
        </div>
      )}

      <form
        className="grid grid-cols-1 gap-4 rounded-lg border border-gray-200 bg-white p-6 sm:grid-cols-2"
        onSubmit={onSave}
      >
        <label className={labelClass}>
          Title
          <input
            className={inputClass}
            value={document.title ?? ''}
            onChange={(event) => setDocument({ ...document, title: event.target.value })}
          />
          {document.title_original && document.title_original !== document.title && (
            <span className="text-xs font-normal text-gray-500">
              Original: {document.title_original}
            </span>
          )}
        </label>

        <label className={labelClass}>
          Document date
          <input
            type="date"
            className={inputClass}
            value={document.document_date?.slice(0, 10) ?? ''}
            onChange={(event) => setDocument({ ...document, document_date: event.target.value })}
          />
        </label>

        <label className={labelClass}>
          Document type
          <input
            className={inputClass}
            value={documentTypeInput}
            onChange={(event) => setDocumentTypeInput(event.target.value)}
          />
        </label>

        <label className={labelClass}>
          Correspondent
          <input
            className={inputClass}
            value={correspondentInput}
            onChange={(event) => setCorrespondentInput(event.target.value)}
          />
          {document.expand?.correspondent?.name_original &&
            document.expand.correspondent.name_original !== document.expand.correspondent.name && (
              <span className="text-xs font-normal text-gray-500">
                Original: {document.expand.correspondent.name_original}
              </span>
            )}
        </label>

        <label className={`${labelClass} sm:col-span-2`}>
          Purpose
          <input
            className={inputClass}
            value={document.purpose ?? ''}
            onChange={(event) => setDocument({ ...document, purpose: event.target.value })}
          />
          {document.purpose_original && document.purpose_original !== document.purpose && (
            <span className="text-xs font-normal text-gray-500">
              Original: {document.purpose_original}
            </span>
          )}
        </label>

        <label className={`${labelClass} sm:col-span-2`}>
          Tags (comma separated)
          <input
            className={inputClass}
            value={tagInput}
            onChange={(event) => setTagInput(event.target.value)}
          />
        </label>

        <label className={`${labelClass} sm:col-span-2`}>
          Summary
          <textarea
            rows={4}
            className={inputClass}
            value={document.summary ?? ''}
            onChange={(event) => setDocument({ ...document, summary: event.target.value })}
          />
          {document.summary_original && document.summary_original !== document.summary && (
            <span className="text-xs font-normal text-gray-500">
              Original: {document.summary_original}
            </span>
          )}
        </label>

        <label className={`${labelClass} sm:col-span-2`}>
          OCR text
          <textarea
            rows={10}
            readOnly
            className={`${inputClass} bg-gray-50 font-mono text-xs`}
            value={document.ocr_text ?? ''}
          />
        </label>

        <div className="flex items-center gap-4 sm:col-span-2">
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save corrections'}
          </button>
          {message && <p className="text-sm text-green-600">{message}</p>}
          {error && <p className="text-sm text-red-600">{error}</p>}
        </div>
      </form>
    </section>
  )
}

const labelClass = 'flex flex-col gap-1.5 text-sm font-medium text-gray-700'
const inputClass =
  'w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm font-normal text-gray-900 outline-none placeholder:text-gray-400 focus:border-gray-900 focus:ring-1 focus:ring-gray-900'
