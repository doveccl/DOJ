import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import App from './App.vue'
import { i18n } from './i18n'
import './style.css'

const routes: RouteRecordRaw[] = [
  { path: '/', component: () => import('./pages/HomePage.vue') },
  { path: '/problems', component: () => import('./pages/ProblemListPage.vue') },
  { path: '/problems/:id', component: () => import('./pages/ProblemDetailPage.vue') },
  { path: '/assignments', component: () => import('./pages/AssignmentListPage.vue') },
  { path: '/assignments/:id', component: () => import('./pages/AssignmentDetailPage.vue') },
  { path: '/contests', component: () => import('./pages/ContestListPage.vue') },
  { path: '/contests/:id', component: () => import('./pages/ContestDetailPage.vue') },
  { path: '/bbs', component: () => import('./pages/BbsListPage.vue') },
  { path: '/bbs/:id', component: () => import('./pages/BbsDetailPage.vue') },
  { path: '/rank', component: () => import('./pages/RankPage.vue') },
  { path: '/submissions', component: () => import('./pages/SubmissionListPage.vue') },
  { path: '/submissions/:id', component: () => import('./pages/SubmissionDetailPage.vue') },
  { path: '/admin/groups', component: () => import('./pages/AdminGroupsPage.vue') },
  { path: '/admin/users', component: () => import('./pages/AdminUsersPage.vue') },
  { path: '/admin/problems', component: () => import('./pages/AdminProblemsPage.vue') },
  { path: '/admin/assignments', component: () => import('./pages/AdminAssignmentsPage.vue') },
  { path: '/admin/contests', component: () => import('./pages/AdminContestsPage.vue') },
  { path: '/admin/languages', component: () => import('./pages/AdminLanguagesPage.vue') },
  { path: '/admin/runners', component: () => import('./pages/AdminRunnersPage.vue') }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

createApp(App).use(createPinia()).use(i18n).use(router).mount('#app')
