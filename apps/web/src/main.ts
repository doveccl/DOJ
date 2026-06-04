import { createApp } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import App from './App.vue'
import AdminAssignmentsPage from './pages/AdminAssignmentsPage.vue'
import AdminContestsPage from './pages/AdminContestsPage.vue'
import AdminGroupsPage from './pages/AdminGroupsPage.vue'
import AdminLanguagesPage from './pages/AdminLanguagesPage.vue'
import AdminProblemsPage from './pages/AdminProblemsPage.vue'
import AdminRunnersPage from './pages/AdminRunnersPage.vue'
import AssignmentDetailPage from './pages/AssignmentDetailPage.vue'
import AssignmentListPage from './pages/AssignmentListPage.vue'
import BbsDetailPage from './pages/BbsDetailPage.vue'
import BbsListPage from './pages/BbsListPage.vue'
import ContestDetailPage from './pages/ContestDetailPage.vue'
import ContestListPage from './pages/ContestListPage.vue'
import HomePage from './pages/HomePage.vue'
import ProblemDetailPage from './pages/ProblemDetailPage.vue'
import ProblemListPage from './pages/ProblemListPage.vue'
import RankPage from './pages/RankPage.vue'
import SubmissionDetailPage from './pages/SubmissionDetailPage.vue'
import SubmissionListPage from './pages/SubmissionListPage.vue'
import './style.css'

const routes: RouteRecordRaw[] = [
  { path: '/', component: HomePage },
  { path: '/problems', component: ProblemListPage },
  { path: '/problems/:id', component: ProblemDetailPage },
  { path: '/assignments', component: AssignmentListPage },
  { path: '/assignments/:id', component: AssignmentDetailPage },
  { path: '/contests', component: ContestListPage },
  { path: '/contests/:id', component: ContestDetailPage },
  { path: '/bbs', component: BbsListPage },
  { path: '/bbs/:id', component: BbsDetailPage },
  { path: '/rank', component: RankPage },
  { path: '/submissions', component: SubmissionListPage },
  { path: '/submissions/:id', component: SubmissionDetailPage },
  { path: '/admin/groups', component: AdminGroupsPage },
  { path: '/admin/problems', component: AdminProblemsPage },
  { path: '/admin/assignments', component: AdminAssignmentsPage },
  { path: '/admin/contests', component: AdminContestsPage },
  { path: '/admin/languages', component: AdminLanguagesPage },
  { path: '/admin/runners', component: AdminRunnersPage }
]

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      app: 'DOJ',
      problems: '题库',
      assignments: '作业',
      contests: '比赛',
      discussion: '讨论',
      rank: '排名',
      submissions: '提交',
      home: '首页',
      admin: '管理',
      groups: '用户组',
      manageProblems: '题目',
      languages: '语言',
      runners: '评测机'
    },
    en: {
      app: 'DOJ',
      problems: 'Problems',
      assignments: 'Assignments',
      contests: 'Contests',
      discussion: 'Discussion',
      rank: 'Rank',
      submissions: 'Submissions',
      home: 'Home',
      admin: 'Admin',
      groups: 'Groups',
      manageProblems: 'Problems',
      languages: 'Languages',
      runners: 'Runners'
    }
  }
})

const router = createRouter({
  history: createWebHistory(),
  routes
})

createApp(App).use(createPinia()).use(i18n).use(router).mount('#app')
