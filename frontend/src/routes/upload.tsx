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

      const record = await pb.collection('documents').create(formData)
      navigate({ to: '/document/$documentId', params: { documentId: record.id } })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  return (
    <section className="mx-auto flex max-w-xl flex-col gap-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900">Upload document</h2>
        <p className="text-sm text-gray-500">Supported formats: PDF, JPEG, PNG, WebP, plain text.</p>
      </div>

      <form className="flex flex-col gap-4" onSubmit={onSubmit}>
        <label className="flex min-h-44 cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-gray-300 bg-white p-6 text-center transition-colors hover:border-gray-400">
          <input
            type="file"
            accept=".pdf,.jpg,.jpeg,.png,.webp,.txt,application/pdf,image/*,text/plain"
            onChange={(event) => setFile(event.target.files?.[0] ?? null)}
            className="hidden"
          />
          <span className="text-sm font-medium text-gray-900">
            {file ? file.name : 'Choose a file'}
          </span>
          {!file && <span className="text-xs text-gray-400">or drop it here</span>}
        </label>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          type="submit"
          disabled={uploading || !file}
          className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {uploading ? 'Uploading...' : 'Upload and process'}
        </button>
      </form>
    </section>
  )
}
