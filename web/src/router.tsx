import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from '@/components/Layout'
import RequireAuth from '@/components/RequireAuth'

const TeacherList = lazy(() => import('./pages/TeacherList'))
const TeacherDetail = lazy(() => import('./pages/TeacherDetail'))
const Resume = lazy(() => import('./pages/Resume'))
const Login = lazy(() => import('./pages/Login'))
const SettingsLayout = lazy(() => import('./pages/Settings/SettingsLayout'))
const General = lazy(() => import('./pages/Settings/General'))
const Dicts = lazy(() => import('./pages/Settings/Dicts'))
const LLM = lazy(() => import('./pages/Settings/LLM'))
const Data = lazy(() => import('./pages/Settings/Data'))
const NotFound = lazy(() => import('./pages/NotFound'))

export default function Router() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-background">
          <div className="size-7 animate-spin rounded-full border-2 border-[var(--track)] border-t-[var(--accent)]" />
        </div>
      }
    >
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<RequireAuth />}>
          <Route element={<Layout />}>
            <Route path="/" element={<Navigate to="/teachers" replace />} />
            <Route path="/teachers" element={<TeacherList />} />
            <Route path="/teachers/:id" element={<TeacherDetail />} />
            <Route path="/resume" element={<Resume />} />
            <Route path="/settings" element={<SettingsLayout />}>
              <Route index element={<Navigate to="/settings/general" replace />} />
              <Route path="general" element={<General />} />
              <Route path="dicts" element={<Dicts />} />
              <Route path="llm" element={<LLM />} />
              <Route path="data" element={<Data />} />
            </Route>
          </Route>
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Suspense>
  )
}
