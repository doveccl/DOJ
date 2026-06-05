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
  },
  assignments: {
    title: '作业',
    subtitle: '分配给你的题目集。',
    signIn: '登录后查看作业。',
    empty: '还没有作业',
    due: '截止',
    late: '补交',
    allowed: '允许',
    closed: '关闭',
    ai: 'AI',
    on: '开',
    off: '关',
    fallback: '作业题目集。',
    lateAllowed: '允许补交',
    lateClosed: '补交关闭',
    duePrefix: '截止'
  },
  contests: {
    title: '比赛',
    type: '类型',
    start: '开始',
    end: '结束',
    key: '编号',
    score: '分值',
    fallback: '比赛题目集。',
    to: '至',
    freezes: '封榜',
    scoreboard: '榜单',
    frozen: '榜单已冻结。封榜后的提交将在揭榜前隐藏。',
    revealed: '管理员揭榜视图正在显示最终排名。',
    penalty: '罚时'
  },
  rank: {
    title: '排名',
    intro: '简介'
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
  },
  assignments: {
    title: 'Assignments',
    subtitle: 'Your assigned problem sets.',
    signIn: 'Sign in to view assignments.',
    empty: 'No assignments yet',
    due: 'Due',
    late: 'Late',
    allowed: 'allowed',
    closed: 'closed',
    ai: 'AI',
    on: 'on',
    off: 'off',
    fallback: 'Assigned problem set.',
    lateAllowed: 'late allowed',
    lateClosed: 'late closed',
    duePrefix: 'Due'
  },
  contests: {
    title: 'Contests',
    type: 'Type',
    start: 'Start',
    end: 'End',
    key: 'Key',
    score: 'Score',
    fallback: 'Contest problem set.',
    to: 'to',
    freezes: 'freezes',
    scoreboard: 'Scoreboard',
    frozen: 'Scoreboard is frozen. Submissions after the freeze are hidden until reveal.',
    revealed: 'Admin reveal view is showing final standings.',
    penalty: 'Penalty'
  },
  rank: {
    title: 'Rank',
    intro: 'Intro'
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
