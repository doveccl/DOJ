import { createI18n } from 'vue-i18n'

export const supportedLocales = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en' }
] as const

export type SupportedLocale = (typeof supportedLocales)[number]['value']

const fallbackLocale: SupportedLocale = 'zh-CN'

const zhCNMessages = {
  app: {
    brand: 'DOJ',
    signIn: '登录',
    signUp: '注册',
    signOut: '退出登录',
    cancel: '取消',
    user: '用户名或邮箱',
    userName: '用户名',
    email: '邮箱',
    password: '密码',
    locale: '语言',
    colorMode: '颜色模式',
    light: '亮',
    dark: '暗'
  },
  nav: {
    home: '首页',
    problems: '题库',
    assignments: '作业',
    contests: '比赛',
    discussion: '讨论',
    rank: '排名',
    submissions: '提交',
    admin: '管理',
    groups: '用户组',
    users: '用户',
    manageProblems: '题目',
    languages: '语言',
    runners: '评测机'
  },
  common: {
    id: '编号',
    problem: '题目',
    user: '用户',
    status: '状态',
    language: '语言',
    time: '时间',
    memory: '内存',
    message: '消息',
    title: '标题',
    tags: '标签',
    solved: '通过'
  },
  dashboard: {
    title: '概览',
    subtitle: '系统活跃度、任务量和最近评测结果。',
    problems: '题目',
    submissions: '提交',
    users: '用户',
    contests: '比赛',
    assignments: '作业',
    recentSubmissions: '最近提交'
  },
  problems: {
    title: '题库',
    empty: '还没有题目'
  },
  submissions: {
    title: '提交记录'
  }
}

const enMessages = {
  app: {
    brand: 'DOJ',
    signIn: 'Sign in',
    signUp: 'Sign up',
    signOut: 'Sign out',
    cancel: 'Cancel',
    user: 'User or email',
    userName: 'Name',
    email: 'Email',
    password: 'Password',
    locale: 'Language',
    colorMode: 'Color mode',
    light: 'Light',
    dark: 'Dark'
  },
  nav: {
    home: 'Home',
    problems: 'Problems',
    assignments: 'Assignments',
    contests: 'Contests',
    discussion: 'Discussion',
    rank: 'Rank',
    submissions: 'Submissions',
    admin: 'Admin',
    groups: 'Groups',
    users: 'Users',
    manageProblems: 'Problems',
    languages: 'Languages',
    runners: 'Runners'
  },
  common: {
    id: 'ID',
    problem: 'Problem',
    user: 'User',
    status: 'Status',
    language: 'Language',
    time: 'Time',
    memory: 'Memory',
    message: 'Message',
    title: 'Title',
    tags: 'Tags',
    solved: 'Solved'
  },
  dashboard: {
    title: 'Dashboard',
    subtitle: 'System activity, workload, and recent judging results.',
    problems: 'Problems',
    submissions: 'Submissions',
    users: 'Users',
    contests: 'Contests',
    assignments: 'Assignments',
    recentSubmissions: 'Recent submissions'
  },
  problems: {
    title: 'Problems',
    empty: 'No problems yet'
  },
  submissions: {
    title: 'Submissions'
  }
}

function getInitialLocale(): SupportedLocale {
  const saved = localStorage.getItem('doj.locale')
  if (saved === 'zh') return 'zh-CN'
  if (saved === 'zh-CN' || saved === 'en') return saved
  if (navigator.language.toLowerCase().startsWith('en')) return 'en'
  return fallbackLocale
}

export const i18n = createI18n({
  legacy: false,
  locale: getInitialLocale(),
  fallbackLocale,
  messages: {
    zh: zhCNMessages,
    'zh-CN': zhCNMessages,
    en: enMessages
  }
})

export function setLocale(locale: SupportedLocale) {
  i18n.global.locale.value = locale
  localStorage.setItem('doj.locale', locale)
}
