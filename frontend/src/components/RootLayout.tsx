import { useEffect, useState } from 'react'
import { Link, Outlet } from '@tanstack/react-router'
import { ensureAuth, getUserDisplayName, logout, pb, pbAdminUrl } from '../lib/pocketbase'
import { LoginPage } from './LoginPage'

const navLinkClass =
  'rounded-md px-3 py-1.5 text-sm font-medium text-stone-600 transition-colors hover:bg-stone-200/70 hover:text-stone-950'
const navLinkActiveClass = 'bg-gray-900 text-white hover:bg-gray-900 hover:text-white'
const iconButtonClass =
  'rounded-md p-1.5 text-stone-500 transition-colors hover:bg-stone-200/70 hover:text-stone-950 cursor-pointer'

function LogoutIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-4 w-4"
      aria-hidden="true"
    >
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <polyline points="16 17 21 12 16 7" />
      <line x1="21" y1="12" x2="9" y2="12" />
    </svg>
  )
}

export function RootLayout() {
  const [authState, setAuthState] = useState<'loading' | 'authenticated' | 'unauthenticated'>('loading')
  const userDisplayName = authState === 'authenticated' ? getUserDisplayName() : ''

  useEffect(() => {
    let active = true

    async function initAuth() {
      try {
        await ensureAuth()
        if (active) {
          setAuthState('authenticated')
        }
      } catch {
        if (active) {
          setAuthState('unauthenticated')
        }
      }
    }

    void initAuth()

    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (authState === 'loading') {
      return
    }

    return pb.authStore.onChange(() => {
      setAuthState(pb.authStore.isValid ? 'authenticated' : 'unauthenticated')
    })
  }, [authState])

  if (authState === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-stone-100 text-sm text-stone-500">
        Loading...
      </div>
    )
  }

  if (authState === 'unauthenticated') {
    return <LoginPage onSuccess={() => setAuthState('authenticated')} />
  }

  return (
    <div className="min-h-screen bg-stone-100 text-stone-900">
      <header className="border-b border-stone-200 bg-stone-50/95">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <Link to="/" className="flex items-center gap-2 font-semibold text-stone-950">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-gray-900 text-sm text-white">
              P
            </span>
            Paperless Go
          </Link>
          <div className="flex items-center gap-4">
            <nav className="flex items-center gap-1">
              <Link
                to="/"
                className={navLinkClass}
                activeOptions={{ exact: true }}
                activeProps={{ className: `${navLinkClass} ${navLinkActiveClass}` }}
              >
                Documents
              </Link>
              <Link
                to="/upload"
                className={navLinkClass}
                activeProps={{ className: `${navLinkClass} ${navLinkActiveClass}` }}
              >
                Upload
              </Link>
              <Link
                to="/ocr-test"
                className={navLinkClass}
                activeProps={{ className: `${navLinkClass} ${navLinkActiveClass}` }}
              >
                OCR test
              </Link>
              <a
                href={pbAdminUrl}
                target="_blank"
                rel="noopener noreferrer"
                className={navLinkClass}
              >
                Admin
              </a>
            </nav>
            <div className="flex items-center gap-2 border-l border-stone-200 pl-4">
              {userDisplayName && (
                <span className="max-w-40 truncate text-sm text-stone-600" title={userDisplayName}>
                  {userDisplayName}
                </span>
              )}
              <button
                type="button"
                onClick={logout}
                className={iconButtonClass}
                aria-label="Log out"
                title="Log out"
              >
                <LogoutIcon />
              </button>
            </div>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
