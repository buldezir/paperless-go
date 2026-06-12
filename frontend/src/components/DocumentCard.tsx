import { Link } from '@tanstack/react-router'
import type { DocumentRecord } from '../lib/pocketbase'

type Props = {
  document: DocumentRecord
}

const statusLabels: Record<DocumentRecord['processing_status'], string> = {
  pending: 'Pending',
  processing: 'Processing',
  completed: 'Completed',
  failed: 'Failed',
  needs_review: 'Needs review',
}

const statusStyles: Record<DocumentRecord['processing_status'], string> = {
  pending: 'bg-amber-50 text-amber-700 ring-amber-200',
  processing: 'bg-blue-50 text-blue-700 ring-blue-200',
  completed: 'bg-green-50 text-green-700 ring-green-200',
  failed: 'bg-red-50 text-red-700 ring-red-200',
  needs_review: 'bg-amber-50 text-amber-700 ring-amber-200',
}

export function DocumentCard({ document }: Props) {
  const tags = document.expand?.tags?.map((tag) => tag.name) ?? []

  return (
    <Link
      to="/documents/$documentId"
      params={{ documentId: document.id }}
      className="flex flex-col gap-3 rounded-lg border border-gray-200 bg-white p-4 transition-colors hover:border-gray-300 hover:shadow-sm"
    >
      <div className="flex items-center justify-between gap-2">
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${statusStyles[document.processing_status]}`}
        >
          {statusLabels[document.processing_status]}
        </span>
        {document.document_date && (
          <span className="text-xs text-gray-400">{document.document_date.slice(0, 10)}</span>
        )}
      </div>

      <div>
        <h3 className="font-medium text-gray-900">{document.title || 'Untitled document'}</h3>
        <p className="text-xs text-gray-500">{document.document_type || 'Unknown type'}</p>
      </div>

      <p className="line-clamp-3 text-sm text-gray-600">
        {document.summary || document.purpose || 'No summary yet.'}
      </p>

      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {tags.map((tag) => (
            <span key={tag} className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
              {tag}
            </span>
          ))}
        </div>
      )}
    </Link>
  )
}
