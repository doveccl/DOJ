<script setup lang="ts">
import { apiFetch, getItems, type Paged, type PublicConfig } from '../api'
import { useAuthStore } from '../stores/auth'

interface HomeProblem {
  id: number
  title: string
  tags: string[]
  passRate: number
}

interface HomeAssignment {
  id: number
  title: string
  endAt: string | null
  completed?: number
  total?: number
}

interface HomeContest {
  id: number
  title: string
  type: string
  startAt: string
  endAt: string
}

type HeatmapDay = { date: string; count: number }

const router = useRouter()
const auth = useAuthStore()
const loading = ref(true)
const error = ref('')
const notice = ref('')
const recommendedProblems = ref<HomeProblem[]>([])
const myAssignments = ref<HomeAssignment[]>([])
const contests = ref<HomeContest[]>([])
const heatmap = ref<HeatmapDay[]>([])
const heatmapNeedsLogin = ref(false)

const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' })

const heatmapWeeks = computed(() => {
  const weeks: HeatmapDay[][] = []
  for (let index = 0; index < heatmap.value.length; index += 7) weeks.push(heatmap.value.slice(index, index + 7))
  return weeks
})

const maxHeatmapCount = computed(() => Math.max(1, ...heatmap.value.map((item) => item.count)))
const visibleRecommendedProblems = computed(() => recommendedProblems.value.slice(0, 6))

onMounted(() => {
  void loadHome()
})

watch(
  () => auth.signedIn,
  () => {
    void loadHome()
  }
)

