import { createApp } from 'vue'
import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import {
  createRouter,
  createWebHistory,
  type RouteRecordRaw
} from 'vue-router'
import App from './App.vue'
import AdminGroupsPage from './pages/AdminGroupsPage.vue'
import HomePage from './pages/HomePage.vue'
import ProblemDetailPage from './pages/ProblemDetailPage.vue'
import ProblemListPage from './pages/ProblemListPage.vue'
import SubmissionDetailPage from './pages/SubmissionDetailPage.vue'
import SubmissionListPage from './pages/SubmissionListPage.vue'
import './style.css'

const routes: RouteRecordRaw[] = [
  { path: '/', component: HomePage },
  { path: '/problems', component: ProblemListPage },
  { path: '/problems/:id', component: ProblemDetailPage },
  { path: '/submissions', component: SubmissionListPage },
  { path: '/submissions/:id', component: SubmissionDetailPage },
  { path: '/admin/groups', component: AdminGroupsPage }
]

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      app: 'DOJ',
      problems: '题库',
      submissions: '提交',
      home: '首页',
      admin: '管理'
    },
    en: {
      app: 'DOJ',
      problems: 'Problems',
      submissions: 'Submissions',
      home: 'Home',
      admin: 'Admin'
    }
  }
})

const router = createRouter({
  history: createWebHistory(),
  routes
})

createApp(App).use(createPinia()).use(i18n).use(router).mount('#app')
