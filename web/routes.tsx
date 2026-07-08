import { lazy, Suspense } from 'react'
import type { ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { AppLayout } from './components/app-layout'
import { LoadingBlock } from './components/state'

const AdminPage = lazy(() => import('./pages/admin').then((mod) => ({ default: mod.AdminPage })))
const AssignmentDetailPage = lazy(() => import('./pages/assignments/detail').then((mod) => ({ default: mod.AssignmentDetailPage })))
const AssignmentsPage = lazy(() => import('./pages/assignments').then((mod) => ({ default: mod.AssignmentsPage })))
const ContestDetailPage = lazy(() => import('./pages/contests/detail').then((mod) => ({ default: mod.ContestDetailPage })))
const ContestsPage = lazy(() => import('./pages/contests').then((mod) => ({ default: mod.ContestsPage })))
const DiscussionDetailPage = lazy(() => import('./pages/discussions/detail').then((mod) => ({ default: mod.DiscussionDetailPage })))
const DiscussionsPage = lazy(() => import('./pages/discussions').then((mod) => ({ default: mod.DiscussionsPage })))
const HomePage = lazy(() => import('./pages/home').then((mod) => ({ default: mod.HomePage })))
const ProblemDetailPage = lazy(() => import('./pages/problems/detail').then((mod) => ({ default: mod.ProblemDetailPage })))
const ProblemsPage = lazy(() => import('./pages/problems').then((mod) => ({ default: mod.ProblemsPage })))
const RankPage = lazy(() => import('./pages/rank').then((mod) => ({ default: mod.RankPage })))
const SubmissionDetailPage = lazy(() => import('./pages/submissions/detail').then((mod) => ({ default: mod.SubmissionDetailPage })))
const SubmissionsPage = lazy(() => import('./pages/submissions').then((mod) => ({ default: mod.SubmissionsPage })))
const UserPage = lazy(() => import('./pages/user').then((mod) => ({ default: mod.UserPage })))

function page(node: ReactNode) {
  return <Suspense fallback={<LoadingBlock />}>{node}</Suspense>
}

export function AppRoutes() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={page(<HomePage />)} />
          <Route path="problems" element={page(<ProblemsPage />)} />
          <Route path="problems/:id" element={page(<ProblemDetailPage />)} />
          <Route path="assignments" element={page(<AssignmentsPage />)} />
          <Route path="assignments/:id" element={page(<AssignmentDetailPage />)} />
          <Route path="contests" element={page(<ContestsPage />)} />
          <Route path="contests/:id" element={page(<ContestDetailPage />)} />
          <Route path="discussion" element={page(<DiscussionsPage />)} />
          <Route path="discussion/:id" element={page(<DiscussionDetailPage />)} />
          <Route path="rank" element={page(<RankPage />)} />
          <Route path="users/:name" element={page(<UserPage />)} />
          <Route path="submissions" element={page(<SubmissionsPage />)} />
          <Route path="submissions/:id" element={page(<SubmissionDetailPage />)} />
          <Route path="admin" element={page(<AdminPage />)} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
