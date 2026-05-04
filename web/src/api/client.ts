const BASE_URL = ''

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (res.status === 401) {
    window.location.href = '/admin/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const data = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(data.error || 'Request failed')
  }

  return res.json()
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),

  post: <T>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'POST', body: JSON.stringify(body) }),

  put: <T>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'PUT', body: JSON.stringify(body) }),

  delete: <T>(path: string) => apiFetch<T>(path, { method: 'DELETE' }),
}