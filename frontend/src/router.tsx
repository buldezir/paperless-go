import { createRootRoute, createRoute, createRouter, Outlet, Link } from '@tanstack/react-router'
import { IndexPage } from './routes/index'
import { UploadPage } from './routes/upload'
import { DocumentDetailPage } from './routes/documents.$documentId'

const rootRoute = createRootRoute({
  component: RootLayout,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: IndexPage,
})

const uploadRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/upload',
  component: UploadPage,
})

const documentRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/documents/$documentId',
  component: DocumentDetailPage,
})

const routeTree = rootRoute.addChildren([indexRoute, uploadRoute, documentRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

function RootLayout() {
  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">Paperless Go</p>
          <h1>Document Storage</h1>
        </div>
        <nav className="app-nav">
          <Link to="/" activeProps={{ className: 'active' }}>
            Documents
          </Link>
          <Link to="/upload" activeProps={{ className: 'active' }}>
            Upload
          </Link>
        </nav>
      </header>
      <main className="app-main">
        <Outlet />
      </main>
    </div>
  )
}
