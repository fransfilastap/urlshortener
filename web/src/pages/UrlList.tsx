import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'

interface UrlItem {
  original_url: string
  short_url: string
  short_code: string
  title: string
  created_at: string
  clicks: number
  creator_reference: string
}

export default function UrlList() {
  const [urls, setUrls] = useState<UrlItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get<UrlItem[]>('/api/urls/creator/all')
      .then((data) => setUrls(Array.isArray(data) ? data : []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const handleDelete = async (code: string, creatorRef: string) => {
    if (!confirm('Delete this URL?')) return
    try {
      await api.delete(`/api/urls/${code}?creator_reference=${creatorRef}`)
      setUrls((prev) => prev.filter((u) => u.short_code !== code))
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  if (loading) return <div className="text-center py-8 text-gray-500">Loading...</div>
  if (error) return <div className="text-center py-8 text-red-600">{error}</div>

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">URLs</h2>
        <Link to="/admin/urls/create" className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700">Create URL</Link>
      </div>
      {urls.length === 0 ? (
        <div className="text-center py-8 text-gray-500">No URLs found.</div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Short Code</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Original URL</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Clicks</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {urls.map((url) => (
                <tr key={url.short_code} className="hover:bg-gray-50">
                  <td className="px-6 py-4 text-sm"><Link to={`/admin/urls/${url.short_code}`} className="text-blue-600 hover:underline">{url.short_code}</Link></td>
                  <td className="px-6 py-4 text-sm text-gray-600 max-w-xs truncate">{url.original_url}</td>
                  <td className="px-6 py-4 text-sm text-gray-600">{url.clicks}</td>
                  <td className="px-6 py-4 text-sm"><button onClick={() => handleDelete(url.short_code, url.creator_reference)} className="text-red-600 hover:text-red-800">Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}