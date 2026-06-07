import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', component: () => import('./pages/Home.vue') },
  { path: '/problems', component: () => import('./pages/ProblemList.vue') },
  { path: '/problems/:id', component: () => import('./pages/ProblemDetail.vue') },
  { path: '/assignments', component: () => import('./pages/AssignmentList.vue') },
  { path: '/assignments/:id', component: () => import('./pages/AssignmentDetail.vue') },
  { path: '/contests', component: () => import('./pages/ContestList.vue') },
  { path: '/contests/:id', component: () => import('./pages/ContestDetail.vue') },
  { path: '/discussion', component: () => import('./pages/DiscussionList.vue') },
  { path: '/discussion/:id', component: () => import('./pages/DiscussionDetail.vue') },
  { path: '/rank', component: () => import('./pages/Rank.vue') },
  { path: '/profile', component: () => import('./pages/Profile.vue') },
  { path: '/submissions', component: () => import('./pages/SubmissionList.vue') },
  { path: '/submissions/:id', component: () => import('./pages/SubmissionDetail.vue') },
  {
    path: '/admin',
    component: () => import('./pages/admin/Layout.vue'),
    redirect: '/admin/settings',
    children: [
      { path: 'settings', component: () => import('./pages/admin/Settings.vue') },
      { path: 'members', component: () => import('./pages/admin/Members.vue') },
      { path: 'problems', component: () => import('./pages/admin/Problems.vue') },
      { path: 'assignments', component: () => import('./pages/admin/Assignments.vue') },
      { path: 'contests', component: () => import('./pages/admin/Contests.vue') },
      { path: 'languages', component: () => import('./pages/admin/Languages.vue') },
      { path: 'agents', component: () => import('./pages/admin/Agents.vue') }
    ]
  }
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
