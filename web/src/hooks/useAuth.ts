import { useState, useEffect } from 'react'

interface AuthState {
  authenticated: boolean
  loading: boolean
}

export function useAuth(): AuthState {
  const [auth, setAuth] = useState({ authenticated: false, loading: true })

  useEffect(() => {
    fetch('/auth/me', { credentials: 'include' })
      .then((res) => {
        if (res.ok) {
          setAuth({ authenticated: true, loading: false })
        } else {
          setAuth({ authenticated: false, loading: false })
        }
      })
      .catch(() => setAuth({ authenticated: false, loading: false }))
  }, [])

  return auth
}