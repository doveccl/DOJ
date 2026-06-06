import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import App from './App.vue'
import { i18n } from './i18n'
import './style.scss'

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
    component: () => import('./pages/AdminLayout.vue'),
    redirect: '/admin/settings',
    children: [
      { path: 'settings', component: () => import('./pages/AdminSettings.vue') },
      { path: 'members', component: () => import('./pages/AdminMembers.vue') },
      { path: 'problems', component: () => import('./pages/AdminProblems.vue') },
      { path: 'assignments', component: () => import('./pages/AdminAssignments.vue') },
      { path: 'contests', component: () => import('./pages/AdminContests.vue') },
      { path: 'languages', component: () => import('./pages/AdminLanguages.vue') },
      { path: 'agents', component: () => import('./pages/AdminAgents.vue') }
    ]
  },
  { path: '/admin/runners', redirect: '/admin/agents' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

createApp(App).use(createPinia()).use(i18n).use(router).mount('#app')
