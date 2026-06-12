import { type FormEvent, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ensureAuth, pb } from '../lib/pocketbase'

export function UploadPage() {
  const navigate = useNavigate()
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (!file) {
      setError('Choose a file to upload.')
      return
    }

    try {
      setUploading(true)
      setError('')
      await ensureAuth()

      const formData = new FormData()
      formData.append('file', file)
      formData.append('user', pb.authStore.record?.id ?? '')
      formData.append('processing_status', 'pending')
      formData.append('metadata_source', 'ai')

      const record = await pb.collection('documents').create(formData)
      navigate({ to: '/documents/$documentId', params: { documentId: record.id } })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  return (
    <section className="panel narrow">
      <div className="panel-header">
        <div>
          <h2>Upload document</h2>
          <p className="muted">Supported formats: PDF, JPEG, PNG, WebP, plain text.</p>
        </div>
      </div>

      <form className="stack" onSubmit={onSubmit}>
        <label className="file-drop">
          <input
            type="file"
            accept=".pdf,.jpg,.jpeg,.png,.webp,.txt,application/pdf,image/*,text/plain"
            onChange={(event) => setFile(event.target.files?.[0] ?? null)}
          />
          <span>{file ? file.name : 'Choose a file or drop it here'}</span>
        </label>

        {error && <p className="error">{error}</p>}

        <button type="submit" className="button" disabled={uploading || !file}>
          {uploading ? 'Uploading...' : 'Upload and process'}
        </button>
      </form>
    </section>
  )
}
