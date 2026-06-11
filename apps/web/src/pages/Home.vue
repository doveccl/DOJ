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

const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' })

const visibleRecommendedProblems = computed(() => recommendedProblems.value.slice(0, 5))
const visibleAssignments = computed(() => myAssignments.value.slice(0, 5))
const visibleContests = computed(() => contests.value.slice(0, 5))
const heatmapData = computed(() =>
  heatmap.value.map((item) => ({
    timestamp: new Date(`${item.date}T12:00:00`).getTime(),
    value: item.count
  }))
)
const heatmapTotal = computed(() => heatmap.value.reduce((sum, item) => sum + item.count, 0))

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
      fallbackForGuest(apiFetch<HomeProblem[]>('/api/home/recommended-problems'), []),
      fallbackForGuest(apiFetch<HomeContest[]>('/api/home/contests'), []),
      auth.signedIn
        ? apiFetch<Paged<HomeAssignment>>('/api/my/assignments?pageSize=5')
        : Promise.resolve({ items: [], page: 1, pageSize: 50, total: 0 }),
      auth.signedIn ? apiFetch<HeatmapDay[]>(heatmapUrl) : Promise.resolve(null)
    ])

    notice.value = config.notice.trim()
    recommendedProblems.value = problems
    contests.value = contestPage
    myAssignments.value = getItems(assignmentPage)
    heatmap.value = heatmapData ?? []
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function fallbackForGuest<T>(promise: Promise<T>, fallback: T) {
  try {
    return await promise
  } catch (cause) {
    if (!auth.signedIn) return fallback
    throw cause
  }
}

function formatDateTime(value: string | null) {
  return value ? dateTime.format(new Date(value)) : '-'
}

function go(path: string) {
  void router.push(path)
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
                <n-list-item
                  v-for="problem in visibleRecommendedProblems"
                  :key="problem.id"
                  class="home-list-item"
                  @click="go('/problems/' + problem.id)"
                >
                  <n-thing>
                    <template #header>
                      <span class="home-row-title">P{{ problem.id }} {{ problem.title }}</span>
                    </template>
                    <template #description>
                      <n-space :size="6" align="center" class="home-row-meta">
                        <n-tag
                          v-for="tagItem in problem.tags.slice(0, 2)"
                          :key="tagItem"
                          size="small"
                          :bordered="false"
                        >
                          {{ tagItem }}
                        </n-tag>
                        <span class="muted">AC {{ Math.round(problem.passRate * 100) }}%</span>
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
                <n-list v-if="visibleAssignments.length" hoverable clickable>
                  <n-list-item
                    v-for="assignment in visibleAssignments"
                    :key="assignment.id"
                    class="home-list-item"
                    @click="go('/assignments/' + assignment.id)"
                  >
                    <n-thing>
                      <template #header>
                        <span class="home-row-title">{{ assignment.title }}</span>
                      </template>
                      <template #description>
                        <span class="muted home-row-meta">
                          截止 {{ formatDateTime(assignment.endAt) }}
                          <template v-if="assignmentProgress(assignment)">
                            · {{ assignmentProgress(assignment) }}
                          </template>
                        </span>
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
              <n-list v-if="visibleContests.length" hoverable clickable>
                <n-list-item
                  v-for="contest in visibleContests"
                  :key="contest.id"
                  class="home-list-item"
                  @click="go('/contests/' + contest.id)"
                >
                  <n-thing>
                    <template #header>
                      <span class="home-row-title">{{ contest.title }}</span>
                    </template>
                    <template #description>
                      <span class="muted home-row-meta">
                        {{ contest.type }} · {{ formatDateTime(contest.startAt) }}
                      </span>
                    </template>
                  </n-thing>
                </n-list-item>
              </n-list>
              <p v-else class="muted empty-copy">暂无未结束比赛</p>
            </n-card>
          </n-gi>
        </n-grid>

        <n-card v-if="auth.signedIn" title="全年提交热力图" :bordered="false" class="heatmap-card">
          <template #header-extra>
            <span class="muted heatmap-total">{{ heatmapTotal }} 次提交</span>
          </template>
          <n-heatmap
            v-if="heatmapData.length"
            :data="heatmapData"
            :first-day-of-week="1"
            :show-week-labels="true"
            :show-month-labels="true"
            :show-color-indicator="true"
            :tooltip="{ placement: 'top' }"
            size="small"
          >
            <template #tooltip="{ timestamp, value }">
              {{ new Date(timestamp).toLocaleDateString() }} · {{ value ?? 0 }} 次
            </template>
            <template #indicator-leading-text>少</template>
            <template #indicator-trailing-text>多</template>
          </n-heatmap>
          <p v-else class="muted empty-copy">过去一年暂无提交</p>
        </n-card>
      </template>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.home-page { display: grid; gap: 18px; }
.notice, .home-card, .heatmap-card { box-shadow: var(--shadow-sm); }
.home-card { height: 100%; }
.home-card :deep(.n-card__content) { min-height: 238px; }
.home-list-item {
  min-height: 44px;
}
.home-row-title,
.home-row-meta {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.empty-copy { margin: 16px 0 0; }
.heatmap-card :deep(.n-card__content) {
  overflow-x: auto;
  padding-top: 6px;
}
.heatmap-total {
  font-size: 13px;
}
</style>
