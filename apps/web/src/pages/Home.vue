<script setup lang="ts">
import { useI18n } from 'vue-i18n'
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
const { t, locale } = useI18n()
const auth = useAuthStore()
const loading = ref(true)
const error = ref('')
const notice = ref('')
const recommendedProblems = ref<HomeProblem[]>([])
const myAssignments = ref<HomeAssignment[]>([])
const contests = ref<HomeContest[]>([])
const heatmap = ref<HeatmapDay[]>([])

const dateTime = computed(
  () =>
    new Intl.DateTimeFormat(locale.value === 'en' ? 'en-US' : 'zh-CN', {
      dateStyle: 'medium',
      timeStyle: 'short'
    })
)
const dateOnly = computed(
  () => new Intl.DateTimeFormat(locale.value === 'en' ? 'en-US' : 'zh-CN', { dateStyle: 'medium' })
)

const visibleNewProblems = computed(() => recommendedProblems.value.slice(0, 5))
const visibleAssignments = computed(() => myAssignments.value.slice(0, 5))
const visibleContests = computed(() => contests.value.slice(0, 5))
const heatmapData = computed(() =>
  heatmap.value.map((item) => ({
    timestamp: new Date(`${item.date}T12:00:00`).getTime(),
    value: item.count
  }))
)
const heatmapTotal = computed(() => heatmap.value.reduce((sum, item) => sum + item.count, 0))
const heatmapTotalLabel = computed(() => t('home.yearSubmissions', { count: heatmapTotal.value }))

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
  return value ? dateTime.value.format(new Date(value)) : '-'
}

function formatHeatmapDate(timestamp: number) {
  return dateOnly.value.format(new Date(timestamp))
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
        <n-grid class="home-grid" cols="1 m:3" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
          <n-gi v-if="notice" span="1 m:3">
            <n-alert type="info" :show-icon="false" class="notice">{{ notice }}</n-alert>
          </n-gi>

          <n-gi v-if="auth.signedIn" span="1 m:3">
            <n-card :bordered="false" class="heatmap-card">
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
                  {{ formatHeatmapDate(timestamp) }} · {{ t('home.submissionCount', { count: value ?? 0 }) }}
                </template>
                <template #footer>
                  <span class="muted heatmap-total">{{ heatmapTotalLabel }}</span>
                </template>
                <template #indicator-leading-text>{{ t('home.less') }}</template>
                <template #indicator-trailing-text>{{ t('home.more') }}</template>
              </n-heatmap>
              <n-empty v-else size="small" />
            </n-card>
          </n-gi>

          <n-gi>
            <n-card :title="t('home.newProblems')" :bordered="false" class="home-card">
              <n-list v-if="visibleNewProblems.length" hoverable clickable>
                <n-list-item
                  v-for="problem in visibleNewProblems"
                  :key="problem.id"
                  class="home-list-item"
                  @click="go('/problems/' + problem.id)"
                >
                  <div class="home-row">
                    <n-ellipsis class="home-row-title">
                      P{{ problem.id }} {{ problem.title }}
                    </n-ellipsis>
                    <div class="home-row-meta">
                      <n-tag
                        v-for="tagItem in problem.tags.slice(0, 2)"
                        :key="tagItem"
                        size="small"
                        :bordered="false"
                      >
                        {{ tagItem }}
                      </n-tag>
                      <span class="muted">AC {{ Math.round(problem.passRate * 100) }}%</span>
                    </div>
                  </div>
                </n-list-item>
              </n-list>
              <n-empty v-else size="small" />
            </n-card>
          </n-gi>

          <n-gi>
            <n-card :title="t('home.myAssignments')" :bordered="false" class="home-card">
              <template v-if="auth.signedIn">
                <n-list v-if="visibleAssignments.length" hoverable clickable>
                  <n-list-item
                    v-for="assignment in visibleAssignments"
                    :key="assignment.id"
                    class="home-list-item"
                    @click="go('/assignments/' + assignment.id)"
                  >
                    <div class="home-row">
                      <n-ellipsis class="home-row-title">
                        {{ assignment.title }}
                      </n-ellipsis>
                      <n-ellipsis class="muted home-row-meta">
                        {{ t('assignments.due') }} {{ formatDateTime(assignment.endAt) }}
                        <template v-if="assignmentProgress(assignment)">
                          · {{ assignmentProgress(assignment) }}
                        </template>
                      </n-ellipsis>
                    </div>
                  </n-list-item>
                </n-list>
                <n-empty v-else size="small" />
              </template>
              <n-empty v-else size="small" />
            </n-card>
          </n-gi>

          <n-gi>
            <n-card :title="t('home.recentContests')" :bordered="false" class="home-card">
              <n-list v-if="visibleContests.length" hoverable clickable>
                <n-list-item
                  v-for="contest in visibleContests"
                  :key="contest.id"
                  class="home-list-item"
                  @click="go('/contests/' + contest.id)"
                >
                  <div class="home-row">
                    <n-ellipsis class="home-row-title">
                      {{ contest.title }}
                    </n-ellipsis>
                    <n-ellipsis class="muted home-row-meta">
                      {{ contest.type }} · {{ formatDateTime(contest.startAt) }}
                    </n-ellipsis>
                  </div>
                </n-list-item>
              </n-list>
              <n-empty v-else size="small" />
            </n-card>
          </n-gi>
        </n-grid>
      </template>
    </n-spin>
  </main>
</template>

<style scoped lang="scss">
.home-page { min-width: 0; }
.notice, .home-card, .heatmap-card { box-shadow: var(--shadow-sm); }
.home-card { height: 100%; }
.home-card :deep(.n-card__content) { min-height: 238px; }
.home-list-item {
  min-height: 44px;

  :deep(.n-list-item__main) {
    min-width: 0;
    max-width: 100%;
    overflow: hidden;
  }
}
.home-row {
  display: grid;
  gap: 6px;
  min-width: 0;
  width: 100%;
  max-width: 100%;
  overflow: hidden;
}
.home-row-title,
.home-row-meta {
  min-width: 0;
  width: 100%;
  max-width: 100%;
}
.home-row-title {
  display: block;
}
.home-row-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
  white-space: nowrap;
}
.heatmap-total {
  font-size: 13px;
}
.heatmap-card :deep(.n-card__content) {
  overflow-x: auto;
}
</style>
