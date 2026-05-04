import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'

interface CreateResponse {
  original_url: string
  short_url: string
  short_code: string
  title: string
  clicks: number
}

export default function UrlCreate() {
  const navigate = useNavigate()
  const [url, setUrl] = useState('')
  const [customCode, setCustomCode] = useState('')
  const [title, setTitle] = useState('')
  const [creatorRef, setCreatorRef] = useState('')
  const [expiry, setExpiry] = useState(0)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<CreateResponse | null>(null)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    setResult(null)

    try {
      const body: Record<string, unknown> = { url }
      if (customCode) body.custom_code = customCode
      if (title) body.title = title
      if (creatorRef) body.creator_reference = creatorRef
      if (expiry > 0) body.expiry = expiry

      const data = await api.post<CreateResponse>('/api/shorten', body)
      setResult(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create URL')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Create Short URL</h2>
      {result && (
        <div className="mb-6 bg-green-50 border border-green-200 rounded-lg p-4">
          <p className="text-sm text-green-800">URL created successfully!</p>
          <p className="mt-2 text-lg font-mono"><a href={result.short_url} className="text-blue-600 hover:underline">{result.short_url}</a></p>
        </div>
      )}
      {error && <div className="mb-6 bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-800">{error}</div>}
      <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-gray-200 p-6 space-y-4 max-w-lg">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Original URL *</label>
          <input type="url" required value={url} onChange={(e) => setUrl(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" placeholder="https://example.com" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Custom Code</label>
          <input type="text" value={customCode} onChange={(e) => setCustomCode(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" placeholder="my-custom-code" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input type="text" value={title} onChange={(e) => setTitle(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Creator Reference</label>
          <input type="text" value={creatorRef} onChange={(e) => setCreatorRef(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Expiry (seconds, 0 = never)</label>
          <input type="number" value={expiry} onChange={(e) => setExpiry(Number(e.target.value))} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" min={0} />
        </div>
        <div className="flex gap-3">
          <button type="submit" disabled={loading} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? 'Creating...' : 'Create'}</button>
          <button type="button" onClick={() => navigate(-1)} className="text-gray-600 px-4 py-2 rounded-md text-sm hover:bg-gray-100">Cancel</button>
        </div>
      </form>
    </div>
  )
}