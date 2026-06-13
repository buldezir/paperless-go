import { type FormEvent, useEffect, useRef, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { chatWithDocument, ensureAuth, pb, type ChatMessage, type DocumentRecord } from '../lib/pocketbase'

export function DocumentAskPage() {
  const { documentId } = useParams({ from: '/document/$documentId/ask' })
  const [document, setDocument] = useState<DocumentRecord | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let active = true

    async function load() {
      try {
        setLoading(true)
        await ensureAuth()
        const doc = await pb.collection('documents').getOne<DocumentRecord>(documentId)
        if (active) {
          setDocument(doc)
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

    void load()

    return () => {
      active = false
    }
  }, [documentId])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, sending])

  const hasOcrText = Boolean(document?.ocr_text?.trim())

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    const text = input.trim()
    if (!text || sending || !hasOcrText) {
      return
    }

    const userMessage: ChatMessage = { role: 'user', content: text }
    const nextMessages = [...messages, userMessage]

    try {
      setSending(true)
      setInput('')
      setError('')
      setMessages(nextMessages)

      const reply = await chatWithDocument(documentId, nextMessages)
      setMessages([...nextMessages, reply])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to get AI response')
      setMessages(messages)
      setInput(text)
    } finally {
      setSending(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-gray-500">Loading...</p>
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
    <section className="flex flex-col gap-4">
      <div>
        <Link
          to="/document/$documentId"
          params={{ documentId }}
          className="text-sm text-gray-500 hover:text-gray-900"
        >
          &larr; Back to document
        </Link>
        <h2 className="mt-1 text-xl font-semibold text-gray-900">
          Ask AI: {document.title || 'Untitled document'}
        </h2>
        <p className="text-sm text-gray-500">
          Questions are answered using the document&apos;s OCR text as context.
        </p>
      </div>

      {!hasOcrText ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          This document has no OCR text yet. Run full processing before asking questions.
        </div>
      ) : (
        <div className="flex min-h-128 flex-col overflow-hidden rounded-lg border border-gray-200 bg-white">
          <div className="flex-1 space-y-4 overflow-y-auto p-4">
            {messages.length === 0 && (
              <p className="text-sm text-gray-400">
                Ask a question about this document, for example: &quot;What is the total amount?&quot;
              </p>
            )}
            {messages.map((message, index) => (
              <div
                key={index}
                className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[85%] rounded-lg px-4 py-2.5 text-sm leading-relaxed whitespace-pre-wrap ${
                    message.role === 'user'
                      ? 'bg-gray-900 text-white'
                      : 'border border-gray-200 bg-gray-50 text-gray-900'
                  }`}
                >
                  {message.content}
                </div>
              </div>
            ))}
            {sending && (
              <div className="flex justify-start">
                <div className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-2.5 text-sm text-gray-500">
                  Thinking...
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          <form onSubmit={onSubmit} className="border-t border-gray-200 p-4">
            <div className="flex items-end gap-3">
              <textarea
                rows={2}
                value={input}
                onChange={(event) => setInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey) {
                    event.preventDefault()
                    void onSubmit(event)
                  }
                }}
                autoFocus
                disabled={sending}
                placeholder="Ask a question about this document..."
                className="min-h-12 flex-1 resize-y rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 outline-none placeholder:text-gray-400 focus:border-gray-900 focus:ring-1 focus:ring-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
              />
              <button
                type="submit"
                disabled={sending || !input.trim()}
                className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {sending ? 'Sending...' : 'Send'}
              </button>
            </div>
            {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
          </form>
        </div>
      )}
    </section>
  )
}
