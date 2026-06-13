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
          expand: 'tags,document_type',
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
        doc.expand?.document_type?.name,
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
    <section className="flex flex-col gap-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Documents</h2>
          <p className="text-sm text-gray-500">Upload, search, and review AI-extracted metadata.</p>
        </div>
        <Link
          to="/upload"
          className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700"
        >
          Upload document
        </Link>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row">
        <input
          type="search"
          placeholder="Search title, tags, summary..."
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none placeholder:text-gray-400 focus:border-gray-900 focus:ring-1 focus:ring-gray-900"
        />
        <select
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value)}
          className="rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:border-gray-900 focus:ring-1 focus:ring-gray-900 sm:w-48"
        >
          <option value="all">All statuses</option>
          <option value="pending">Pending</option>
          <option value="processing">Processing</option>
          <option value="completed">Completed</option>
          <option value="needs_review">Needs review</option>
          <option value="failed">Failed</option>
        </select>
      </div>

      {loading && <p className="text-sm text-gray-500">Loading documents...</p>}
      {error && <p className="text-sm text-red-600">{error}</p>}

      {!loading && filtered.length === 0 && (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white py-12 text-center">
          <p className="text-sm text-gray-500">No documents yet.</p>
          <Link to="/upload" className="mt-1 inline-block text-sm font-medium text-gray-900 underline">
            Upload your first document
          </Link>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filtered.map((document) => (
          <DocumentCard key={document.id} document={document} />
        ))}
      </div>
    </section>
  )
}
