import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../api/client'

interface UrlInfo {
  original_url: string
  short_url: string
  short_code: string
  title: string
  expires_at: string
  created_at: string
  clicks: number
  creator_reference: string
}

interface Analytics {
  total_clicks: number
  browsers: Record<string, number>
  devices: Record<string, number>
  locations: Record<string, number>
}

export default function UrlDetail() {
  const { code } = useParams<{ code: string }>()
  const [url, setUrl] = useState<UrlInfo | null>(null)
  const [analytics, setAnalytics] = useState<Analytics | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!code) return
    api.get<UrlInfo>(`/api/urls/${code}`)
      .then(setUrl)
      .catch((err) => setError(err.message))

    api.get<Analytics>(`/api/urls/${code}/analytics`)
      .then(setAnalytics)
      .catch(() => {})
  }, [code])

  if (error) return <div className="text-center py-8 text-red-600">{error}</div>
  if (!url) return <div className="text-center py-8">Loading...</div>

  return (
    <div>
      <div className="flex items-center gap-2 mb-6">
        <Link to="/admin/urls" className="text-sm text-gray-500 hover:text-gray-700">&larr; Back to URLs</Link>
        <h2 className="text-2xl font-bold">URL Details</h2>
      </div>
      <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-4 max-w-2xl">
        <div><span className="text-sm text-gray-500">Short Code</span><p className="font-mono text-lg">{url.short_code}</p></div>
        <div><span className="text-sm text-gray-500">Original URL</span><p className="text-sm break-all"><a href={url.original_url} className="text-blue-600 hover:underline">{url.original_url}</a></p></div>
        <div><span className="text-sm text-gray-500">Short URL</span><p className="font-mono text-sm"><a href={url.short_url} className="text-blue-600 hover:underline">{url.short_url}</a></p></div>
        <div className="grid grid-cols-2 gap-4">
          <div><span className="text-sm text-gray-500">Title</span><p className="text-sm">{url.title || '\u2014'}</p></div>
          <div><span className="text-sm text-gray-500">Total Clicks</span><p className="text-2xl font-bold">{url.clicks}</p></div>
          <div><span className="text-sm text-gray-500">Created</span><p className="text-sm">{new Date(url.created_at).toLocaleDateString()}</p></div>
          <div><span className="text-sm text-gray-500">Expires</span><p className="text-sm">{url.expires_at ? new Date(url.expires_at).toLocaleDateString() : 'Never'}</p></div>
        </div>
      </div>
      {analytics && (
        <div className="mt-8">
          <h3 className="text-lg font-bold mb-4">Analytics</h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm text-gray-500 mb-2">Browsers</h4>
              <ul className="space-y-1">{Object.entries(analytics.browsers || {}).map(([k, v]) => <li key={k} className="text-sm">{k}: {v}</li>)}</ul>
            </div>
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm text-gray-500 mb-2">Devices</h4>
              <ul className="space-y-1">{Object.entries(analytics.devices || {}).map(([k, v]) => <li key={k} className="text-sm">{k}: {v}</li>)}</ul>
            </div>
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm text-gray-500 mb-2">Locations</h4>
              <ul className="space-y-1">{Object.entries(analytics.locations || {}).map(([k, v]) => <li key={k} className="text-sm">{k}: {v}</li>)}</ul>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}