import { createRouter, createWebHistory } from 'vue-router'

import HomePage from '@pages/HomePage.vue'
import ProblemList from '@pages/ProblemList.vue'
import ContestList from '@pages/ContestList.vue'
import SubmissionList from '@pages/SubmissionList.vue'

export const routes = [
  { path: '/', component: HomePage },
  { path: '/problems', name: 'problems', component: ProblemList },
  { path: '/contests', name: 'contests', component: ContestList },
  { path: '/submissions', name: 'submissions', component: SubmissionList }
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
