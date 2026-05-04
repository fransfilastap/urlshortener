import { Navigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { authenticated, loading } = useAuth()

  if (loading) {
    return <div className="flex items-center justify-center min-h-screen"><div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full"></div></div>
  }

  if (!authenticated) {
    return <Navigate to="/admin/login" replace />
  }

  return <>{children}</>
}