async function loadHome() {
  loading.value = true
  error.value = ''
  try {
    const heatmapUrl = '/api/home/heatmap?tz=' + encodeURIComponent(Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC')
    const [config, problems, contestPage, assignmentPage, heatmapData] = await Promise.all([
      apiFetch<PublicConfig>('/api/config'),
      apiFetch<HomeProblem[]>('/api/home/recommended-problems'),
      apiFetch<HomeContest[]>('/api/home/contests'),
      auth.signedIn
        ? apiFetch<Paged<HomeAssignment>>('/api/my/assignments?pageSize=10')
        : Promise.resolve({ items: [], page: 1, pageSize: 50, total: 0 }),
      auth.signedIn ? apiFetch<HeatmapDay[]>(heatmapUrl) : Promise.resolve(null)
    ])

    notice.value = config.notice.trim()
    recommendedProblems.value = problems
    contests.value = contestPage
    myAssignments.value = getItems(assignmentPage)
    heatmap.value = heatmapData ?? []
    heatmapNeedsLogin.value = !auth.signedIn
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function formatDateTime(value: string | null) {
  return value ? dateTime.format(new Date(value)) : '-'
}

function go(path: string) {
  void router.push(path)
}

function heatmapLevel(count: number) {
  if (!count) return 0
  return Math.min(4, Math.ceil((count / maxHeatmapCount.value) * 4))
}

function assignmentProgress(assignment: HomeAssignment) {
  if (assignment.total === undefined || assignment.completed === undefined) return ''
  return assignment.completed + '/' + assignment.total
}
</script>

<template>
  <main class="page home-page">
    <n-spin :show="loading">
      <p v-if="error" class="form-error">{{ error }}</p>
      <template v-else>
        <n-alert v-if="notice" type="info" :show-icon="false" class="notice">{{ notice }}</n-alert>

        <n-grid class="home-columns" cols="1 760:3" :x-gap="16" :y-gap="16">
          <n-gi>
            <n-card title="推荐题目" :bordered="false" class="home-card">
              <template #header-extra>
                <n-button text type="primary" size="small" @click="go('/problems')">题库</n-button>
              </template>
              <n-list v-if="visibleRecommendedProblems.length" hoverable clickable>
                <n-list-item v-for="problem in visibleRecommendedProblems" :key="problem.id" @click="go('/problems/' + problem.id)">
                  <n-thing :title="'P' + problem.id + ' ' + problem.title">
                    <template #description>
                      <n-space :size="6" align="center">
                        <n-tag v-for="tag in problem.tags.slice(0, 3)" :key="tag" size="small" :bordered="false">{{ tag }}</n-tag>
                        <span class="muted">通过率 {{ Math.round(problem.passRate * 100) }}%</span>
                      </n-space>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
              <p v-else class="muted empty-copy">暂无推荐题目</p>
            </n-card>
          </n-gi>

          <n-gi>
            <n-card title="我的作业" :bordered="false" class="home-card">
              <template v-if="auth.signedIn">
                <n-list v-if="myAssignments.length" hoverable clickable>
                  <n-list-item v-for="assignment in myAssignments" :key="assignment.id" @click="go('/assignments/' + assignment.id)">
                    <n-thing :title="assignment.title">
                      <template #description>
                        <span class="muted">截止 {{ formatDateTime(assignment.endAt) }}<template v-if="assignmentProgress(assignment)"> · 完成 {{ assignmentProgress(assignment) }}</template></span>
                      </template>
                    </n-thing>
                  </n-list-item>
                </n-list>
                <p v-else class="muted empty-copy">暂无作业</p>
              </template>
              <div v-else class="empty-copy">
                <p class="muted">登录后查看你的作业。</p>
              </div>
            </n-card>
          </n-gi>

          <n-gi>
            <n-card title="近期比赛" :bordered="false" class="home-card">
              <n-list v-if="contests.length" hoverable clickable>
                <n-list-item v-for="contest in contests" :key="contest.id" @click="go('/contests/' + contest.id)">
                  <n-thing :title="contest.title"><template #description><span class="muted">{{ contest.type }} · {{ formatDateTime(contest.startAt) }}</span></template></n-thing>
                </n-list-item>
              </n-list>
              <p v-else class="muted empty-copy">暂无未结束比赛</p>
            </n-card>
          </n-gi>
        </n-grid>

        <n-card title="全年提交热力图" :bordered="false" class="heatmap-card">
          <p v-if="heatmapNeedsLogin" class="muted">登录后显示你的全年提交热力图。</p>
          <p v-else-if="heatmap.length" class="visually-hidden">
            过去一年共提交 {{ heatmap.reduce((sum, item) => sum + item.count, 0) }} 次。
          </p>
          <div v-if="!heatmapNeedsLogin && heatmap.length" class="heatmap" aria-hidden="true">
            <div v-for="(week, weekIndex) in heatmapWeeks" :key="weekIndex" class="heatmap-week">
              <span v-for="day in week" :key="day.date" class="heatmap-day" :class="'level-' + heatmapLevel(day.count)" :title="day.date + ': ' + day.count" />
            </div>
          </div>
          <p v-else-if="!heatmapNeedsLogin" class="muted empty-copy">过去一年暂无提交</p>
        </n-card>
      </template>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.home-page { display: grid; gap: 16px; }
.notice, .home-card, .heatmap-card { box-shadow: var(--shadow-sm); }
.home-columns { margin-bottom: 16px; }
.home-card { height: 100%; }
.home-card :deep(.n-card__content) { min-height: 172px; }
.home-card :deep(.n-thing-header__title) { overflow-wrap: anywhere; }
.empty-copy { margin: 16px 0 0; }
.heatmap { display: flex; gap: 3px; overflow-x: auto; padding-bottom: 4px; }
.heatmap-week { display: grid; grid-template-rows: repeat(7, 10px); gap: 3px; }
.heatmap-day { width: 10px; height: 10px; border-radius: 2px; background: color-mix(in srgb, var(--muted-color) 14%, transparent); }
.heatmap-day.level-1 { background: color-mix(in srgb, var(--brand) 25%, transparent); }
.heatmap-day.level-2 { background: color-mix(in srgb, var(--brand) 45%, transparent); }
.heatmap-day.level-3 { background: color-mix(in srgb, var(--brand) 70%, transparent); }
.heatmap-day.level-4 { background: var(--brand); }
</style>
