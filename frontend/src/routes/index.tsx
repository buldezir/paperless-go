import { useEffect, useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ensureAuth, pb } from '../lib/pocketbase'
import type { CorrespondentRecord, DocumentRecord, DocumentTypeRecord } from '../lib/pocketbase'
import { DocumentCard } from '../components/DocumentCard'

const selectClassName =
  'rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:border-gray-900 focus:ring-1 focus:ring-gray-900'

function buildDocumentFilter(filters: {
  status: string
  dateFrom: string
  dateTo: string
  documentType: string
  correspondent: string
}) {
  const parts: string[] = []

  if (filters.status !== 'all') {
    parts.push(`processing_status = "${filters.status}"`)
  }
  if (filters.documentType !== 'all') {
    parts.push(`document_type = "${filters.documentType}"`)
  }
  if (filters.correspondent !== 'all') {
    parts.push(`correspondent = "${filters.correspondent}"`)
  }
  if (filters.dateFrom) {
    parts.push(`document_date >= "${filters.dateFrom}"`)
  }
  if (filters.dateTo) {
    parts.push(`document_date <= "${filters.dateTo}"`)
  }

  return parts.length > 0 ? parts.join(' && ') : undefined
}

export function IndexPage() {
  const [documents, setDocuments] = useState<DocumentRecord[]>([])
  const [documentTypes, setDocumentTypes] = useState<DocumentTypeRecord[]>([])
  const [correspondents, setCorrespondents] = useState<CorrespondentRecord[]>([])
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [documentTypeFilter, setDocumentTypeFilter] = useState('all')
  const [correspondentFilter, setCorrespondentFilter] = useState('all')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true

    async function loadFilterOptions() {
      try {
        await ensureAuth()
        const [types, cors] = await Promise.all([
          pb.collection('document_types').getFullList<DocumentTypeRecord>({ sort: 'name' }),
          pb.collection('correspondents').getFullList<CorrespondentRecord>({ sort: 'name' }),
        ])
        if (active) {
          setDocumentTypes(types)
          setCorrespondents(cors)
        }
      } catch (err) {
        if (active) {
          setError(err instanceof Error ? err.message : 'Failed to load filter options')
        }
      }
    }

    void loadFilterOptions()

    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    let active = true

    async function load() {
      try {
        setLoading(true)
        await ensureAuth()
        const filter = buildDocumentFilter({
          status: statusFilter,
          dateFrom,
          dateTo,
          documentType: documentTypeFilter,
          correspondent: correspondentFilter,
        })
        const result = await pb.collection('documents').getList<DocumentRecord>(1, 50, {
          sort: '-created',
          expand: 'tags,document_type,correspondent',
          ...(filter ? { filter } : {}),
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
  }, [statusFilter, dateFrom, dateTo, documentTypeFilter, correspondentFilter])

  const hasActiveFilters =
    statusFilter !== 'all' ||
    dateFrom !== '' ||
    dateTo !== '' ||
    documentTypeFilter !== 'all' ||
    correspondentFilter !== 'all' ||
    search !== ''

  const filtered = useMemo(() => {
    return documents.filter((doc) => {
      const haystack = [
        doc.title,
        doc.purpose,
        doc.expand?.document_type?.name,
        doc.expand?.document_type?.name_original,
        doc.expand?.correspondent?.name,
        doc.expand?.correspondent?.name_original,
        doc.summary,
        ...(doc.expand?.tags?.map((tag) => tag.name) ?? []),
      ]
        .join(' ')
        .toLowerCase()

      return !search || haystack.includes(search.toLowerCase())
    })
  }, [documents, search])

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

      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-3 sm:flex-row">
          <input
            type="search"
            placeholder="Search title, correspondent, tags, summary..."
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none placeholder:text-gray-400 focus:border-gray-900 focus:ring-1 focus:ring-gray-900"
          />
          <select
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
            className={`${selectClassName} sm:w-48`}
          >
            <option value="all">All statuses</option>
            <option value="pending">Pending</option>
            <option value="processing">Processing</option>
            <option value="completed">Completed</option>
            <option value="needs_review">Needs review</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-gray-500">From date</span>
            <input
              type="date"
              value={dateFrom}
              onChange={(event) => setDateFrom(event.target.value)}
              className={selectClassName}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-gray-500">To date</span>
            <input
              type="date"
              value={dateTo}
              onChange={(event) => setDateTo(event.target.value)}
              className={selectClassName}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-gray-500">Document type</span>
            <select
              value={documentTypeFilter}
              onChange={(event) => setDocumentTypeFilter(event.target.value)}
              className={selectClassName}
            >
              <option value="all">All types</option>
              {documentTypes.map((type) => (
                <option key={type.id} value={type.id}>
                  {type.name}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-gray-500">Correspondent</span>
            <select
              value={correspondentFilter}
              onChange={(event) => setCorrespondentFilter(event.target.value)}
              className={selectClassName}
            >
              <option value="all">All correspondents</option>
              {correspondents.map((correspondent) => (
                <option key={correspondent.id} value={correspondent.id}>
                  {correspondent.name}
                </option>
              ))}
            </select>
          </label>
        </div>
      </div>

      {loading && <p className="text-sm text-gray-500">Loading documents...</p>}
      {error && <p className="text-sm text-red-600">{error}</p>}

      {!loading && filtered.length === 0 && (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white py-12 text-center">
          {hasActiveFilters ? (
            <p className="text-sm text-gray-500">No documents match your filters.</p>
          ) : (
            <>
              <p className="text-sm text-gray-500">No documents yet.</p>
              <Link to="/upload" className="mt-1 inline-block text-sm font-medium text-gray-900 underline">
                Upload your first document
              </Link>
            </>
          )}
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
