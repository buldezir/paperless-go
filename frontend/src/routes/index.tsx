import { useEffect, useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ensureAuth, pb } from '../lib/pocketbase'
import type { DocumentRecord } from '../lib/pocketbase'
import { DocumentCard } from '../components/DocumentCard'

export function IndexPage() {
  const [documents, setDocuments] = useState<DocumentRecord[]>([])
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true

    async function load() {
      try {
        setLoading(true)
        await ensureAuth()
        const result = await pb.collection('documents').getList<DocumentRecord>(1, 50, {
          sort: '-created',
          expand: 'tags',
        })
        if (active) {
          setDocuments(result.items)
          setError('')
        }
      } catch (err) {
        if (active) {
          setError(err instanceof Error ? err.message : 'Failed to load documents')
        }
      } finally {
        if (active) {
          setLoading(false)
        }
      }
    }

    load()

    let unsubscribe: (() => void) | undefined

    void pb.collection('documents').subscribe('*', () => {
      load()
    }).then((fn) => {
      unsubscribe = fn
    })

    return () => {
      active = false
      unsubscribe?.()
    }
  }, [])

  const filtered = useMemo(() => {
    return documents.filter((doc) => {
      const matchesStatus = statusFilter === 'all' || doc.processing_status === statusFilter
      const haystack = [
        doc.title,
        doc.purpose,
        doc.document_type,
        doc.summary,
        ...(doc.expand?.tags?.map((tag) => tag.name) ?? []),
      ]
        .join(' ')
        .toLowerCase()

      const matchesSearch = !search || haystack.includes(search.toLowerCase())
      return matchesStatus && matchesSearch
    })
  }, [documents, search, statusFilter])

  return (
    <section className="panel">
      <div className="panel-header">
        <div>
          <h2>Documents</h2>
          <p className="muted">Upload, search, and review AI-extracted metadata.</p>
        </div>
        <Link to="/upload" className="button">
          Upload document
        </Link>
      </div>

      <div className="filters">
        <input
          type="search"
          placeholder="Search title, tags, summary..."
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
        <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
          <option value="all">All statuses</option>
          <option value="pending">Pending</option>
          <option value="processing">Processing</option>
          <option value="completed">Completed</option>
          <option value="needs_review">Needs review</option>
          <option value="failed">Failed</option>
        </select>
      </div>

      {loading && <p className="muted">Loading documents...</p>}
      {error && <p className="error">{error}</p>}

      {!loading && filtered.length === 0 && (
        <div className="empty-state">
          <p>No documents yet.</p>
          <Link to="/upload">Upload your first document</Link>
        </div>
      )}

      <div className="document-grid">
        {filtered.map((document) => (
          <DocumentCard key={document.id} document={document} />
        ))}
      </div>
    </section>
  )
}
