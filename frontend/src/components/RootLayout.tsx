import { Link, Outlet } from '@tanstack/react-router'

const navLinkClass =
  'rounded-md px-3 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900'
const navLinkActiveClass = 'bg-gray-900 text-white hover:bg-gray-900 hover:text-white'

export function RootLayout() {
  return (
    <div className="min-h-screen bg-gray-50 text-gray-900">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
          <Link to="/" className="flex items-center gap-2 font-semibold text-gray-900">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-gray-900 text-sm text-white">
              P
            </span>
            Paperless Go
          </Link>
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
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
