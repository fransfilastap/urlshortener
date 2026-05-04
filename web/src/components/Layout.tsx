import { Link, Outlet } from 'react-router-dom'

export default function Layout() {
  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between max-w-7xl mx-auto">
          <div className="flex items-center gap-6">
            <Link to="/admin/dashboard" className="text-lg font-bold text-gray-900">URL Shortener</Link>
            <Link to="/admin/urls" className="text-sm text-gray-600 hover:text-gray-900">URLs</Link>
            <Link to="/admin/urls/create" className="text-sm text-gray-600 hover:text-gray-900">Create</Link>
          </div>
          <form action="/auth/logout" method="POST">
            <button type="submit" className="text-sm text-gray-500 hover:text-gray-700">Sign Out</button>
          </form>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}