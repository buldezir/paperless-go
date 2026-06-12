import { type FormEvent, useEffect, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { ensureAuth, fileUrl, pb } from '../lib/pocketbase'
import type { DocumentRecord, ProcessingJobRecord } from '../lib/pocketbase'

export function DocumentDetailPage() {
  const { documentId } = useParams({ from: '/documents/$documentId' })
  const [document, setDocument] = useState<DocumentRecord | null>(null)
  const [job, setJob] = useState<ProcessingJobRecord | null>(null)
  const [tagInput, setTagInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')

  useEffect(() => {
    let active = true

    async function load() {
      try {
        setLoading(true)
        await ensureAuth()

        const doc = await pb.collection('documents').getOne<DocumentRecord>(documentId, {
          expand: 'tags',
        })

        const jobs = await pb.collection('processing_jobs').getList<ProcessingJobRecord>(1, 1, {
          filter: `document = "${documentId}"`,
          sort: '-created',
        })

        if (active) {
          setDocument(doc)
          setJob(jobs.items[0] ?? null)
          setTagInput((doc.expand?.tags ?? []).map((tag) => tag.name).join(', '))
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

      const updated = await pb.collection('documents').update<DocumentRecord>(document.id, {
        title: document.title,
        purpose: document.purpose,
        document_date: document.document_date || null,
        document_type: document.document_type,
        summary: document.summary,
        tags: tagIds,
        metadata_source: 'user',
        processing_status:
          document.processing_status === 'needs_review' ? 'completed' : document.processing_status,
      })

      setDocument(updated)
      setMessage('Metadata saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save metadata')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="muted">Loading document...</p>
  }

  if (!document) {
    return (
      <section className="panel">
        <p className="error">{error || 'Document not found.'}</p>
        <Link to="/">Back to documents</Link>
      </section>
    )
  }

  return (
    <section className="panel">
      <div className="panel-header">
        <div>
          <Link to="/" className="muted back-link">
            Back to documents
          </Link>
          <h2>{document.title || 'Untitled document'}</h2>
          <p className="muted">Status: {document.processing_status}</p>
        </div>
        {document.file && (
          <a className="button secondary" href={fileUrl(document)} target="_blank" rel="noreferrer">
            Open file
          </a>
        )}
      </div>

      {job && (
        <div className="job-card">
          <h3>Processing job</h3>
          <dl>
            <div>
              <dt>Status</dt>
              <dd>{job.status}</dd>
            </div>
            <div>
              <dt>OCR provider</dt>
              <dd>{job.ocr_provider || 'n/a'}</dd>
            </div>
            <div>
              <dt>AI provider</dt>
              <dd>{job.ai_provider || 'n/a'}</dd>
            </div>
            <div>
              <dt>Prompt version</dt>
              <dd>{job.prompt_version || 'n/a'}</dd>
            </div>
            {job.error_message && (
              <div>
                <dt>Error</dt>
                <dd className="error">{job.error_message}</dd>
              </div>
            )}
          </dl>
        </div>
      )}

      <form className="detail-grid" onSubmit={onSave}>
        <label>
          Title
          <input
            value={document.title ?? ''}
            onChange={(event) => setDocument({ ...document, title: event.target.value })}
          />
        </label>

        <label>
          Document date
          <input
            type="date"
            value={document.document_date?.slice(0, 10) ?? ''}
            onChange={(event) => setDocument({ ...document, document_date: event.target.value })}
          />
        </label>

        <label>
          Document type
          <input
            value={document.document_type ?? ''}
            onChange={(event) => setDocument({ ...document, document_type: event.target.value })}
          />
        </label>

        <label className="full-width">
          Purpose
          <input
            value={document.purpose ?? ''}
            onChange={(event) => setDocument({ ...document, purpose: event.target.value })}
          />
        </label>

        <label className="full-width">
          Tags (comma separated)
          <input value={tagInput} onChange={(event) => setTagInput(event.target.value)} />
        </label>

        <label className="full-width">
          Summary
          <textarea
            rows={4}
            value={document.summary ?? ''}
            onChange={(event) => setDocument({ ...document, summary: event.target.value })}
          />
        </label>

        <label className="full-width">
          OCR text
          <textarea rows={10} readOnly value={document.ocr_text ?? ''} />
        </label>

        <div className="full-width actions">
          {message && <p className="success">{message}</p>}
          {error && <p className="error">{error}</p>}
          <button type="submit" className="button" disabled={saving}>
            {saving ? 'Saving...' : 'Save corrections'}
          </button>
        </div>
      </form>
    </section>
  )
}
