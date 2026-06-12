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

export function DocumentCard({ document }: Props) {
  const tags = document.expand?.tags?.map((tag) => tag.name) ?? []

  return (
    <article className={`document-card status-${document.processing_status}`}>
      <div className="document-card-header">
        <span className={`status-pill status-${document.processing_status}`}>
          {statusLabels[document.processing_status]}
        </span>
        {document.document_date && <span className="muted">{document.document_date.slice(0, 10)}</span>}
      </div>

      <h3>{document.title || 'Untitled document'}</h3>
      <p className="muted">{document.document_type || 'Unknown type'}</p>
      <p>{document.summary || document.purpose || 'No summary yet.'}</p>

      {tags.length > 0 && (
        <div className="tag-list">
          {tags.map((tag) => (
            <span key={tag} className="tag">
              {tag}
            </span>
          ))}
        </div>
      )}

      <Link to="/documents/$documentId" params={{ documentId: document.id }} className="card-link">
        Review document
      </Link>
    </article>
  )
}
