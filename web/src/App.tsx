import { Routes, Route, Navigate } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import Layout from './components/Layout'
import ProtectedRoute from './components/ProtectedRoute'

const Login = lazy(() => import('./pages/Login'))
const Dashboard = lazy(() => import('./pages/Dashboard'))
const UrlList = lazy(() => import('./pages/UrlList'))
const UrlCreate = lazy(() => import('./pages/UrlCreate'))
const UrlDetail = lazy(() => import('./pages/UrlDetail'))

export default function App() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center min-h-screen"><div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full"></div></div>}>
      <Routes>
        <Route path="/admin/login" element={<Login />} />
        <Route path="/admin" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/admin/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="urls" element={<UrlList />} />
          <Route path="urls/create" element={<UrlCreate />} />
          <Route path="urls/:code" element={<UrlDetail />} />
        </Route>
        <Route path="*" element={<Navigate to="/admin/login" replace />} />
      </Routes>
    </Suspense>
  )
}