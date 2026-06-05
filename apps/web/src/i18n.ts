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
    title: '提交记录',
    detailTitle: '提交',
    contest: '比赛',
    yes: '是',
    no: '否',
    source: '源码',
    restricted: '比赛提交详情仅提交者和管理员可见。',
    judgeMessage: '评测消息',
    testCases: '测试点',
    aiCoaching: 'AI 辅导',
    coachingUnavailable: 'AI 辅导仅用于比赛外的非 AC 提交。',
    getCoaching: '获取辅导'
  },
  problemDetail: {
    assignmentContext: '正在提交作业',
    contestContext: '正在提交比赛',
    submit: '提交',
    signIn: '登录后提交'
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
  },
  bbs: {
    title: '讨论',
    by: '作者',
    problem: '题目',
    contest: '比赛',
    reply: '回复',
    signInReply: '登录后回复。',
    signInTopic: '登录后发帖。',
    newTopic: '新主题',
    topic: '主题',
    author: '作者',
    updated: '更新',
    content: '内容',
    publish: '发布'
  },
  admin: {
    title: '管理',
    requireAdmin: '需要 admin 用户组权限。',
    actions: '操作',
    create: '创建',
    save: '保存',
    saveNewVersion: '保存新版本',
    cancel: '取消',
    edit: '编辑',
    load: '加载',
    upload: '上传',
    enabled: '启用',
    disabled: '停用',
    active: '正常',
    enable: '启用',
    disable: '停用',
    status: '状态',
    key: '标识',
    name: '名称',
    description: '描述',
    optionalNotes: '可选备注',
    sortOrder: '排序',
    groups: {
      title: '用户组',
      subtitle: '管理系统里的粗粒度访问分组。',
      create: '创建用户组',
      addMember: '添加成员',
      members: '成员',
      type: '类型',
      builtin: '内置',
      custom: '自定义',
      role: '角色',
      manager: '管理员',
      member: '成员',
      group: '用户组',
      groupManager: '用户组管理员'
    },
    users: {
      title: '用户',
      subtitle: '查看账号并在需要时暂停访问。',
      joined: '加入时间',
      submissions: '提交',
      solved: '通过'
    },
    languages: {
      title: '语言',
      subtitle: '配置启用的评测语言和 Docker 构建配方。',
      new: '新增语言',
      config: '语言配置',
      source: '源文件',
      sourceFile: '源文件',
      dockerfile: 'Dockerfile',
      commandOverride: '命令覆盖',
      commandPlaceholder: '每行一个 argv 项；留空则使用 Docker CMD'
    },
    runners: {
      title: '评测机',
      subtitle: '配置本地或远程 Docker API 评测后端。',
      new: '新增评测机',
      config: '评测机配置',
      endpoint: 'Docker 端点',
      local: '本地',
      concurrency: '并发',
      check: '检查'
    },
    problems: {
      title: '题目',
      subtitle: '创建题面版本并管理内联测试点。',
      create: '创建题目',
      edit: '编辑题目',
      uploadTestdata: '上传测试数据',
      visible: '可见',
      yes: '是',
      no: '否',
      slug: 'Slug',
      statement: '题面',
      timeMs: '时间 ms',
      memoryMb: '内存 MB',
      outputMb: '输出 MB',
      testCasesJson: '测试点 JSON',
      problemId: '题目编号',
      zipFile: 'ZIP 文件',
      publicVisible: '在公开题库显示',
      saved: '题目 {id} 已保存为版本 {version}。',
      uploaded: '已为题目 {id} 上传 {count} 个测试点。',
      testdata: '测试数据'
    },
    assignments: {
      title: '作业',
      subtitle: '向用户组发布题目集，可设置截止时间。',
      create: '创建作业',
      due: '截止',
      late: '补交',
      allowed: '允许',
      closed: '关闭',
      ai: 'AI',
      on: '开',
      off: '关',
      report: '报告',
      student: '学生',
      submitted: '已提交',
      dueAt: '截止时间',
      allowLate: '允许补交',
      aiCoaching: 'AI 辅导',
      selectGroups: '选择用户组',
      selectProblems: '选择题目'
    },
    contests: {
      title: '比赛',
      subtitle: '创建限时题目集，提交仅归属比赛。',
      create: '创建比赛',
      type: '类型',
      start: '开始',
      end: '结束',
      freeze: '封榜',
      startAt: '开始时间',
      endAt: '结束时间',
      freezeAt: '封榜时间',
      selectProblems: '选择题目'
    }
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
    title: 'Submissions',
    detailTitle: 'Submission',
    contest: 'Contest',
    yes: 'Yes',
    no: 'No',
    source: 'Source',
    restricted: 'Contest submission details are visible to the owner and admins.',
    judgeMessage: 'Judge Message',
    testCases: 'Test Cases',
    aiCoaching: 'AI Coaching',
    coachingUnavailable: 'Coaching is available for non-AC submissions outside contests.',
    getCoaching: 'Get coaching'
  },
  problemDetail: {
    assignmentContext: 'Submitting for assignment',
    contestContext: 'Submitting for contest',
    submit: 'Submit',
    signIn: 'Sign in to submit.'
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
  },
  bbs: {
    title: 'Discussion',
    by: 'by',
    problem: 'Problem',
    contest: 'Contest',
    reply: 'Reply',
    signInReply: 'Sign in to reply.',
    signInTopic: 'Sign in to start a topic.',
    newTopic: 'New topic',
    topic: 'Topic',
    author: 'Author',
    updated: 'Updated',
    content: 'Content',
    publish: 'Publish'
  },
  admin: {
    title: 'Admin',
    requireAdmin: 'Admin group is required.',
    actions: 'Action',
    create: 'Create',
    save: 'Save',
    saveNewVersion: 'Save new version',
    cancel: 'Cancel',
    edit: 'Edit',
    load: 'Load',
    upload: 'Upload',
    enabled: 'Enabled',
    disabled: 'Disabled',
    active: 'Active',
    enable: 'Enable',
    disable: 'Disable',
    status: 'Status',
    key: 'Key',
    name: 'Name',
    description: 'Description',
    optionalNotes: 'Optional notes',
    sortOrder: 'Sort order',
    groups: {
      title: 'Groups',
      subtitle: 'Manage coarse-grained access groups for the system.',
      create: 'Create group',
      addMember: 'Add member',
      members: 'Members',
      type: 'Type',
      builtin: 'Builtin',
      custom: 'Custom',
      role: 'Role',
      manager: 'Manager',
      member: 'Member',
      group: 'Group',
      groupManager: 'Group manager'
    },
    users: {
      title: 'Users',
      subtitle: 'Review accounts and suspend access when needed.',
      joined: 'Joined',
      submissions: 'Submissions',
      solved: 'Solved'
    },
    languages: {
      title: 'Languages',
      subtitle: 'Configure enabled judging languages and their Docker build recipes.',
      new: 'New language',
      config: 'Language config',
      source: 'Source',
      sourceFile: 'Source file',
      dockerfile: 'Dockerfile',
      commandOverride: 'Command override',
      commandPlaceholder: 'One argv item per line; leave empty to use Docker CMD'
    },
    runners: {
      title: 'Runners',
      subtitle: 'Configure local or remote Docker API judging backends.',
      new: 'New runner',
      config: 'Runner config',
      endpoint: 'Docker endpoint',
      local: 'local',
      concurrency: 'Concurrency',
      check: 'Check'
    },
    problems: {
      title: 'Problems',
      subtitle: 'Create statement versions with inline test cases.',
      create: 'Create problem',
      edit: 'Edit problem',
      uploadTestdata: 'Upload testdata',
      visible: 'Visible',
      yes: 'Yes',
      no: 'No',
      slug: 'Slug',
      statement: 'Statement',
      timeMs: 'Time ms',
      memoryMb: 'Memory MB',
      outputMb: 'Output MB',
      testCasesJson: 'Test cases JSON',
      problemId: 'Problem ID',
      zipFile: 'ZIP file',
      publicVisible: 'Show in public problem set',
      saved: 'Saved problem {id} as version {version}.',
      uploaded: 'Uploaded {count} cases for problem {id}.',
      testdata: 'Testdata'
    },
    assignments: {
      title: 'Assignments',
      subtitle: 'Publish problem sets to groups with optional deadlines.',
      create: 'Create assignment',
      due: 'Due',
      late: 'Late',
      allowed: 'allowed',
      closed: 'closed',
      ai: 'AI',
      on: 'on',
      off: 'off',
      report: 'Report',
      student: 'Student',
      submitted: 'Submitted',
      dueAt: 'Due at',
      allowLate: 'Allow late submissions',
      aiCoaching: 'AI coaching',
      selectGroups: 'Select groups',
      selectProblems: 'Select problems'
    },
    contests: {
      title: 'Contests',
      subtitle: 'Create timed problem sets with contest-only submissions.',
      create: 'Create contest',
      type: 'Type',
      start: 'Start',
      end: 'End',
      freeze: 'Freeze',
      startAt: 'Start at',
      endAt: 'End at',
      freezeAt: 'Freeze at',
      selectProblems: 'Select problems'
    }
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
