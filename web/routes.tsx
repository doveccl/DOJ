import { lazy, Suspense } from 'react'
import type { ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import { Shell } from './components/shell'
import { LoadingBlock } from './components/state'

const AdminPage = lazy(() => import('./pages/admin').then((mod) => ({ default: mod.AdminPage })))
const AssignmentDetailPage = lazy(() => import('./pages/assignment').then((mod) => ({ default: mod.AssignmentDetailPage })))
const AssignmentsPage = lazy(() => import('./pages/assignments').then((mod) => ({ default: mod.AssignmentsPage })))
const ContestDetailPage = lazy(() => import('./pages/contest').then((mod) => ({ default: mod.ContestDetailPage })))
const ContestsPage = lazy(() => import('./pages/contests').then((mod) => ({ default: mod.ContestsPage })))
const DiscussionPage = lazy(() => import('./pages/discussion').then((mod) => ({ default: mod.DiscussionPage })))
const HomePage = lazy(() => import('./pages/home').then((mod) => ({ default: mod.HomePage })))
const PostPage = lazy(() => import('./pages/post').then((mod) => ({ default: mod.PostPage })))
const ProblemDetailPage = lazy(() => import('./pages/problem').then((mod) => ({ default: mod.ProblemDetailPage })))
const ProblemsPage = lazy(() => import('./pages/problems').then((mod) => ({ default: mod.ProblemsPage })))
const ProfilePage = lazy(() => import('./pages/profile').then((mod) => ({ default: mod.ProfilePage })))
const RankPage = lazy(() => import('./pages/rank').then((mod) => ({ default: mod.RankPage })))
const SubmissionDetailPage = lazy(() => import('./pages/submission').then((mod) => ({ default: mod.SubmissionDetailPage })))
const SubmissionsPage = lazy(() => import('./pages/submissions').then((mod) => ({ default: mod.SubmissionsPage })))
const UserPage = lazy(() => import('./pages/user').then((mod) => ({ default: mod.UserPage })))

function page(node: ReactNode) {
  return <Suspense fallback={<LoadingBlock />}>{node}</Suspense>
}

export function AppRoutes() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Shell />}>
          <Route index element={page(<HomePage />)} />
          <Route path="problems" element={page(<ProblemsPage />)} />
          <Route path="problems/:id" element={page(<ProblemDetailPage />)} />
          <Route path="assignments" element={page(<AssignmentsPage />)} />
          <Route path="assignments/:id" element={page(<AssignmentDetailPage />)} />
          <Route path="contests" element={page(<ContestsPage />)} />
          <Route path="contests/:id" element={page(<ContestDetailPage />)} />
          <Route path="discussion" element={page(<DiscussionPage />)} />
          <Route path="discussion/:id" element={page(<PostPage />)} />
          <Route path="rank" element={page(<RankPage />)} />
          <Route path="users/:name" element={page(<UserPage />)} />
          <Route path="submissions" element={page(<SubmissionsPage />)} />
          <Route path="submissions/:id" element={page(<SubmissionDetailPage />)} />
          <Route path="admin" element={page(<AdminPage />)} />
          <Route path="profile" element={page(<ProfilePage />)} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
