import { type FormEvent, useEffect, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { ensureAuth, fileUrl, pb, reprocessDocument, type ReprocessMode } from '../lib/pocketbase'
import type { DocumentRecord, ProcessingJobRecord } from '../lib/pocketbase'

export function DocumentDetailPage() {
  const { documentId } = useParams({ from: '/document/$documentId' })
  const [document, setDocument] = useState<DocumentRecord | null>(null)
  const [job, setJob] = useState<ProcessingJobRecord | null>(null)
  const [tagInput, setTagInput] = useState('')
  const [documentTypeInput, setDocumentTypeInput] = useState('')
  const [correspondentInput, setCorrespondentInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [reprocessing, setReprocessing] = useState(false)
  const [showProcessingJob, setShowProcessingJob] = useState(false)
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

  function requestReprocess(mode: ReprocessMode) {
    const confirmed = window.confirm(
      mode === 'full'
        ? 'Run OCR and extraction again from the original file? Existing metadata may be overwritten.'
        : 'Re-run extraction using the current OCR text? Existing metadata may be overwritten.',
    )
    if (confirmed) {
      void onReprocess(mode)
    }
  }

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
    if (!document || !editing) {
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
          const created = await pb.collection('document_types').create({
            name: documentTypeName,
            name_original: documentTypeName,
          })
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
      setEditing(false)
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
          <Link
            to="/document/$documentId/ask"
            params={{ documentId }}
            aria-disabled={!document.ocr_text?.trim()}
            title={
              document.ocr_text?.trim()
                ? 'Ask questions about this document'
                : 'OCR text required before asking AI'
            }
            className={`rounded-md border px-4 py-2 text-sm font-medium transition-colors ${
              document.ocr_text?.trim()
                ? 'border-gray-900 bg-gray-900 text-white hover:bg-gray-700'
                : 'pointer-events-none border-gray-200 bg-gray-100 text-gray-400'
            }`}
          >
            Ask AI
          </Link>
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
          {job ? (
            <button
              type="button"
              onClick={() => setShowProcessingJob((visible) => !visible)}
              aria-label={showProcessingJob ? 'Hide processing job details' : 'Show processing job details'}
              aria-pressed={showProcessingJob}
              title={showProcessingJob ? 'Hide processing job' : 'Show processing job'}
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-md border transition-colors cursor-pointer ${
                showProcessingJob
                  ? 'border-gray-900 bg-gray-900 text-white hover:bg-gray-700'
                  : 'border-gray-300 bg-white text-gray-500 hover:bg-gray-50 hover:text-gray-700'
              }`}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                className="h-4 w-4"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          ) : null}
        </div>
      </div>

      {job && showProcessingJob && (
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
              <dt className="text-xs text-gray-400">AI model</dt>
              <dd className="text-sm text-gray-700">{job.ai_model || 'n/a'}</dd>
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

          <div className="mt-4 flex flex-wrap gap-2 border-t border-gray-100 pt-4">
            <button
              type="button"
              onClick={() => requestReprocess('full')}
              disabled={!canReprocess || reprocessing}
              className="rounded-md border border-red-300 bg-red-50 px-3 py-1.5 text-sm font-medium text-red-700 transition-colors hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
            >
              {reprocessing ? 'Reprocessing...' : 'Reprocess full (with OCR)'}
            </button>
            <button
              type="button"
              onClick={() => requestReprocess('extraction')}
              disabled={!canReprocessExtraction || reprocessing}
              title={!canReprocessExtraction ? 'OCR text required' : undefined}
              className="rounded-md border border-amber-300 bg-amber-50 px-3 py-1.5 text-sm font-medium text-amber-700 transition-colors hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
            >
              {reprocessing ? 'Reprocessing...' : 'Reprocess extraction'}
            </button>
          </div>
        </div>
      )}

      <form
        className="grid grid-cols-1 gap-4 rounded-lg border border-gray-200 bg-white p-6 sm:grid-cols-2"
        onSubmit={onSave}
      >
        <label className={labelClass}>
          Title
          <input
            className={fieldClass(editing)}
            readOnly={!editing}
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
            className={fieldClass(editing)}
            readOnly={!editing}
            value={document.document_date?.slice(0, 10) ?? ''}
            onChange={(event) => setDocument({ ...document, document_date: event.target.value })}
          />
        </label>

        <label className={labelClass}>
          Document type
          <input
            className={fieldClass(editing)}
            readOnly={!editing}
            value={documentTypeInput}
            onChange={(event) => setDocumentTypeInput(event.target.value)}
          />
          {document.expand?.document_type?.name_original &&
            document.expand.document_type.name_original !== document.expand.document_type.name && (
              <span className="text-xs font-normal text-gray-500">
                Original: {document.expand.document_type.name_original}
              </span>
            )}
        </label>

        <label className={labelClass}>
          Correspondent
          <input
            className={fieldClass(editing)}
            readOnly={!editing}
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
            className={fieldClass(editing)}
            readOnly={!editing}
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
            className={fieldClass(editing)}
            readOnly={!editing}
            value={tagInput}
            onChange={(event) => setTagInput(event.target.value)}
          />
        </label>

        <label className={`${labelClass} sm:col-span-2`}>
          Summary
          <textarea
            rows={8}
            className={textareaClass(editing)}
            readOnly={!editing}
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
            rows={18}
            readOnly
            className={`${textareaClass(false)} min-h-96 font-mono text-xs leading-relaxed cursor-not-allowed`}
            value={document.ocr_text ?? ''}
          />
        </label>

        <div className="flex items-center gap-4 sm:col-span-2">
          {editing ? (
            <button
              type="submit"
              disabled={saving}
              className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
            >
              {saving ? 'Saving...' : 'Save corrections'}
            </button>
          ) : (
            <button
              type="button"
              onClick={(event) => {
                event.preventDefault()
                setEditing(true)
              }}
              className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 cursor-pointer"
            >
              Unblock editing
            </button>
          )}
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
const readonlyClass =
  'cursor-default border-gray-200 bg-gray-100/70 text-gray-700 shadow-inner focus:border-gray-200 focus:ring-0'

function fieldClass(editing: boolean) {
  return editing ? inputClass : `${inputClass} ${readonlyClass}`
}

function textareaClass(editing: boolean) {
  return `${fieldClass(editing)} min-h-48 resize-y`
}
